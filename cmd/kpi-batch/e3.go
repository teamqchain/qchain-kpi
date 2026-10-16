package main

import "qchain-kpi/internal/plan"

// E3 (integration-pattern cost): estimate. Its algorithm blocks use the
// generic per-arm estimate (every layer at the operation's E1 median); the
// null block is estimated here.

// e3NullNS are assumed per-call costs of the null block's arms: a cgo call
// to an empty C function, and two C clock readings.
const (
	e3NullCgoNS   = 60
	e3NullEmptyNS = 50
)

// e3NullBlock estimates the null block: 10,000 batched samples of 1,000 null
// cgo calls, and 10,000 samples of the empty C region.
func (x *est) e3NullBlock(bl plan.Block) (secs, rows float64) {
	s1, r1 := armSeconds(e3NullCgoNS, 0, "fast", 1000)
	s2, r2 := armSeconds(e3NullEmptyNS, 0, "fast", 1)
	return x.f.overheadNS/1e9 + s1 + s2, r1 + r2
}
