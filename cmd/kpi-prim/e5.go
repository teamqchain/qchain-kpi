package main

import (
	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
)

// E5: platform and runtime comparison. The arm is E1's reuse arm; the block
// runs natively on the host (no container, no CPU pinning; on macOS the
// thread asks for user-interactive QoS, recorded in the raw header).
func e5Arms(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	return e1Arms(a, op, p, seed)
}
