package main

import (
	"os"
	"path/filepath"
	"testing"

	"qchain-kpi/internal/proto"
)

// TestCalibrateE7AddOn: only the E7 arms' batch sizes are new (the
// classification is the operation's, from the first trial).
func TestCalibrateE7AddOn(t *testing.T) {
	run := "20261014-0200_mac-m1_colima-vz_b1-prod"
	order := []string{proto.ArmValid, proto.ArmBadSig, proto.ArmWrongMsg}
	r := probeResult("probe-E7", "ecdsa-p256", "verify", map[string]int64{proto.ArmValid: 40000, proto.ArmBadSig: 300, proto.ArmWrongMsg: 40000}, order)
	dir := probeTrialFolder(t, run, []proto.Result{r})
	base := baseThresholds()
	cal, err := calibrate(dir, "mac-m1", base)
	if err != nil {
		t.Fatal(err)
	}
	m := cal.Thresholds
	if cal.AddOn != "E7" || m.BatchCalibratedFrom["E7"] != run || m.ClassCalibratedFrom["E7"] != "" {
		t.Fatalf("markers %q %v %v", cal.AddOn, m.BatchCalibratedFrom, m.ClassCalibratedFrom)
	}
	if m.Batch["ecdsa-p256/verify/badsig"] != 14 || len(m.Batch) != 1 || len(m.Class) != len(base.Class) {
		t.Fatalf("batch %v class %d", m.Batch, len(m.Class))
	}
	if err := writeJSON(filepath.Join(dir, fileCalib), cal); err != nil {
		t.Fatal(err)
	}
	plans := t.TempDir()
	os.WriteFile(filepath.Join(plans, "e7.json"), []byte(`{"name":"e7","kind":"main","description":"","reps":5,"items":[{"exp":"E7","algs":["ecdsa-p256"]}]}`), 0o644)
	e, err := estimate(dir, plans, "mac-m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plans) != 1 || e.Plans[0].Blocks != 5 || !(e.Plans[0].ByExpHours["E7"] > 0) {
		t.Fatalf("estimate %+v", e.Plans)
	}
}

func TestE7BlockIDsCanBeExtended(t *testing.T) {
	for _, id := range []string{"E7.ml-kem-768.decaps.r2", "E7.ecdsa-p256.verify.r5"} {
		if !blockIDPattern.MatchString(id) {
			t.Fatalf("%s not accepted", id)
		}
	}
	if blockIDPattern.MatchString("probe-E7.ml-kem-768.decaps.r1") {
		t.Fatal("a probe block id was accepted")
	}
}
