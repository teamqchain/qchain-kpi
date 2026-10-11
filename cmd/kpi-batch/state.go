package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"qchain-kpi/internal/plan"
)

// State is the resumable state of a batch (state.json, rewritten atomically
// after every change). A resumed batch continues exactly where it stopped,
// and refuses to continue if anything that defines the measurements changed.
type State struct {
	Schema         int                `json:"schema"`
	RunID          string             `json:"run_id"`
	Machine        string             `json:"machine"`
	PlanName       string             `json:"plan_name"`
	PlanKind       string             `json:"plan_kind"`
	PlanHash       string             `json:"plan_hash"`
	Seed           uint64             `json:"seed"`
	ImageID        string             `json:"image_id"`
	HarnessCommit  string             `json:"harness_commit"`
	ThresholdsHash string             `json:"thresholds_hash"`
	StartedUTC     string             `json:"started_utc"`
	Status         string             `json:"status"` // running, stopped, failed, done
	StatusDetail   string             `json:"status_detail,omitempty"`
	Sessions       []Session          `json:"sessions"`
	Planned        int                `json:"planned_blocks"`
	Queue          []plan.Block       `json:"queue"`
	Done           []string           `json:"done"`
	Class          map[string]string  `json:"classification"` // alg/op -> fast|slow, from this batch's probe pass
	Requeued       map[string]bool    `json:"requeued"`       // attempt key -> its repeat was queued
	Diagnosed      map[string]bool    `json:"diagnosed"`      // block id -> its diagnostic re-run was queued
	CanarySeq      int                `json:"canary_seq"`
	FirstCanary    float64            `json:"first_canary_rate"`
	LastCanaryNS   int64              `json:"last_canary_unix_ns"`
	Preps          map[string]PrepRec `json:"preps"`
	// Extensions: blocks that had no clean attempt were run again in a later
	// session of the same run (see extend.go). Base[id] is the number of
	// non-diagnostic attempts the block had when it was extended; the cap of
	// maxAttempts applies to the attempts made after that point.
	Extensions []Extension    `json:"extensions,omitempty"`
	Base       map[string]int `json:"extension_base,omitempty"`
	// Facts the quality rules need (kept so calibrate can re-run them later).
	CellReps     map[string]int `json:"cell_reps"`
	VMCPUs       int            `json:"vm_cpus"`
	HostCPUs     int            `json:"host_cpus"`
	HostPeriodNS int64          `json:"host_period_ns"`
	CanaryEveryS int            `json:"canary_every_s"`
}

// Extension records one extension of a finished run.
type Extension struct {
	Session        int      `json:"session"`
	Blocks         []string `json:"blocks"`
	OldImageID     string   `json:"old_image_id,omitempty"`
	OldHarnessCmt  string   `json:"old_harness_commit,omitempty"`
	NewImageID     string   `json:"new_image_id,omitempty"`
	NewHarnessCmt  string   `json:"new_harness_commit,omitempty"`
	// OrchestratorChanged lists the cmd/kpi-batch files that differ from the
	// original run's commit (the launcher verified that no measuring source differs).
	OrchestratorChanged []string `json:"orchestrator_files_changed,omitempty"`
}

// PrepRec records one generated input set.
type PrepRec struct {
	Pool        int    `json:"pool"`
	PrepNS      int64  `json:"prep_ns"`
	Fingerprint string `json:"fingerprint"`
	E10         bool   `json:"e10"`
	Session     int    `json:"session"`
}

func loadState(dir string) (*State, error) {
	b, err := os.ReadFile(filepath.Join(dir, fileState))
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", fileState, err)
	}
	if s.Class == nil {
		s.Class = map[string]string{}
	}
	if s.Requeued == nil {
		s.Requeued = map[string]bool{}
	}
	if s.Diagnosed == nil {
		s.Diagnosed = map[string]bool{}
	}
	if s.Preps == nil {
		s.Preps = map[string]PrepRec{}
	}
	return &s, nil
}

func (s *State) save(dir string) error { return writeJSON(filepath.Join(dir, fileState), s) }

// identity is what must not change between sessions of one batch.
type identity struct {
	RunID, Machine, PlanHash, ImageID, HarnessCommit, ThresholdsHash string
	Seed                                                             uint64
}

func (s *State) identity() identity {
	return identity{s.RunID, s.Machine, s.PlanHash, s.ImageID, s.HarnessCommit, s.ThresholdsHash, s.Seed}
}

// resumeProblems lists every difference between the saved batch and the one
// being started (empty = the batch may resume).
func resumeProblems(saved, now identity) []string {
	var p []string
	check := func(name, a, b string) {
		if a != b {
			p = append(p, fmt.Sprintf("%s changed (%s -> %s)", name, a, b))
		}
	}
	check("run id", saved.RunID, now.RunID)
	check("machine", saved.Machine, now.Machine)
	check("plan", saved.PlanHash, now.PlanHash)
	check("image", saved.ImageID, now.ImageID)
	check("harness commit", saved.HarnessCommit, now.HarnessCommit)
	check("thresholds", saved.ThresholdsHash, now.ThresholdsHash)
	if saved.Seed != now.Seed {
		p = append(p, fmt.Sprintf("seed changed (%d -> %d)", saved.Seed, now.Seed))
	}
	return p
}

// strayFiles lists files in a fresh run folder that a new batch must not
// overwrite (the folder must be empty apart from the launcher's deliveries).
func strayFiles(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		switch e.Name() {
		case dirMonitor, sessionFileName, fileStop:
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// cleanTemp removes temporary files an interrupted block may have left.
func cleanTemp(dir string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), ".tmp-") {
			if err := os.Remove(p); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}
