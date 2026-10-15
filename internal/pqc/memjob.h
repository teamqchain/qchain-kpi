/*
 * E9: one liboqs operation called through liboqs's C API, run as a job on
 * internal/memprobe's runner thread (its stack and C heap are measured). All
 * pointers are C memory set up before the call.
 */
#ifndef KPI_PQC_MEMJOB_H
#define KPI_PQC_MEMJOB_H

#include <stddef.h>
#include <stdint.h>
#include <oqs/oqs.h>

enum { KPI_MEM_KEYGEN = 1, KPI_MEM_SIGN, KPI_MEM_VERIFY, KPI_MEM_ENCAPS, KPI_MEM_DECAPS };

typedef struct {
    int op;
    OQS_SIG *sig;            /* signature operations */
    OQS_KEM *kem;            /* KEM operations */
    const uint8_t *key;      /* sign, decaps: secret key; verify, encaps: public key */
    const uint8_t *in1;      /* sign, verify: message; decaps: ciphertext */
    size_t in1len;
    const uint8_t *in2;      /* verify: signature */
    size_t in2len;
    uint8_t *out1;           /* keygen: public key; sign: signature; encaps: ciphertext; decaps: shared secret */
    size_t out1len;          /* sign: the signature's length */
    uint8_t *out2;           /* keygen: secret key; encaps: shared secret */
} kpi_pqc_job;

/* Returns 1 on success (verify: valid), 0 for an invalid signature, -1 on error. */
int kpi_pqc_job_run(void *job);

#endif
