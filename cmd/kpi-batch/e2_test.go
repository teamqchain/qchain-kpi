package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// e2TrialFolder writes a synthetic add-on trial: one probe-E2 block for
// ML-KEM-768 encaps, whose components are below 100x the timer resolution.
func e2TrialFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	st := &State{Schema: 1, RunID: "20261012-0100_mac-m1_colima-vz_b1-prod", Machine: "mac-m1", PlanKind: plan.KindTrial,
		Seed: 1, Sessions: []Session{{N: 1, StartNS: 0}}, CellReps: map[string]int{}, VMCPUs: 4, HostCPUs: 8, HostPeriodNS: 5e9,
		Class: map[string]string{}, Requeued: map[string]bool{}, Diagnosed: map[string]bool{}, Preps: map[string]PrepRec{}}
	must(st.save(dir))
	must(os.MkdirAll(filepath.Join(dir, dirEnv), 0o755))
	must(os.WriteFile(filepath.Join(dir, dirEnv, "e0-s1.json"), []byte(`{"clock":{"effective_resolution_ns":41}}`), 0o644))
	a := algs.MustByID("ml-kem-768")
	med := map[string]int64{proto.ArmReuse: 14000, proto.ArmReinit: 14500, proto.ArmProd: 16000,
		proto.ArmHexDecodePK: 2000, proto.ArmInit: 400, proto.ArmClean: 100, proto.ArmHexEncodeCT: 1500, proto.ArmHexEncodeSS: 41}
	var arms []proto.ArmSummary
	for _, n := range proto.ArmsOf("probe-E2", a, "encaps") {
		m := med[n]
		arms = append(arms, proto.ArmSummary{Name: n, ProbeNS: []int64{m, m, m + 10, m - 10, m}})
	}
	r := proto.Result{Block: plan.Block{ID: "probe-E2.ml-kem-768.encaps.r1", Exp: "probe-E2", Alg: a.ID, Op: "encaps", Rep: 1, Attempt: 1},
		Arms: arms, StartNS: 10e9, EndNS: 12e9, WallNS: 2e9}
	r.Key = r.Block.Key()
	must(appendJSONL(filepath.Join(dir, fileBlocksJL), r))
	hdr := []string{"utc_ns", "event", "key", "detail"}
	must(appendCSV(filepath.Join(dir, fileJournal), hdr, []string{strconv.FormatInt(r.StartNS, 10), "block_start", r.Key, ""}))
	must(appendCSV(filepath.Join(dir, fileJournal), hdr, []string{strconv.FormatInt(r.EndNS+2e8, 10), "block_done", r.Key, ""}))
	c := canaryRec{Seq: 1, Reason: "start", Ratio: 1}
	c.StartNS, c.EndNS, c.MedianRate = 1e9, 2e9, 2.5e6
	must(appendJSONL(filepath.Join(dir, fileCanaryJL), c))
	must(os.MkdirAll(filepath.Join(dir, dirMonitor), 0o755))
	rows := "utc_ns,other_pct,throttle,on_ac,alert,detail\n"
	for ts := int64(5e9); ts <= 30e9; ts += 5e9 {
		rows += strconv.FormatInt(ts, 10) + ",1.0,0,1,0,x\n"
	}
	must(os.WriteFile(filepath.Join(dir, dirMonitor, "host-s1-000001.csv"), []byte(rows), 0o644))
	must(os.WriteFile(filepath.Join(dir, dirMonitor, "host-s1-000001.ok"), nil, 0o644))
	return dir
}

func TestCalibrateE2AddOnAddsOnlyBatchSizes(t *testing.T) {
	dir := e2TrialFolder(t)
	base := quality.Defaults()
	base.CalibratedFrom = "20261010-1819_mac-m1_colima-vz_b1-prod"
	base.DriftFrac, base.CanaryFrac = 0.0416, 0.03
	base.Class["ml-kem-768/encaps"] = "fast"
	base.Batch = map[string]int{}
	if _, err := calibrate(dir, "mac-m1", quality.Defaults()); err == nil {
		t.Fatal("an add-on trial must refuse uncalibrated base thresholds")
	}
	cal, err := calibrate(dir, "mac-m1", base)
	if err != nil {
		t.Fatal(err)
	}
	m := cal.Thresholds
	if cal.AddOn != "E2" || m.BatchCalibratedFrom["E2"] != "20261012-0100_mac-m1_colima-vz_b1-prod" {
		t.Fatalf("add-on markers %q %v", cal.AddOn, m.BatchCalibratedFrom)
	}
	if m.CalibratedFrom != base.CalibratedFrom || m.DriftFrac != 0.0416 || m.Class["ml-kem-768/encaps"] != "fast" {
		t.Fatalf("base thresholds were not copied unchanged: %+v", m)
	}
	// 100 x 41 ns = 4100 ns: hex-decode-pk (2000) -> 3, init (400) -> 11,
	// clean (100) -> 41, hex-encode-ct (1500) -> 3, hex-encode-ss (41) -> 100.
	want := map[string]int{"hex-decode-pk": 3, "init": 11, "clean": 41, "hex-encode-ct": 3, "hex-encode-ss": 100}
	for arm, n := range want {
		if got := m.Batch["ml-kem-768/encaps/"+arm]; got != n {
			t.Fatalf("batch %s = %d, want %d (%v)", arm, got, n, m.Batch)
		}
	}
	if len(m.Batch) != len(want) {
		t.Fatalf("unexpected batch entries (reuse/reinit/prod must not be batched here): %v", m.Batch)
	}
	if base.BatchCalibratedFrom != nil || len(base.Batch) != 0 {
		t.Fatal("the base thresholds were modified")
	}

	// The estimate covers E2 with the add-on's per-arm probe medians.
	if err := writeJSON(filepath.Join(dir, fileCalib), cal); err != nil {
		t.Fatal(err)
	}
	plans := t.TempDir()
	os.WriteFile(filepath.Join(plans, "e2.json"), []byte(`{"name":"e2","kind":"main","description":"","reps":5,"items":[{"exp":"E2","algs":["ml-kem-768"],"ops":["encaps"]},{"exp":"E2c","algs":["ml-kem-768"],"ops":["encaps"]}]}`), 0o644)
	e, err := estimate(dir, plans, "mac-m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plans) != 1 || e.Plans[0].Blocks != 10 || !(e.Plans[0].ByExpHours["E2"] > 0) || !(e.Plans[0].ByExpHours["E2c"] > 0) {
		t.Fatalf("estimate %+v", e.Plans)
	}
}

func TestE2BlockIDsCanBeExtended(t *testing.T) {
	for _, id := range []string{"E2.ml-dsa-44.sign.r3", "E2c.dhkem-x25519.decaps.r1", "E1.hqc-3.encaps.r3", "E10.ml-dsa-87.sign.r2"} {
		if !blockIDPattern.MatchString(id) {
			t.Fatalf("%s not accepted", id)
		}
	}
	for _, id := range []string{"probe-E2.ml-dsa-44.sign.r1", "probe.ml-dsa-44.sign.r1", "E5.ml-dsa-44.sign.r1"} {
		if blockIDPattern.MatchString(id) {
			t.Fatalf("%s accepted", id)
		}
	}
}

func TestE2PlansExpand(t *testing.T) {
	root := filepath.Join("..", "..", "plans")
	p, err := plan.Load(filepath.Join(root, "e2.json"))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := plan.Expand(p, 7)
	if err != nil {
		t.Fatal(err)
	}
	ops := 0
	for _, a := range algs.All() {
		ops += len(a.Ops())
	}
	if len(blocks) != 2*ops*5 {
		t.Fatalf("e2: %d blocks, want %d", len(blocks), 2*ops*5)
	}
	tr, err := plan.Load(filepath.Join(root, "trial-e2.json"))
	if err != nil {
		t.Fatal(err)
	}
	tb, _ := plan.Expand(tr, 7)
	if len(tb) != ops || tb[0].Exp != "probe-E2" {
		t.Fatalf("trial-e2: %d blocks", len(tb))
	}
	sm, err := plan.Load(filepath.Join(root, "smoke-e2.json"))
	if err != nil {
		t.Fatal(err)
	}
	sb, _ := plan.Expand(sm, 7)
	if len(sb) != 4*ops || !proto.IsProbe(sb[0].Exp) || !proto.IsProbe(sb[2*ops-1].Exp) || proto.IsProbe(sb[2*ops].Exp) {
		t.Fatalf("smoke-e2: %d blocks, probes must come first", len(sb))
	}
}
