package main

import (
	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
)

// E4: build sensitivity. The arm is E1's reuse arm; only the liboqs build in
// the image differs (recorded in the run's build-info and manifest).
func e4Arms(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	return e1Arms(a, op, p, seed)
}
