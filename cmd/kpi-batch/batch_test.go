package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// result builds a synthetic E1 attempt running from start to end (ns) with
// block median p50.
func result(id string, rep, attempt int, diag bool, start, end int64, p50 float64) proto.Result {
	parts := strings.Split(id, ".")
	b := plan.Block{ID: id, Exp: parts[0], Alg: parts[1], Op: parts[2], Rep: rep, Attempt: attempt, Diagnostic: diag,
		Profile: plan.ProfileApproved, Mode: "fast"}
	return proto.Result{Key: b.Key(), Block: b, StartNS: start, EndNS: end,
		Arms: []proto.ArmSummary{{Name: proto.ArmReuse, Stats: stats.Summary{N: 10000, P50: p50, BootSE50: p50 * 0.001}}}}
}

func canaryAt(seq int, start, end int64, ratio float64) canaryRec {
	c := canaryRec{Seq: seq, Ratio: ratio}
	c.StartNS, c.EndNS, c.MedianRate = start, end, 1e6*ratio
	return c
}

// hostEvery returns host samples every 5 s from 0 to last (host clock = container clock).
func hostEvery(last int64, other float64) []quality.HostSample {
	var s []quality.HostSample
	for t := int64(5e9); t <= last; t += 5e9 {
		s = append(s, quality.HostSample{UnixNS: t, OtherPct: other, Throttle: 0, OnAC: 1})
	}
	return s
}

func baseInput(kind string, results []proto.Result, final bool) evalInput {
	return evalInput{Kind: kind, Results: results,
		Canaries:  []canaryRec{canaryAt(1, 0, 1e9, 1), canaryAt(2, 900e9, 901e9, 1)},
		Host:      hostEvery(1000e9, 1),
		Sessions:  []Session{{N: 1, StartNS: 0}},
		TH:        quality.Defaults(),
		CellReps:  map[string]int{"E1.ml-dsa-44.sign": 3},
		Final:     final,
		HostCPUs:  8, VMCPUs: 4, PeriodNS: 5e9, HostCheck: kind != plan.KindSmoke}
}

func TestCleanBlocksNeedNoAction(t *testing.T) {
	rs := []proto.Result{
		result("E1.ml-dsa-44.sign.r1", 1, 1, false, 10e9, 20e9, 100),
		result("E1.ml-dsa-44.sign.r2", 2, 1, false, 30e9, 40e9, 101),
		result("E1.ml-dsa-44.sign.r3", 3, 1, false, 50e9, 60e9, 99),
	}
	vs := evaluate(baseInput(plan.KindMain, rs, false))
	ds := decide(plan.KindMain, vs, 1, map[string]bool{}, map[string]bool{})
	if len(ds) != 0 {
		t.Fatalf("unexpected decisions %+v", ds)
	}
	for _, v := range vs {
		if !v.Headline || v.flagged() || v.ExtPending || v.Drift {
			t.Fatalf("verdict %+v", v)
		}
	}
}

func TestExternalFlagRepeatsAtMostTwice(t *testing.T) {
	in := baseInput(plan.KindMain, []proto.Result{result("E1.ml-dsa-44.sign.r1", 1, 1, false, 10e9, 20e9, 100)}, false)
	in.Results[0].Quality.StealPct = 7 // above the 2 % limit
	requeued := map[string]bool{}
	ds := decide(plan.KindMain, evaluate(in), 1, requeued, map[string]bool{})
	if len(ds) != 1 || ds[0].Block.Attempt != 2 || ds[0].Block.Diagnostic || !strings.Contains(ds[0].Reason, "steal") {
		t.Fatalf("decisions %+v", ds)
	}
	if ds[0].Block.Seed != plan.DeriveSeed(1, "E1.ml-dsa-44.sign.r1.a2") {
		t.Fatal("repeat seed is not derived from its key")
	}
	// The same evaluation again must not queue a second repeat.
	if again := decide(plan.KindMain, evaluate(in), 1, requeued, map[string]bool{}); len(again) != 0 {
		t.Fatalf("repeat queued twice: %+v", again)
	}
	// Attempts 2 and 3 also flagged: the block is exhausted, nothing more.
	r2 := result("E1.ml-dsa-44.sign.r1", 1, 2, false, 30e9, 40e9, 100)
	r2.Quality.StealPct = 9
	in.Results = append(in.Results, r2)
	ds = decide(plan.KindMain, evaluate(in), 1, requeued, map[string]bool{})
	if len(ds) != 1 || ds[0].Block.Attempt != 3 {
		t.Fatalf("third attempt not queued: %+v", ds)
	}
	r3 := result("E1.ml-dsa-44.sign.r1", 1, 3, false, 50e9, 60e9, 100)
	r3.Quality.StealPct = 9
	in.Results = append(in.Results, r3)
	vs := evaluate(in)
	ds = decide(plan.KindMain, vs, 1, requeued, map[string]bool{})
	if len(ds) != 0 {
		t.Fatalf("a fourth attempt was queued: %+v", ds)
	}
	if vs[2].Action != "exhausted" || vs[0].Headline || vs[1].Headline || vs[2].Headline {
		t.Fatalf("exhausted block: %+v / %+v / %+v", vs[0], vs[1], vs[2])
	}
}

func TestRepeatBecomesHeadline(t *testing.T) {
	r1 := result("E1.ml-dsa-44.sign.r1", 1, 1, false, 10e9, 20e9, 100)
	r1.Quality.OtherPct = 40 // VM-other share 40 * 4/8 = 20 % > 5 %
	r2 := result("E1.ml-dsa-44.sign.r1", 1, 2, false, 30e9, 40e9, 100)
	in := baseInput(plan.KindMain, []proto.Result{r1, r2}, false)
	vs := evaluate(in)
	decide(plan.KindMain, vs, 1, map[string]bool{r1.Key: true}, map[string]bool{})
	if vs[0].Headline || !vs[1].Headline {
		t.Fatalf("headline should be attempt 2: %+v %+v", vs[0], vs[1])
	}
	if vs[0].External[0] != "other" {
		t.Fatalf("flags %v", vs[0].External)
	}
}

func TestPendingUntilHostAndCanaryCover(t *testing.T) {
	r := result("E1.ml-dsa-44.sign.r1", 1, 1, false, 950e9, 960e9, 100) // after the last canary
	in := baseInput(plan.KindMain, []proto.Result{r}, false)
	in.Host = hostEvery(955e9, 1) // host samples stop before the block ends
	vs := evaluate(in)
	if !vs[0].ExtPending {
		t.Fatal("block should be pending")
	}
	if ds := decide(plan.KindMain, vs, 1, map[string]bool{}, map[string]bool{}); len(ds) != 0 {
		t.Fatalf("pending block acted on: %+v", ds)
	}
	// Final evaluation: missing host coverage becomes an external flag.
	in.Final = true
	vs = evaluate(in)
	if vs[0].ExtPending || len(vs[0].External) != 1 || vs[0].External[0] != "host_missing" {
		t.Fatalf("final verdict %+v", vs[0])
	}
}

func TestCanaryAndCoreFlags(t *testing.T) {
	r := result("E1.ml-dsa-44.sign.r1", 1, 1, false, 10e9, 20e9, 100)
	in := baseInput(plan.KindMain, []proto.Result{r}, false)
	in.Canaries[1].Ratio = 0.95 // 5 % slower than the first canary, limit 3 %
	in.Canaries[0].Core = "E-like"
	vs := evaluate(in)
	got := strings.Join(vs[0].External, ",")
	if got != "canary,core" {
		t.Fatalf("flags %q", got)
	}
}

func TestDriftDiagnosticMainOnly(t *testing.T) {
	rs := []proto.Result{
		result("E1.ml-dsa-44.sign.r1", 1, 1, false, 10e9, 20e9, 100),
		result("E1.ml-dsa-44.sign.r2", 2, 1, false, 30e9, 40e9, 100),
		result("E1.ml-dsa-44.sign.r3", 3, 1, false, 50e9, 60e9, 120), // 20 % off the reference
	}
	vs := evaluate(baseInput(plan.KindMain, rs, false))
	if !vs[2].Drift || vs[0].Drift {
		t.Fatalf("drift %v %v", vs[0].Drift, vs[2].Drift)
	}
	diagnosed := map[string]bool{}
	ds := decide(plan.KindMain, vs, 1, map[string]bool{}, diagnosed)
	if len(ds) != 1 || !ds[0].Block.Diagnostic || ds[0].Block.Attempt != 2 || !vs[2].Headline {
		t.Fatalf("diagnostic decision %+v (headline kept: %v)", ds, vs[2].Headline)
	}
	if again := decide(plan.KindMain, evaluate(baseInput(plan.KindMain, rs, false)), 1, map[string]bool{}, diagnosed); len(again) != 0 {
		t.Fatal("diagnostic queued twice")
	}
	// A diagnostic attempt never replaces the headline.
	d := result("E1.ml-dsa-44.sign.r3", 3, 2, true, 70e9, 80e9, 100)
	vs = evaluate(baseInput(plan.KindMain, append(rs, d), false))
	decide(plan.KindMain, vs, 1, map[string]bool{}, diagnosed)
	if !vs[2].Headline || vs[3].Headline {
		t.Fatal("diagnostic attempt took the headline")
	}
	// Trial: drift is recorded only.
	vs = evaluate(baseInput(plan.KindTrial, rs, false))
	if ds := decide(plan.KindTrial, vs, 1, map[string]bool{}, map[string]bool{}); len(ds) != 0 || vs[2].Action != "record" {
		t.Fatalf("trial drift: %+v action %q", ds, vs[2].Action)
	}
	// Reference not frozen until all first attempts exist.
	vs = evaluate(baseInput(plan.KindMain, rs[:2], false))
	if !vs[0].DriftPending {
		t.Fatal("drift should wait for every first attempt")
	}
}

func TestSmokeNeverQueues(t *testing.T) {
	r := result("E1.ml-dsa-44.sign.r1", 1, 1, false, 10e9, 20e9, 100)
	r.Quality.StealPct = 50
	in := baseInput(plan.KindSmoke, []proto.Result{r}, true)
	vs := evaluate(in)
	if ds := decide(plan.KindSmoke, vs, 1, map[string]bool{}, map[string]bool{}); len(ds) != 0 || vs[0].Action != "record" {
		t.Fatalf("smoke acted: %+v %q", ds, vs[0].Action)
	}
}

func TestInsertAtDeterministicAndInRange(t *testing.T) {
	for n := 0; n < 50; n++ {
		p := insertAt(7, "E1.x.sign.r1.a2", n)
		if p < 0 || p > n || p != insertAt(7, "E1.x.sign.r1.a2", n) {
			t.Fatalf("position %d for queue length %d", p, n)
		}
	}
}

func TestResumeRefusesChanges(t *testing.T) {
	a := identity{"r", "mac-m1", "p", "img", "c", "t", 1}
	if len(resumeProblems(a, a)) != 0 {
		t.Fatal("identical batches refused")
	}
	for _, b := range []identity{
		{"r2", "mac-m1", "p", "img", "c", "t", 1}, {"r", "vm-x86v2", "p", "img", "c", "t", 1},
		{"r", "mac-m1", "p2", "img", "c", "t", 1}, {"r", "mac-m1", "p", "img2", "c", "t", 1},
		{"r", "mac-m1", "p", "img", "c2", "t", 1}, {"r", "mac-m1", "p", "img", "c", "t2", 1},
		{"r", "mac-m1", "p", "img", "c", "t", 2},
	} {
		if len(resumeProblems(a, b)) != 1 {
			t.Fatalf("change not detected: %+v", b)
		}
	}
}

func TestSumsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "raw", "E1"), 0o755)
	os.WriteFile(filepath.Join(dir, "raw", "E1", "a.csv.gz"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "blocks.csv"), []byte("y"), 0o644)
	os.WriteFile(filepath.Join(dir, ".tmp-z"), []byte("z"), 0o644)
	n, err := writeSums(dir)
	if err != nil || n != 2 {
		t.Fatalf("writeSums %d %v", n, err)
	}
	if c, un, err := verifySums(dir); err != nil || c != 2 || len(un) != 0 {
		t.Fatalf("verify %d %v %v", c, un, err)
	}
	os.WriteFile(filepath.Join(dir, "console.txt"), []byte("late"), 0o644)
	if _, un, err := verifySums(dir); err != nil || len(un) != 1 || un[0] != "console.txt" {
		t.Fatalf("unlisted %v %v", un, err)
	}
	os.WriteFile(filepath.Join(dir, "blocks.csv"), []byte("tampered"), 0o644)
	if _, _, err := verifySums(dir); err == nil {
		t.Fatal("tampering not detected")
	}
}

func TestReadJSONLToleratesCutLastLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.jsonl")
	os.WriteFile(p, []byte("{\"seq\":1}\n{\"seq\":2}\n{\"se"), 0o644)
	got, err := readJSONL[canaryRec](p)
	if err != nil || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	os.WriteFile(p, []byte("{\"seq\":1}\nbad\n{\"seq\":2}\n"), 0o644)
	if _, err := readJSONL[canaryRec](p); err == nil {
		t.Fatal("corrupt middle line accepted")
	}
}

func TestSessionFileAllowList(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, sessionFileName), []byte(`{"host_offset_ns":5,"host":{"runtime":"colima-vz"}}`), 0o644)
	sf, err := readSessionFile(dir)
	if err != nil || sf.HostOffsetNS != 5 || sf.Host["runtime"] != "colima-vz" {
		t.Fatalf("%+v %v", sf, err)
	}
	os.WriteFile(filepath.Join(dir, sessionFileName), []byte(`{"host":{"hostname":"x"}}`), 0o644)
	if _, err := readSessionFile(dir); err == nil {
		t.Fatal("a host fact outside the allow-list was accepted")
	}
	os.WriteFile(filepath.Join(dir, sessionFileName), []byte(`{"user":"x"}`), 0o644)
	if _, err := readSessionFile(dir); err == nil {
		t.Fatal("an unknown session field was accepted")
	}
}

func TestStrayFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, dirMonitor), 0o755)
	os.WriteFile(filepath.Join(dir, sessionFileName), []byte("{}"), 0o644)
	if s, _ := strayFiles(dir); len(s) != 0 {
		t.Fatalf("launcher files counted as stray: %v", s)
	}
	os.WriteFile(filepath.Join(dir, "blocks.csv"), nil, 0o644)
	if s, _ := strayFiles(dir); len(s) != 1 {
		t.Fatalf("stray %v", s)
	}
}

func TestRunIDPattern(t *testing.T) {
	if !runIDPattern.MatchString("20261010-1830_mac-m1_colima-vz_b1-prod") {
		t.Fatal("valid run id rejected")
	}
	for _, bad := range []string{"x", "20261010-1830_mac-m1_colima vz_b1", "../x_mac-m1_a_b"} {
		if runIDPattern.MatchString(bad) {
			t.Fatalf("bad run id %q accepted", bad)
		}
	}
}
