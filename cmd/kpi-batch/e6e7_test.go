package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// probeTrialFolder writes a synthetic add-on trial holding the given probe
// results (timer resolution 41 ns: batching below 4,100 ns).
func probeTrialFolder(t *testing.T, runID string, results []proto.Result) string {
	t.Helper()
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	st := &State{Schema: 1, RunID: runID, Machine: "mac-m1", PlanKind: plan.KindTrial,
		Seed: 1, Sessions: []Session{{N: 1, StartNS: 0}}, CellReps: map[string]int{}, VMCPUs: 4, HostCPUs: 8, HostPeriodNS: 5e9,
		Class: map[string]string{}, Requeued: map[string]bool{}, Diagnosed: map[string]bool{}, Preps: map[string]PrepRec{}}
	must(st.save(dir))
	must(os.MkdirAll(filepath.Join(dir, dirEnv), 0o755))
	must(os.WriteFile(filepath.Join(dir, dirEnv, "e0-s1.json"), []byte(`{"clock":{"effective_resolution_ns":41}}`), 0o644))
	hdr := []string{"utc_ns", "event", "key", "detail"}
	var t0 int64 = 10e9
	for _, r := range results {
		r.Key, r.StartNS, r.EndNS, r.WallNS = r.Block.Key(), t0, t0+1e9, 1e9
		t0 += 2e9
		must(appendJSONL(filepath.Join(dir, fileBlocksJL), r))
		must(appendCSV(filepath.Join(dir, fileJournal), hdr, []string{strconv.FormatInt(r.StartNS, 10), "block_start", r.Key, ""}))
		must(appendCSV(filepath.Join(dir, fileJournal), hdr, []string{strconv.FormatInt(r.EndNS+1e8, 10), "block_done", r.Key, ""}))
	}
	c := canaryRec{Seq: 1, Reason: "start", Ratio: 1}
	c.StartNS, c.EndNS, c.MedianRate = 1e9, 2e9, 2.5e6
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

func probeResult(exp, alg, op string, arms map[string]int64, order []string) proto.Result {
	b := plan.Block{ID: exp + "." + alg + "." + op + ".r1", Exp: exp, Alg: alg, Op: op, Rep: 1, Attempt: 1}
	var as []proto.ArmSummary
	for _, n := range order {
		m := arms[n]
		as = append(as, proto.ArmSummary{Name: n, ProbeNS: []int64{m, m, m + 10, m - 10, m}})
	}
	return proto.Result{Block: b, Arms: as}
}

func baseThresholds() *quality.Machine {
	m := quality.Defaults()
	m.CalibratedFrom = "20261010-1819_mac-m1_colima-vz_b1-prod"
	m.DriftFrac, m.CanaryFrac = 0.0416, 0.03
	m.Class["ml-dsa-44/verify"], m.Class["ecdsa-p256/verify"], m.Class["ml-dsa-44/sign"] = "fast", "fast", "fast"
	m.Batch = map[string]int{}
	return m
}

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

func TestE6E7BlockIDsCanBeExtended(t *testing.T) {
	for _, id := range []string{"E6.ml-dsa-44.sign-m1048576.r3", "E6.falcon-512.verify-m64.r1", "E7.ml-kem-768.decaps.r2", "E7.ecdsa-p256.verify.r5"} {
		if !blockIDPattern.MatchString(id) {
			t.Fatalf("%s not accepted", id)
		}
	}
	for _, id := range []string{"probe-E6.ml-dsa-44.sign-m64.r1", "E6.ml-dsa-44.keygen-m64.r1", "probe-E7.ml-kem-768.decaps.r1"} {
		if blockIDPattern.MatchString(id) {
			t.Fatalf("%s accepted", id)
		}
	}
}
