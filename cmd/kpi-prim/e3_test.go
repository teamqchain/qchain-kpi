package main

import (
	"path/filepath"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// TestE3Blocks: every layer of a post-quantum and a classical signature and
// KEM runs and checks every call, the raw-C layer is not slower than the Go
// layers that wrap it, and the null block reports a cgo call and the C
// timer's floor.
func TestE3Blocks(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	cases := []struct{ alg, op string }{
		{"ml-dsa-44", "sign"}, {"ml-dsa-44", "verify"}, {"ml-kem-768", "decaps"}, {"ml-kem-768", "keygen"},
		{"ecdsa-p256", "sign"}, {"ed25519", "verify"}, {"dhkem-x25519", "encaps"}, {"dhkem-p256", "decaps"}, {"ed25519", "keygen"},
	}
	prepped := map[string]bool{}
	for _, c := range cases {
		if !prepped[c.alg] {
			prep(t, dir, c.alg, 1, 8, false)
			prepped[c.alg] = true
		}
		b := plan.Block{ID: "E3." + c.alg + "." + c.op + ".r1", Exp: "E3", Alg: c.alg, Op: c.op, Rep: 1, Attempt: 1,
			Profile: plan.ProfileSmoke, Mode: "fast", Seed: 3}
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		want := proto.E3Arms(algs.MustByID(c.alg))
		if len(r.Arms) != len(want) {
			t.Fatalf("%s: arms %+v", b.Key(), r.Arms)
		}
		p50, fastest := map[string]float64{}, map[string]float64{}
		for k, a := range r.Arms {
			if a.Name != want[k] || a.Stats.N != 200 || a.Stats.P50 <= 0 {
				t.Fatalf("%s: arm %d %+v", b.Key(), k, a)
			}
			p50[a.Name] = a.Stats.P50
			fastest[a.Name] = a.Stats.Min
		}
		t.Logf("%s: median %v, fastest %v", b.Key(), p50, fastest)
		// The C call timed in C cannot be slower than the same call timed
		// around it from Go, nor than the binding. Compared on the fastest
		// sample, the cost without interference: the test build runs other
		// packages' tests in parallel on the same CPUs, and once (25 Oct) the
		// median of 200 samples moved by 13 % (l0 66.5 us, l0g 59.0 us). The
		// 10 % margin also covers the 1 us clock of a native macOS test run.
		if fastest[proto.ArmL0] > 1.10*fastest[proto.ArmL0g] || fastest[proto.ArmL0] > 1.10*fastest[proto.ArmL1] {
			t.Fatalf("%s: fastest l0 %v slower than l0g %v / l1 %v", b.Key(), fastest[proto.ArmL0], fastest[proto.ArmL0g], fastest[proto.ArmL1])
		}
	}
	b := plan.Block{ID: "E3.none.null.r1", Exp: "E3", Alg: plan.E3NullAlg, Op: plan.E3NullOp, Rep: 1, Attempt: 1,
		Profile: plan.ProfileSmoke, Mode: "fast", Seed: 3}
	r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Arms) != 2 || r.Arms[0].Name != proto.ArmCgoNull || r.Arms[0].Batch != proto.E3NullBatch || r.Arms[1].Name != proto.ArmCEmpty {
		t.Fatalf("null block arms %+v", r.Arms)
	}
	if c := r.Arms[0].Stats.P50; c <= 0 || c > 2000 {
		t.Fatalf("null cgo call %v ns", c)
	}
	if e := r.Arms[1].Stats.P50; e < 0 || e > 2000 {
		t.Fatalf("C timer floor %v ns", e)
	}
	t.Logf("null: cgo call %.1f ns, C timer floor %.1f ns", r.Arms[0].Stats.P50, r.Arms[1].Stats.P50)
}
