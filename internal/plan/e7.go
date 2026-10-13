package plan

import (
	"fmt"

	"qchain-kpi/internal/algs"
)

// e7Ops returns E7's single operation for an algorithm: one block per
// algorithm, verify (signatures) or decaps (KEMs).
func e7Ops(it Item, a algs.Alg) ([]string, error) {
	op := "verify"
	if a.Track == algs.KEM {
		op = "decaps"
	}
	if len(it.Ops) > 0 && (len(it.Ops) != 1 || it.Ops[0] != op) {
		return nil, fmt.Errorf("E7's operation for %s is %q", a.ID, op)
	}
	return []string{op}, nil
}
