package record

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RawWriter writes one block's samples as a gzipped CSV:
//
//	# key=value header lines (block identity, arm list, batch sizes, ...)
//	arm,ns[,bytes][,ok]
//
// The sample index is implicit (row order). "arm" is a short label; a row whose
// arm is "cold:<arm>" or "first:<arm>" is the very first call of that arm in
// the process. "bytes" is only present when the output size varies (e.g.
// Falcon or ECDSA signatures); "ok" only when an outcome may legitimately be a
// rejection (E7). The file is renamed into place only by Close.
type RawWriter struct {
	final    string
	tmp      *os.File
	gz       *gzip.Writer
	bw       *bufio.Writer
	hasBytes bool
	hasOK    bool
	rows     int
	closed   bool
}

// NewRawWriter creates the writer. header keys are written sorted.
func NewRawWriter(path string, header map[string]string, hasBytes, hasOK bool) (*RawWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	w := &RawWriter{final: path, tmp: f, gz: gz, bw: bufio.NewWriterSize(gz, 1<<16), hasBytes: hasBytes, hasOK: hasOK}
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := strings.ReplaceAll(header[k], "\n", " ")
		fmt.Fprintf(w.bw, "# %s=%s\n", k, v)
	}
	cols := "arm,ns"
	if hasBytes {
		cols += ",bytes"
	}
	if hasOK {
		cols += ",ok"
	}
	w.bw.WriteString(cols + "\n")
	return w, nil
}

// Row appends one sample.
func (w *RawWriter) Row(arm string, ns int64, nbytes int, ok bool) {
	w.bw.WriteString(arm)
	w.bw.WriteByte(',')
	w.bw.WriteString(strconv.FormatInt(ns, 10))
	if w.hasBytes {
		w.bw.WriteByte(',')
		w.bw.WriteString(strconv.Itoa(nbytes))
	}
	if w.hasOK {
		if ok {
			w.bw.WriteString(",1")
		} else {
			w.bw.WriteString(",0")
		}
	}
	w.bw.WriteByte('\n')
	w.rows++
}

// Rows returns the number of sample rows written.
func (w *RawWriter) Rows() int { return w.rows }

// Close flushes, syncs and renames the file into place.
func (w *RawWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	tmp := w.tmp.Name()
	if err := w.bw.Flush(); err != nil {
		w.tmp.Close()
		os.Remove(tmp)
		return err
	}
	if err := w.gz.Close(); err != nil {
		w.tmp.Close()
		os.Remove(tmp)
		return err
	}
	if err := w.tmp.Sync(); err != nil {
		w.tmp.Close()
		os.Remove(tmp)
		return err
	}
	if err := w.tmp.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, w.final); err != nil {
		os.Remove(tmp)
		return err
	}
	return syncDir(filepath.Dir(w.final))
}

// Abort discards the partial file.
func (w *RawWriter) Abort() {
	if w.closed {
		return
	}
	w.closed = true
	w.tmp.Close()
	os.Remove(w.tmp.Name())
}
