package plan

import (
	"fmt"

	"qchain-kpi/internal/algs"
)

// e13Ops returns E13's single size block for an algorithm: Fabric
// transactions for a signature scheme, IPFS envelopes for a KEM.
func e13Ops(it Item, a algs.Alg) ([]string, error) {
	op := "ledger"
	if a.Track == algs.KEM {
		op = "ipfs"
	}
	if len(it.Ops) > 0 && (len(it.Ops) != 1 || it.Ops[0] != op) {
		return nil, fmt.Errorf("E13's operation for %s is %q", a.ID, op)
	}
	return []string{op}, nil
}
