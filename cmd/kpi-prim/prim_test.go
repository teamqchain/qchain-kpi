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
