package main

import (
	"os"
	"path/filepath"
	"testing"

	"qchain-kpi/internal/proto"
)

// TestCalibrateE6AddOn: the E6 cells get their own classification and batch
// sizes; nothing else changes; the estimate covers E6 from the add-on alone.
func TestCalibrateE6AddOn(t *testing.T) {
	run := "20261014-0100_mac-m1_colima-vz_b1-prod"
	reuse := []string{proto.ArmReuse}
	var results []proto.Result
	for _, size := range proto.E6Sizes {
		for _, op := range []string{"sign", "verify"} {
			ns := int64(60e3)
			if size == 1<<20 {
				ns = 2.5e6 // above 1 ms: slow
			}
			if op == "verify" && size == 64 {
				ns = 2000 // below 4,100 ns: batched
			}
			results = append(results, probeResult("probe-E6", "ml-dsa-44", proto.E6Op(op, size), map[string]int64{proto.ArmReuse: ns}, reuse))
		}
	}
	dir := probeTrialFolder(t, run, results)
	base := baseThresholds()
	cal, err := calibrate(dir, "mac-m1", base)
	if err != nil {
		t.Fatal(err)
	}
	m := cal.Thresholds
	if cal.AddOn != "E6" || m.ClassCalibratedFrom["E6"] != run || m.BatchCalibratedFrom["E6"] != run {
		t.Fatalf("markers %q %v %v", cal.AddOn, m.ClassCalibratedFrom, m.BatchCalibratedFrom)
	}
	if m.Class["ml-dsa-44/sign-m1048576"] != "slow" || m.Class["ml-dsa-44/verify-m1024"] != "fast" || m.Class["ml-dsa-44/sign"] != "fast" {
		t.Fatalf("classification %v", m.Class)
	}
	if m.Batch["ml-dsa-44/verify-m64/reuse"] != 3 || len(m.Batch) != 1 {
		t.Fatalf("batch %v", m.Batch)
	}
	if base.ClassCalibratedFrom != nil || len(base.Batch) != 0 || m.DriftFrac != base.DriftFrac {
		t.Fatal("the base thresholds were modified or not copied")
	}
	if err := writeJSON(filepath.Join(dir, fileCalib), cal); err != nil {
		t.Fatal(err)
	}
	plans := t.TempDir()
	os.WriteFile(filepath.Join(plans, "e6.json"), []byte(`{"name":"e6","kind":"main","description":"","reps":5,"items":[{"exp":"E6","algs":["ml-dsa-44"]}]}`), 0o644)
	os.WriteFile(filepath.Join(plans, "e1.json"), []byte(`{"name":"e1","kind":"main","description":"","reps":5,"items":[{"exp":"E1","algs":["ml-dsa-44"]}]}`), 0o644)
	e, err := estimate(dir, plans, "mac-m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plans) != 1 || e.Plans[0].Blocks != 50 || !(e.Plans[0].ByExpHours["E6"] > 0) || len(e.Plans[0].Missing) != 0 || !(e.Plans[0].PrepHours > 0) {
		t.Fatalf("estimate %+v", e.Plans)
	}
}

func TestE6BlockIDsCanBeExtended(t *testing.T) {
	for _, id := range []string{"E6.ml-dsa-44.sign-m1048576.r3", "E6.falcon-512.verify-m64.r1"} {
		if !blockIDPattern.MatchString(id) {
			t.Fatalf("%s not accepted", id)
		}
	}
	for _, id := range []string{"probe-E6.ml-dsa-44.sign-m64.r1", "E6.ml-dsa-44.keygen-m64.r1"} {
		if blockIDPattern.MatchString(id) {
			t.Fatalf("%s accepted", id)
		}
	}
}
