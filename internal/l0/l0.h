/*
 * E3's raw-C layer (L0): a C operation timed inside C, so the measurement
 * contains the library call and nothing of the Go-to-C boundary.
 */
#ifndef KPI_L0_H
#define KPI_L0_H

#include <stdint.h>

typedef int (*kpi_job_fn)(void *);

/* Runs fn(arg) between two CLOCK_MONOTONIC readings (the clock Go's own
 * timer reads) and returns the elapsed nanoseconds; *rc gets fn's result. */
int64_t kpi_l0_time(kpi_job_fn fn, void *arg, int *rc);
/* Runs fn(arg) without timing (E3's L0g arm: timed from Go around this call). */
int kpi_l0_call(kpi_job_fn fn, void *arg);
/* Two clock readings with nothing between them: the C timer's floor. */
int64_t kpi_l0_empty(void);
/* Does nothing: the bare cost of a cgo call. */
void kpi_l0_null(void);
/* Test job: sleeps for *(long *)arg microseconds and returns 7. */
int kpi_l0_test_sleep(void *arg);

#endif
