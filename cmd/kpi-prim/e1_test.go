package main

import (
	"path/filepath"
	"strings"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

func TestPrepAndBlocks(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	cases := []struct {
		alg  string
		ops  []string
		slow map[string]bool
	}{
		{"ml-dsa-44", []string{"keygen", "sign", "verify"}, nil},
		{"falcon-512", []string{"sign", "verify"}, nil},
		{"ml-kem-768", []string{"keygen", "encaps", "decaps"}, nil},
		{"ecdsa-p256", []string{"keygen", "sign", "verify"}, nil},
		{"rsa3072-pss", []string{"keygen", "sign"}, map[string]bool{"keygen": true, "sign": true}},
		{"dhkem-x25519", []string{"keygen", "encaps", "decaps"}, nil},
	}
	for _, c := range cases {
		pr := prep(t, dir, c.alg, 1, 8, c.alg == "ml-dsa-44")
		if pr.Pool != 8 || pr.Fingerprint == "" || pr.E10 != (c.alg == "ml-dsa-44") {
			t.Fatalf("prep result %+v", pr)
		}
		for _, op := range c.ops {
			mode := "fast"
			if c.slow[op] {
				mode = "slow"
			}
			b := plan.Block{ID: "E1." + c.alg + "." + op + ".r1", Exp: "E1", Alg: c.alg, Op: op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileSmoke, Mode: mode, Seed: 7}
			raw := filepath.Join(out, b.RawPath())
			r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
			if err != nil {
				t.Fatalf("%s: %v", b.Key(), err)
			}
			a := r.Arms[0]
			if r.Key != b.Key() || len(r.Arms) != 1 || a.Name != proto.ArmReuse || a.FirstLabel != "cold" || a.FirstNS <= 0 {
				t.Fatalf("%s: result %+v", b.Key(), r)
			}
			if mode == "fast" && a.Stats.N != 200 {
				t.Fatalf("%s: %d samples, want 200", b.Key(), a.Stats.N)
			}
			if mode == "slow" && a.Stats.N < 3 {
				t.Fatalf("%s: %d samples, want at least 3", b.Key(), a.Stats.N)
			}
			if op == "keygen" && r.KeysChecked < 3 {
				t.Fatalf("%s: only %d keys fully checked", b.Key(), r.KeysChecked)
			}
			h, rows := rawRows(t, raw)
			if len(rows) != a.Stats.N+1 || r.Rows != len(rows) || !strings.HasPrefix(rows[0], "cold:0,") {
				t.Fatalf("%s: raw rows %d (first %q), result rows %d", b.Key(), len(rows), rows[0], r.Rows)
			}
			if h["key"] != b.Key() || h["prep_id"] != pr.Fingerprint || h["pool"] != "8" {
				t.Fatalf("%s: raw header %v", b.Key(), h)
			}
			if got, _ := fileSHA256(raw); got != r.RawSHA256 {
				t.Fatalf("%s: raw checksum mismatch", b.Key())
			}
		}
	}

	// Probe: its own small input set, cold call + 5 calls.
	prep(t, dir, "ml-dsa-44", proto.ProbeRep, proto.ProbeCalls+1, false)
	b := plan.Block{ID: "probe.ml-dsa-44.sign.r1", Exp: "probe", Alg: "ml-dsa-44", Op: "sign", Rep: 1, Attempt: 1,
		Profile: plan.ProfileApproved, Seed: 3}
	r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Arms[0].ProbeNS) != proto.ProbeCalls || r.Arms[0].FirstNS <= 0 {
		t.Fatalf("probe %+v", r.Arms[0])
	}
}
