//go:build darwin

package main

/*
#include <time.h>

static long long kpi_clock_res_ns(void) {
	struct timespec ts;
	if (clock_getres(CLOCK_MONOTONIC, &ts) != 0) return -1;
	return (long long)ts.tv_sec * 1000000000LL + ts.tv_nsec;
}
*/
import "C"

// clockResNS is clock_getres(CLOCK_MONOTONIC).
func clockResNS() int64 { return int64(C.kpi_clock_res_ns()) }

// glibcVersion: macOS has no glibc; its C library is libSystem.
func glibcVersion() string { return "none (macOS libSystem)" }
