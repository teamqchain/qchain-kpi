package main

import (
	"math"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E12 (concurrency): estimate.

// e12PrepNS is a worker's input preparation when the trial has no E11 probe
// of the stage (it signs and seals 32 credentials: about a second for the
// slowest E12 suite).
const e12PrepNS = 1e9

// e12Block estimates an E12 block: fixed warm-up and window, plus each
// worker's own preparation (E11's measured preparation of the stage where
// the trial has it). Raw rows: about one per second plus the histogram.
func (x *est) e12Block(bl plan.Block) (secs, rows float64) {
	stageOp, w, err := proto.ParseE12Op(bl.Op)
	if err != nil {
		return math.NaN(), 0
	}
	bud, err := proto.E12BudgetFor(bl.Profile)
	if err != nil {
		return math.NaN(), 0
	}
	prep := e12PrepNS
	if v, ok := x.f.prep[bl.Alg+"/"+stageOp]; ok {
		prep = v
	}
	secs = x.f.overheadNS/1e9 + float64(w)*prep/1e9 + (bud.Warmup + bud.Window).Seconds()
	return secs, bud.Window.Seconds() + float64(w) + 300
}
