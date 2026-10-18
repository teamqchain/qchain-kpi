//go:build !linux

package memprobe

import (
	"errors"
	"unsafe"
)

// E9's memory measurements need Linux: the LD_PRELOAD heap counter, the
// painted-stack runner and /proc. On other systems (E5's native macOS runs)
// the package compiles but reports that nothing is available; E9 and E12
// refuse to run there.

var errUnsupported = errors.New("memprobe: memory measurement needs Linux")

// Counting is always false: the heap counter is a Linux LD_PRELOAD library.
func Counting() bool { return false }

// Preloaded runs f directly.
func Preloaded(f func() int) int { return f() }

// ThreadHeap returns zero counters.
func ThreadHeap() Heap { return Heap{} }

// ResetPeak does nothing.
func ResetPeak() {}

// Runner is not available.
type Runner struct{}

// NewRunner fails.
func NewRunner(int) (*Runner, error) { return nil, errUnsupported }

// Run is never reached (NewRunner fails).
func (r *Runner) Run(Job) Result { return Result{RC: -1} }

// Close does nothing.
func (r *Runner) Close() {}

// Empty is the empty job (no function on this system).
var Empty = Job{Fn: unsafe.Pointer(nil)}

// ResetPeakRSS fails.
func ResetPeakRSS() error { return errUnsupported }

// RSS fails.
func RSS() (rss, hwm int64, err error) { return 0, 0, errUnsupported }
