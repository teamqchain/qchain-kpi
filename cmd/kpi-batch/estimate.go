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
}

// trialFacts are the measurements the estimate is built from.
type trialFacts struct {
	med         map[string]float64 // alg/op -> probe median ns
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
	f := &trialFacts{med: map[string]float64{}, cal: &cal}
	var rawBytes, rawRows float64
	wall := map[string]int64{}
	for _, r := range results {
		k := r.Block.Alg + "/" + r.Block.Op
		if proto.IsProbe(r.Block.Exp) {
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
		missing := map[string]bool{}
		get := func(k string) float64 {
			if v, ok := f.med[k]; ok {
				return v
			}
			missing[k] = true
			return 0
		}
		var secs, rows float64
		preps := map[string]bool{}
		for _, bl := range blocks {
			a := algs.MustByID(bl.Alg)
			k := bl.Alg + "/" + bl.Op
			med := get(k)
			check := 0.0
			switch bl.Op {
			case "sign":
				check = get(bl.Alg + "/verify")
			case "encaps":
				check = get(bl.Alg + "/decaps")
			}
			mode := th.Class[k]
			if bl.Exp == "E2c" {
				mode = "fast"
			}
			bs := f.overheadNS / 1e9
			for _, arm := range proto.ArmsOf(bl.Exp, a, bl.Op) {
				am, ac := med, check
				if v, ok := f.med[k+"/"+arm]; ok && (bl.Exp == "E2" || bl.Exp == "E2c") {
					am = v
				}
				if bl.Exp == "E2c" {
					// Components check cheaply; init/clean restore their
					// object outside the timing (about one more call).
					ac = 0
					if arm == proto.ArmInit || arm == proto.ArmClean {
						ac = am
					}
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
			pe.ByExpHours[bl.Exp] += bs / 3600
			secs += bs
			pk := proto.PrepName(bl.Alg, bl.Rep)
			if !preps[pk] {
				preps[pk] = true
				pool := float64(a.PoolSize())
				var ps float64
				if a.Track == algs.Sig {
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
		e.Plans = append(e.Plans, pe)
		e.TotalHours += pe.Hours
		e.TotalRawMB += pe.RawMB
	}
	return e, nil
}
