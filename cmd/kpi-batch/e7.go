package main

import (
	"fmt"

	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// E7 (rejection time): estimate and add-on calibration.

// e7Median is an E7 block's operation median: probe-E7's valid arm (an E7
// add-on trial has no plain probe of the operation).
func (x *est) e7Median(k string) (float64, bool) {
	v, ok := x.f.med[k+"/"+proto.ArmValid]
	return v, ok
}

// calibrateE7AddOn calibrates an E7 add-on trial from its probe pass with the
// first trial's batch rule: only the E7 arms' batch sizes are new, since an
// E7 block takes its operation's classification (verify or decaps), already
// frozen by the first trial. Every other threshold is copied unchanged.
func calibrateE7AddOn(dir, machine string, st *State, vs []*Verdict, base *quality.Machine) (*Calibration, error) {
	m, err := addOnMachine(base, st, machine)
	if err != nil {
		return nil, err
	}
	res, err := timerResolution(dir)
	if err != nil {
		return nil, fmt.Errorf("calibrate: %w", err)
	}
	m.Inputs["e7_timer_effective_resolution_ns"] = res
	m.BatchCalibratedFrom["E7"] = st.RunID
	cal := &Calibration{Machine: machine, TrialRun: st.RunID, Formulas: quality.FormulaText, Thresholds: m, AddOn: "E7",
		Notes: []string{"add-on calibration: only the batch sizes of the E7 arms are new (each E7 block takes its operation's classification from the first trial); every other value is the machine's committed thresholds, unchanged"}}
	probes, batched := 0, 0
	for _, v := range vs {
		r := v.Res
		if r.Block.Exp != "probe-E7" || r.Block.Attempt != 1 || len(r.Arms) == 0 {
			continue
		}
		probes++
		batched += probeBatches(m, r, res)
	}
	if probes == 0 {
		return nil, fmt.Errorf("calibrate: trial %s has no probe-E7 blocks", st.RunID)
	}
	m.Inputs["e7_probe_blocks"] = float64(probes)
	m.Inputs["e7_batched_arms"] = float64(batched)
	return cal, nil
}
