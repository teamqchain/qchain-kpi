package main

import (
	"path/filepath"
	"testing"

	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// TestE11Blocks runs E11 and probe-E11 blocks end to end through the block
// runner (smoke budgets), including verification's second arm.
func TestE11Blocks(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	for _, c := range []cells.Cell{
		{Stage: cells.Activation, Sig: "ml-dsa-44", KEM: "ml-kem-768"},
		{Stage: cells.Issuance, Sig: "ed25519", KEM: "dhkem-x25519", N: 20, V: 16},
		{Stage: cells.Retrieval, KEM: "ml-kem-512", N: 5, V: 256},
		{Stage: cells.Presentation, Sig: "falcon-512", K: 3, V: 16},
		{Stage: cells.Verification, Sig: "ml-dsa-44", KEM: "hqc-1", N: 10, K: 5, V: 16},
		{Stage: cells.Chained, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 1, K: 1, V: 16},
	} {
		b := plan.Block{ID: "E11." + c.Alg() + "." + c.Op() + ".r1", Exp: "E11", Alg: c.Alg(), Op: c.Op(), Rep: 1, Attempt: 1,
			Profile: plan.ProfileSmoke, Mode: "fast", Seed: 3}
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		want := proto.E11Arms(b.Op)
		if len(r.Arms) != len(want) || r.PrepNS <= 0 {
			t.Fatalf("%s: arms %d (want %v), prep %d", b.Key(), len(r.Arms), want, r.PrepNS)
		}
		for k, a := range r.Arms {
			if a.Name != want[k] || a.Stats.N != 50 || a.FirstLabel != "first" {
				t.Fatalf("%s: arm %+v", b.Key(), a)
			}
		}
		h, rows := rawRows(t, filepath.Join(out, b.RawPath()))
		if len(rows) != len(want)*51 || h["gc"] == "" || h["prep_ns"] == "" {
			t.Fatalf("%s: %d raw rows, header %v", b.Key(), len(rows), h)
		}
		// The same cell as a probe.
		pb := b
		pb.ID, pb.Exp, pb.Mode, pb.Profile = "probe-E11."+c.Alg()+"."+c.Op()+".r1", "probe-E11", "", plan.ProfileApproved
		pr, err := block(t, proto.Spec{Block: pb, PrepDir: dir, RawFile: filepath.Join(out, pb.RawPath())})
		if err != nil {
			t.Fatalf("%s: %v", pb.Key(), err)
		}
		if len(pr.Arms) != len(want) || len(pr.Arms[0].ProbeNS) != proto.ProbeCalls {
			t.Fatalf("%s: probe %+v", pb.Key(), pr.Arms)
		}
	}
}
