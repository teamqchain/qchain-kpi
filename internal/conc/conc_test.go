package conc

import (
	"errors"
	"math"
	"math/rand/v2"
	"sort"
	"testing"
	"time"
)

// TestBuckets: every value lies in its bucket, buckets tile the line without
// gaps, and no bucket is wider than 1/64 of its lower bound.
func TestBuckets(t *testing.T) {
	for b := 0; b < BucketOf(math.MaxInt64); b++ {
		if Lower(b)+Width(b) != Lower(b+1) {
			t.Fatalf("bucket %d [%d, +%d) does not meet bucket %d at %d", b, Lower(b), Width(b), b+1, Lower(b+1))
		}
		if b >= 64 && float64(Width(b)) > float64(Lower(b))/64 {
			t.Fatalf("bucket %d too wide", b)
		}
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 100000; i++ {
		v := rng.Int64N(1 << uint(rng.IntN(62)+1))
		b := BucketOf(v)
		if v < Lower(b) || v >= Lower(b)+Width(b) {
			t.Fatalf("%d in bucket %d [%d, +%d)", v, b, Lower(b), Width(b))
		}
	}
	if BucketOf(math.MaxInt64) >= NBuckets {
		t.Fatal("NBuckets too small")
	}
}

// TestQuantiles: within half a bucket of the exact order statistic.
func TestQuantiles(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	var h Hist
	var xs []int64
	for i := 0; i < 50000; i++ {
		v := int64(20000 + rng.ExpFloat64()*50000)
		h.Add(v)
		xs = append(xs, v)
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	for _, q := range []float64{0.5, 0.9, 0.99, 0.999} {
		exact := float64(xs[int(math.Ceil(q*float64(len(xs))))-1])
		if got := h.Quantile(q); math.Abs(got-exact)/exact > 0.008 {
			t.Fatalf("q%.3f = %.0f, exact %.0f", q, got, exact)
		}
	}
	if h.N != 50000 || h.Min != xs[0] || h.Max != xs[len(xs)-1] {
		t.Fatalf("n %d min %d max %d", h.N, h.Min, h.Max)
	}
	var a, b Hist
	for i, x := range xs {
		if i%2 == 0 {
			a.Add(x)
		} else {
			b.Add(x)
		}
	}
	a.Merge(&b)
	if a != h {
		t.Fatal("merged halves differ from the whole")
	}
}

func sleeper(d time.Duration) Worker {
	return Worker{Call: func(int) (int, error) { time.Sleep(d); return 0, nil }}
}

// TestRunScales: every call in the window is counted once, the per-second
// series adds up, and the run obeys Little's law for a closed loop
// (throughput x mean latency = W, whatever each sleep actually takes), with
// the workers sharing the work evenly.
func TestRunScales(t *testing.T) {
	cfg := Config{Warmup: 100 * time.Millisecond, Window: 1500 * time.Millisecond, CheckEvery: 4}
	for _, w := range []int{1, 4} {
		var ws []Worker
		checks := make([]int, w)
		for k := 0; k < w; k++ {
			x := sleeper(2 * time.Millisecond)
			k := k
			x.Check = func(int, int) error { checks[k]++; return nil }
			ws = append(ws, x)
		}
		r, err := Run(ws, cfg)
		if err != nil {
			t.Fatal(err)
		}
		var sum, per int64
		for _, c := range r.PerSecond {
			sum += c
		}
		for _, c := range r.PerWorker {
			per += c
		}
		if sum != r.Hist.N || per != r.Hist.N || len(r.PerSecond) != 2 || r.WarmupCalls == 0 {
			t.Fatalf("w=%d: n %d, per-second %v, per-worker %v, warm-up %d", w, r.Hist.N, r.PerSecond, r.PerWorker, r.WarmupCalls)
		}
		var nChk int
		for _, c := range checks {
			nChk += c
		}
		if int64(nChk) != r.Checked || r.Checked < r.WarmupCalls+r.Hist.N/4-int64(w) {
			t.Fatalf("w=%d: %d checks counted, %d reported, warm-up %d, window %d", w, nChk, r.Checked, r.WarmupCalls, r.Hist.N)
		}
		if r.Hist.Quantile(0.5) < 2e6 {
			t.Fatalf("w=%d: median %.0f ns below the 2 ms sleep", w, r.Hist.Quantile(0.5))
		}
		// A call completed in the window started at most one call earlier,
		// so the law holds to within one call per worker.
		if l := r.Throughput() * r.Hist.Mean() / 1e9; math.Abs(l-float64(w)) > 0.05*float64(w) {
			t.Fatalf("w=%d: throughput %.1f/s x mean latency %.0f ns = %.3f, want %d", w, r.Throughput(), r.Hist.Mean(), l, w)
		}
		lo, hi := r.PerWorker[0], r.PerWorker[0]
		for _, n := range r.PerWorker {
			lo, hi = min(lo, n), max(hi, n)
		}
		if float64(hi-lo) > 0.25*float64(hi) {
			t.Fatalf("w=%d: uneven workers %v", w, r.PerWorker)
		}
	}
}

// TestRunStopsOnError: the first error stops every worker and is returned.
func TestRunStopsOnError(t *testing.T) {
	bad := errors.New("broken")
	ws := []Worker{sleeper(time.Millisecond), {Call: func(i int) (int, error) {
		if i == 10 {
			return 0, bad
		}
		time.Sleep(time.Millisecond)
		return 0, nil
	}}}
	t0 := time.Now()
	_, err := Run(ws, Config{Warmup: 0, Window: 10 * time.Second, CheckEvery: 1})
	if !errors.Is(err, bad) || time.Since(t0) > 2*time.Second {
		t.Fatalf("err %v after %v", err, time.Since(t0))
	}
}
