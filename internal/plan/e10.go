package plan

import (
	"fmt"

	"qchain-kpi/internal/algs"
)

// e10Check refuses an E10 item for anything but ML-DSA.
func e10Check(it Item) error {
	for _, a := range mustExpand(it.Algs) {
		if a.Family != "ML-DSA" {
			return fmt.Errorf("E10 is defined for ML-DSA only, got %s", a.ID)
		}
	}
	return nil
}

// e10Ops is E10's single operation: sign.
func e10Ops(a algs.Alg) ([]string, error) {
	ops := []string{"sign"}
	return ops, checkOps(a, ops)
}
