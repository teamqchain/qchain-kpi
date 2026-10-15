package main

// E12: concurrency. One block per (suite, stage, W): W workers, each with its
// own stage prepared by internal/flow (E11's copied QChain code and its own
// inputs), call the stage back to back in one process with GOMAXPROCS = W and
// Go's default GC (production-like), for a warm-up and then a fixed window
// (internal/conc). Reported: throughput (calls completed in the window per
// second, and per second of the window), the latency distribution of those
// calls (histogram), and the process's peak resident memory over the run.
// Every warm-up call is checked; inside the window every CheckEvery-th call
// of each worker (the checks take worker time, as they would in production).

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/conc"
	"qchain-kpi/internal/flow"
	"qchain-kpi/internal/memprobe"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/record"
	"qchain-kpi/internal/stats"
)

func e12Block(sp proto.Spec) error {
	b := sp.Block
	stageOp, w, err := proto.ParseE12Op(b.Op)
	if err != nil {
		return fmt.Errorf("block: %w", err)
	}
	c, err := cells.ParseCell(b.Alg, stageOp)
	if err != nil {
		return fmt.Errorf("block: %w", err)
	}
	cpus, cpuErr := quality.CPUSet()
	if cpuErr == nil && w > len(cpus) {
		return fmt.Errorf("block %s: %d workers but the container has %d CPU(s) %v", b.Key(), w, len(cpus), cpus)
	}
	bud, err := proto.E12BudgetFor(b.Profile)
	if err != nil {
		return err
	}
	oomSet := procsetup.OOMAdj()
	runtime.GOMAXPROCS(w)

	// Every worker's own stage, prepared one after the other (untimed).
	t0 := time.Now()
	setups := make([]*flow.Setup, w)
	workers := make([]conc.Worker, w)
	for k := range setups {
		st, err := flow.Prepare(c, plan.DeriveSeed(b.Seed, fmt.Sprintf("worker%d", k)))
		if err != nil {
			return fmt.Errorf("block %s: prepare worker %d: %w", b.Key(), k, err)
		}
		if len(st.Arms) == 0 || st.Arms[0].Name != cells.ArmStage {
			return fmt.Errorf("block %s: worker %d has no stage arm", b.Key(), k)
		}
		setups[k] = st
		workers[k] = conc.Worker{Call: st.Arms[0].Call, Check: st.Arms[0].Check}
	}
	prepNS := time.Since(t0).Nanoseconds()

	runtime.GC()
	if err := memprobe.ResetPeakRSS(); err != nil {
		return fmt.Errorf("block: reset peak RSS: %w", err)
	}
	rssBefore, _, err := memprobe.RSS()
	if err != nil {
		return err
	}
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	gc0 := procsetup.AutoGCCycles()
	q0, _ := quality.Take()
	startNS := time.Now().UnixNano()
	res, err := conc.Run(workers, conc.Config{Warmup: bud.Warmup, Window: bud.Window, CheckEvery: bud.CheckEvery})
	endNS := time.Now().UnixNano()
	q1, _ := quality.Take()
	gc1 := procsetup.AutoGCCycles()
	runtime.ReadMemStats(&m1)
	if err != nil {
		return fmt.Errorf("block %s: %w", b.Key(), err)
	}
	_, rssPeak, err := memprobe.RSS()
	if err != nil {
		return err
	}
	if res.Hist.N == 0 {
		return fmt.Errorf("block %s: no call completed in the %s window", b.Key(), bud.Window)
	}
	keysChecked := 0
	for k, st := range setups {
		n, err := st.After()
		if err != nil {
			return fmt.Errorf("block %s: worker %d: deferred checks: %v", b.Key(), k, err)
		}
		keysChecked += n
	}

	// Raw file: the per-second series, the per-worker counts and the
	// latency histogram (non-empty buckets).
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	header := map[string]string{
		"key": b.Key(), "id": b.ID, "exp": b.Exp, "alg": b.Alg, "op": b.Op,
		"rep": strconv.Itoa(b.Rep), "attempt": strconv.Itoa(b.Attempt), "profile": b.Profile,
		"seed": strconv.FormatUint(b.Seed, 10), "stage": stageOp, "workers": strconv.Itoa(w),
		"gomaxprocs": strconv.Itoa(runtime.GOMAXPROCS(0)), "cpus": intsCSV(cpus),
		"pool":    strconv.Itoa(cells.PoolSize(c)) + " per worker",
		"budget":  fmt.Sprintf("warm-up %s (every call checked), window %s (every %d-th call of each worker checked)", bud.Warmup, bud.Window, bud.CheckEvery),
		"counted": "a call counts when it completes inside the window; no call starts after it; latency = that call's start to end",
		"histogram": "bucket rows give the lower bound in ns; a bucket holds every value v with lower <= v < lower + width, " +
			"width 1 ns below 128 ns and lower/64 above (64 buckets per power of two); quantiles are bucket midpoints (error <= 0.8 %)",
		"calls_window": strconv.FormatInt(res.Hist.N, 10), "calls_warmup": strconv.FormatInt(res.WarmupCalls, 10),
		"calls_checked": strconv.FormatInt(res.Checked, 10), "throughput_per_s": strconv.FormatFloat(res.Throughput(), 'f', 3, 64),
		"latency_min_ns": strconv.FormatInt(res.Hist.Min, 10), "latency_max_ns": strconv.FormatInt(res.Hist.Max, 10),
		"latency_mean_ns": strconv.FormatFloat(res.Hist.Mean(), 'f', 1, 64),
		"rss_before":      strconv.FormatInt(rssBefore, 10), "rss_peak": strconv.FormatInt(rssPeak, 10),
		"rss":            "peak resident set (VmHWM, reset after the workers' preparation) over warm-up and window; Linux >= 6.2 reads it from per-CPU counters, precise to about 32 pages per CPU",
		"gc":             fmt.Sprintf("Go default (production-like): %d cycles, %d ns of pauses during the run; heap in use %d -> %d bytes", m1.NumGC-m0.NumGC, m1.PauseTotalNs-m0.PauseTotalNs, m0.HeapInuse, m1.HeapInuse),
		"prep_ns":        strconv.FormatInt(prepNS, 10),
		"harness_commit": buildinfo.HarnessCommit, "go": runtime.Version(), "liboqs": pqc.Version(),
		"openssl": classical.OpenSSLVersion(), "start_unix_ns": strconv.FormatInt(startNS, 10),
		"columns": "kind,key,count: second,<s>,<calls completed in second s of the window>; worker,<k>,<calls of worker k in the window>; bucket,<lower ns>,<calls>",
	}
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(gz, "# %s=%s\n", k, strings.ReplaceAll(header[k], "\n", " "))
	}
	fmt.Fprintln(gz, "kind,key,count")
	rows := 0
	for s, n := range res.PerSecond {
		fmt.Fprintf(gz, "second,%d,%d\n", s, n)
		rows++
	}
	for k, n := range res.PerWorker {
		fmt.Fprintf(gz, "worker,%d,%d\n", k, n)
		rows++
	}
	for bk, n := range res.Hist.Counts {
		if n > 0 {
			fmt.Fprintf(gz, "bucket,%d,%d\n", conc.Lower(bk), n)
			rows++
		}
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
		Key: b.Key(), Block: b, StartNS: startNS, EndNS: endNS, WallNS: res.WallNS,
		Budget:  proto.BudgetInfo{Mode: "window", TimeBoxS: bud.Window.Seconds()},
		GCAuto:  gc1 - gc0,
		Quality: quality.Diff(q0, q1, cpus), CPUs: cpus, GOMAXPROCS: runtime.GOMAXPROCS(0), OOMAdj: oomSet,
		Pool: cells.PoolSize(c), KeysChecked: keysChecked, PrepNS: prepNS, Rows: rows, RawSHA256: sum,
	}
	out.Arms = e12Summaries(&res, rssBefore, rssPeak)
	return emit(out)
}

// e12Summaries: latency from the histogram, throughput over the seconds of
// the window, and the block's resident memory.
func e12Summaries(res *conc.Result, rssBefore, rssPeak int64) []proto.ArmSummary {
	h := &res.Hist
	lat := stats.Summary{N: int(h.N), Min: float64(h.Min), Max: float64(h.Max), Mean: h.Mean(), SD: h.SD(),
		P50: h.Quantile(0.5), P90: h.Quantile(0.9), P95: h.Quantile(0.95), P99: h.Quantile(0.99), P999: h.Quantile(0.999)}
	if lat.Mean > 0 {
		lat.CoV = lat.SD / lat.Mean
	}
	per := make([]float64, len(res.PerSecond))
	for s, n := range res.PerSecond {
		per[s] = float64(n)
	}
	one := func(name string, v int64) proto.ArmSummary {
		return proto.ArmSummary{Name: name, Batch: 1, Stats: stats.Summarize([]float64{float64(v)}, 0, 0),
			MinBytes: int(v), MaxBytes: int(v)}
	}
	return []proto.ArmSummary{
		{Name: proto.QLatency, Batch: 1, Stats: lat, SumNS: h.Sum},
		{Name: proto.QThroughput, Batch: 1, Stats: stats.Summarize(per, 0, 0)},
		one(proto.QRSSGrowth, rssPeak-rssBefore),
		one(proto.QRSSPeak, rssPeak),
	}
}
