package main

// Run plans for E14 (fabric/plans/*.json) and their expansion into blocks of
// work. A block is one (point, mode, repetition): a number of transactions of
// one payload size, either one after the other (sequential) or all at once
// (burst). Drift-control blocks (the smallest transaction, sequential) are
// placed at the start, the middle and the end of the run.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"sort"
)

// Point is one payload size: the envelope size E13 computed for QChain's
// issueCredential under one signature scheme (N = 5 attributes of 16 bytes,
// ECDSA Fabric identities), or the smallest transaction (control).
type Point struct {
	Name           string `json:"name"`            // "tiny", "ecdsa-p256", ..., "slh-dsa-sha2-256f"
	TargetEnvelope int    `json:"target_envelope"` // bytes; 0 = the smallest transaction (1-byte payload)
	E13PerBlock    int    `json:"e13_per_block"`   // transactions per block E13 predicts (0: not applicable)
}

// Plan is a run plan file.
type Plan struct {
	Name        string  `json:"name"`
	Kind        string  `json:"kind"` // smoke or main
	Description string  `json:"description"`
	Points      []Point `json:"points"`
	Reps        int     `json:"reps"`
	Sequential  int     `json:"sequential_tx"` // transactions per sequential block
	Burst       int     `json:"burst_tx"`      // transactions per burst block
	Controls    int     `json:"control_tx"`    // transactions per drift-control block (3 blocks)
	WarmupRun   int     `json:"warmup_run_tx"` // sequential smallest transactions before the first block
	WarmupBlock int     `json:"warmup_block_tx"`
	CanaryEvery int     `json:"canary_every_s"`
	Hash        string  `json:"-"`
}

// Block is one block of work.
type Block struct {
	ID    string `json:"id"` // E14.<point>.<mode>.r<rep>, or E14.tiny.drift.d<n>
	Point Point  `json:"point"`
	Mode  string `json:"mode"` // sequential, burst or drift
	Rep   int    `json:"rep"`
	N     int    `json:"n"` // transactions
	Seq   int    `json:"seq"`
}

const (
	ModeSequential = "sequential"
	ModeBurst      = "burst"
	ModeDrift      = "drift"
)

var pointName = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)

// LoadPlan reads and checks a plan file.
func LoadPlan(path string) (*Plan, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("plan %s: %w", path, err)
	}
	sum := sha256.Sum256(raw)
	p.Hash = fmt.Sprintf("%x", sum)
	if err := p.check(); err != nil {
		return nil, fmt.Errorf("plan %s: %w", path, err)
	}
	return &p, nil
}

func (p *Plan) check() error {
	if p.Name == "" || (p.Kind != "smoke" && p.Kind != "main") {
		return fmt.Errorf("name and kind (smoke or main) are required")
	}
	if len(p.Points) == 0 || p.Reps < 1 || p.Sequential < 1 || p.Burst < 1 || p.Controls < 1 {
		return fmt.Errorf("points, reps, sequential_tx, burst_tx and control_tx must be positive")
	}
	seen := map[string]bool{}
	tiny := false
	for _, pt := range p.Points {
		if !pointName.MatchString(pt.Name) || seen[pt.Name] {
			return fmt.Errorf("bad or repeated point name %q", pt.Name)
		}
		seen[pt.Name] = true
		if pt.Name == "tiny" {
			tiny = true
			if pt.TargetEnvelope != 0 {
				return fmt.Errorf("the tiny point is the smallest transaction (target_envelope 0)")
			}
		} else if pt.TargetEnvelope < 1000 {
			return fmt.Errorf("point %s: target_envelope %d is too small", pt.Name, pt.TargetEnvelope)
		}
	}
	if !tiny {
		return fmt.Errorf("the plan needs the tiny point (drift controls use it)")
	}
	return nil
}

// Expand returns the plan's blocks in their run order: drift control 1, the
// shuffled work blocks with drift control 2 in the middle, drift control 3.
// The order depends only on the plan and the seed.
func (p *Plan) Expand(seed uint64) []Block {
	var work []Block
	for rep := 1; rep <= p.Reps; rep++ {
		for _, pt := range p.Points {
			work = append(work,
				Block{ID: fmt.Sprintf("E14.%s.%s.r%d", pt.Name, ModeSequential, rep), Point: pt, Mode: ModeSequential, Rep: rep, N: p.Sequential},
				Block{ID: fmt.Sprintf("E14.%s.%s.r%d", pt.Name, ModeBurst, rep), Point: pt, Mode: ModeBurst, Rep: rep, N: p.Burst})
		}
	}
	sort.Slice(work, func(i, j int) bool { return work[i].ID < work[j].ID })
	r := rand.New(rand.NewChaCha8(seedBytes(seed, "order")))
	r.Shuffle(len(work), func(i, j int) { work[i], work[j] = work[j], work[i] })
	var tiny Point
	for _, pt := range p.Points {
		if pt.Name == "tiny" {
			tiny = pt
		}
	}
	drift := func(n int) Block {
		return Block{ID: fmt.Sprintf("E14.tiny.%s.d%d", ModeDrift, n), Point: tiny, Mode: ModeDrift, Rep: n, N: p.Controls}
	}
	out := []Block{drift(1)}
	out = append(out, work[:len(work)/2]...)
	out = append(out, drift(2))
	out = append(out, work[len(work)/2:]...)
	out = append(out, drift(3))
	for i := range out {
		out[i].Seq = i + 1
	}
	return out
}

// seedBytes derives a 32-byte ChaCha8 seed from the run seed and a label.
func seedBytes(seed uint64, label string) [32]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], seed)
	return sha256.Sum256(append(b[:], []byte(label)...))
}
