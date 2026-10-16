package gostd

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"qchain-kpi/internal/algs"
)

// DHKEM(P-256, HKDF-SHA256) and DHKEM(X25519, HKDF-SHA256) exactly as RFC 9180
// section 4.1 specifies them, on crypto/ecdh and crypto/hkdf (Go 1.24 has no
// HPKE package). crypto/ecdh validates P-256 points and rejects an all-zero
// X25519 result, as the RFC requires. Checked against the RFC's official
// vectors and against internal/classical's OpenSSL implementation.

type dhkem struct {
	curve   ecdh.Curve
	suite   []byte // "KEM" || I2OSP(kem_id, 2)
	nsecret int
}

func kemOf(a algs.Alg) (*dhkem, error) {
	switch a.ID {
	case "dhkem-p256":
		return &dhkem{curve: ecdh.P256(), suite: []byte{'K', 'E', 'M', 0x00, 0x10}, nsecret: 32}, nil
	case "dhkem-x25519":
		return &dhkem{curve: ecdh.X25519(), suite: []byte{'K', 'E', 'M', 0x00, 0x20}, nsecret: 32}, nil
	}
	return nil, fmt.Errorf("gostd: %s is not a classical KEM", a.ID)
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// LabeledExtract(salt, label, ikm) = Extract(salt, "HPKE-v1" || suite_id || label || ikm).
func (k *dhkem) labeledExtract(salt []byte, label string, ikm []byte) ([]byte, error) {
	return hkdf.Extract(sha256.New, cat([]byte("HPKE-v1"), k.suite, []byte(label), ikm), salt)
}

// LabeledExpand(prk, label, info, L) = Expand(prk, I2OSP(L, 2) || "HPKE-v1" || suite_id || label || info, L).
func (k *dhkem) labeledExpand(prk []byte, label string, info []byte, l int) ([]byte, error) {
	li := cat([]byte{byte(l >> 8), byte(l)}, []byte("HPKE-v1"), k.suite, []byte(label), info)
	return hkdf.Expand(sha256.New, prk, string(li), l)
}

func (k *dhkem) extractAndExpand(dh, kemContext []byte) ([]byte, error) {
	prk, err := k.labeledExtract(nil, "eae_prk", dh)
	if err != nil {
		return nil, err
	}
	return k.labeledExpand(prk, "shared_secret", kemContext, k.nsecret)
}

// encapWith is Encap(pkR) with the given ephemeral key.
func (k *dhkem) encapWith(skE *ecdh.PrivateKey, pkR *ecdh.PublicKey, pkRm []byte) (enc, ss []byte, err error) {
	dh, err := skE.ECDH(pkR)
	if err != nil {
		return nil, nil, err
	}
	enc = skE.PublicKey().Bytes()
	ss, err = k.extractAndExpand(dh, cat(enc, pkRm))
	return enc, ss, err
}

// KEMKeygen generates a key pair in the shared formats (P-256: 32-byte
// scalar and 65-byte SEC1 point; X25519: 32 bytes each).
func KEMKeygen(a algs.Alg) (pk, sk []byte, err error) {
	k, err := kemOf(a)
	if err != nil {
		return nil, nil, err
	}
	s, err := k.curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return s.PublicKey().Bytes(), s.Bytes(), nil
}

// Encapsulator encapsulates to one parsed public key.
type Encapsulator func() (ct, ss []byte, err error)

// Decapsulator decapsulates with one parsed key pair.
type Decapsulator func(ct []byte) ([]byte, error)

// NewEncapsulator parses the recipient's public key once.
func NewEncapsulator(a algs.Alg, pk []byte) (Encapsulator, error) {
	k, err := kemOf(a)
	if err != nil {
		return nil, err
	}
	pkR, err := k.curve.NewPublicKey(pk)
	if err != nil {
		return nil, err
	}
	pkRm := append([]byte(nil), pk...)
	return func() ([]byte, []byte, error) {
		skE, err := k.curve.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return k.encapWith(skE, pkR, pkRm)
	}, nil
}

// NewDecapsulator parses the recipient's key pair once.
func NewDecapsulator(a algs.Alg, sk, pk []byte) (Decapsulator, error) {
	k, err := kemOf(a)
	if err != nil {
		return nil, err
	}
	skR, err := k.curve.NewPrivateKey(sk)
	if err != nil {
		return nil, err
	}
	pkRm := append([]byte(nil), pk...)
	return func(enc []byte) ([]byte, error) {
		pkE, err := k.curve.NewPublicKey(enc)
		if err != nil {
			return nil, err
		}
		dh, err := skR.ECDH(pkE)
		if err != nil {
			return nil, err
		}
		return k.extractAndExpand(dh, cat(enc, pkRm))
	}, nil
}
