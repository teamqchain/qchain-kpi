// Package algs is the registry of the 30 algorithms in the study:
// 25 post-quantum algorithms from liboqs 0.16.0 and 5 classical baselines.
//
// The liboqs names are the exact OQS_SIG_alg_* / OQS_KEM_alg_* strings at tag
// 0.16.0 (src/sig/sig.h, src/kem/kem.h), verified on 2026-10-10.
package algs

import "fmt"

// Track is the family of operations an algorithm supports.
type Track string

const (
	Sig Track = "sig" // keygen, sign, verify
	KEM Track = "kem" // keygen, encaps, decaps
)

// Alg describes one algorithm of the study.
type Alg struct {
	ID       string // stable identifier used in plans and file names
	Track    Track
	Family   string // ML-DSA, SLH-DSA, Falcon, ML-KEM, HQC, ECDSA, EdDSA, RSA-PSS, DHKEM
	Tier     string // "A" final FIPS standard, "B" NIST-selected, "C" classical baseline
	Category int    // NIST security category (1-5); 0 for classical baselines
	OQSName  string // exact liboqs mechanism name; "" for classical baselines
	Spec     string // standard or specification the implementation follows
}

// Classical reports whether the algorithm is one of the classical baselines.
func (a Alg) Classical() bool { return a.OQSName == "" }

// Ops returns the operations measured for the algorithm, in a fixed order.
func (a Alg) Ops() []string {
	if a.Track == Sig {
		return []string{"keygen", "sign", "verify"}
	}
	return []string{"keygen", "encaps", "decaps"}
}

// SizeOps returns the operations E8 samples output sizes from: key
// generation (public and secret key) and sign (signature) or encaps
// (ciphertext and shared secret). Verify and decaps produce no new object.
func (a Alg) SizeOps() []string {
	if a.Track == Sig {
		return []string{"keygen", "sign"}
	}
	return []string{"keygen", "encaps"}
}

// HasOp reports whether op is one of the algorithm's operations.
func (a Alg) HasOp(op string) bool {
	for _, o := range a.Ops() {
		if o == op {
			return true
		}
	}
	return false
}

// PoolSize returns the approved input-pool size P (coding plan §6): 256, or
// 64 for the SLH-DSA "s" (small-signature) sets, whose signing takes up to
// seconds per signature, so a 256-entry pool alone could take many minutes
// to generate.
func (a Alg) PoolSize() int {
	if a.Family == "SLH-DSA" && a.ID[len(a.ID)-1] == 's' {
		return 64
	}
	return 256
}

func slh(hash string, level int, variant string, category int) Alg {
	return Alg{
		ID:       fmt.Sprintf("slh-dsa-%s-%d%s", hash, level, variant),
		Track:    Sig,
		Family:   "SLH-DSA",
		Tier:     "A",
		Category: category,
		OQSName:  fmt.Sprintf("SLH_DSA_PURE_%s_%d%s", upper(hash), level, upper(variant)),
		Spec:     "FIPS 205 (pure)",
	}
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}

var registry = buildRegistry()

func buildRegistry() []Alg {
	r := []Alg{
		{"ml-dsa-44", Sig, "ML-DSA", "A", 2, "ML-DSA-44", "FIPS 204"},
		{"ml-dsa-65", Sig, "ML-DSA", "A", 3, "ML-DSA-65", "FIPS 204"},
		{"ml-dsa-87", Sig, "ML-DSA", "A", 5, "ML-DSA-87", "FIPS 204"},
	}
	for _, hash := range []string{"sha2", "shake"} {
		for _, lv := range []struct {
			level, cat int
		}{{128, 1}, {192, 3}, {256, 5}} {
			for _, v := range []string{"s", "f"} {
				r = append(r, slh(hash, lv.level, v, lv.cat))
			}
		}
	}
	falconSpec := "Falcon round-3 spec 2021-11-01 (NIST-selected; FN-DSA not final)"
	r = append(r,
		Alg{"falcon-512", Sig, "Falcon", "B", 1, "Falcon-512", falconSpec},
		Alg{"falcon-1024", Sig, "Falcon", "B", 5, "Falcon-1024", falconSpec},
		Alg{"falcon-padded-512", Sig, "Falcon", "B", 1, "Falcon-padded-512", falconSpec},
		Alg{"falcon-padded-1024", Sig, "Falcon", "B", 5, "Falcon-padded-1024", falconSpec},
		Alg{"ecdsa-p256", Sig, "ECDSA", "C", 0, "", "FIPS 186-5 ECDSA over P-256 with SHA-256 (DER signatures)"},
		Alg{"ed25519", Sig, "EdDSA", "C", 0, "", "RFC 8032 Ed25519"},
		Alg{"rsa3072-pss", Sig, "RSA-PSS", "C", 0, "", "RFC 8017 RSASSA-PSS, 3072-bit, SHA-256, salt 32"},
		Alg{"ml-kem-512", KEM, "ML-KEM", "A", 1, "ML-KEM-512", "FIPS 203"},
		Alg{"ml-kem-768", KEM, "ML-KEM", "A", 3, "ML-KEM-768", "FIPS 203"},
		Alg{"ml-kem-1024", KEM, "ML-KEM", "A", 5, "ML-KEM-1024", "FIPS 203"},
		Alg{"hqc-1", KEM, "HQC", "B", 1, "HQC-1", "HQC spec 2025-08-22 (NIST-selected; not final)"},
		Alg{"hqc-3", KEM, "HQC", "B", 3, "HQC-3", "HQC spec 2025-08-22 (NIST-selected; not final)"},
		Alg{"hqc-5", KEM, "HQC", "B", 5, "HQC-5", "HQC spec 2025-08-22 (NIST-selected; not final)"},
		Alg{"dhkem-p256", KEM, "DHKEM", "C", 0, "", "RFC 9180 DHKEM(P-256, HKDF-SHA256)"},
		Alg{"dhkem-x25519", KEM, "DHKEM", "C", 0, "", "RFC 9180 DHKEM(X25519, HKDF-SHA256)"},
	)
	return r
}

// All returns every algorithm in a fixed order (a copy).
func All() []Alg {
	out := make([]Alg, len(registry))
	copy(out, registry)
	return out
}

// ByID looks an algorithm up by its identifier.
func ByID(id string) (Alg, bool) {
	for _, a := range registry {
		if a.ID == id {
			return a, true
		}
	}
	return Alg{}, false
}

// MustByID is ByID for identifiers known at compile time.
func MustByID(id string) Alg {
	a, ok := ByID(id)
	if !ok {
		panic("algs: unknown algorithm " + id)
	}
	return a
}

// OQSNames returns the liboqs names of the post-quantum algorithms.
func OQSNames(track Track) []string {
	var out []string
	for _, a := range registry {
		if a.Track == track && !a.Classical() {
			out = append(out, a.OQSName)
		}
	}
	return out
}

// Expand resolves a plan's algorithm list: "all", "pqc", "classical", "sig",
// "kem", a family name such as "SLH-DSA", or explicit identifiers.
func Expand(spec []string) ([]Alg, error) {
	seen := map[string]bool{}
	var out []Alg
	add := func(a Alg) {
		if !seen[a.ID] {
			seen[a.ID] = true
			out = append(out, a)
		}
	}
	for _, s := range spec {
		matched := false
		for _, a := range registry {
			if s == "all" ||
				(s == "pqc" && !a.Classical()) ||
				(s == "classical" && a.Classical()) ||
				(s == "sig" && a.Track == Sig) ||
				(s == "kem" && a.Track == KEM) ||
				s == a.Family || s == a.ID {
				add(a)
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("unknown algorithm selector %q", s)
		}
	}
	return out, nil
}
