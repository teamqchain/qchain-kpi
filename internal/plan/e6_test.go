package plan

import (
	"testing"
)

// TestE6Expansion: E6 is sign and verify at five sizes for the 22 signature
// schemes, and refuses KEMs.
func TestE6Expansion(t *testing.T) {
	count := func(file string) (int, map[string]int) {
		p, err := Load("../../plans/" + file)
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
		return len(bl), ops
	}
	if n, ops := count("e6.json"); n != 22*2*5*5 || ops["sign-m1048576"] != 22*5 || ops["verify-m64"] != 22*5 {
		t.Fatalf("e6: %d blocks %v", n, ops)
	}
	if n, _ := count("trial-e6.json"); n != 22*2*5 {
		t.Fatalf("trial-e6: %d blocks", n)
	}
	p := &Plan{Name: "x", Kind: KindMain, Reps: 1, Items: []Item{{Exp: "E6", Algs: []string{"ml-kem-768"}}}}
	if _, err := Expand(p, 1); err == nil {
		t.Fatal("E6 for a KEM must be refused")
	}
}
