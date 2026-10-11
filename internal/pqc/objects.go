package pqc

import (
	"github.com/open-quantum-safe/liboqs-go/oqs"

	"qchain-kpi/internal/scheme"
)

// E2 object slots: Open is liboqs Init, Close is liboqs Clean, exactly the
// calls QChain's backend makes around every operation. A Public object keeps
// a reference to the public key (liboqs takes it per call); a Private object
// is handed its secret key at Init, which liboqs-go keeps and Clean zeroes.

type sigObject struct {
	name string
	o    oqs.Signature
	pk   []byte
	open bool
}

// NewSigObject returns an empty slot (no library call).
func (s *sig) NewSigObject() scheme.SigObject { return &sigObject{name: s.a.OQSName} }

func (x *sigObject) Open(kind scheme.ObjKind, sk, pk []byte) error {
	var key []byte
	if kind == scheme.Private {
		key = sk
	}
	if err := x.o.Init(x.name, key); err != nil {
		return err
	}
	x.pk, x.open = pk, true
	return nil
}

func (x *sigObject) Keygen() ([]byte, []byte, error) {
	pub, err := x.o.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return pub, x.o.ExportSecretKey(), nil
}

func (x *sigObject) Sign(msg []byte) ([]byte, error) { return x.o.Sign(msg) }

func (x *sigObject) Verify(msg, sg []byte) (bool, error) { return x.o.Verify(msg, sg, x.pk) }

func (x *sigObject) Close() {
	if x.open {
		x.o.Clean()
		x.open = false
	}
}

type kemObject struct {
	name string
	o    oqs.KeyEncapsulation
	pk   []byte
	open bool
}

// NewKEMObject returns an empty slot (no library call).
func (k *kem) NewKEMObject() scheme.KEMObject { return &kemObject{name: k.a.OQSName} }

func (x *kemObject) Open(kind scheme.ObjKind, sk, pk []byte) error {
	var key []byte
	if kind == scheme.Private {
		key = sk
	}
	if err := x.o.Init(x.name, key); err != nil {
		return err
	}
	x.pk, x.open = pk, true
	return nil
}

func (x *kemObject) Keygen() ([]byte, []byte, error) {
	pub, err := x.o.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return pub, x.o.ExportSecretKey(), nil
}

func (x *kemObject) Encaps() ([]byte, []byte, error) { return x.o.EncapSecret(x.pk) }

func (x *kemObject) Decaps(ct []byte) ([]byte, error) { return x.o.DecapSecret(ct) }

func (x *kemObject) Close() {
	if x.open {
		x.o.Clean()
		x.open = false
	}
}
