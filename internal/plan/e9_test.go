package plan

import (
	"testing"
)

// TestE9Expansion: every operation of every algorithm, three routes each.
func TestE9Expansion(t *testing.T) {
	p, err := Load("../../plans/mac/e9.json")
	if err != nil {
		t.Fatal(err)
	}
	bl, err := Expand(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	routes := map[string]int{}
	for _, b := range bl {
		if b.Exp != "E9" {
			t.Fatalf("block %+v", b)
		}
		for _, r := range E9Arms {
			if len(b.Op) > len(r) && b.Op[len(b.Op)-len(r)-1:] == "-"+r {
				routes[r]++
			}
		}
	}
	// 22 signature schemes and 8 KEMs, three operations each, five repetitions.
	if len(bl) != 90*3*5 || routes["c"] != 450 || routes["reuse"] != 450 || routes["prod"] != 450 {
		t.Fatalf("%d blocks %v", len(bl), routes)
	}
	s, err := Load("../../plans/mac/smoke-e9.json")
	if err != nil {
		t.Fatal(err)
	}
	if sb, err := Expand(s, 1); err != nil || len(sb) != 6*3*3 {
		t.Fatalf("smoke-e9: %d blocks (%v)", len(sb), err)
	}
	bad := &Plan{Name: "x", Kind: KindMain, Reps: 1, Items: []Item{{Exp: "E9", Algs: []string{"ml-kem-768"}, Ops: []string{"sign"}, Profile: ProfileApproved}}}
	if _, err := Expand(bad, 1); err == nil {
		t.Fatal("E9 must refuse an operation the algorithm does not have")
	}
}
