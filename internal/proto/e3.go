package proto

import (
	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
)

// E3: integration-pattern cost. A block measures one operation in several
// layers, interleaved in one process, each with the full approved budget:
//   - l0:  the C library called directly, timed inside C (internal/l0): the
//     algorithm's own time;
//   - l0g: the same C call timed from Go around the cgo call, so
//     l0g - l0 is the Go-to-C boundary;
//   - l1:  E1's reuse arm (liboqs-go, or this harness's OpenSSL cgo calls),
//     so l1 - l0g is the Go binding's own work;
//   - l2:  classical algorithms only: Go's standard library (internal/gostd).
//
// The null block (plan.E3NullAlg / plan.E3NullOp) measures a cgo call to a C
// function that does nothing (batched) and two C clock readings with
// nothing between them (the C timer's floor).
const (
	ArmL0      = "l0"
	ArmL0g     = "l0g"
	ArmL1      = "l1"
	ArmL2      = "l2"
	ArmCgoNull = "cgo-null"
	ArmCEmpty  = "c-empty"
)

// E3NullBatch is the number of null cgo calls per sample (each takes tens of
// nanoseconds; the timer's resolution is 41 ns on the Mac).
const E3NullBatch = 1000

// E3Arms returns the layers of an E3 block, in build order.
func E3Arms(a algs.Alg) []string {
	if a.Classical() {
		return []string{ArmL0, ArmL0g, ArmL1, ArmL2}
	}
	return []string{ArmL0, ArmL0g, ArmL1}
}

// E3NullArms are the arms of the null block.
var E3NullArms = []string{ArmCgoNull, ArmCEmpty}

// IsE3Null reports whether a block is E3's null-call block.
func IsE3Null(exp, alg, op string) bool {
	return plan.IsE3Null(plan.Block{Exp: exp, Alg: alg, Op: op})
}
