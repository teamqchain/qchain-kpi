package main

import (
	"fmt"
	"time"

	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/flow"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E11: lifecycle composites (cells in internal/cells, stages in internal/flow).

// e11Arms prepares an E11 (or probe-E11) block: the cell's stage through
// QChain's copied production code (internal/flow), with its inputs made
// before timing; the preparation's duration is recorded as prep_ns.
func e11Arms(b plan.Block) (*armSet, int, int64, error) {
	c, err := cells.ParseCell(b.Alg, b.Op)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("block: %w", err)
	}
	t0 := time.Now()
	fs, err := flow.Prepare(c, b.Seed)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("block %s: prepare: %w", b.Key(), err)
	}
	prepNS := time.Since(t0).Nanoseconds()
	set := &armSet{arms: fs.Arms, after: fs.After}
	if err := checkArmOrder(set, proto.E11Arms(b.Op)); err != nil {
		return nil, 0, 0, fmt.Errorf("block %s: %w", b.Key(), err)
	}
	return set, fs.P, prepNS, nil
}
