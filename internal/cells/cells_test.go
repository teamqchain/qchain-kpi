package cells

import "testing"

func TestGrid(t *testing.T) {
	g := Grid()
	count := map[string]int{}
	for _, c := range g {
		if err := c.Validate(); err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
		back, err := ParseCell(c.Alg(), c.Op())
		if err != nil || back != c {
			t.Fatalf("round trip %+v -> %s / %s -> %+v (%v)", c, c.Alg(), c.Op(), back, err)
		}
		count[c.Stage]++
	}
	want := map[string]int{Activation: 176, Issuance: 250, Retrieval: 50, Presentation: 108, Verification: 394, Chained: 13}
	for s, n := range want {
		if count[s] != n {
			t.Fatalf("%s: %d cells, want %d (all: %v)", s, count[s], n, count)
		}
	}
	if len(g) != 991 {
		t.Fatalf("%d cells", len(g))
	}
	if ks := Ks(1); len(ks) != 1 || ks[0] != 1 {
		t.Fatalf("Ks(1) = %v", ks)
	}
	if ks := Ks(64); len(ks) != 3 || ks[1] != 32 {
		t.Fatalf("Ks(64) = %v", ks)
	}
}

func TestParseCellRejects(t *testing.T) {
	for _, x := range [][2]string{
		{"ml-dsa-44+ml-kem-768", "issuance-n10"},            // V missing
		{"ml-kem-768", "presentation-k5-v16"},               // presentation needs a signature
		{"ml-dsa-44+ml-kem-768", "verification-n5-k10-v16"}, // k > N
		{"ml-dsa-44+ml-kem-768", "teleport"},
		{"ml-kem-768+ml-dsa-44", "activation"}, // order swapped
	} {
		if _, err := ParseCell(x[0], x[1]); err == nil {
			t.Fatalf("%v accepted", x)
		}
	}
}
