package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// E2 (integration cost): estimate and add-on calibration.

// e2cCheck is an E2c arm's per-sample check cost: components check cheaply;
// init and clean restore their object outside the timing (about one more call).
func e2cCheck(arm string, am float64) float64 {
	if arm == proto.ArmInit || arm == proto.ArmClean {
		return am
	}
	return 0
}

// calibrateE2AddOn derives the batch sizes of the E2 and E2c arms from an
// add-on trial's probe-E2 pass, with the same rule as the first trial
// (quality.FormulaText): batch = ceil(100 * effective timer resolution / probe
// median) for arms whose probe median is below 100x the resolution. The reuse
// arm is skipped: it is E1's arm, batched (or not) by the first trial. Every
// other threshold is copied unchanged from the machine's committed section,
// which is what this trial ran with.
func calibrateE2AddOn(dir, machine string, st *State, vs []*Verdict, base *quality.Machine) (*Calibration, error) {
	if base == nil || base.CalibratedFrom == "" {
		return nil, fmt.Errorf("calibrate: the add-on trial %s ran without calibrated thresholds for %s", st.RunID, machine)
	}
	raw, err := json.Marshal(base)
	if err != nil {
		return nil, err
	}
	m := &quality.Machine{}
	if err := json.Unmarshal(raw, m); err != nil {
		return nil, err
	}
	if m.Batch == nil {
		m.Batch = map[string]int{}
	}
	if m.Inputs == nil {
		m.Inputs = map[string]float64{}
	}
	if m.BatchCalibratedFrom == nil {
		m.BatchCalibratedFrom = map[string]string{}
	}
	m.BatchCalibratedFrom["E2"] = st.RunID
	cal := &Calibration{Machine: machine, TrialRun: st.RunID, Formulas: quality.FormulaText, Thresholds: m, AddOn: "E2",
		Notes: []string{"add-on calibration: only the batch sizes of the E2 and E2c arms are new; every other value is the machine's committed thresholds, unchanged"}}

	res, err := timerResolution(dir)
	if err != nil {
		return nil, fmt.Errorf("calibrate: %w", err)
	}
	m.Inputs["e2_timer_effective_resolution_ns"] = res
	seen := map[string]bool{}
	probes := 0
	for _, v := range vs {
		r := v.Res
		if r.Block.Exp != "probe-E2" || r.Block.Attempt != 1 {
			continue
		}
		probes++
		k := r.Block.Alg + "/" + r.Block.Op
		seen[k] = true
		for _, a := range r.Arms {
			if a.Name == proto.ArmReuse || len(a.ProbeNS) == 0 {
				continue
			}
			xs := make([]float64, len(a.ProbeNS))
			for i, x := range a.ProbeNS {
				xs[i] = float64(x)
			}
			med := stats.Median(xs)
			if med > 0 && med < 100*res {
				m.Batch[k+"/"+a.Name] = int(math.Ceil(100 * res / med))
			}
		}
	}
	m.Inputs["e2_probe_blocks"] = float64(probes)
	var missing []string
	for _, a := range algs.All() {
		for _, op := range a.Ops() {
			if !seen[a.ID+"/"+op] {
				missing = append(missing, a.ID+"/"+op)
			}
		}
	}
	if len(missing) > 0 {
		cal.Notes = append(cal.Notes, "no probe-E2 for: "+strings.Join(missing, ", "))
	}
	return cal, nil
}
