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

// Helpers for the add-on trial tests (E6, E7).

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
