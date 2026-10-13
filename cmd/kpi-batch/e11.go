package main

import (
	"fmt"
	"math"

	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// E11 (lifecycle composites): estimate and add-on calibration.

// e11Block estimates an E11 block: 20 warm-up calls + 2,000 samples, or the
// slow rule (at least 60 s and 30 samples) at 30 ms and above, per arm; the
// block's own input preparation is untimed but takes wall time.
func (x *est) e11Block(bl plan.Block) (secs, rows float64) {
	k := bl.Alg + "/" + bl.Op
	bs := x.f.overheadNS/1e9 + x.f.prep[k]/1e9
	for _, arm := range proto.E11Arms(bl.Op) {
		med, ok := x.f.med[k+"/"+arm]
		if !ok {
			x.missing[k+"/"+arm] = true
			continue
		}
		check := 0.25 * med // signature and outcome checks after each call
		var n float64
		if x.th.Class[k] == "slow" {
			n = math.Max(30, math.Ceil(60e9/med))
			bs += ((3+n)*med + n*check) / 1e9
		} else {
			n = 2000
			bs += ((20+n)*med + n*check) / 1e9
		}
		rows += n + 1
	}
	return bs, rows
}

// calibrateE11AddOn classifies every E11 cell of an add-on trial's probe-E11
// pass as fast or slow with the approved composite rule (probe median of the
// whole stage at least 30 ms = slow). Every other threshold is copied
// unchanged from the machine's committed section.
func calibrateE11AddOn(machine string, st *State, vs []*Verdict, base *quality.Machine) (*Calibration, error) {
	if base == nil || base.CalibratedFrom == "" {
		return nil, fmt.Errorf("calibrate: the add-on trial %s ran without calibrated thresholds for %s", st.RunID, machine)
	}
	m, err := copyMachine(base)
	if err != nil {
		return nil, err
	}
	if m.Class == nil {
		m.Class = map[string]string{}
	}
	if m.Inputs == nil {
		m.Inputs = map[string]float64{}
	}
	if m.ClassCalibratedFrom == nil {
		m.ClassCalibratedFrom = map[string]string{}
	}
	m.ClassCalibratedFrom["E11"] = st.RunID
	cal := &Calibration{Machine: machine, TrialRun: st.RunID, Formulas: quality.FormulaText, Thresholds: m, AddOn: "E11",
		Notes: []string{"add-on calibration: only the fast/slow classification of the E11 cells is new (probe median of the whole stage >= 30 ms = slow); every other value is the machine's committed thresholds, unchanged"}}
	probes, slow := 0, 0
	for _, v := range vs {
		r := v.Res
		if r.Block.Exp != "probe-E11" || r.Block.Attempt != 1 || len(r.Arms) == 0 {
			continue
		}
		probes++
		c := quality.ClassifyComposite(r.Arms[0].ProbeNS, cells.SlowCompositeNS)
		m.Class[r.Block.Alg+"/"+r.Block.Op] = c
		if c == "slow" {
			slow++
		}
	}
	if probes == 0 {
		return nil, fmt.Errorf("calibrate: trial %s has no probe-E11 blocks", st.RunID)
	}
	m.Inputs["e11_probe_blocks"] = float64(probes)
	m.Inputs["e11_slow_cells"] = float64(slow)
	if want := len(cells.Grid()); probes < want {
		cal.Notes = append(cal.Notes, fmt.Sprintf("probe-E11 covered %d of the %d grid cells", probes, want))
	}
	return cal, nil
}
