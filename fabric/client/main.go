// Command kpi-fabric is E14's client (QChain KPI study): it submits QChain-
// shaped transactions to the E14 Fabric network, the way QChain's backend
// does, and records how long each takes and which ledger block holds it.
//
//	kpi-fabric run --plan P --run-id ID --machine M --seed N --image-id I [--out /out] [--material /kpi]
//	kpi-fabric clock          # prints this container's Unix time in ns (clock offset)
//
// Exit codes: 0 done, 5 stopped (resume later), 2 refused, 1 failed.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	_ "time/tzdata" // Dubai time from the binary, as QChain's backend (offchain/config.go)
)

// harnessCommit is set at build time (-ldflags -X main.harnessCommit=...).
var harnessCommit = "unknown"

const (
	exitDone    = 0
	exitFailed  = 1
	exitRefused = 2
	exitStopped = 5
)

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kpi-fabric run|clock ...")
		os.Exit(exitRefused)
	}
	switch os.Args[1] {
	case "clock":
		fmt.Println(time.Now().UnixNano())
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(exitRefused)
	}
}

// SessionFile is what the launcher delivers before every start or resume
// (the harness's format; host facts are allow-listed).
type SessionFile struct {
	HostOffsetNS      int64             `json:"host_offset_ns"`
	HostOffsetUncNS   int64             `json:"host_offset_unc_ns"`
	NativeP           float64           `json:"native_p_rate"`
	NativeE           float64           `json:"native_e_rate"`
	HostLogicalCPUs   int               `json:"host_logical_cpus"`
	HostMonitorPeriod int               `json:"host_monitor_period_s"`
	Host              map[string]string `json:"host"`
}

// allowedHostKeys: the harness's internal/proto AllowedHostKeys.
var allowedHostKeys = map[string]bool{
	"runtime": true, "host_os": true, "host_cpu": true, "host_mem_gb": true,
	"docker": true, "colima": true, "lima": true, "wsl": true, "docker_desktop": true,
	"vm_cpus": true, "vm_mem_gb": true, "container_mem": true, "container_cpuset": true,
	"power": true, "battery_pct": true, "cooling": true, "launcher": true, "native_go": true,
	"low_power_mode": true, "thermal_at_start": true,
}

func readSession(dir string) (*SessionFile, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var sf SessionFile
	if err := dec.Decode(&sf); err != nil {
		return nil, fmt.Errorf("session.json: %w", err)
	}
	for k, v := range sf.Host {
		if !allowedHostKeys[k] || len(v) > 120 || strings.ContainsAny(v, "\n\r") {
			return nil, fmt.Errorf("session.json: host fact %q is not allowed", k)
		}
	}
	return &sf, nil
}

// Manifest describes the run (manifest.json).
type Manifest struct {
	Schema        int                    `json:"schema"`
	Experiment    string                 `json:"experiment"`
	RunID         string                 `json:"run_id"`
	Machine       string                 `json:"machine"`
	Plan          map[string]string      `json:"plan"`
	Seed          uint64                 `json:"seed"`
	ImageID       string                 `json:"image_id"`
	HarnessCommit string                 `json:"harness_commit"`
	Go            string                 `json:"go"`
	Network       json.RawMessage        `json:"network,omitempty"`
	Host          map[string]string      `json:"host,omitempty"`
	Sessions      []SessionRec           `json:"sessions"`
	Status        string                 `json:"status"`
	StatusDetail  string                 `json:"status_detail,omitempty"`
	StartedUTC    string                 `json:"started_utc"`
	UpdatedUTC    string                 `json:"updated_utc"`
	Counts        map[string]int         `json:"counts"`
	Calibration   map[string]Calibration `json:"calibration,omitempty"`
	Notes         []string               `json:"notes"`
}

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	planPath := fs.String("plan", "", "plan file")
	runID := fs.String("run-id", "", "run id")
	machine := fs.String("machine", "", "machine id")
	seed := fs.Uint64("seed", 0, "run seed")
	imageID := fs.String("image-id", "", "client image id")
	out := fs.String("out", "/out", "run folder (the run's named volume)")
	material := fs.String("material", "/kpi", "the network's material volume (certificates)")
	if err := fs.Parse(args); err != nil {
		return exitRefused
	}
	if *planPath == "" || *runID == "" || *machine == "" || *seed == 0 || *imageID == "" {
		logf("run: --plan, --run-id, --machine, --seed and --image-id are required")
		return exitRefused
	}
	plan, err := LoadPlan(*planPath)
	if err != nil {
		logf("%v", err)
		return exitRefused
	}
	blocks := plan.Expand(*seed)
	sf, err := readSession(*out)
	if err != nil {
		logf("%v", err)
		return exitRefused
	}
	if sf == nil && plan.Kind == "main" {
		logf("run: no session.json from the launcher (required for main plans)")
		return exitRefused
	}
	if sf == nil {
		sf = &SessionFile{}
	}

	statePath := filepath.Join(*out, "state.json")
	var st *State
	if raw, err := os.ReadFile(statePath); err == nil {
		st = &State{}
		if err := json.Unmarshal(raw, st); err != nil {
			logf("state.json: %v", err)
			return exitFailed
		}
		var pr []string
		check := func(what, was, now string) {
			if was != now {
				pr = append(pr, fmt.Sprintf("%s was %q, now %q", what, was, now))
			}
		}
		check("run id", st.RunID, *runID)
		check("machine", st.Machine, *machine)
		check("plan", st.PlanHash, plan.Hash)
		check("seed", fmt.Sprint(st.Seed), fmt.Sprint(*seed))
		check("image", st.ImageID, *imageID)
		check("harness commit", st.HarnessCommit, harnessCommit)
		if len(pr) > 0 {
			logf("run: refusing to resume: %s", strings.Join(pr, "; "))
			return exitRefused
		}
		if st.Status == "done" {
			logf("run: this run is already complete")
			return exitRefused
		}
		cleanTemp(*out)
	} else {
		ents, _ := os.ReadDir(*out)
		for _, e := range ents {
			switch e.Name() {
			case "monitor", "session.json", "network.json", "STOP":
			default:
				logf("run: %s is not empty (%s); use a new run id and volume", *out, e.Name())
				return exitRefused
			}
		}
		ids := make([]string, len(blocks))
		for i, b := range blocks {
			ids[i] = b.ID
		}
		st = &State{Schema: 1, RunID: *runID, Machine: *machine, PlanName: plan.Name, PlanKind: plan.Kind, PlanHash: plan.Hash,
			Seed: *seed, ImageID: *imageID, HarnessCommit: harnessCommit, StartedUTC: time.Now().UTC().Format(time.RFC3339),
			Order: ids, Done: map[string]DoneRec{}}
	}
	session := len(st.Sessions) + 1
	st.Sessions = append(st.Sessions, SessionRec{N: session, StartedUTC: time.Now().UTC().Format(time.RFC3339),
		HostOffsetNS: sf.HostOffsetNS, NativeP: sf.NativeP, NativeE: sf.NativeE})
	st.Status, st.StatusDetail = "running", ""
	os.Remove(filepath.Join(*out, "STOP"))

	r := &Runner{Plan: plan, Blocks: blocks, Out: *out, St: st, Session: session, Stop: &atomic.Bool{}, Log: logf, Now: time.Now}
	if err := r.saveState(); err != nil {
		logf("%v", err)
		return exitFailed
	}
	network, _ := os.ReadFile(filepath.Join(*out, "network.json"))
	if len(network) > 0 {
		if err := writeFileAtomic(filepath.Join(*out, "env", fmt.Sprintf("network-s%d.json", session)), network); err != nil {
			logf("%v", err)
			return exitFailed
		}
		os.Remove(filepath.Join(*out, "network.json"))
	}
	os.Remove(filepath.Join(*out, "session.json"))
	if err := writeJSON(filepath.Join(*out, "env", fmt.Sprintf("e0-s%d.json", session)), envCapture()); err != nil {
		logf("%v", err)
		return exitFailed
	}
	writeManifest := func() error {
		m := Manifest{Schema: 1, Experiment: "E14", RunID: st.RunID, Machine: st.Machine,
			Plan: map[string]string{"name": plan.Name, "kind": plan.Kind, "sha256": plan.Hash, "description": plan.Description},
			Seed: st.Seed, ImageID: st.ImageID, HarnessCommit: st.HarnessCommit, Go: runtime.Version(),
			Host: sf.Host, Sessions: st.Sessions, Status: st.Status, StatusDetail: st.StatusDetail,
			StartedUTC: st.StartedUTC, UpdatedUTC: time.Now().UTC().Format(time.RFC3339),
			// "attempts" = finished blocks and "flagged_attempts" = 0 (E14 repeats no block), for the launcher's INDEX row.
			Counts:      map[string]int{"blocks": len(blocks), "blocks_done": len(st.Done), "attempts": len(st.Done), "flagged_attempts": 0, "transactions": st.Transactions, "mvcc_conflicts": st.FalseConflict},
			Calibration: st.Calibration,
			Notes: []string{"QChain's Fabric network rebuilt on the host in Docker (scripts/e14-network-mac.sh); every transaction committed as valid; " +
				"run quality is recorded (host monitor, canary, drift-control blocks), not repeated (decision 21 Oct); " +
				"transactions are valid, or flagged MVCC_READ_CONFLICT by the second-validating peer of the shared CouchDB with their record read back correctly (counted in mvcc_conflicts; decision 22 Oct)"}}
		if len(network) > 0 {
			m.Network = json.RawMessage(network)
		}
		if plan.Kind == "smoke" {
			m.Notes = append(m.Notes, "smoke plan: pipeline test with tiny budgets; never used for results")
		}
		return writeJSON(filepath.Join(*out, "manifest.json"), m)
	}
	if err := writeManifest(); err != nil {
		logf("%v", err)
		return exitFailed
	}
	r.journal("session_start", "", fmt.Sprintf("session %d; host offset %d ns", session, sf.HostOffsetNS))

	ledger, err := NewGatewayLedger(DefaultGateway(*material))
	if err != nil {
		logf("connect to the network: %v", err)
		st.Status, st.StatusDetail = "failed", err.Error()
		r.saveState()
		writeManifest()
		return exitFailed
	}
	defer ledger.Close()
	r.Ledger = ledger

	// Ctrl-C and SIGTERM stop at once (the current block is discarded); a
	// STOP file stops after the current block.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		r.Stop.Store(true)
		cancel()
	}()
	go func() {
		for !r.Stop.Load() {
			if _, err := os.Stat(filepath.Join(*out, "STOP")); err == nil {
				logf("STOP file found: stopping after the current block")
				r.Stop.Store(true)
				return
			}
			time.Sleep(2 * time.Second)
		}
	}()

	logf("run %s session %d: plan %s (%s), %d blocks, %d done", st.RunID, session, plan.Name, plan.Kind, len(blocks), len(st.Done))
	err = r.Run(ctx)
	code := exitDone
	switch {
	case err == nil && len(st.Done) == len(blocks):
		st.Status = "done"
		r.journal("run_done", "", fmt.Sprintf("%d blocks, %d transactions", len(st.Done), st.Transactions))
	case err == nil || errors.Is(err, errStopped):
		st.Status, st.StatusDetail = "stopped", "stopped (the current block is discarded and runs again on resume)"
		r.journal("stopped", "", "")
		code = exitStopped
	default:
		st.Status, st.StatusDetail = "failed", err.Error()
		r.journal("failed", "", err.Error())
		code = exitFailed
	}
	if err := r.saveState(); err != nil {
		logf("%v", err)
		return exitFailed
	}
	if err := writeManifest(); err != nil {
		logf("%v", err)
		return exitFailed
	}
	if code == exitDone {
		if err := writeSums(*out); err != nil {
			logf("%v", err)
			return exitFailed
		}
		logf("done: %d blocks, %d transactions", len(st.Done), st.Transactions)
	} else {
		logf("%s: %s", st.Status, st.StatusDetail)
	}
	return code
}

// envCapture records the client's environment (nothing that identifies the host).
func envCapture() map[string]any {
	read := func(p string) string {
		b, _ := os.ReadFile(p)
		return strings.TrimSpace(string(b))
	}
	cpu := ""
	if f, err := os.Open("/proc/cpuinfo"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			k, v, ok := strings.Cut(sc.Text(), ":")
			k = strings.TrimSpace(k)
			if ok && (k == "model name" || k == "CPU part" || k == "CPU implementer") {
				cpu += k + "=" + strings.TrimSpace(v) + " "
			}
			if cpu != "" && k == "" {
				break // processor 0 only
			}
		}
		f.Close()
	}
	mem := ""
	if f, err := os.Open("/proc/meminfo"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "MemTotal:") {
				mem = strings.TrimSpace(strings.TrimPrefix(sc.Text(), "MemTotal:"))
			}
		}
		f.Close()
	}
	return map[string]any{
		"schema": 1, "captured_utc": time.Now().UTC().Format(time.RFC3339), "harness_commit": harnessCommit,
		"go": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"num_cpu": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"kernel": read("/proc/sys/kernel/osrelease"), "cpu": strings.TrimSpace(cpu), "mem_total": mem,
		"timezone": os.Getenv("TZ"), "local_zone": time.Now().Location().String(),
	}
}
