package main

/*
#include <time.h>
#include <gnu/libc-version.h>

static long long kpi_clock_res_ns(void) {
	struct timespec ts;
	if (clock_getres(CLOCK_MONOTONIC, &ts) != 0) return -1;
	return (long long)ts.tv_sec * 1000000000LL + ts.tv_nsec;
}
*/
import "C"

// clockResNS is clock_getres(CLOCK_MONOTONIC), the clock Go's monotonic time uses.
func clockResNS() int64 { return int64(C.kpi_clock_res_ns()) }

// glibcVersion is the C library the binary runs against.
func glibcVersion() string { return C.GoString(C.gnu_get_libc_version()) }
