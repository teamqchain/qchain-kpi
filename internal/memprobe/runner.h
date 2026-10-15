/*
 * E9 memory runner: runs C operations on a dedicated thread whose stack is
 * painted with a pattern before every operation, so the deepest stack byte
 * the operation wrote can be found afterwards (its stack high-water mark), and
 * reads that thread's C-heap counters (csrc/kpi_malloc.c) around it.
 */
#ifndef KPI_MEMPROBE_RUNNER_H
#define KPI_MEMPROBE_RUNNER_H

#include <stddef.h>

/* The counters of csrc/kpi_malloc.c (same layout). */
typedef struct {
    long long live, peak, bytes, requested, allocs, frees;
} kpi_heap;

/* A job is one call: fn(arg), returning the operation's own status code. */
typedef int (*kpi_job_fn)(void *);

typedef struct kpi_runner kpi_runner;

/* Result of one job. stack = bytes between the painted top and the deepest
 * byte written; before/after = the thread's counters around the call, with
 * before.peak reset to before.live, so after.peak - before.live is the peak
 * the call reached above what was live when it started. */
typedef struct {
    int rc;
    long long stack;
    kpi_heap before, after;
} kpi_job_result;

int kpi_heap_available(void);
/* The calling thread's counters (all zero when the counter is not loaded). */
void kpi_heap_read(kpi_heap *out);
/* Sets the calling thread's peak to its live bytes. */
void kpi_heap_reset_peak(void);

/* Creates the runner thread with a stack of stack_size bytes (plus a guard
 * page). Returns NULL on failure. */
kpi_runner *kpi_runner_new(size_t stack_size);
void kpi_runner_run(kpi_runner *r, kpi_job_fn fn, void *arg, kpi_job_result *out);
void kpi_runner_free(kpi_runner *r);

/* Jobs for the baseline and the method's own tests. */
int kpi_job_empty(void *arg);
int kpi_job_test_stack(void *arg);  /* writes a 64 KiB local array */
int kpi_job_test_malloc(void *arg); /* malloc 1000 (kept, stored in *arg), malloc 100000 and free it */
void *kpi_test_malloc_here(size_t n); /* malloc on the calling thread (Go-side tests) */
void kpi_test_free(void *p);

#endif
