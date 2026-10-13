package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func writePlan(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExpandRepMajorDeterministic(t *testing.T) {
	p, err := Load(writePlan(t, `{"name":"t","kind":"trial","description":"","reps":3,"canary_every_s":300,
	 "items":[{"exp":"probe","algs":["ml-dsa-44"]},{"exp":"E1","algs":["ml-dsa-44","ml-kem-768"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a, err := Expand(p, 7)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Expand(p, 7)
	// 3 probe blocks (rep 1 only) + 6 E1 cells x 3 reps.
	if len(a) != 3+18 {
		t.Fatalf("%d blocks", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("expansion is not deterministic")
		}
	}
	// Probes first, and every rep-1 block precedes every rep-2 block.
	for i := 0; i < 3; i++ {
		if a[i].Exp != "probe" {
			t.Fatalf("block %d is %s, want probe first", i, a[i].Exp)
		}
	}
	last1 := -1
	first2 := len(a)
	for i, x := range a {
		if x.Rep == 1 && i > last1 {
			last1 = i
		}
		if x.Rep == 2 && i < first2 {
			first2 = i
		}
	}
	if last1 > first2 {
		t.Fatal("not rep-major")
	}
	ids := map[string]bool{}
	for _, x := range a {
		if ids[x.Key()] {
			t.Fatalf("duplicate block %s", x.Key())
		}
		ids[x.Key()] = true
	}
}

func TestDeriveSeedStable(t *testing.T) {
	if DeriveSeed(1, "x") != DeriveSeed(1, "x") || DeriveSeed(1, "x") == DeriveSeed(2, "x") {
		t.Fatal("seed derivation")
	}
}

func TestValidation(t *testing.T) {
	if _, err := Load(writePlan(t, `{"name":"t","kind":"bogus","description":"","reps":1,"items":[]}`)); err == nil {
		t.Fatal("bad kind accepted")
	}
	if _, err := Load(writePlan(t, `{"name":"t","kind":"main","description":"","reps":1,"items":[{"exp":"E10","algs":["falcon-512"]}]}`)); err == nil {
		t.Fatal("E10 on Falcon accepted")
	}
	if _, err := Load(writePlan(t, `{"name":"t","kind":"main","description":"","reps":1,"extra":1,"items":[]}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
}

// TestE13Expansion: one ledger block per signature scheme, one IPFS block per KEM.
func TestE13Expansion(t *testing.T) {
	p, err := Load("../../plans/e13.json")
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
