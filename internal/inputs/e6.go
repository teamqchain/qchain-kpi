package inputs

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"time"

	"qchain-kpi/internal/scheme"
)

// E6: per message size, a pool of messages generated from a seed and valid
// signatures over them.

// E6Pool is the E6 input pool of one message size.
type E6Pool struct {
	Size int
	Seed uint64
	P    int
	Sigs [][]byte
}

// Messages regenerates the pool's messages.
func (e *E6Pool) Messages() [][]byte { return E6Messages(e.Seed, e.Size, e.P) }

// E6Messages returns n random messages of size bytes, determined by seed
// (ChaCha8 keyed with SHA-256 of the seed and the message index).
func E6Messages(seed uint64, size, n int) [][]byte {
	out := make([][]byte, n)
	var in [16]byte
	binary.LittleEndian.PutUint64(in[:8], seed)
	for i := range out {
		binary.LittleEndian.PutUint64(in[8:], uint64(i))
		r := rand.NewChaCha8(sha256.Sum256(in[:]))
		out[i] = make([]byte, size)
		r.Read(out[i])
	}
	return out
}

// AddE6 adds E6 pools to a signature prep: for each size, poolOf(size)
// messages and their signatures, every one verified.
func AddE6(p *Prep, s scheme.Signature, sizes []int, poolOf func(int) int, seed uint64) error {
	start := time.Now()
	rng := rand.New(rand.NewPCG(seed, 0x6b70692d65362d70)) // "kpi-e6-p"
	for _, size := range sizes {
		e := E6Pool{Size: size, Seed: rng.Uint64(), P: poolOf(size)}
		for i, m := range e.Messages() {
			sg, err := s.Sign(p.SK, p.PK, m)
			if err != nil {
				return err
			}
			if ok, err := s.Verify(p.PK, m, sg); err != nil || !ok {
				return fmt.Errorf("prep E6 %s: signature %d over %d bytes does not verify (%v)", s.Alg().ID, i, size, err)
			}
			e.Sigs = append(e.Sigs, sg)
		}
		p.E6 = append(p.E6, e)
	}
	p.PrepNS += time.Since(start).Nanoseconds()
	return nil
}

// HasE6 reports whether the prep carries a complete E6 pool for every size.
func (p *Prep) HasE6(sizes []int) bool {
	for _, z := range sizes {
		e, ok := p.E6Pool(z)
		if !ok || e.P < 1 || len(e.Sigs) != e.P {
			return false
		}
	}
	return true
}

// E6Pool returns the pool of one message size.
func (p *Prep) E6Pool(size int) (*E6Pool, bool) {
	for i := range p.E6 {
		if p.E6[i].Size == size {
			return &p.E6[i], true
		}
	}
	return nil, false
}
