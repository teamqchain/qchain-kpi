/*
 * E9: one classical operation (the C functions of ossl.h, with the key object
 * kept as in the reuse arm), run as a job on internal/memprobe's runner thread
 * (its stack and C heap are measured). All pointers are C memory set up
 * before the call.
 */
#ifndef KPI_OSSL_MEMJOB_H
#define KPI_OSSL_MEMJOB_H

#include <stddef.h>
#include "ossl.h"

enum { KPI_MEM_KEYGEN = 1, KPI_MEM_SIGN, KPI_MEM_VERIFY, KPI_MEM_ENCAPS, KPI_MEM_DECAPS };

typedef struct {
    int op;
    int kem;                    /* 1 for a DHKEM, 0 for a signature */
    int alg;                    /* KPI_ECDSA_P256 ... or KPI_DHKEM_P256 ... */
    EVP_PKEY *key;              /* sign, decaps: private; verify, encaps: public */
    const unsigned char *pkm;   /* encaps, decaps: the recipient's serialised public key */
    size_t pkmlen;
    const unsigned char *in1;   /* sign, verify: message; decaps: ciphertext (enc) */
    size_t in1len;
    const unsigned char *in2;   /* verify: signature */
    size_t in2len;
    unsigned char *out1;        /* keygen: public key; sign: signature; encaps: enc; decaps: shared secret */
    size_t out1len, out1cap;
    unsigned char *out2;        /* keygen: secret key; encaps: shared secret */
    size_t out2len, out2cap;
} kpi_ossl_job;

/* Returns 1 on success (verify: valid), 0 for an invalid signature, -1 on error. */
int kpi_ossl_job_run(void *job);

#endif
