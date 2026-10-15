/*
 * E9 C-heap counter, loaded with LD_PRELOAD into E9's measuring processes
 * only (no other experiment runs with it).
 *
 * It replaces the C library's allocation functions with wrappers that call
 * glibc's own allocator (__libc_malloc and friends, so the allocator and its
 * behaviour are unchanged) and count, PER THREAD, what that thread allocates
 * and frees. Per-thread counters keep the Go runtime's own threads out of a
 * measurement: an operation measured on one thread sees only its own calls.
 *
 * Sizes are counted twice:
 *   - requested: what the caller asked for (malloc's argument);
 *   - usable:    malloc_usable_size() of the block, which free() can also
 *                look up, so live and peak bytes are kept in usable bytes.
 *
 * A measuring thread reads its counters through kpi_heap_self(), found with
 * dlsym (internal/memprobe); its absence means the library is not loaded.
 *
 * glibc 2.36 (Debian bookworm): the malloc hooks were removed in 2.34, and
 * glibc documents replacing malloc, free, calloc, realloc, aligned_alloc,
 * memalign, posix_memalign, pvalloc and valloc as the supported way to
 * interpose the allocator; it calls the replacements from inside the C
 * library too.
 */
#define _GNU_SOURCE
#include <errno.h>
#include <malloc.h>
#include <stddef.h>
#include <stdint.h>
#include <unistd.h>

extern void *__libc_malloc(size_t);
extern void __libc_free(void *);
extern void *__libc_calloc(size_t, size_t);
extern void *__libc_realloc(void *, size_t);
extern void *__libc_memalign(size_t, size_t);

struct kpi_heap {
    long long live;      /* usable bytes allocated minus freed by this thread */
    long long peak;      /* highest live since the last reset (a reader sets peak = live) */
    long long bytes;     /* usable bytes allocated */
    long long requested; /* bytes requested */
    long long allocs;    /* successful allocations (realloc counts as one) */
    long long frees;     /* blocks freed (realloc of a block counts as one) */
};

/* initial-exec: the library is loaded at start-up, so its TLS is static and
 * reading it never allocates. */
static __thread struct kpi_heap h __attribute__((tls_model("initial-exec")));

__attribute__((visibility("default"))) struct kpi_heap *kpi_heap_self(void) { return &h; }

static inline void on_alloc(void *p, size_t req) {
    if (p == NULL) {
        return;
    }
    long long u = (long long)malloc_usable_size(p);
    h.live += u;
    h.bytes += u;
    h.requested += (long long)req;
    h.allocs++;
    if (h.live > h.peak) {
        h.peak = h.live;
    }
}

static inline void on_free(void *p) {
    if (p == NULL) {
        return;
    }
    h.live -= (long long)malloc_usable_size(p);
    h.frees++;
}

void *malloc(size_t n) {
    void *p = __libc_malloc(n);
    on_alloc(p, n);
    return p;
}

void free(void *p) {
    on_free(p);
    __libc_free(p);
}

void *calloc(size_t n, size_t size) {
    void *p = __libc_calloc(n, size);
    on_alloc(p, n * size); /* __libc_calloc has refused an overflowing product */
    return p;
}

void *realloc(void *old, size_t n) {
    long long before = old ? (long long)malloc_usable_size(old) : 0;
    void *p = __libc_realloc(old, n);
    if (old != NULL && n == 0) { /* glibc frees the block and returns NULL */
        h.live -= before;
        h.frees++;
        return p;
    }
    if (p == NULL) {
        return NULL; /* the old block is untouched */
    }
    if (old != NULL) {
        h.live -= before;
        h.frees++;
    }
    on_alloc(p, n);
    return p;
}

void *memalign(size_t align, size_t n) {
    void *p = __libc_memalign(align, n);
    on_alloc(p, n);
    return p;
}

void *aligned_alloc(size_t align, size_t n) { return memalign(align, n); }

int posix_memalign(void **out, size_t align, size_t n) {
    if (align < sizeof(void *) || (align & (align - 1)) != 0) {
        return EINVAL;
    }
    void *p = memalign(align, n);
    if (p == NULL) {
        return ENOMEM;
    }
    *out = p;
    return 0;
}

void *valloc(size_t n) { return memalign((size_t)sysconf(_SC_PAGESIZE), n); }

void *pvalloc(size_t n) {
    size_t pg = (size_t)sysconf(_SC_PAGESIZE);
    return memalign(pg, (n + pg - 1) & ~(pg - 1));
}
