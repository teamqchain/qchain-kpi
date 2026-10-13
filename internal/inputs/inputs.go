// Package inputs prepares, saves and loads the untimed inputs of a block:
// keys and fixed pools of P messages/signatures/ciphertexts (coding plan §6).
//
// They are generated once per (algorithm, repetition) by a separate "prep"
// process, so that (a) no measured process has run any crypto operation
// before its cold call, (b) slow signers are not re-run for every block, and
// (c) every repetition uses its own keys and messages, so the spread between
// repetitions includes key-to-key variation. Pools are cycled during sampling
// (call i uses entry i mod P). Keys come from the libraries' real randomness;
// messages come from a seeded generator.
package inputs

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"qchain-kpi/internal/record"
	"qchain-kpi/internal/scheme"
)

// Prep is the saved input set of one (algorithm, repetition).
type Prep struct {
	Alg         string
	Rep         int
	Seed        uint64
	P           int
	Sizes       scheme.Sizes
	PK, SK      []byte
	Fingerprint string // first 16 hex digits of SHA-256(PK): identifies the key in raw headers

	// Signatures: messages are fresh 64-character lowercase hex strings (what
	// QChain signs); Sigs are valid signatures over them (verify inputs).
	Msgs [][]byte
	Sigs [][]byte

	// KEMs: ciphertexts and the shared secrets they decapsulate to.
	CTs [][]byte
	SSs [][]byte

	// E10 (ML-DSA only): a separate holder key and three message pools:
	// issuer = commitment hashes (N = 10 attributes), holder = presentation
	// payload hashes (N = 10, k = 5 disclosed), issuer-again = a second,
	// independent pool of commitment hashes (the A/A arm).
	HolderPK, HolderSK                      []byte
	HolderFingerprint                       string
	IssuerMsgs, HolderMsgs, IssuerAgainMsgs [][]byte

	// E6 (signature schemes): for each message size, a pool of messages
	// generated from a seed (not stored: 1 MiB messages would make every prep
	// file hundreds of megabytes) and valid signatures over them (stored).
	E6 []E6Pool

	PrepNS int64 // time taken to generate (feeds the runtime estimate)
}

// Fingerprint returns the first 16 hex digits of SHA-256(b).
func Fingerprint(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:8])
}

// HexMessage returns a fresh 64-character lowercase hex message.
func HexMessage(rng *rand.Rand) []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(rng.Uint32())
	}
	return []byte(hex.EncodeToString(b))
}

// GenerateSig prepares a signature algorithm's inputs. withE10 adds the E10
// pools and holder key.
func GenerateSig(s scheme.Signature, rep, p int, seed uint64, withE10 bool) (*Prep, error) {
	start := time.Now()
	rng := rand.New(rand.NewPCG(seed, 0x6b70692d70726570)) // "kpi-prep"
	pk, sk, err := s.GenerateKey()
	if err != nil {
		return nil, err
	}
	if !s.Sizes().KeyLengthsOK(pk, sk) {
		return nil, fmt.Errorf("prep %s: key lengths %d/%d do not match %+v", s.Alg().ID, len(pk), len(sk), s.Sizes())
	}
	out := &Prep{Alg: s.Alg().ID, Rep: rep, Seed: seed, P: p, Sizes: s.Sizes(), PK: pk, SK: sk, Fingerprint: Fingerprint(pk)}
	for i := 0; i < p; i++ {
		m := HexMessage(rng)
		sg, err := s.Sign(sk, pk, m)
		if err != nil {
			return nil, err
		}
		if ok, err := s.Verify(pk, m, sg); err != nil || !ok {
			return nil, fmt.Errorf("prep %s: signature %d does not verify (%v)", s.Alg().ID, i, err)
		}
		out.Msgs = append(out.Msgs, m)
		out.Sigs = append(out.Sigs, sg)
	}
	if withE10 {
		if err := addE10(out, s, rng, p); err != nil {
			return nil, err
		}
	}
	out.PrepNS = time.Since(start).Nanoseconds()
	return out, nil
}

// GenerateKEM prepares a KEM's inputs.
func GenerateKEM(k scheme.KEM, rep, p int, seed uint64) (*Prep, error) {
	start := time.Now()
	pk, sk, err := k.GenerateKey()
	if err != nil {
		return nil, err
	}
	if !k.Sizes().KeyLengthsOK(pk, sk) {
		return nil, fmt.Errorf("prep %s: key lengths %d/%d do not match %+v", k.Alg().ID, len(pk), len(sk), k.Sizes())
	}
	out := &Prep{Alg: k.Alg().ID, Rep: rep, Seed: seed, P: p, Sizes: k.Sizes(), PK: pk, SK: sk, Fingerprint: Fingerprint(pk)}
	for i := 0; i < p; i++ {
		ct, ss, err := k.Encaps(pk)
		if err != nil {
			return nil, err
		}
		ss2, err := k.Decaps(sk, pk, ct)
		if err != nil || !bytes.Equal(ss, ss2) {
			return nil, fmt.Errorf("prep %s: ciphertext %d does not decapsulate (%v)", k.Alg().ID, i, err)
		}
		out.CTs = append(out.CTs, ct)
		out.SSs = append(out.SSs, ss)
	}
	out.PrepNS = time.Since(start).Nanoseconds()
	return out, nil
}

// Save writes the prep file atomically.
func Save(path string, p *Prep) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(p); err != nil {
		return err
	}
	return record.WriteFileAtomic(path, buf.Bytes(), 0o600)
}

// Load reads a prep file and checks that it is complete.
func Load(path string) (*Prep, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var p Prep
	if err := gob.NewDecoder(f).Decode(&p); err != nil {
		return nil, fmt.Errorf("prep %s: %w", path, err)
	}
	if p.P < 1 || len(p.PK) == 0 || len(p.SK) == 0 {
		return nil, fmt.Errorf("prep %s: incomplete", path)
	}
	if len(p.Msgs) != 0 && (len(p.Msgs) != p.P || len(p.Sigs) != p.P) {
		return nil, fmt.Errorf("prep %s: signature pool has %d/%d entries, want %d", path, len(p.Msgs), len(p.Sigs), p.P)
	}
	if len(p.CTs) != 0 && (len(p.CTs) != p.P || len(p.SSs) != p.P) {
		return nil, fmt.Errorf("prep %s: KEM pool has %d/%d entries, want %d", path, len(p.CTs), len(p.SSs), p.P)
	}
	if len(p.Msgs) == 0 && len(p.CTs) == 0 {
		return nil, fmt.Errorf("prep %s: empty pools", path)
	}
	return &p, nil
}
