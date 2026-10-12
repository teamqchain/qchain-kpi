package main

import (
	"path/filepath"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// TestE8IsNeverFlaggedOrRepeated: sizes do not depend on machine noise, so an
// E8 block measured while the host was busy is still the result.
func TestE8IsNeverFlaggedOrRepeated(t *testing.T) {
	bl := plan.Block{ID: "E8.falcon-512.sign.r1", Exp: "E8", Alg: "falcon-512", Op: "sign", Rep: 1, Attempt: 1, Mode: "sizes"}
	r := proto.Result{Key: bl.Key(), Block: bl, StartNS: 10e9, EndNS: 20e9,
		Arms: []proto.ArmSummary{{Name: proto.ArmSig, Stats: stats.Summary{N: 10000, P50: 655}, MinBytes: 617, MaxBytes: 690}}}
	var host []quality.HostSample
	for ts := int64(5e9); ts <= 30e9; ts += 5e9 {
		host = append(host, quality.HostSample{UnixNS: ts, OtherPct: 40, OnAC: 1})
	}
	in := evalInput{Kind: plan.KindMain, Results: []proto.Result{r}, Host: host, Sessions: []Session{{N: 1}},
		TH: quality.Defaults(), CellReps: map[string]int{}, HostCPUs: 8, VMCPUs: 4, PeriodNS: 5e9, HostCheck: true}
	vs := evaluate(in)
	if len(vs) != 1 || vs[0].flagged() || vs[0].ExtPending || vs[0].Drift {
		t.Fatalf("E8 verdict %+v", vs[0])
	}
	if ds := decideBase(plan.KindMain, vs, 1, map[string]bool{}, map[string]bool{}, nil); len(ds) != 0 {
		t.Fatalf("E8 block was requeued: %+v", ds)
	}
	if !vs[0].Headline {
		t.Fatal("the first E8 attempt must be the result")
	}
	if rows := qualityRows(vs); rows[0][10] != "n/a" {
		t.Fatalf("drift column %q, want n/a", rows[0][10])
	}
}

func TestE8PlansExpand(t *testing.T) {
	root := filepath.Join("..", "..", "plans")
	for _, name := range []string{"e8.json", "smoke-e8.json"} {
		p, err := plan.Load(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		blocks, err := plan.Expand(p, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(blocks) != 2*len(algs.All()) {
			t.Fatalf("%s: %d blocks, want %d", name, len(blocks), 2*len(algs.All()))
		}
		for _, b := range blocks {
			if b.Op != "keygen" && b.Op != "sign" && b.Op != "encaps" {
				t.Fatalf("%s: unexpected op %s", name, b.Op)
			}
		}
	}
	bad := &plan.Plan{Name: "x", Kind: plan.KindMain, Reps: 1, Items: []plan.Item{{Exp: "E8", Algs: []string{"ml-dsa-44"}, Ops: []string{"verify"}, Profile: plan.ProfileApproved}}}
	if _, err := plan.Expand(bad, 1); err == nil {
		t.Fatal("E8 must refuse verify")
	}
}
