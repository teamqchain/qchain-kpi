package main

// Executing an E14 plan: warm-up, payload calibration, then the blocks of
// work in the plan's seeded order. A block is discarded and run again in the
// next session if it is interrupted; finished blocks are kept (state.json).
// Every transaction must commit as valid; the blocks holding them are read
// back from the peer; a seeded sample of payloads is read back and compared.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// State is the resumable state of a run (state.json).
type State struct {
	Schema        int                    `json:"schema"`
	RunID         string                 `json:"run_id"`
	Machine       string                 `json:"machine"`
	PlanName      string                 `json:"plan_name"`
	PlanKind      string                 `json:"plan_kind"`
	PlanHash      string                 `json:"plan_hash"`
	Seed          uint64                 `json:"seed"`
	ImageID       string                 `json:"image_id"`
	HarnessCommit string                 `json:"harness_commit"`
	Status        string                 `json:"status"` // running, stopped, failed, done
	StatusDetail  string                 `json:"status_detail,omitempty"`
	StartedUTC    string                 `json:"started_utc"`
	Sessions      []SessionRec           `json:"sessions"`
	Order         []string               `json:"order"`
	Done          map[string]DoneRec     `json:"done"`
	Calibration   map[string]Calibration `json:"calibration"`
	CanarySeq     int                    `json:"canary_seq"`
	Transactions  int                    `json:"transactions"`    // recorded (in finished blocks)
	FalseConflict int                    `json:"false_conflicts"` // of them, committed with MVCC_READ_CONFLICT (see put)
}

// SessionRec is one start or resume.
type SessionRec struct {
	N            int     `json:"n"`
	StartedUTC   string  `json:"started_utc"`
	HostOffsetNS int64   `json:"host_offset_ns"`
	NativeP      float64 `json:"native_p_rate,omitempty"`
	NativeE      float64 `json:"native_e_rate,omitempty"`
}

// DoneRec is a finished block.
type DoneRec struct {
	Session int    `json:"session"`
	RawFile string `json:"raw_file"`
}

var errStopped = errors.New("stopped")

// Runner runs one session.
type Runner struct {
	Plan     *Plan
	Blocks   []Block
	Out      string
	Ledger   Ledger
	St       *State
	Session  int
	Stop     *atomic.Bool
	Log      func(string, ...any)
	Now      func() time.Time
	lastCanary time.Time
}

func (r *Runner) path(name string) string { return filepath.Join(r.Out, name) }

func (r *Runner) saveState() error { return writeJSON(r.path("state.json"), r.St) }

func (r *Runner) journal(event, key, detail string) error {
	return appendCSV(r.path("journal.csv"), []string{"utc_ns", "event", "key", "detail"},
		[]string{strconv.FormatInt(r.Now().UnixNano(), 10), event, key, detail})
}

// txID is a fixed-length transaction key, unique per run, block, session and
// position (a block run again in a later session gets new keys).
func (r *Runner) txID(prefix, key string, i int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%d", r.St.Seed, key, i)))
	return prefix + hex.EncodeToString(h[:])[:23]
}

// payloadHex is n seeded random bytes as lowercase hex (incompressible, like
// keys and signatures).
func (r *Runner) payloadHex(key string, i, n int) string {
	g := rand.New(rand.NewChaCha8(seedBytes(r.St.Seed, "payload|"+key+"|"+strconv.Itoa(i))))
	b := make([]byte, n)
	for j := 0; j < n; j += 8 {
		v := g.Uint64()
		for k := 0; k < 8 && j+k < n; k++ {
			b[j+k] = byte(v >> (8 * k))
		}
	}
	return hex.EncodeToString(b)
}

// envelopeOf finds a committed transaction in its block.
func (r *Runner) envelopeOf(ctx context.Context, t TxResult) (BlockTx, *LedgerBlock, error) {
	raw, err := r.Ledger.Block(ctx, t.Block)
	if err != nil {
		return BlockTx{}, nil, fmt.Errorf("read block %d: %w", t.Block, err)
	}
	b, err := ParseBlock(raw)
	if err != nil {
		return BlockTx{}, nil, err
	}
	for _, tx := range b.Txs {
		if tx.TxID == t.TxID {
			return tx, b, nil
		}
	}
	return BlockTx{}, nil, fmt.Errorf("transaction %s not in block %d", t.TxID, t.Block)
}

// CodeMVCCConflict is Fabric's TxValidationCode MVCC_READ_CONFLICT.
const CodeMVCCConflict = 11

// put submits one transaction. It must commit as valid (code 0), or with
// MVCC_READ_CONFLICT: QChain's two peers share one CouchDB, so the peer that
// validates a block second finds the records the first peer already wrote,
// flags the transactions as conflicting in its own copy of the ledger and
// skips their writes; the records are in the state all the same (the E14
// smoke run, 22 Oct). That outcome is data, not a failure, if the record
// reads back correctly: the caller verifies it with verifyConflict, outside
// the timed region. Any other code stops the run.
func (r *Runner) put(ctx context.Context, id, payload string) (TxResult, error) {
	t, err := r.Ledger.Put(ctx, id, payload)
	if err != nil {
		if r.Stop.Load() {
			return t, errStopped
		}
		return t, err
	}
	if t.Code != 0 && t.Code != CodeMVCCConflict {
		return t, fmt.Errorf("transaction %s committed with validation code %d", t.TxID, t.Code)
	}
	return t, nil
}

// verifyConflict checks that a transaction flagged MVCC_READ_CONFLICT wrote
// its record: it must read back from the state, byte for byte.
func (r *Runner) verifyConflict(ctx context.Context, id, payload string) error {
	got, err := r.Ledger.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("transaction on %s was flagged MVCC_READ_CONFLICT and its record is missing: %w", id, err)
	}
	var rec struct {
		ID      string
		Payload string
	}
	if err := json.Unmarshal(got, &rec); err != nil || rec.ID != id || rec.Payload != payload {
		return fmt.Errorf("transaction on %s was flagged MVCC_READ_CONFLICT and its record differs from what was sent", id)
	}
	return nil
}

// putChecked is put plus the immediate verification of a conflict (warm-up
// and calibration transactions, which are not timed).
func (r *Runner) putChecked(ctx context.Context, id, payload string) (TxResult, error) {
	t, err := r.put(ctx, id, payload)
	if err == nil && t.Code == CodeMVCCConflict {
		err = r.verifyConflict(ctx, id, payload)
	}
	return t, err
}
// warmup submits n untimed transactions of the given payload size.
func (r *Runner) warmup(ctx context.Context, what string, n, payload int) error {
	key := fmt.Sprintf("warmup.%s.s%d", what, r.Session)
	for i := 0; i < n; i++ {
		if r.Stop.Load() {
			return errStopped
		}
		if _, err := r.putChecked(ctx, r.txID("W", key, i), r.payloadHex(key, i, payload)); err != nil {
			return fmt.Errorf("warm-up %s: %w", what, err)
		}
	}
	return nil
}

// calibrate fills the calibration of every point not yet calibrated.
func (r *Runner) calibrate(ctx context.Context) error {
	if r.St.Calibration == nil {
		r.St.Calibration = map[string]Calibration{}
	}
	for _, pt := range r.Plan.Points {
		if _, ok := r.St.Calibration[pt.Name]; ok {
			continue
		}
		key := fmt.Sprintf("cal.%s.s%d", pt.Name, r.Session)
		i := 0
		measure := func(p int) (int, error) {
			if r.Stop.Load() {
				return 0, errStopped
			}
			t, err := r.putChecked(ctx, r.txID("C", key, i), r.payloadHex(key, i, p))
			i++
			if err != nil {
				return 0, err
			}
			tx, _, err := r.envelopeOf(ctx, t)
			return tx.EnvelopeBytes, err
		}
		c, err := Calibrate(pt.Name, pt.TargetEnvelope, measure)
		if err != nil {
			return err
		}
		r.St.Calibration[pt.Name] = c
		r.Log("calibrated %s: payload %d B -> envelope %d B (target %d, %+.2f%%, %d transactions)",
			pt.Name, c.Payload, c.Envelope, c.Target, c.ErrorPct, len(c.Steps))
		if err := r.journal("calibrated", pt.Name, fmt.Sprintf("payload %d envelope %d target %d", c.Payload, c.Envelope, c.Target)); err != nil {
			return err
		}
		if err := r.saveState(); err != nil {
			return err
		}
	}
	return writeJSON(r.path("calibration.json"), r.St.Calibration)
}

// canary runs the canary and appends it to canary.csv.
func (r *Runner) canary(reason string) error {
	c := RunCanary(15, 100*time.Millisecond)
	r.St.CanarySeq++
	r.lastCanary = r.Now()
	r.Log("canary %d (%s): %.0f hashes/s", r.St.CanarySeq, reason, c.MedianRate)
	return appendCSV(r.path("canary.csv"),
		[]string{"seq", "reason", "session", "utc_ns", "median_rate_per_s", "windows", "window_ms", "digest"},
		[]string{strconv.Itoa(r.St.CanarySeq), reason, strconv.Itoa(r.Session), strconv.FormatInt(r.Now().UnixNano(), 10),
			fmt.Sprintf("%.1f", c.MedianRate), strconv.Itoa(c.Windows), strconv.Itoa(c.WindowMS), c.Digest})
}

// Run executes the blocks not yet done.
func (r *Runner) Run(ctx context.Context) error {
	if r.Now == nil {
		r.Now = time.Now
	}
	if err := r.canary("start"); err != nil {
		return err
	}
	if err := r.warmup(ctx, "run", r.Plan.WarmupRun, 1); err != nil {
		return err
	}
	if err := r.calibrate(ctx); err != nil {
		return err
	}
	total := len(r.Blocks)
	for _, b := range r.Blocks {
		if _, done := r.St.Done[b.ID]; done {
			continue
		}
		if r.Stop.Load() {
			return errStopped
		}
		if r.Plan.CanaryEvery > 0 && r.Now().Sub(r.lastCanary) >= time.Duration(r.Plan.CanaryEvery)*time.Second {
			if err := r.canary("periodic"); err != nil {
				return err
			}
		}
		if err := r.journal("block_start", b.ID, ""); err != nil {
			return err
		}
		n, err := r.runBlock(ctx, b)
		if err != nil {
			if errors.Is(err, errStopped) {
				r.journal("block_interrupted", b.ID, "discarded; it runs again on resume")
				return errStopped
			}
			r.journal("block_failed", b.ID, err.Error())
			return fmt.Errorf("%s: %w", b.ID, err)
		}
		r.St.Transactions += n
		if err := r.saveState(); err != nil {
			return err
		}
		r.Log("%d/%d %s done", len(r.St.Done), total, b.ID)
	}
	return r.canary("end")
}

// runBlock runs one block and writes its raw file and summary row.
func (r *Runner) runBlock(ctx context.Context, b Block) (int, error) {
	cal, ok := r.St.Calibration[b.Point.Name]
	if !ok {
		return 0, fmt.Errorf("no calibration for %s", b.Point.Name)
	}
	if err := r.warmup(ctx, b.ID, r.Plan.WarmupBlock, cal.Payload); err != nil {
		return 0, err
	}
	key := fmt.Sprintf("%s.s%d", b.ID, r.Session)
	ids := make([]string, b.N)
	payloads := make([]string, b.N)
	for i := range ids {
		ids[i] = r.txID("K", key, i)
	}
	res := make([]TxResult, b.N)
	start := r.Now()
	switch b.Mode {
	case ModeSequential, ModeDrift:
		for i := 0; i < b.N; i++ {
			if r.Stop.Load() {
				return 0, errStopped
			}
			payloads[i] = r.payloadHex(key, i, cal.Payload)
			t, err := r.put(ctx, ids[i], payloads[i])
			if err != nil {
				return 0, fmt.Errorf("transaction %d: %w", i+1, err)
			}
			res[i] = t
		}
	case ModeBurst:
		for i := range payloads {
			payloads[i] = r.payloadHex(key, i, cal.Payload)
		}
		errs := make([]error, b.N)
		release := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-release
				res[i], errs[i] = r.put(ctx, ids[i], payloads[i])
			}(i)
		}
		close(release)
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				return 0, fmt.Errorf("burst transaction %d: %w", i+1, err)
			}
		}
	default:
		return 0, fmt.Errorf("unknown mode %q", b.Mode)
	}
	end := r.Now()

	// The ledger blocks that hold this block's transactions.
	byTx := map[string]int{}
	var nums []uint64
	seen := map[uint64]bool{}
	for i, t := range res {
		byTx[t.TxID] = i
		if !seen[t.Block] {
			seen[t.Block] = true
			nums = append(nums, t.Block)
		}
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	type found struct {
		block, index int
		tx           BlockTx
	}
	where := make([]found, b.N)
	located := 0
	var lblocks []*LedgerBlock
	for _, n := range nums {
		raw, err := r.Ledger.Block(ctx, n)
		if err != nil {
			return 0, fmt.Errorf("read block %d: %w", n, err)
		}
		lb, err := ParseBlock(raw)
		if err != nil {
			return 0, err
		}
		lblocks = append(lblocks, lb)
		for idx, tx := range lb.Txs {
			if i, ok := byTx[tx.TxID]; ok {
				if tx.Code != 0 && tx.Code != CodeMVCCConflict {
					return 0, fmt.Errorf("transaction %s is marked %d in block %d", tx.TxID, tx.Code, n)
				}
				where[i] = found{block: int(n), index: idx, tx: tx}
				located++
			}
		}
	}
	if located != b.N {
		return 0, fmt.Errorf("found %d of %d transactions in their blocks", located, b.N)
	}

	// Transactions flagged MVCC_READ_CONFLICT: the status the Gateway gave
	// and the block's flag must agree, and the record must be in the state.
	conflicts := 0
	for i, t := range res {
		if int(t.Code) != int(where[i].tx.Code) {
			return 0, fmt.Errorf("transaction %s: commit status %d but its block says %d", t.TxID, t.Code, where[i].tx.Code)
		}
		if t.Code == CodeMVCCConflict {
			conflicts++
			if err := r.verifyConflict(ctx, ids[i], payloads[i]); err != nil {
				return 0, err
			}
		}
	}

	// Read back a seeded sample (one per block of work) and compare.
	g := rand.New(rand.NewChaCha8(seedBytes(r.St.Seed, "readback|"+key)))
	k := g.IntN(b.N)
	got, err := r.Ledger.Get(ctx, ids[k])
	if err != nil {
		return 0, fmt.Errorf("read back %s: %w", ids[k], err)
	}
	var rec struct {
		ID      string
		Payload string
	}
	if err := json.Unmarshal(got, &rec); err != nil || rec.ID != ids[k] || rec.Payload != payloads[k] {
		return 0, fmt.Errorf("read back of %s does not match what was sent", ids[k])
	}

	// Cut reasons: within a burst, a block is cut on size if the next one of
	// this block's transactions would not have fitted.
	cut := map[string]int{}
	ours := make([]int, len(lblocks))
	rows := [][]string{{"kind", "seq", "start_unix_ns", "connect_ns", "endorse_ns", "submit_ns", "commit_ns", "total_ns",
		"block", "index", "code", "envelope_bytes", "orderer_bytes"}}
	for i, t := range res {
		w := where[i]
		rows = append(rows, []string{"tx", strconv.Itoa(i + 1), strconv.FormatInt(t.StartUnixNS, 10),
			strconv.FormatInt(t.ConnectNS, 10), strconv.FormatInt(t.EndorseNS, 10), strconv.FormatInt(t.SubmitNS, 10),
			strconv.FormatInt(t.CommitNS, 10), strconv.FormatInt(t.TotalNS, 10), strconv.Itoa(w.block), strconv.Itoa(w.index),
			strconv.Itoa(int(w.tx.Code)), strconv.Itoa(w.tx.EnvelopeBytes), strconv.Itoa(w.tx.OrdererBytes)})
	}
	for j, lb := range lblocks {
		for _, tx := range lb.Txs {
			if _, ok := byTx[tx.TxID]; ok {
				ours[j]++
			}
		}
		next := 0
		if b.Mode == ModeBurst && j+1 < len(lblocks) {
			next = lblocks[j+1].Txs[0].OrdererBytes
		}
		reason := CutReason(lb, next)
		cut[reason]++
		rows = append(rows, []string{"block", strconv.FormatUint(lb.Number, 10), strconv.Itoa(len(lb.Txs)), strconv.Itoa(ours[j]),
			strconv.Itoa(lb.Bytes), strconv.Itoa(lb.OrdererSum()), reason, "", "", "", "", "", ""})
	}

	header := map[string]string{
		"exp": "E14", "id": b.ID, "key": key, "point": b.Point.Name, "mode": b.Mode, "rep": strconv.Itoa(b.Rep),
		"n": strconv.Itoa(b.N), "payload_bytes": strconv.Itoa(cal.Payload), "target_envelope": strconv.Itoa(b.Point.TargetEnvelope),
		"e13_per_block": strconv.Itoa(b.Point.E13PerBlock), "session": strconv.Itoa(r.Session),
		"start_unix_ns": strconv.FormatInt(start.UnixNano(), 10), "end_unix_ns": strconv.FormatInt(end.UnixNano(), 10),
		"harness_commit": harnessCommit, "plan": r.Plan.Name, "seed": strconv.FormatUint(r.St.Seed, 10),
		"warmup_tx": strconv.Itoa(r.Plan.WarmupBlock),
		"columns": "tx rows: one per transaction (times in ns from the client's monotonic clock; start_unix_ns wall clock); " +
			"block rows: kind=block, then ledger block number, transactions in it, of them this block's, block bytes, " +
			"orderer-counted bytes (payload + signature of every envelope), inferred cut reason (count, size, timeout)",
		"readback": ids[k],
	}
	gz, err := gzipCSV(header, rows)
	if err != nil {
		return 0, err
	}
	rawRel := filepath.ToSlash(filepath.Join("raw", "E14", b.Point.Name, rawName(b)))
	if err := writeFileAtomic(r.path(rawRel), gz); err != nil {
		return 0, err
	}

	// Summary row.
	f := func(sel func(TxResult) int64) Summary {
		xs := make([]float64, len(res))
		for i, t := range res {
			xs[i] = float64(sel(t))
		}
		return Summarise(xs)
	}
	tot := f(func(t TxResult) int64 { return t.TotalNS })
	env := make([]float64, b.N)
	for i := range where {
		env[i] = float64(where[i].tx.EnvelopeBytes)
	}
	es := Summarise(env)
	minOurs, maxOurs := b.N, 0
	for _, o := range ours {
		minOurs = min(minOurs, o)
		maxOurs = max(maxOurs, o)
	}
	row := []string{key, b.ID, b.Point.Name, b.Mode, strconv.Itoa(b.Rep), strconv.Itoa(r.Session), strconv.Itoa(b.N),
		strconv.Itoa(cal.Payload), strconv.Itoa(b.Point.TargetEnvelope), ff(es.Mean), ff(es.Min), ff(es.Max),
		ff(tot.P50), ff(tot.P90), ff(tot.P99), ff(tot.Mean), ff(tot.Min), ff(tot.Max),
		ff(f(func(t TxResult) int64 { return t.ConnectNS }).P50), ff(f(func(t TxResult) int64 { return t.EndorseNS }).P50),
		ff(f(func(t TxResult) int64 { return t.SubmitNS }).P50), ff(f(func(t TxResult) int64 { return t.CommitNS }).P50),
		strconv.Itoa(len(lblocks)), strconv.Itoa(minOurs), strconv.Itoa(maxOurs),
		strconv.Itoa(cut["count"]), strconv.Itoa(cut["size"]), strconv.Itoa(cut["timeout"]), strconv.Itoa(conflicts), strconv.Itoa(b.Point.E13PerBlock),
		strconv.FormatInt(start.UnixNano(), 10), strconv.FormatInt(end.UnixNano(), 10), strconv.FormatInt(end.Sub(start).Nanoseconds(), 10),
		rawRel, sha256Hex(gz)}
	if err := appendCSV(r.path("blocks.csv"), blocksHeader, row); err != nil {
		return 0, err
	}
	r.St.Done[b.ID] = DoneRec{Session: r.Session, RawFile: rawRel}
	r.St.FalseConflict += conflicts
	detail := fmt.Sprintf("p50 %.1f ms, %d ledger blocks, %d to %d per block (count %d, size %d, timeout %d), %d MVCC conflicts",
		tot.P50/1e6, len(lblocks), minOurs, maxOurs, cut["count"], cut["size"], cut["timeout"], conflicts)
	if err := r.journal("block_done", b.ID, detail); err != nil {
		return 0, err
	}
	r.Log("%s: %s", b.ID, detail)
	return b.N, nil
}

func rawName(b Block) string {
	if b.Mode == ModeDrift {
		return fmt.Sprintf("drift-d%d.csv.gz", b.Rep)
	}
	return fmt.Sprintf("%s-r%d.csv.gz", b.Mode, b.Rep)
}

var blocksHeader = []string{"key", "id", "point", "mode", "rep", "session", "n", "payload_bytes", "target_envelope",
	"envelope_mean", "envelope_min", "envelope_max", "total_p50_ns", "total_p90_ns", "total_p99_ns", "total_mean_ns",
	"total_min_ns", "total_max_ns", "connect_p50_ns", "endorse_p50_ns", "submit_p50_ns", "commit_p50_ns",
	"ledger_blocks", "ours_per_block_min", "ours_per_block_max", "cut_count", "cut_size", "cut_timeout", "mvcc_conflicts", "e13_per_block",
	"start_unix_ns", "end_unix_ns", "wall_ns", "raw_file", "raw_sha256"}

// cleanTemp removes temporary files an interrupted write may have left.
func cleanTemp(dir string) {
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Ext(p) == ".tmp" {
			os.Remove(p)
		}
		return nil
	})
}
