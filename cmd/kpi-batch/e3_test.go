package main

import (
	"math"
	"path/filepath"
	"testing"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// TestE3UsesE1Classification: a main E3 run needs only E1's fast/slow classes;
// the null block is always fast, needs no class, no batch from the thresholds
// and no input preparation.
func TestE3UsesE1Classification(t *testing.T) {
	th := quality.Defaults()
	th.Class = map[string]string{"ml-dsa-44/sign": "fast"}
	b := &batch{p: &plan.Plan{Kind: plan.KindMain}, th: th}
	sign := plan.Block{ID: "E3.ml-dsa-44.sign.r1", Exp: "E3", Alg: "ml-dsa-44", Op: "sign", Rep: 1}
	null := plan.Block{ID: "E3.none.null.r1", Exp: "E3", Alg: plan.E3NullAlg, Op: plan.E3NullOp, Rep: 1}
	if m := b.missingClassification([]plan.Block{sign, null}); len(m) != 0 {
		t.Fatalf("missing %v", m)
	}
	if m, err := b.modeFor(sign); err != nil || m != "fast" {
		t.Fatalf("sign mode %q %v", m, err)
	}
	if m, err := b.modeFor(null); err != nil || m != "fast" {
		t.Fatalf("null mode %q %v", m, err)
	}
	if err := b.ensurePrep(null); err != nil {
		t.Fatal(err)
	}
	if got := proto.ArmsOfBlock("E3", plan.E3NullAlg, plan.E3NullOp); len(got) != 2 {
		t.Fatalf("null arms %v", got)
	}
	if got := proto.ArmsOf("E3", algs.MustByID("dhkem-p256"), "decaps"); len(got) != 4 || got[3] != proto.ArmL2 {
		t.Fatalf("classical arms %v", got)
	}
	for _, id := range []string{"E3.ml-dsa-44.sign.r2", "E3.none.null.r5"} {
		if !blockIDPattern.MatchString(id) {
			t.Fatalf("%s cannot be extended", id)
		}
	}
	x := &est{f: &trialFacts{med: map[string]float64{}, overheadNS: 0.3e9}, missing: map[string]bool{}}
	if s, _ := x.e3NullBlock(null); math.IsNaN(s) || s < 0.3 || s > 5 {
		t.Fatalf("null block estimate %v s", s)
	}
}

func TestE3PlansExpand(t *testing.T) {
	for _, c := range []struct {
		name   string
		blocks int
	}{{"e3.json", 5 * (90 + 1)}, {"smoke-e3.json", 0}} {
		p, err := plan.Load(filepath.Join("..", "..", "plans", c.name))
		if err != nil {
			t.Fatal(err)
		}
		bl, err := plan.Expand(p, 3)
		if err != nil {
			t.Fatal(err)
		}
		nulls := 0
		for _, b := range bl {
			if plan.IsE3Null(b) {
				nulls++
			}
		}
		if (c.blocks > 0 && len(bl) != c.blocks) || nulls != p.Reps {
			t.Fatalf("%s: %d blocks, %d null blocks", c.name, len(bl), nulls)
		}
	}
}
