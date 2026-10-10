package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"qchain-kpi/internal/record"
)

// File names inside a run folder (see docs/data-dictionary.md).
const (
	fileState     = "state.json"
	fileManifest  = "manifest.json"
	fileJournal   = "journal.csv"
	fileBlocksJL  = "blocks.jsonl"
	fileBlocksCSV = "blocks.csv"
	fileCanaryJL  = "canary.jsonl"
	fileCanaryCSV = "canary.csv"
	fileQuality   = "quality.csv"
	filePrepCSV   = "prep.csv"
	fileSums      = "SHA256SUMS"
	fileCalib     = "calibration.json"
	fileEstimate  = "estimate.json"
	fileStop      = "STOP"
	dirMonitor    = "monitor"
	dirEnv        = "env"
)

// csvLine encodes one CSV record (RFC 4180 quoting) without the newline.
func csvLine(fields []string) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(fields)
	w.Flush()
	return strings.TrimSuffix(buf.String(), "\n")
}

// appendCSV appends one row, writing the header first if the file is new.
func appendCSV(path string, header, row []string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := record.AppendLineDurable(path, csvLine(header)); err != nil {
			return err
		}
	}
	return record.AppendLineDurable(path, csvLine(row))
}

// writeCSV replaces a whole CSV file atomically.
func writeCSV(path string, header []string, rows [][]string) error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(header)
	w.WriteAll(rows)
	if err := w.Error(); err != nil {
		return err
	}
	return record.WriteFileAtomic(path, buf.Bytes(), 0o644)
}

// writeJSON replaces a JSON file atomically (indented, trailing newline).
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return record.WriteFileAtomic(path, append(b, '\n'), 0o644)
}

// appendJSONL appends one compact JSON line durably.
func appendJSONL(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return record.AppendLineDurable(path, string(b))
}

// readJSONL reads every line of a JSON-lines file. A missing file is empty.
// A final line without a newline (an append cut short by a crash) is
// ignored; any other malformed line is an error.
func readJSONL[T any](path string) ([]T, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []T
	complete := bytes.HasSuffix(b, []byte("\n"))
	lines := bytes.Split(bytes.TrimSuffix(b, []byte("\n")), []byte("\n"))
	for i, l := range lines {
		if len(bytes.TrimSpace(l)) == 0 {
			continue
		}
		var v T
		if err := json.Unmarshal(l, &v); err != nil {
			if i == len(lines)-1 && !complete {
				break
			}
			return nil, fmt.Errorf("%s line %d: %w", path, i+1, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// fileHash returns the SHA-256 of a file.
func fileHash(path string) (string, error) {
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

// listFiles returns every regular file under root (slash-separated relative
// paths, sorted), skipping temporary files and the given names.
func listFiles(root string, skip map[string]bool) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if skip[rel] || strings.HasPrefix(d.Name(), ".tmp-") {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	sort.Strings(out)
	return out, err
}

// writeSums writes SHA256SUMS ("<hash>  <path>", the format of shasum -c and
// sha256sum -c) for every file in the run folder.
func writeSums(root string) (int, error) {
	files, err := listFiles(root, map[string]bool{fileSums: true, fileStop: true})
	if err != nil {
		return 0, err
	}
	var buf bytes.Buffer
	for _, f := range files {
		if strings.ContainsAny(f, " \n\\") {
			return 0, fmt.Errorf("file name %q cannot be listed in SHA256SUMS", f)
		}
		h, err := fileHash(filepath.Join(root, f))
		if err != nil {
			return 0, err
		}
		fmt.Fprintf(&buf, "%s  %s\n", h, f)
	}
	return len(files), record.WriteFileAtomic(filepath.Join(root, fileSums), buf.Bytes(), 0o644)
}

// verifySums checks every listed file and reports unlisted ones.
func verifySums(root string) (checked int, unlisted []string, err error) {
	f, err := os.Open(filepath.Join(root, fileSums))
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()
	listed := map[string]bool{}
	var bad []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		h, name, ok := strings.Cut(line, "  ")
		if !ok || len(h) != 64 {
			return checked, nil, fmt.Errorf("malformed line %q", line)
		}
		listed[name] = true
		got, err := fileHash(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			bad = append(bad, name+" (missing)")
			continue
		}
		if got != h {
			bad = append(bad, name+" (checksum mismatch)")
			continue
		}
		checked++
	}
	if err := sc.Err(); err != nil {
		return checked, nil, err
	}
	all, err := listFiles(root, map[string]bool{fileSums: true})
	if err != nil {
		return checked, nil, err
	}
	for _, a := range all {
		if !listed[a] {
			unlisted = append(unlisted, a)
		}
	}
	if len(bad) > 0 {
		return checked, unlisted, fmt.Errorf("%d file(s) failed: %s", len(bad), strings.Join(bad, "; "))
	}
	return checked, unlisted, nil
}

// copyDir copies the regular files of src (one level) into dst.
func copyDir(src, dst string) error {
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.Type().IsRegular() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := record.WriteFileAtomic(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
