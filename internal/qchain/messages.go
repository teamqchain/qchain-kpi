package qchain

// Harness code (not copied): generators for realistic QChain inputs, built
// with the verbatim production functions (commitmentHash, saltedFieldHash,
// sha3Hex) so their bytes are exactly what production would produce.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"time"
)

// SetSignatureAlgorithm selects the liboqs signature used by the copied
// pqcSign/pqcVerify (production hard-codes ML-DSA-44). One process measures
// one algorithm, so a package-level setting is sufficient.
func SetSignatureAlgorithm(name string) { sigName = name }

const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// RandomValue returns an ASCII alphanumeric attribute value of length n.
// Values are restricted to this alphabet so Go and Dart JSON-encode them
// identically.
func RandomValue(rng *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alnum[rng.IntN(len(alnum))]
	}
	return string(b)
}

// RandomSalt returns 32 lowercase hex characters (16 random bytes), the salt
// format production uses (commitment.go newFieldSalt). Pools use a seeded
// PRNG; the timed lifecycle code uses the verbatim crypto/rand version.
func RandomSalt(rng *rand.Rand) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rng.Uint32())
	}
	return hex.EncodeToString(b)
}

// FieldName returns the i-th synthetic attribute name ("field01", ...).
func FieldName(i int) string { return fmt.Sprintf("field%02d", i) }

const b58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// FakeCIDv0 returns a well-formed IPFS CIDv0 ("Qm..." , 46 characters): the
// base58btc encoding of the sha2-256 multihash of data. IPFS is outside the
// measured path, so this stands in for the CID the daemon would return.
func FakeCIDv0(data []byte) string {
	sum := sha256.Sum256(data)
	mh := append([]byte{0x12, 0x20}, sum[:]...)
	// Base58 encoding (no leading zero bytes in a multihash starting 0x12).
	digits := []byte{0}
	for _, c := range mh {
		carry := int(c)
		for j := range digits {
			carry += int(digits[j]) << 8
			digits[j] = byte(carry % 58)
			carry /= 58
		}
		for carry > 0 {
			digits = append(digits, byte(carry%58))
			carry /= 58
		}
	}
	out := make([]byte, len(digits))
	for i, d := range digits {
		out[len(digits)-1-i] = b58[d]
	}
	return string(out)
}

// Commitment builds a realistic signed commitment with n attributes and
// returns it with the attribute values and salts.
func Commitment(rng *rand.Rand, n int, issuedAt time.Time) (CredentialCommitment, map[string]string, map[string]string) {
	attrs := make(map[string]string, n)
	salts := make(map[string]string, n)
	hashes := make(map[string]string, n)
	for i := 1; i <= n; i++ {
		k := FieldName(i)
		attrs[k] = RandomValue(rng, 16)
		salts[k] = RandomSalt(rng)
		hashes[k] = saltedFieldHash(salts[k], k, attrs[k])
	}
	cidSeed := make([]byte, 32)
	for i := range cidSeed {
		cidSeed[i] = byte(rng.Uint32())
	}
	c := CredentialCommitment{
		HolderID:       "H-0001",
		CredentialType: "Bachelor of Science in Computer Science",
		IssuedAt:       issuedAt.In(mustLoadLocation("Asia/Dubai")).Truncate(time.Second).Format(time.RFC3339),
		IssuerOrgID:    "GeneralMSP",
		ExpiryDate:     "2030-06-30",
		CID:            FakeCIDv0(cidSeed),
		FieldHashes:    hashes,
	}
	return c, attrs, salts
}

// IssuerMessage is what the issuer signs: commitmentHash(c), a 64-character
// lowercase hex string (commitment.go signCommitment).
func IssuerMessage(rng *rand.Rand, n int) (string, error) {
	t := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC).Add(time.Duration(rng.Int64N(int64(90 * 24 * time.Hour))))
	c, _, _ := Commitment(rng, n, t)
	return commitmentHash(c)
}

// HolderMessage is what the holder signs: sha3Hex of the canonical payload
// disclosing k of n attributes (crypto_service.dart:656-664).
func HolderMessage(rng *rand.Rand, n, k int) (string, error) {
	t := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC).Add(time.Duration(rng.Int64N(int64(90 * 24 * time.Hour))))
	_, attrs, salts := Commitment(rng, n, t)
	disclosed := make(map[string]string, k)
	ds := make(map[string]string, k)
	for i := 1; i <= k; i++ {
		f := FieldName(i)
		disclosed[f] = attrs[f]
		ds[f] = salts[f]
	}
	ts := t.Add(time.Duration(rng.Int64N(int64(30*24*time.Hour))) + time.Duration(rng.Int64N(1e6))*time.Microsecond)
	raw, err := BuildDisclosedPayload(fmt.Sprintf("CRED-%04d", 1+rng.IntN(9999)), disclosed, ds, WalletTimestamp(ts))
	if err != nil {
		return "", err
	}
	return PayloadHash(raw), nil
}
