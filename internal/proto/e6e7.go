package proto

import (
	"fmt"
	"strconv"
	"strings"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
)

// E6: signing and verification time against message size. One block per
// (signature scheme, sign | verify, size) measures the library's reuse arm
// (E1's path) over messages of that size. The operation field names both:
// "sign-m1024".
//
// E7: rejection time. One block per algorithm interleaves a valid call with
// invalid ones (coding plan §7): for a signature scheme, verify with a valid
// signature, with the same signature with one bit flipped ("badsig"), and
// with a valid signature over a different message of the same length
// ("wrongmsg"); for a KEM, decaps of a valid ciphertext and of the same
// ciphertext with one bit flipped ("tampered").

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

// E7 arm names.
const (
	ArmValid    = "valid"    // the operation on valid input (the baseline)
	ArmBadSig   = "badsig"   // verify: the signature with one bit flipped
	ArmWrongMsg = "wrongmsg" // verify: a valid signature over a different message of the same length
	ArmTampered = "tampered" // decaps: the ciphertext with one bit flipped
)

// E7Op is E7's operation for an algorithm: verify or decaps.
func E7Op(a algs.Alg) string {
	if a.Track == algs.Sig {
		return "verify"
	}
	return "decaps"
}

// E7Arms are an E7 block's arms, in build order.
func E7Arms(a algs.Alg) []string {
	if a.Track == algs.Sig {
		return []string{ArmValid, ArmBadSig, ArmWrongMsg}
	}
	return []string{ArmValid, ArmTampered}
}

// E7 outcome codes, recorded per sample in the raw file's bytes column.
const (
	E7Rejected      = 0 // verify returned false, or decaps returned a different secret (implicit rejection)
	E7RejectedError = 1 // the library returned an error (malformed input)
	E7Accepted      = 2 // verify returned true, or decaps returned the encapsulated secret
)
