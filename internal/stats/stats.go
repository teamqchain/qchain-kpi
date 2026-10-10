// Package stats holds the small set of statistics the harness computes while
// running (block summaries, drift checks, calibration). The full analysis for
// the paper (bootstrap confidence intervals over repetitions, Mann-Whitney U,
// Cliff's delta, TOST) is done offline from the raw samples.
package stats

import (
	"math"
	"math/rand/v2"
	"sort"
)

// Summary describes one arm of one block.
type Summary struct {
	N        int     `json:"n"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Mean     float64 `json:"mean"`
	SD       float64 `json:"sd"`
	CoV      float64 `json:"cov"`
	P50      float64 `json:"p50"`
	P90      float64 `json:"p90"`
	P95      float64 `json:"p95"`
	P99      float64 `json:"p99"`
	P999     float64 `json:"p999"`
	BootSE50 float64 `json:"boot_se_p50"` // bootstrap standard error of the median
}

// Quantile returns the q-quantile (0..1) of an ascending-sorted slice using
// linear interpolation between closest ranks (Hyndman & Fan type 7, the
// default of NumPy and R).
func Quantile(sorted []float64, q float64) float64 {
	n := len(sorted)
	switch {
	case n == 0:
		return math.NaN()
	case n == 1 || q <= 0:
		return sorted[0]
	case q >= 1:
		return sorted[n-1]
	}
	h := float64(n-1) * q
	lo := math.Floor(h)
	i := int(lo)
	if i+1 >= n {
		return sorted[n-1]
	}
	return sorted[i] + (h-lo)*(sorted[i+1]-sorted[i])
}

// Median returns the median of xs (xs is not modified).
func Median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return Quantile(s, 0.5)
}

// RobustSD returns 1.4826 x the median absolute deviation, a robust estimate of
// the standard deviation.
func RobustSD(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	m := Median(xs)
	dev := make([]float64, len(xs))
	for i, x := range xs {
		dev[i] = math.Abs(x - m)
	}
	return 1.4826 * Median(dev)
}

// Summarize computes the block summary of a set of samples (nanoseconds per
// operation). bootReps bootstrap resamples (with a fixed seed) estimate the
// standard error of the median; 0 skips the bootstrap.
func Summarize(ns []float64, bootReps int, seed uint64) Summary {
	n := len(ns)
	if n == 0 {
		return Summary{}
	}
	s := append([]float64(nil), ns...)
	sort.Float64s(s)
	var sum float64
	for _, x := range s {
		sum += x
	}
	mean := sum / float64(n)
	var ss float64
	for _, x := range s {
		ss += (x - mean) * (x - mean)
	}
	sd := 0.0
	if n > 1 {
		sd = math.Sqrt(ss / float64(n-1))
	}
	out := Summary{
		N: n, Min: s[0], Max: s[n-1], Mean: mean, SD: sd,
		P50: Quantile(s, 0.50), P90: Quantile(s, 0.90), P95: Quantile(s, 0.95),
		P99: Quantile(s, 0.99), P999: Quantile(s, 0.999),
	}
	if mean != 0 {
		out.CoV = sd / mean
	}
	if bootReps > 0 && n > 1 {
		out.BootSE50 = BootstrapSEMedian(ns, bootReps, seed)
	}
	return out
}

// BootstrapSEMedian estimates the standard error of the median by resampling
// with replacement reps times (deterministic for a given seed).
func BootstrapSEMedian(xs []float64, reps int, seed uint64) float64 {
	n := len(xs)
	if n < 2 || reps < 2 {
		return 0
	}
	rng := rand.New(rand.NewPCG(seed, 0x6b70692d626f6f74)) // "kpi-boot"
	buf := make([]float64, n)
	meds := make([]float64, reps)
	for r := 0; r < reps; r++ {
		for i := range buf {
			buf[i] = xs[rng.IntN(n)]
		}
		sort.Float64s(buf)
		meds[r] = Quantile(buf, 0.5)
	}
	var mean float64
	for _, m := range meds {
		mean += m
	}
	mean /= float64(reps)
	var ss float64
	for _, m := range meds {
		ss += (m - mean) * (m - mean)
	}
	return math.Sqrt(ss / float64(reps-1))
}

// tq95 holds one-sided 95 % (two-sided 90 %) Student-t quantiles for df 1..30.
var tq95 = [...]float64{0, 6.314, 2.920, 2.353, 2.132, 2.015, 1.943, 1.895, 1.860, 1.833, 1.812,
	1.796, 1.782, 1.771, 1.761, 1.753, 1.746, 1.740, 1.734, 1.729, 1.725,
	1.721, 1.717, 1.714, 1.711, 1.708, 1.706, 1.703, 1.701, 1.699, 1.697}

// TQuantile95 returns the 0.95 quantile of Student's t with df degrees of
// freedom (the critical value of a 90 % two-sided interval, as used by TOST).
func TQuantile95(df int) float64 {
	if df < 1 {
		return math.NaN()
	}
	if df < len(tq95) {
		return tq95[df]
	}
	return 1.645
}

// MeanCI90 returns the mean of xs and the half-width of its two-sided 90 %
// t-interval.
func MeanCI90(xs []float64) (mean, half float64) {
	n := len(xs)
	if n == 0 {
		return math.NaN(), math.NaN()
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(n)
	if n < 2 {
		return mean, math.Inf(1)
	}
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	sd := math.Sqrt(ss / float64(n-1))
	return mean, TQuantile95(n-1) * sd / math.Sqrt(float64(n))
}
