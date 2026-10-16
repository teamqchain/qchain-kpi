#include "l0.h"

#include <time.h>

static inline int64_t now_ns(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec * 1000000000LL + (int64_t)ts.tv_nsec;
}

int64_t kpi_l0_time(kpi_job_fn fn, void *arg, int *rc) {
    int64_t t0 = now_ns();
    int r = fn(arg);
    int64_t t1 = now_ns();
    *rc = r;
    return t1 - t0;
}

int kpi_l0_call(kpi_job_fn fn, void *arg) { return fn(arg); }

int64_t kpi_l0_empty(void) {
    int64_t t0 = now_ns();
    int64_t t1 = now_ns();
    return t1 - t0;
}

__attribute__((noinline)) void kpi_l0_null(void) { __asm__ volatile("" ::: "memory"); }

/* Test job: sleeps for *(long *)arg microseconds, returns 7. */
int kpi_l0_test_sleep(void *arg) {
    long us = *(long *)arg;
    struct timespec ts = {us / 1000000, (us % 1000000) * 1000};
    nanosleep(&ts, NULL);
    return 7;
}
