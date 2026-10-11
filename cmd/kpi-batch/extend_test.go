package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// finishedRun writes a finished main run with one exhausted block (E1 ... r1:
// three flagged attempts) and one healthy block (r2).
func finishedRun(t *testing.T) (*batch, string) {
	t.Helper()
	dir := t.TempDir()
	st := &State{Schema: 1, RunID: "20261010-2057_mac-m1_colima-vz_b1-prod", Machine: "mac-m1", PlanKind: plan.KindMain, Seed: 5,
		Status: "done", Sessions: []Session{{N: 1, StartNS: 0}}, CellReps: map[string]int{"E1.ml-dsa-44.sign": 2},
		VMCPUs: 4, HostCPUs: 8, HostPeriodNS: 5e9, Class: map[string]string{}, Requeued: map[string]bool{},
		Diagnosed: map[string]bool{}, Preps: map[string]PrepRec{}, ImageID: "sha256:old", HarnessCommit: "oldcommit"}
	var t0 int64 = 10e9
	add := func(rep, att int, steal float64) {
		b := plan.Block{ID: "E1.ml-dsa-44.sign.r" + strconv.Itoa(rep), Exp: "E1", Alg: "ml-dsa-44", Op: "sign", Rep: rep, Attempt: att,
			Profile: plan.ProfileApproved, Mode: "fast"}
		r := proto.Result{Key: b.Key(), Block: b, StartNS: t0, EndNS: t0 + 2e9, WallNS: 2e9,
			Arms: []proto.ArmSummary{{Name: proto.ArmReuse, Stats: stats.Summary{N: 10000, P50: 60000, BootSE50: 50}}}}
		r.Quality.StealPct = steal
		t0 += 3e9
		if err := appendJSONL(filepath.Join(dir, fileBlocksJL), r); err != nil {
			t.Fatal(err)
		}
	}
	for a := 1; a <= 3; a++ {
		add(1, a, 9) // flagged: steal above 2 %
	}
	add(2, 1, 0)
	canary := canaryRec{Seq: 1, Ratio: 1}
	canary.StartNS, canary.EndNS, canary.MedianRate = t0+1e9, t0+2e9, 2.5e6
	if err := appendJSONL(filepath.Join(dir, fileCanaryJL), canary); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(dir, dirMonitor), 0o755)
	rows := "utc_ns,other_pct,throttle,on_ac,alert,detail\n"
	for ts := int64(5e9); ts <= t0+10e9; ts += 5e9 {
		rows += strconv.FormatInt(ts, 10) + ",1.0,0,1,0,x\n"
	}
	os.WriteFile(filepath.Join(dir, dirMonitor, "host-s1-000001.csv"), []byte(rows), 0o644)
	os.WriteFile(filepath.Join(dir, dirMonitor, "host-s1-000001.ok"), nil, 0o644)
	b := &batch{c: runConfig{Out: dir, Seed: 5, ImageID: "sha256:new"}, st: st, th: quality.Defaults(),
		sess: Session{N: 2}, p: &plan.Plan{Name: "e1", Kind: plan.KindMain}}
	st.Sessions = append(st.Sessions, b.sess)
	return b, dir
}

func TestExtendQueuesOnlyBlocksWithoutHeadline(t *testing.T) {
	b, _ := finishedRun(t)
	if err := b.extendBlocks([]string{"E1.ml-dsa-44.sign.r2"}, "sha256:old", "oldcommit"); err == nil {
		t.Fatal("a block with a clean headline attempt was extended")
	}
	if err := b.extendBlocks([]string{"E1.ml-dsa-44.sign.r9"}, "", ""); err == nil {
		t.Fatal("an unknown block was extended")
	}
	if err := b.extendBlocks([]string{"nonsense"}, "", ""); err == nil {
		t.Fatal("a malformed id was accepted")
	}
	if err := b.extendBlocks([]string{"E1.ml-dsa-44.sign.r1"}, "sha256:old", "oldcommit"); err != nil {
		t.Fatal(err)
	}
	if len(b.st.Queue) != 1 || b.st.Queue[0].Attempt != 4 || b.st.Queue[0].Diagnostic || b.st.Base["E1.ml-dsa-44.sign.r1"] != 3 {
		t.Fatalf("queue %+v base %v", b.st.Queue, b.st.Base)
	}
	if b.st.Queue[0].Seed != plan.DeriveSeed(5, "E1.ml-dsa-44.sign.r1.a4") {
		t.Fatal("extension seed is not derived from the attempt key")
	}
	if len(b.st.Extensions) != 1 || b.st.Extensions[0].OldImageID != "sha256:old" || b.st.Planned != 1 {
		t.Fatalf("extension record %+v", b.st.Extensions)
	}
	if n := extensionNotes(b.st); len(n) != 2 {
		t.Fatalf("notes %v", n)
	}
}

func TestExtendedBlockGetsAFreshCapOfThreeAttempts(t *testing.T) {
	b, dir := finishedRun(t)
	if err := b.extendBlocks([]string{"E1.ml-dsa-44.sign.r1"}, "", ""); err != nil {
		t.Fatal(err)
	}
	// All of a4, a5, a6 are flagged: after a4 and a5 a repeat is queued, after
	// a6 the block is exhausted again (6 attempts in total, 3 since the extension).
	results, _ := readJSONL[proto.Result](filepath.Join(dir, fileBlocksJL))
	last := results[len(results)-1].EndNS
	var queued []int
	for att := 4; att <= 6; att++ {
		bl := plan.Block{ID: "E1.ml-dsa-44.sign.r1", Exp: "E1", Alg: "ml-dsa-44", Op: "sign", Rep: 1, Attempt: att, Profile: plan.ProfileApproved, Mode: "fast"}
		r := proto.Result{Key: bl.Key(), Block: bl, StartNS: last + 3e9, EndNS: last + 5e9, WallNS: 2e9,
			Arms: []proto.ArmSummary{{Name: proto.ArmReuse, Stats: stats.Summary{N: 10000, P50: 60000, BootSE50: 50}}}}
		r.Quality.StealPct = 9
		last += 3e9
		if err := appendJSONL(filepath.Join(dir, fileBlocksJL), r); err != nil {
			t.Fatal(err)
		}
		in, err := loadEval(dir, b.st, b.th, true)
		if err != nil {
			t.Fatal(err)
		}
		ds := decideBase(plan.KindMain, evaluate(in), 5, copyMap(b.st.Requeued), copyMap(b.st.Diagnosed), b.st.Base)
		for _, d := range ds {
			queued = append(queued, d.Block.Attempt)
		}
	}
	if len(queued) != 2 || queued[0] != 5 || queued[1] != 6 {
		t.Fatalf("repeats queued for attempts %v, want [5 6]", queued)
	}
	// Without the extension base the old cap (3) would already be used up.
	in, _ := loadEval(dir, b.st, b.th, true)
	if ds := decide(plan.KindMain, evaluate(in), 5, map[string]bool{}, map[string]bool{}); len(ds) != 0 {
		t.Fatalf("plain decide queued %+v for a block with 6 attempts", ds)
	}
}

func TestExtendIdentityIgnoresOnlyTheBuild(t *testing.T) {
	saved := identity{"r", "mac-m1", "p", "img-old", "c-old", "t", 1}
	now := identity{"r", "mac-m1", "p", "img-new", "c-new", "t", 1}
	if len(resumeProblems(saved, now)) != 2 {
		t.Fatal("plain resume must refuse a changed image and commit")
	}
	if p := resumeProblems(saved, withoutBuild(now, saved)); len(p) != 0 {
		t.Fatalf("problems %v", p)
	}
	now.ThresholdsHash = "t2"
	if p := resumeProblems(saved, withoutBuild(now, saved)); len(p) != 1 {
		t.Fatalf("a changed thresholds file must still refuse: %v", p)
	}
}
