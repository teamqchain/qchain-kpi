package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"qchain-kpi/internal/stats"
)

// Starting values (research plan §9, approved 10 Oct). Each machine's trial
// run replaces the calibrated ones with numbers derived by the formulas below.
const (
	DefaultStealPct   = 2.0  // VM steal, % of the pinned vCPU's time
	DefaultOtherPct   = 5.0  // other CPU, % of capacity
	DefaultDriftFrac  = 0.03 // block median vs reference
	DefaultCanaryFrac = 0.03 // canary vs the batch's first canary
	DefaultE10Margin  = 0.02 // equivalence margin (fraction)
)

// Machine holds one machine's thresholds.
type Machine struct {
	CalibratedFrom string             `json:"calibrated_from"` // trial run id, "" = starting values
	StealPct       float64            `json:"steal_pct"`
	OtherPct       float64            `json:"other_pct"`
	DriftFrac      float64            `json:"drift_frac"`
	CanaryFrac     float64            `json:"canary_frac"`
	E10Margin      map[string]float64 `json:"e10_margin"`      // per ML-DSA parameter set
	Class          map[string]string  `json:"classification"`  // "<alg>/<op>" -> "fast" | "slow"
	Batch          map[string]int     `json:"batch,omitempty"` // "<alg>/<op>/<arm>" -> calls per sample
	// BatchCalibratedFrom records the add-on trials (probe passes of arms
	// that the machine's first trial did not cover) whose batch sizes were
	// merged into Batch: experiment -> trial run id. E2 main runs need "E2".
	BatchCalibratedFrom map[string]string  `json:"batch_calibrated_from,omitempty"`
	Inputs              map[string]float64 `json:"calibration_inputs,omitempty"` // d95, canary robust SD, A/A floors
}

// File is config/thresholds.json.
type File struct {
	Version  int                 `json:"version"`
	Formulas string              `json:"formulas"`
	Machines map[string]*Machine `json:"machines"`
}

// FormulaText is recorded in the file so the rules travel with the numbers.
const FormulaText = "drift_frac = max(0.03, 2*d95) where d95 = 95th percentile of |rep median / median of rep medians - 1| over all trial (alg,op) cells; " +
	"canary_frac = max(0.03, 3*robustSD(canary_t/canary_first - 1)); " +
	"e10_margin[set] = max(0.02, 2*F) where F = |mean| + 90% t-half-width of the per-rep log(median issuer / median issuer-again); " +
	"classification = fast if the probe median < 1 ms, else slow; " +
	"batch[alg/op/arm] = ceil(100 * effective timer resolution / probe median) when the probe median is below 100x the resolution, else 1; " +
	"steal_pct = 2, other_pct = 5 (approved starting values; not recalibrated)"

// Defaults returns the starting values for a machine.
func Defaults() *Machine {
	return &Machine{StealPct: DefaultStealPct, OtherPct: DefaultOtherPct, DriftFrac: DefaultDriftFrac,
		CanaryFrac: DefaultCanaryFrac, E10Margin: map[string]float64{}, Class: map[string]string{}}
}

// Load reads the thresholds file; a missing file yields an empty one.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &File{Version: 1, Formulas: FormulaText, Machines: map[string]*Machine{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("thresholds: %w", err)
	}
	if f.Machines == nil {
		f.Machines = map[string]*Machine{}
	}
	return &f, nil
}

// For returns a machine's thresholds, or the starting values.
func (f *File) For(machine string) (*Machine, bool) {
	if m, ok := f.Machines[machine]; ok && m != nil {
		return m, true
	}
	return Defaults(), false
}

// Hash identifies the exact thresholds a batch ran with (resume refuses if it changes).
func HashFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "absent"
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// --------------------------------------------------------------- calibration

// DriftFromTrial computes d95 and the drift limit from rep medians per cell.
func DriftFromTrial(cells map[string][]float64) (d95, limit float64) {
	var devs []float64
	for _, meds := range cells {
		if len(meds) < 3 {
			continue
		}
		m := stats.Median(meds)
		if m <= 0 {
			continue
		}
		for _, x := range meds {
			devs = append(devs, math.Abs(x/m-1))
		}
	}
	if len(devs) == 0 {
		return math.NaN(), DefaultDriftFrac
	}
	sort.Float64s(devs)
	d95 = stats.Quantile(devs, 0.95)
	return d95, math.Max(DefaultDriftFrac, 2*d95)
}

// CanaryFromTrial computes the robust SD of canary deviations and the limit.
func CanaryFromTrial(rates []float64) (sd, limit float64) {
	if len(rates) < 3 || rates[0] <= 0 {
		return math.NaN(), DefaultCanaryFrac
	}
	devs := make([]float64, 0, len(rates)-1)
	for _, r := range rates[1:] {
		devs = append(devs, r/rates[0]-1)
	}
	sd = stats.RobustSD(devs)
	return sd, math.Max(DefaultCanaryFrac, 3*sd)
}

// AAFloor computes F from per-rep medians of the two identical issuer arms.
func AAFloor(issuer, issuerAgain []float64) (floor, margin float64) {
	n := len(issuer)
	if n != len(issuerAgain) || n < 2 {
		return math.NaN(), DefaultE10Margin
	}
	lr := make([]float64, n)
	for i := range issuer {
		lr[i] = math.Log(issuer[i] / issuerAgain[i])
	}
	mean, half := stats.MeanCI90(lr)
	floor = math.Abs(mean) + half
	return floor, math.Max(DefaultE10Margin, 2*floor)
}

// Classify returns "fast" when the probe median is below 1 ms.
func Classify(probeNS []int64) string {
	xs := make([]float64, len(probeNS))
	for i, v := range probeNS {
		xs[i] = float64(v)
	}
	if stats.Median(xs) < 1e6 {
		return "fast"
	}
	return "slow"
}
