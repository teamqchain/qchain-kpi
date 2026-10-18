package main

import (
	"path/filepath"
	"testing"

	"qchain-kpi/internal/plan"
)

// TestE4RefusesTheBaselineBuild: trial and main E4 runs need a non-default
// liboqs build; smoke runs and other experiments are not affected.
func TestE4RefusesTheBaselineBuild(t *testing.T) {
	p, err := plan.Load(filepath.Join("..", "..", "plans", "e4.json"))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := plan.Expand(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 90*5 {
		t.Fatalf("%d E4 blocks, want 450", len(blocks))
	}
	for _, c := range []struct {
		variant string
		ok      bool
	}{{"b1-prod", false}, {"", false}, {"b2-portable", true}, {"b3-native", true}, {"b4-noossl", true}} {
		err := e4VariantCheck(p, blocks, map[string]string{"variant": c.variant})
		if (err == nil) != c.ok {
			t.Fatalf("variant %q: %v", c.variant, err)
		}
	}
	smoke, err := plan.Load(filepath.Join("..", "..", "plans", "smoke-e4.json"))
	if err != nil {
		t.Fatal(err)
	}
	sb, _ := plan.Expand(smoke, 1)
	if err := e4VariantCheck(smoke, sb, map[string]string{"variant": "b1-prod"}); err != nil {
		t.Fatalf("smoke refused: %v", err)
	}
	e1, _ := plan.Load(filepath.Join("..", "..", "plans", "e1.json"))
	eb, _ := plan.Expand(e1, 1)
	if err := e4VariantCheck(e1, eb, map[string]string{"variant": "b1-prod"}); err != nil {
		t.Fatalf("E1 refused: %v", err)
	}
}
