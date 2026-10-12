package main

import (
	"path/filepath"
	"strings"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// TestE8Blocks: every item length is recorded and checked, fixed lengths
// match the library, and variable ones stay within their limits.
func TestE8Blocks(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	cases := []struct {
		alg, op string
		fixed   map[string]int // item -> exact length (0 = varies)
	}{
		{"ml-dsa-44", "keygen", map[string]int{"pk": 1312, "sk": 2560}},
		{"ml-dsa-44", "sign", map[string]int{"sig": 2420}},
		{"falcon-512", "sign", map[string]int{"sig": 0}},
		{"ecdsa-p256", "sign", map[string]int{"sig": 0}},
		{"rsa3072-pss", "keygen", map[string]int{"pk": 422, "sk": 0}},
		{"ml-kem-768", "encaps", map[string]int{"ct": 1088, "ss": 32}},
		{"dhkem-x25519", "keygen", map[string]int{"pk": 32, "sk": 32}},
		{"slh-dsa-sha2-128f", "sign", map[string]int{"sig": 17088}},
	}
	for _, c := range cases {
		b := plan.Block{ID: "E8." + c.alg + "." + c.op + ".r1", Exp: "E8", Alg: c.alg, Op: c.op, Rep: 1, Attempt: 1,
			Profile: plan.ProfileSmoke, Mode: "sizes", Seed: 5}
		raw := filepath.Join(out, b.RawPath())
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		want := proto.E8Arms(algs.MustByID(c.alg), c.op)
		if len(r.Arms) != len(want) || r.Budget.Mode != "sizes" {
			t.Fatalf("%s: arms %+v", b.Key(), r.Arms)
		}
		n := r.Arms[0].Stats.N
		if n < 5 {
			t.Fatalf("%s: only %d items", b.Key(), n)
		}
		for k, a := range r.Arms {
			if a.Name != want[k] || a.Stats.N != n {
				t.Fatalf("%s: arm %+v", b.Key(), a)
			}
			if f := c.fixed[a.Name]; f > 0 && (a.MinBytes != f || a.MaxBytes != f) {
				t.Fatalf("%s/%s: lengths %d..%d, want %d", b.Key(), a.Name, a.MinBytes, a.MaxBytes, f)
			}
			if a.MinBytes <= 0 || a.MaxBytes < a.MinBytes {
				t.Fatalf("%s/%s: lengths %d..%d", b.Key(), a.Name, a.MinBytes, a.MaxBytes)
			}
		}
		if c.alg == "falcon-512" && r.Arms[0].MaxBytes > 752 {
			t.Fatalf("falcon-512 signature of %d bytes", r.Arms[0].MaxBytes)
		}
		if c.alg == "ecdsa-p256" && (r.Arms[0].MinBytes < 8 || r.Arms[0].MaxBytes > 72) {
			t.Fatalf("ECDSA DER signature lengths %d..%d", r.Arms[0].MinBytes, r.Arms[0].MaxBytes)
		}
		if c.op == "keygen" && r.KeysChecked < 3 {
			t.Fatalf("%s: %d keys fully checked", b.Key(), r.KeysChecked)
		}
		h, rows := rawRows(t, raw)
		if len(rows) != n*len(want) || r.Rows != len(rows) || !strings.HasPrefix(rows[0], want[0]+",") {
			t.Fatalf("%s: %d raw rows (first %q), want %d", b.Key(), len(rows), rows[0], n*len(want))
		}
		if !strings.Contains(h["encoding"], "2 x bytes") || h["key"] != b.Key() {
			t.Fatalf("%s: raw header %v", b.Key(), h)
		}
		if got, _ := fileSHA256(raw); got != r.RawSHA256 {
			t.Fatalf("%s: raw checksum mismatch", b.Key())
		}
	}
	b := plan.Block{ID: "E8.ml-dsa-44.verify.r1", Exp: "E8", Alg: "ml-dsa-44", Op: "verify", Rep: 1, Attempt: 1, Profile: plan.ProfileSmoke, Seed: 5}
	if _, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())}); err == nil {
		t.Fatal("E8 verify must be refused")
	}
}
