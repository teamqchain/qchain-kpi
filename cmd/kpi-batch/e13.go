package main

import (
	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E13 (ledger and IPFS bytes): estimate.

// e13Block estimates an E13 block (sizes only): a ledger block issues one
// credential per attribute count (signed and verified) and samples 1,000
// signatures of a variable-length scheme; an IPFS block seals and opens two
// envelopes per setting with ML-DSA-44 signing.
func (x *est) e13Block(bl plan.Block, a algs.Alg) (secs, rows float64) {
	bs := x.f.overheadNS / 1e9
	if a.Track == algs.Sig {
		n := float64(len(cells.Ns))
		if proto.E13SigVariable(a) {
			n += 1000
		}
		bs += (2*x.get(bl.Alg+"/keygen") + n*(x.get(bl.Alg+"/sign")+x.get(bl.Alg+"/verify"))) / 1e9
		return bs, 7 * 7 * (5*float64(len(cells.Ns)) + 9)
	}
	per := x.get("ml-dsa-44/sign") + x.get(bl.Alg+"/encaps") + x.get(bl.Alg+"/decaps")
	bs += (x.get(bl.Alg+"/keygen") + 2*15*per) / 1e9
	return bs, 4 * 15
}
