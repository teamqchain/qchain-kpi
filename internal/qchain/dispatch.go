package qchain

// Harness code (not copied), decision A of 11 Oct: suite selection, and the
// OpenSSL twins that let the copied production code run the classical
// baselines. Production calls four liboqs helpers (pqcSign, pqcVerify,
// kemEncap, kemDecap; kemGenerateKeypair for key generation). Their copied
// bodies are kept unchanged under ...Liboqs names; the functions below carry
// the original names, so every copied caller (signCommitment, sealAttributes,
// openAttributes, verifyPresentation, ...) is untouched. For a liboqs algorithm
// they call the copied body directly; for a classical one they run the same
// pattern on OpenSSL: hex-decode, a fresh key object, the operation, free the
// object, hex-encode.
//
// Classical keys need the public half in two places where production passes
// only the secret key: an ECDSA scalar is imported together with its public
// point, and RFC 9180 DHKEM decapsulation uses the recipient's public key.
// The harness keeps the public key of every classical secret key it generates
// (pubOf, keyed by the raw secret key) and looks it up there; the lookup
// costs nanoseconds against operations of tens of microseconds.

import (
	"encoding/hex"
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/scheme"
)

type sigTwin struct {
	a   algs.Alg
	s   scheme.Signature
	obj scheme.SigObject
}

type kemTwin struct {
	a   algs.Alg
	k   scheme.KEM
	obj scheme.KEMObject
}

var (
	clSig *sigTwin // nil: the suite's signature is a liboqs one (sigName)
	clKEM *kemTwin // nil: the suite's KEM is a liboqs one (kemName)
	pubOf = map[string][]byte{}
)

// SetSuite selects the signature and KEM every copied function uses. One
// process measures one suite, so package-level state is sufficient. It also
// sets the activation key lengths (decision B: production hard-codes the
// ML-KEM-768 and ML-DSA-44 lengths).
func SetSuite(sig, kem algs.Alg) error {
	if sig.Track != algs.Sig || kem.Track != algs.KEM {
		return fmt.Errorf("qchain: suite %s + %s is not a signature + KEM", sig.ID, kem.ID)
	}
	clSig, clKEM = nil, nil
	if sig.Classical() {
		s, err := classical.NewSignature(sig)
		if err != nil {
			return err
		}
		clSig = &sigTwin{a: sig, s: s, obj: s.(scheme.SigObjects).NewSigObject()}
		sigName = sig.ID
	} else {
		sigName = sig.OQSName
	}
	if kem.Classical() {
		k, err := classical.NewKEM(kem)
		if err != nil {
			return err
		}
		clKEM = &kemTwin{a: kem, k: k, obj: k.(scheme.KEMObjects).NewKEMObject()}
		kemName = kem.ID
	} else {
		kemName = kem.OQSName
	}
	sz, err := sigSizes(sig)
	if err != nil {
		return err
	}
	kz, err := kemSizes(kem)
	if err != nil {
		return err
	}
	dsaPublicKeyHexLen = 2 * sz.PublicKey
	kemPublicKeyHexLen = 2 * kz.PublicKey
	return nil
}

func sigSizes(a algs.Alg) (scheme.Sizes, error) {
	if a.Classical() {
		s, err := classical.NewSignature(a)
		if err != nil {
			return scheme.Sizes{}, err
		}
		return s.Sizes(), nil
	}
	s, err := pqc.NewSignature(a)
	if err != nil {
		return scheme.Sizes{}, err
	}
	return s.Sizes(), nil
}

func kemSizes(a algs.Alg) (scheme.Sizes, error) {
	if a.Classical() {
		k, err := classical.NewKEM(a)
		if err != nil {
			return scheme.Sizes{}, err
		}
		return k.Sizes(), nil
	}
	k, err := pqc.NewKEM(a)
	if err != nil {
		return scheme.Sizes{}, err
	}
	return k.Sizes(), nil
}

// classicalKEMFor returns the twin when name is the selected classical KEM
// (the wallet mirror passes the envelope's kemAlg).
func classicalKEMFor(name string) *kemTwin {
	if clKEM != nil && name == clKEM.a.ID {
		return clKEM
	}
	return nil
}

// ------------------------------------------------------------- dispatchers

func pqcSign(message, privateKeyHex string) (string, error) {
	if clSig != nil {
		return clSig.sign(message, privateKeyHex)
	}
	return pqcSignLiboqs(message, privateKeyHex)
}

func pqcVerify(message, signatureHex, publicKeyHex string) (bool, error) {
	if clSig != nil {
		return clSig.verify(message, signatureHex, publicKeyHex)
	}
	return pqcVerifyLiboqs(message, signatureHex, publicKeyHex)
}

func kemGenerateKeypair() (pubHex, secHex string, err error) {
	if clKEM != nil {
		return clKEM.keygen()
	}
	return kemGenerateKeypairLiboqs()
}

func kemEncap(pubHex string) (kemCtHex, ssHex string, err error) {
	if clKEM != nil {
		return clKEM.encap(pubHex)
	}
	return kemEncapLiboqs(pubHex)
}

func kemDecap(kemCtHex, secHex string) (ssHex string, err error) {
	if clKEM != nil {
		return clKEM.decap(kemCtHex, secHex)
	}
	return kemDecapLiboqs(kemCtHex, secHex)
}

// sigGenerateKeypair is the wallet's (and the keygen tool's) signature key
// generation: liboqs via the ProdSigKeygen mirror, classical via the twin.
func sigGenerateKeypair() (pubHex, privHex string, err error) {
	if clSig != nil {
		return clSig.keygen()
	}
	return ProdSigKeygen()
}

// ------------------------------------------------------------ OpenSSL twins

func (t *sigTwin) keygen() (string, string, error) {
	pk, sk, err := t.s.GenerateKey()
	if err != nil {
		return "", "", err
	}
	pubOf[string(sk)] = pk
	return hex.EncodeToString(pk), hex.EncodeToString(sk), nil
}

func (t *sigTwin) sign(message, privateKeyHex string) (string, error) {
	sk, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("decoding private key hex: %w", err)
	}
	defer t.obj.Close()
	if err := t.obj.Open(scheme.Private, sk, pubOf[string(sk)]); err != nil {
		return "", fmt.Errorf("load %s key: %w", t.a.ID, err)
	}
	sig, err := t.obj.Sign([]byte(message))
	if err != nil {
		return "", fmt.Errorf("signing: %w", err)
	}
	return hex.EncodeToString(sig), nil
}

func (t *sigTwin) verify(message, signatureHex, publicKeyHex string) (bool, error) {
	pk, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, fmt.Errorf("decoding public key hex: %w", err)
	}
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("decoding signature hex: %w", err)
	}
	defer t.obj.Close()
	if err := t.obj.Open(scheme.Public, nil, pk); err != nil {
		return false, fmt.Errorf("load %s public key: %w", t.a.ID, err)
	}
	return t.obj.Verify([]byte(message), sig)
}

func (t *kemTwin) keygen() (string, string, error) {
	pk, sk, err := t.k.GenerateKey()
	if err != nil {
		return "", "", err
	}
	pubOf[string(sk)] = pk
	return hex.EncodeToString(pk), hex.EncodeToString(sk), nil
}

func (t *kemTwin) encap(pubHex string) (string, string, error) {
	pk, err := hex.DecodeString(pubHex)
	if err != nil {
		return "", "", fmt.Errorf("decode KEM public key: %w", err)
	}
	defer t.obj.Close()
	if err := t.obj.Open(scheme.Public, nil, pk); err != nil {
		return "", "", fmt.Errorf("load %s public key: %w", t.a.ID, err)
	}
	ct, ss, err := t.obj.Encaps()
	if err != nil {
		return "", "", fmt.Errorf("KEM encapsulate: %w", err)
	}
	return hex.EncodeToString(ct), hex.EncodeToString(ss), nil
}

func (t *kemTwin) decap(kemCtHex, secHex string) (string, error) {
	ct, err := hex.DecodeString(kemCtHex)
	if err != nil {
		return "", fmt.Errorf("decode KEM ciphertext: %w", err)
	}
	sec, err := hex.DecodeString(secHex)
	if err != nil {
		return "", fmt.Errorf("decode KEM secret key: %w", err)
	}
	ss, err := t.decapRaw(sec, ct)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ss), nil
}

// decapRaw is the decapsulation step on raw bytes (the wallet mirror's form).
func (t *kemTwin) decapRaw(sec, ct []byte) ([]byte, error) {
	pk, ok := pubOf[string(sec)]
	if !ok {
		return nil, fmt.Errorf("%s: no public key known for this secret key", t.a.ID)
	}
	defer t.obj.Close()
	if err := t.obj.Open(scheme.Private, sec, pk); err != nil {
		return nil, fmt.Errorf("load %s key: %w", t.a.ID, err)
	}
	return t.obj.Decaps(ct)
}

// RegisterClassicalKey records the public half of a classical key pair made
// outside the twins (input preparation), so the twins can find it.
func RegisterClassicalKey(pk, sk []byte) { pubOf[string(sk)] = pk }
