#include "memjob.h"

int kpi_pqc_job_run(void *p) {
    kpi_pqc_job *j = p;
    switch (j->op) {
    case KPI_MEM_KEYGEN:
        if (j->sig != NULL) {
            return OQS_SIG_keypair(j->sig, j->out1, j->out2) == OQS_SUCCESS ? 1 : -1;
        }
        return OQS_KEM_keypair(j->kem, j->out1, j->out2) == OQS_SUCCESS ? 1 : -1;
    case KPI_MEM_SIGN:
        return OQS_SIG_sign(j->sig, j->out1, &j->out1len, j->in1, j->in1len, j->key) == OQS_SUCCESS ? 1 : -1;
    case KPI_MEM_VERIFY:
        /* liboqs reports an invalid signature as OQS_ERROR. */
        return OQS_SIG_verify(j->sig, j->in1, j->in1len, j->in2, j->in2len, j->key) == OQS_SUCCESS ? 1 : 0;
    case KPI_MEM_ENCAPS:
        return OQS_KEM_encaps(j->kem, j->out1, j->out2, j->key) == OQS_SUCCESS ? 1 : -1;
    case KPI_MEM_DECAPS:
        return OQS_KEM_decaps(j->kem, j->out1, j->in1, j->key) == OQS_SUCCESS ? 1 : -1;
    }
    return -1;
}
