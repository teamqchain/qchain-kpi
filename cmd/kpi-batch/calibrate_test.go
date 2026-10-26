package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// trialFolder writes a small synthetic trial run: a probe pass for ML-DSA-44,
// 5 E1 repetitions of its sign operation, 5 E10 repetitions, 5 canaries,
// host samples covering everything, E0 and the journal.
func trialFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	st := &State{Schema: 1, RunID: "20261010-1900_mac-m1_colima-vz_b1-prod", Machine: "mac-m1", PlanKind: plan.KindTrial,
		Seed: 1, Sessions: []Session{{N: 1, StartNS: 0}}, CellReps: map[string]int{"E1.ml-dsa-44.sign": 5, "E10.ml-dsa-44.sign": 5},
		VMCPUs: 4, HostCPUs: 8, HostPeriodNS: 5e9, Class: map[string]string{}, Requeued: map[string]bool{},
		Diagnosed: map[string]bool{}, Preps: map[string]PrepRec{}}
	must(st.save(dir))
	must(writeJSON(filepath.Join(dir, fileManifest), Manifest{Thresholds: ManifestTH{Values: quality.Defaults()}}))
	must(os.MkdirAll(filepath.Join(dir, dirEnv), 0o755))
	must(os.WriteFile(filepath.Join(dir, dirEnv, "e0-s1.json"), []byte(`{"clock":{"effective_resolution_ns":41}}`), 0o644))

	var t0 int64 = 10e9
	add := func(r proto.Result) {
		r.StartNS, r.EndNS, r.WallNS = t0, t0+2e9, 2e9
		t0 += 3e9
		r.Key = r.Block.Key()
		must(appendJSONL(filepath.Join(dir, fileBlocksJL), r))
		must(appendCSV(filepath.Join(dir, fileJournal), []string{"utc_ns", "event", "key", "detail"},
			[]string{strconv.FormatInt(r.StartNS, 10), "block_start", r.Key, ""}))
		must(appendCSV(filepath.Join(dir, fileJournal), []string{"utc_ns", "event", "key", "detail"},
			[]string{strconv.FormatInt(r.EndNS+2e8, 10), "block_done", r.Key, ""}))
	}
	for _, op := range []string{"keygen", "sign", "verify"} {
		med := map[string]int64{"keygen": 35000, "sign": 70000, "verify": 29000}[op]
		add(proto.Result{Block: plan.Block{ID: "probe.ml-dsa-44." + op + ".r1", Exp: "probe", Alg: "ml-dsa-44", Op: op, Rep: 1, Attempt: 1},
			Arms: []proto.ArmSummary{{Name: proto.ArmReuse, ProbeNS: []int64{med, med + 100, med - 100, med, med + 50}}}})
	}
	for rep := 1; rep <= 5; rep++ {
		p50 := 70000 * (1 + 0.004*float64(rep-3))
		add(proto.Result{Block: plan.Block{ID: "E1.ml-dsa-44.sign.r" + strconv.Itoa(rep), Exp: "E1", Alg: "ml-dsa-44", Op: "sign",
			Rep: rep, Attempt: 1, Mode: "fast"}, Arms: []proto.ArmSummary{{Name: proto.ArmReuse, Stats: stats.Summary{N: 10000, P50: p50, BootSE50: 50}}}})
		add(proto.Result{Block: plan.Block{ID: "E10.ml-dsa-44.sign.r" + strconv.Itoa(rep), Exp: "E10", Alg: "ml-dsa-44", Op: "sign",
			Rep: rep, Attempt: 1, Mode: "fast"}, Arms: []proto.ArmSummary{
			{Name: proto.ArmIssuer, Stats: stats.Summary{N: 10000, P50: 70000 + float64(rep), BootSE50: 50}},
			{Name: proto.ArmHolder, Stats: stats.Summary{N: 10000, P50: 70010, BootSE50: 50}},
			{Name: proto.ArmIssuerAgain, Stats: stats.Summary{N: 10000, P50: 70000 - float64(rep), BootSE50: 50}}}})
	}
	for i := 0; i < 5; i++ {
		c := canaryRec{Seq: i + 1, Reason: "periodic", Ratio: 1 + 0.002*float64(i%2)}
		c.StartNS, c.EndNS = int64(i)*12e9, int64(i)*12e9+1e9
		c.MedianRate = 2.5e6 * c.Ratio
		must(appendJSONL(filepath.Join(dir, fileCanaryJL), c))
	}
	c := canaryRec{Seq: 6, Reason: "end", Ratio: 1}
	c.StartNS, c.EndNS, c.MedianRate = t0+1e9, t0+2e9, 2.5e6
	must(appendJSONL(filepath.Join(dir, fileCanaryJL), c))
	must(os.MkdirAll(filepath.Join(dir, dirMonitor), 0o755))
	rows := "utc_ns,other_pct,throttle,on_ac,alert,detail\n"
	for ts := int64(5e9); ts <= t0+10e9; ts += 5e9 {
		rows += strconv.FormatInt(ts, 10) + ",1.0,0,1,0,x\n"
	}
	must(os.WriteFile(filepath.Join(dir, dirMonitor, "host-s1-000001.csv"), []byte(rows), 0o644))
	must(os.WriteFile(filepath.Join(dir, dirMonitor, "host-s1-000001.ok"), nil, 0o644))
	return dir
}

// TestCalibrateFixedCanaryReference: with --canary-reference median the
// calibration stores the median canary of the trial as the machine's fixed
// reference; by default (the Mac) it stores none.
func TestCalibrateFixedCanaryReference(t *testing.T) {
	dir := trialFolder(t)
	first, err := calibrate(dir, "mac-m1", quality.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if first.Thresholds.CanaryRef != 0 || first.Thresholds.Inputs["canary_ref_rate"] != 0 {
		t.Fatalf("the default keeps the first-canary rule: %+v", first.Thresholds)
	}
	med, err := calibrateWith(dir, "mac-m1", quality.Defaults(), canaryRefMedian)
	if err != nil {
		t.Fatal(err)
	}
	m := med.Thresholds
	if math.Abs(m.CanaryRef-2.5e6) > 1e4 || m.Inputs["canary_ref_rate"] != m.CanaryRef || m.CanaryFrac < quality.DefaultCanaryFrac {
		t.Fatalf("fixed reference: %+v", m)
	}
	if m.DriftFrac != first.Thresholds.DriftFrac || len(m.Class) != len(first.Thresholds.Class) {
		t.Fatal("only the canary part may differ")
	}
	if _, err := json.Marshal(med); err != nil {
		t.Fatal(err)
	}
}

func TestCalibrateAndEstimate(t *testing.T) {
	dir := trialFolder(t)
	cal, err := calibrate(dir, "mac-m1", quality.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	m := cal.Thresholds
	if m.CalibratedFrom == "" || m.Class["ml-dsa-44/sign"] != "fast" || len(m.Class) != 3 {
		t.Fatalf("classification %v", m.Class)
	}
	if m.DriftFrac != quality.DefaultDriftFrac || m.CanaryFrac != quality.DefaultCanaryFrac {
		t.Fatalf("quiet trial should keep the floors: drift %v canary %v", m.DriftFrac, m.CanaryFrac)
	}
	if v := m.E10Margin["ml-dsa-44"]; !(v >= quality.DefaultE10Margin) || math.IsNaN(v) {
		t.Fatalf("E10 margin %v", v)
	}
	if len(m.Batch) != 0 {
		t.Fatalf("no ML-DSA operation is below 100x the timer resolution: %v", m.Batch)
	}
	if len(cal.Notes) == 0 {
		t.Fatal("missing probes and E10 sets must be noted")
	}
	// The calibration must serialise (no NaN) and pass the committed-file rules
	// apart from the 90-cell classification of a full trial.
	if _, err := json.Marshal(cal); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, fileCalib), cal); err != nil {
		t.Fatal(err)
	}
	plans := t.TempDir()
	os.WriteFile(filepath.Join(plans, "e1.json"), []byte(`{"name":"e1","kind":"main","description":"","reps":5,"items":[{"exp":"E1","algs":["ml-dsa-44"],"ops":["sign"]}]}`), 0o644)
	os.WriteFile(filepath.Join(plans, "trial.json"), []byte(`{"name":"trial","kind":"trial","description":"","reps":1,"items":[]}`), 0o644)
	e, err := estimate(dir, plans, "mac-m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plans) != 1 || e.Plans[0].Blocks != 5 || !(e.TotalHours > 0) || !(e.TotalRawMB > 0) || len(e.Plans[0].Missing) != 0 {
		t.Fatalf("estimate %+v", e)
	}
	// 5 blocks of 11,001 signs at 70 us plus 11,001 verifies at 29 us is about
	// 5.5 s of operations; with the per-block overhead it is well under an hour.
	if e.TotalHours > 0.1 {
		t.Fatalf("estimate too large: %v h", e.TotalHours)
	}
	if _, err := json.Marshal(e); err != nil {
		t.Fatal(err)
	}

	// A first trial cannot estimate the experiments with their own add-on
	// trial: they are listed, not given a meaningless figure or NaN (an E6
	// plan gave NaN, which cannot be written to estimate.json).
	os.WriteFile(filepath.Join(plans, "e6.json"), []byte(`{"name":"e6","kind":"main","description":"","reps":5,"items":[{"exp":"E6","algs":["ml-dsa-44"]}]}`), 0o644)
	os.WriteFile(filepath.Join(plans, "e2.json"), []byte(`{"name":"e2","kind":"main","description":"","reps":5,"items":[{"exp":"E2","algs":["ml-dsa-44"],"ops":["sign"]}]}`), 0o644)
	e, err = estimate(dir, plans, "mac-m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plans) != 1 || e.NotEstimated["e6"] != "needs its add-on trial (trial-e6)" || e.NotEstimated["e2"] != "needs its add-on trial (trial-e2)" {
		t.Fatalf("estimate %+v", e)
	}
	if err := writeJSON(filepath.Join(dir, fileEstimate), e); err != nil {
		t.Fatalf("estimate.json: %v", err)
	}
}
