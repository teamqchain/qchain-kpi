package plan

import (
	"qchain-kpi/internal/algs"
)

// E9: memory. Every operation of every algorithm, measured three ways, one
// block (process) each:
//   - "c":     the C library called directly on a thread with a painted stack
//     (stack high-water mark and the algorithm's own C heap);
//   - "reuse": E1's arm (the library's long-lived object through liboqs-go or
//     cgo): C heap, Go heap;
//   - "prod":  E2's prod arm (QChain's production pattern): C heap, Go heap.
//
// The operation field names both: "sign-c".

// E9Arms are E9's measurement routes, in plan order.
var E9Arms = []string{"c", "reuse", "prod"}

// e9Ops returns an E9 item's operation fields for one algorithm.
func e9Ops(it Item, a algs.Alg) ([]string, error) {
	base := it.Ops
	if len(base) == 0 {
		base = a.Ops()
	}
	if err := checkOps(a, base); err != nil {
		return nil, err
	}
	var ops []string
	for _, op := range base {
		for _, arm := range E9Arms {
			ops = append(ops, op+"-"+arm)
		}
	}
	return ops, nil
}
