package main

import (
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// E6 (message size): estimate and add-on calibration.

// e6SignCheck is the per-sample check of an E6 sign block: every signature is
// verified, at the same message size.
func (x *est) e6SignCheck(bl plan.Block) (float64, bool) {
	op, size, err := proto.ParseE6Op(bl.Op)
	if err != nil || bl.Exp != "E6" || op != "sign" {
		return 0, false
	}
	return x.get(bl.Alg + "/" + proto.E6Op("verify", size)), true
}

// e6Prep estimates an E6 input set: the prep's own pool signs 64-character
// messages (E6's 64-byte cells), then every size's pool is signed and
// verified. Its one key generation is counted when a probe of it exists (an
// E6 add-on trial has none; one key per preparation is negligible against
// the blocks).
func (x *est) e6Prep(bl plan.Block, a algs.Alg) float64 {
	pool := float64(a.PoolSize())
	ps := x.f.med[bl.Alg+"/keygen"] + pool*(x.get(bl.Alg+"/"+proto.E6Op("sign", 64))+x.get(bl.Alg+"/"+proto.E6Op("verify", 64)))
	for _, size := range proto.E6Sizes {
		n := float64(proto.E6PoolSize(a.PoolSize(), size))
		ps += n * (x.get(bl.Alg+"/"+proto.E6Op("sign", size)) + x.get(bl.Alg+"/"+proto.E6Op("verify", size)))
	}
	return ps
}

// calibrateE6AddOn calibrates an E6 add-on trial from its probe pass with the
// first trial's rules (quality.FormulaText): each E6 cell (scheme, operation,
// message size) gets its fast/slow classification (probe median below 1 ms =
// fast) and, below 100x the effective timer resolution, a batch size. Every
// other threshold is copied unchanged from the machine's committed section.
func calibrateE6AddOn(dir, machine string, st *State, vs []*Verdict, base *quality.Machine) (*Calibration, error) {
	m, err := addOnMachine(base, st, machine)
	if err != nil {
		return nil, err
	}
	res, err := timerResolution(dir)
	if err != nil {
		return nil, fmt.Errorf("calibrate: %w", err)
	}
	m.Inputs["e6_timer_effective_resolution_ns"] = res
	m.ClassCalibratedFrom["E6"] = st.RunID
	m.BatchCalibratedFrom["E6"] = st.RunID
	cal := &Calibration{Machine: machine, TrialRun: st.RunID, Formulas: quality.FormulaText, Thresholds: m, AddOn: "E6",
		Notes: []string{"add-on calibration: only the fast/slow classification and batch sizes of the E6 cells (scheme, operation, message size) are new; every other value is the machine's committed thresholds, unchanged"}}
	probes, slow, batched := 0, 0, 0
	for _, v := range vs {
		r := v.Res
		if r.Block.Exp != "probe-E6" || r.Block.Attempt != 1 || len(r.Arms) == 0 {
			continue
		}
		probes++
		c := quality.Classify(r.Arms[0].ProbeNS)
		m.Class[r.Block.Alg+"/"+r.Block.Op] = c
		if c == "slow" {
			slow++
		}
		batched += probeBatches(m, r, res)
	}
	if probes == 0 {
		return nil, fmt.Errorf("calibrate: trial %s has no probe-E6 blocks", st.RunID)
	}
	m.Inputs["e6_probe_blocks"] = float64(probes)
	m.Inputs["e6_batched_arms"] = float64(batched)
	m.Inputs["e6_slow_cells"] = float64(slow)
	return cal, nil
}
