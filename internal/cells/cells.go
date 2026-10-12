// Package cells defines E11's measurement cells and grid. It is plain Go
// (no cgo), so the orchestrator can expand plans without linking liboqs; the
// measuring code is in internal/flow.
//
// A Cell is one stage at one setting. Each stage depends on a subset of the
// settings, and is measured once per value of those (decision of 10 Oct, and
// decision D of 12 Oct):
//
//	activation   (signature, KEM)           wallet key generation + backend key checks
//	issuance     (signature, KEM, N, V)     the issuance handler's computation
//	retrieval    (KEM, N, V)                backend envelope check + wallet decryption
//	presentation (signature, k, V)          wallet payload + signature + backend request check
//	verification (signature, KEM, N, k, V)  chain-record decoding + verifyPresentation
//	chained      (signature, KEM, N, k, V)  the five stages back to back (validation of the sums)
//
// N = attributes per credential, k = disclosed attributes, V = bytes per
// attribute value. Verification depends on the KEM because the holder record
// the chain returns carries the holder's KEM public key.
package cells

import (
	"fmt"
	"strconv"
	"strings"

	"qchain-kpi/internal/algs"
)

// Stages.
const (
	Activation   = "activation"
	Issuance     = "issuance"
	Retrieval    = "retrieval"
	Presentation = "presentation"
	Verification = "verification"
	Chained      = "chained"
)

// Arm names of an E11 block.
const (
	ArmStage       = "stage"        // the whole stage
	ArmChainDecode = "chain-decode" // verification only: decoding the chain record, reported separately
)

// Cell is one stage at one setting. Sig or KEM is "" when the stage does not
// depend on it; N, K and V are 0 when it does not depend on them.
type Cell struct {
	Stage    string
	Sig, KEM string
	N, K, V  int
}

// Alg is the block's algorithm field: "sig+kem", or the one algorithm the
// stage depends on.
func (c Cell) Alg() string {
	switch {
	case c.Sig != "" && c.KEM != "":
		return c.Sig + "+" + c.KEM
	case c.Sig != "":
		return c.Sig
	}
	return c.KEM
}

// Op is the block's operation field: the stage plus the settings it depends on.
func (c Cell) Op() string {
	s := c.Stage
	if c.N > 0 {
		s += "-n" + strconv.Itoa(c.N)
	}
	if c.K > 0 {
		s += "-k" + strconv.Itoa(c.K)
	}
	if c.V > 0 {
		s += "-v" + strconv.Itoa(c.V)
	}
	return s
}

// ParseCell is the inverse of Alg and Op.
func ParseCell(alg, op string) (Cell, error) {
	var c Cell
	parts := strings.Split(op, "-")
	c.Stage = parts[0]
	for _, p := range parts[1:] {
		if len(p) < 2 {
			return c, fmt.Errorf("flow: bad operation %q", op)
		}
		n, err := strconv.Atoi(p[1:])
		if err != nil || n < 1 {
			return c, fmt.Errorf("flow: bad operation %q", op)
		}
		switch p[0] {
		case 'n':
			c.N = n
		case 'k':
			c.K = n
		case 'v':
			c.V = n
		default:
			return c, fmt.Errorf("flow: bad operation %q", op)
		}
	}
	sig, kem, two := strings.Cut(alg, "+")
	switch {
	case two:
		c.Sig, c.KEM = sig, kem
	default:
		a, ok := algs.ByID(alg)
		if !ok {
			return c, fmt.Errorf("flow: unknown algorithm %q", alg)
		}
		if a.Track == algs.Sig {
			c.Sig = alg
		} else {
			c.KEM = alg
		}
	}
	return c, c.Validate()
}

// Validate checks that the cell names exactly the settings its stage depends on.
func (c Cell) Validate() error {
	need := map[string][5]bool{ // sig, kem, n, k, v
		Activation:   {true, true, false, false, false},
		Issuance:     {true, true, true, false, true},
		Retrieval:    {false, true, true, false, true},
		Presentation: {true, false, false, true, true},
		Verification: {true, true, true, true, true},
		Chained:      {true, true, true, true, true},
	}
	w, ok := need[c.Stage]
	if !ok {
		return fmt.Errorf("flow: unknown stage %q", c.Stage)
	}
	have := [5]bool{c.Sig != "", c.KEM != "", c.N > 0, c.K > 0, c.V > 0}
	if have != w {
		return fmt.Errorf("flow: %s needs (sig, kem, N, k, V) = %v, got %v", c.Stage, w, have)
	}
	if c.Sig != "" {
		if a, ok := algs.ByID(c.Sig); !ok || a.Track != algs.Sig {
			return fmt.Errorf("flow: %q is not a signature algorithm", c.Sig)
		}
	}
	if c.KEM != "" {
		if a, ok := algs.ByID(c.KEM); !ok || a.Track != algs.KEM {
			return fmt.Errorf("flow: %q is not a KEM", c.KEM)
		}
	}
	if c.N > 64 || (c.K > 0 && c.N > 0 && c.K > c.N) || c.K > 64 {
		return fmt.Errorf("flow: N=%d k=%d out of range (N <= 64, k <= N)", c.N, c.K)
	}
	return nil
}

// PoolSize is the number of distinct inputs a block cycles through. Inputs
// whose preparation needs signatures (verification) use fewer for the slow
// SLH-DSA "s" sets.
func PoolSize(c Cell) int {
	if c.Sig != "" && (c.Stage == Verification || c.Stage == Chained) {
		if a := algs.MustByID(c.Sig); a.Family == "SLH-DSA" && strings.HasSuffix(a.ID, "s") {
			return 8
		}
	}
	return 32
}

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

// SlowCompositeNS is the classification rule for composites (approved 10 Oct,
// coding plan section 13 item 3): a cell whose probe median is at least 30 ms
// follows the slow-operation rule (60 s and at least 30 samples); every other
// cell takes 20 warm-up calls and 2,000 samples.
const SlowCompositeNS = 30e6
