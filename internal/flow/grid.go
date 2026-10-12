package flow

import (
	"qchain-kpi/internal/algs"
)

// CuratedSuites are the 16 curated signature + KEM suites of the research
// plan (section 7). Variants (decided 10 Oct): SLH-DSA entries are SHA2,
// Falcon entries are the standard (unpadded) variant. The first is QChain's
// production suite.
var CuratedSuites = [][2]string{
	{"ml-dsa-44", "ml-kem-768"}, // production
	{"ml-dsa-44", "ml-kem-512"}, {"falcon-512", "ml-kem-512"}, {"slh-dsa-sha2-128f", "ml-kem-512"},
	{"slh-dsa-sha2-128s", "ml-kem-512"}, {"ml-dsa-44", "hqc-1"},
	{"ml-dsa-65", "ml-kem-768"}, {"slh-dsa-sha2-192f", "ml-kem-768"}, {"ml-dsa-65", "hqc-3"},
	{"ml-dsa-87", "ml-kem-1024"}, {"falcon-1024", "ml-kem-1024"}, {"slh-dsa-sha2-256f", "ml-kem-1024"}, {"ml-dsa-87", "hqc-5"},
	{"ecdsa-p256", "dhkem-p256"}, {"ed25519", "dhkem-x25519"}, {"rsa3072-pss", "dhkem-p256"},
}

// Grid settings (research plan section 7, coding plan section 7).
var (
	Ns         = []int{1, 5, 10, 20, 64}
	ValueSizes = []int{16, 256, 4096} // production suite only
)

const (
	DefaultV = 16 // 16-character ASCII values
	PairN    = 10 // the all-pairs sweep runs at N = 10, k = 5
	PairK    = 5
)

// Ks returns the disclosure counts for n attributes: {1, ceil(n/2), n},
// without duplicates.
func Ks(n int) []int {
	out := []int{1}
	for _, k := range []int{(n + 1) / 2, n} {
		if k != out[len(out)-1] {
			out = append(out, k)
		}
	}
	return out
}

// Grid returns every E11 cell, each stage once per setting it depends on:
//   - the 16 curated suites at every N and k, with 16-byte values;
//   - all 176 signature x KEM pairs at N = 10, k = 5;
//   - the production suite at every N and k with 256- and 4096-byte values;
//   - the production suite chained at every N and k (validation of the sums).
func Grid() []Cell {
	var out []Cell
	seen := map[Cell]bool{}
	add := func(c Cell) {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	type setting struct {
		sig, kem string
		n, k, v  int
	}
	var settings []setting
	for _, s := range CuratedSuites {
		for _, n := range Ns {
			for _, k := range Ks(n) {
				settings = append(settings, setting{s[0], s[1], n, k, DefaultV})
			}
		}
	}
	for _, sig := range byTrack(algs.Sig) {
		for _, kem := range byTrack(algs.KEM) {
			settings = append(settings, setting{sig.ID, kem.ID, PairN, PairK, DefaultV})
		}
	}
	prod := CuratedSuites[0]
	for _, v := range ValueSizes[1:] {
		for _, n := range Ns {
			for _, k := range Ks(n) {
				settings = append(settings, setting{prod[0], prod[1], n, k, v})
			}
		}
	}
	// Stage by stage, so the plan expands in a readable order.
	for _, s := range settings {
		add(Cell{Stage: Activation, Sig: s.sig, KEM: s.kem})
	}
	for _, s := range settings {
		add(Cell{Stage: Issuance, Sig: s.sig, KEM: s.kem, N: s.n, V: s.v})
	}
	for _, s := range settings {
		add(Cell{Stage: Retrieval, KEM: s.kem, N: s.n, V: s.v})
	}
	for _, s := range settings {
		add(Cell{Stage: Presentation, Sig: s.sig, K: s.k, V: s.v})
	}
	for _, s := range settings {
		add(Cell{Stage: Verification, Sig: s.sig, KEM: s.kem, N: s.n, K: s.k, V: s.v})
	}
	for _, n := range Ns {
		for _, k := range Ks(n) {
			add(Cell{Stage: Chained, Sig: prod[0], KEM: prod[1], N: n, K: k, V: DefaultV})
		}
	}
	return out
}

func byTrack(t algs.Track) []algs.Alg {
	var out []algs.Alg
	for _, a := range algs.All() {
		if a.Track == t {
			out = append(out, a)
		}
	}
	return out
}
