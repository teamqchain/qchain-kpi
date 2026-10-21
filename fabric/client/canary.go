// Canary: a verbatim copy of the harness's internal/canary (the client is a
// separate Go module and cannot import it), so E14's canary is the same
// workload as every other run's: SHA3-256 of a 64-byte message, 15 windows
// of 100 ms, the median rate. Only the package name and the type and
// function names (Canary, RunCanary) differ.
package main

import (
	"sort"
	"time"

	"golang.org/x/crypto/sha3"
)

// Canary is the result of one canary run.
type Canary struct {
	WindowMS   int       `json:"window_ms"`
	Windows    int       `json:"windows"`
	RatesPerS  []float64 `json:"rates_per_s"`   // SHA3-256 hashes per second in each window
	MedianRate float64   `json:"median_rate_per_s"`
	Digest     string    `json:"digest_check"` // proves the work was done
}

// message is a fixed 64-byte input (the size QChain signs: a 64-char hex string).
var message = []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")

// RunCanary executes windows x window of SHA3-256 hashing and reports the median
// rate. The default canary is 15 windows of 100 ms (1.5 s).
func RunCanary(windows int, window time.Duration) Canary {
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
	return Canary{WindowMS: int(window / time.Millisecond), Windows: windows, RatesPerS: rates, MedianRate: med, Digest: string(d)}
}
