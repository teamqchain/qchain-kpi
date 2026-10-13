package main

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

func prepE6(t *testing.T, dir, alg string, rep, pool int) proto.PrepResult {
	t.Helper()
	args := []string{"--alg", alg, "--rep", strconv.Itoa(rep), "--pool", strconv.Itoa(pool), "--seed", "42", "--dir", dir, "--e6"}
	out, err := capture(t, func() error { return cmdPrep(args) })
	if err != nil {
		t.Fatalf("prep %s: %v", alg, err)
	}
	var pr proto.PrepResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &pr); err != nil {
		t.Fatalf("prep output %q: %v", out, err)
	}
	return pr
}

// TestE6Blocks: sign and verify at the smallest and largest message sizes,
// the pools' messages are regenerated identically, and a prep without E6
// pools is refused.
func TestE6Blocks(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	for _, alg := range []string{"ml-dsa-44", "falcon-512", "ed25519"} {
		if pr := prepE6(t, dir, alg, 1, 8); !pr.E6 || pr.Pool != 8 {
			t.Fatalf("prep %+v", pr)
		}
		p, err := inputs.Load(proto.PrepPath(dir, alg, 1))
		if err != nil {
			t.Fatal(err)
		}
		pool, _ := p.E6Pool(1 << 20)
		m1, m2 := pool.Messages(), pool.Messages()
		if pool.P != 8 || len(m1[0]) != 1<<20 || string(m1[3]) != string(m2[3]) || string(m1[0]) == string(m1[1]) {
			t.Fatalf("%s: 1 MiB pool P=%d", alg, pool.P)
		}
		for _, c := range []struct{ op, mode string }{{proto.E6Op("sign", 64), "fast"}, {proto.E6Op("verify", 64), "fast"},
			{proto.E6Op("sign", 1<<20), "slow"}, {proto.E6Op("verify", 1<<20), "slow"}} {
			b := plan.Block{ID: "E6." + alg + "." + c.op + ".r1", Exp: "E6", Alg: alg, Op: c.op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileSmoke, Mode: c.mode, Seed: 9}
			r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
			if err != nil {
				t.Fatalf("%s: %v", b.Key(), err)
			}
			if len(r.Arms) != 1 || r.Arms[0].Name != proto.ArmReuse || r.Arms[0].FirstLabel != "cold" || r.Arms[0].Stats.N < 3 {
				t.Fatalf("%s: %+v", b.Key(), r.Arms)
			}
		}
	}
	// The 1 MiB sign takes longer than the 64-byte one (the message is hashed).
	// A prep without the E6 pools is refused.
	pr := prep(t, dir, "ml-dsa-65", 1, 8, false)
	if pr.E6 {
		t.Fatal("prep without --e6 reports E6 pools")
	}
	b := plan.Block{ID: "E6.ml-dsa-65.sign-m64.r1", Exp: "E6", Alg: "ml-dsa-65", Op: "sign-m64", Rep: 1, Attempt: 1, Profile: plan.ProfileSmoke, Mode: "fast", Seed: 9}
	if _, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())}); err == nil || !strings.Contains(err.Error(), "no E6 pool") {
		t.Fatalf("E6 without pools: %v", err)
	}
}

// TestE6Probe: the E6 probe pass runs the reuse arm (cold call + 5).
func TestE6Probe(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	prepE6(t, dir, "ml-dsa-44", proto.ProbeRep, proto.ProbeCalls+1)
	b := plan.Block{ID: "probe-E6.ml-dsa-44.verify-m102400.r1", Exp: "probe-E6", Alg: "ml-dsa-44", Op: "verify-m102400", Rep: 1, Attempt: 1, Seed: 3}
	r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
	if err != nil {
		t.Fatalf("%s: %v", b.Key(), err)
	}
	if len(r.Arms) != 1 || len(r.Arms[0].ProbeNS) != proto.ProbeCalls {
		t.Fatalf("%s: %+v", b.Key(), r.Arms)
	}
}
