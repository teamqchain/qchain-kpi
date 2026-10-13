package plan

import (
	"fmt"

	"qchain-kpi/internal/algs"
)

// E6: sign and verify at every message size, signature schemes only.

// E6Sizes are E6's message sizes in bytes (internal/proto names them; the
// plan needs them to expand E6's cells).
var E6Sizes = []int{64, 1024, 10 * 1024, 100 * 1024, 1024 * 1024}

// e6Ops returns an E6 item's operation fields for one scheme: "sign-m64" ...
func e6Ops(it Item, a algs.Alg) ([]string, error) {
	if a.Track != algs.Sig {
		return nil, fmt.Errorf("E6 is defined for signature schemes, not %s", a.ID)
	}
	base := it.Ops
	if len(base) == 0 {
		base = []string{"sign", "verify"}
	}
	var ops []string
	for _, op := range base {
		if op != "sign" && op != "verify" {
			return nil, fmt.Errorf("E6 measures sign and verify, not %q", op)
		}
		for _, size := range E6Sizes {
			ops = append(ops, fmt.Sprintf("%s-m%d", op, size))
		}
	}
	return ops, nil
}
