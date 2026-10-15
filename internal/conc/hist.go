// Package conc runs E12's closed-loop concurrency measurement: W workers each
// call their own copy of an operation back to back for a warm-up and a fixed
// window, and every call completed in the window is counted (throughput) and
// its latency recorded (a log-linear histogram, so millions of calls need no
// per-call storage).
package conc

import (
	"math"
	"math/bits"
)

// subBits sets the histogram's resolution: 2^subBits buckets per power of
// two, so every bucket is at most 1/64 (1.6 %) of its lower bound wide.
// Values below 64 ns have one bucket each.
const subBits = 6

// NBuckets covers every int64 latency.
const NBuckets = (64 - subBits + 1) << subBits

// BucketOf returns the bucket of a latency in nanoseconds (negative values
// count as 0).
func BucketOf(ns int64) int {
	if ns < 1<<subBits {
		if ns < 0 {
			return 0
		}
		return int(ns)
	}
	p := bits.Len64(uint64(ns)) - 1 // >= subBits
	shift := p - subBits
	return (shift+1)<<subBits + int((uint64(ns)>>shift)&(1<<subBits-1))
}

// Lower returns the smallest latency in bucket b.
func Lower(b int) int64 {
	if b < 1<<subBits {
		return int64(b)
	}
	shift := b>>subBits - 1
	return (1<<subBits + int64(b&(1<<subBits-1))) << shift
}

// Width returns the number of distinct nanosecond values in bucket b.
func Width(b int) int64 {
	if b < 2<<subBits {
		return 1
	}
	return 1 << (b>>subBits - 1)
}

// Hist is a latency histogram with exact count, sum, minimum and maximum.
type Hist struct {
	Counts   [NBuckets]int64
	N        int64
	Sum      int64
	Min, Max int64
}

// Add records one latency.
func (h *Hist) Add(ns int64) {
	h.Counts[BucketOf(ns)]++
	if h.N == 0 || ns < h.Min {
		h.Min = ns
	}
	if ns > h.Max {
		h.Max = ns
	}
	h.N++
	h.Sum += ns
}

// Merge adds o's counts to h.
func (h *Hist) Merge(o *Hist) {
	if o.N == 0 {
		return
	}
	for b, c := range o.Counts {
		h.Counts[b] += c
	}
	if h.N == 0 || o.Min < h.Min {
		h.Min = o.Min
	}
	if o.Max > h.Max {
		h.Max = o.Max
	}
	h.N += o.N
	h.Sum += o.Sum
}

// Quantile returns the q-quantile (0 < q <= 1): the midpoint of the bucket
// holding the call of rank ceil(q*N), clamped to the exact minimum and
// maximum. Its error is at most half a bucket (0.8 %).
func (h *Hist) Quantile(q float64) float64 {
	if h.N == 0 {
		return math.NaN()
	}
	rank := int64(math.Ceil(q * float64(h.N)))
	rank = min(max(rank, 1), h.N)
	var seen int64
	for b, c := range h.Counts {
		seen += c
		if seen >= rank {
			mid := float64(Lower(b)) + float64(Width(b)-1)/2
			return math.Min(math.Max(mid, float64(h.Min)), float64(h.Max))
		}
	}
	return float64(h.Max)
}

// Mean is the exact mean latency.
func (h *Hist) Mean() float64 {
	if h.N == 0 {
		return math.NaN()
	}
	return float64(h.Sum) / float64(h.N)
}

// SD is the standard deviation with every call at its bucket's midpoint (an
// approximation within the buckets' 1.6 % resolution); 0 for fewer than 2 calls.
func (h *Hist) SD() float64 {
	if h.N < 2 {
		return 0
	}
	m := h.Mean()
	var ss float64
	for b, c := range h.Counts {
		if c == 0 {
			continue
		}
		mid := float64(Lower(b)) + float64(Width(b)-1)/2
		ss += float64(c) * (mid - m) * (mid - m)
	}
	return math.Sqrt(ss / float64(h.N-1))
}
