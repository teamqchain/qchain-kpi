package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"google.golang.org/protobuf/proto"
)

// fakeLedger stands in for the network. A transaction's envelope is about
// 3,300 bytes plus 4 per payload byte (as on the real network). Batches are
// cut like Fabric's orderer: at 10 transactions, before a transaction that
// would pass 512 KB, or when the batch timer (30 ms here, 2 s in QChain)
// expires; Put returns when its block is cut, as a commit wait does.
type fakeLedger struct {
	mu      sync.Mutex
	next    uint64
	blocks  map[uint64][][]byte
	state   map[string]string
	pending [][]byte
	done    chan struct{} // closed when the pending batch is cut
	failAt  int           // fail the n-th Put (1-based), 0 = never
	puts    int
	corrupt bool // Get returns a different payload
}

func newFake() *fakeLedger {
	return &fakeLedger{next: 5, blocks: map[uint64][][]byte{}, state: map[string]string{}}
}

func envelope(txid string, payloadHexLen int) []byte {
	ch, _ := proto.Marshal(&common.ChannelHeader{TxId: txid})
	pl, _ := proto.Marshal(&common.Payload{Header: &common.Header{ChannelHeader: ch}, Data: make([]byte, 3300+2*payloadHexLen)})
	env, _ := proto.Marshal(&common.Envelope{Payload: pl, Signature: make([]byte, 71)})
	return env
}

func (f *fakeLedger) cutLocked() {
	if len(f.pending) == 0 {
		return
	}
	f.blocks[f.next] = f.pending
	f.next++
	f.pending = nil
	close(f.done)
	f.done = nil
}

func (f *fakeLedger) Put(ctx context.Context, id, payloadHex string) (TxResult, error) {
	f.mu.Lock()
	f.puts++
	if f.failAt > 0 && f.puts == f.failAt {
		f.mu.Unlock()
		return TxResult{}, fmt.Errorf("endorse: simulated failure")
	}
	if err := ctx.Err(); err != nil {
		f.mu.Unlock()
		return TxResult{}, err
	}
	txid := "tx-" + id
	env := envelope(txid, len(payloadHex))
	f.state[id] = payloadHex
	size := 0
	for _, e := range f.pending {
		size += len(e)
	}
	if len(f.pending) > 0 && size+len(env) > PreferredMaxBytes {
		f.cutLocked()
	}
	if f.done == nil {
		d := make(chan struct{})
		f.done = d
		time.AfterFunc(30*time.Millisecond, func() {
			f.mu.Lock()
			if f.done == d {
				f.cutLocked()
			}
			f.mu.Unlock()
		})
	}
	f.pending = append(f.pending, env)
	block, done := f.next, f.done
	if len(f.pending) == MaxMessageCount {
		f.cutLocked()
	}
	f.mu.Unlock()
	<-done
	return TxResult{StartUnixNS: time.Now().UnixNano(), ConnectNS: 1, EndorseNS: 2, SubmitNS: 3, CommitNS: 4, TotalNS: 10,
		TxID: txid, Block: block, Valid: true}, nil
}

func (f *fakeLedger) Block(ctx context.Context, n uint64) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	envs, ok := f.blocks[n]
	if !ok {
		return nil, fmt.Errorf("no block %d", n)
	}
	filter := make([]byte, len(envs))
	b := &common.Block{Header: &common.BlockHeader{Number: n}, Data: &common.BlockData{Data: envs},
		Metadata: &common.BlockMetadata{Metadata: [][]byte{nil, nil, filter, nil, nil}}}
	return proto.Marshal(b)
}

func (f *fakeLedger) Get(ctx context.Context, id string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.state[id]
	if f.corrupt {
		p = "00" + p[2:]
	}
	return json.Marshal(map[string]string{"DocType": "kpi-payload", "ID": id, "Payload": p})
}

func (f *fakeLedger) Close() {}

func testPlan() *Plan {
	return &Plan{Name: "t", Kind: "smoke", Reps: 2, Sequential: 3, Burst: 25, Controls: 2, WarmupRun: 1, WarmupBlock: 1,
		Points: []Point{{Name: "tiny"}, {Name: "ml-dsa-44", TargetEnvelope: 21529, E13PerBlock: 10},
			{Name: "slh-dsa-sha2-256f", TargetEnvelope: 206294, E13PerBlock: 2}}}
}

func TestPlanExpand(t *testing.T) {
	p := testPlan()
	if err := p.check(); err != nil {
		t.Fatal(err)
	}
	a, b := p.Expand(42), p.Expand(42)
	if len(a) != 2*3*2+3 {
		t.Fatalf("%d blocks", len(a))
	}
	ids := map[string]bool{}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Seq != i+1 {
			t.Fatal("expansion is not deterministic")
		}
		if ids[a[i].ID] {
			t.Fatalf("repeated id %s", a[i].ID)
		}
		ids[a[i].ID] = true
	}
	if a[0].ID != "E14.tiny.drift.d1" || a[len(a)-1].ID != "E14.tiny.drift.d3" || a[len(a)/2].ID != "E14.tiny.drift.d2" {
		t.Fatalf("drift controls misplaced: %s %s %s", a[0].ID, a[len(a)/2].ID, a[len(a)-1].ID)
	}
	if c := p.Expand(43); c[1].ID == a[1].ID && c[2].ID == a[2].ID && c[3].ID == a[3].ID {
		t.Fatal("the seed does not change the order")
	}
	bad := *p
	bad.Points = []Point{{Name: "ml-dsa-44", TargetEnvelope: 21529}}
	if bad.check() == nil {
		t.Fatal("a plan without the tiny point was accepted")
	}
}

func TestPlanFiles(t *testing.T) {
	for _, name := range []string{"smoke-e14.json", "e14.json"} {
		p, err := LoadPlan(filepath.Join("..", "plans", name))
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Expand(1)) != 2*len(p.Points)*p.Reps+3 {
			t.Fatalf("%s: block count", name)
		}
	}
}

func TestCalibrate(t *testing.T) {
	// envelope = 3333 + 4.02 p, give or take 2 bytes
	n := 0
	measure := func(p int) (int, error) {
		n++
		return int(3333+4.02*float64(p)) + n%3 - 1, nil
	}
	for _, target := range []int{7141, 21529, 206294} {
		c, err := Calibrate("x", target, measure)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(c.ErrorPct) > calTolPct || len(c.Steps) > 2+calRounds {
			t.Fatalf("target %d: %+v", target, c)
		}
	}
	c, err := Calibrate("tiny", 0, measure)
	if err != nil || c.Payload != 1 || len(c.Steps) != 1 {
		t.Fatalf("tiny: %+v %v", c, err)
	}
	if _, err := Calibrate("x", 2000, measure); err == nil {
		t.Fatal("a target below the smallest envelope was accepted")
	}
}

func TestParseBlockAndCut(t *testing.T) {
	envs := [][]byte{envelope("a", 100), envelope("b", 100)}
	raw, _ := proto.Marshal(&common.Block{Header: &common.BlockHeader{Number: 7}, Data: &common.BlockData{Data: envs},
		Metadata: &common.BlockMetadata{Metadata: [][]byte{nil, nil, {0, 11}, nil, nil}}})
	b, err := ParseBlock(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b.Number != 7 || len(b.Txs) != 2 || b.Txs[0].TxID != "a" || b.Txs[1].Code != 11 || b.Txs[0].EnvelopeBytes != len(envs[0]) || b.Bytes != len(raw) {
		t.Fatalf("%+v", b)
	}
	if b.Txs[0].OrdererBytes >= b.Txs[0].EnvelopeBytes || b.Txs[0].OrdererBytes < b.Txs[0].EnvelopeBytes-10 {
		t.Fatalf("orderer bytes %d, envelope %d", b.Txs[0].OrdererBytes, b.Txs[0].EnvelopeBytes)
	}
	if CutReason(b, 0) != "timeout" || CutReason(b, PreferredMaxBytes) != "size" {
		t.Fatal("cut reason")
	}
	b.Txs = make([]BlockTx, MaxMessageCount)
	if CutReason(b, 0) != "count" {
		t.Fatal("count")
	}
}

func TestSummarise(t *testing.T) {
	s := Summarise([]float64{5, 1, 4, 2, 3, 6, 7, 8, 9, 10})
	if s.N != 10 || s.Min != 1 || s.Max != 10 || s.P50 != 5.5 || s.P90 != 9 || s.P99 != 10 || s.Mean != 5.5 {
		t.Fatalf("%+v", s)
	}
}

func newRunner(t *testing.T, l Ledger) (*Runner, string) {
	t.Helper()
	dir := t.TempDir()
	p := testPlan()
	st := &State{Schema: 1, RunID: "r", Seed: 99, Done: map[string]DoneRec{}}
	return &Runner{Plan: p, Blocks: p.Expand(99), Out: dir, Ledger: l, St: st, Session: 1, Stop: &atomic.Bool{},
		Log: func(string, ...any) {}, Now: time.Now}, dir
}

func readGz(t *testing.T, path string) (map[string]string, [][]string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	head := map[string]string{}
	var body []string
	for _, line := range strings.Split(strings.TrimSpace(string(plain)), "\n") {
		if k, v, ok := strings.Cut(strings.TrimPrefix(line, "# "), "="); ok && strings.HasPrefix(line, "# ") {
			head[k] = v
			continue
		}
		body = append(body, line)
	}
	rows, err := csv.NewReader(strings.NewReader(strings.Join(body, "\n"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return head, rows
}

// TestRunnerBlocks: a whole smoke-sized run on the fake ledger: calibration,
// sequential blocks (one transaction per ledger block, cut by the timer),
// bursts (cut by count for ML-DSA-44, by size for SLH-DSA-256f), raw files,
// the summary table and the state.
func TestRunnerBlocks(t *testing.T) {
	r, dir := newRunner(t, newFake())
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.St.Done) != len(r.Blocks) {
		t.Fatalf("%d of %d blocks done", len(r.St.Done), len(r.Blocks))
	}
	for name, c := range r.St.Calibration {
		if name != "tiny" && math.Abs(c.ErrorPct) > calTolPct {
			t.Fatalf("calibration %s: %+v", name, c)
		}
	}
	head, rows := readGz(t, filepath.Join(dir, "raw/E14/slh-dsa-sha2-256f/burst-r1.csv.gz"))
	if head["mode"] != "burst" || head["n"] != "25" {
		t.Fatalf("header %v", head)
	}
	var tx, blocks int
	cuts := map[string]int{}
	for _, row := range rows[1:] {
		switch row[0] {
		case "tx":
			tx++
		case "block":
			blocks++
			cuts[row[6]]++
			if row[6] == "size" && row[3] != "2" {
				t.Fatalf("an SLH-DSA-256f block cut on size holds %s transactions, want 2", row[3])
			}
		}
	}
	if tx != 25 || blocks != 13 || cuts["size"] != 12 || cuts["timeout"] != 1 {
		t.Fatalf("tx %d, blocks %d, cuts %v", tx, blocks, cuts)
	}
	_, rows = readGz(t, filepath.Join(dir, "raw/E14/ml-dsa-44/burst-r2.csv.gz"))
	cuts = map[string]int{}
	for _, row := range rows[1:] {
		if row[0] == "block" {
			cuts[row[6]]++
		}
	}
	if cuts["count"] != 2 || cuts["timeout"] != 1 {
		t.Fatalf("ML-DSA-44 burst cuts %v", cuts)
	}
	_, rows = readGz(t, filepath.Join(dir, "raw/E14/ml-dsa-44/sequential-r1.csv.gz"))
	for _, row := range rows[1:] {
		if row[0] == "block" && (row[3] != "1" || row[6] != "timeout") {
			t.Fatalf("sequential block row %v", row)
		}
	}
	tab, err := os.ReadFile(filepath.Join(dir, "blocks.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(tab), "\n"); n != len(r.Blocks)+1 {
		t.Fatalf("blocks.csv has %d lines", n)
	}
	if err := writeSums(dir); err != nil {
		t.Fatal(err)
	}
	sums, _ := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if !strings.Contains(string(sums), "  raw/E14/tiny/drift-d1.csv.gz\n") || strings.Contains(string(sums), "SHA256SUMS") {
		t.Fatalf("SHA256SUMS:\n%s", sums)
	}
}

func TestFakeBatches(t *testing.T) {
	f := newFake()
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			f.Put(context.Background(), fmt.Sprint("k", i), strings.Repeat("ab", 51000))
		}(i)
	}
	wg.Wait()
	if len(f.blocks) != 13 {
		t.Fatalf("%d blocks for 25 transactions of about 207 KB", len(f.blocks))
	}
}

func TestRunnerFailuresAndStop(t *testing.T) {
	f := newFake()
	f.failAt = 30
	r, _ := newRunner(t, f)
	if err := r.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "simulated failure") {
		t.Fatalf("a failed endorsement did not stop the run: %v", err)
	}
	f = newFake()
	f.corrupt = true
	r, _ = newRunner(t, f)
	if err := r.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("a wrong read-back was not caught: %v", err)
	}
	f = newFake()
	r, _ = newRunner(t, f)
	r.Stop.Store(true)
	if err := r.Run(context.Background()); err != errStopped {
		t.Fatalf("stop: %v", err)
	}
}
