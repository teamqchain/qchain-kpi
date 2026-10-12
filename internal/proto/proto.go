// Package proto is the contract between the orchestrator (kpi-batch) and the
// measuring process (kpi-prim): what a block is told to do, and the one-line
// JSON result it prints on stdout when it finishes. Both binaries import this
// package, so the two sides cannot disagree about the format.
package proto

import (
	"fmt"
	"path/filepath"
	"time"

	"qchain-kpi/internal/canary"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
	"qchain-kpi/internal/timing"
)

// Exit codes of kpi-prim.
const (
	ExitOK          = 0
	ExitError       = 1 // anything unexpected
	ExitUsage       = 2
	ExitGate        = 3 // wrong Go / liboqs version, algorithm not enabled
	ExitCorrectness = 4 // a result failed its correctness check: the data is invalid
)

// Arm names.
const (
	ArmReuse       = "reuse"        // E1 and probe: the library's reuse API
	ArmIssuer      = "issuer"       // E10: commitment hash, issuer key
	ArmHolder      = "holder"       // E10: presentation payload hash, holder key
	ArmIssuerAgain = "issuer-again" // E10: second commitment pool, issuer key (A/A)
)

// ProbeCalls is the number of calls after the cold call in a probe block.
const ProbeCalls = 5

// ProbeRep is the "repetition" of the small input set used by probe blocks
// (a cold call plus 5 calls need only ProbeCalls+1 inputs, so the probe pass
// does not pay for the full pools of the slow SLH-DSA sets).
const ProbeRep = 0

// PrepName is the file stem of an (algorithm, repetition) input set.
func PrepName(alg string, rep int) string {
	if rep == ProbeRep {
		return alg + ".probe"
	}
	return fmt.Sprintf("%s.r%d", alg, rep)
}

// PrepPath is the input-set file of an (algorithm, repetition) in dir.
func PrepPath(dir, alg string, rep int) string {
	return filepath.Join(dir, PrepName(alg, rep)+".gob")
}

// SessionFile is delivered by the launcher into the run folder (docker cp)
// before every start or resume of a batch; kpi-batch reads and removes it.
type SessionFile struct {
	HostOffsetNS      int64             `json:"host_offset_ns"`      // host clock minus container clock
	HostOffsetUncNS   int64             `json:"host_offset_unc_ns"`  // half the round trip of the measurement
	NativeP           float64           `json:"native_p_rate"`       // Mac: native canary, plain (P-core reference)
	NativeE           float64           `json:"native_e_rate"`       // Mac: native canary under taskpolicy -b (E-core reference)
	HostLogicalCPUs   int               `json:"host_logical_cpus"`   // CPUs of the host machine
	HostMonitorPeriod int               `json:"host_monitor_period_s"`
	Host              map[string]string `json:"host"` // allow-listed facts (see AllowedHostKeys)
}

// AllowedHostKeys are the only host facts a run may record (coding plan §10).
var AllowedHostKeys = map[string]bool{
	"runtime": true, "host_os": true, "host_cpu": true, "host_mem_gb": true,
	"docker": true, "colima": true, "lima": true, "wsl": true, "docker_desktop": true,
	"vm_cpus": true, "vm_mem_gb": true, "container_mem": true, "container_cpuset": true,
	"power": true, "battery_pct": true, "cooling": true, "launcher": true, "native_go": true,
	"low_power_mode": true, "thermal_at_start": true,
}

// BootReps is the number of bootstrap resamples for the standard error of a
// block median (used by the drift rule).
const BootReps = 200

// Spec is what kpi-batch passes to "kpi-prim block" (as JSON).
type Spec struct {
	Block   plan.Block     `json:"block"`
	PrepDir string         `json:"prep_dir"`
	RawFile string         `json:"raw_file"`        // absolute path of the block's raw file
	Batch   map[string]int `json:"batch,omitempty"` // arm -> calls per sample (default 1)
	MaxWall int            `json:"max_wall_s"`      // safety limit for the whole block
}

// Budget returns the sampling configuration for a profile and mode.
func Budget(profile string, mode timing.Mode, seed uint64, firstLabel string) (timing.Config, error) {
	var c timing.Config
	switch {
	case profile == plan.ProfileApproved && mode == timing.Fast:
		c = timing.FastConfig(seed, firstLabel)
	case profile == plan.ProfileApproved && mode == timing.Slow:
		c = timing.SlowConfig(seed, firstLabel)
	case profile == plan.ProfileSmoke && mode == timing.Fast:
		// Pipeline tests only: never used for results.
		c = timing.Config{Mode: timing.Fast, Warmup: 10, Samples: 200, FirstLabel: firstLabel, GCEvery: 100, Seed: seed}
	case profile == plan.ProfileSmoke && mode == timing.Slow:
		c = timing.Config{Mode: timing.Slow, Warmup: 1, TimeBox: time.Second, MinSamples: 3, FirstLabel: firstLabel, GCEvery: 100, Seed: seed}
	default:
		return c, fmt.Errorf("no budget for profile %q mode %q", profile, mode)
	}
	return c, nil
}

// BudgetE11 returns the sampling configuration of an E11 composite (approved
// 10 Oct): 20 untimed warm-up calls and 2,000 samples, or the slow-operation
// rule (60 s and at least 30 samples) for cells whose probe median is at least
// 30 ms. Garbage collection stays at Go's default (production-like), so no
// collections are forced between samples.
func BudgetE11(profile string, mode timing.Mode, seed uint64) (timing.Config, error) {
	var c timing.Config
	switch {
	case profile == plan.ProfileApproved && mode == timing.Fast:
		c = timing.Config{Mode: timing.Fast, Warmup: 20, Samples: 2000, FirstLabel: "first", Seed: seed}
	case profile == plan.ProfileApproved && mode == timing.Slow:
		c = timing.Config{Mode: timing.Slow, Warmup: 3, TimeBox: 60 * time.Second, MinSamples: 30, FirstLabel: "first", Seed: seed}
	case profile == plan.ProfileSmoke && mode == timing.Fast:
		c = timing.Config{Mode: timing.Fast, Warmup: 2, Samples: 50, FirstLabel: "first", Seed: seed}
	case profile == plan.ProfileSmoke && mode == timing.Slow:
		c = timing.Config{Mode: timing.Slow, Warmup: 1, TimeBox: time.Second, MinSamples: 3, FirstLabel: "first", Seed: seed}
	default:
		return c, fmt.Errorf("no E11 budget for profile %q mode %q", profile, mode)
	}
	return c, nil
}

// BudgetInfo records the budget a block ran with.
type BudgetInfo struct {
	Mode       string  `json:"mode"`
	Warmup     int     `json:"warmup"`
	Samples    int     `json:"samples,omitempty"`
	TimeBoxS   float64 `json:"time_box_s,omitempty"`
	MinSamples int     `json:"min_samples,omitempty"`
	GCEvery    int     `json:"gc_every"`
}

// Info describes a timing configuration.
func Info(c timing.Config) BudgetInfo {
	return BudgetInfo{Mode: string(c.Mode), Warmup: c.Warmup, Samples: c.Samples,
		TimeBoxS: c.TimeBox.Seconds(), MinSamples: c.MinSamples, GCEvery: c.GCEvery}
}

// ArmSummary is the result of one arm.
type ArmSummary struct {
	Name        string        `json:"name"`
	Batch       int           `json:"batch"`
	FirstLabel  string        `json:"first_label"` // "cold" or "first"
	FirstNS     int64         `json:"first_ns"`
	Stats       stats.Summary `json:"stats"` // ns per operation over the timed samples
	SumNS       int64         `json:"sum_ns"`
	MinBytes    int           `json:"min_bytes"`
	MaxBytes    int           `json:"max_bytes"`
	HeapBytesOp float64       `json:"heap_bytes_per_op"`
	AllocsOp    float64       `json:"allocs_per_op"`
	ProbeNS     []int64       `json:"probe_ns,omitempty"` // probe blocks: the calls after the cold call
}

// Result is the one-line JSON a block prints on stdout.
type Result struct {
	Key         string           `json:"key"`
	Block       plan.Block       `json:"block"`
	Budget      BudgetInfo       `json:"budget"`
	Arms        []ArmSummary     `json:"arms"`
	Truncated   bool             `json:"truncated"`
	StartNS     int64            `json:"start_unix_ns"` // just before the first call
	EndNS       int64            `json:"end_unix_ns"`   // just after the last timed sample
	WallNS      int64            `json:"wall_ns"`       // first call to last sample
	GCForced    int              `json:"gc_forced"`
	GCAuto      uint64           `json:"gc_automatic"` // GC cycles the runtime started itself (expected 0)
	Quality     quality.Interval `json:"quality"`
	CPUs        []int            `json:"cpus"`
	GOMAXPROCS  int              `json:"gomaxprocs"`
	OOMAdj      bool             `json:"oom_score_adj_set"`
	Pool        int              `json:"pool"`
	PrepID      string           `json:"prep_id"`
	KeysChecked int              `json:"keys_fully_checked,omitempty"`
	PrepNS      int64            `json:"prep_ns,omitempty"` // E11: untimed input preparation inside the block
	Rows        int              `json:"rows"`
	RawSHA256   string           `json:"raw_sha256"`
}

// PrepResult is what "kpi-prim prep" prints.
type PrepResult struct {
	Alg         string `json:"alg"`
	Rep         int    `json:"rep"`
	Pool        int    `json:"pool"`
	PrepNS      int64  `json:"prep_ns"`
	Fingerprint string `json:"fingerprint"`
	E10         bool   `json:"e10"`
}

// CanaryResult is what "kpi-prim canary" prints.
type CanaryResult struct {
	canary.Result
	StartNS int64 `json:"start_unix_ns"`
	EndNS   int64 `json:"end_unix_ns"`
	CPUs    []int `json:"cpus"`
}
