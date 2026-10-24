package plan

import "qchain-kpi/internal/algs"

// E4: build sensitivity. E1's cells (every operation of every algorithm,
// the library's reuse arm) measured on an image whose liboqs is built
// differently (b2-portable, b3-native, b4-noossl; the variant is the
// image's, chosen with the launcher's --variant). The classical cells use OpenSSL,
// which the liboqs build does not touch: they are the controls.

// e4Ops returns an E4 item's operations for one algorithm (default: all).
func e4Ops(it Item, a algs.Alg) ([]string, error) {
	ops := it.Ops
	if len(ops) == 0 {
		ops = a.Ops()
	}
	return ops, checkOps(a, ops)
}
