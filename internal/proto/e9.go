package proto

import (
	"fmt"
	"strings"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
)

// E9: memory (internal/plan describes the three routes). Nothing is timed. A
// block makes up to E9Budget calls of one operation and records, per call,
// what the route can see:
//   - c:            stack high-water mark and the call's C heap;
//   - reuse, prod:  the call's C heap and Go heap;
//
// and, per block, the growth of the process's peak resident memory over the
// calls. The "arms" of an E9 block's summary are these quantities, in their
// own units (bytes or counts), not nanoseconds.

// ArmC is E9's direct-C route.
const ArmC = "c"

// E9Arms are E9's routes (one definition, in internal/plan).
var E9Arms = plan.E9Arms

// E9 quantities.
const (
	QStack        = "stack"         // bytes: deepest stack byte written, above the empty-job baseline
	QHeapPeak     = "heap-peak"     // bytes: highest live C heap during the call, above its start
	QHeapBytes    = "heap-bytes"    // bytes: C heap allocated during the call (usable sizes)
	QHeapAllocs   = "heap-allocs"   // count: C allocations during the call
	QHeapUnfreed  = "heap-unfreed"  // count: C allocations minus frees during the call (blocks left allocated)
	QHeapRetained = "heap-retained" // bytes: C heap still held after the call (leaked or cached)
	QGoBytes      = "go-bytes"      // bytes: Go heap allocated during the call
	QGoAllocs     = "go-allocs"     // count: Go heap objects allocated during the call
	QRSSGrowth    = "rss-growth"    // bytes, per block: peak resident set over the calls minus the resident set before them
	QRSSPeak      = "rss-peak"      // bytes, per block: the process's peak resident set over the calls
)

// E9Quantities returns the summary quantities of an E9 route, in order.
func E9Quantities(arm string) []string {
	if arm == ArmC {
		return []string{QStack, QHeapPeak, QHeapBytes, QHeapAllocs, QHeapUnfreed, QHeapRetained, QRSSGrowth, QRSSPeak}
	}
	return []string{QHeapPeak, QHeapBytes, QHeapAllocs, QHeapUnfreed, QHeapRetained, QGoBytes, QGoAllocs, QRSSGrowth, QRSSPeak}
}

// E9PerBlock reports whether a quantity has one value per block (not per call).
func E9PerBlock(q string) bool { return q == QRSSGrowth || q == QRSSPeak }

// E9Op is the operation field of an E9 block: "sign-c".
func E9Op(op, arm string) string { return op + "-" + arm }

// ParseE9Op splits an E9 operation field into the operation and the route.
func ParseE9Op(s string) (op, arm string, err error) {
	op, arm, ok := strings.Cut(s, "-")
	if ok {
		for _, r := range E9Arms {
			if r == arm {
				return op, arm, nil
			}
		}
	}
	return "", "", fmt.Errorf("E9 operation %q is not <op>-<route> with route one of %v", s, E9Arms)
}

// E9Budget is E9's sampling rule: make Target calls, but stop earlier once
// TimeBox has passed and at least Min calls are done (slow SLH-DSA signing
// and RSA key generation). The first call is reported on its own; the
// summaries are over the calls after it.
type E9Budget struct {
	Target  int
	Min     int
	TimeBox time.Duration
}

// E9BudgetFor returns the budget of a profile.
func E9BudgetFor(profile string) (E9Budget, error) {
	switch profile {
	case plan.ProfileApproved:
		return E9Budget{Target: 100, Min: 10, TimeBox: 20 * time.Second}, nil
	case plan.ProfileSmoke:
		return E9Budget{Target: 5, Min: 2, TimeBox: 2 * time.Second}, nil
	}
	return E9Budget{}, fmt.Errorf("no E9 budget for profile %q", profile)
}

// E9PoolSize is the input pool of an E9 run's (algorithm, repetition): calls
// cycle through it, and memory does not depend on which message or
// ciphertext is used, so a small pool saves preparation time (16 for the
// slow SLH-DSA "s" sets, whose preparation signs every entry; 32 otherwise).
func E9PoolSize(a algs.Alg) int {
	if a.PoolSize() < 256 {
		return 16
	}
	return 32
}

// E9MallocLib is the heap counter's path in the image (csrc/kpi_malloc.c);
// kpi-batch starts E9 blocks with it in LD_PRELOAD.
const E9MallocLib = "/opt/kpi/lib/libkpi_malloc.so"
