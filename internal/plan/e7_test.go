package plan

import (
	"testing"
)

// TestE7Expansion: E7 is one block per algorithm (verify or decaps).
func TestE7Expansion(t *testing.T) {
	for file, want := range map[string]int{"e7.json": 30 * 5, "trial-e7.json": 30} {
		p, err := Load("../../plans/mac/" + file)
		if err != nil {
			t.Fatal(err)
		}
		bl, err := Expand(p, 1)
		if err != nil {
			t.Fatal(err)
		}
		ops := map[string]int{}
		for _, b := range bl {
			ops[b.Op]++
		}
		if len(bl) != want || ops["verify"]*8 != ops["decaps"]*22 {
			t.Fatalf("%s: %d blocks %v", file, len(bl), ops)
		}
	}
}
