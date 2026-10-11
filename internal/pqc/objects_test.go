package pqc

import (
	"bytes"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/scheme"
)

// TestObjectsReopenAndZero checks the E2 object slots: a slot can be opened
// and closed repeatedly, Close zeroes the secret key slice handed to Open
// (which is why E2 gives every Open its own copy), and the secret key that
// Keygen returns is zeroed by Close unless copied first.
func TestObjectsReopenAndZero(t *testing.T) {
	a := algs.MustByID("ml-dsa-44")
	s, err := NewSignature(a)
	if err != nil {
		t.Fatal(err)
	}
	pk, sk, err := s.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	x := s.(scheme.SigObjects).NewSigObject()
	msg := []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")
	for i := 0; i < 3; i++ {
		key := bytes.Clone(sk)
		if err := x.Open(scheme.Private, key, pk); err != nil {
			t.Fatal(err)
		}
		sg, err := x.Sign(msg)
		if err != nil {
			t.Fatal(err)
		}
		x.Close()
		if !bytes.Equal(key, make([]byte, len(key))) {
			t.Fatal("Close did not zero the key handed to Open")
		}
		if ok, err := s.Verify(pk, msg, sg); err != nil || !ok {
			t.Fatalf("signature from a reopened slot does not verify (%v)", err)
		}
	}
	if err := x.Open(scheme.Keyless, nil, nil); err != nil {
		t.Fatal(err)
	}
	_, alias, err := x.Keygen()
	if err != nil {
		t.Fatal(err)
	}
	kept := bytes.Clone(alias)
	x.Close()
	if !bytes.Equal(alias, make([]byte, len(alias))) || bytes.Equal(kept, alias) {
		t.Fatal("expected Close to zero the secret key returned by Keygen")
	}

	k, err := NewKEM(algs.MustByID("ml-kem-768"))
	if err != nil {
		t.Fatal(err)
	}
	kpk, ksk, err := k.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ko := k.(scheme.KEMObjects).NewKEMObject()
	if err := ko.Open(scheme.Public, nil, kpk); err != nil {
		t.Fatal(err)
	}
	ct, ss, err := ko.Encaps()
	ko.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := ko.Open(scheme.Private, bytes.Clone(ksk), kpk); err != nil {
		t.Fatal(err)
	}
	ss2, err := ko.Decaps(ct)
	ko.Close()
	if err != nil || !bytes.Equal(ss, ss2) {
		t.Fatalf("KEM slot round trip failed (%v)", err)
	}
}
