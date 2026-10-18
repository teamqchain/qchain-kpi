//go:build linux

// Package memprobe measures memory for E9:
//   - the stack high-water mark of a C operation, run on a dedicated thread
//     whose stack is painted with a pattern before every call (runner.c);
//   - C-heap use per call, from per-thread counters kept by csrc/kpi_malloc.c,
//     which E9's measuring processes load with LD_PRELOAD;
//   - the process's peak resident memory (Linux /proc).
//
// Nothing here is timed.
package memprobe

/*
#cgo LDFLAGS: -ldl -lpthread
#include <stdlib.h>
#include "runner.h"
*/
import "C"

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"
)

func heapOf(h C.kpi_heap) Heap {
	return Heap{Live: int64(h.live), Peak: int64(h.peak), Bytes: int64(h.bytes),
		Requested: int64(h.requested), Allocs: int64(h.allocs), Frees: int64(h.frees)}
}

// Counting reports whether the heap counter is loaded into this process.
func Counting() bool { return C.kpi_heap_available() == 1 }

// Preloaded runs f in a process with the heap counter loaded. If the counter
// is missing and LibEnv names a library, it re-executes the current binary
// with LD_PRELOAD set and returns the child's exit code (tests use this).
func Preloaded(f func() int) int {
	lib := os.Getenv(LibEnv)
	if Counting() || lib == "" || os.Getenv("KPI_MEMPROBE_REEXEC") != "" {
		return f()
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "memprobe:", err)
		return 2
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "LD_PRELOAD="+lib, "KPI_MEMPROBE_REEXEC=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "memprobe:", err)
		return 2
	}
	return 0
}

// ThreadHeap returns the calling OS thread's counters. The caller must have
// locked its goroutine to the thread (runtime.LockOSThread), or the reading
// may come from another thread.
func ThreadHeap() Heap {
	var h C.kpi_heap
	C.kpi_heap_read(&h)
	return heapOf(h)
}

// ResetPeak sets the calling OS thread's peak to its live bytes.
func ResetPeak() { C.kpi_heap_reset_peak() }

// Runner is a dedicated C thread with a painted stack.
type Runner struct{ r *C.kpi_runner }

// NewRunner starts a runner thread with a stack of size bytes.
func NewRunner(size int) (*Runner, error) {
	r := C.kpi_runner_new(C.size_t(size))
	if r == nil {
		return nil, errors.New("memprobe: cannot start the runner thread")
	}
	return &Runner{r}, nil
}

// Run runs one job on the runner thread and waits for it.
func (r *Runner) Run(j Job) Result {
	var out C.kpi_job_result
	C.kpi_runner_run(r.r, C.kpi_job_fn(j.Fn), j.Arg, &out)
	return Result{RC: int(out.rc), Stack: int64(out.stack), Heap: Delta(heapOf(out.before), heapOf(out.after))}
}

// Close stops the runner thread and frees its stack.
func (r *Runner) Close() {
	if r.r != nil {
		C.kpi_runner_free(r.r)
		r.r = nil
	}
}

// Empty is the job that does nothing: its stack measurement is the runner's
// baseline, subtracted from every other job's.
var Empty = Job{Fn: unsafe.Pointer(C.kpi_job_empty)}

// ------------------------------------------------------------ resident memory

// ResetPeakRSS sets the process's peak resident set (VmHWM) to its current
// resident set (Linux: "5" to /proc/self/clear_refs).
func ResetPeakRSS() error { return os.WriteFile("/proc/self/clear_refs", []byte("5"), 0) }

// RSS returns the process's resident set and its peak (VmRSS, VmHWM) in bytes.
// Since Linux 6.2 the kernel keeps these in per-CPU counters and reads them
// approximately (up to 32 pages per CPU the process ran on), so a difference
// of two readings is precise to about 128 KiB per CPU; the C- and Go-heap
// figures are exact.
func RSS() (rss, hwm int64, err error) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok || (k != "VmRSS" && k != "VmHWM") {
			continue
		}
		fields := strings.Fields(v)
		if len(fields) != 2 || fields[1] != "kB" {
			return 0, 0, fmt.Errorf("memprobe: unexpected %s line %q", k, v)
		}
		n, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		if k == "VmRSS" {
			rss = n * 1024
		} else {
			hwm = n * 1024
		}
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}
	if rss == 0 || hwm == 0 {
		return 0, 0, errors.New("memprobe: VmRSS/VmHWM not found in /proc/self/status")
	}
	return rss, hwm, nil
}

// ------------------------------------------------------------- test helpers

// The method's own checks (memprobe_test.go): cgo is not allowed in test files.

func testStackJob() Job { return Job{Fn: unsafe.Pointer(C.kpi_job_test_stack)} }

// testMallocJob returns the job and the C slot it stores its kept block in.
func testMallocJob() (Job, unsafe.Pointer) {
	slot := C.calloc(1, C.size_t(unsafe.Sizeof(uintptr(0))))
	return Job{Fn: unsafe.Pointer(C.kpi_job_test_malloc), Arg: slot}, slot
}

func testFreeSlot(slot unsafe.Pointer) {
	C.kpi_test_free(*(*unsafe.Pointer)(slot))
	C.free(slot)
}

func testMallocHere(n int) unsafe.Pointer { return C.kpi_test_malloc_here(C.size_t(n)) }

func testFree(p unsafe.Pointer) { C.kpi_test_free(p) }
