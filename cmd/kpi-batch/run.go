package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/timing"
)

// Exit codes of "kpi-batch run".
const (
	exitDone        = 0
	exitFailed      = 1 // a block or a step failed unexpectedly
	exitUsage       = 2
	exitRefused     = 3 // preflight or resume refused; nothing was measured
	exitCorrectness = 4 // a correctness check failed: the data of that block is invalid
	exitStopped     = 5 // stopped on request (STOP file, SIGTERM) or host monitor stale; resumable
)

const sessionFileName = "session.json"

// Machines of the study (coding plan §10).
var machines = map[string]bool{"mac-m1": true, "win-ryzen": true, "vm-x86v2": true}

var runIDPattern = regexp.MustCompile(`^[0-9]{8}-[0-9]{4}_[a-z0-9-]+_[a-z0-9-]+_[a-z0-9-]+$`)

type runConfig struct {
	PlanPath       string
	RunID          string
	Machine        string
	Seed           uint64
	ImageID        string
	ThresholdsPath string
	Out            string
	PrepDir        string
	PrimPath       string
	MallocLib      string // E9: the C-heap counter its blocks run with (LD_PRELOAD)
	EnvPath        string
	BuildInfo      string
	MaxBlockS      int
	MonitorStaleS  int
	CoverWaitS     int
	Extend         string // comma-separated block ids of a finished run to run again
	AllowRebuild   bool   // with Extend: the image and harness commit may differ (sources verified by the launcher)
	OrchChanged    string // with Extend: cmd/kpi-batch files that differ from the original run (from git, comma-separated)
}

type batch struct {
	c        runConfig
	p        *plan.Plan
	th       *quality.Machine
	thHash   string
	strict   bool
	st       *State
	sess     Session
	hostCPUs int
	vmCPUs   int
	periodNS int64
	cellReps map[string]int
	e10Algs  map[string]bool
	e6Algs   map[string]bool // signature schemes whose preps need the E6 message pools
	cur      atomic.Pointer[exec.Cmd]
	stopSig  atomic.Bool
	lastEnd  int64 // end of the most recent block (container clock)

	extendFrom identity // image and harness commit the run was made with (extension only)

	container map[string]string // preflight facts (manifest)
	host      map[string]string // allow-listed host facts (manifest)
}

var errInterrupted = errors.New("interrupted")

func cmdRun(args []string) (int, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var c runConfig
	fs.StringVar(&c.PlanPath, "plan", "", "run plan (JSON)")
	fs.StringVar(&c.RunID, "run-id", "", "run id: <UTC yyyymmdd-hhmm>_<machine>_<runtime>_<variant>")
	fs.StringVar(&c.Machine, "machine", "", "mac-m1, win-ryzen or vm-x86v2")
	fs.Uint64Var(&c.Seed, "seed", 0, "batch seed (recorded; drives order and inputs)")
	fs.StringVar(&c.ImageID, "image-id", "", "image ID reported by the launcher (docker image inspect)")
	fs.StringVar(&c.ThresholdsPath, "thresholds", "/opt/kpi/config/thresholds.json", "thresholds file")
	fs.StringVar(&c.Out, "out", "/out", "run folder (the run's named volume)")
	fs.StringVar(&c.PrepDir, "prep-dir", "/tmp/kpi-prep", "input sets (inside the container, never copied out)")
	fs.StringVar(&c.PrimPath, "prim", "/opt/kpi/bin/kpi-prim", "kpi-prim binary")
	fs.StringVar(&c.MallocLib, "malloc-lib", proto.E9MallocLib, "E9: C-heap counter library, preloaded into E9 blocks only")
	fs.StringVar(&c.EnvPath, "env", "/opt/kpi/bin/kpi-env", "kpi-env binary")
	fs.StringVar(&c.BuildInfo, "build-info", "/opt/kpi/build-info", "build records of the image")
	fs.IntVar(&c.MaxBlockS, "max-block-s", 2700, "safety limit for one block")
	fs.IntVar(&c.MonitorStaleS, "monitor-stale-s", 180, "trial/main: stop if no host sample is newer than this")
	fs.IntVar(&c.CoverWaitS, "cover-wait-s", 240, "at the end, wait up to this long for host samples to cover the last block")
	fs.StringVar(&c.Extend, "extend", "", "extend a finished run: block ids (E1.alg.op.rN, comma-separated) that have no clean attempt")
	fs.StringVar(&c.OrchChanged, "orchestrator-changed", "", "with --extend: cmd/kpi-batch files that differ from the original run's commit (recorded)")
	fs.BoolVar(&c.AllowRebuild, "allow-rebuild", false, "with --extend: accept a different image and harness commit (recorded)")
	if err := fs.Parse(args); err != nil {
		return exitUsage, err
	}
	b, code, err := newBatch(c)
	if err != nil {
		return code, err
	}
	return b.run()
}

func newBatch(c runConfig) (*batch, int, error) {
	switch {
	case c.PlanPath == "" || c.RunID == "" || c.Machine == "" || c.ImageID == "":
		return nil, exitUsage, errors.New("run: --plan, --run-id, --machine and --image-id are required")
	case !machines[c.Machine]:
		return nil, exitUsage, fmt.Errorf("run: unknown machine %q", c.Machine)
	case !runIDPattern.MatchString(c.RunID):
		return nil, exitUsage, fmt.Errorf("run: run id %q does not match <yyyymmdd-hhmm>_<machine>_<runtime>_<variant>", c.RunID)
	case !strings.Contains(c.RunID, "_"+c.Machine+"_"):
		return nil, exitUsage, fmt.Errorf("run: run id %q does not name machine %s", c.RunID, c.Machine)
	}
	p, err := plan.Load(c.PlanPath)
	if err != nil {
		return nil, exitUsage, err
	}
	thf, err := quality.Load(c.ThresholdsPath)
	if err != nil {
		return nil, exitRefused, err
	}
	th, calibrated := thf.For(c.Machine)
	if p.Kind == plan.KindMain && (!calibrated || th.CalibratedFrom == "") {
		return nil, exitRefused, fmt.Errorf("run: %s has no calibrated thresholds; run and calibrate the trial first", c.Machine)
	}
	b := &batch{c: c, p: p, th: th, thHash: quality.HashFile(c.ThresholdsPath), strict: p.Kind != plan.KindSmoke,
		cellReps: map[string]int{}, e10Algs: map[string]bool{}, e6Algs: map[string]bool{}}

	blocks, err := plan.Expand(p, c.Seed)
	if err != nil {
		return nil, exitUsage, err
	}
	needE2Batch, needE11Class, needE6Class, needE7Batch := false, false, false, false
	for _, bl := range blocks {
		if proto.Measured(bl.Exp) {
			b.cellReps[bl.Cell()]++
		}
		if bl.Exp == "E2" || bl.Exp == "E2c" {
			needE2Batch = true
		}
		if bl.Exp == "E11" {
			needE11Class = true
		}
		if bl.Exp == "E6" || bl.Exp == "probe-E6" {
			b.e6Algs[bl.Alg] = true
			needE6Class = needE6Class || bl.Exp == "E6"
		}
		if bl.Exp == "E7" {
			needE7Batch = true
		}
		if bl.Exp == "E10" {
			b.e10Algs[bl.Alg] = true
		}
	}
	if p.Kind == plan.KindMain {
		if missing := b.missingClassification(blocks); len(missing) > 0 {
			return nil, exitRefused, fmt.Errorf("run: thresholds for %s lack the classification of %s", c.Machine, strings.Join(missing, ", "))
		}
		if needE2Batch && th.BatchCalibratedFrom["E2"] == "" {
			return nil, exitRefused, fmt.Errorf("run: thresholds for %s have no E2 batch sizes; run the trial-e2 plan and merge its calibration first", c.Machine)
		}
		if needE11Class && th.ClassCalibratedFrom["E11"] == "" {
			return nil, exitRefused, fmt.Errorf("run: thresholds for %s have no E11 classification; run the trial-e11 plan and merge its calibration first", c.Machine)
		}
		if needE6Class && th.ClassCalibratedFrom["E6"] == "" {
			return nil, exitRefused, fmt.Errorf("run: thresholds for %s have no E6 classification; run the trial-e6 plan and merge its calibration first", c.Machine)
		}
		if needE7Batch && th.BatchCalibratedFrom["E7"] == "" {
			return nil, exitRefused, fmt.Errorf("run: thresholds for %s have no E7 batch sizes; run the trial-e7 plan and merge its calibration first", c.Machine)
		}
	}

	if err := os.MkdirAll(filepath.Join(c.Out, dirMonitor), 0o755); err != nil {
		return nil, exitFailed, err
	}
	// Preflight inside the container.
	wantCPUs := 1
	if p.CPUSet != "" {
		if wantCPUs, err = plan.CPUCount(p.CPUSet); err != nil {
			return nil, exitUsage, err
		}
	}
	facts, problems := preflight(c.Out, wantCPUs)
	if len(problems) > 0 {
		for _, pr := range problems {
			logf("preflight: %s", pr)
		}
		if b.strict {
			return nil, exitRefused, fmt.Errorf("run: preflight failed (%d problem(s)); smoke plans only record them", len(problems))
		}
	}
	snap, err := quality.Take()
	if err != nil {
		return nil, exitRefused, fmt.Errorf("run: cannot read CPU counters: %w", err)
	}
	b.vmCPUs = len(snap.PerCPU)

	// Session file from the launcher.
	sf, err := readSessionFile(c.Out)
	if err != nil {
		return nil, exitRefused, err
	}
	if sf == nil && b.strict {
		return nil, exitRefused, errors.New("run: no session.json from the launcher (required for trial and main plans)")
	}
	if sf == nil {
		sf = &proto.SessionFile{}
	}
	b.hostCPUs = sf.HostLogicalCPUs
	if b.hostCPUs <= 0 {
		b.hostCPUs = b.vmCPUs
	}
	period := sf.HostMonitorPeriod
	if period <= 0 {
		period = 5
	}
	b.periodNS = int64(period) * int64(time.Second)

	now := identity{RunID: c.RunID, Machine: c.Machine, PlanHash: p.Hash, ImageID: c.ImageID,
		HarnessCommit: buildinfo.HarnessCommit, ThresholdsHash: b.thHash, Seed: c.Seed}
	if _, err := os.Stat(filepath.Join(c.Out, fileState)); err == nil {
		st, err := loadState(c.Out)
		if err != nil {
			return nil, exitRefused, err
		}
		cmp := now
		if c.Extend != "" && c.AllowRebuild {
			cmp = withoutBuild(now, st.identity())
		}
		if pr := resumeProblems(st.identity(), cmp); len(pr) > 0 {
			return nil, exitRefused, fmt.Errorf("run: refusing to resume: %s", strings.Join(pr, "; "))
		}
		switch {
		case c.Extend == "" && st.Status == "done":
			return nil, exitRefused, errors.New("run: this batch is already complete")
		case c.Extend != "" && st.Status != "done":
			return nil, exitRefused, fmt.Errorf("run: only a finished run can be extended (status %q); resume it instead", st.Status)
		}
		b.st = st
		b.extendFrom = identity{ImageID: st.ImageID, HarnessCommit: st.HarnessCommit}
		if c.Extend != "" {
			b.st.ImageID, b.st.HarnessCommit = c.ImageID, buildinfo.HarnessCommit
		}
		n, err := cleanTemp(c.Out)
		if err != nil {
			return nil, exitFailed, err
		}
		logf("resuming %s: %d block(s) queued, %d done, %d temporary file(s) removed", c.RunID, len(st.Queue), len(st.Done), n)
	} else {
		stray, err := strayFiles(c.Out)
		if err != nil {
			return nil, exitFailed, err
		}
		if len(stray) > 0 {
			return nil, exitRefused, fmt.Errorf("run: %s is not empty (%s); use a new run id and volume", c.Out, strings.Join(stray, ", "))
		}
		b.st = &State{Schema: 1, RunID: c.RunID, Machine: c.Machine, PlanName: p.Name, PlanKind: p.Kind, PlanHash: p.Hash,
			Seed: c.Seed, ImageID: c.ImageID, HarnessCommit: buildinfo.HarnessCommit, ThresholdsHash: b.thHash,
			StartedUTC: time.Now().UTC().Format(time.RFC3339), Planned: len(blocks), Queue: blocks,
			Class: map[string]string{}, Requeued: map[string]bool{}, Diagnosed: map[string]bool{}, Preps: map[string]PrepRec{}}
	}
	b.sess = Session{N: len(b.st.Sessions) + 1, StartedUTC: time.Now().UTC().Format(time.RFC3339), StartNS: time.Now().UnixNano(),
		HostOffsetNS: sf.HostOffsetNS, NativeP: sf.NativeP, NativeE: sf.NativeE}
	b.st.Sessions = append(b.st.Sessions, b.sess)
	b.st.CellReps, b.st.VMCPUs, b.st.HostCPUs = b.cellReps, b.vmCPUs, b.hostCPUs
	b.st.HostPeriodNS, b.st.CanaryEveryS = b.periodNS, p.CanaryEveryS
	if c.Extend != "" {
		if err := b.extendBlocks(parseBlockList(c.Extend), b.extendFrom.ImageID, b.extendFrom.HarnessCommit); err != nil {
			return nil, exitRefused, err
		}
	}
	b.st.Status = "running"
	b.st.StatusDetail = ""
	if err := b.st.save(c.Out); err != nil {
		return nil, exitFailed, err
	}
	// The session file is consumed: a later session must bring its own.
	os.Remove(filepath.Join(c.Out, sessionFileName))
	if err := b.journal("session_start", "", fmt.Sprintf("session %d; host offset %d ns", b.sess.N, b.sess.HostOffsetNS)); err != nil {
		return nil, exitFailed, err
	}
	if err := b.writeManifest(facts, sf); err != nil {
		return nil, exitFailed, err
	}
	return b, 0, nil
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

func readSessionFile(dir string) (*proto.SessionFile, error) {
	raw, err := os.ReadFile(filepath.Join(dir, sessionFileName))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var sf proto.SessionFile
	if err := dec.Decode(&sf); err != nil {
		return nil, fmt.Errorf("session.json: %w", err)
	}
	for k, v := range sf.Host {
		if !proto.AllowedHostKeys[k] {
			return nil, fmt.Errorf("session.json: host fact %q is not on the allow-list", k)
		}
		if len(v) > 120 || strings.ContainsAny(v, "\n\r") {
			return nil, fmt.Errorf("session.json: host fact %q has an unacceptable value", k)
		}
	}
	return &sf, nil
}

// preflight checks the container (coding plan §9). Problems refuse trial and
// main plans; smoke plans record them.
func preflight(out string, wantCPUs int) (map[string]string, []string) {
	facts := map[string]string{}
	var problems []string
	facts["gomaxprocs"] = strconv.Itoa(runtime.GOMAXPROCS(0))
	if runtime.GOMAXPROCS(0) != 1 {
		problems = append(problems, fmt.Sprintf("GOMAXPROCS is %d, want 1", runtime.GOMAXPROCS(0)))
	}
	cpus, err := quality.CPUSet()
	facts["cpuset"] = fmt.Sprint(cpus)
	if err != nil || len(cpus) != wantCPUs {
		problems = append(problems, fmt.Sprintf("container has CPUs %v, the plan needs %d (%v)", cpus, wantCPUs, err))
	}
	if ents, err := os.ReadDir("/sys/class/net"); err == nil {
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		facts["net_interfaces"] = strings.Join(names, " ")
		if len(names) != 1 || names[0] != "lo" {
			problems = append(problems, "network is not disabled (--network none)")
		}
	}
	mem := readTrim("/sys/fs/cgroup/memory.max")
	swap := readTrim("/sys/fs/cgroup/memory.swap.max")
	facts["memory_max"], facts["memory_swap_max"] = mem, swap
	if mem == "" || mem == "max" {
		problems = append(problems, "no container memory limit")
	}
	if swap != "0" {
		problems = append(problems, fmt.Sprintf("swap is allowed (memory.swap.max %q)", swap))
	}
	// Pinning gives the container exactly one CPU. A CPU-time quota on top
	// (docker --cpus) throttles a fully busy process every few periods
	// (measured 10 Oct: 29 of 100 periods, ~1.8 ms each), so none may be set.
	cpuMax := readTrim("/sys/fs/cgroup/cpu.max")
	facts["cpu_max"] = cpuMax
	if f := strings.Fields(cpuMax); len(f) == 0 || f[0] != "max" {
		problems = append(problems, fmt.Sprintf("a CPU-time quota is set (cpu.max %q); pin with --cpuset-cpus only, without --cpus", cpuMax))
	}
	if os.Getuid() == 0 {
		problems = append(problems, "running as root")
	}
	facts["tz"] = os.Getenv("TZ")
	if os.Getenv("TZ") != "Asia/Dubai" {
		problems = append(problems, "TZ is not Asia/Dubai (production parity)")
	}
	var sfs syscall.Statfs_t
	if syscall.Statfs(out, &sfs) == nil {
		free := sfs.Bavail * uint64(sfs.Bsize)
		facts["out_free_gb"] = strconv.FormatFloat(float64(free)/1e9, 'f', 1, 64)
		if free < 2e9 {
			problems = append(problems, "less than 2 GB free in the run volume")
		}
	}
	return facts, problems
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// missingClassification lists the cells a main plan needs but the thresholds lack.
func (b *batch) missingClassification(blocks []plan.Block) []string {
	seen := map[string]bool{}
	var out []string
	for _, bl := range blocks {
		if proto.IsProbe(bl.Exp) || bl.Exp == "E2c" || proto.Untimed(bl.Exp) || bl.Exp == "E12" || proto.IsE3Null(bl.Exp, bl.Alg, bl.Op) { // E2c, E3 null: always fast; E8, E9, E13: not timed; E12: fixed window
			continue
		}
		k := bl.Alg + "/" + bl.Op
		if _, ok := b.th.Class[k]; !ok && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func (b *batch) journal(event, key, detail string) error {
	return appendCSV(filepath.Join(b.c.Out, fileJournal), []string{"utc_ns", "event", "key", "detail"},
		[]string{strconv.FormatInt(time.Now().UnixNano(), 10), event, key, detail})
}

// --------------------------------------------------------------- main loop

func (b *batch) run() (int, error) {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for range sigs {
			b.stopSig.Store(true)
			if c := b.cur.Load(); c != nil && c.Process != nil {
				c.Process.Kill()
			}
		}
	}()

	if err := b.captureEnv(); err != nil {
		return b.fail(exitFailed, err)
	}
	reason := "start"
	if b.sess.N > 1 {
		reason = "resume"
	}
	if b.c.Extend != "" {
		reason = "extend"
	}
	if err := b.canary(reason); err != nil {
		return b.stopOrFail(err)
	}
	for {
		for len(b.st.Queue) > 0 {
			if code, err, stop := b.checkStop(); stop {
				return code, err
			}
			if time.Since(time.Unix(0, b.st.LastCanaryNS)) >= time.Duration(b.p.CanaryEveryS)*time.Second {
				if code, err, stop := b.checkpoint(); stop {
					return code, err
				}
				continue // the checkpoint may have queued a repeat at the head
			}
			if err := b.runBlock(b.st.Queue[0]); err != nil {
				return b.stopOrFail(err)
			}
		}
		if err := b.canary("end"); err != nil {
			return b.stopOrFail(err)
		}
		if code, err, stop := b.waitCoverage(); stop {
			return code, err
		}
		added, err := b.evaluate(true)
		if err != nil {
			return b.fail(exitFailed, err)
		}
		if added == 0 {
			break
		}
	}
	return b.finish()
}

func (b *batch) checkStop() (int, error, bool) {
	if b.stopSig.Load() {
		return b.stopped("stopped by signal"), nil, true
	}
	if _, err := os.Stat(filepath.Join(b.c.Out, fileStop)); err == nil {
		// The launcher's file is owned by root; it is removed (the journal records the stop).
		os.Remove(filepath.Join(b.c.Out, fileStop))
		return b.stopped("stopped by STOP file"), nil, true
	}
	return 0, nil, false
}

func (b *batch) stopped(why string) int {
	logf("%s; the batch can be resumed", why)
	b.journal("stopped", "", why)
	b.st.Status, b.st.StatusDetail = "stopped", why
	b.st.save(b.c.Out)
	return exitStopped
}

func (b *batch) stopOrFail(err error) (int, error) {
	if errors.Is(err, errInterrupted) || b.stopSig.Load() {
		return b.stopped("stopped by signal (current block discarded; it re-runs on resume)"), nil
	}
	var ce correctnessError
	if errors.As(err, &ce) {
		return b.fail(exitCorrectness, err)
	}
	return b.fail(exitFailed, err)
}

func (b *batch) fail(code int, err error) (int, error) {
	b.journal("failed", "", err.Error())
	b.st.Status, b.st.StatusDetail = "failed", err.Error()
	b.st.save(b.c.Out)
	return code, err
}

type correctnessError struct{ msg string }

func (e correctnessError) Error() string { return e.msg }

// exec runs kpi-prim (or kpi-env) as a fresh child process.
func (b *batch) exec(bin string, args ...string) ([]byte, []byte, int, error) {
	return b.execEnv(nil, bin, args...)
}

// execEnv is exec with extra environment variables (E9's LD_PRELOAD).
func (b *batch) execEnv(env []string, bin string, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.Command(bin, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	if err := cmd.Start(); err != nil {
		return nil, nil, -1, err
	}
	b.cur.Store(cmd)
	err := cmd.Wait()
	b.cur.Store(nil)
	code := cmd.ProcessState.ExitCode()
	if b.stopSig.Load() {
		return so.Bytes(), se.Bytes(), code, errInterrupted
	}
	return so.Bytes(), se.Bytes(), code, err
}

func lastLine(b []byte) []byte {
	b = bytes.TrimRight(b, "\n")
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		return b[i+1:]
	}
	return b
}

func tail(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		s = "..." + s[len(s)-n:]
	}
	return strings.ReplaceAll(s, "\n", " | ")
}

// captureEnv runs E0 for this session and keeps the image's build records.
func (b *batch) captureEnv() error {
	envDir := filepath.Join(b.c.Out, dirEnv)
	if b.sess.N == 1 {
		if err := copyDir(b.c.BuildInfo, filepath.Join(envDir, "build-info")); err != nil {
			return fmt.Errorf("copy build records: %w", err)
		}
	} else if b.c.Extend != "" && b.c.AllowRebuild {
		if err := copyDir(b.c.BuildInfo, extendBuildInfoDir(b.c.Out, b.sess.N)); err != nil {
			return fmt.Errorf("copy build records: %w", err)
		}
	}
	so, se, code, err := b.exec(b.c.EnvPath, "--out", b.c.Out)
	if err != nil || code != 0 {
		return fmt.Errorf("E0 (kpi-env) failed: exit %d %v %s", code, err, tail(se, 400))
	}
	var check map[string]any
	if err := json.Unmarshal(so, &check); err != nil {
		return fmt.Errorf("E0 output is not JSON: %w", err)
	}
	path := filepath.Join(envDir, fmt.Sprintf("e0-s%d.json", b.sess.N))
	if err := writeFileAtomic(path, so); err != nil {
		return err
	}
	return b.journal("e0", "", filepath.Base(path))
}

// ------------------------------------------------------------------ canary

var canaryHeader = []string{"seq", "reason", "start_unix_ns", "end_unix_ns", "median_rate_per_s", "ratio_to_first",
	"core_class", "windows", "window_ms", "rates_per_s", "digest_check", "cpus"}

func (b *batch) canary(reason string) error {
	so, se, code, err := b.exec(b.c.PrimPath, "canary")
	if err != nil || code != 0 {
		if errors.Is(err, errInterrupted) {
			return err
		}
		return fmt.Errorf("canary failed: exit %d %v %s", code, err, tail(se, 400))
	}
	var r proto.CanaryResult
	if err := json.Unmarshal(lastLine(so), &r); err != nil {
		return fmt.Errorf("canary output: %w", err)
	}
	if r.MedianRate <= 0 {
		return errors.New("canary reported no work")
	}
	b.st.CanarySeq++
	if b.st.FirstCanary == 0 {
		b.st.FirstCanary = r.MedianRate
	}
	rec := canaryRec{Seq: b.st.CanarySeq, Reason: reason, CanaryResult: r, Ratio: r.MedianRate / b.st.FirstCanary}
	if b.sess.NativeP > 0 && b.sess.NativeE > 0 {
		if r.MedianRate >= math.Sqrt(b.sess.NativeP*b.sess.NativeE) {
			rec.Core = "P-like"
		} else {
			rec.Core = "E-like"
		}
	}
	if err := appendJSONL(filepath.Join(b.c.Out, fileCanaryJL), rec); err != nil {
		return err
	}
	rates := make([]string, len(r.RatesPerS))
	for i, x := range r.RatesPerS {
		rates[i] = strconv.FormatFloat(x, 'f', 0, 64)
	}
	if err := appendCSV(filepath.Join(b.c.Out, fileCanaryCSV), canaryHeader, []string{
		strconv.Itoa(rec.Seq), reason, strconv.FormatInt(r.StartNS, 10), strconv.FormatInt(r.EndNS, 10),
		strconv.FormatFloat(r.MedianRate, 'f', 0, 64), strconv.FormatFloat(rec.Ratio, 'f', 5, 64), rec.Core,
		strconv.Itoa(r.Windows), strconv.Itoa(r.WindowMS), strings.Join(rates, ";"), r.Digest, intsCSV(r.CPUs)}); err != nil {
		return err
	}
	b.st.LastCanaryNS = r.EndNS
	if err := b.st.save(b.c.Out); err != nil {
		return err
	}
	logf("canary %d (%s): %.0f hashes/s, ratio %.4f %s", rec.Seq, reason, r.MedianRate, rec.Ratio, rec.Core)
	return b.journal("canary", "", fmt.Sprintf("%d %s ratio %.5f %s", rec.Seq, reason, rec.Ratio, rec.Core))
}

func intsCSV(xs []int) string {
	s := make([]string, len(xs))
	for i, x := range xs {
		s[i] = strconv.Itoa(x)
	}
	return strings.Join(s, " ")
}

// -------------------------------------------------------------- checkpoint

func (b *batch) checkpoint() (int, error, bool) {
	if err := b.canary("periodic"); err != nil {
		code, err := b.stopOrFail(err)
		return code, err, true
	}
	added, err := b.evaluate(false)
	if err != nil {
		code, err := b.fail(exitFailed, err)
		return code, err, true
	}
	if b.strict {
		if stale, why := b.monitorStale(); stale {
			return b.stopped(why), nil, true
		}
	}
	b.journal("checkpoint", "", fmt.Sprintf("%d block(s) queued; %d remaining", added, len(b.st.Queue)))
	return 0, nil, false
}

// newestHostNS returns the newest host sample on the container clock (0 if none).
func (b *batch) newestHostNS() int64 {
	s, _, err := quality.ReadHostChunks(filepath.Join(b.c.Out, dirMonitor))
	if err != nil || len(s) == 0 {
		return 0
	}
	return s[len(s)-1].UnixNS - b.sess.HostOffsetNS
}

func (b *batch) monitorStale() (bool, string) {
	limit := time.Duration(b.c.MonitorStaleS) * time.Second
	if time.Since(time.Unix(0, b.sess.StartNS)) < limit {
		return false, ""
	}
	newest := b.newestHostNS()
	if newest == 0 || time.Since(time.Unix(0, newest)) > limit {
		return true, fmt.Sprintf("host monitor stale (no host sample in the last %s)", limit)
	}
	return false, ""
}

// waitCoverage waits until host samples cover the last block (trial/main).
func (b *batch) waitCoverage() (int, error, bool) {
	if !b.strict || b.lastEnd == 0 {
		return 0, nil, false
	}
	deadline := time.Now().Add(time.Duration(b.c.CoverWaitS) * time.Second)
	for time.Now().Before(deadline) {
		if b.newestHostNS() >= b.lastEnd {
			return 0, nil, false
		}
		if code, err, stop := b.checkStop(); stop {
			return code, err, true
		}
		time.Sleep(5 * time.Second)
	}
	b.journal("cover_timeout", "", "host samples do not cover the last block")
	logf("warning: host samples do not cover the last block after %ds", b.c.CoverWaitS)
	return 0, nil, false
}

// evaluate applies the quality rules, queues repeats and diagnostic re-runs,
// and rewrites quality.csv. It returns the number of blocks queued.
func (b *batch) evaluate(final bool) (int, error) {
	in, err := loadEval(b.c.Out, b.st, b.th, final)
	if err != nil {
		return 0, err
	}
	vs := evaluate(in)
	ds := decideBase(b.p.Kind, vs, b.c.Seed, b.st.Requeued, b.st.Diagnosed, b.st.Base)
	for _, d := range ds {
		d.Block.Mode = ""
		pos := insertAt(b.c.Seed, d.Block.Key(), len(b.st.Queue))
		q := make([]plan.Block, 0, len(b.st.Queue)+1)
		q = append(q, b.st.Queue[:pos]...)
		q = append(q, d.Block)
		q = append(q, b.st.Queue[pos:]...)
		b.st.Queue = q
		logf("queued %s (%s) at position %d", d.Block.Key(), d.Reason, pos)
		if err := b.journal("queued", d.Block.Key(), fmt.Sprintf("%s; position %d", d.Reason, pos)); err != nil {
			return 0, err
		}
	}
	if err := writeCSV(filepath.Join(b.c.Out, fileQuality), qualityHeader, qualityRows(vs)); err != nil {
		return 0, err
	}
	return len(ds), b.st.save(b.c.Out)
}

// ------------------------------------------------------------------ blocks

func (b *batch) modeFor(bl plan.Block) (string, error) {
	if proto.IsProbe(bl.Exp) {
		return "", nil
	}
	if bl.Exp == "E2c" || proto.IsE3Null(bl.Exp, bl.Alg, bl.Op) {
		// The components (hex coding, object creation and destruction) are
		// all short; batching keeps every sample above the timer's resolution.
		return string(timing.Fast), nil
	}
	if proto.SizeOnly(bl.Exp) {
		return "sizes", nil
	}
	if bl.Exp == "E9" {
		return "memory", nil
	}
	if bl.Exp == "E12" {
		return "window", nil // fixed warm-up and window: nothing to classify
	}
	k := bl.Alg + "/" + bl.Op
	var m string
	if b.p.Kind == plan.KindMain {
		m = b.th.Class[k]
	} else {
		m = b.st.Class[k]
	}
	if m != string(timing.Fast) && m != string(timing.Slow) {
		return "", fmt.Errorf("no fast/slow classification for %s (main plans take it from the thresholds; trial and smoke plans from their own probe pass)", k)
	}
	return m, nil
}

func (b *batch) batchFor(bl plan.Block) map[string]int {
	if b.p.Kind != plan.KindMain || proto.IsProbe(bl.Exp) || proto.Untimed(bl.Exp) || bl.Exp == "E11" || bl.Exp == "E12" {
		return nil // E11 composites are far above the timer's resolution
	}
	out := map[string]int{}
	for _, arm := range proto.ArmsOfBlock(bl.Exp, bl.Alg, bl.Op) {
		if n := b.th.Batch[bl.Alg+"/"+bl.Op+"/"+arm]; n > 1 {
			out[arm] = n
		}
	}
	return out
}

var prepHeader = []string{"utc_ns", "session", "alg", "rep", "pool", "e10", "prep_ns", "fingerprint"}

// ensurePrep generates the input set of a block if it does not exist yet.
func (b *batch) ensurePrep(bl plan.Block) error {
	if proto.SizeOnly(bl.Exp) || bl.Exp == "E11" || bl.Exp == "probe-E11" || bl.Exp == "E12" || proto.IsE3Null(bl.Exp, bl.Alg, bl.Op) {
		return nil // E8, E11 and E12 blocks prepare their own inputs; E3's null block needs none
	}
	rep := bl.Rep
	if proto.IsProbe(bl.Exp) {
		rep = proto.ProbeRep
	}
	path := proto.PrepPath(b.c.PrepDir, bl.Alg, rep)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	a := algs.MustByID(bl.Alg)
	pool := a.PoolSize()
	switch {
	case rep == proto.ProbeRep:
		pool = proto.ProbeCalls + 1
	case b.p.Kind == plan.KindSmoke:
		pool = 8
	case bl.Exp == "E9":
		pool = proto.E9PoolSize(a)
	}
	e10 := rep != proto.ProbeRep && b.e10Algs[bl.Alg]
	seed := plan.DeriveSeed(b.c.Seed, "prep."+proto.PrepName(bl.Alg, rep))
	args := []string{"prep", "--alg", bl.Alg, "--rep", strconv.Itoa(rep), "--pool", strconv.Itoa(pool),
		"--seed", strconv.FormatUint(seed, 10), "--dir", b.c.PrepDir}
	if e10 {
		args = append(args, "--e10")
	}
	if b.e6Algs[bl.Alg] {
		args = append(args, "--e6")
	}
	so, se, code, err := b.exec(b.c.PrimPath, args...)
	if err != nil || code != 0 {
		if errors.Is(err, errInterrupted) {
			return err
		}
		msg := fmt.Sprintf("prep %s failed: exit %d %v %s", proto.PrepName(bl.Alg, rep), code, err, tail(se, 600))
		if code == proto.ExitCorrectness {
			return correctnessError{msg}
		}
		return errors.New(msg)
	}
	var pr proto.PrepResult
	if err := json.Unmarshal(lastLine(so), &pr); err != nil {
		return fmt.Errorf("prep output: %w", err)
	}
	b.st.Preps[proto.PrepName(bl.Alg, rep)] = PrepRec{Pool: pr.Pool, PrepNS: pr.PrepNS, Fingerprint: pr.Fingerprint, E10: pr.E10, Session: b.sess.N}
	if err := appendCSV(filepath.Join(b.c.Out, filePrepCSV), prepHeader, []string{strconv.FormatInt(time.Now().UnixNano(), 10),
		strconv.Itoa(b.sess.N), pr.Alg, strconv.Itoa(pr.Rep), strconv.Itoa(pr.Pool), strconv.FormatBool(pr.E10),
		strconv.FormatInt(pr.PrepNS, 10), pr.Fingerprint}); err != nil {
		return err
	}
	logf("prepared %s: P=%d in %.1fs", proto.PrepName(bl.Alg, rep), pr.Pool, float64(pr.PrepNS)/1e9)
	return b.journal("prep", proto.PrepName(bl.Alg, rep), fmt.Sprintf("pool %d e10 %v %d ns", pr.Pool, pr.E10, pr.PrepNS))
}

func (b *batch) runBlock(bl plan.Block) error {
	mode, err := b.modeFor(bl)
	if err != nil {
		return err
	}
	bl.Mode = mode
	if err := b.ensurePrep(bl); err != nil {
		return err
	}
	spec := proto.Spec{Block: bl, PrepDir: b.c.PrepDir, RawFile: filepath.Join(b.c.Out, bl.RawPath()),
		Batch: b.batchFor(bl), MaxWall: b.c.MaxBlockS}
	js, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := b.journal("block_start", bl.Key(), mode); err != nil {
		return err
	}
	var env []string
	if bl.Exp == "E9" {
		env = []string{"LD_PRELOAD=" + b.c.MallocLib}
	}
	so, se, code, err := b.execEnv(env, b.c.PrimPath, "block", "--spec", string(js))
	if errors.Is(err, errInterrupted) {
		b.journal("block_interrupted", bl.Key(), "")
		return err
	}
	if err != nil || code != 0 {
		msg := fmt.Sprintf("block %s failed: exit %d %v %s", bl.Key(), code, err, tail(se, 800))
		b.journal("block_failed", bl.Key(), msg)
		if code == proto.ExitCorrectness {
			return correctnessError{msg}
		}
		return errors.New(msg)
	}
	var res proto.Result
	if err := json.Unmarshal(lastLine(so), &res); err != nil {
		return fmt.Errorf("block %s output: %w", bl.Key(), err)
	}
	if res.Key != bl.Key() {
		return fmt.Errorf("block %s reported key %s", bl.Key(), res.Key)
	}
	if res.Truncated {
		msg := fmt.Sprintf("block %s hit the %ds safety limit before meeting its budget", bl.Key(), b.c.MaxBlockS)
		b.journal("block_failed", bl.Key(), msg)
		return errors.New(msg)
	}
	if got, err := fileHash(spec.RawFile); err != nil || got != res.RawSHA256 {
		return fmt.Errorf("block %s: raw file missing or altered (%v)", bl.Key(), err)
	}
	if err := appendJSONL(filepath.Join(b.c.Out, fileBlocksJL), res); err != nil {
		return err
	}
	for _, row := range blockRows(res) {
		if err := appendCSV(filepath.Join(b.c.Out, fileBlocksCSV), blocksHeader, row); err != nil {
			return err
		}
	}
	if bl.Exp == "probe" && b.p.Kind != plan.KindMain && len(res.Arms) == 1 {
		b.st.Class[bl.Alg+"/"+bl.Op] = quality.Classify(res.Arms[0].ProbeNS)
	}
	if bl.Exp == "probe-E6" && b.p.Kind != plan.KindMain && len(res.Arms) == 1 {
		b.st.Class[bl.Alg+"/"+bl.Op] = quality.Classify(res.Arms[0].ProbeNS)
	}
	if bl.Exp == "probe-E7" && b.p.Kind != plan.KindMain && len(res.Arms) > 0 {
		// E7's classification is its operation's (the valid arm; main runs use E1's).
		b.st.Class[bl.Alg+"/"+bl.Op] = quality.Classify(res.Arms[0].ProbeNS)
	}
	if bl.Exp == "probe-E11" && b.p.Kind != plan.KindMain && len(res.Arms) > 0 {
		b.st.Class[bl.Alg+"/"+bl.Op] = quality.ClassifyComposite(res.Arms[0].ProbeNS, cells.SlowCompositeNS)
	}
	b.lastEnd = res.EndNS
	b.st.Queue = b.st.Queue[1:]
	b.st.Done = append(b.st.Done, bl.Key())
	if err := b.st.save(b.c.Out); err != nil {
		return err
	}
	if err := b.journal("block_done", bl.Key(), ""); err != nil {
		return err
	}
	a0 := res.Arms[0]
	if bl.Exp == "E12" && len(res.Arms) >= 2 {
		lat, tp := res.Arms[0].Stats, res.Arms[1].Stats
		logf("%d/%d %s throughput %.1f/s, latency p50 %s p99 %s (n=%d)", len(b.st.Done), len(b.st.Done)+len(b.st.Queue), bl.Key(),
			tp.Mean, fmtNS(lat.P50), fmtNS(lat.P99), lat.N)
		return nil
	}
	if bl.Exp == "E9" {
		var parts []string
		for _, a := range res.Arms {
			if !proto.E9PerBlock(a.Name) {
				parts = append(parts, fmt.Sprintf("%s %.0f", a.Name, a.Stats.P50))
			}
		}
		logf("%d/%d %s memory %s (n=%d; first call reported separately)", len(b.st.Done), len(b.st.Done)+len(b.st.Queue), bl.Key(),
			strings.Join(parts, ", "), a0.Stats.N)
		return nil
	}
	if proto.SizeOnly(bl.Exp) {
		var parts []string
		for _, a := range res.Arms {
			if a.MinBytes == a.MaxBytes {
				parts = append(parts, fmt.Sprintf("%s %d B", a.Name, a.MinBytes))
			} else {
				parts = append(parts, fmt.Sprintf("%s %d-%d B (median %.0f)", a.Name, a.MinBytes, a.MaxBytes, a.Stats.P50))
			}
		}
		logf("%d/%d %s sizes %s (n=%d)", len(b.st.Done), len(b.st.Done)+len(b.st.Queue), bl.Key(), strings.Join(parts, ", "), a0.Stats.N)
		return nil
	}
	logf("%d/%d %s %s p50 %s (n=%d) first %s", len(b.st.Done), len(b.st.Done)+len(b.st.Queue), bl.Key(), modeLabel(mode),
		fmtNS(a0.Stats.P50), a0.Stats.N, fmtNS(float64(a0.FirstNS)))
	return nil
}

func modeLabel(m string) string {
	if m == "" {
		return "probe"
	}
	return m
}

func fmtNS(ns float64) string {
	switch {
	case ns >= 1e9:
		return strconv.FormatFloat(ns/1e9, 'f', 3, 64) + "s"
	case ns >= 1e6:
		return strconv.FormatFloat(ns/1e6, 'f', 3, 64) + "ms"
	case ns >= 1e3:
		return strconv.FormatFloat(ns/1e3, 'f', 2, 64) + "us"
	}
	return strconv.FormatFloat(ns, 'f', 0, 64) + "ns"
}

var blocksHeader = []string{"key", "id", "exp", "alg", "op", "rep", "attempt", "diagnostic", "profile", "mode", "seed",
	"arm", "batch", "first_label", "first_ns", "n", "p50_ns", "mean_ns", "sd_ns", "cov", "p90_ns", "p95_ns", "p99_ns",
	"p999_ns", "min_ns", "max_ns", "boot_se_p50_ns", "sum_ns", "min_bytes", "max_bytes", "heap_bytes_per_op",
	"allocs_per_op", "wall_ns", "start_unix_ns", "end_unix_ns", "gc_forced", "gc_automatic", "steal_pct",
	"vm_other_pct", "vm_busy_pct", "nr_throttled", "throttled_usec", "pool", "prep_id", "keys_checked", "rows",
	"raw_file", "raw_sha256"}

func ff(x float64) string { return strconv.FormatFloat(x, 'g', 10, 64) }

func blockRows(r proto.Result) [][]string {
	b := r.Block
	var rows [][]string
	for _, a := range r.Arms {
		s := a.Stats
		rows = append(rows, []string{r.Key, b.ID, b.Exp, b.Alg, b.Op, strconv.Itoa(b.Rep), strconv.Itoa(b.Attempt),
			strconv.FormatBool(b.Diagnostic), b.Profile, modeLabel(b.Mode), strconv.FormatUint(b.Seed, 10),
			a.Name, strconv.Itoa(a.Batch), a.FirstLabel, strconv.FormatInt(a.FirstNS, 10), strconv.Itoa(s.N),
			ff(s.P50), ff(s.Mean), ff(s.SD), ff(s.CoV), ff(s.P90), ff(s.P95), ff(s.P99), ff(s.P999), ff(s.Min), ff(s.Max),
			ff(s.BootSE50), strconv.FormatInt(a.SumNS, 10), strconv.Itoa(a.MinBytes), strconv.Itoa(a.MaxBytes),
			ff(a.HeapBytesOp), ff(a.AllocsOp), strconv.FormatInt(r.WallNS, 10), strconv.FormatInt(r.StartNS, 10),
			strconv.FormatInt(r.EndNS, 10), strconv.Itoa(r.GCForced), strconv.FormatUint(r.GCAuto, 10),
			ff(r.Quality.StealPct), ff(r.Quality.OtherPct), ff(r.Quality.VMBusyPct), strconv.FormatUint(r.Quality.NrThrottled, 10),
			strconv.FormatUint(r.Quality.ThrottledUS, 10), strconv.Itoa(r.Pool), r.PrepID, strconv.Itoa(r.KeysChecked),
			strconv.Itoa(r.Rows), b.RawPath(), r.RawSHA256})
	}
	return rows
}

// ------------------------------------------------------------------ finish

func (b *batch) finish() (int, error) {
	b.st.Status, b.st.StatusDetail = "done", ""
	if err := b.journal("batch_done", "", fmt.Sprintf("%d attempt(s)", len(b.st.Done))); err != nil {
		return b.fail(exitFailed, err)
	}
	if err := b.st.save(b.c.Out); err != nil {
		return b.fail(exitFailed, err)
	}
	if b.p.Kind == plan.KindTrial {
		if err := calibrateRun(b.c.Out, b.c.Machine, b.th); err != nil {
			logf("calibration failed: %v", err)
			return b.fail(exitFailed, fmt.Errorf("calibration: %w", err))
		}
		if err := estimateAll(b.c.Out, filepath.Dir(b.c.PlanPath), b.c.Machine); err != nil {
			logf("estimate failed: %v", err)
			return b.fail(exitFailed, fmt.Errorf("estimate: %w", err))
		}
	}
	if err := b.finalManifest(); err != nil {
		return b.fail(exitFailed, err)
	}
	n, err := writeSums(b.c.Out)
	if err != nil {
		return b.fail(exitFailed, err)
	}
	logf("batch complete: %d attempt(s); %d file(s) in SHA256SUMS", len(b.st.Done), n)
	return exitDone, nil
}

func writeFileAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeBytes(path, b)
}
