/*
 * Classical baselines over OpenSSL 3.0 (Debian bookworm), called from Go via
 * cgo with ONE C call per measured operation, the same Go-to-C boundary pattern
 * liboqs-go uses for the post-quantum algorithms.
 *
 * Signatures: ECDSA P-256 with SHA-256 (DER signatures), Ed25519 (RFC 8032),
 *             RSASSA-PSS 3072-bit with SHA-256 and a 32-byte salt (RFC 8017).
 * KEMs:       DHKEM(P-256, HKDF-SHA256) and DHKEM(X25519, HKDF-SHA256) exactly
 *             as specified in RFC 9180 section 4.1 (OpenSSL 3.0 has no HPKE API).
 *
 * Key formats (also used for the size KPIs):
 *   ECDSA / DHKEM P-256: secret = 32-byte scalar, public = SEC1 uncompressed (65 B)
 *   Ed25519 / X25519:    secret = 32-byte raw,    public = 32-byte raw
 *   RSA-3072:            secret = PKCS#8 DER,     public = SubjectPublicKeyInfo DER
 *
 * All functions return 1 on success and 0 on failure (and clear OpenSSL's
 * error queue on failure). kpi_sig_verify returns 1 valid, 0 invalid, -1 error.
 */
#ifndef KPI_OSSL_H
#define KPI_OSSL_H

#include <stddef.h>
#include <openssl/evp.h>

enum { KPI_ECDSA_P256 = 1, KPI_ED25519 = 2, KPI_RSA3072_PSS = 3 };
enum { KPI_DHKEM_P256 = 1, KPI_DHKEM_X25519 = 2 };

int kpi_ossl_init(void);
const char *kpi_ossl_version(void);

/* Signatures */
int kpi_sig_keygen(int alg, unsigned char *pk, size_t *pklen, unsigned char *sk, size_t *sklen);
EVP_PKEY *kpi_sig_load_private(int alg, const unsigned char *sk, size_t sklen,
                               const unsigned char *pk, size_t pklen);
EVP_PKEY *kpi_sig_load_public(int alg, const unsigned char *pk, size_t pklen);
int kpi_sig_sign(int alg, EVP_PKEY *key, const unsigned char *msg, size_t msglen,
                 unsigned char *sig, size_t *siglen);
int kpi_sig_verify(int alg, EVP_PKEY *key, const unsigned char *msg, size_t msglen,
                   const unsigned char *sig, size_t siglen);

/* RFC 9180 DHKEM */
int kpi_kem_keygen(int alg, unsigned char *pk, size_t *pklen, unsigned char *sk, size_t *sklen);
EVP_PKEY *kpi_kem_load_private(int alg, const unsigned char *sk, size_t sklen,
                               const unsigned char *pk, size_t pklen);
EVP_PKEY *kpi_kem_load_public(int alg, const unsigned char *pk, size_t pklen);
int kpi_kem_encap(int alg, EVP_PKEY *pkR, const unsigned char *pkRm, size_t pkRmlen,
                  unsigned char *enc, size_t *enclen, unsigned char *ss);
/* Test-only: encapsulation with a given ephemeral key (RFC 9180 test vectors). */
int kpi_kem_encap_with_ephemeral(int alg, EVP_PKEY *pkR, const unsigned char *pkRm, size_t pkRmlen,
                                 const unsigned char *skEm, size_t skEmlen,
                                 const unsigned char *pkEm, size_t pkEmlen,
                                 unsigned char *enc, size_t *enclen, unsigned char *ss);
int kpi_kem_decap(int alg, EVP_PKEY *skR, const unsigned char *pkRm, size_t pkRmlen,
                  const unsigned char *enc, size_t enclen, unsigned char *ss);

void kpi_key_free(EVP_PKEY *key);

#endif
