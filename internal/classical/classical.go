// Package classical provides the 5 classical baselines through OpenSSL 3.0
// (see ossl.h for algorithms and key formats). Each measured operation is one
// cgo call, the same Go-to-C boundary the liboqs-go algorithms cross. Go
// memory is passed by pointer (byte slices contain no Go pointers), so no
// copies are made on the way in.
package classical

/*
#cgo pkg-config: libcrypto
#include <stdlib.h>
#include "ossl.h"
*/
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/scheme"
)

var initOnce sync.Once
var initErr error

func ensureInit() error {
	initOnce.Do(func() {
		if C.kpi_ossl_init() != 1 {
			initErr = errors.New("classical: OpenSSL initialisation failed")
		}
	})
	return initErr
}

// OpenSSLVersion returns the linked OpenSSL version string (E0).
func OpenSSLVersion() string { return C.GoString(C.kpi_ossl_version()) }

func ptr(b []byte) *C.uchar {
	if len(b) == 0 {
		return nil
	}
	return (*C.uchar)(unsafe.Pointer(&b[0]))
}

var errOp = errors.New("OpenSSL operation failed")

// ---------------------------------------------------------------- signatures

type sig struct {
	a     algs.Alg
	code  C.int
	sizes scheme.Sizes
	sigCap int
}

// NewSignature returns the OpenSSL implementation of a classical signature.
// It performs no cryptographic operation (a measured process must not run
// any before its cold call).
func NewSignature(a algs.Alg) (scheme.Signature, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	s := &sig{a: a}
	switch a.ID {
	case "ecdsa-p256":
		s.code, s.sigCap = C.KPI_ECDSA_P256, 72
		s.sizes = scheme.Sizes{PublicKey: 65, SecretKey: 32, Signature: 72, SigVariable: true}
	case "ed25519":
		s.code, s.sigCap = C.KPI_ED25519, 64
		s.sizes = scheme.Sizes{PublicKey: 32, SecretKey: 32, Signature: 64}
	case "rsa3072-pss":
		s.code, s.sigCap = C.KPI_RSA3072_PSS, 384
		// SubjectPublicKeyInfo DER of a 3072-bit modulus with e = 65537 is
		// always 422 bytes; the PKCS#8 private key varies by a few bytes
		// (DER drops leading zero bytes of its CRT components).
		s.sizes = scheme.Sizes{PublicKey: 422, SKVariable: true, Signature: 384}
	default:
		return nil, fmt.Errorf("classical: %s is not a classical signature", a.ID)
	}
	return s, nil
}

func (s *sig) Alg() algs.Alg       { return s.a }
func (s *sig) Sizes() scheme.Sizes { return s.sizes }

func (s *sig) GenerateKey() ([]byte, []byte, error) {
	pk := make([]byte, 1024)
	sk := make([]byte, 4096)
	pkl, skl := C.size_t(len(pk)), C.size_t(len(sk))
	if C.kpi_sig_keygen(s.code, ptr(pk), &pkl, ptr(sk), &skl) != 1 {
		return nil, nil, fmt.Errorf("%s keygen: %w", s.a.ID, errOp)
	}
	return bytes.Clone(pk[:pkl]), bytes.Clone(sk[:skl]), nil
}

func (s *sig) loadPrivate(sk, pk []byte) (*C.EVP_PKEY, error) {
	k := C.kpi_sig_load_private(s.code, ptr(sk), C.size_t(len(sk)), ptr(pk), C.size_t(len(pk)))
	if k == nil {
		return nil, fmt.Errorf("%s: load private key: %w", s.a.ID, errOp)
	}
	return k, nil
}

func (s *sig) loadPublic(pk []byte) (*C.EVP_PKEY, error) {
	k := C.kpi_sig_load_public(s.code, ptr(pk), C.size_t(len(pk)))
	if k == nil {
		return nil, fmt.Errorf("%s: load public key: %w", s.a.ID, errOp)
	}
	return k, nil
}

func (s *sig) signWith(k *C.EVP_PKEY, msg []byte) ([]byte, error) {
	out := make([]byte, s.sigCap)
	n := C.size_t(len(out))
	if C.kpi_sig_sign(s.code, k, ptr(msg), C.size_t(len(msg)), ptr(out), &n) != 1 {
		return nil, fmt.Errorf("%s sign: %w", s.a.ID, errOp)
	}
	return out[:n], nil
}

func (s *sig) verifyWith(k *C.EVP_PKEY, msg, sg []byte) (bool, error) {
	switch C.kpi_sig_verify(s.code, k, ptr(msg), C.size_t(len(msg)), ptr(sg), C.size_t(len(sg))) {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, fmt.Errorf("%s verify: %w", s.a.ID, errOp)
	}
}

// Sign is the untimed signing helper. ECDSA imports the secret scalar together
// with its public point; Ed25519 (raw) and RSA (PKCS#8) ignore pk.
func (s *sig) Sign(sk, pk, msg []byte) ([]byte, error) {
	k, err := s.loadPrivate(sk, pk)
	if err != nil {
		return nil, err
	}
	defer C.kpi_key_free(k)
	out, err := s.signWith(k, msg)
	if err != nil {
		return nil, err
	}
	return bytes.Clone(out), nil
}

func (s *sig) Verify(pk, msg, sg []byte) (bool, error) {
	k, err := s.loadPublic(pk)
	if err != nil {
		return false, err
	}
	defer C.kpi_key_free(k)
	return s.verifyWith(k, msg, sg)
}

type sigKeygen struct{ s *sig }

func (s *sig) NewKeygenReuse() (scheme.KeygenReuse, error) { return &sigKeygen{s: s}, nil }
func (r *sigKeygen) Keygen() ([]byte, []byte, error)        { return r.s.GenerateKey() }
func (r *sigKeygen) Close()                                 {}

type signer struct {
	s *sig
	k *C.EVP_PKEY
}

func (s *sig) NewSignerReuse(sk, pk []byte) (scheme.SignerReuse, error) {
	k, err := s.loadPrivate(sk, pk)
	if err != nil {
		return nil, err
	}
	return &signer{s: s, k: k}, nil
}
func (r *signer) Sign(msg []byte) ([]byte, error) { return r.s.signWith(r.k, msg) }
func (r *signer) Close() {
	if r.k != nil {
		C.kpi_key_free(r.k)
		r.k = nil
	}
}

type verifier struct {
	s *sig
	k *C.EVP_PKEY
}

func (s *sig) NewVerifierReuse(pk []byte) (scheme.VerifierReuse, error) {
	k, err := s.loadPublic(pk)
	if err != nil {
		return nil, err
	}
	return &verifier{s: s, k: k}, nil
}
func (r *verifier) Verify(msg, sg []byte) (bool, error) { return r.s.verifyWith(r.k, msg, sg) }
func (r *verifier) Close() {
	if r.k != nil {
		C.kpi_key_free(r.k)
		r.k = nil
	}
}

// ---------------------------------------------------------------------- KEMs

type kem struct {
	a     algs.Alg
	code  C.int
	sizes scheme.Sizes
}

// NewKEM returns the RFC 9180 DHKEM implementation of a classical KEM.
func NewKEM(a algs.Alg) (scheme.KEM, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	k := &kem{a: a}
	switch a.ID {
	case "dhkem-p256":
		k.code = C.KPI_DHKEM_P256
		k.sizes = scheme.Sizes{PublicKey: 65, SecretKey: 32, Ciphertext: 65, SharedSecret: 32}
	case "dhkem-x25519":
		k.code = C.KPI_DHKEM_X25519
		k.sizes = scheme.Sizes{PublicKey: 32, SecretKey: 32, Ciphertext: 32, SharedSecret: 32}
	default:
		return nil, fmt.Errorf("classical: %s is not a classical KEM", a.ID)
	}
	return k, nil
}

func (k *kem) Alg() algs.Alg       { return k.a }
func (k *kem) Sizes() scheme.Sizes { return k.sizes }

func (k *kem) GenerateKey() ([]byte, []byte, error) {
	pk := make([]byte, 65)
	sk := make([]byte, 32)
	pkl, skl := C.size_t(len(pk)), C.size_t(len(sk))
	if C.kpi_kem_keygen(k.code, ptr(pk), &pkl, ptr(sk), &skl) != 1 {
		return nil, nil, fmt.Errorf("%s keygen: %w", k.a.ID, errOp)
	}
	return bytes.Clone(pk[:pkl]), bytes.Clone(sk[:skl]), nil
}

func (k *kem) loadPublic(pk []byte) (*C.EVP_PKEY, error) {
	p := C.kpi_kem_load_public(k.code, ptr(pk), C.size_t(len(pk)))
	if p == nil {
		return nil, fmt.Errorf("%s: load public key: %w", k.a.ID, errOp)
	}
	return p, nil
}

func (k *kem) loadPrivate(sk, pk []byte) (*C.EVP_PKEY, error) {
	p := C.kpi_kem_load_private(k.code, ptr(sk), C.size_t(len(sk)), ptr(pk), C.size_t(len(pk)))
	if p == nil {
		return nil, fmt.Errorf("%s: load private key: %w", k.a.ID, errOp)
	}
	return p, nil
}

func (k *kem) encapWith(pkR *C.EVP_PKEY, pkRm []byte) ([]byte, []byte, error) {
	enc := make([]byte, k.sizes.Ciphertext)
	ss := make([]byte, 32)
	n := C.size_t(len(enc))
	if C.kpi_kem_encap(k.code, pkR, ptr(pkRm), C.size_t(len(pkRm)), ptr(enc), &n, ptr(ss)) != 1 {
		return nil, nil, fmt.Errorf("%s encaps: %w", k.a.ID, errOp)
	}
	return enc[:n], ss, nil
}

func (k *kem) decapWith(skR *C.EVP_PKEY, pkRm, enc []byte) ([]byte, error) {
	ss := make([]byte, 32)
	if C.kpi_kem_decap(k.code, skR, ptr(pkRm), C.size_t(len(pkRm)), ptr(enc), C.size_t(len(enc)), ptr(ss)) != 1 {
		return nil, fmt.Errorf("%s decaps: %w", k.a.ID, errOp)
	}
	return ss, nil
}

func (k *kem) Encaps(pk []byte) ([]byte, []byte, error) {
	p, err := k.loadPublic(pk)
	if err != nil {
		return nil, nil, err
	}
	defer C.kpi_key_free(p)
	return k.encapWith(p, pk)
}

func (k *kem) Decaps(sk, pk, ct []byte) ([]byte, error) {
	p, err := k.loadPrivate(sk, pk)
	if err != nil {
		return nil, err
	}
	defer C.kpi_key_free(p)
	return k.decapWith(p, pk, ct)
}

// encapsWithEphemeral is the RFC 9180 test-vector path (never timed).
func (k *kem) encapsWithEphemeral(pkR, skEm, pkEm []byte) ([]byte, []byte, error) {
	p, err := k.loadPublic(pkR)
	if err != nil {
		return nil, nil, err
	}
	defer C.kpi_key_free(p)
	enc := make([]byte, k.sizes.Ciphertext)
	ss := make([]byte, 32)
	n := C.size_t(len(enc))
	if C.kpi_kem_encap_with_ephemeral(k.code, p, ptr(pkR), C.size_t(len(pkR)),
		ptr(skEm), C.size_t(len(skEm)), ptr(pkEm), C.size_t(len(pkEm)), ptr(enc), &n, ptr(ss)) != 1 {
		return nil, nil, fmt.Errorf("%s encaps (test vector): %w", k.a.ID, errOp)
	}
	return enc[:n], ss, nil
}

type kemKeygen struct{ k *kem }

func (k *kem) NewKeygenReuse() (scheme.KeygenReuse, error) { return &kemKeygen{k: k}, nil }
func (r *kemKeygen) Keygen() ([]byte, []byte, error)        { return r.k.GenerateKey() }
func (r *kemKeygen) Close()                                 {}

type encapsulator struct {
	k    *kem
	pkR  *C.EVP_PKEY
	pkRm []byte
}

func (k *kem) NewEncapsulatorReuse(pk []byte) (scheme.EncapsulatorReuse, error) {
	p, err := k.loadPublic(pk)
	if err != nil {
		return nil, err
	}
	return &encapsulator{k: k, pkR: p, pkRm: bytes.Clone(pk)}, nil
}
func (r *encapsulator) Encaps() ([]byte, []byte, error) { return r.k.encapWith(r.pkR, r.pkRm) }
func (r *encapsulator) Close() {
	if r.pkR != nil {
		C.kpi_key_free(r.pkR)
		r.pkR = nil
	}
}

type decapsulator struct {
	k    *kem
	skR  *C.EVP_PKEY
	pkRm []byte
}

func (k *kem) NewDecapsulatorReuse(sk, pk []byte) (scheme.DecapsulatorReuse, error) {
	p, err := k.loadPrivate(sk, pk)
	if err != nil {
		return nil, err
	}
	return &decapsulator{k: k, skR: p, pkRm: bytes.Clone(pk)}, nil
}
func (r *decapsulator) Decaps(ct []byte) ([]byte, error) { return r.k.decapWith(r.skR, r.pkRm, ct) }
func (r *decapsulator) Close() {
	if r.skR != nil {
		C.kpi_key_free(r.skR)
		r.skR = nil
	}
}
