package gostd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/classical"
)

var msg = []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")

func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestRFC9180Vectors: Go's DHKEM reproduces all 8 official vectors (the
// vector's ephemeral key injected for encapsulation).
func TestRFC9180Vectors(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/rfc9180/dhkem-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Vectors []struct {
			KemID        int    `json:"kem_id"`
			SkEm         string `json:"skEm"`
			PkEm         string `json:"pkEm"`
			SkRm         string `json:"skRm"`
			PkRm         string `json:"pkRm"`
			Enc          string `json:"enc"`
			SharedSecret string `json:"shared_secret"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Vectors) != 8 {
		t.Fatalf("%d vectors, want 8", len(file.Vectors))
	}
	for i, v := range file.Vectors {
		a := algs.MustByID(map[int]string{16: "dhkem-p256", 32: "dhkem-x25519"}[v.KemID])
		k, err := kemOf(a)
		if err != nil {
			t.Fatal(err)
		}
		skE, err := k.curve.NewPrivateKey(unhex(t, v.SkEm))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(skE.PublicKey().Bytes(), unhex(t, v.PkEm)) {
			t.Fatalf("vector %d: ephemeral public key mismatch", i)
		}
		pkRm := unhex(t, v.PkRm)
		pkR, err := k.curve.NewPublicKey(pkRm)
		if err != nil {
			t.Fatal(err)
		}
		enc, ss, err := k.encapWith(skE, pkR, pkRm)
		if err != nil || !bytes.Equal(enc, unhex(t, v.Enc)) || !bytes.Equal(ss, unhex(t, v.SharedSecret)) {
			t.Fatalf("vector %d (%s): encapsulation differs (%v)", i, a.ID, err)
		}
		dec, err := NewDecapsulator(a, unhex(t, v.SkRm), pkRm)
		if err != nil {
			t.Fatal(err)
		}
		if ss2, err := dec(unhex(t, v.Enc)); err != nil || !bytes.Equal(ss2, unhex(t, v.SharedSecret)) {
			t.Fatalf("vector %d (%s): decapsulation differs (%v)", i, a.ID, err)
		}
	}
}

// TestSignaturesInteroperate: for each classical signature, keys made by
// either library work in the other, and signatures made by either verify in
// the other (so both measure the same scheme and formats).
func TestSignaturesInteroperate(t *testing.T) {
	for _, id := range []string{"ecdsa-p256", "ed25519", "rsa3072-pss"} {
		a := algs.MustByID(id)
		ossl, err := classical.NewSignature(a)
		if err != nil {
			t.Fatal(err)
		}
		gpk, gsk, err := SigKeygen(a)
		if err != nil {
			t.Fatal(err)
		}
		if !ossl.Sizes().KeyLengthsOK(gpk, gsk) {
			t.Fatalf("%s: Go key lengths %d/%d differ from OpenSSL's format", id, len(gpk), len(gsk))
		}
		opk, osk, err := ossl.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		for _, kp := range [][2][]byte{{gpk, gsk}, {opk, osk}} {
			pk, sk := kp[0], kp[1]
			sign, err := NewSigner(a, sk, pk)
			if err != nil {
				t.Fatalf("%s: Go signer: %v", id, err)
			}
			verify, err := NewVerifier(a, pk)
			if err != nil {
				t.Fatalf("%s: Go verifier: %v", id, err)
			}
			gsig, err := sign(msg)
			if err != nil {
				t.Fatal(err)
			}
			if ok, err := ossl.Verify(pk, msg, gsig); err != nil || !ok {
				t.Fatalf("%s: OpenSSL rejects Go's signature (%v)", id, err)
			}
			osig, err := ossl.Sign(sk, pk, msg)
			if err != nil {
				t.Fatal(err)
			}
			if ok, _ := verify(msg, osig); !ok {
				t.Fatalf("%s: Go rejects OpenSSL's signature", id)
			}
			bad := bytes.Clone(osig)
			bad[len(bad)/2] ^= 1
			if ok, err := verify(msg, bad); ok || err != nil {
				t.Fatalf("%s: Go accepts a corrupted signature (%v)", id, err)
			}
		}
	}
}

// TestKEMsInteroperate: Go encapsulates to OpenSSL keys and OpenSSL
// decapsulates, and the other way round, for both DHKEMs; a ciphertext that
// is not a valid point is refused.
func TestKEMsInteroperate(t *testing.T) {
	for _, id := range []string{"dhkem-p256", "dhkem-x25519"} {
		a := algs.MustByID(id)
		ossl, err := classical.NewKEM(a)
		if err != nil {
			t.Fatal(err)
		}
		gpk, gsk, err := KEMKeygen(a)
		if err != nil {
			t.Fatal(err)
		}
		if !ossl.Sizes().KeyLengthsOK(gpk, gsk) {
			t.Fatalf("%s: Go key lengths %d/%d", id, len(gpk), len(gsk))
		}
		opk, osk, err := ossl.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		for _, kp := range [][2][]byte{{gpk, gsk}, {opk, osk}} {
			pk, sk := kp[0], kp[1]
			enc, err := NewEncapsulator(a, pk)
			if err != nil {
				t.Fatal(err)
			}
			dec, err := NewDecapsulator(a, sk, pk)
			if err != nil {
				t.Fatal(err)
			}
			ct, ss, err := enc()
			if err != nil || len(ct) != ossl.Sizes().Ciphertext || len(ss) != 32 {
				t.Fatalf("%s: Go encapsulation %d/%d bytes (%v)", id, len(ct), len(ss), err)
			}
			if ss2, err := ossl.Decaps(sk, pk, ct); err != nil || !bytes.Equal(ss, ss2) {
				t.Fatalf("%s: OpenSSL decapsulates Go's ciphertext differently (%v)", id, err)
			}
			oct, oss, err := ossl.Encaps(pk)
			if err != nil {
				t.Fatal(err)
			}
			if ss3, err := dec(oct); err != nil || !bytes.Equal(ss3, oss) {
				t.Fatalf("%s: Go decapsulates OpenSSL's ciphertext differently (%v)", id, err)
			}
		}
	}
	pk, sk, err := KEMKeygen(algs.MustByID("dhkem-p256"))
	if err != nil {
		t.Fatal(err)
	}
	dec, err := NewDecapsulator(algs.MustByID("dhkem-p256"), sk, pk)
	if err != nil {
		t.Fatal(err)
	}
	bad := make([]byte, 65)
	bad[0] = 4
	if _, err := dec(bad); err == nil {
		t.Fatal("a P-256 ciphertext off the curve was accepted")
	}
}
