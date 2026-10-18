package main

import (
	"path/filepath"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// TestE5RunsOnlyNatively: E5 is refused in the container, and natively only
// E5 (with its smoke plan's probe pass) runs.
func TestE5RunsOnlyNatively(t *testing.T) {
	load := func(name string) []plan.Block {
		p, err := plan.Load(filepath.Join("..", "..", "plans", name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := plan.Expand(p, 1)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	e5, smoke, e1 := load("e5.json"), load("smoke-e5.json"), load("e1.json")
	if len(e5) != 90*5 {
		t.Fatalf("%d E5 blocks", len(e5))
	}
	for _, c := range []struct {
		blocks []plan.Block
		goos   string
		ok     bool
	}{
		{e5, "darwin", true}, {smoke, "darwin", true}, {e5, "linux", false},
		{e1, "darwin", false}, {e1, "linux", true},
	} {
		if err := e5PlatformCheck(c.blocks, c.goos); (err == nil) != c.ok {
			t.Fatalf("%s on %s (%d blocks): %v", c.blocks[0].Exp, c.goos, len(c.blocks), err)
		}
	}
}

// TestNativeOtherCPU: natively the host monitor and the block's own reading
// describe the same machine, so the larger counts (in the container they add).
func TestNativeOtherCPU(t *testing.T) {
	bl := plan.Block{ID: "E5.ml-dsa-44.sign.r1", Exp: "E5", Alg: "ml-dsa-44", Op: "sign", Rep: 1, Attempt: 1, Mode: "fast"}
	r := proto.Result{Key: bl.Key(), Block: bl, StartNS: 10e9, EndNS: 20e9, Quality: quality.Interval{OtherPct: 3},
		Arms: []proto.ArmSummary{{Name: proto.ArmReuse, Stats: stats.Summary{N: 10000, P50: 60e3}}}}
	var host []quality.HostSample
	for ts := int64(5e9); ts <= 30e9; ts += 5e9 {
		host = append(host, quality.HostSample{UnixNS: ts, OtherPct: 4, OnAC: 1})
	}
	in := evalInput{Kind: plan.KindMain, Results: []proto.Result{r}, Host: host, Sessions: []Session{{N: 1}},
		TH: quality.Defaults(), CellReps: map[string]int{}, HostCPUs: 8, VMCPUs: 8, PeriodNS: 5e9, HostCheck: true, Final: true}
	if v := evaluate(in)[0]; v.CombinedOther != 7 || !v.flagged() {
		t.Fatalf("container: combined other %v, flagged %v", v.CombinedOther, v.flagged())
	}
	in.Native = true
	if v := evaluate(in)[0]; v.CombinedOther != 4 || v.flagged() {
		t.Fatalf("native: combined other %v, flagged %v %v", v.CombinedOther, v.flagged(), v.External)
	}
}
