package main

import (
	"math"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// TestE12FlagsAndRepeats: E12 is timed, so a block measured while the host
// was busy is flagged and repeated; it has no drift rule.
func TestE12FlagsAndRepeats(t *testing.T) {
	bl := plan.Block{ID: "E12.ml-dsa-44+ml-kem-768.verification-n10-k5-v16-w2.r1", Exp: "E12",
		Alg: "ml-dsa-44+ml-kem-768", Op: "verification-n10-k5-v16-w2", Rep: 1, Attempt: 1, Mode: "window"}
	r := proto.Result{Key: bl.Key(), Block: bl, StartNS: 10e9, EndNS: 20e9,
		Arms: []proto.ArmSummary{{Name: proto.QLatency, Stats: stats.Summary{N: 9000, P50: 300e3}}}}
	var host []quality.HostSample
	for ts := int64(5e9); ts <= 30e9; ts += 5e9 {
		host = append(host, quality.HostSample{UnixNS: ts, OtherPct: 40, OnAC: 1})
	}
	in := evalInput{Kind: plan.KindMain, Results: []proto.Result{r}, Host: host, Sessions: []Session{{N: 1}},
		TH: quality.Defaults(), CellReps: map[string]int{}, HostCPUs: 8, VMCPUs: 4, PeriodNS: 5e9, HostCheck: true, Final: true}
	vs := evaluate(in)
	if len(vs) != 1 || !vs[0].flagged() || vs[0].Drift || vs[0].DriftPending {
		t.Fatalf("E12 verdict %+v", vs[0])
	}
	ds := decideBase(plan.KindMain, vs, 1, map[string]bool{}, map[string]bool{}, nil)
	if len(ds) != 1 || ds[0].Block.Attempt != 2 {
		t.Fatalf("E12 block was not repeated: %+v", ds)
	}
	if rows := qualityRows(vs); rows[0][10] != "n/a" {
		t.Fatalf("drift column %q, want n/a", rows[0][10])
	}
}

// TestE12NeedsNoClassificationBatchesOrPrep: a main E12 run starts with the
// window mode and no thresholds of its own.
func TestE12NeedsNoClassificationBatchesOrPrep(t *testing.T) {
	b := &batch{p: &plan.Plan{Kind: plan.KindMain}, th: quality.Defaults()}
	bl := plan.Block{ID: "E12.ecdsa-p256+dhkem-p256.issuance-n10-v16-w4.r1", Exp: "E12", Alg: "ecdsa-p256+dhkem-p256",
		Op: "issuance-n10-v16-w4", Rep: 1}
	if m := b.missingClassification([]plan.Block{bl}); len(m) != 0 {
		t.Fatalf("missing %v", m)
	}
	if m, err := b.modeFor(bl); err != nil || m != "window" {
		t.Fatalf("mode %q %v", m, err)
	}
	if b.batchFor(bl) != nil {
		t.Fatal("E12 has no batches")
	}
	if err := b.ensurePrep(bl); err != nil {
		t.Fatalf("E12 prepares its own inputs: %v", err)
	}
}

// TestE12Estimate: warm-up + window + each worker's preparation.
func TestE12Estimate(t *testing.T) {
	x := &est{f: &trialFacts{med: map[string]float64{}, prep: map[string]float64{
		"ml-dsa-44+ml-kem-768/verification-n10-k5-v16": 0.2e9}, overheadNS: 0.3e9}, missing: map[string]bool{}}
	s, _ := x.e12Block(plan.Block{Exp: "E12", Alg: "ml-dsa-44+ml-kem-768", Op: "verification-n10-k5-v16-w4", Profile: plan.ProfileApproved})
	if math.Abs(s-(0.3+4*0.2+35)) > 1e-9 {
		t.Fatalf("estimate %v s", s)
	}
	s, _ = x.e12Block(plan.Block{Exp: "E12", Alg: "falcon-512+ml-kem-512", Op: "issuance-n10-v16-w2", Profile: plan.ProfileApproved})
	if math.Abs(s-(0.3+2*1+35)) > 1e-9 {
		t.Fatalf("estimate without an E11 probe %v s", s)
	}
}
