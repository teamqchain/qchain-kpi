package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// TestE9Blocks: every route of every kind of operation runs, checks its
// calls, and reports its quantities; the stack of a lattice signature is
// measured, the production pattern's liboqs-go Init leak (two C strings per
// call) shows as retained C heap, and the reuse arm and the direct C call
// retain nothing.
func TestE9Blocks(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("E9 runs in the container only (LD_PRELOAD heap counter, /proc memory counters)")
	}
	dir := t.TempDir()
	out := t.TempDir()
	for _, a := range []string{"ml-dsa-44", "ml-kem-768", "ecdsa-p256", "dhkem-x25519", "falcon-512"} {
		prep(t, dir, a, 1, 8, false)
	}
	cases := []struct{ alg, op string }{
		{"ml-dsa-44", "keygen"}, {"ml-dsa-44", "sign"}, {"ml-dsa-44", "verify"},
		{"ml-kem-768", "keygen"}, {"ml-kem-768", "encaps"}, {"ml-kem-768", "decaps"},
		{"ecdsa-p256", "sign"}, {"ecdsa-p256", "verify"}, {"dhkem-x25519", "decaps"}, {"falcon-512", "sign"},
	}
	p50 := map[string]float64{}
	for _, c := range cases {
		for _, route := range proto.E9Arms {
			op := proto.E9Op(c.op, route)
			b := plan.Block{ID: "E9." + c.alg + "." + op + ".r1", Exp: "E9", Alg: c.alg, Op: op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileSmoke, Mode: "memory", Seed: 9}
			raw := filepath.Join(out, b.RawPath())
			r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
			if err != nil {
				t.Fatalf("%s: %v", b.Key(), err)
			}
			want := proto.E9Quantities(route)
			if r.Budget.Mode != "memory" || len(r.Arms) != len(want) || r.Rows < 2 {
				t.Fatalf("%s: budget %+v, %d rows, arms %+v", b.Key(), r.Budget, r.Rows, r.Arms)
			}
			for k, a := range r.Arms {
				if a.Name != want[k] {
					t.Fatalf("%s: quantity %d is %q, want %q", b.Key(), k, a.Name, want[k])
				}
				if !proto.E9PerBlock(a.Name) && (a.FirstLabel != "first" || a.Stats.N != r.Rows-1) {
					t.Fatalf("%s/%s: %+v", b.Key(), a.Name, a)
				}
				p50[c.alg+"/"+op+"/"+a.Name] = a.Stats.P50
			}
			h, rows := rawRows(t, raw)
			if h["route"] != route || h["operation"] != c.op || len(rows) != r.Rows || !strings.Contains(h["heap_counter"], "sha256 ") {
				t.Fatalf("%s: raw header %v, %d rows", b.Key(), h, len(rows))
			}
			if (route == proto.ArmC) != (h["stack_baseline"] != "") {
				t.Fatalf("%s: stack baseline %q", b.Key(), h["stack_baseline"])
			}
		}
	}
	t.Log(p50)
	// ML-DSA-44 signing keeps its matrix and vectors on the stack: tens of KiB.
	if s := p50["ml-dsa-44/sign-c/stack"]; s < 8<<10 || s > 1<<20 {
		t.Fatalf("ml-dsa-44 sign stack %v bytes", s)
	}
	if s := p50["falcon-512/sign-c/stack"]; s < 1<<10 {
		t.Fatalf("falcon-512 sign stack %v bytes", s)
	}
	// Init leaks C.CString(algorithm name) twice: "ML-DSA-44\x00" is 10 bytes,
	// at least 24 usable each with glibc's allocator.
	if r := p50["ml-dsa-44/sign-prod/heap-retained"]; r < 20 || r > 128 {
		t.Fatalf("ml-dsa-44 prod sign retains %v bytes per call", r)
	}
	for _, k := range []string{"ml-dsa-44/sign-prod/heap-unfreed", "ml-kem-768/decaps-prod/heap-unfreed"} {
		if p50[k] != 2 {
			t.Fatalf("%s = %v, want the two leaked strings", k, p50[k])
		}
	}
	for _, k := range []string{"ml-dsa-44/sign-reuse/heap-retained", "ml-dsa-44/sign-c/heap-retained", "ml-kem-768/decaps-c/heap-retained",
		"ml-dsa-44/sign-reuse/heap-unfreed", "ml-dsa-44/sign-c/heap-unfreed", "ecdsa-p256/sign-prod/heap-unfreed"} {
		if p50[k] != 0 {
			t.Fatalf("%s = %v", k, p50[k])
		}
	}
	// The production pattern allocates Go memory (hex strings); the direct C
	// route reports none.
	if p50["ml-dsa-44/sign-prod/go-bytes"] <= p50["ml-dsa-44/sign-reuse/go-bytes"] {
		t.Fatalf("prod Go heap %v, reuse %v", p50["ml-dsa-44/sign-prod/go-bytes"], p50["ml-dsa-44/sign-reuse/go-bytes"])
	}
}
