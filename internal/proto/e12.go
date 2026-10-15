package proto

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"qchain-kpi/internal/plan"
)

// E12: concurrency (internal/plan describes the grid). A block runs W
// workers in one process (GOMAXPROCS = W, Go's default GC) through
// internal/conc. Its summary "arms" are quantities:
const (
	QLatency    = "latency"    // ns per call completed in the window (from the histogram; first_ns unused)
	QThroughput = "throughput" // calls completed per second of the window (statistics over the seconds)
)

// E12Arms are the summary quantities of an E12 block, in order (plus the
// per-block resident memory of E9: rss-growth, rss-peak).
var E12Arms = []string{QLatency, QThroughput, QRSSGrowth, QRSSPeak}

// ParseE12Op splits an E12 operation field into the stage's E11 operation
// and the worker count: "verification-n10-k5-v16-w4" -> ("verification-n10-k5-v16", 4).
func ParseE12Op(op string) (stageOp string, workers int, err error) {
	i := strings.LastIndex(op, "-w")
	if i <= 0 {
		return "", 0, fmt.Errorf("E12 operation %q has no -w<workers> suffix", op)
	}
	w, err := strconv.Atoi(op[i+2:])
	if err != nil || w < 1 {
		return "", 0, fmt.Errorf("E12 operation %q: bad worker count", op)
	}
	if plan.E12Op(op[:i], w) != op {
		return "", 0, fmt.Errorf("E12 operation %q is not canonical", op)
	}
	return op[:i], w, nil
}

// E12Budget is the time plan of an E12 block.
type E12Budget struct {
	Warmup     time.Duration
	Window     time.Duration
	CheckEvery int // window calls checked: every n-th of each worker (warm-up calls: all)
}

// E12BudgetFor returns the budget of a profile.
func E12BudgetFor(profile string) (E12Budget, error) {
	switch profile {
	case plan.ProfileApproved:
		return E12Budget{Warmup: 5 * time.Second, Window: 30 * time.Second, CheckEvery: 64}, nil
	case plan.ProfileSmoke:
		return E12Budget{Warmup: time.Second, Window: 2 * time.Second, CheckEvery: 8}, nil
	}
	return E12Budget{}, fmt.Errorf("no E12 budget for profile %q", profile)
}
