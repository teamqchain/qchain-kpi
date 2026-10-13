package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

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

// TestE7Probes: the E7 probe pass runs every arm (first call + 5).
func TestE7Probes(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	prep(t, dir, "ml-dsa-44", proto.ProbeRep, proto.ProbeCalls+1, false)
	prep(t, dir, "ml-kem-512", proto.ProbeRep, proto.ProbeCalls+1, false)
	for _, b := range []plan.Block{
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
