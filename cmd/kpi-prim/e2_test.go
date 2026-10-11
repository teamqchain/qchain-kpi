package main

import (
	"path/filepath"
	"strings"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// TestE2AndE2cBlocks runs every operation of a spread of algorithms (lattice,
// Falcon, hash-based, HQC, every classical one) through E2 and E2c blocks
// with smoke budgets, E2c with batches > 1 so the init/clean state handling
// is exercised, and through the probe-E2 pass.
func TestE2AndE2cBlocks(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	slow := map[string]bool{"rsa3072-pss/keygen": true, "rsa3072-pss/sign": true, "slh-dsa-sha2-128f/sign": true}
	for _, id := range []string{"ml-dsa-44", "falcon-512", "slh-dsa-sha2-128f", "ml-kem-768", "hqc-1",
		"ecdsa-p256", "ed25519", "rsa3072-pss", "dhkem-p256", "dhkem-x25519"} {
		a := algs.MustByID(id)
		pool := 8
		prep(t, dir, id, 1, pool, false)
		prep(t, dir, id, proto.ProbeRep, proto.ProbeCalls+1, false)
		for _, op := range a.Ops() {
			mode := "fast"
			if slow[id+"/"+op] {
				mode = "slow"
			}
			// E2: reuse, reinit, prod.
			b := plan.Block{ID: "E2." + id + "." + op + ".r1", Exp: "E2", Alg: id, Op: op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileSmoke, Mode: mode, Seed: 11}
			r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
			if err != nil {
				t.Fatalf("%s: %v", b.Key(), err)
			}
			want := proto.E2Arms(a, op)
			if len(r.Arms) != len(want) {
				t.Fatalf("%s: arms %d, want %v", b.Key(), len(r.Arms), want)
			}
			total := 0
			for k, arm := range r.Arms {
				if arm.Name != want[k] || arm.FirstLabel != "first" || arm.FirstNS <= 0 {
					t.Fatalf("%s: arm %d %+v", b.Key(), k, arm)
				}
				if mode == "fast" && arm.Stats.N != 200 {
					t.Fatalf("%s/%s: %d samples", b.Key(), arm.Name, arm.Stats.N)
				}
				total += arm.Stats.N + 1
			}
			if op == "keygen" && r.KeysChecked < 3*len(want) {
				t.Fatalf("%s: only %d keys fully checked", b.Key(), r.KeysChecked)
			}
			_, rows := rawRows(t, filepath.Join(out, b.RawPath()))
			if len(rows) != total {
				t.Fatalf("%s: raw rows %d, want %d", b.Key(), len(rows), total)
			}

			// E2c: every component, batched 3 calls per sample.
			batch := map[string]int{}
			for _, n := range proto.E2cArms(a, op) {
				batch[n] = 3
			}
			c := plan.Block{ID: "E2c." + id + "." + op + ".r1", Exp: "E2c", Alg: id, Op: op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileSmoke, Mode: "fast", Seed: 13}
			r, err = block(t, proto.Spec{Block: c, PrepDir: dir, RawFile: filepath.Join(out, c.RawPath()), Batch: batch})
			if err != nil {
				t.Fatalf("%s: %v", c.Key(), err)
			}
			wantc := proto.E2cArms(a, op)
			if len(r.Arms) != len(wantc) {
				t.Fatalf("%s: arms %d, want %v", c.Key(), len(r.Arms), wantc)
			}
			for k, arm := range r.Arms {
				if arm.Name != wantc[k] || arm.Batch != 3 || arm.Stats.N != 200 || !(arm.Stats.P50 > 0) {
					t.Fatalf("%s: arm %d %+v", c.Key(), k, arm)
				}
			}
			h, _ := rawRows(t, filepath.Join(out, c.RawPath()))
			if !strings.Contains(h["batch"], ":3") {
				t.Fatalf("%s: raw header batch %q", c.Key(), h["batch"])
			}

			// probe-E2: every arm, first call + ProbeCalls calls.
			pb := plan.Block{ID: "probe-E2." + id + "." + op + ".r1", Exp: "probe-E2", Alg: id, Op: op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileApproved, Seed: 17}
			r, err = block(t, proto.Spec{Block: pb, PrepDir: dir, RawFile: filepath.Join(out, pb.RawPath())})
			if err != nil {
				t.Fatalf("%s: %v", pb.Key(), err)
			}
			wantp := proto.ArmsOf("probe-E2", a, op)
			if len(r.Arms) != len(wantp) {
				t.Fatalf("%s: arms %d, want %d", pb.Key(), len(r.Arms), len(wantp))
			}
			for k, arm := range r.Arms {
				if arm.Name != wantp[k] || len(arm.ProbeNS) != proto.ProbeCalls || arm.FirstNS <= 0 {
					t.Fatalf("%s: arm %d %+v", pb.Key(), k, arm)
				}
			}
		}
	}
}
