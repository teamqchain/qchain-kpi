package main

// E9: memory. One block per (algorithm, operation, route) makes up to
// E9Budget calls in a fresh process and records per call what its route can
// see (internal/proto/e9.go), plus the process's peak resident memory over
// the calls. Nothing is timed. The block runs with the C-heap counter
// (csrc/kpi_malloc.c) preloaded; it refuses to run without it.
//
//   - c: the C library called directly on memprobe's runner thread, whose
//     8 MiB stack is painted before every call: the stack high-water mark
//     (above the runner's empty-job baseline) and the call's C heap. Key
//     objects are kept, as in the reuse arm.
//   - reuse / prod: E1's reuse arm and E2's prod arm (QChain's production
//     pattern), called from this goroutine locked to its OS thread, so the
//     C-heap counters of that thread are exactly the cgo calls' allocations;
//     the Go heap from runtime.MemStats around each call.
//
// GC: Go's default (production-like). The Go heap a call allocates is
// counted whether or not a collection runs, and the resident memory then
// reflects a production process rather than garbage left uncollected.

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/memprobe"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/record"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/stats"
	"qchain-kpi/internal/timing"
)

// memJob is a direct C call prepared by internal/pqc or internal/classical.
type memJob interface {
	Call() (fn, arg unsafe.Pointer)
	SetInputs(in1, in2 []byte)
	Outputs() (out1, out2 []byte)
	Close()
}

func newMemJob(a algs.Alg, op string, p *inputs.Prep) (memJob, error) {
	if a.Classical() {
		m, err := classical.NewMemJob(a, op, p.SK, p.PK)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	m, err := pqc.NewMemJob(a, op, p.SK, p.PK)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// e9Row is one call.
type e9Row struct {
	stack    int64 // c route: bytes above the baseline
	heap     memprobe.Use
	goBytes  uint64 // reuse / prod routes
	goAllocs uint64
}

// e9Run is what a block measured.
type e9Run struct {
	rows               []e9Row
	baseline           int64 // c route: the empty job's stack measurement
	rssBefore, rssPeak int64
	keysChecked        int
}

func e9Block(sp proto.Spec, a algs.Alg) error {
	b := sp.Block
	op, route, err := proto.ParseE9Op(b.Op)
	if err != nil {
		return err
	}
	if !a.HasOp(op) {
		return fmt.Errorf("block: %s has no operation %q", a.ID, op)
	}
	if !memprobe.Counting() {
		return fmt.Errorf("block: E9 needs the C-heap counter loaded (LD_PRELOAD=%s)", proto.E9MallocLib)
	}
	lib := os.Getenv("LD_PRELOAD")
	libSum, err := fileSHA256(lib)
	if err != nil {
		return fmt.Errorf("block: heap counter %q: %w", lib, err)
	}
	bud, err := proto.E9BudgetFor(b.Profile)
	if err != nil {
		return err
	}
	oomSet := procsetup.OOMAdj()
	p, err := inputs.Load(proto.PrepPath(sp.PrepDir, a.ID, b.Rep))
	if err != nil {
		return err
	}
	if p.Alg != a.ID || p.Rep != b.Rep {
		return fmt.Errorf("block: prep is for %s rep %d, not %s rep %d", p.Alg, p.Rep, a.ID, b.Rep)
	}

	// Every C call of the Go routes, and every counter read, on this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cpus, _ := quality.CPUSet()
	q0, _ := quality.Take()
	startNS := time.Now().UnixNano()
	t0 := time.Now()
	var r e9Run
	if route == proto.ArmC {
		err = e9C(a, op, p, b.Seed, bud, &r)
	} else {
		err = e9Go(a, op, route, p, b.Seed, bud, &r)
	}
	wall := time.Since(t0).Nanoseconds()
	endNS := time.Now().UnixNano()
	q1, _ := quality.Take()
	if err != nil {
		return err
	}

	// Raw file: one row per call.
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	header := map[string]string{
		"key": b.Key(), "id": b.ID, "exp": b.Exp, "alg": b.Alg, "op": b.Op,
		"rep": strconv.Itoa(b.Rep), "attempt": strconv.Itoa(b.Attempt), "profile": b.Profile,
		"seed": strconv.FormatUint(b.Seed, 10), "route": route, "operation": op,
		"pool": strconv.Itoa(p.P), "prep_id": p.Fingerprint,
		"budget": fmt.Sprintf("target %d calls, at least %d, stop after %s once the minimum is reached; call 1 is reported on its own (first)",
			bud.Target, bud.Min, bud.TimeBox),
		"heap_counter": "csrc/kpi_malloc.c via LD_PRELOAD, sha256 " + libSum + "; per-thread counters; heap bytes are malloc_usable_size, heap_requested what callers asked for",
		"rss_before":   strconv.FormatInt(r.rssBefore, 10),
		"rss_peak":     strconv.FormatInt(r.rssPeak, 10),
		"rss": "peak resident set (VmHWM, reset with /proc/self/clear_refs after setup) over the calls and their correctness checks; " +
			"Linux >= 6.2 reads it from per-CPU counters, precise to about 32 pages per CPU the process ran on",
		"gc": "Go default (production-like)",
		"columns": "call = 1-based call number; first = 1 for call 1; stack = bytes (c route, above the baseline); " +
			"heap_* = the call's C heap on the measuring thread (peak above the start, usable bytes allocated, bytes requested, allocations, frees, retained after the call); " +
			"go_* = Go heap bytes and objects allocated by the call (reuse and prod routes)",
		"harness_commit": buildinfo.HarnessCommit, "go": runtime.Version(), "liboqs": pqc.Version(),
		"openssl": classical.OpenSSLVersion(), "start_unix_ns": strconv.FormatInt(startNS, 10),
		"cpus": intsCSV(cpus),
	}
	if route == proto.ArmC {
		header["stack"] = fmt.Sprintf("runner thread with a %d-byte stack painted before every call; baseline (empty job) %d bytes, subtracted",
			memprobe.StackSize, r.baseline)
		header["stack_baseline"] = strconv.FormatInt(r.baseline, 10)
	}
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(gz, "# %s=%s\n", k, strings.ReplaceAll(header[k], "\n", " "))
	}
	fmt.Fprintln(gz, "call,first,stack,heap_peak,heap_bytes,heap_requested,heap_allocs,heap_frees,heap_retained,go_bytes,go_allocs")
	for i, row := range r.rows {
		first := 0
		if i == 0 {
			first = 1
		}
		stack, goBytes, goAllocs := "", "", ""
		if route == proto.ArmC {
			stack = strconv.FormatInt(row.stack, 10)
		} else {
			goBytes, goAllocs = strconv.FormatUint(row.goBytes, 10), strconv.FormatUint(row.goAllocs, 10)
		}
		h := row.heap
		fmt.Fprintf(gz, "%d,%d,%s,%d,%d,%d,%d,%d,%d,%s,%s\n", i+1, first, stack,
			h.Peak, h.Bytes, h.Requested, h.Allocs, h.Frees, h.Retained, goBytes, goAllocs)
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := record.WriteFileAtomic(sp.RawFile, buf.Bytes(), 0o644); err != nil {
		return err
	}
	sum, err := fileSHA256(sp.RawFile)
	if err != nil {
		return err
	}

	out := proto.Result{
		Key: b.Key(), Block: b, StartNS: startNS, EndNS: endNS, WallNS: wall,
		Budget:  proto.BudgetInfo{Mode: "memory", Samples: bud.Target, MinSamples: bud.Min, TimeBoxS: bud.TimeBox.Seconds()},
		Quality: quality.Diff(q0, q1, cpus), CPUs: cpus, GOMAXPROCS: runtime.GOMAXPROCS(0), OOMAdj: oomSet,
		Pool: p.P, PrepID: p.Fingerprint, KeysChecked: r.keysChecked, Rows: len(r.rows), RawSHA256: sum,
	}
	out.Arms = e9Summaries(route, &r)
	return emit(out)
}

// e9Summaries summarises every quantity of the route: call 1 on its own
// (first), the statistics over the calls after it; per-block quantities have
// one value.
func e9Summaries(route string, r *e9Run) []proto.ArmSummary {
	var out []proto.ArmSummary
	for _, q := range proto.E9Quantities(route) {
		if proto.E9PerBlock(q) {
			v := r.rssPeak
			if q == proto.QRSSGrowth {
				v = r.rssPeak - r.rssBefore
			}
			out = append(out, proto.ArmSummary{Name: q, Batch: 1, Stats: stats.Summarize([]float64{float64(v)}, 0, 0),
				MinBytes: int(v), MaxBytes: int(v)})
			continue
		}
		vals := make([]float64, len(r.rows))
		for i, row := range r.rows {
			vals[i] = e9Value(q, row)
		}
		rest := vals[1:]
		lo, hi := rest[0], rest[0]
		for _, v := range rest {
			lo, hi = min(lo, v), max(hi, v)
		}
		out = append(out, proto.ArmSummary{Name: q, Batch: 1, FirstLabel: "first", FirstNS: int64(vals[0]),
			Stats: stats.Summarize(rest, 0, 0), MinBytes: int(lo), MaxBytes: int(hi)})
	}
	return out
}

func e9Value(q string, row e9Row) float64 {
	switch q {
	case proto.QStack:
		return float64(row.stack)
	case proto.QHeapPeak:
		return float64(row.heap.Peak)
	case proto.QHeapBytes:
		return float64(row.heap.Bytes)
	case proto.QHeapAllocs:
		return float64(row.heap.Allocs)
	case proto.QHeapUnfreed:
		return float64(row.heap.Allocs - row.heap.Frees)
	case proto.QHeapRetained:
		return float64(row.heap.Retained)
	case proto.QGoBytes:
		return float64(row.goBytes)
	case proto.QGoAllocs:
		return float64(row.goAllocs)
	}
	panic("e9: unknown quantity " + q)
}

// e9More is E9's sampling rule.
func e9More(bud proto.E9Budget, t0 time.Time) func(n int) bool {
	return func(n int) bool {
		if n >= bud.Target {
			return false
		}
		return n < bud.Min || time.Since(t0) < bud.TimeBox
	}
}

// e9Inputs are call i's inputs from the pool: the message (sign, verify) or
// ciphertext (decaps), and the signature (verify).
func e9Inputs(op string, p *inputs.Prep, i int) (in1, in2 []byte) {
	switch op {
	case "sign":
		return p.Msgs[i%p.P], nil
	case "verify":
		return p.Msgs[i%p.P], p.Sigs[i%p.P]
	case "decaps":
		return p.CTs[i%p.P], nil
	}
	return nil, nil
}

// ------------------------------------------------------------------ c route

func e9C(a algs.Alg, op string, p *inputs.Prep, seed uint64, bud proto.E9Budget, r *e9Run) error {
	check, after, err := e9CChecks(a, op, p, seed)
	if err != nil {
		return err
	}
	job, err := newMemJob(a, op, p)
	if err != nil {
		return err
	}
	defer job.Close()
	run, err := memprobe.NewRunner(memprobe.StackSize)
	if err != nil {
		return err
	}
	defer run.Close()
	fn, arg := job.Call()
	j := memprobe.Job{Fn: fn, Arg: arg}

	// The first job paints the whole stack, before the resident-set reset, so
	// the painted pages do not count as growth.
	base := run.Run(memprobe.Empty)
	if again := run.Run(memprobe.Empty); again.Stack != base.Stack {
		return fmt.Errorf("block: runner baseline unstable (%d, then %d bytes)", base.Stack, again.Stack)
	}
	r.baseline = base.Stack
	if err := memprobe.ResetPeakRSS(); err != nil {
		return fmt.Errorf("block: reset peak RSS: %w", err)
	}
	if r.rssBefore, _, err = memprobe.RSS(); err != nil {
		return err
	}
	more := e9More(bud, time.Now())
	for i := 0; more(i); i++ {
		job.SetInputs(e9Inputs(op, p, i))
		res := run.Run(j)
		out1, out2 := job.Outputs()
		if err := check(i, res.RC, out1, out2); err != nil {
			return fmt.Errorf("%w: %s %s call %d: %v", timing.ErrCheck, a.ID, op, i+1, err)
		}
		r.rows = append(r.rows, e9Row{stack: res.Stack - base.Stack, heap: res.Heap})
	}
	if _, r.rssPeak, err = memprobe.RSS(); err != nil {
		return err
	}
	if r.keysChecked, err = after(); err != nil {
		return fmt.Errorf("%w: %v", timing.ErrCheck, err)
	}
	return nil
}

// e9CChecks returns the correctness check of every direct call (run on this
// thread, after the call) and the deferred check of a key subset (keygen).
func e9CChecks(a algs.Alg, op string, p *inputs.Prep, seed uint64) (func(i, rc int, out1, out2 []byte) error, func() (int, error), error) {
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return nil, nil, err
		}
		return e9SigChecks(s, op, p, seed)
	}
	k, err := newKEM(a)
	if err != nil {
		return nil, nil, err
	}
	return e9KEMChecks(k, op, p, seed)
}

func e9SigChecks(s scheme.Signature, op string, p *inputs.Prep, seed uint64) (func(int, int, []byte, []byte) error, func() (int, error), error) {
	sizes := s.Sizes()
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		return func(i, rc int, pk, sk []byte) error {
			if rc != 1 {
				return fmt.Errorf("status %d", rc)
			}
			if !sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("key lengths %d/%d, want %+v", len(pk), len(sk), sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		}, exerciseSig(s, sub), nil
	case "sign":
		return func(i, rc int, sg, _ []byte) error {
			if rc != 1 {
				return fmt.Errorf("status %d", rc)
			}
			if err := sigLengthOK(sizes, sg); err != nil {
				return err
			}
			if ok, err := s.Verify(p.PK, p.Msgs[i%p.P], sg); err != nil || !ok {
				return fmt.Errorf("signature does not verify (%v)", err)
			}
			return nil
		}, noAfter, nil
	case "verify":
		return func(i, rc int, _, _ []byte) error {
			if rc != 1 {
				return fmt.Errorf("valid signature %d rejected (status %d)", i%p.P, rc)
			}
			return nil
		}, noAfter, nil
	}
	return nil, nil, fmt.Errorf("%s has no operation %q", s.Alg().ID, op)
}

func e9KEMChecks(k scheme.KEM, op string, p *inputs.Prep, seed uint64) (func(int, int, []byte, []byte) error, func() (int, error), error) {
	sizes := k.Sizes()
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		return func(i, rc int, pk, sk []byte) error {
			if rc != 1 {
				return fmt.Errorf("status %d", rc)
			}
			if !sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("key lengths %d/%d, want %+v", len(pk), len(sk), sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		}, exerciseKEM(k, sub), nil
	case "encaps":
		return func(i, rc int, ct, ss []byte) error {
			if rc != 1 {
				return fmt.Errorf("status %d", rc)
			}
			if len(ct) != sizes.Ciphertext || len(ss) != sizes.SharedSecret {
				return fmt.Errorf("ciphertext/secret lengths %d/%d", len(ct), len(ss))
			}
			ss2, err := k.Decaps(p.SK, p.PK, ct)
			if err != nil || !equalBytes(ss, ss2) {
				return fmt.Errorf("encapsulation does not decapsulate to the same secret (%v)", err)
			}
			return nil
		}, noAfter, nil
	case "decaps":
		return func(i, rc int, ss, _ []byte) error {
			if rc != 1 {
				return fmt.Errorf("status %d", rc)
			}
			if !equalBytes(ss, p.SSs[i%p.P]) {
				return fmt.Errorf("decapsulation %d returned the wrong secret", i%p.P)
			}
			return nil
		}, noAfter, nil
	}
	return nil, nil, fmt.Errorf("%s has no operation %q", k.Alg().ID, op)
}

// --------------------------------------------------------- reuse/prod routes

func e9Go(a algs.Alg, op, route string, p *inputs.Prep, seed uint64, bud proto.E9Budget, r *e9Run) error {
	var set *armSet
	var err error
	switch route {
	case proto.ArmReuse:
		set, err = e1Arms(a, op, p, seed)
	case proto.ArmProd:
		set, err = e9ProdArm(a, op, p, seed)
	default:
		err = fmt.Errorf("block: E9 route %q", route)
	}
	if err != nil {
		return err
	}
	defer set.close()
	if err := checkArmOrder(set, []string{route}); err != nil {
		return err
	}
	arm, reset := set.arms[0], set.resetOf(0)

	if err := memprobe.ResetPeakRSS(); err != nil {
		return fmt.Errorf("block: reset peak RSS: %w", err)
	}
	if r.rssBefore, _, err = memprobe.RSS(); err != nil {
		return err
	}
	var m0, m1 runtime.MemStats
	more := e9More(bud, time.Now())
	for i := 0; more(i); i++ {
		runtime.ReadMemStats(&m0)
		memprobe.ResetPeak()
		h0 := memprobe.ThreadHeap()
		n, err := arm.Call(i)
		h1 := memprobe.ThreadHeap()
		runtime.ReadMemStats(&m1)
		if err != nil {
			return err
		}
		if arm.Check != nil {
			if err := arm.Check(i, n); err != nil {
				return fmt.Errorf("%w: %s %s call %d: %v", timing.ErrCheck, a.ID, op, i+1, err)
			}
		}
		if reset != nil {
			if err := reset(); err != nil {
				return err
			}
		}
		r.rows = append(r.rows, e9Row{heap: memprobe.Delta(h0, h1),
			goBytes: m1.TotalAlloc - m0.TotalAlloc, goAllocs: m1.Mallocs - m0.Mallocs})
	}
	if _, r.rssPeak, err = memprobe.RSS(); err != nil {
		return err
	}
	if r.keysChecked, err = set.after(); err != nil {
		return fmt.Errorf("%w: %v", timing.ErrCheck, err)
	}
	return nil
}

// e9ProdArm is E2's prod arm alone.
func e9ProdArm(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return nil, err
		}
		return sigProdArm(s, op, newE2Inputs(a, p, s.Sizes()), seed)
	}
	k, err := newKEM(a)
	if err != nil {
		return nil, err
	}
	return kemProdArm(k, op, newE2Inputs(a, p, k.Sizes()), seed)
}
