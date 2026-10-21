package main

// Files of an E14 run, in the harness's conventions (docs/data-dictionary.md):
// writes go to a temporary file that is synced and renamed; tables are CSV
// with a header; raw files are gzip-compressed CSV with "# key=value" header
// lines; SHA256SUMS is in `shasum -c` format and written last.

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// writeFileAtomic writes data to path through a synced temporary file.
func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(b, '\n'))
}

// appendCSV appends one row, writing the header first if the file is new.
func appendCSV(path string, header, row []string) error {
	_, err := os.Stat(path)
	isNew := os.IsNotExist(err)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if isNew {
		w.Write(header)
	}
	w.Write(row)
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// gzipCSV renders header lines ("# k=v", sorted by key) and rows as gzip.
func gzipCSV(header map[string]string, rows [][]string) ([]byte, error) {
	var plain bytes.Buffer
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&plain, "# %s=%s\n", k, strings.ReplaceAll(header[k], "\n", " "))
	}
	w := csv.NewWriter(&plain)
	if err := w.WriteAll(rows); err != nil {
		return nil, err
	}
	var gz bytes.Buffer
	zw, err := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(plain.Bytes()); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return gz.Bytes(), nil
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// writeSums writes SHA256SUMS for every file under dir except itself and the
// temporary files.
func writeSums(dir string) error {
	var lines []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "SHA256SUMS" || strings.HasSuffix(rel, ".tmp") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		lines = append(lines, sha256Hex(b)+"  "+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i][66:] < lines[j][66:] })
	return writeFileAtomic(filepath.Join(dir, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"))
}

// Summary of a sample of durations (nanoseconds).
type Summary struct {
	N    int
	Min  float64
	P50  float64
	P90  float64
	P99  float64
	Max  float64
	Mean float64
}

// Summarise sorts a copy of xs. The median of an even count is the mean of
// the two middle values; other percentiles are nearest-rank (the value at
// position ceil(q n)).
func Summarise(xs []float64) Summary {
	if len(xs) == 0 {
		nan := math.NaN()
		return Summary{Min: nan, P50: nan, P90: nan, P99: nan, Max: nan, Mean: nan}
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	rank := func(q float64) float64 {
		i := int(math.Ceil(q*float64(n))) - 1
		if i < 0 {
			i = 0
		}
		return s[i]
	}
	med := s[n/2]
	if n%2 == 0 {
		med = (s[n/2-1] + s[n/2]) / 2
	}
	sum := 0.0
	for _, x := range s {
		sum += x
	}
	return Summary{N: n, Min: s[0], P50: med, P90: rank(0.90), P99: rank(0.99), Max: s[n-1], Mean: sum / float64(n)}
}

func ff(x float64) string {
	if math.IsNaN(x) {
		return ""
	}
	return fmt.Sprintf("%.0f", x)
}
