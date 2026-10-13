package plan

import "qchain-kpi/internal/cells"

// e11Cells returns an E11 item's cells as (algorithm field, operation field):
// E11's grid is fixed (internal/cells); the item's algorithms select the cells
// whose signature and KEM are all among them.
func e11Cells(it Item) [][2]string {
	sel := map[string]bool{}
	for _, a := range mustExpand(it.Algs) {
		sel[a.ID] = true
	}
	var out [][2]string
	for _, c := range cells.Grid() {
		if (c.Sig == "" || sel[c.Sig]) && (c.KEM == "" || sel[c.KEM]) {
			out = append(out, [2]string{c.Alg(), c.Op()})
		}
	}
	return out
}
