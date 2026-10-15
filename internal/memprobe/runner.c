#define _GNU_SOURCE
#include "runner.h"

#include <dlfcn.h>
#include <pthread.h>
#include <signal.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>

/* ------------------------------------------------------------ heap counter */

typedef kpi_heap *(*heap_self_fn)(void);
static heap_self_fn heap_self;
static pthread_once_t heap_once = PTHREAD_ONCE_INIT;

static void heap_lookup(void) { heap_self = (heap_self_fn)dlsym(RTLD_DEFAULT, "kpi_heap_self"); }

static kpi_heap *heap_this(void) {
    pthread_once(&heap_once, heap_lookup);
    return heap_self ? heap_self() : NULL;
}

int kpi_heap_available(void) { return heap_this() != NULL; }

void kpi_heap_read(kpi_heap *out) {
    kpi_heap *h = heap_this();
    if (h == NULL) {
        memset(out, 0, sizeof *out);
        return;
    }
    *out = *h;
}

void kpi_heap_reset_peak(void) {
    kpi_heap *h = heap_this();
    if (h != NULL) {
        h->peak = h->live;
    }
}

/* ------------------------------------------------------------------ runner */

/* The paint pattern ("kpi-mem9"), written as 64-bit words. */
#define KPI_PAINT 0x6b70692d6d656d39ULL
/* Unpainted bytes directly below the runner's own frame (the return
 * addresses of the helpers below live there). */
#define KPI_GAP 64

enum { R_IDLE, R_POSTED, R_DONE, R_QUIT };

struct kpi_runner {
    pthread_t th;
    pthread_mutex_t mu;
    pthread_cond_t cv;
    void *map;
    size_t maplen;
    uint64_t *lo; /* lowest word of the stack (just above the guard page) */
    int state;
    kpi_job_fn fn;
    void *arg;
    kpi_job_result res;
};

/* The address of a byte in a frame one call below the caller's: where the
 * frames of the caller's callees begin. */
__attribute__((noinline)) static uintptr_t below_here(void) {
    volatile char c = 0;
    return (uintptr_t)&c;
}

__attribute__((noinline)) static void paint(uint64_t *lo, uint64_t *hi) {
    for (volatile uint64_t *w = lo; w < hi; w++) {
        *w = KPI_PAINT;
    }
}

__attribute__((noinline)) static uint64_t *lowest_written(uint64_t *lo, uint64_t *hi) {
    volatile uint64_t *w = lo;
    while (w < hi && *w == KPI_PAINT) {
        w++;
    }
    return (uint64_t *)w;
}

/* invoke writes a 256-byte pad that always reaches below the painted top, so
 * even an empty job measures a fixed, non-zero depth (the baseline). A job's
 * own stack is its measurement minus the baseline. The write after the call
 * keeps the compiler from turning the call into a tail call (which would
 * start the job's frames above the pad). */
__attribute__((noinline)) static int invoke(kpi_job_fn fn, void *arg) {
    volatile uint64_t pad[32];
    for (int i = 0; i < 32; i++) {
        pad[i] = (uint64_t)i;
    }
    int rc = fn(arg);
    pad[1] = (uint64_t)rc;
    return rc;
}

static void *runner_main(void *p) {
    kpi_runner *r = p;
    /* Signals go to Go's threads, never into the measured stack. */
    sigset_t all;
    sigfillset(&all);
    pthread_sigmask(SIG_BLOCK, &all, NULL);
    kpi_heap *h = heap_this();
    uint64_t *hi = (uint64_t *)((below_here() - KPI_GAP) & ~(uintptr_t)7);

    pthread_mutex_lock(&r->mu);
    for (;;) {
        while (r->state != R_POSTED && r->state != R_QUIT) {
            pthread_cond_wait(&r->cv, &r->mu);
        }
        if (r->state == R_QUIT) {
            break;
        }
        kpi_job_fn fn = r->fn;
        void *arg = r->arg;
        pthread_mutex_unlock(&r->mu);

        paint(r->lo, hi);
        kpi_heap before = {0}, after = {0};
        if (h != NULL) {
            h->peak = h->live;
            before = *h;
        }
        int rc = invoke(fn, arg);
        if (h != NULL) {
            after = *h;
        }
        uint64_t *d = lowest_written(r->lo, hi);

        pthread_mutex_lock(&r->mu);
        r->res.rc = rc;
        r->res.stack = (long long)((uintptr_t)hi - (uintptr_t)d);
        r->res.before = before;
        r->res.after = after;
        r->state = R_DONE;
        pthread_cond_broadcast(&r->cv);
    }
    pthread_mutex_unlock(&r->mu);
    return NULL;
}

kpi_runner *kpi_runner_new(size_t stack_size) {
    (void)heap_this(); /* resolve the counter here, not on the measured thread */
    size_t pg = (size_t)sysconf(_SC_PAGESIZE);
    stack_size = (stack_size + pg - 1) & ~(pg - 1);
    kpi_runner *r = calloc(1, sizeof *r);
    if (r == NULL) {
        return NULL;
    }
    r->maplen = stack_size + pg;
    r->map = mmap(NULL, r->maplen, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK, -1, 0);
    if (r->map == MAP_FAILED) {
        free(r);
        return NULL;
    }
    /* Guard page: a job that overflows the stack faults instead of writing
     * into other memory. */
    if (mprotect(r->map, pg, PROT_NONE) != 0) {
        munmap(r->map, r->maplen);
        free(r);
        return NULL;
    }
    r->lo = (uint64_t *)((char *)r->map + pg);
    pthread_mutex_init(&r->mu, NULL);
    pthread_cond_init(&r->cv, NULL);
    pthread_attr_t at;
    pthread_attr_init(&at);
    int err = pthread_attr_setstack(&at, r->lo, stack_size);
    if (err == 0) {
        err = pthread_create(&r->th, &at, runner_main, r);
    }
    pthread_attr_destroy(&at);
    if (err != 0) {
        pthread_cond_destroy(&r->cv);
        pthread_mutex_destroy(&r->mu);
        munmap(r->map, r->maplen);
        free(r);
        return NULL;
    }
    return r;
}

void kpi_runner_run(kpi_runner *r, kpi_job_fn fn, void *arg, kpi_job_result *out) {
    pthread_mutex_lock(&r->mu);
    r->fn = fn;
    r->arg = arg;
    r->state = R_POSTED;
    pthread_cond_broadcast(&r->cv);
    while (r->state != R_DONE) {
        pthread_cond_wait(&r->cv, &r->mu);
    }
    *out = r->res;
    r->state = R_IDLE;
    pthread_mutex_unlock(&r->mu);
}

void kpi_runner_free(kpi_runner *r) {
    pthread_mutex_lock(&r->mu);
    r->state = R_QUIT;
    pthread_cond_broadcast(&r->cv);
    pthread_mutex_unlock(&r->mu);
    pthread_join(r->th, NULL);
    pthread_cond_destroy(&r->cv);
    pthread_mutex_destroy(&r->mu);
    munmap(r->map, r->maplen);
    free(r);
}

/* -------------------------------------------------------------------- jobs */

int kpi_job_empty(void *arg) {
    (void)arg;
    return 0;
}

int kpi_job_test_stack(void *arg) {
    (void)arg;
    volatile unsigned char buf[65536];
    for (size_t i = 0; i < sizeof buf; i++) {
        buf[i] = (unsigned char)(i * 7 + 1);
    }
    return buf[123];
}

/* Called through volatile pointers so the compiler cannot elide the pairs. */
static void *(*volatile malloc_p)(size_t) = malloc;
static void (*volatile free_p)(void *) = free;

int kpi_job_test_malloc(void *arg) {
    void *keep = malloc_p(1000);
    void *tmp = malloc_p(100000);
    if (keep == NULL || tmp == NULL) {
        return -1;
    }
    memset(tmp, 1, 100000);
    free_p(tmp);
    *(void **)arg = keep;
    return 0;
}

void *kpi_test_malloc_here(size_t n) { return malloc_p(n); }

void kpi_test_free(void *p) { free_p(p); }
