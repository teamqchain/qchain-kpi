package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/timing"
)

// capture runs f with os.Stdout redirected and returns what it printed.
func capture(t *testing.T, f func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	ferr := f()
	w.Close()
	os.Stdout = old
	return <-done, ferr
}

func prep(t *testing.T, dir, alg string, rep, pool int, e10 bool) proto.PrepResult {
	t.Helper()
	args := []string{"--alg", alg, "--rep", strconv.Itoa(rep), "--pool", strconv.Itoa(pool), "--seed", "42", "--dir", dir}
	if e10 {
		args = append(args, "--e10")
	}
	out, err := capture(t, func() error { return cmdPrep(args) })
	if err != nil {
		t.Fatalf("prep %s: %v", alg, err)
	}
	var pr proto.PrepResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &pr); err != nil {
		t.Fatalf("prep output %q: %v", out, err)
	}
	return pr
}

func block(t *testing.T, sp proto.Spec) (proto.Result, error) {
	t.Helper()
	js, _ := json.Marshal(sp)
	out, err := capture(t, func() error { return cmdBlock([]string{"--spec", string(js)}) })
	if err != nil {
		return proto.Result{}, err
	}
	var r proto.Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &r); err != nil {
		t.Fatalf("block output %q: %v", out, err)
	}
	return r, nil
}

// rawRows reads a raw file and returns its header map and data rows.
func rawRows(t *testing.T, path string) (map[string]string, []string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	h := map[string]string{}
	var rows []string
	sc := bufio.NewScanner(gz)
	first := true
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, "# ") {
			k, v, _ := strings.Cut(strings.TrimPrefix(l, "# "), "=")
			h[k] = v
			continue
		}
		if first {
			first = false // column line
			continue
		}
		rows = append(rows, l)
	}
	return h, rows
}

func TestPrepAndBlocks(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	cases := []struct {
		alg  string
		ops  []string
		slow map[string]bool
	}{
		{"ml-dsa-44", []string{"keygen", "sign", "verify"}, nil},
		{"falcon-512", []string{"sign", "verify"}, nil},
		{"ml-kem-768", []string{"keygen", "encaps", "decaps"}, nil},
		{"ecdsa-p256", []string{"keygen", "sign", "verify"}, nil},
		{"rsa3072-pss", []string{"keygen", "sign"}, map[string]bool{"keygen": true, "sign": true}},
		{"dhkem-x25519", []string{"keygen", "encaps", "decaps"}, nil},
	}
	for _, c := range cases {
		pr := prep(t, dir, c.alg, 1, 8, c.alg == "ml-dsa-44")
		if pr.Pool != 8 || pr.Fingerprint == "" || pr.E10 != (c.alg == "ml-dsa-44") {
			t.Fatalf("prep result %+v", pr)
		}
		for _, op := range c.ops {
			mode := "fast"
			if c.slow[op] {
				mode = "slow"
			}
			b := plan.Block{ID: "E1." + c.alg + "." + op + ".r1", Exp: "E1", Alg: c.alg, Op: op, Rep: 1, Attempt: 1,
				Profile: plan.ProfileSmoke, Mode: mode, Seed: 7}
			raw := filepath.Join(out, b.RawPath())
			r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
			if err != nil {
				t.Fatalf("%s: %v", b.Key(), err)
			}
			a := r.Arms[0]
			if r.Key != b.Key() || len(r.Arms) != 1 || a.Name != proto.ArmReuse || a.FirstLabel != "cold" || a.FirstNS <= 0 {
				t.Fatalf("%s: result %+v", b.Key(), r)
			}
			if mode == "fast" && a.Stats.N != 200 {
				t.Fatalf("%s: %d samples, want 200", b.Key(), a.Stats.N)
			}
			if mode == "slow" && a.Stats.N < 3 {
				t.Fatalf("%s: %d samples, want at least 3", b.Key(), a.Stats.N)
			}
			if op == "keygen" && r.KeysChecked < 3 {
				t.Fatalf("%s: only %d keys fully checked", b.Key(), r.KeysChecked)
			}
			h, rows := rawRows(t, raw)
			if len(rows) != a.Stats.N+1 || r.Rows != len(rows) || !strings.HasPrefix(rows[0], "cold:0,") {
				t.Fatalf("%s: raw rows %d (first %q), result rows %d", b.Key(), len(rows), rows[0], r.Rows)
			}
			if h["key"] != b.Key() || h["prep_id"] != pr.Fingerprint || h["pool"] != "8" {
				t.Fatalf("%s: raw header %v", b.Key(), h)
			}
			if got, _ := fileSHA256(raw); got != r.RawSHA256 {
				t.Fatalf("%s: raw checksum mismatch", b.Key())
			}
		}
	}

	// E10: three interleaved arms with their own pools.
	b := plan.Block{ID: "E10.ml-dsa-44.sign.r1", Exp: "E10", Alg: "ml-dsa-44", Op: "sign", Rep: 1, Attempt: 1,
		Profile: plan.ProfileSmoke, Mode: "fast", Seed: 9}
	r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Arms) != 3 || r.Arms[0].Name != proto.ArmIssuer || r.Arms[1].Name != proto.ArmHolder ||
		r.Arms[2].Name != proto.ArmIssuerAgain || r.Arms[0].FirstLabel != "first" {
		t.Fatalf("E10 arms %+v", r.Arms)
	}
	_, rows := rawRows(t, filepath.Join(out, b.RawPath()))
	if len(rows) != 3*201 {
		t.Fatalf("E10 raw rows %d", len(rows))
	}

	// Probe: its own small input set, cold call + 5 calls.
	prep(t, dir, "ml-dsa-44", proto.ProbeRep, proto.ProbeCalls+1, false)
	b = plan.Block{ID: "probe.ml-dsa-44.sign.r1", Exp: "probe", Alg: "ml-dsa-44", Op: "sign", Rep: 1, Attempt: 1,
		Profile: plan.ProfileApproved, Seed: 3}
	r, err = block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Arms[0].ProbeNS) != proto.ProbeCalls || r.Arms[0].FirstNS <= 0 {
		t.Fatalf("probe %+v", r.Arms[0])
	}
}

func TestCorrectnessFailureIsReported(t *testing.T) {
	dir := t.TempDir()
	prep(t, dir, "ml-kem-768", 1, 8, false)
	path := proto.PrepPath(dir, "ml-kem-768", 1)
	p, err := inputs.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p.SSs[3][0] ^= 1 // one wrong expected secret
	if err := inputs.Save(path, p); err != nil {
		t.Fatal(err)
	}
	b := plan.Block{ID: "E1.ml-kem-768.decaps.r1", Exp: "E1", Alg: "ml-kem-768", Op: "decaps", Rep: 1, Attempt: 1,
		Profile: plan.ProfileSmoke, Mode: "fast", Seed: 1}
	raw := filepath.Join(t.TempDir(), b.RawPath())
	_, err = block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
	if !errors.Is(err, timing.ErrCheck) {
		t.Fatalf("want a correctness failure, got %v", err)
	}
	if _, err := os.Stat(raw); !os.IsNotExist(err) {
		t.Fatal("a raw file was written for an invalid block")
	}
}
