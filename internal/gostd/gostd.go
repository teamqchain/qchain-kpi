// Package gostd is E3's L2 layer: the 5 classical baselines through Go's
// standard library (crypto/ecdsa, crypto/ed25519, crypto/rsa, crypto/ecdh,
// crypto/hkdf), a secondary reference for the OpenSSL-through-cgo baselines
// of internal/classical. It uses the same key formats as internal/classical
// (see its ossl.h), so the same prepared keys and inputs feed every layer,
// and every result is checked against OpenSSL:
//
//	ECDSA P-256 with SHA-256: secret = 32-byte scalar, public = SEC1 uncompressed (65 B), DER signatures
//	Ed25519 (RFC 8032):       secret = 32-byte seed,   public = 32 B
//	RSA-3072 PSS, SHA-256, 32-byte salt: secret = PKCS#8 DER, public = SubjectPublicKeyInfo DER
//	DHKEM(P-256 / X25519, HKDF-SHA256), RFC 9180 section 4.1 (dhkem.go)
//
// As in the reuse arm (E1), the key objects are parsed once; only the
// operation itself is done per call. Signing hashes the message inside the
// call, as OpenSSL's one-shot signing does.
package gostd

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"

	"qchain-kpi/internal/algs"
)

// Signer signs messages with one parsed key.
type Signer func(msg []byte) ([]byte, error)

// Verifier verifies signatures against one parsed public key.
type Verifier func(msg, sig []byte) (bool, error)

var pssOpts = &rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256}

// SigKeygen generates a key pair in the shared formats.
func SigKeygen(a algs.Alg) (pk, sk []byte, err error) {
	switch a.ID {
	case "ecdsa-p256":
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		pub, err := k.PublicKey.ECDH()
		if err != nil {
			return nil, nil, err
		}
		return pub.Bytes(), k.D.FillBytes(make([]byte, 32)), nil
	case "ed25519":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return pub, priv.Seed(), nil
	case "rsa3072-pss":
		k, err := rsa.GenerateKey(rand.Reader, 3072)
		if err != nil {
			return nil, nil, err
		}
		if sk, err = x509.MarshalPKCS8PrivateKey(k); err != nil {
			return nil, nil, err
		}
		if pk, err = x509.MarshalPKIXPublicKey(&k.PublicKey); err != nil {
			return nil, nil, err
		}
		return pk, sk, nil
	}
	return nil, nil, fmt.Errorf("gostd: %s is not a classical signature", a.ID)
}

// NewSigner parses the key pair once and returns the signing operation.
func NewSigner(a algs.Alg, sk, pk []byte) (Signer, error) {
	switch a.ID {
	case "ecdsa-p256":
		pub, err := ecdsaPublic(pk)
		if err != nil {
			return nil, err
		}
		if len(sk) != 32 {
			return nil, fmt.Errorf("gostd: ECDSA secret key of %d bytes", len(sk))
		}
		k := &ecdsa.PrivateKey{PublicKey: *pub, D: new(big.Int).SetBytes(sk)}
		return func(msg []byte) ([]byte, error) {
			h := sha256.Sum256(msg)
			return ecdsa.SignASN1(rand.Reader, k, h[:])
		}, nil
	case "ed25519":
		if len(sk) != ed25519.SeedSize {
			return nil, fmt.Errorf("gostd: Ed25519 seed of %d bytes", len(sk))
		}
		k := ed25519.NewKeyFromSeed(sk)
		return func(msg []byte) ([]byte, error) { return ed25519.Sign(k, msg), nil }, nil
	case "rsa3072-pss":
		key, err := x509.ParsePKCS8PrivateKey(sk)
		if err != nil {
			return nil, err
		}
		k, ok := key.(*rsa.PrivateKey)
		if !ok || k.N.BitLen() != 3072 {
			return nil, errors.New("gostd: not an RSA-3072 private key")
		}
		return func(msg []byte) ([]byte, error) {
			h := sha256.Sum256(msg)
			return rsa.SignPSS(rand.Reader, k, crypto.SHA256, h[:], pssOpts)
		}, nil
	}
	return nil, fmt.Errorf("gostd: %s is not a classical signature", a.ID)
}

// NewVerifier parses the public key once and returns the verification.
// A wrong signature is (false, nil), as in internal/classical.
func NewVerifier(a algs.Alg, pk []byte) (Verifier, error) {
	switch a.ID {
	case "ecdsa-p256":
		pub, err := ecdsaPublic(pk)
		if err != nil {
			return nil, err
		}
		return func(msg, sig []byte) (bool, error) {
			h := sha256.Sum256(msg)
			return ecdsa.VerifyASN1(pub, h[:], sig), nil
		}, nil
	case "ed25519":
		if len(pk) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("gostd: Ed25519 public key of %d bytes", len(pk))
		}
		pub := ed25519.PublicKey(append([]byte(nil), pk...))
		return func(msg, sig []byte) (bool, error) { return ed25519.Verify(pub, msg, sig), nil }, nil
	case "rsa3072-pss":
		key, err := x509.ParsePKIXPublicKey(pk)
		if err != nil {
			return nil, err
		}
		pub, ok := key.(*rsa.PublicKey)
		if !ok || pub.N.BitLen() != 3072 {
			return nil, errors.New("gostd: not an RSA-3072 public key")
		}
		return func(msg, sig []byte) (bool, error) {
			h := sha256.Sum256(msg)
			return rsa.VerifyPSS(pub, crypto.SHA256, h[:], sig, pssOpts) == nil, nil
		}, nil
	}
	return nil, fmt.Errorf("gostd: %s is not a classical signature", a.ID)
}

// ecdsaPublic parses a SEC1 uncompressed P-256 point (crypto/ecdh checks it
// is on the curve).
func ecdsaPublic(pk []byte) (*ecdsa.PublicKey, error) {
	if _, err := ecdh.P256().NewPublicKey(pk); err != nil {
		return nil, fmt.Errorf("gostd: ECDSA public key: %w", err)
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(pk[1:33]), Y: new(big.Int).SetBytes(pk[33:65])}, nil
}
