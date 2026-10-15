package plan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"qchain-kpi/internal/cells"
)

// E12: concurrency. W workers run one of QChain's backend stages (E11's code)
// back to back for a fixed window; one block per (suite, stage, W). The
// algorithm field is the suite ("sig+kem", as in E11), the operation field
// the stage's E11 operation plus the worker count: "verification-n10-k5-v16-w4".

// E12Suites are the production suite and three others (user, 15 Oct): the
// compact lattice signature, the hash-based one with fast signing, and the
// classical baseline.
var E12Suites = [][2]string{
	{"ml-dsa-44", "ml-kem-768"}, // production
	{"falcon-512", "ml-kem-512"},
	{"slh-dsa-sha2-128f", "ml-kem-512"},
	{"ecdsa-p256", "dhkem-p256"},
}

// E12Stages are the backend stages E12 loads: verification (N = 10, k = 5)
// and issuance (N = 10), 16-byte values.
func E12Stages(sig, kem string) []cells.Cell {
	return []cells.Cell{
		{Stage: cells.Verification, Sig: sig, KEM: kem, N: 10, K: 5, V: cells.DefaultV},
		{Stage: cells.Issuance, Sig: sig, KEM: kem, N: 10, V: cells.DefaultV},
	}
}

// E12Op is the operation field of an E12 block.
func E12Op(stageOp string, workers int) string { return stageOp + "-w" + strconv.Itoa(workers) }

// e12Cells returns an E12 item's cells as (algorithm field, operation field):
// the suites whose signature and KEM are both among the item's algorithms,
// both stages, every worker count.
func e12Cells(it Item) [][2]string {
	sel := map[string]bool{}
	for _, a := range mustExpand(it.Algs) {
		sel[a.ID] = true
	}
	var out [][2]string
	for _, s := range E12Suites {
		if !sel[s[0]] || !sel[s[1]] {
			continue
		}
		for _, c := range E12Stages(s[0], s[1]) {
			for _, w := range it.Workers {
				out = append(out, [2]string{c.Alg(), E12Op(c.Op(), w)})
			}
		}
	}
	return out
}

// CPUCount is the number of CPUs in a docker CPU list ("1", "0-3", "0,2").
func CPUCount(list string) (int, error) {
	n := 0
	for _, part := range strings.Split(list, ",") {
		lo, hi, rng := strings.Cut(part, "-")
		a, err := strconv.Atoi(lo)
		if err != nil || a < 0 {
			return 0, fmt.Errorf("bad CPU list %q", list)
		}
		b := a
		if rng {
			if b, err = strconv.Atoi(hi); err != nil || b < a {
				return 0, fmt.Errorf("bad CPU list %q", list)
			}
		}
		n += b - a + 1
	}
	return n, nil
}

// cpusetCheck: a plan with several CPUs holds E12 items only (every other
// experiment is measured pinned to one CPU), every E12 plan names its CPUs,
// and no E12 item has more workers than CPUs.
func cpusetCheck(p *Plan) error {
	hasE12, other := false, false
	maxW := 0
	for _, it := range p.Items {
		if it.Exp != "E12" {
			other = true
			continue
		}
		hasE12 = true
		ws := append([]int(nil), it.Workers...)
		sort.Ints(ws)
		for k, w := range ws {
			if w < 1 || (k > 0 && w == ws[k-1]) {
				return fmt.Errorf("E12 workers %v must be distinct and at least 1", it.Workers)
			}
		}
		maxW = max(maxW, ws[len(ws)-1])
	}
	if p.CPUSet == "" {
		if hasE12 {
			return fmt.Errorf("an E12 plan names its container CPUs (cpuset)")
		}
		return nil
	}
	n, err := CPUCount(p.CPUSet)
	if err != nil {
		return err
	}
	if other {
		return fmt.Errorf("cpuset %q is for E12 plans only; every other experiment runs pinned to one CPU", p.CPUSet)
	}
	if maxW > n {
		return fmt.Errorf("%d workers but %d CPUs in cpuset %q", maxW, n, p.CPUSet)
	}
	return nil
}
