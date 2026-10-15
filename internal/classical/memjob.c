#include "memjob.h"

int kpi_ossl_job_run(void *p) {
    kpi_ossl_job *j = p;
    j->out1len = j->out1cap;
    j->out2len = j->out2cap;
    switch (j->op) {
    case KPI_MEM_KEYGEN:
        if (j->kem) {
            return kpi_kem_keygen(j->alg, j->out1, &j->out1len, j->out2, &j->out2len) == 1 ? 1 : -1;
        }
        return kpi_sig_keygen(j->alg, j->out1, &j->out1len, j->out2, &j->out2len) == 1 ? 1 : -1;
    case KPI_MEM_SIGN:
        return kpi_sig_sign(j->alg, j->key, j->in1, j->in1len, j->out1, &j->out1len) == 1 ? 1 : -1;
    case KPI_MEM_VERIFY:
        return kpi_sig_verify(j->alg, j->key, j->in1, j->in1len, j->in2, j->in2len);
    case KPI_MEM_ENCAPS:
        return kpi_kem_encap(j->alg, j->key, j->pkm, j->pkmlen, j->out1, &j->out1len, j->out2) == 1 ? 1 : -1;
    case KPI_MEM_DECAPS:
        return kpi_kem_decap(j->alg, j->key, j->pkm, j->pkmlen, j->in1, j->in1len, j->out1) == 1 ? 1 : -1;
    }
    return -1;
}
