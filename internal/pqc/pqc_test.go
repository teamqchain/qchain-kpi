package pqc

import (
	"bytes"
	"testing"

	"qchain-kpi/internal/algs"
)

func TestLibraryGate(t *testing.T) {
	if err := CheckLibrary(); err != nil {
		t.Fatal(err)
	}
}

// TestKeysSurviveClean guards the liboqs-go trap: Clean() zeroes the slice
// that was handed to Init, and ExportSecretKey aliases it.
func TestKeysSurviveClean(t *testing.T) {
	s, err := NewSignature(algs.MustByID("ml-dsa-44"))
	if err != nil {
		t.Fatal(err)
	}
	pk, sk, err := s.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	keep := bytes.Clone(sk)
	r, err := s.NewSignerReuse(sk, pk)
	if err != nil {
		t.Fatal(err)
	}
	r.Close() // zeroes the signer's private copy only
	if !bytes.Equal(sk, keep) {
		t.Fatal("caller's secret key was modified by Clean")
	}
	if bytes.Equal(sk, make([]byte, len(sk))) {
		t.Fatal("secret key is all zero")
	}
	msg := []byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	sg, err := s.Sign(sk, pk, msg)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.Verify(pk, msg, sg); err != nil || !ok {
		t.Fatalf("signature with the surviving key does not verify: %v %v", ok, err)
	}
}

func TestAllPQCRoundTrip(t *testing.T) {
	msg := []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")
	for _, a := range algs.All() {
		if a.Classical() {
			continue
		}
		a := a
		t.Run(a.ID, func(t *testing.T) {
			if a.Track == algs.Sig {
				s, err := NewSignature(a)
				if err != nil {
					t.Fatal(err)
				}
				pk, sk, err := s.GenerateKey()
				if err != nil {
					t.Fatal(err)
				}
				if len(pk) != s.Sizes().PublicKey || len(sk) != s.Sizes().SecretKey {
					t.Fatalf("key sizes %d/%d vs %+v", len(pk), len(sk), s.Sizes())
				}
				sr, _ := s.NewSignerReuse(sk, pk)
				vr, _ := s.NewVerifierReuse(pk)
				defer sr.Close()
				defer vr.Close()
				sg, err := sr.Sign(msg)
				if err != nil {
					t.Fatal(err)
				}
				if ok, err := vr.Verify(msg, sg); err != nil || !ok {
					t.Fatalf("verify: %v %v", ok, err)
				}
				bad := bytes.Clone(sg)
				bad[len(bad)/2] ^= 0x01
				if ok, _ := vr.Verify(msg, bad); ok {
					t.Fatal("corrupted signature accepted")
				}
				return
			}
			k, err := NewKEM(a)
			if err != nil {
				t.Fatal(err)
			}
			pk, sk, err := k.GenerateKey()
			if err != nil {
				t.Fatal(err)
			}
			er, _ := k.NewEncapsulatorReuse(pk)
			dr, _ := k.NewDecapsulatorReuse(sk, pk)
			defer er.Close()
			defer dr.Close()
			ct, ss, err := er.Encaps()
			if err != nil {
				t.Fatal(err)
			}
			ss2, err := dr.Decaps(ct)
			if err != nil || !bytes.Equal(ss, ss2) {
				t.Fatalf("decaps mismatch: %v", err)
			}
		})
	}
}
