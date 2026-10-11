package classical

/*
#include "ossl.h"
*/
import "C"

import "qchain-kpi/internal/scheme"

// E2 object slots for the classical algorithms: Open loads the key into an
// OpenSSL key object (EVP_PKEY), Close frees it. OpenSSL copies the key
// bytes, so nothing the caller passes is modified. A Keyless object has no
// OpenSSL object: OpenSSL's key generation creates a fresh key object on
// every call anyway.

type sigObject struct {
	s  *sig
	k  *C.EVP_PKEY
	pk []byte
}

// NewSigObject returns an empty slot (no library call).
func (s *sig) NewSigObject() scheme.SigObject { return &sigObject{s: s} }

func (x *sigObject) Open(kind scheme.ObjKind, sk, pk []byte) error {
	var err error
	switch kind {
	case scheme.Private:
		x.k, err = x.s.loadPrivate(sk, pk)
	case scheme.Public:
		x.k, err = x.s.loadPublic(pk)
	}
	x.pk = pk
	return err
}

func (x *sigObject) Keygen() ([]byte, []byte, error)     { return x.s.GenerateKey() }
func (x *sigObject) Sign(msg []byte) ([]byte, error)     { return x.s.signWith(x.k, msg) }
func (x *sigObject) Verify(msg, sg []byte) (bool, error) { return x.s.verifyWith(x.k, msg, sg) }

func (x *sigObject) Close() {
	if x.k != nil {
		C.kpi_key_free(x.k)
		x.k = nil
	}
}

type kemObject struct {
	k  *kem
	p  *C.EVP_PKEY
	pk []byte
}

// NewKEMObject returns an empty slot (no library call).
func (k *kem) NewKEMObject() scheme.KEMObject { return &kemObject{k: k} }

func (x *kemObject) Open(kind scheme.ObjKind, sk, pk []byte) error {
	var err error
	switch kind {
	case scheme.Private:
		x.p, err = x.k.loadPrivate(sk, pk)
	case scheme.Public:
		x.p, err = x.k.loadPublic(pk)
	}
	x.pk = pk
	return err
}

func (x *kemObject) Keygen() ([]byte, []byte, error)  { return x.k.GenerateKey() }
func (x *kemObject) Encaps() ([]byte, []byte, error)  { return x.k.encapWith(x.p, x.pk) }
func (x *kemObject) Decaps(ct []byte) ([]byte, error) { return x.k.decapWith(x.p, x.pk, ct) }

func (x *kemObject) Close() {
	if x.p != nil {
		C.kpi_key_free(x.p)
		x.p = nil
	}
}
