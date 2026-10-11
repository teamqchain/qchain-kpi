package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/record"
)

// Manifest is manifest.json: what a run is, built only from allow-listed
// facts (coding plan §10). The environment itself is in env/e0-s<N>.json.
type Manifest struct {
	Schema        int               `json:"schema"`
	RunID         string            `json:"run_id"`
	Machine       string            `json:"machine"`
	Plan          ManifestPlan      `json:"plan"`
	Seed          uint64            `json:"seed"`
	ImageID       string            `json:"image_id"`
	HarnessCommit string            `json:"harness_commit"`
	Go            string            `json:"go"`
	Build         map[string]string `json:"build"`       // oqs-build.txt of the image (variant, liboqs commit, arch, gcc, cmake)
	AptSources    map[string]string `json:"apt_sources"` // snapshot or live-mirror fallback, per image stage
	Thresholds    ManifestTH        `json:"thresholds"`
	Container     map[string]string `json:"container"` // preflight facts of the latest session
	Host          map[string]string `json:"host"`      // allow-listed facts from the launcher, latest session
	Sessions      []Session         `json:"sessions"`
	Status        string            `json:"status"`
	StatusDetail  string            `json:"status_detail,omitempty"`
	StartedUTC    string            `json:"started_utc"`
	UpdatedUTC    string            `json:"updated_utc"`
	Counts        map[string]int    `json:"counts,omitempty"`
	Notes         []string          `json:"notes,omitempty"`
	Extensions    []Extension       `json:"extensions,omitempty"`
}

type ManifestPlan struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Description  string `json:"description"`
	Hash         string `json:"sha256"`
	Reps         int    `json:"reps"`
	CanaryEveryS int    `json:"canary_every_s"`
	Blocks       int    `json:"planned_blocks"`
}

type ManifestTH struct {
	SHA256     string           `json:"file_sha256"`
	Calibrated bool             `json:"calibrated"`
	Values     *quality.Machine `json:"values"`
}

func writeBytes(path string, b []byte) error { return record.WriteFileAtomic(path, b, 0o644) }

// keyValueFile parses "key=value" lines.
func keyValueFile(path string) map[string]string {
	out := map[string]string{}
	for _, l := range strings.Split(readTrim(path), "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

func (b *batch) manifest() Manifest {
	m := Manifest{
		Schema: 1, RunID: b.st.RunID, Machine: b.st.Machine,
		Plan: ManifestPlan{Name: b.p.Name, Kind: b.p.Kind, Description: b.p.Description, Hash: b.p.Hash,
			Reps: b.p.Reps, CanaryEveryS: b.p.CanaryEveryS, Blocks: b.st.Planned},
		Seed: b.st.Seed, ImageID: b.st.ImageID, HarnessCommit: buildinfo.HarnessCommit, Go: runtime.Version(),
		Build:      keyValueFile(filepath.Join(b.c.BuildInfo, "oqs-build.txt")),
		AptSources: map[string]string{},
		Thresholds: ManifestTH{SHA256: b.thHash, Calibrated: b.th.CalibratedFrom != "", Values: b.th},
		Sessions:   b.st.Sessions, Status: b.st.Status, StatusDetail: b.st.StatusDetail,
		StartedUTC: b.st.StartedUTC, UpdatedUTC: time.Now().UTC().Format(time.RFC3339),
	}
	// The variant line lists the minimal-build tokens; keep the manifest short.
	delete(m.Build, "minimal_build")
	for _, stage := range []string{"builder", "runtime"} {
		if v := readTrim(filepath.Join(b.c.BuildInfo, "apt-source-"+stage+".txt")); v != "" {
			m.AptSources[stage] = v
			if strings.HasPrefix(v, "LIVE") {
				m.Notes = append(m.Notes, "image stage "+stage+" was built from live Debian mirrors (snapshot unavailable)")
			}
		}
	}
	m.Extensions = b.st.Extensions
	m.Notes = append(m.Notes, extensionNotes(b.st)...)
	if b.p.Kind == plan.KindSmoke {
		m.Notes = append(m.Notes, "smoke plan: pipeline test with tiny budgets; never used for results")
	}
	if !m.Thresholds.Calibrated {
		m.Notes = append(m.Notes, "thresholds are the approved starting values (machine not yet calibrated)")
	}
	return m
}

func (b *batch) writeManifest(facts map[string]string, sf *proto.SessionFile) error {
	m := b.manifest()
	m.Container = facts
	m.Host = sf.Host
	b.container, b.host = facts, sf.Host
	return writeJSON(filepath.Join(b.c.Out, fileManifest), m)
}

func (b *batch) finalManifest() error {
	m := b.manifest()
	m.Container, m.Host = b.container, b.host
	in, err := loadEval(b.c.Out, b.st, b.th, true)
	if err != nil {
		return err
	}
	vs := evaluate(in)
	decideBase(b.p.Kind, vs, b.c.Seed, copyMap(b.st.Requeued), copyMap(b.st.Diagnosed), b.st.Base)
	c := map[string]int{"attempts": len(vs)}
	ids := map[string]bool{}
	headline := map[string]bool{}
	for _, v := range vs {
		bl := v.Res.Block
		if bl.Exp == "probe" {
			c["probe_blocks"]++
			continue
		}
		ids[bl.ID] = true
		if v.Headline {
			headline[bl.ID] = true
		}
		if v.flagged() {
			c["flagged_attempts"]++
		}
		if bl.Diagnostic {
			c["diagnostic_attempts"]++
		} else if bl.Attempt > 1 {
			c["repeat_attempts"]++
		}
		if v.Drift {
			c["drift_flags"]++
		}
		if v.Action == "exhausted" {
			c["exhausted_blocks"]++
		}
	}
	c["blocks"] = len(ids)
	c["blocks_with_headline"] = len(headline)
	m.Counts = c
	return writeJSON(filepath.Join(b.c.Out, fileManifest), m)
}

func copyMap(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// exists reports whether a path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
