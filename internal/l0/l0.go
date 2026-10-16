// Package l0 is E3's raw-C layer: an operation prepared as a C job (the
// liboqs and OpenSSL jobs of internal/pqc and internal/classical, also used
// by E9) is timed inside C with CLOCK_MONOTONIC, the clock Go's time.Now
// reads, so L0 and the Go-timed layers share one time base.
package l0

/*
#include <stdlib.h>
#include "l0.h"
*/
import "C"

import "unsafe"

// Time runs the C job fn(arg) between two clock readings taken in C and
// returns the elapsed nanoseconds and the job's result.
func Time(fn, arg unsafe.Pointer) (ns int64, rc int) {
	var r C.int
	d := C.kpi_l0_time(C.kpi_job_fn(fn), arg, &r)
	return int64(d), int(r)
}

// Call runs the C job without timing it (the L0g arm is timed from Go
// around this call: L0g - L0 is the Go-to-C boundary).
func Call(fn, arg unsafe.Pointer) int { return int(C.kpi_l0_call(C.kpi_job_fn(fn), arg)) }

// Empty is the C timer's floor: two readings with nothing between them.
func Empty() int64 { return int64(C.kpi_l0_empty()) }

// Null makes a cgo call to a C function that does nothing.
func Null() { C.kpi_l0_null() }

// testSleep returns a job that sleeps for us microseconds, with its argument
// in C memory, and the function that frees it (cgo is not allowed in test files).
func testSleep(us int64) (fn, arg unsafe.Pointer, free func()) {
	p := C.malloc(C.size_t(unsafe.Sizeof(C.long(0))))
	*(*C.long)(p) = C.long(us)
	return unsafe.Pointer(C.kpi_l0_test_sleep), p, func() { C.free(p) }
}
