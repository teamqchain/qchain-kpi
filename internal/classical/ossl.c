/* See ossl.h. OpenSSL 3.0 API only. */
#include "ossl.h"

#include <string.h>
#include <openssl/core_names.h>
#include <openssl/crypto.h>
#include <openssl/err.h>
#include <openssl/kdf.h>
#include <openssl/param_build.h>
#include <openssl/rsa.h>
#include <openssl/x509.h>

static EVP_MD *g_sha256 = NULL;
static EVP_KDF *g_hkdf = NULL;

int kpi_ossl_init(void) {
    if (g_sha256 == NULL) g_sha256 = EVP_MD_fetch(NULL, "SHA256", NULL);
    if (g_hkdf == NULL) g_hkdf = EVP_KDF_fetch(NULL, "HKDF", NULL);
    if (g_sha256 == NULL || g_hkdf == NULL) {
        ERR_clear_error();
        return 0;
    }
    return 1;
}

const char *kpi_ossl_version(void) { return OpenSSL_version(OPENSSL_VERSION); }

void kpi_key_free(EVP_PKEY *key) { EVP_PKEY_free(key); }

static int fail(void) {
    ERR_clear_error();
    return 0;
}

/* ------------------------------------------------------------------ EC helpers */

/* Exports a P-256 key as (32-byte scalar, 65-byte SEC1 uncompressed point). */
static int ec_export(EVP_PKEY *k, unsigned char *pk, size_t *pklen, unsigned char *sk, size_t *sklen) {
    BIGNUM *d = NULL;
    size_t cap = *pklen, n = 0;
    if (EVP_PKEY_get_octet_string_param(k, OSSL_PKEY_PARAM_ENCODED_PUBLIC_KEY, pk, cap, &n) != 1 || n != 65)
        return 0;
    *pklen = n;
    if (sk != NULL) {
        if (*sklen < 32 || EVP_PKEY_get_bn_param(k, OSSL_PKEY_PARAM_PRIV_KEY, &d) != 1) return 0;
        if (BN_bn2binpad(d, sk, 32) != 32) { BN_clear_free(d); return 0; }
        BN_clear_free(d);
        *sklen = 32;
    }
    return 1;
}

/* Builds a P-256 key from a scalar (optional) and a SEC1 point. Importing the
 * point checks that it lies on the curve. */
static EVP_PKEY *ec_import(const unsigned char *sk, size_t sklen, const unsigned char *pk, size_t pklen) {
    OSSL_PARAM_BLD *bld = NULL;
    OSSL_PARAM *params = NULL;
    EVP_PKEY_CTX *ctx = NULL;
    EVP_PKEY *key = NULL;
    BIGNUM *d = NULL;
    int selection = EVP_PKEY_PUBLIC_KEY;

    if (pk == NULL || pklen != 65) goto done;
    bld = OSSL_PARAM_BLD_new();
    if (bld == NULL) goto done;
    if (OSSL_PARAM_BLD_push_utf8_string(bld, OSSL_PKEY_PARAM_GROUP_NAME, "prime256v1", 0) != 1) goto done;
    if (OSSL_PARAM_BLD_push_octet_string(bld, OSSL_PKEY_PARAM_PUB_KEY, pk, pklen) != 1) goto done;
    if (sk != NULL) {
        if (sklen != 32) goto done;
        d = BN_bin2bn(sk, (int)sklen, NULL);
        if (d == NULL || OSSL_PARAM_BLD_push_BN(bld, OSSL_PKEY_PARAM_PRIV_KEY, d) != 1) goto done;
        selection = EVP_PKEY_KEYPAIR;
    }
    params = OSSL_PARAM_BLD_to_param(bld);
    if (params == NULL) goto done;
    ctx = EVP_PKEY_CTX_new_from_name(NULL, "EC", NULL);
    if (ctx == NULL || EVP_PKEY_fromdata_init(ctx) != 1) goto done;
    if (EVP_PKEY_fromdata(ctx, &key, selection, params) != 1) key = NULL;
done:
    EVP_PKEY_CTX_free(ctx);
    OSSL_PARAM_free(params);
    OSSL_PARAM_BLD_free(bld);
    BN_clear_free(d);
    if (key == NULL) ERR_clear_error();
    return key;
}

/* ------------------------------------------------------------------ signatures */

int kpi_sig_keygen(int alg, unsigned char *pk, size_t *pklen, unsigned char *sk, size_t *sklen) {
    EVP_PKEY *k = NULL;
    int ok = 0;
    switch (alg) {
    case KPI_ECDSA_P256:
        k = EVP_PKEY_Q_keygen(NULL, NULL, "EC", "P-256");
        if (k == NULL) break;
        ok = ec_export(k, pk, pklen, sk, sklen);
        break;
    case KPI_ED25519: {
        size_t a = *pklen, b = *sklen;
        k = EVP_PKEY_Q_keygen(NULL, NULL, "ED25519");
        if (k == NULL) break;
        if (EVP_PKEY_get_raw_public_key(k, pk, &a) != 1 || EVP_PKEY_get_raw_private_key(k, sk, &b) != 1) break;
        *pklen = a;
        *sklen = b;
        ok = 1;
        break;
    }
    case KPI_RSA3072_PSS: {
        PKCS8_PRIV_KEY_INFO *p8 = NULL;
        int n;
        unsigned char *p;
        k = EVP_PKEY_Q_keygen(NULL, NULL, "RSA", (size_t)3072);
        if (k == NULL) break;
        n = i2d_PUBKEY(k, NULL);
        if (n <= 0 || (size_t)n > *pklen) break;
        p = pk;
        if (i2d_PUBKEY(k, &p) != n) break;
        *pklen = (size_t)n;
        p8 = EVP_PKEY2PKCS8(k);
        if (p8 == NULL) break;
        n = i2d_PKCS8_PRIV_KEY_INFO(p8, NULL);
        if (n <= 0 || (size_t)n > *sklen) { PKCS8_PRIV_KEY_INFO_free(p8); break; }
        p = sk;
        if (i2d_PKCS8_PRIV_KEY_INFO(p8, &p) != n) { PKCS8_PRIV_KEY_INFO_free(p8); break; }
        PKCS8_PRIV_KEY_INFO_free(p8);
        *sklen = (size_t)n;
        ok = 1;
        break;
    }
    default:
        break;
    }
    EVP_PKEY_free(k);
    return ok ? 1 : fail();
}

EVP_PKEY *kpi_sig_load_private(int alg, const unsigned char *sk, size_t sklen,
                               const unsigned char *pk, size_t pklen) {
    EVP_PKEY *k = NULL;
    switch (alg) {
    case KPI_ECDSA_P256:
        return ec_import(sk, sklen, pk, pklen);
    case KPI_ED25519:
        k = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, NULL, sk, sklen);
        break;
    case KPI_RSA3072_PSS: {
        const unsigned char *p = sk;
        k = d2i_AutoPrivateKey(NULL, &p, (long)sklen);
        break;
    }
    default:
        break;
    }
    if (k == NULL) ERR_clear_error();
    return k;
}

EVP_PKEY *kpi_sig_load_public(int alg, const unsigned char *pk, size_t pklen) {
    EVP_PKEY *k = NULL;
    switch (alg) {
    case KPI_ECDSA_P256:
        return ec_import(NULL, 0, pk, pklen);
    case KPI_ED25519:
        k = EVP_PKEY_new_raw_public_key(EVP_PKEY_ED25519, NULL, pk, pklen);
        break;
    case KPI_RSA3072_PSS: {
        const unsigned char *p = pk;
        k = d2i_PUBKEY(NULL, &p, (long)pklen);
        break;
    }
    default:
        break;
    }
    if (k == NULL) ERR_clear_error();
    return k;
}

static int pss_params(int alg, EVP_PKEY_CTX *pctx) {
    if (alg != KPI_RSA3072_PSS) return 1;
    return EVP_PKEY_CTX_set_rsa_padding(pctx, RSA_PKCS1_PSS_PADDING) == 1 &&
           EVP_PKEY_CTX_set_rsa_pss_saltlen(pctx, RSA_PSS_SALTLEN_DIGEST) == 1;
}

int kpi_sig_sign(int alg, EVP_PKEY *key, const unsigned char *msg, size_t msglen,
                 unsigned char *sig, size_t *siglen) {
    EVP_MD_CTX *mctx = EVP_MD_CTX_new();
    EVP_PKEY_CTX *pctx = NULL;
    const EVP_MD *md = (alg == KPI_ED25519) ? NULL : g_sha256;
    int ok = 0;
    if (mctx == NULL || key == NULL) goto done;
    if (EVP_DigestSignInit(mctx, &pctx, md, NULL, key) != 1) goto done;
    if (!pss_params(alg, pctx)) goto done;
    if (EVP_DigestSign(mctx, sig, siglen, msg, msglen) != 1) goto done;
    ok = 1;
done:
    EVP_MD_CTX_free(mctx);
    return ok ? 1 : fail();
}

int kpi_sig_verify(int alg, EVP_PKEY *key, const unsigned char *msg, size_t msglen,
                   const unsigned char *sig, size_t siglen) {
    EVP_MD_CTX *mctx = EVP_MD_CTX_new();
    EVP_PKEY_CTX *pctx = NULL;
    const EVP_MD *md = (alg == KPI_ED25519) ? NULL : g_sha256;
    int r = -1;
    if (mctx == NULL || key == NULL) goto done;
    if (EVP_DigestVerifyInit(mctx, &pctx, md, NULL, key) != 1) goto done;
    if (!pss_params(alg, pctx)) goto done;
    r = EVP_DigestVerify(mctx, sig, siglen, msg, msglen);
    if (r != 1) r = (r == 0) ? 0 : -1;
done:
    EVP_MD_CTX_free(mctx);
    if (r != 1) ERR_clear_error();
    return r;
}

/* ------------------------------------------------------------- RFC 9180 DHKEM */

#define NSECRET 32
#define LBUF 512

static size_t suite_id(int alg, unsigned char out[5]) {
    unsigned int kem_id = (alg == KPI_DHKEM_P256) ? 0x0010u : 0x0020u;
    out[0] = 'K'; out[1] = 'E'; out[2] = 'M';
    out[3] = (unsigned char)(kem_id >> 8);
    out[4] = (unsigned char)(kem_id & 0xff);
    return 5;
}

static int hkdf(int mode, const unsigned char *key, size_t keylen,
                const unsigned char *info, size_t infolen, unsigned char *out, size_t outlen) {
    EVP_KDF_CTX *kctx = EVP_KDF_CTX_new(g_hkdf);
    OSSL_PARAM params[5], *p = params;
    int ok;
    if (kctx == NULL) return 0;
    *p++ = OSSL_PARAM_construct_utf8_string(OSSL_KDF_PARAM_DIGEST, (char *)"SHA256", 0);
    *p++ = OSSL_PARAM_construct_int(OSSL_KDF_PARAM_MODE, &mode);
    *p++ = OSSL_PARAM_construct_octet_string(OSSL_KDF_PARAM_KEY, (void *)key, keylen);
    if (mode == EVP_KDF_HKDF_MODE_EXPAND_ONLY)
        *p++ = OSSL_PARAM_construct_octet_string(OSSL_KDF_PARAM_INFO, (void *)info, infolen);
    *p = OSSL_PARAM_construct_end();
    ok = EVP_KDF_derive(kctx, out, outlen, params) == 1;
    EVP_KDF_CTX_free(kctx);
    return ok;
}

/* LabeledExtract("", label, ikm) = HKDF-Extract(salt = "", "HPKE-v1" || suite_id || label || ikm).
 * An absent salt is a zero-length HMAC key, identical to RFC 5869's string of
 * HashLen zeros. */
static int labeled_extract(int alg, const char *label, const unsigned char *ikm, size_t ikmlen,
                           unsigned char prk[NSECRET]) {
    unsigned char buf[LBUF], sid[5];
    size_t n = 0, ll = strlen(label), sl = suite_id(alg, sid);
    if (7 + sl + ll + ikmlen > sizeof buf) return 0;
    memcpy(buf + n, "HPKE-v1", 7); n += 7;
    memcpy(buf + n, sid, sl); n += sl;
    memcpy(buf + n, label, ll); n += ll;
    if (ikmlen) { memcpy(buf + n, ikm, ikmlen); n += ikmlen; }
    return hkdf(EVP_KDF_HKDF_MODE_EXTRACT_ONLY, buf, n, NULL, 0, prk, NSECRET);
}

/* LabeledExpand(prk, label, info, L) = HKDF-Expand(prk, I2OSP(L, 2) || "HPKE-v1" || suite_id || label || info, L). */
static int labeled_expand(int alg, const unsigned char prk[NSECRET], const char *label,
                          const unsigned char *info, size_t infolen, unsigned char *out, size_t L) {
    unsigned char buf[LBUF], sid[5];
    size_t n = 0, ll = strlen(label), sl = suite_id(alg, sid);
    if (2 + 7 + sl + ll + infolen > sizeof buf || L > 0xffff) return 0;
    buf[n++] = (unsigned char)(L >> 8);
    buf[n++] = (unsigned char)(L & 0xff);
    memcpy(buf + n, "HPKE-v1", 7); n += 7;
    memcpy(buf + n, sid, sl); n += sl;
    memcpy(buf + n, label, ll); n += ll;
    if (infolen) { memcpy(buf + n, info, infolen); n += infolen; }
    return hkdf(EVP_KDF_HKDF_MODE_EXPAND_ONLY, prk, NSECRET, buf, n, out, L);
}

/* ExtractAndExpand(dh, kem_context) */
static int extract_and_expand(int alg, const unsigned char *dh, size_t dhlen,
                              const unsigned char *ctx, size_t ctxlen, unsigned char ss[NSECRET]) {
    unsigned char prk[NSECRET];
    int ok = labeled_extract(alg, "eae_prk", dh, dhlen, prk) &&
             labeled_expand(alg, prk, "shared_secret", ctx, ctxlen, ss, NSECRET);
    OPENSSL_cleanse(prk, sizeof prk);
    return ok;
}

/* DH(sk, pk). Setting the peer validates its public key. RFC 9180 7.1.4: an
 * all-zero X25519 result must be rejected. */
static int dh(int alg, EVP_PKEY *sk, EVP_PKEY *pk, unsigned char *out, size_t *outlen) {
    EVP_PKEY_CTX *c = EVP_PKEY_CTX_new(sk, NULL);
    size_t i;
    unsigned char acc = 0;
    int ok = 0;
    if (c == NULL) return 0;
    if (EVP_PKEY_derive_init(c) != 1 || EVP_PKEY_derive_set_peer(c, pk) != 1) goto done;
    if (EVP_PKEY_derive(c, out, outlen) != 1 || *outlen != 32) goto done;
    if (alg == KPI_DHKEM_X25519) {
        for (i = 0; i < *outlen; i++) acc |= out[i];
        if (acc == 0) goto done;
    }
    ok = 1;
done:
    EVP_PKEY_CTX_free(c);
    return ok;
}

static int serialize_public(int alg, EVP_PKEY *k, unsigned char *out, size_t *outlen) {
    if (alg == KPI_DHKEM_P256) return ec_export(k, out, outlen, NULL, NULL);
    return EVP_PKEY_get_raw_public_key(k, out, outlen) == 1 && *outlen == 32;
}

int kpi_kem_keygen(int alg, unsigned char *pk, size_t *pklen, unsigned char *sk, size_t *sklen) {
    EVP_PKEY *k = NULL;
    int ok = 0;
    if (alg == KPI_DHKEM_P256) {
        k = EVP_PKEY_Q_keygen(NULL, NULL, "EC", "P-256");
        ok = k != NULL && ec_export(k, pk, pklen, sk, sklen);
    } else if (alg == KPI_DHKEM_X25519) {
        size_t a = *pklen, b = *sklen;
        k = EVP_PKEY_Q_keygen(NULL, NULL, "X25519");
        ok = k != NULL && EVP_PKEY_get_raw_public_key(k, pk, &a) == 1 && EVP_PKEY_get_raw_private_key(k, sk, &b) == 1;
        if (ok) { *pklen = a; *sklen = b; }
    }
    EVP_PKEY_free(k);
    return ok ? 1 : fail();
}

EVP_PKEY *kpi_kem_load_private(int alg, const unsigned char *sk, size_t sklen,
                               const unsigned char *pk, size_t pklen) {
    EVP_PKEY *k = NULL;
    if (alg == KPI_DHKEM_P256) return ec_import(sk, sklen, pk, pklen);
    if (alg == KPI_DHKEM_X25519) k = EVP_PKEY_new_raw_private_key(EVP_PKEY_X25519, NULL, sk, sklen);
    if (k == NULL) ERR_clear_error();
    return k;
}

EVP_PKEY *kpi_kem_load_public(int alg, const unsigned char *pk, size_t pklen) {
    EVP_PKEY *k = NULL;
    if (alg == KPI_DHKEM_P256) return ec_import(NULL, 0, pk, pklen);
    if (alg == KPI_DHKEM_X25519 && pklen == 32) k = EVP_PKEY_new_raw_public_key(EVP_PKEY_X25519, NULL, pk, pklen);
    if (k == NULL) ERR_clear_error();
    return k;
}

static int encap_with(int alg, EVP_PKEY *skE, EVP_PKEY *pkR, const unsigned char *pkRm, size_t pkRmlen,
                      unsigned char *enc, size_t *enclen, unsigned char *ss) {
    unsigned char dhv[32], ctx[LBUF];
    size_t dhlen = sizeof dhv, n;
    int ok = 0;
    if (!dh(alg, skE, pkR, dhv, &dhlen)) goto done;
    if (!serialize_public(alg, skE, enc, enclen)) goto done;
    if (*enclen + pkRmlen > sizeof ctx) goto done;
    memcpy(ctx, enc, *enclen);
    memcpy(ctx + *enclen, pkRm, pkRmlen);
    n = *enclen + pkRmlen;
    ok = extract_and_expand(alg, dhv, dhlen, ctx, n, ss);
done:
    OPENSSL_cleanse(dhv, sizeof dhv);
    return ok;
}

int kpi_kem_encap(int alg, EVP_PKEY *pkR, const unsigned char *pkRm, size_t pkRmlen,
                  unsigned char *enc, size_t *enclen, unsigned char *ss) {
    EVP_PKEY *skE = NULL;
    int ok;
    if (alg == KPI_DHKEM_P256) skE = EVP_PKEY_Q_keygen(NULL, NULL, "EC", "P-256");
    else if (alg == KPI_DHKEM_X25519) skE = EVP_PKEY_Q_keygen(NULL, NULL, "X25519");
    if (skE == NULL || pkR == NULL) { EVP_PKEY_free(skE); return fail(); }
    ok = encap_with(alg, skE, pkR, pkRm, pkRmlen, enc, enclen, ss);
    EVP_PKEY_free(skE);
    return ok ? 1 : fail();
}

int kpi_kem_encap_with_ephemeral(int alg, EVP_PKEY *pkR, const unsigned char *pkRm, size_t pkRmlen,
                                 const unsigned char *skEm, size_t skEmlen,
                                 const unsigned char *pkEm, size_t pkEmlen,
                                 unsigned char *enc, size_t *enclen, unsigned char *ss) {
    EVP_PKEY *skE = kpi_kem_load_private(alg, skEm, skEmlen, pkEm, pkEmlen);
    int ok;
    if (skE == NULL || pkR == NULL) { EVP_PKEY_free(skE); return fail(); }
    ok = encap_with(alg, skE, pkR, pkRm, pkRmlen, enc, enclen, ss);
    EVP_PKEY_free(skE);
    return ok ? 1 : fail();
}

int kpi_kem_decap(int alg, EVP_PKEY *skR, const unsigned char *pkRm, size_t pkRmlen,
                  const unsigned char *enc, size_t enclen, unsigned char *ss) {
    EVP_PKEY *pkE = NULL;
    unsigned char dhv[32], ctx[LBUF];
    size_t dhlen = sizeof dhv;
    int ok = 0;
    if (skR == NULL || enclen + pkRmlen > sizeof ctx) return fail();
    pkE = kpi_kem_load_public(alg, enc, enclen);
    if (pkE == NULL) return fail();
    if (!dh(alg, skR, pkE, dhv, &dhlen)) goto done;
    memcpy(ctx, enc, enclen);
    memcpy(ctx + enclen, pkRm, pkRmlen);
    ok = extract_and_expand(alg, dhv, dhlen, ctx, enclen + pkRmlen, ss);
done:
    OPENSSL_cleanse(dhv, sizeof dhv);
    EVP_PKEY_free(pkE);
    return ok ? 1 : fail();
}
