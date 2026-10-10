// Package pqc adapts liboqs-go v0.16.0 to the scheme interfaces.
//
// liboqs-go traps handled here (verified in its v0.16.0 source):
//   - Init(alg, sk) keeps a reference to the caller's sk slice and Clean()
//     zeroes it (MemCleanse), and ExportSecretKey() returns that internal
//     slice. Every key handed to Init is therefore a private copy, and every
//     exported key is copied before Clean runs.
//   - Verify returns (false, nil) for a wrong signature and an error only for
//     a wrong length.
//   - Sign returns a slice whose capacity is the maximum signature length.
package pqc

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/open-quantum-safe/liboqs-go/oqs"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/scheme"
)

// Version returns the liboqs version the binary is linked against.
func Version() string { return oqs.LiboqsVersion() }

// CheckLibrary verifies the pinned liboqs version and that all 25 post-quantum
// algorithms of the study are enabled in this build.
func CheckLibrary() error {
	if v := oqs.LiboqsVersion(); v != buildinfo.ExpectedLiboqsVersion {
		return fmt.Errorf("liboqs version %q, expected %q", v, buildinfo.ExpectedLiboqsVersion)
	}
	var missing []string
	for _, n := range algs.OQSNames(algs.Sig) {
		if !oqs.IsSigEnabled(n) {
			missing = append(missing, n)
		}
	}
	for _, n := range algs.OQSNames(algs.KEM) {
		if !oqs.IsKEMEnabled(n) {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("liboqs build lacks %v", missing)
	}
	return nil
}

// EnabledAlgorithms lists everything this liboqs build enables (for E0).
func EnabledAlgorithms() (sigs, kems []string) {
	return oqs.EnabledSigs(), oqs.EnabledKEMs()
}

// ---------------------------------------------------------------- signatures

type sig struct {
	a     algs.Alg
	sizes scheme.Sizes
}

// NewSignature returns the liboqs implementation of a post-quantum signature.
func NewSignature(a algs.Alg) (scheme.Signature, error) {
	if a.Track != algs.Sig || a.Classical() {
		return nil, fmt.Errorf("pqc: %s is not a post-quantum signature", a.ID)
	}
	var o oqs.Signature
	if err := o.Init(a.OQSName, nil); err != nil {
		return nil, err
	}
	d := o.Details()
	o.Clean()
	return &sig{a: a, sizes: scheme.Sizes{
		PublicKey: d.LengthPublicKey, SecretKey: d.LengthSecretKey, Signature: d.MaxLengthSignature,
		// Falcon (unpadded) signatures vary in length; every other liboqs
		// signature in the study has a fixed length.
		SigVariable: a.Family == "Falcon" && (a.ID == "falcon-512" || a.ID == "falcon-1024"),
	}}, nil
}

func (s *sig) Alg() algs.Alg       { return s.a }
func (s *sig) Sizes() scheme.Sizes { return s.sizes }

func (s *sig) GenerateKey() (pk, sk []byte, err error) {
	var o oqs.Signature
	if err := o.Init(s.a.OQSName, nil); err != nil {
		return nil, nil, err
	}
	defer o.Clean()
	pub, err := o.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	// Copies are taken before the deferred Clean zeroes the internal key.
	return bytes.Clone(pub), bytes.Clone(o.ExportSecretKey()), nil
}

func (s *sig) Sign(sk, _ []byte, msg []byte) ([]byte, error) {
	var o oqs.Signature
	if err := o.Init(s.a.OQSName, bytes.Clone(sk)); err != nil {
		return nil, err
	}
	defer o.Clean()
	out, err := o.Sign(msg)
	if err != nil {
		return nil, err
	}
	return bytes.Clone(out), nil
}

func (s *sig) Verify(pk, msg, sg []byte) (bool, error) {
	var o oqs.Signature
	if err := o.Init(s.a.OQSName, nil); err != nil {
		return false, err
	}
	defer o.Clean()
	return o.Verify(msg, sg, pk)
}

type sigKeygen struct{ o oqs.Signature }

func (s *sig) NewKeygenReuse() (scheme.KeygenReuse, error) {
	r := &sigKeygen{}
	if err := r.o.Init(s.a.OQSName, nil); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *sigKeygen) Keygen() ([]byte, []byte, error) {
	pub, err := r.o.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return pub, r.o.ExportSecretKey(), nil
}
func (r *sigKeygen) Close() { r.o.Clean() }

type signer struct{ o oqs.Signature }

func (s *sig) NewSignerReuse(sk, _ []byte) (scheme.SignerReuse, error) {
	r := &signer{}
	if err := r.o.Init(s.a.OQSName, bytes.Clone(sk)); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *signer) Sign(msg []byte) ([]byte, error) { return r.o.Sign(msg) }
func (r *signer) Close()                            { r.o.Clean() }

type verifier struct {
	o  oqs.Signature
	pk []byte
}

func (s *sig) NewVerifierReuse(pk []byte) (scheme.VerifierReuse, error) {
	r := &verifier{pk: bytes.Clone(pk)}
	if err := r.o.Init(s.a.OQSName, nil); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *verifier) Verify(msg, sg []byte) (bool, error) { return r.o.Verify(msg, sg, r.pk) }
func (r *verifier) Close()                               { r.o.Clean() }

// ---------------------------------------------------------------------- KEMs

type kem struct {
	a     algs.Alg
	sizes scheme.Sizes
}

// NewKEM returns the liboqs implementation of a post-quantum KEM.
func NewKEM(a algs.Alg) (scheme.KEM, error) {
	if a.Track != algs.KEM || a.Classical() {
		return nil, fmt.Errorf("pqc: %s is not a post-quantum KEM", a.ID)
	}
	var o oqs.KeyEncapsulation
	if err := o.Init(a.OQSName, nil); err != nil {
		return nil, err
	}
	d := o.Details()
	o.Clean()
	return &kem{a: a, sizes: scheme.Sizes{
		PublicKey: d.LengthPublicKey, SecretKey: d.LengthSecretKey,
		Ciphertext: d.LengthCiphertext, SharedSecret: d.LengthSharedSecret,
	}}, nil
}

func (k *kem) Alg() algs.Alg       { return k.a }
func (k *kem) Sizes() scheme.Sizes { return k.sizes }

func (k *kem) GenerateKey() (pk, sk []byte, err error) {
	var o oqs.KeyEncapsulation
	if err := o.Init(k.a.OQSName, nil); err != nil {
		return nil, nil, err
	}
	defer o.Clean()
	pub, err := o.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return bytes.Clone(pub), bytes.Clone(o.ExportSecretKey()), nil
}

func (k *kem) Encaps(pk []byte) (ct, ss []byte, err error) {
	var o oqs.KeyEncapsulation
	if err := o.Init(k.a.OQSName, nil); err != nil {
		return nil, nil, err
	}
	defer o.Clean()
	c, s, err := o.EncapSecret(pk)
	if err != nil {
		return nil, nil, err
	}
	return bytes.Clone(c), bytes.Clone(s), nil
}

func (k *kem) Decaps(sk, _ []byte, ct []byte) ([]byte, error) {
	var o oqs.KeyEncapsulation
	if err := o.Init(k.a.OQSName, bytes.Clone(sk)); err != nil {
		return nil, err
	}
	defer o.Clean()
	s, err := o.DecapSecret(ct)
	if err != nil {
		return nil, err
	}
	return bytes.Clone(s), nil
}

type kemKeygen struct{ o oqs.KeyEncapsulation }

func (k *kem) NewKeygenReuse() (scheme.KeygenReuse, error) {
	r := &kemKeygen{}
	if err := r.o.Init(k.a.OQSName, nil); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *kemKeygen) Keygen() ([]byte, []byte, error) {
	pub, err := r.o.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return pub, r.o.ExportSecretKey(), nil
}
func (r *kemKeygen) Close() { r.o.Clean() }

type encapsulator struct {
	o  oqs.KeyEncapsulation
	pk []byte
}

func (k *kem) NewEncapsulatorReuse(pk []byte) (scheme.EncapsulatorReuse, error) {
	r := &encapsulator{pk: bytes.Clone(pk)}
	if err := r.o.Init(k.a.OQSName, nil); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *encapsulator) Encaps() ([]byte, []byte, error) { return r.o.EncapSecret(r.pk) }
func (r *encapsulator) Close()                          { r.o.Clean() }

type decapsulator struct{ o oqs.KeyEncapsulation }

func (k *kem) NewDecapsulatorReuse(sk, _ []byte) (scheme.DecapsulatorReuse, error) {
	r := &decapsulator{}
	if err := r.o.Init(k.a.OQSName, bytes.Clone(sk)); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *decapsulator) Decaps(ct []byte) ([]byte, error) { return r.o.DecapSecret(ct) }
func (r *decapsulator) Close()                           { r.o.Clean() }

// ErrNotPQC is returned for classical algorithms.
var ErrNotPQC = errors.New("not a post-quantum algorithm")
