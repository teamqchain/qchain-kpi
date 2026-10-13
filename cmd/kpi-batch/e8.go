package main

import (
	"math"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E8 (sizes): estimate.

// e8Block estimates an E8 block: target 1,000 keys or 10,000 outputs, each
// checked (verify / decaps); at least 100, stopping after 60 s.
func (x *est) e8Block(bl plan.Block, a algs.Alg, med, check float64) (secs, rows float64) {
	cost := med + check
	target := 10000.0
	if bl.Op == "keygen" {
		target = 1000
	}
	n := target
	if cost > 0 {
		n = math.Min(target, math.Max(100, 60e9/cost))
	}
	bs := x.f.overheadNS/1e9 + n*cost/1e9
	if bl.Op == "keygen" {
		sub := math.Max(3, n/50)
		if a.Track == algs.Sig {
			bs += sub * (x.get(bl.Alg+"/sign") + x.get(bl.Alg+"/verify")) / 1e9
		} else {
			bs += sub * (x.get(bl.Alg+"/encaps") + x.get(bl.Alg+"/decaps")) / 1e9
		}
	}
	return bs, n * float64(len(proto.E8Arms(a, bl.Op)))
}
