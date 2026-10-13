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

// TestE7Blocks: valid input is accepted and every invalid input rejected, in
// the way each family rejects (false, an error, or a different secret).
func TestE7Blocks(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	cases := []struct {
		alg string
		// allowed outcomes of the invalid arms
		bad map[int]bool
	}{
		{"ml-dsa-44", map[int]bool{proto.E7Rejected: true}},
		{"slh-dsa-sha2-128f", map[int]bool{proto.E7Rejected: true}},
		{"falcon-512", map[int]bool{proto.E7Rejected: true}},
		{"ecdsa-p256", map[int]bool{proto.E7Rejected: true, proto.E7RejectedError: true}}, // a flip can break the DER
		{"ed25519", map[int]bool{proto.E7Rejected: true}},
		{"ml-kem-768", map[int]bool{proto.E7Rejected: true}},      // implicit rejection
		{"hqc-1", map[int]bool{proto.E7Rejected: true}},           // implicit rejection
		{"dhkem-p256", map[int]bool{proto.E7RejectedError: true}}, // off-curve point
		{"dhkem-x25519", map[int]bool{proto.E7Rejected: true, proto.E7RejectedError: true}},
	}
	for _, c := range cases {
		prep(t, dir, c.alg, 1, 8, false)
		op := "verify"
		if strings.HasPrefix(c.alg, "ml-kem") || strings.HasPrefix(c.alg, "hqc") || strings.HasPrefix(c.alg, "dhkem") {
			op = "decaps"
		}
		b := plan.Block{ID: "E7." + c.alg + "." + op + ".r1", Exp: "E7", Alg: c.alg, Op: op, Rep: 1, Attempt: 1,
			Profile: plan.ProfileSmoke, Mode: "fast", Seed: 11}
		raw := filepath.Join(out, b.RawPath())
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		want := []string{proto.ArmValid, proto.ArmBadSig, proto.ArmWrongMsg}
		if op == "decaps" {
			want = []string{proto.ArmValid, proto.ArmTampered}
		}
		if len(r.Arms) != len(want) {
			t.Fatalf("%s: arms %+v", b.Key(), r.Arms)
		}
		for k, a := range r.Arms {
			if a.Name != want[k] || a.FirstLabel != "first" || a.Stats.N != 200 {
				t.Fatalf("%s: arm %+v", b.Key(), a)
			}
		}
		_, rows := rawRows(t, raw)
		for _, row := range rows {
			f := strings.Split(row, ",")
			arm, _ := strconv.Atoi(strings.TrimPrefix(f[0], "first:"))
			got, _ := strconv.Atoi(f[len(f)-1])
			if arm == 0 && got != proto.E7Accepted || arm > 0 && !c.bad[got] {
				t.Fatalf("%s: arm %s outcome %d (row %q)", b.Key(), want[arm], got, row)
			}
		}
	}
	// A KEM block named verify is refused.
	b := plan.Block{ID: "E7.ml-kem-768.verify.r1", Exp: "E7", Alg: "ml-kem-768", Op: "verify", Rep: 1, Attempt: 1, Profile: plan.ProfileSmoke, Mode: "fast", Seed: 11}
	if _, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())}); err == nil {
		t.Fatal("E7 verify for a KEM must be refused")
	}
}

// TestE6E7Probes: the probe passes run every arm (cold or first call + 5).
func TestE6E7Probes(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	prepE6(t, dir, "ml-dsa-44", proto.ProbeRep, proto.ProbeCalls+1)
	prep(t, dir, "ml-kem-512", proto.ProbeRep, proto.ProbeCalls+1, false)
	for _, b := range []plan.Block{
		{ID: "probe-E6.ml-dsa-44.verify-m102400.r1", Exp: "probe-E6", Alg: "ml-dsa-44", Op: "verify-m102400", Rep: 1, Attempt: 1, Seed: 3},
		{ID: "probe-E7.ml-dsa-44.verify.r1", Exp: "probe-E7", Alg: "ml-dsa-44", Op: "verify", Rep: 1, Attempt: 1, Seed: 3},
		{ID: "probe-E7.ml-kem-512.decaps.r1", Exp: "probe-E7", Alg: "ml-kem-512", Op: "decaps", Rep: 1, Attempt: 1, Seed: 3},
	} {
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		for _, a := range r.Arms {
			if len(a.ProbeNS) != proto.ProbeCalls {
				t.Fatalf("%s/%s: %d probe calls", b.Key(), a.Name, len(a.ProbeNS))
			}
		}
	}
}
