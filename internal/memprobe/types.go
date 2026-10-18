package memprobe

import "unsafe"

// Types shared by the Linux implementation (memprobe.go) and the stub of other
// systems (memprobe_other.go).

// StackSize is the runner thread's stack: 8 MiB, glibc's default thread stack
// on Linux and so what the threads that run QChain's cgo calls get.
const StackSize = 8 << 20

// Heap is one thread's C-heap counters. Bytes, Live and Peak are usable
// bytes (malloc_usable_size); Requested is what callers asked for.
type Heap struct {
	Live, Peak, Bytes, Requested, Allocs, Frees int64
}

// Use is the C heap one call used, from the counters before (with the peak
// reset to the live bytes) and after it.
type Use struct {
	Peak      int64 // highest live bytes during the call, above what was live at its start
	Bytes     int64 // usable bytes allocated
	Requested int64 // bytes requested
	Allocs    int64
	Frees     int64
	Retained  int64 // live bytes after the call minus before (still held: leaked or cached)
}

// Delta is the use between two snapshots of the same thread, the first taken
// right after ResetPeak.
func Delta(before, after Heap) Use {
	return Use{Peak: after.Peak - before.Live, Bytes: after.Bytes - before.Bytes,
		Requested: after.Requested - before.Requested, Allocs: after.Allocs - before.Allocs,
		Frees: after.Frees - before.Frees, Retained: after.Live - before.Live}
}

// LibEnv names the environment variable that holds the counter library's
// path (set in the image).
const LibEnv = "KPI_MALLOC_LIB"

// Job is one C call for the runner: fn(arg), where fn is a C function
// `int fn(void *)` and arg points to C memory.
type Job struct {
	Fn, Arg unsafe.Pointer
}

// Result is one call on the runner.
type Result struct {
	RC    int   // the job's return value
	Stack int64 // painted top to deepest byte written (the baseline included)
	Heap  Use
}
