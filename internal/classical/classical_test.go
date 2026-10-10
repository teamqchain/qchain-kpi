package classical

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"qchain-kpi/internal/algs"
)

type dhkemVector struct {
	KemID        int    `json:"kem_id"`
	SkEm         string `json:"skEm"`
	PkEm         string `json:"pkEm"`
	SkRm         string `json:"skRm"`
	PkRm         string `json:"pkRm"`
	Enc          string `json:"enc"`
	SharedSecret string `json:"shared_secret"`
}

func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestRFC9180Vectors checks DHKEM encapsulation (with the vector's ephemeral
// key injected) and decapsulation against all 8 official vectors.
func TestRFC9180Vectors(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/rfc9180/dhkem-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Vectors []dhkemVector `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Vectors) != 8 {
		t.Fatalf("%d vectors, want 8", len(file.Vectors))
	}
	for i, v := range file.Vectors {
		id := map[int]string{16: "dhkem-p256", 32: "dhkem-x25519"}[v.KemID]
		s, err := NewKEM(algs.MustByID(id))
		if err != nil {
			t.Fatal(err)
		}
		k := s.(*kem)
		enc, ss, err := k.encapsWithEphemeral(unhex(t, v.PkRm), unhex(t, v.SkEm), unhex(t, v.PkEm))
		if err != nil {
			t.Fatalf("vector %d (%s) encap: %v", i, id, err)
		}
		if !bytes.Equal(enc, unhex(t, v.Enc)) {
			t.Fatalf("vector %d (%s): enc mismatch", i, id)
		}
		if !bytes.Equal(ss, unhex(t, v.SharedSecret)) {
			t.Fatalf("vector %d (%s): encap shared secret mismatch", i, id)
		}
		ss2, err := k.Decaps(unhex(t, v.SkRm), unhex(t, v.PkRm), unhex(t, v.Enc))
		if err != nil {
			t.Fatalf("vector %d (%s) decap: %v", i, id, err)
		}
		if !bytes.Equal(ss2, unhex(t, v.SharedSecret)) {
			t.Fatalf("vector %d (%s): decap shared secret mismatch", i, id)
		}
	}
}

func TestSignaturesRoundTrip(t *testing.T) {
	msg := []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")
	for _, id := range []string{"ecdsa-p256", "ed25519", "rsa3072-pss"} {
		s, err := NewSignature(algs.MustByID(id))
		if err != nil {
			t.Fatal(err)
		}
		pk, sk, err := s.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		if !s.Sizes().KeyLengthsOK(pk, sk) {
			t.Fatalf("%s key lengths %d/%d vs %+v", id, len(pk), len(sk), s.Sizes())
		}
		sg, err := s.Sign(sk, pk, msg)
		if err != nil {
			t.Fatalf("%s sign: %v", id, err)
		}
		ok, err := s.Verify(pk, msg, sg)
		if err != nil || !ok {
			t.Fatalf("%s verify: %v %v", id, ok, err)
		}
		// Reuse objects agree with the helpers.
		sr, err := s.NewSignerReuse(sk, pk)
		if err != nil {
			t.Fatal(err)
		}
		vr, err := s.NewVerifierReuse(pk)
		if err != nil {
			t.Fatal(err)
		}
		sg2, err := sr.Sign(msg)
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := vr.Verify(msg, sg2); err != nil || !ok {
			t.Fatalf("%s reuse verify: %v %v", id, ok, err)
		}
		bad := bytes.Clone(sg2)
		bad[len(bad)/2] ^= 0x01
		if ok, _ := vr.Verify(msg, bad); ok {
			t.Fatalf("%s accepted a corrupted signature", id)
		}
		sr.Close()
		vr.Close()
	}
}

func TestKEMsRoundTrip(t *testing.T) {
	for _, id := range []string{"dhkem-p256", "dhkem-x25519"} {
		k, err := NewKEM(algs.MustByID(id))
		if err != nil {
			t.Fatal(err)
		}
		pk, sk, err := k.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		if !k.Sizes().KeyLengthsOK(pk, sk) {
			t.Fatalf("%s key lengths %d/%d vs %+v", id, len(pk), len(sk), k.Sizes())
		}
		er, err := k.NewEncapsulatorReuse(pk)
		if err != nil {
			t.Fatal(err)
		}
		dr, err := k.NewDecapsulatorReuse(sk, pk)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20; i++ {
			ct, ss, err := er.Encaps()
			if err != nil {
				t.Fatal(err)
			}
			ss2, err := dr.Decaps(ct)
			if err != nil || !bytes.Equal(ss, ss2) {
				t.Fatalf("%s round trip %d: %v", id, i, err)
			}
		}
		er.Close()
		dr.Close()
	}
}
