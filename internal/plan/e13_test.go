package plan

import (
	"testing"
)

// TestE13Expansion: one ledger block per signature scheme, one IPFS block per KEM.
func TestE13Expansion(t *testing.T) {
	p, err := Load("../../plans/mac/e13.json")
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
		if b.Exp != "E13" || b.Rep != 1 {
			t.Fatalf("block %+v", b)
		}
	}
	if len(bl) != 30 || ops["ledger"] != 22 || ops["ipfs"] != 8 {
		t.Fatalf("%d blocks %v", len(bl), ops)
	}
}
