// Package scheme is the common interface over the post-quantum (liboqs-go) and
// classical (OpenSSL via cgo) implementations, so every experiment measures
// both through the same code path.
//
// "Reuse" objects are the E1 arm: each library used the way its API is meant
// to be used repeatedly (a liboqs Signature/KeyEncapsulation object kept
// alive; an OpenSSL key object kept alive). Their methods are the ONLY work
// inside a timed region.
package scheme

import "qchain-kpi/internal/algs"

// Sizes are the byte lengths reported by the implementation.
type Sizes struct {
	PublicKey int `json:"public_key"`
	// SecretKey is 0 when SKVariable is set: an RSA PKCS#8 key's DER length
	// varies by a few bytes from key to key (E8 measures the distribution).
	SecretKey    int  `json:"secret_key"`
	SKVariable   bool `json:"secret_key_variable,omitempty"`
	Signature    int  `json:"signature_max,omitempty"`
	SigVariable  bool `json:"signature_variable,omitempty"`
	Ciphertext   int  `json:"ciphertext,omitempty"`
	SharedSecret int  `json:"shared_secret,omitempty"`
}

// KeyLengthsOK reports whether a generated key pair has the reported lengths.
func (s Sizes) KeyLengthsOK(pk, sk []byte) bool {
	if len(pk) != s.PublicKey {
		return false
	}
	if s.SKVariable {
		return len(sk) > 0
	}
	return len(sk) == s.SecretKey
}

// Signature is one signature algorithm.
type Signature interface {
	Alg() algs.Alg
	Sizes() Sizes

	// Untimed helpers (input preparation and correctness checks). Returned
	// slices are owned by the caller.
	GenerateKey() (pk, sk []byte, err error)
	// Sign takes the public key too: an ECDSA secret scalar is imported
	// together with its public point. liboqs ignores pk.
	Sign(sk, pk, msg []byte) ([]byte, error)
	Verify(pk, msg, sig []byte) (bool, error)

	// Reuse-arm objects for timing. Close releases native resources.
	NewKeygenReuse() (KeygenReuse, error)
	NewSignerReuse(sk, pk []byte) (SignerReuse, error)
	NewVerifierReuse(pk []byte) (VerifierReuse, error)
}

// KEM is one key-encapsulation mechanism.
type KEM interface {
	Alg() algs.Alg
	Sizes() Sizes

	GenerateKey() (pk, sk []byte, err error)
	Encaps(pk []byte) (ct, ss []byte, err error)
	Decaps(sk, pk, ct []byte) ([]byte, error)

	NewKeygenReuse() (KeygenReuse, error)
	NewEncapsulatorReuse(pk []byte) (EncapsulatorReuse, error)
	NewDecapsulatorReuse(sk, pk []byte) (DecapsulatorReuse, error)
}

// KeygenReuse generates key pairs with one long-lived object. The returned
// slices may alias internal buffers and are only valid until the next call.
type KeygenReuse interface {
	Keygen() (pk, sk []byte, err error)
	Close()
}

// SignerReuse signs with one long-lived key object.
type SignerReuse interface {
	Sign(msg []byte) ([]byte, error)
	Close()
}

// VerifierReuse verifies against one public key. For liboqs the raw public key
// is passed on every call (that is liboqs's API); for OpenSSL the parsed key
// object is kept.
type VerifierReuse interface {
	Verify(msg, sig []byte) (bool, error)
	Close()
}

// EncapsulatorReuse encapsulates to one recipient public key.
type EncapsulatorReuse interface {
	Encaps() (ct, ss []byte, err error)
	Close()
}

// DecapsulatorReuse decapsulates with one long-lived secret key object.
type DecapsulatorReuse interface {
	Decaps(ct []byte) ([]byte, error)
	Close()
}

// ---------------------------------------------------------------- E2 objects

// ObjKind is the kind of library object an operation needs.
type ObjKind int

const (
	Keyless ObjKind = iota // key generation
	Public                 // verify, encaps
	Private                // sign, decaps
)

// KindOf returns the object kind an operation needs.
func KindOf(op string) ObjKind {
	switch op {
	case "sign", "decaps":
		return Private
	case "verify", "encaps":
		return Public
	}
	return Keyless
}

// SigObject is one reusable slot for a library object of a signature
// algorithm (E2): a liboqs Signature, or an OpenSSL key object. It is
// allocated once, before timing; Open and Close create and destroy the library
// object inside it, so they can be timed without a Go allocation.
type SigObject interface {
	// Open creates the library object: liboqs Init, or loading the key into
	// an OpenSSL key object. For liboqs a Private object keeps the sk slice
	// it is given and Close zeroes it, so the caller passes a copy it does
	// not need afterwards. Open performs no cryptographic operation.
	Open(kind ObjKind, sk, pk []byte) error
	// Keygen returns the new key pair. For liboqs the secret key aliases the
	// object's internal buffer and is zeroed by Close: copy it first.
	Keygen() (pk, sk []byte, err error)
	Sign(msg []byte) ([]byte, error)
	Verify(msg, sig []byte) (bool, error)
	// Close destroys the library object (liboqs Clean, or freeing the OpenSSL
	// key object). The slot can be opened again.
	Close()
}

// KEMObject is the KEM counterpart of SigObject.
type KEMObject interface {
	Open(kind ObjKind, sk, pk []byte) error
	Keygen() (pk, sk []byte, err error)
	Encaps() (ct, ss []byte, err error)
	Decaps(ct []byte) ([]byte, error)
	Close()
}

// SigObjects is implemented by every signature algorithm (E2).
type SigObjects interface {
	NewSigObject() SigObject
}

// KEMObjects is implemented by every KEM (E2).
type KEMObjects interface {
	NewKEMObject() KEMObject
}
