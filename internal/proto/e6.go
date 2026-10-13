package proto

import (
	"fmt"
	"strconv"
	"strings"

	"qchain-kpi/internal/plan"
)

// E6: signing and verification time against message size. One block per
// (signature scheme, sign | verify, size) measures the library's reuse arm
// (E1's path) over messages of that size. The operation field names both:
// "sign-m1024".
//

// E6Sizes are the message sizes in bytes: 64 B, 1 KiB, 10 KiB, 100 KiB, 1 MiB
// (one definition, in internal/plan, which expands the cells).
var E6Sizes = plan.E6Sizes

// E6Op is the operation field of an E6 block: "sign-m1024".
func E6Op(op string, size int) string { return fmt.Sprintf("%s-m%d", op, size) }

// ParseE6Op splits an E6 operation field into the operation and the size.
func ParseE6Op(s string) (op string, size int, err error) {
	op, m, ok := strings.Cut(s, "-m")
	if !ok || (op != "sign" && op != "verify") {
		return "", 0, fmt.Errorf("E6 operation %q is not sign-m<bytes> or verify-m<bytes>", s)
	}
	size, err = strconv.Atoi(m)
	if err != nil {
		return "", 0, fmt.Errorf("E6 operation %q: %w", s, err)
	}
	for _, z := range E6Sizes {
		if z == size {
			return op, size, nil
		}
	}
	return "", 0, fmt.Errorf("E6 operation %q: size %d is not one of %v", s, size, E6Sizes)
}

// E6MaxPoolBytes caps the messages a block holds in memory: a pool keeps
// min(P, max(8, 64 MiB / size)) messages, so 1 MiB messages cycle 64.
const E6MaxPoolBytes = 64 << 20

// E6PoolSize is the pool size of one message size given the algorithm's P.
func E6PoolSize(p, size int) int {
	n := E6MaxPoolBytes / size
	if n < 8 {
		n = 8
	}
	return min(p, n)
}
