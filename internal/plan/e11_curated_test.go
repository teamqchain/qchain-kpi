package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qchain-kpi/internal/cells"
)

// The curated selection keeps every cell of the 16 curated suites and drops
// only cells that exist for the all-pairs sweep alone.
func TestE11CuratedCells(t *testing.T) {
	all := e11ItemCells(Item{Exp: "E11", Algs: []string{"all"}})
	cur := e11ItemCells(Item{Exp: "E11", Algs: []string{"all"}, Suites: SuitesCurated})
	if len(all) != len(cells.Grid()) || len(all) != 991 || E11CellCount("") != 991 {
		t.Fatalf("whole grid: %d cells", len(all))
	}
	// 16 activations; 90 issuances (16 suites x 5 N, plus the production
	// suite's 256- and 4096-byte values x 5 N); 50 retrievals (every KEM is in
	// a curated suite); 98 presentations (12 signatures x 7 k, plus 2 x 7);
	// 234 verifications (16 suites x 13 (N, k), plus 2 x 13); 13 chained runs.
	want := map[string]int{cells.Activation: 16, cells.Issuance: 90, cells.Retrieval: 50,
		cells.Presentation: 98, cells.Verification: 234, cells.Chained: 13}
	got := map[string]int{}
	for _, c := range cur {
		got[strings.SplitN(c[1], "-", 2)[0]]++
	}
	for st, n := range want {
		if got[st] != n {
			t.Errorf("%s: %d cells, want %d", st, got[st], n)
		}
	}
	if len(cur) != 501 || E11CellCount(SuitesCurated) != 501 {
		t.Fatalf("curated: %d cells, want 501", len(cur))
	}
	// A subset of the whole grid, in the grid's order.
	j := 0
	for _, c := range all {
		if j < len(cur) && c == cur[j] {
			j++
		}
	}
	if j != len(cur) {
		t.Fatal("the curated cells are not an ordered subset of the grid")
	}
	// Every dropped cell is a pairs-sweep cell (N = 10, k = 5, 16-byte values)
	// of a pair, or a signature, that no curated suite has.
	kept := map[[2]string]bool{}
	for _, c := range cur {
		kept[c] = true
	}
	for _, c := range cells.Grid() {
		if kept[[2]string{c.Alg(), c.Op()}] {
			continue
		}
		if (c.V != 0 && c.V != cells.DefaultV) || (c.N != 0 && c.N != cells.PairN) || (c.K != 0 && c.K != cells.PairK) ||
			c.Stage == cells.Retrieval || c.Stage == cells.Chained || curatedCell(c) {
			t.Fatalf("dropped a cell that is not a pairs-only cell: %+v", c)
		}
	}
	// probe-E11 selects the same cells.
	if p := e11ItemCells(Item{Exp: "probe-E11", Algs: []string{"all"}, Suites: SuitesCurated}); len(p) != 501 {
		t.Fatalf("probe-E11 curated: %d cells", len(p))
	}
}

func TestE11SuitesChecked(t *testing.T) {
	dir := t.TempDir()
	write := func(name, items string) string {
		f := filepath.Join(dir, name+".json")
		os.WriteFile(f, []byte(`{"name":"`+name+`","kind":"main","description":"","reps":1,"items":[`+items+`]}`), 0o644)
		return f
	}
	if _, err := Load(write("ok", `{"exp":"E11","algs":["all"],"suites":"curated"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(write("e1", `{"exp":"E1","algs":["all"],"suites":"curated"}`)); err == nil {
		t.Fatal("suites on an E1 item were accepted")
	}
	if _, err := Load(write("odd", `{"exp":"E11","algs":["all"],"suites":"pairs"}`)); err == nil {
		t.Fatal("an unknown suites value was accepted")
	}
}
