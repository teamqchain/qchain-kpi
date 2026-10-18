package plan

import "qchain-kpi/internal/algs"

// E5: platform and runtime comparison. E1's cells (every operation of every
// algorithm, the library's reuse arm) run natively on the host operating
// system, outside the container (on the Mac: macOS with liboqs and OpenSSL
// built by Apple clang; scripts/native-mac.sh). Compared with E1 in the
// container on the same machine.

// e5Ops returns an E5 item's operations for one algorithm (default: all).
func e5Ops(it Item, a algs.Alg) ([]string, error) {
	ops := it.Ops
	if len(ops) == 0 {
		ops = a.Ops()
	}
	return ops, checkOps(a, ops)
}
