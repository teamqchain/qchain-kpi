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
	"strconv"
	"strings"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// Estimate is estimate.json: projected run time and raw-data size of the
// main plans on one machine, from that machine's trial run.
type Estimate struct {
	Machine     string         `json:"machine"`
	TrialRun    string         `json:"trial_run"`
	Plans       []PlanEstimate `json:"plans"`
	TotalHours  float64        `json:"total_hours"`
	TotalRawMB  float64        `json:"total_raw_mb"`
	Assumptions []string       `json:"assumptions"`
	// NotEstimated are the main plans this trial cannot estimate, with why.
	NotEstimated map[string]string `json:"not_estimated,omitempty"`
}

type PlanEstimate struct {
	Plan        string             `json:"plan"`
	File        string             `json:"file"`
	Blocks      int                `json:"blocks"`
	Hours       float64            `json:"hours"`
	PrepHours   float64            `json:"prep_hours"`
	CanaryHours float64            `json:"canary_hours"`
	ByExpHours  map[string]float64 `json:"by_experiment_hours"`
	RawMB       float64            `json:"raw_mb"`
	Missing     []string           `json:"missing_probe,omitempty"`
}

func cmdEstimate(args []string) error {
	fs := flag.NewFlagSet("estimate", flag.ContinueOnError)
	dir := fs.String("trial", "", "trial run folder (with calibration.json)")
	plans := fs.String("plans", "/opt/kpi/plans", "folder of plan files")
	machine := fs.String("machine", "", "machine id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *machine == "" {
		return errors.New("estimate: --trial and --machine are required")
	}
	e, err := estimate(*dir, *plans, *machine)
	if err != nil {
		return err
	}
	printEstimate(e)
	return nil
}

func estimateAll(dir, plansDir, machine string) error {
	e, err := estimate(dir, plansDir, machine)
	if err != nil {
		return err
	}
	printEstimate(e)
	return writeJSON(filepath.Join(dir, fileEstimate), e)
}

func printEstimate(e *Estimate) {
	fmt.Fprintf(os.Stderr, "estimate for %s from trial %s (excluding repeats):\n", e.Machine, e.TrialRun)
	for _, p := range e.Plans {
		fmt.Fprintf(os.Stderr, "  %-10s %5d blocks  %7.2f h  %8.1f MB raw\n", p.Plan, p.Blocks, p.Hours, p.RawMB)
	}
	fmt.Fprintf(os.Stderr, "  total               %7.2f h  %8.1f MB raw\n", e.TotalHours, e.TotalRawMB)
	var names []string
	for n := range e.NotEstimated {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(os.Stderr, "  %-10s not estimated: %s\n", n, e.NotEstimated[n])
	}
}

// addOnTrials are the experiments whose cells only their own add-on trial
// probes (E2's arms, E6's message sizes, E7's arms, E11's cells).
var addOnTrials = map[string]string{"E2": "trial-e2", "E6": "trial-e6", "E7": "trial-e7", "E11": "trial-e11"}

// trialFacts are the measurements the estimate is built from.
type trialFacts struct {
	med         map[string]float64 // alg/op -> probe median ns
	prep        map[string]float64 // E11: alg/op -> the block's own input preparation, ns
	overheadNS  float64            // per-block fixed cost (process, bootstrap, files)
	bytesPerRow float64
	cal         *Calibration
}

func loadTrialFacts(dir string) (*trialFacts, error) {
	raw, err := os.ReadFile(filepath.Join(dir, fileCalib))
	if err != nil {
		return nil, fmt.Errorf("estimate needs %s from the trial: %w", fileCalib, err)
	}
	var cal Calibration
	if err := json.Unmarshal(raw, &cal); err != nil {
		return nil, err
	}
	results, err := readJSONL[proto.Result](filepath.Join(dir, fileBlocksJL))
	if err != nil {
		return nil, err
	}
	f := &trialFacts{med: map[string]float64{}, prep: map[string]float64{}, cal: &cal}
	var rawBytes, rawRows float64
	wall := map[string]int64{}
	for _, r := range results {
		k := r.Block.Alg + "/" + r.Block.Op
		if proto.IsProbe(r.Block.Exp) {
			if r.PrepNS > 0 {
				f.prep[k] = float64(r.PrepNS)
			}
			// probe: the reuse arm (alg/op); probe-E2: every arm (alg/op/arm).
			for _, a := range r.Arms {
				xs := make([]float64, len(a.ProbeNS))
				for i, x := range a.ProbeNS {
					xs[i] = float64(x)
				}
				if len(xs) == 0 {
					continue
				}
				f.med[k+"/"+a.Name] = stats.Median(xs)
				if a.Name == proto.ArmReuse {
					if _, ok := f.med[k]; !ok || r.Block.Exp == "probe" {
						f.med[k] = f.med[k+"/"+a.Name]
					}
				}
			}
			continue
		}
		wall[r.Key] = r.WallNS
		if st, err := os.Stat(filepath.Join(dir, r.Block.RawPath())); err == nil {
			rawBytes += float64(st.Size())
			rawRows += float64(r.Rows)
		}
	}
	f.bytesPerRow = 2.5
	if rawRows > 0 {
		f.bytesPerRow = rawBytes / rawRows
	}
	// Per-block overhead from the journal: (done - start) - measured wall.
	jr, err := os.ReadFile(filepath.Join(dir, fileJournal))
	if err != nil {
		return nil, err
	}
	start := map[string]int64{}
	var over []float64
	for _, line := range strings.Split(string(jr), "\n") {
		fields := strings.SplitN(line, ",", 4)
		if len(fields) < 3 {
			continue
		}
		t, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		switch fields[1] {
		case "block_start":
			start[fields[2]] = t
		case "block_done":
			if s, ok := start[fields[2]]; ok {
				if w, ok := wall[fields[2]]; ok {
					over = append(over, float64(t-s-w))
				}
			}
		}
	}
	f.overheadNS = 0.5e9
	if len(over) > 0 {
		f.overheadNS = math.Max(stats.Median(over), 0)
	}
	return f, nil
}

// armSeconds models one arm of a block (coding plan §6 budgets).
func armSeconds(med, check float64, mode string, batchN int) (secs float64, rows float64) {
	if batchN < 1 {
		batchN = 1
	}
	if mode == "fast" {
		calls := 1 + 1000 + 10000*float64(batchN)
		checks := 1.0 + 1000 + 10000
		return (calls*med + checks*check + 100*med) / 1e9, 10001
	}
	n := math.Max(30, math.Ceil(60e9/(med*float64(batchN))))
	calls := 1 + 3 + n*float64(batchN)
	checks := 1 + 3 + n
	return (calls*med + checks*check + 2*med) / 1e9, n + 1
}

func estimate(dir, plansDir, machine string) (*Estimate, error) {
	f, err := loadTrialFacts(dir)
	if err != nil {
		return nil, err
	}
	if f.cal.Machine != machine {
		return nil, fmt.Errorf("estimate: calibration is for %s, not %s", f.cal.Machine, machine)
	}
	th := f.cal.Thresholds
	e := &Estimate{Machine: machine, TrialRun: f.cal.TrialRun, Assumptions: []string{
		"probe medians from the trial (cold call excluded)",
		fmt.Sprintf("per-block fixed overhead %.2f s (median of the trial)", f.overheadNS/1e9),
		fmt.Sprintf("raw data %.2f bytes per sample row (trial average, gzip)", f.bytesPerRow),
		"input-set generation per (algorithm, repetition) = key generation + P x (sign + verify) or P x (encaps + decaps)",
		"repeats and diagnostic re-runs are not included",
	}}
	files, _ := filepath.Glob(filepath.Join(plansDir, "*.json"))
	sort.Strings(files)
	for _, file := range files {
		p, err := plan.Load(file)
		if err != nil || p.Kind != plan.KindMain {
			continue
		}
		blocks, err := plan.Expand(p, 1)
		if err != nil {
			return nil, err
		}
		pe := PlanEstimate{Plan: p.Name, File: filepath.Base(file), Blocks: len(blocks), ByExpHours: map[string]float64{}}
		x := &est{f: f, th: th, missing: map[string]bool{}}
		missing, get := x.missing, x.get
		var secs, rows float64
		preps := map[string]bool{}
		add := func(exp string, bs, r float64) {
			pe.ByExpHours[exp] += bs / 3600
			secs += bs
			rows += r
		}
		for _, bl := range blocks {
			k := bl.Alg + "/" + bl.Op
			if bl.Exp == "E11" {
				bs, r := x.e11Block(bl)
				add(bl.Exp, bs, r)
				continue
			}
			if bl.Exp == "E12" {
				bs, r := x.e12Block(bl)
				add(bl.Exp, bs, r)
				continue
			}
			if proto.IsE3Null(bl.Exp, bl.Alg, bl.Op) {
				bs, r := x.e3NullBlock(bl)
				add(bl.Exp, bs, r)
				continue
			}
			a := algs.MustByID(bl.Alg)
			if bl.Exp == "E13" {
				bs, r := x.e13Block(bl, a)
				add(bl.Exp, bs, r)
				continue
			}
			if bl.Exp == "E9" {
				bs, r := x.e9Block(bl, a)
				add(bl.Exp, bs, r)
				if pk := proto.PrepName(bl.Alg, bl.Rep); !preps[pk] {
					preps[pk] = true
					pe.PrepHours += x.e9Prep(a) / 1e9 / 3600
				}
				continue
			}
			var med float64
			if v, ok := x.e7Median(k); ok && bl.Exp == "E7" {
				med = v
			} else {
				med = get(k)
			}
			check := 0.0
			switch bl.Op {
			case "sign":
				check = get(bl.Alg + "/verify")
			case "encaps":
				check = get(bl.Alg + "/decaps")
			}
			if c, ok := x.e6SignCheck(bl); ok {
				check = c
			}
			if bl.Exp == "E8" {
				bs, r := x.e8Block(bl, a, med, check)
				add(bl.Exp, bs, r)
				continue
			}
			mode := th.Class[k]
			if bl.Exp == "E2c" {
				mode = "fast"
			}
			bs := f.overheadNS / 1e9
			for _, arm := range proto.ArmsOf(bl.Exp, a, bl.Op) {
				am, ac := med, check
				if v, ok := f.med[k+"/"+arm]; ok && (bl.Exp == "E2" || bl.Exp == "E2c" || bl.Exp == "E7") {
					am = v
				}
				if bl.Exp == "E2c" {
					ac = e2cCheck(arm, am)
				}
				s, r := armSeconds(am, ac, mode, th.Batch[k+"/"+arm])
				bs += s
				rows += r
				if bl.Op == "keygen" && bl.Exp != "E2c" {
					// Deferred full checks of every 50th key (at least 3).
					n := math.Max(3, r/50)
					if a.Track == algs.Sig {
						bs += n * (get(bl.Alg+"/sign") + get(bl.Alg+"/verify")) / 1e9
					} else {
						bs += n * (get(bl.Alg+"/encaps") + get(bl.Alg+"/decaps")) / 1e9
					}
				}
			}
			add(bl.Exp, bs, 0)
			pk := proto.PrepName(bl.Alg, bl.Rep)
			if !preps[pk] {
				preps[pk] = true
				pool := float64(a.PoolSize())
				var ps float64
				if bl.Exp == "E6" {
					ps = x.e6Prep(bl, a)
				} else if a.Track == algs.Sig {
					ps = get(bl.Alg+"/keygen") + pool*(get(bl.Alg+"/sign")+get(bl.Alg+"/verify"))
				} else {
					ps = get(bl.Alg+"/keygen") + pool*(get(bl.Alg+"/encaps")+get(bl.Alg+"/decaps"))
				}
				pe.PrepHours += ps / 1e9 / 3600
			}
		}
		canarySecs := (secs/float64(p.CanaryEveryS) + 2) * 1.6
		pe.CanaryHours = canarySecs / 3600
		pe.Hours = (secs+canarySecs)/3600 + pe.PrepHours
		pe.RawMB = (rows*f.bytesPerRow + float64(len(blocks))*1024) / 1e6
		for k := range missing {
			pe.Missing = append(pe.Missing, k)
		}
		sort.Strings(pe.Missing)
		if f.cal.AddOn != "" && !planHasExp(blocks, f.cal.AddOn) {
			// An add-on trial probes only its own experiment's cells; the
			// plans of other experiments (E1, E2, ... in an E11 add-on) have
			// no probes to be estimated from and are left out.
			continue
		}
		// A machine's first trial cannot estimate the experiments that have
		// their own add-on trial (it has no probes of their cells), and no
		// estimate may be non-finite (NaN cannot be written to JSON: it
		// failed trial-e11 at its last step on 12 Oct). Probes missing only
		// for the input preparation stay listed in missing_probe.
		why := ""
		for exp, trial := range addOnTrials {
			if f.cal.AddOn != exp && planHasExp(blocks, exp) {
				why = "needs its add-on trial (" + trial + ")"
			}
		}
		if why == "" && (!finite(pe.Hours) || !finite(pe.RawMB)) {
			why = "the estimate is not finite"
		}
		if why != "" {
			if e.NotEstimated == nil {
				e.NotEstimated = map[string]string{}
			}
			e.NotEstimated[p.Name] = why
			continue
		}
		e.Plans = append(e.Plans, pe)
		e.TotalHours += pe.Hours
		e.TotalRawMB += pe.RawMB
	}
	return e, nil
}

// est is one plan's estimate in progress: the trial's facts, the machine's
// thresholds, and the probe medians a lookup found missing.
type est struct {
	f       *trialFacts
	th      *quality.Machine
	missing map[string]bool
}

// get is a probe median; a missing one is recorded and counts as 0.
func (x *est) get(k string) float64 {
	if v, ok := x.f.med[k]; ok {
		return v
	}
	x.missing[k] = true
	return 0
}

// planHasExp reports whether any block belongs to the experiment exp or its
// component variant (E2c).
func planHasExp(blocks []plan.Block, exp string) bool {
	for _, b := range blocks {
		if b.Exp == exp || b.Exp == exp+"c" {
			return true
		}
	}
	return false
}
