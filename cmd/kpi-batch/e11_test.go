package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

func TestE11PlansExpand(t *testing.T) {
	root := filepath.Join("..", "..", "plans")
	p, err := plan.Load(filepath.Join(root, "e11.json"))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := plan.Expand(p, 9)
	if err != nil {
		t.Fatal(err)
	}
	grid := len(cells.Grid())
	if len(blocks) != 5*grid {
		t.Fatalf("e11: %d blocks, want %d", len(blocks), 5*grid)
	}
	for _, b := range blocks[:grid] {
		if !blockIDPattern.MatchString(b.ID) {
			t.Fatalf("E11 block id %q cannot be extended", b.ID)
		}
		if _, err := cells.ParseCell(b.Alg, b.Op); err != nil {
			t.Fatalf("%s: %v", b.ID, err)
		}
	}
	tr, err := plan.Load(filepath.Join(root, "trial-e11.json"))
	if err != nil {
		t.Fatal(err)
	}
	tb, _ := plan.Expand(tr, 9)
	if len(tb) != grid || tb[0].Exp != "probe-E11" {
		t.Fatalf("trial-e11: %d blocks", len(tb))
	}
	sm, err := plan.Load(filepath.Join(root, "smoke-e11.json"))
	if err != nil {
		t.Fatal(err)
	}
	sb, _ := plan.Expand(sm, 9)
	half := len(sb) / 2
	if len(sb)%2 != 0 || half < 50 || !proto.IsProbe(sb[0].Exp) || !proto.IsProbe(sb[half-1].Exp) || proto.IsProbe(sb[half].Exp) {
		t.Fatalf("smoke-e11: %d blocks; probes must come first", len(sb))
	}
	stages := map[string]bool{}
	for _, b := range sb[half:] {
		c, _ := cells.ParseCell(b.Alg, b.Op)
		stages[c.Stage] = true
	}
	if len(stages) != 6 {
		t.Fatalf("smoke-e11 covers stages %v", stages)
	}
}

func TestCalibrateE11AddOn(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	st := &State{Schema: 1, RunID: "20261012-2000_mac-m1_colima-vz_b1-prod", Machine: "mac-m1", PlanKind: plan.KindTrial,
		Seed: 1, Sessions: []Session{{N: 1}}, CellReps: map[string]int{}, VMCPUs: 4, HostCPUs: 8, HostPeriodNS: 5e9,
		Class: map[string]string{}, Requeued: map[string]bool{}, Diagnosed: map[string]bool{}, Preps: map[string]PrepRec{}}
	must(st.save(dir))
	fast := cells.Cell{Stage: cells.Issuance, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 10, V: 16}
	slow := cells.Cell{Stage: cells.Issuance, Sig: "slh-dsa-sha2-128s", KEM: "ml-kem-512", N: 10, V: 16}
	hdr := []string{"utc_ns", "event", "key", "detail"}
	var t0 int64 = 10e9
	for _, c := range []struct {
		cell cells.Cell
		ns   int64
	}{{fast, 400e3}, {slow, 450e6}} {
		b := plan.Block{ID: "probe-E11." + c.cell.Alg() + "." + c.cell.Op() + ".r1", Exp: "probe-E11", Alg: c.cell.Alg(), Op: c.cell.Op(), Rep: 1, Attempt: 1}
		r := proto.Result{Key: b.Key(), Block: b, StartNS: t0, EndNS: t0 + 2e9, WallNS: 2e9, PrepNS: 3e9,
			Arms: []proto.ArmSummary{{Name: cells.ArmStage, ProbeNS: []int64{c.ns, c.ns, c.ns, c.ns, c.ns}}}}
		t0 += 3e9
		must(appendJSONL(filepath.Join(dir, fileBlocksJL), r))
		must(appendCSV(filepath.Join(dir, fileJournal), hdr, []string{strconv.FormatInt(r.StartNS, 10), "block_start", r.Key, ""}))
		must(appendCSV(filepath.Join(dir, fileJournal), hdr, []string{strconv.FormatInt(r.EndNS+1e8, 10), "block_done", r.Key, ""}))
	}
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

	base := quality.Defaults()
	base.CalibratedFrom = "20261010-1819_mac-m1_colima-vz_b1-prod"
	base.Class["ml-dsa-44/sign"] = "fast"
	cal, err := calibrate(dir, "mac-m1", base)
	if err != nil {
		t.Fatal(err)
	}
	m := cal.Thresholds
	if cal.AddOn != "E11" || m.ClassCalibratedFrom["E11"] != st.RunID || m.Class["ml-dsa-44/sign"] != "fast" {
		t.Fatalf("add-on markers %q %v %v", cal.AddOn, m.ClassCalibratedFrom, m.Class)
	}
	if m.Class[fast.Alg()+"/"+fast.Op()] != "fast" || m.Class[slow.Alg()+"/"+slow.Op()] != "slow" {
		t.Fatalf("classification %v", m.Class)
	}
	if len(base.Class) != 1 || base.ClassCalibratedFrom != nil {
		t.Fatal("the base thresholds were modified")
	}
	// The estimate covers E11 with the probe medians and preparation times.
	must(writeJSON(filepath.Join(dir, fileCalib), cal))
	plans := t.TempDir()
	os.WriteFile(filepath.Join(plans, "e11.json"), []byte(`{"name":"e11","kind":"main","description":"","reps":5,"items":[{"exp":"E11","algs":["ml-dsa-44","ml-kem-768","slh-dsa-sha2-128s","ml-kem-512"]}]}`), 0o644)
	// A plan the add-on trial never probed (E1) must be left out, not estimated
	// from nothing (that produced NaN and failed the 12 Oct trial-e11 run at
	// its last step).
	os.WriteFile(filepath.Join(plans, "e1.json"), []byte(`{"name":"e1","kind":"main","description":"","reps":5,"items":[{"exp":"E1","algs":["ml-dsa-44"]}]}`), 0o644)
	e, err := estimate(dir, plans, "mac-m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plans) != 1 || e.Plans[0].Plan != "e11" || !(e.Plans[0].ByExpHours["E11"] > 0) {
		t.Fatalf("estimate %+v", e.Plans)
	}
	if err := writeJSON(filepath.Join(dir, "estimate-test.json"), e); err != nil {
		t.Fatalf("the estimate must serialise: %v", err)
	}
}
