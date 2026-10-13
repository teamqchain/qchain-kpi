package proto

import (
	"fmt"
	"strings"
	"time"

	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/timing"
)

// E11: lifecycle composites (cells in internal/cells, stages in internal/flow).

// E11Arms returns the arms of an E11 (or probe-E11) block: the whole stage,
// plus the chain-record decoding for verification.
func E11Arms(op string) []string {
	if strings.HasPrefix(op, cells.Verification+"-") {
		return []string{cells.ArmStage, cells.ArmChainDecode}
	}
	return []string{cells.ArmStage}
}

// BudgetE11 returns the sampling configuration of an E11 composite (approved
// 10 Oct): 20 untimed warm-up calls and 2,000 samples, or the slow-operation
// rule (60 s and at least 30 samples) for cells whose probe median is at least
// 30 ms. Garbage collection stays at Go's default (production-like), so no
// collections are forced between samples.
func BudgetE11(profile string, mode timing.Mode, seed uint64) (timing.Config, error) {
	var c timing.Config
	switch {
	case profile == plan.ProfileApproved && mode == timing.Fast:
		c = timing.Config{Mode: timing.Fast, Warmup: 20, Samples: 2000, FirstLabel: "first", Seed: seed}
	case profile == plan.ProfileApproved && mode == timing.Slow:
		c = timing.Config{Mode: timing.Slow, Warmup: 3, TimeBox: 60 * time.Second, MinSamples: 30, FirstLabel: "first", Seed: seed}
	case profile == plan.ProfileSmoke && mode == timing.Fast:
		c = timing.Config{Mode: timing.Fast, Warmup: 2, Samples: 50, FirstLabel: "first", Seed: seed}
	case profile == plan.ProfileSmoke && mode == timing.Slow:
		c = timing.Config{Mode: timing.Slow, Warmup: 1, TimeBox: time.Second, MinSamples: 3, FirstLabel: "first", Seed: seed}
	default:
		return c, fmt.Errorf("no E11 budget for profile %q mode %q", profile, mode)
	}
	return c, nil
}
