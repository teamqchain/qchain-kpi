package plan

import (
	"fmt"

	"qchain-kpi/internal/algs"
)

// e8Ops returns an E8 item's operations for one algorithm: key generation
// and sign (signatures) or encaps (KEMs), the operations that produce new
// objects.
func e8Ops(it Item, a algs.Alg) ([]string, error) {
	ops := it.Ops
	if len(ops) == 0 {
		ops = a.SizeOps()
	}
	for _, op := range ops {
		if op != "keygen" && op != "sign" && op != "encaps" {
			return nil, fmt.Errorf("E8 measures keygen, sign and encaps only, not %q", op)
		}
	}
	return ops, checkOps(a, ops)
}
