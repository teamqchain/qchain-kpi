package main

import (
	"math"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E9 (memory): estimate.

// e9CallNS is E9's fixed cost per call besides the operation: painting and
// scanning the 8 MiB stack (c route) or reading the Go heap statistics twice
// (reuse and prod routes).
const e9CallNS = 1e6

// e9Block estimates an E9 block: target 100 calls, each checked (verify /
// decaps), at least 10, stopping after 20 s; the prod route's operation takes
// E2's prod median where the trial has one.
func (x *est) e9Block(bl plan.Block, a algs.Alg) (secs, rows float64) {
	op, route, err := proto.ParseE9Op(bl.Op)
	if err != nil {
		return math.NaN(), 0
	}
	bud, _ := proto.E9BudgetFor(bl.Profile)
	k := bl.Alg + "/" + op
	med := x.get(k)
	if v, ok := x.f.med[k+"/"+proto.ArmProd]; ok && route == proto.ArmProd {
		med = v
	}
	check := 0.0
	switch op {
	case "sign":
		check = x.get(bl.Alg + "/verify")
	case "encaps":
		check = x.get(bl.Alg + "/decaps")
	}
	cost := med + check + e9CallNS
	n := math.Min(float64(bud.Target), math.Max(float64(bud.Min), bud.TimeBox.Seconds()*1e9/cost))
	bs := x.f.overheadNS/1e9 + n*cost/1e9
	if op == "keygen" {
		sub := math.Max(3, n/50)
		if a.Track == algs.Sig {
			bs += sub * (x.get(bl.Alg+"/sign") + x.get(bl.Alg+"/verify")) / 1e9
		} else {
			bs += sub * (x.get(bl.Alg+"/encaps") + x.get(bl.Alg+"/decaps")) / 1e9
		}
	}
	return bs, n
}

// e9Prep estimates an E9 input set (its pool is proto.E9PoolSize).
func (x *est) e9Prep(a algs.Alg) float64 {
	pool := float64(proto.E9PoolSize(a))
	if a.Track == algs.Sig {
		return x.get(a.ID+"/keygen") + pool*(x.get(a.ID+"/sign")+x.get(a.ID+"/verify"))
	}
	return x.get(a.ID+"/keygen") + pool*(x.get(a.ID+"/encaps")+x.get(a.ID+"/decaps"))
}
