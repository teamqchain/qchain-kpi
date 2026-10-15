package main

import (
	"math"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// TestE9IsNeverFlaggedOrRepeated: memory does not depend on machine noise, so
// an E9 block measured while the host was busy is still the result.
func TestE9IsNeverFlaggedOrRepeated(t *testing.T) {
	bl := plan.Block{ID: "E9.ml-dsa-44.sign-c.r1", Exp: "E9", Alg: "ml-dsa-44", Op: "sign-c", Rep: 1, Attempt: 1, Mode: "memory"}
	r := proto.Result{Key: bl.Key(), Block: bl, StartNS: 10e9, EndNS: 20e9,
		Arms: []proto.ArmSummary{{Name: proto.QStack, Stats: stats.Summary{N: 99, P50: 40000}, MinBytes: 39000, MaxBytes: 41000}}}
	var host []quality.HostSample
	for ts := int64(5e9); ts <= 30e9; ts += 5e9 {
		host = append(host, quality.HostSample{UnixNS: ts, OtherPct: 40, OnAC: 1})
	}
	in := evalInput{Kind: plan.KindMain, Results: []proto.Result{r}, Host: host, Sessions: []Session{{N: 1}},
		TH: quality.Defaults(), CellReps: map[string]int{}, HostCPUs: 8, VMCPUs: 4, PeriodNS: 5e9, HostCheck: true}
	vs := evaluate(in)
	if len(vs) != 1 || vs[0].flagged() || vs[0].ExtPending || vs[0].Drift {
		t.Fatalf("E9 verdict %+v", vs[0])
	}
	if ds := decideBase(plan.KindMain, vs, 1, map[string]bool{}, map[string]bool{}, nil); len(ds) != 0 {
		t.Fatalf("E9 block was requeued: %+v", ds)
	}
	if !vs[0].Headline {
		t.Fatal("the first E9 attempt must be the result")
	}
	if rows := qualityRows(vs); rows[0][10] != "n/a" {
		t.Fatalf("drift column %q, want n/a", rows[0][10])
	}
}

// TestE9NeedsNoClassificationOrBatches: a main E9 run starts without any
// fast/slow class or batch marker, with the memory mode and the small pool.
func TestE9NeedsNoClassificationOrBatches(t *testing.T) {
	b := &batch{p: &plan.Plan{Kind: plan.KindMain}, th: quality.Defaults()}
	bl := plan.Block{ID: "E9.slh-dsa-sha2-128s.sign-prod.r1", Exp: "E9", Alg: "slh-dsa-sha2-128s", Op: "sign-prod", Rep: 1}
	if m := b.missingClassification([]plan.Block{bl}); len(m) != 0 {
		t.Fatalf("missing %v", m)
	}
	if m, err := b.modeFor(bl); err != nil || m != "memory" {
		t.Fatalf("mode %q %v", m, err)
	}
	if b.batchFor(bl) != nil {
		t.Fatal("E9 has no batches")
	}
	if proto.E9PoolSize(algs.MustByID("slh-dsa-sha2-128s")) != 16 || proto.E9PoolSize(algs.MustByID("ml-kem-768")) != 32 {
		t.Fatal("E9 pool sizes")
	}
}

// TestE9Estimate: 100 calls of a fast operation; a slow one stops at 20 s.
func TestE9Estimate(t *testing.T) {
	x := &est{f: &trialFacts{med: map[string]float64{
		"ml-dsa-44/sign": 100e3, "ml-dsa-44/verify": 40e3,
		"slh-dsa-sha2-128s/sign": 400e6, "slh-dsa-sha2-128s/verify": 1e6,
	}, overheadNS: 0.3e9}, missing: map[string]bool{}}
	bl := plan.Block{Exp: "E9", Alg: "ml-dsa-44", Op: "sign-c", Profile: plan.ProfileApproved}
	s, n := x.e9Block(bl, algs.MustByID("ml-dsa-44"))
	if n != 100 || math.Abs(s-(0.3+100*(100e3+40e3+e9CallNS)/1e9)) > 1e-9 {
		t.Fatalf("ml-dsa-44: %v s, %v calls", s, n)
	}
	bl = plan.Block{Exp: "E9", Alg: "slh-dsa-sha2-128s", Op: "sign-prod", Profile: plan.ProfileApproved}
	s, n = x.e9Block(bl, algs.MustByID("slh-dsa-sha2-128s"))
	if math.Abs(n-20e9/(400e6+1e6+e9CallNS)) > 1e-9 || s < 20 || s > 21 {
		t.Fatalf("slh-dsa-sha2-128s: %v s, %v calls", s, n)
	}
	if len(x.missing) != 0 {
		t.Fatalf("missing %v", x.missing)
	}
}
