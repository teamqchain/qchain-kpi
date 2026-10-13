// Package plan reads run plans (plans/*.json) and expands them into blocks.
//
// A block is one fresh process measuring one (experiment, algorithm,
// operation or arm set, repetition, attempt). Repetitions are rep-major: rep 1
// of every cell in a seeded shuffled order, then rep 2 reshuffled, and so on.
// Every seed derives from (batch seed, block identity), so a resumed batch
// regenerates exactly the same order and inputs.
package plan

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/cells"
)

// Kinds of plan.
const (
	KindSmoke = "smoke" // pipeline test, tiny budgets, quiet-machine checks not enforced
	KindTrial = "trial" // calibration run (probe pass, A/A, canary)
	KindMain  = "main"  // measurement run with calibrated thresholds
)

// Profiles of sampling budget.
const (
	ProfileApproved = "approved" // the study's approved budgets (timing.FastConfig / SlowConfig)
	ProfileSmoke    = "smoke"    // tiny budgets for pipeline tests only
)

// Item is one entry of a plan.
type Item struct {
	Exp     string   `json:"exp"`               // "probe", "probe-E2", "probe-E11", "probe-E6", "probe-E7", "E1", "E10", "E2", "E2c", "E8", "E11", "E13", "E6", "E7"
	Algs    []string `json:"algs"`              // selectors (see algs.Expand)
	Ops     []string `json:"ops,omitempty"`     // default: every op of the algorithm
	Reps    int      `json:"reps,omitempty"`    // default: Plan.Reps
	Profile string   `json:"profile,omitempty"` // default: approved
}

// Plan is a run plan.
type Plan struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Description  string `json:"description"`
	Reps         int    `json:"reps"`
	CanaryEveryS int    `json:"canary_every_s"`
	Items        []Item `json:"items"`
	Hash         string `json:"-"`
}

// Load reads and validates a plan file.
func Load(path string) (*Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("plan %s: %w", path, err)
	}
	sum := sha256.Sum256(b)
	p.Hash = hex.EncodeToString(sum[:])
	switch p.Kind {
	case KindSmoke, KindTrial, KindMain:
	default:
		return nil, fmt.Errorf("plan %s: unknown kind %q", path, p.Kind)
	}
	if p.Reps < 1 {
		return nil, fmt.Errorf("plan %s: reps must be >= 1", path)
	}
	if p.CanaryEveryS <= 0 {
		p.CanaryEveryS = 600
	}
	for i, it := range p.Items {
		switch it.Exp {
		case "probe", "probe-E2", "probe-E11", "probe-E6", "probe-E7", "E1", "E10", "E2", "E2c", "E8", "E11", "E13", "E6", "E7":
		default:
			return nil, fmt.Errorf("plan %s item %d: experiment %q not implemented in this phase", path, i, it.Exp)
		}
		if _, err := algs.Expand(it.Algs); err != nil {
			return nil, fmt.Errorf("plan %s item %d: %w", path, i, err)
		}
		if it.Profile == "" {
			p.Items[i].Profile = ProfileApproved
		} else if it.Profile != ProfileApproved && it.Profile != ProfileSmoke {
			return nil, fmt.Errorf("plan %s item %d: unknown profile %q", path, i, it.Profile)
		}
		if it.Exp == "E10" {
			for _, a := range mustExpand(it.Algs) {
				if a.Family != "ML-DSA" {
					return nil, fmt.Errorf("plan %s item %d: E10 is defined for ML-DSA only, got %s", path, i, a.ID)
				}
			}
		}
	}
	return &p, nil
}

func mustExpand(s []string) []algs.Alg {
	a, err := algs.Expand(s)
	if err != nil {
		panic(err)
	}
	return a
}

// Block is one unit of work. It is passed to kpi-prim as JSON.
type Block struct {
	ID      string `json:"id"` // identity without attempt, e.g. "E1.ml-dsa-44.sign.r3"
	Exp     string `json:"exp"`
	Alg     string `json:"alg"`
	Op      string `json:"op"` // keygen|sign|verify|encaps|decaps (E1, E2, E2c, probes); "sign" (E10)
	Rep     int    `json:"rep"`
	Attempt int    `json:"attempt"`
	Profile string `json:"profile"`
	Mode    string `json:"mode,omitempty"` // fast|slow (frozen classification); "" = unknown
	Seed    uint64 `json:"seed"`
	// Diagnostic marks a re-run triggered only by a drift flag (it never
	// replaces the original attempt in the headline).
	Diagnostic bool `json:"diagnostic,omitempty"`
}

// Key is the attempt-qualified identity, e.g. "E1.ml-dsa-44.sign.r3.a1".
func (b Block) Key() string { return fmt.Sprintf("%s.a%d", b.ID, b.Attempt) }

// Cell is the identity without the repetition, e.g. "E1.ml-dsa-44.sign".
func (b Block) Cell() string { return fmt.Sprintf("%s.%s.%s", b.Exp, b.Alg, b.Op) }

// RawPath is the block's raw-data file, relative to the run folder.
func (b Block) RawPath() string {
	return fmt.Sprintf("raw/%s/%s/%s-r%d-a%d.csv.gz", b.Exp, b.Alg, b.Op, b.Rep, b.Attempt)
}

// DeriveSeed derives a 64-bit seed from the batch seed and an identity string.
func DeriveSeed(batchSeed uint64, id string) uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], batchSeed)
	h := sha256.Sum256(append(buf[:], []byte(id)...))
	return binary.LittleEndian.Uint64(h[:8])
}

// Expand turns the plan into the ordered list of first attempts.
func Expand(p *Plan, batchSeed uint64) ([]Block, error) {
	type cell struct {
		exp, alg, op, profile string
		reps                  int
	}
	var cs []cell
	maxReps := 0
	for _, it := range p.Items {
		reps := it.Reps
		if reps == 0 {
			reps = p.Reps
		}
		if IsProbe(it.Exp) {
			reps = 1
		}
		if reps > maxReps {
			maxReps = reps
		}
		if it.Exp == "E11" || it.Exp == "probe-E11" {
			// E11's grid is fixed (internal/cells); the item's algorithms
			// select the cells whose signature and KEM are all among them.
			sel := map[string]bool{}
			for _, a := range mustExpand(it.Algs) {
				sel[a.ID] = true
			}
			for _, c := range cells.Grid() {
				if (c.Sig == "" || sel[c.Sig]) && (c.KEM == "" || sel[c.KEM]) {
					cs = append(cs, cell{it.Exp, c.Alg(), c.Op(), it.Profile, reps})
				}
			}
			continue
		}
		for _, a := range mustExpand(it.Algs) {
			ops := it.Ops
			if len(ops) == 0 {
				ops = a.Ops()
			}
			if it.Exp == "E10" {
				ops = []string{"sign"}
			}
			if it.Exp == "E8" {
				if len(it.Ops) == 0 {
					ops = a.SizeOps()
				}
				for _, op := range ops {
					if op != "keygen" && op != "sign" && op != "encaps" {
						return nil, fmt.Errorf("E8 measures keygen, sign and encaps only, not %q", op)
					}
				}
			}
			if it.Exp == "E6" || it.Exp == "probe-E6" {
				// Signatures only: sign and verify at every message size.
				if a.Track != algs.Sig {
					return nil, fmt.Errorf("E6 is defined for signature schemes, not %s", a.ID)
				}
				base := it.Ops
				if len(base) == 0 {
					base = []string{"sign", "verify"}
				}
				for _, op := range base {
					if op != "sign" && op != "verify" {
						return nil, fmt.Errorf("E6 measures sign and verify, not %q", op)
					}
					for _, size := range E6Sizes {
						cs = append(cs, cell{it.Exp, a.ID, fmt.Sprintf("%s-m%d", op, size), it.Profile, reps})
					}
				}
				continue
			}
			if it.Exp == "E7" || it.Exp == "probe-E7" {
				// One block per algorithm: verify (signatures) or decaps (KEMs).
				op := "verify"
				if a.Track == algs.KEM {
					op = "decaps"
				}
				if len(it.Ops) > 0 && (len(it.Ops) != 1 || it.Ops[0] != op) {
					return nil, fmt.Errorf("E7's operation for %s is %q", a.ID, op)
				}
				cs = append(cs, cell{it.Exp, a.ID, op, it.Profile, reps})
				continue
			}
			if it.Exp == "E13" {
				// One size block per algorithm: Fabric transactions for a
				// signature scheme, IPFS envelopes for a KEM.
				op := "ledger"
				if a.Track == algs.KEM {
					op = "ipfs"
				}
				if len(it.Ops) > 0 && (len(it.Ops) != 1 || it.Ops[0] != op) {
					return nil, fmt.Errorf("E13's operation for %s is %q", a.ID, op)
				}
				cs = append(cs, cell{it.Exp, a.ID, op, it.Profile, reps})
				continue
			}
			for _, op := range ops {
				if !validOp(a, op) {
					return nil, fmt.Errorf("%s does not support op %q", a.ID, op)
				}
				cs = append(cs, cell{it.Exp, a.ID, op, it.Profile, reps})
			}
		}
	}
	var out []Block
	seen := map[string]bool{}
	for rep := 1; rep <= maxReps; rep++ {
		var round []Block
		for _, c := range cs {
			if rep > c.reps {
				continue
			}
			id := fmt.Sprintf("%s.%s.%s.r%d", c.exp, c.alg, c.op, rep)
			if seen[id] {
				continue // the same cell listed twice in a plan
			}
			seen[id] = true
			round = append(round, Block{ID: id, Exp: c.exp, Alg: c.alg, Op: c.op, Rep: rep, Attempt: 1,
				Profile: c.profile, Seed: DeriveSeed(batchSeed, id)})
		}
		// Probe blocks always come first (they feed the classification of
		// the blocks that follow); everything else is shuffled per rep.
		var probes, rest []Block
		for _, b := range round {
			if IsProbe(b.Exp) {
				probes = append(probes, b)
			} else {
				rest = append(rest, b)
			}
		}
		rng := rand.New(rand.NewPCG(DeriveSeed(batchSeed, fmt.Sprintf("order.r%d", rep)), 0x6b70692d70726d))
		rng.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
		out = append(out, probes...)
		out = append(out, rest...)
	}
	return out, nil
}

// E6Sizes are E6's message sizes in bytes (internal/proto names them; the
// plan needs them to expand E6's cells).
var E6Sizes = []int{64, 1024, 10 * 1024, 100 * 1024, 1024 * 1024}

// IsProbe reports whether an experiment is a probe pass (calibration only).
func IsProbe(exp string) bool {
	return exp == "probe" || exp == "probe-E2" || exp == "probe-E11" || exp == "probe-E6" || exp == "probe-E7"
}

func validOp(a algs.Alg, op string) bool {
	for _, o := range a.Ops() {
		if o == op {
			return true
		}
	}
	return false
}
