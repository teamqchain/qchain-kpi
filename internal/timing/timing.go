// Package timing implements the sampling rules of the study (coding plan §6):
//
//   - first call: the first call of every arm in the process is timed and
//     recorded separately ("cold:<arm>" when no other crypto operation ran
//     before it in the process, otherwise "first:<arm>");
//   - warm-up: untimed calls (fast: 1,000; slow: 3);
//   - fast operations: 10,000 timed samples per arm;
//   - slow operations: sample until the arm's accumulated measured time is at
//     least the time box (60 s) AND it has at least 30 samples; no cap;
//   - several arms in one block are interleaved in a seeded random order per
//     round, and every arm keeps the full budget;
//   - operations below the timer's useful resolution are timed in batches of B
//     calls per sample (the raw file records the batch total; summaries divide
//     by B);
//   - every result is checked for correctness outside the timed region;
//   - runtime.GC() is called only outside the timed region (before sampling and
//     every GCEvery samples), so garbage collection never lands inside a sample.
//
// The caller pins the goroutine to its OS thread (runtime.LockOSThread) and
// sets the GC policy for the process.
package timing

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime"
	"time"
)

// Mode selects the sampling budget.
type Mode string

const (
	Fast Mode = "fast"
	Slow Mode = "slow"
)

// Arm is one measured variant inside a block.
type Arm struct {
	Name string
	// Call performs ONE operation using input pool index i and returns the
	// size of its output in bytes (or 0). It must not do any work that is not
	// part of the operation being measured.
	Call func(i int) (int, error)
	// Check verifies the result of the most recent Call for index i. It runs
	// outside the timed region. Nil means nothing to check.
	Check func(i int, nbytes int) error
	// Batch is the number of operations per timed sample (>= 1).
	Batch int
}

// Config is the budget for one block.
type Config struct {
	Mode       Mode
	Warmup     int           // untimed calls per arm before sampling
	Samples    int           // fast: timed samples per arm
	TimeBox    time.Duration // slow: minimum accumulated measured time per arm
	MinSamples int           // slow: minimum samples per arm
	FirstLabel string        // "cold" or "first"
	GCEvery    int           // untimed runtime.GC() every N samples (0 = never)
	Seed       uint64        // arm order per round
	MaxWall    time.Duration // safety limit for the whole block (0 = none)
	// Sink receives every timed sample (ns = batch total) and every first call.
	Sink func(arm string, ns int64, nbytes int)
}

// FastConfig returns the approved budget for fast operations.
func FastConfig(seed uint64, firstLabel string) Config {
	return Config{Mode: Fast, Warmup: 1000, Samples: 10000, FirstLabel: firstLabel, GCEvery: 1000, Seed: seed}
}

// SlowConfig returns the approved budget for slow operations.
func SlowConfig(seed uint64, firstLabel string) Config {
	return Config{Mode: Slow, Warmup: 3, TimeBox: 60 * time.Second, MinSamples: 30,
		FirstLabel: firstLabel, GCEvery: 1000, Seed: seed}
}

// ArmResult is what one arm produced.
type ArmResult struct {
	Name      string
	Batch     int
	FirstNS   int64     // the first call (cold or first)
	PerOpNS   []float64 // one value per sample: sample total / batch
	SumNS     int64     // accumulated measured time
	LastBytes int
	MinBytes  int
	MaxBytes  int
}

// Result of a block.
type Result struct {
	Arms      []ArmResult
	Truncated bool // MaxWall reached before every arm met its budget
	WallNS    int64
	GCForced  int // untimed runtime.GC() calls made by Run
}

// ErrCheck wraps a correctness failure: the block is invalid and must not be used.
var ErrCheck = errors.New("correctness check failed")

func batchOf(a Arm) int {
	if a.Batch < 1 {
		return 1
	}
	return a.Batch
}

// Run executes the block.
func Run(arms []Arm, cfg Config) (Result, error) {
	if len(arms) == 0 {
		return Result{}, errors.New("timing: no arms")
	}
	if cfg.Sink == nil {
		cfg.Sink = func(string, int64, int) {}
	}
	if cfg.FirstLabel == "" {
		cfg.FirstLabel = "first"
	}
	switch cfg.Mode {
	case Fast:
		if cfg.Samples < 1 {
			return Result{}, errors.New("timing: fast mode needs Samples >= 1")
		}
	case Slow:
		if cfg.TimeBox <= 0 || cfg.MinSamples < 1 {
			return Result{}, errors.New("timing: slow mode needs TimeBox > 0 and MinSamples >= 1")
		}
	default:
		return Result{}, fmt.Errorf("timing: unknown mode %q", cfg.Mode)
	}

	res := Result{Arms: make([]ArmResult, len(arms))}
	idx := make([]int, len(arms))
	start := time.Now()

	// 1. First call of every arm, in declared order (deterministic).
	for k, a := range arms {
		t0 := time.Now()
		nb, err := a.Call(0)
		d := time.Since(t0).Nanoseconds()
		if err != nil {
			return res, fmt.Errorf("arm %s first call: %w", a.Name, err)
		}
		if a.Check != nil {
			if err := a.Check(0, nb); err != nil {
				return res, fmt.Errorf("%w: arm %s first call: %v", ErrCheck, a.Name, err)
			}
		}
		res.Arms[k] = ArmResult{Name: a.Name, Batch: batchOf(a), FirstNS: d, LastBytes: nb, MinBytes: nb, MaxBytes: nb}
		cfg.Sink(cfg.FirstLabel+":"+a.Name, d, nb)
		idx[k] = 1
	}

	// 2. Warm-up (untimed, results still checked).
	for k, a := range arms {
		for w := 0; w < cfg.Warmup; w++ {
			nb, err := a.Call(idx[k])
			if err != nil {
				return res, fmt.Errorf("arm %s warm-up: %w", a.Name, err)
			}
			if a.Check != nil {
				if err := a.Check(idx[k], nb); err != nil {
					return res, fmt.Errorf("%w: arm %s warm-up: %v", ErrCheck, a.Name, err)
				}
			}
			idx[k]++
		}
	}
	if cfg.GCEvery > 0 {
		runtime.GC()
		res.GCForced++
	}

	// 3. Interleaved sampling.
	rng := rand.New(rand.NewPCG(cfg.Seed, 0x6b70692d6f726472)) // "kpi-ordr"
	done := make([]bool, len(arms))
	remaining := len(arms)
	sinceGC := 0
	for remaining > 0 {
		for _, k := range rng.Perm(len(arms)) {
			if done[k] {
				continue
			}
			a := arms[k]
			b := batchOf(a)
			i := idx[k]
			var nb int
			var err error
			t0 := time.Now()
			for j := 0; j < b; j++ {
				if nb, err = a.Call(i + j); err != nil {
					break
				}
			}
			d := time.Since(t0).Nanoseconds()
			if err != nil {
				return res, fmt.Errorf("arm %s sample: %w", a.Name, err)
			}
			if a.Check != nil {
				if err := a.Check(i+b-1, nb); err != nil {
					return res, fmt.Errorf("%w: arm %s sample: %v", ErrCheck, a.Name, err)
				}
			}
			idx[k] += b
			r := &res.Arms[k]
			r.PerOpNS = append(r.PerOpNS, float64(d)/float64(b))
			r.SumNS += d
			r.LastBytes = nb
			if nb < r.MinBytes {
				r.MinBytes = nb
			}
			if nb > r.MaxBytes {
				r.MaxBytes = nb
			}
			cfg.Sink(a.Name, d, nb)

			n := len(r.PerOpNS)
			switch cfg.Mode {
			case Fast:
				done[k] = n >= cfg.Samples
			case Slow:
				done[k] = r.SumNS >= cfg.TimeBox.Nanoseconds() && n >= cfg.MinSamples
			}
			if done[k] {
				remaining--
			}
			sinceGC++
			if cfg.GCEvery > 0 && sinceGC >= cfg.GCEvery {
				runtime.GC()
				res.GCForced++
				sinceGC = 0
			}
		}
		if cfg.MaxWall > 0 && remaining > 0 && time.Since(start) > cfg.MaxWall {
			res.Truncated = true
			break
		}
	}
	res.WallNS = time.Since(start).Nanoseconds()
	return res, nil
}

// Probe times the first call and then k further calls of one arm (used by the
// trial's probe pass to classify operations as fast or slow and to estimate
// run time). It returns the first-call time and the k call times.
func Probe(a Arm, k int) (first int64, ns []int64, err error) {
	t0 := time.Now()
	nb, err := a.Call(0)
	first = time.Since(t0).Nanoseconds()
	if err != nil {
		return 0, nil, err
	}
	if a.Check != nil {
		if err := a.Check(0, nb); err != nil {
			return 0, nil, fmt.Errorf("%w: probe: %v", ErrCheck, err)
		}
	}
	for i := 1; i <= k; i++ {
		t0 := time.Now()
		nb, err := a.Call(i)
		d := time.Since(t0).Nanoseconds()
		if err != nil {
			return 0, nil, err
		}
		if a.Check != nil {
			if err := a.Check(i, nb); err != nil {
				return 0, nil, fmt.Errorf("%w: probe: %v", ErrCheck, err)
			}
		}
		ns = append(ns, d)
	}
	return first, ns, nil
}

// AllocsPerOp measures heap bytes and allocations per call of an arm over n
// untimed calls (like testing.AllocsPerRun). Run it after the timed sampling.
func AllocsPerOp(a Arm, n int, startIdx int) (bytesPerOp, allocsPerOp float64, err error) {
	if n < 1 {
		return 0, 0, nil
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < n; i++ {
		if _, err := a.Call(startIdx + i); err != nil {
			return 0, 0, err
		}
	}
	runtime.ReadMemStats(&after)
	return float64(after.TotalAlloc-before.TotalAlloc) / float64(n),
		float64(after.Mallocs-before.Mallocs) / float64(n), nil
}
