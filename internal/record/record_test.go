package record

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRawWriterRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "raw", "E1.x.sign.r1.a1.csv.gz")
	w, err := NewRawWriter(p, map[string]string{"block": "E1.x.sign.r1.a1", "alg": "x"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	w.Row("cold:reuse", 1234, 666, true)
	w.Row("reuse", 99, 660, true)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("final file must not exist before Close")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	sc := bufio.NewScanner(gz)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	want := []string{"# alg=x", "# block=E1.x.sign.r1.a1", "arm,ns,bytes", "cold:reuse,1234,666", "reuse,99,660"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q", lines)
	}
	// No temp files left behind.
	ents, _ := os.ReadDir(filepath.Dir(p))
	if len(ents) != 1 {
		t.Fatalf("leftover files: %v", ents)
	}
}

func TestAbortLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRawWriter(filepath.Join(dir, "a.csv.gz"), nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	w.Row("reuse", 1, 0, true)
	w.Abort()
	ents, _ := os.ReadDir(dir)
	if len(ents) != 0 {
		t.Fatalf("abort left files: %v", ents)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x", "m.json")
	if err := WriteFileAtomic(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "{}" {
		t.Fatalf("content %q", b)
	}
}
