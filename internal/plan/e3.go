package plan

import "qchain-kpi/internal/algs"

// E3: integration-pattern cost. Every operation of every algorithm, one
// block per (algorithm, operation) with the layers interleaved (internal/proto
// names them), plus one null-call block per repetition that measures the bare
// cost of a Go-to-C call and the C timer's floor. The null block has no
// algorithm: its cell is (E3NullAlg, E3NullOp).
const (
	E3NullAlg = "none"
	E3NullOp  = "null"
)

// IsE3Null reports whether a block is E3's null-call block.
func IsE3Null(b Block) bool { return b.Exp == "E3" && b.Alg == E3NullAlg && b.Op == E3NullOp }

// e3Ops returns an E3 item's operations for one algorithm (default: all).
func e3Ops(it Item, a algs.Alg) ([]string, error) {
	ops := it.Ops
	if len(ops) == 0 {
		ops = a.Ops()
	}
	return ops, checkOps(a, ops)
}
