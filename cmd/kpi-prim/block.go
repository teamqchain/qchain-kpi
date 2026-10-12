package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/record"
	"qchain-kpi/internal/stats"
	"qchain-kpi/internal/timing"
)

// rawRow is one timed call, kept in memory until the timed phase is over so
// that no file I/O or compression happens between samples.
type rawRow struct {
	arm    int8
	first  bool
	ns     int64
	nbytes int32
}

func cmdBlock(args []string) error {
	fs := flag.NewFlagSet("block", flag.ExitOnError)
	specJSON := fs.String("spec", "", "block specification (JSON, see internal/proto)")
	fs.Parse(args)
	var sp proto.Spec
	dec := json.NewDecoder(strings.NewReader(*specJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sp); err != nil {
		return fmt.Errorf("block: bad --spec: %w", err)
	}
	b := sp.Block
	a, ok := algs.ByID(b.Alg)
	if !ok {
		return fmt.Errorf("block: unknown algorithm %q", b.Alg)
	}
	if b.Rep < 1 || b.Attempt < 1 || sp.RawFile == "" || !filepath.IsAbs(sp.RawFile) || sp.PrepDir == "" {
		return fmt.Errorf("block: incomplete spec %+v", sp)
	}
	firstLabel := "cold"
	switch b.Exp {
	case "E1", "probe":
	case "E10":
		if a.Family != "ML-DSA" || b.Op != "sign" {
			return fmt.Errorf("block: E10 is ML-DSA sign only")
		}
		firstLabel = "first" // several arms share the process
	case "E2", "E2c", "probe-E2":
		if !a.HasOp(b.Op) {
			return fmt.Errorf("block: %s has no operation %q", a.ID, b.Op)
		}
		firstLabel = "first"
	case "E8":
		return e8Block(sp, a) // sizes only: nothing is timed
	default:
		return fmt.Errorf("block: experiment %q not implemented", b.Exp)
	}
	probe := proto.IsProbe(b.Exp)
	if !probe && b.Mode != string(timing.Fast) && b.Mode != string(timing.Slow) {
		return fmt.Errorf("block %s: mode %q must be fast or slow", b.Key(), b.Mode)
	}

	oomSet := procsetup.OOMAdj()
	procsetup.MicroGC()

	prepRep := b.Rep
	if probe {
		prepRep = proto.ProbeRep
	}
	p, err := inputs.Load(proto.PrepPath(sp.PrepDir, a.ID, prepRep))
	if err != nil {
		return err
	}
	if p.Alg != a.ID || p.Rep != prepRep {
		return fmt.Errorf("block: prep is for %s rep %d, not %s rep %d", p.Alg, p.Rep, a.ID, prepRep)
	}

	// Build the arms. Nothing here runs a cryptographic operation.
	var set *armSet
	switch b.Exp {
	case "E10":
		if !p.HasE10() {
			return fmt.Errorf("block: prep %s r%d has no E10 pools", a.ID, b.Rep)
		}
		set, err = e10Arms(a, p)
	case "E2":
		set, err = e2Arms(a, b.Op, p, b.Seed, sp.Batch)
	case "E2c":
		set, err = e2cArms(a, b.Op, p, sp.Batch)
	case "probe-E2":
		// Every E2 and E2c arm, unbatched: the probe pass decides the batches.
		var e2, e2c *armSet
		if e2, err = e2Arms(a, b.Op, p, b.Seed, nil); err == nil {
			if e2c, err = e2cArms(a, b.Op, p, nil); err != nil {
				e2.close()
			} else {
				set = mergeSets(e2, e2c)
			}
		}
	default:
		set, err = e1Arms(a, b.Op, p, b.Seed)
	}
	if err != nil {
		return err
	}
	defer set.close()
	if !probe {
		for k := range set.arms {
			if n := sp.Batch[set.arms[k].Name]; n > 1 {
				set.arms[k].Batch = n
			}
		}
	}
	armIdx := map[string]int8{}
	var armList, batchList []string
	for k, arm := range set.arms {
		armIdx[arm.Name] = int8(k)
		armList = append(armList, fmt.Sprintf("%d:%s", k, arm.Name))
		batchList = append(batchList, fmt.Sprintf("%d:%d", k, max(arm.Batch, 1)))
	}

	var cfg timing.Config
	if !probe {
		if cfg, err = proto.Budget(b.Profile, timing.Mode(b.Mode), b.Seed, firstLabel); err != nil {
			return err
		}
		wall := sp.MaxWall
		if wall <= 0 {
			wall = 45 * 60
		}
		cfg.MaxWall = time.Duration(wall) * time.Second
	}
	capRows := 4096
	if cfg.Mode == timing.Fast {
		capRows = len(set.arms) * (cfg.Samples + 1)
	}
	rows := make([]rawRow, 0, capRows)
	prefix := firstLabel + ":"
	cfg.Sink = func(label string, ns int64, nbytes int) {
		first := strings.HasPrefix(label, prefix)
		rows = append(rows, rawRow{arm: armIdx[strings.TrimPrefix(label, prefix)], first: first, ns: ns, nbytes: int32(nbytes)})
	}

	cpus, _ := quality.CPUSet()
	q0, _ := quality.Take()
	gc0 := procsetup.AutoGCCycles()
	startNS := time.Now().UnixNano()
	t0 := time.Now()

	var res timing.Result
	probeNS := make([][]int64, len(set.arms))
	if probe {
		// Each arm in turn: its first call, then ProbeCalls calls.
		for k, arm := range set.arms {
			var first int64
			first, probeNS[k], err = timing.Probe(arm, proto.ProbeCalls)
			if err != nil {
				break
			}
			cfg.Sink(prefix+arm.Name, first, 0)
			for _, d := range probeNS[k] {
				cfg.Sink(arm.Name, d, 0)
			}
			pf := make([]float64, len(probeNS[k]))
			for i, d := range probeNS[k] {
				pf[i] = float64(d)
			}
			res.Arms = append(res.Arms, timing.ArmResult{Name: arm.Name, Batch: 1, FirstNS: first, PerOpNS: pf})
		}
	} else {
		res, err = timing.Run(set.arms, cfg)
	}
	wall := time.Since(t0).Nanoseconds()
	endNS := time.Now().UnixNano()
	gc1 := procsetup.AutoGCCycles()
	q1, _ := quality.Take()
	if err != nil {
		return err
	}
	res.WallNS = wall

	// Deferred correctness checks (keygen subset), then allocation counts.
	keysChecked, err := set.after()
	if err != nil {
		return fmt.Errorf("%w: %v", timing.ErrCheck, err)
	}
	allocCalls := 100
	if cfg.Mode == timing.Slow {
		allocCalls = 2
	}
	heap := make([]float64, len(set.arms))
	allocs := make([]float64, len(set.arms))
	if !probe {
		for k, arm := range set.arms {
			if heap[k], allocs[k], err = allocsPerOp(arm, set.resetOf(k), allocCalls, 1<<20); err != nil {
				return err
			}
		}
	}

	// Raw file.
	header := map[string]string{
		"key": b.Key(), "id": b.ID, "exp": b.Exp, "alg": b.Alg, "op": b.Op,
		"rep": strconv.Itoa(b.Rep), "attempt": strconv.Itoa(b.Attempt), "profile": b.Profile,
		"mode": b.Mode, "seed": strconv.FormatUint(b.Seed, 10), "diagnostic": strconv.FormatBool(b.Diagnostic),
		"arms": strings.Join(armList, ","), "batch": strings.Join(batchList, ","), "first_label": firstLabel,
		"pool": strconv.Itoa(p.P), "prep_id": p.Fingerprint, "harness_commit": buildinfo.HarnessCommit,
		"go": runtime.Version(), "liboqs": pqc.Version(), "openssl": classical.OpenSSLVersion(),
		"gomaxprocs": strconv.Itoa(runtime.GOMAXPROCS(0)), "cpus": intsCSV(cpus),
		"start_unix_ns": strconv.FormatInt(startNS, 10),
		"columns":       "arm = arm index (see arms), or <first_label>:<index> for the first call of that arm; ns = one call, or the total of <batch> calls for samples",
	}
	if b.Exp == "E10" {
		header["holder_prep_id"] = p.HolderFingerprint
	}
	if probe {
		header["budget"] = fmt.Sprintf("%s call + %d calls per arm", firstLabel, proto.ProbeCalls)
	} else {
		bi, _ := json.Marshal(proto.Info(cfg))
		header["budget"] = string(bi)
	}
	w, err := record.NewRawWriter(sp.RawFile, header, set.hasBytes, false)
	if err != nil {
		return err
	}
	for _, r := range rows {
		label := strconv.Itoa(int(r.arm))
		if r.first {
			label = prefix + label
		}
		w.Row(label, r.ns, int(r.nbytes), true)
	}
	nrows := w.Rows()
	if err := w.Close(); err != nil {
		return err
	}
	sum, err := fileSHA256(sp.RawFile)
	if err != nil {
		return err
	}

	out := proto.Result{
		Key: b.Key(), Block: b, Truncated: res.Truncated, StartNS: startNS, EndNS: endNS, WallNS: res.WallNS,
		GCForced: res.GCForced, GCAuto: gc1 - gc0, Quality: quality.Diff(q0, q1, cpus), CPUs: cpus,
		GOMAXPROCS: runtime.GOMAXPROCS(0), OOMAdj: oomSet, Pool: p.P, PrepID: p.Fingerprint,
		KeysChecked: keysChecked, Rows: nrows, RawSHA256: sum,
	}
	if !probe {
		out.Budget = proto.Info(cfg)
	}
	for k, ar := range res.Arms {
		s := proto.ArmSummary{
			Name: ar.Name, Batch: ar.Batch, FirstLabel: firstLabel, FirstNS: ar.FirstNS,
			Stats: stats.Summarize(ar.PerOpNS, proto.BootReps, b.Seed^uint64(k+1)), SumNS: ar.SumNS,
			MinBytes: ar.MinBytes, MaxBytes: ar.MaxBytes, HeapBytesOp: heap[k], AllocsOp: allocs[k],
		}
		if probe {
			s.ProbeNS = probeNS[k]
			s.Stats = stats.Summarize(ar.PerOpNS, 0, 0)
		}
		out.Arms = append(out.Arms, s)
	}
	return emit(out)
}

// allocsPerOp is timing.AllocsPerOp for arms whose state must be restored
// between calls (reset): each call is measured on its own and the state is
// restored outside the measurement.
func allocsPerOp(a timing.Arm, reset func() error, n, start int) (float64, float64, error) {
	if reset == nil {
		return timing.AllocsPerOp(a, n, start)
	}
	runtime.GC()
	var bytesTotal, allocsTotal uint64
	var before, after runtime.MemStats
	for i := 0; i < n; i++ {
		runtime.ReadMemStats(&before)
		if _, err := a.Call(start + i); err != nil {
			return 0, 0, err
		}
		runtime.ReadMemStats(&after)
		bytesTotal += after.TotalAlloc - before.TotalAlloc
		allocsTotal += after.Mallocs - before.Mallocs
		if err := reset(); err != nil {
			return 0, 0, err
		}
	}
	return float64(bytesTotal) / float64(n), float64(allocsTotal) / float64(n), nil
}

func intsCSV(xs []int) string {
	s := make([]string, len(xs))
	for i, x := range xs {
		s[i] = strconv.Itoa(x)
	}
	return strings.Join(s, " ")
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// checkMessage is the fixed message used to exercise generated keys.
var checkMessage = []byte("4f1c8a2b9d3e6071c5a4f2b8e9d1c3a7b6e5f4d3c2b1a09f8e7d6c5b4a392817")

// equalBytes is bytes.Equal (named for readability in checks).
func equalBytes(a, b []byte) bool { return bytes.Equal(a, b) }
