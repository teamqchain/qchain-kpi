// Package canary is the fixed micro-benchmark run at the start, every 10
// minutes and at the end of every batch, on every machine. Its speed should
// stay constant; a change means the machine got hotter, a neighbouring VM got
// busier, or (on the Mac) the work moved to an efficiency core.
//
// It is pure Go (SHA3-256 from golang.org/x/crypto, the same package QChain's
// backend uses), so it does not depend on the liboqs build variant and can
// also run natively on macOS as the performance/efficiency-core reference.
package canary

import (
	"sort"
	"time"

	"golang.org/x/crypto/sha3"
)

// Result of one canary run.
type Result struct {
	WindowMS   int       `json:"window_ms"`
	Windows    int       `json:"windows"`
	RatesPerS  []float64 `json:"rates_per_s"`   // SHA3-256 hashes per second in each window
	MedianRate float64   `json:"median_rate_per_s"`
	Digest     string    `json:"digest_check"` // proves the work was done
}

// message is a fixed 64-byte input (the size QChain signs: a 64-char hex string).
var message = []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")

// Run executes windows x window of SHA3-256 hashing and reports the median
// rate. The default canary is 15 windows of 100 ms (1.5 s).
func Run(windows int, window time.Duration) Result {
	if windows < 1 {
		windows = 15
	}
	if window <= 0 {
		window = 100 * time.Millisecond
	}
	rates := make([]float64, 0, windows)
	var acc [32]byte
	in := make([]byte, len(message))
	copy(in, message)
	for w := 0; w < windows; w++ {
		n := 0
		start := time.Now()
		deadline := start.Add(window)
		for {
			// 64 hashes between clock reads keep timer overhead negligible.
			for j := 0; j < 64; j++ {
				acc = sha3.Sum256(in)
				in[0] ^= acc[0]
			}
			n += 64
			if !time.Now().Before(deadline) {
				break
			}
		}
		el := time.Since(start).Seconds()
		rates = append(rates, float64(n)/el)
	}
	sorted := append([]float64(nil), rates...)
	sort.Float64s(sorted)
	med := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		med = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	const hexdigits = "0123456789abcdef"
	d := make([]byte, 8)
	for i := 0; i < 4; i++ {
		d[2*i] = hexdigits[acc[i]>>4]
		d[2*i+1] = hexdigits[acc[i]&15]
	}
	return Result{WindowMS: int(window / time.Millisecond), Windows: windows, RatesPerS: rates, MedianRate: med, Digest: string(d)}
}
