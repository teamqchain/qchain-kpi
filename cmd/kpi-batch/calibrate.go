package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// Calibration is calibration.json: one machine's thresholds derived from its
// trial run by the formulas fixed in code (quality.FormulaText). It is merged
// into config/thresholds.json and committed before that machine's first main
// run, and never changed afterwards.
type Calibration struct {
	Machine    string           `json:"machine"`
	TrialRun   string           `json:"trial_run"`
	Formulas   string           `json:"formulas"`
	Thresholds *quality.Machine `json:"thresholds"`
	Notes      []string         `json:"notes,omitempty"`
}

func cmdCalibrate(args []string) error {
	fs := flag.NewFlagSet("calibrate", flag.ContinueOnError)
	dir := fs.String("run", "", "trial run folder")
	machine := fs.String("machine", "", "machine id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *machine == "" {
		return errors.New("calibrate: --run and --machine are required")
	}
	th, err := trialThresholds(*dir)
	if err != nil {
		return err
	}
	cal, err := calibrate(*dir, *machine, th)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cal, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// trialThresholds returns the thresholds a run was evaluated with (manifest).
func trialThresholds(dir string) (*quality.Machine, error) {
	raw, err := os.ReadFile(filepath.Join(dir, fileManifest))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Thresholds.Values == nil {
		return nil, errors.New("manifest has no thresholds")
	}
	return m.Thresholds.Values, nil
}

func calibrateRun(dir, machine string, trialTH *quality.Machine) error {
	cal, err := calibrate(dir, machine, trialTH)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, fileCalib), cal)
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// timerResolution reads the effective timer resolution from the run's E0.
func timerResolution(dir string) (float64, error) {
	raw, err := os.ReadFile(filepath.Join(dir, dirEnv, "e0-s1.json"))
	if err != nil {
		return 0, err
	}
	var e struct {
		Clock struct {
			EffectiveResNS int64 `json:"effective_resolution_ns"`
		} `json:"clock"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return 0, err
	}
	if e.Clock.EffectiveResNS <= 0 {
		return 0, errors.New("E0 has no effective timer resolution")
	}
	return float64(e.Clock.EffectiveResNS), nil
}

func calibrate(dir, machine string, trialTH *quality.Machine) (*Calibration, error) {
	st, err := loadState(dir)
	if err != nil {
		return nil, err
	}
	if st.PlanKind != plan.KindTrial {
		return nil, fmt.Errorf("calibrate: %s is a %s run, not a trial", st.RunID, st.PlanKind)
	}
	if st.Machine != machine {
		return nil, fmt.Errorf("calibrate: run %s is from %s, not %s", st.RunID, st.Machine, machine)
	}
	in, err := loadEval(dir, st, trialTH, true)
	if err != nil {
		return nil, err
	}
	vs := evaluate(in)
	decideBase(st.PlanKind, vs, st.Seed, copyMap(st.Requeued), copyMap(st.Diagnosed), st.Base) // marks headline attempts

	cal := &Calibration{Machine: machine, TrialRun: st.RunID, Formulas: quality.FormulaText}
	m := quality.Defaults()
	m.CalibratedFrom = st.RunID
	m.Batch = map[string]int{}
	m.Inputs = map[string]float64{}
	cal.Thresholds = m

	// 1. Fast/slow classification from the probe pass.
	probeMed := map[string]float64{}
	for _, v := range vs {
		r := v.Res
		if r.Block.Exp != "probe" || r.Block.Attempt != 1 || len(r.Arms) != 1 {
			continue
		}
		k := r.Block.Alg + "/" + r.Block.Op
		m.Class[k] = quality.Classify(r.Arms[0].ProbeNS)
		xs := make([]float64, len(r.Arms[0].ProbeNS))
		for i, x := range r.Arms[0].ProbeNS {
			xs[i] = float64(x)
		}
		probeMed[k] = stats.Median(xs)
	}
	var missing []string
	for _, a := range algs.All() {
		for _, op := range a.Ops() {
			if _, ok := m.Class[a.ID+"/"+op]; !ok {
				missing = append(missing, a.ID+"/"+op)
			}
		}
	}
	if len(missing) > 0 {
		cal.Notes = append(cal.Notes, "no probe for: "+strings.Join(missing, ", "))
	}

	// 2. Drift limit from the spread of repetition medians (headline attempts).
	cells := map[string][]float64{}
	for _, v := range vs {
		if v.Res.Block.Exp == "E1" && v.Headline && len(v.Res.Arms) == 1 {
			cells[v.Res.Block.Cell()] = append(cells[v.Res.Block.Cell()], v.Res.Arms[0].Stats.P50)
		}
	}
	d95, lim := quality.DriftFromTrial(cells)
	m.DriftFrac = lim
	if finite(d95) {
		m.Inputs["drift_d95"] = d95
	} else {
		cal.Notes = append(cal.Notes, "no E1 cell with 3 or more repetitions: drift limit stays at the 3% floor")
	}
	m.Inputs["drift_cells"] = float64(len(cells))

	// 3. Canary limit.
	canaries := append([]canaryRec(nil), in.Canaries...)
	sort.Slice(canaries, func(i, j int) bool { return canaries[i].Seq < canaries[j].Seq })
	rates := make([]float64, len(canaries))
	for i, c := range canaries {
		rates[i] = c.MedianRate
	}
	sd, clim := quality.CanaryFromTrial(rates)
	m.CanaryFrac = clim
	if finite(sd) {
		m.Inputs["canary_robust_sd"] = sd
	} else {
		cal.Notes = append(cal.Notes, "fewer than 3 canaries: canary limit stays at the 3% floor")
	}
	m.Inputs["canaries"] = float64(len(rates))

	// 4. E10 equivalence margin per ML-DSA set from the A/A arms.
	type pair struct{ issuer, again float64 }
	byAlg := map[string]map[int]pair{}
	for _, v := range vs {
		r := v.Res
		if r.Block.Exp != "E10" || !v.Headline {
			continue
		}
		var p pair
		for _, a := range r.Arms {
			switch a.Name {
			case proto.ArmIssuer:
				p.issuer = a.Stats.P50
			case proto.ArmIssuerAgain:
				p.again = a.Stats.P50
			}
		}
		if p.issuer > 0 && p.again > 0 {
			if byAlg[r.Block.Alg] == nil {
				byAlg[r.Block.Alg] = map[int]pair{}
			}
			byAlg[r.Block.Alg][r.Block.Rep] = p
		}
	}
	for _, a := range []string{"ml-dsa-44", "ml-dsa-65", "ml-dsa-87"} {
		reps := byAlg[a]
		var keys []int
		for k := range reps {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		var is, ag []float64
		for _, k := range keys {
			is = append(is, reps[k].issuer)
			ag = append(ag, reps[k].again)
		}
		floor, margin := quality.AAFloor(is, ag)
		if !finite(floor) {
			cal.Notes = append(cal.Notes, fmt.Sprintf("%s: fewer than 2 E10 repetitions; margin stays at the 2%% floor", a))
		} else {
			m.Inputs["aa_floor_"+a] = floor
		}
		m.E10Margin[a] = margin
	}

	// 5. Batching of operations below 100x the effective timer resolution.
	res, err := timerResolution(dir)
	if err != nil {
		return nil, fmt.Errorf("calibrate: %w", err)
	}
	m.Inputs["timer_effective_resolution_ns"] = res
	for k, med := range probeMed {
		if med <= 0 || med >= 100*res {
			continue
		}
		n := int(math.Ceil(100 * res / med))
		m.Batch[k+"/"+proto.ArmReuse] = n
		if alg, op, _ := strings.Cut(k, "/"); op == "sign" && strings.HasPrefix(alg, "ml-dsa-") {
			for _, arm := range []string{proto.ArmIssuer, proto.ArmHolder, proto.ArmIssuerAgain} {
				m.Batch[k+"/"+arm] = n
			}
		}
	}

	flagged := 0
	for _, v := range vs {
		if v.flagged() {
			flagged++
		}
	}
	m.Inputs["trial_attempts"] = float64(len(vs))
	m.Inputs["trial_flagged_attempts"] = float64(flagged)
	return cal, nil
}
