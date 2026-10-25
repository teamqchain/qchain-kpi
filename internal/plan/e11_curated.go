package plan

import (
	"fmt"

	"qchain-kpi/internal/cells"
)

// E11 on the 16 curated suites only: the scope on the VM and on Windows
// (decision of 21 Oct, docs/methodology.md section 9). An E11 or probe-E11
// item with "suites": "curated" keeps every cell of the 16 curated suites
// (at every N and k), the production suite's 256- and 4096-byte value cells
// and the chained validation, and drops the cells that exist only for the
// all-pairs sweep (signature x KEM pairs at N = 10, k = 5, on the Mac). An
// item without "suites" is the whole grid, as on the Mac. The cells and the
// code that measures them are E11's own; only the selection is new.

// SuitesCurated is the one value of an item's "suites".
const SuitesCurated = "curated"

func e11SuitesCheck(it Item) error {
	switch {
	case it.Suites == "":
		return nil
	case it.Exp != "E11" && it.Exp != "probe-E11":
		return fmt.Errorf("suites are given for E11 and probe-E11 items only")
	case it.Suites != SuitesCurated:
		return fmt.Errorf("unknown suites %q (the only choice is %q)", it.Suites, SuitesCurated)
	}
	return nil
}

// e11ItemCells: the item's cells (e11Cells), restricted to the curated suites
// when the item asks for them.
func e11ItemCells(it Item) [][2]string {
	all := e11Cells(it)
	if it.Suites != SuitesCurated {
		return all
	}
	keep := map[[2]string]bool{}
	for _, c := range cells.Grid() {
		if curatedCell(c) {
			keep[[2]string{c.Alg(), c.Op()}] = true
		}
	}
	var out [][2]string
	for _, c := range all {
		if keep[c] {
			out = append(out, c)
		}
	}
	return out
}

// curatedCell: a cell belongs to the curated suites when the suite it depends
// on is one of them. A stage that depends on both algorithms needs the pair; a
// presentation (signature only) needs its signature in a curated suite, and a
// retrieval (KEM only) its KEM. The grid gives curated suites every N, k and
// value size, so this keeps exactly their settings; the cells it drops are
// those of the all-pairs sweep that no curated suite has.
func curatedCell(c cells.Cell) bool {
	for _, s := range cells.CuratedSuites {
		switch {
		case c.Sig != "" && c.KEM != "":
			if c.Sig == s[0] && c.KEM == s[1] {
				return true
			}
		case c.Sig != "":
			if c.Sig == s[0] {
				return true
			}
		case c.KEM != "":
			if c.KEM == s[1] {
				return true
			}
		}
	}
	return false
}

// E11CellCount is the number of E11 cells of all algorithms for an item's
// "suites" ("" = the whole grid, or SuitesCurated), for calibration notes.
func E11CellCount(suites string) int {
	return len(e11ItemCells(Item{Exp: "E11", Algs: []string{"all"}, Suites: suites}))
}
