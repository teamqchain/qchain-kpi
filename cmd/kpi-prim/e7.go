package main

import (
	"bytes"
	"fmt"
	"math/rand/v2"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/timing"
)

// E7: rejection time: valid and invalid input, interleaved.

// flipBit returns a copy of b with one bit flipped at a position drawn from rng.
func flipBit(rng *rand.Rand, b []byte) []byte {
	out := bytes.Clone(b)
	pos := rng.IntN(len(out) * 8)
	out[pos/8] ^= 1 << (pos % 8)
	return out
}

// e7Arms builds the interleaved arms of an E7 block. Each arm has its own
// library object. The invalid inputs are made before timing, from the prep's
// pools: entry i's signature or ciphertext with one bit flipped at a seeded
// position (a fresh position per entry), and for wrongmsg the signature of
// entry i over entry i+1's message (same length). Each sample's outcome is
// recorded in the raw file's bytes column (proto.E7Rejected,
// E7RejectedError, E7Accepted; for a batched sample, its last call's); every
// call must accept valid input and reject invalid input.
func e7Arms(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	if op != proto.E7Op(a) {
		return nil, fmt.Errorf("E7's operation for %s is %q, not %q", a.ID, proto.E7Op(a), op)
	}
	rng := rand.New(rand.NewPCG(seed, 0x6b70692d65372d74)) // "kpi-e7-t"
	n := p.P
	if n < 2 {
		return nil, fmt.Errorf("E7 needs a pool of at least 2 entries, prep has %d", n)
	}
	set := &armSet{hasBytes: true, after: noAfter}
	// outcome turns a call's result into its outcome code and fails the
	// block on a wrong one. It runs inside every call (a few nanoseconds), so
	// batched samples check every call, not only their last one.
	outcome := func(name string, i, got, expect int) (int, error) {
		switch {
		case expect == proto.E7Accepted && got != proto.E7Accepted:
			return got, fmt.Errorf("%w: %s: valid input %d rejected (outcome %d)", timing.ErrCheck, name, i%n, got)
		case expect != proto.E7Accepted && got == proto.E7Accepted:
			return got, fmt.Errorf("%w: %s: invalid input %d accepted", timing.ErrCheck, name, i%n)
		}
		return got, nil
	}
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return nil, err
		}
		if len(p.Msgs) != n || len(p.Sigs) != n {
			return nil, fmt.Errorf("prep %s has no signature pool", a.ID)
		}
		bad := make([][]byte, n)
		for i := range bad {
			bad[i] = flipBit(rng, p.Sigs[i])
		}
		verify := func(name string, msg, sig func(int) []byte, expect int) error {
			v, err := s.NewVerifierReuse(p.PK)
			if err != nil {
				return err
			}
			set.closers = append(set.closers, v.Close)
			set.arms = append(set.arms, timing.Arm{
				Name: name,
				Call: func(i int) (int, error) {
					ok, err := v.Verify(msg(i%n), sig(i%n))
					switch {
					case err != nil:
						return outcome(name, i, proto.E7RejectedError, expect)
					case ok:
						return outcome(name, i, proto.E7Accepted, expect)
					}
					return outcome(name, i, proto.E7Rejected, expect)
				},
			})
			return nil
		}
		msgOf := func(i int) []byte { return p.Msgs[i] }
		for _, arm := range []struct {
			name     string
			msg, sig func(int) []byte
			expect   int
		}{
			{proto.ArmValid, msgOf, func(i int) []byte { return p.Sigs[i] }, proto.E7Accepted},
			{proto.ArmBadSig, msgOf, func(i int) []byte { return bad[i] }, proto.E7Rejected},
			{proto.ArmWrongMsg, func(i int) []byte { return p.Msgs[(i+1)%n] }, func(i int) []byte { return p.Sigs[i] }, proto.E7Rejected},
		} {
			if err := verify(arm.name, arm.msg, arm.sig, arm.expect); err != nil {
				set.close()
				return nil, err
			}
		}
		return set, nil
	}
	k, err := newKEM(a)
	if err != nil {
		return nil, err
	}
	if len(p.CTs) != n || len(p.SSs) != n {
		return nil, fmt.Errorf("prep %s has no KEM pool", a.ID)
	}
	tampered := make([][]byte, n)
	for i := range tampered {
		tampered[i] = flipBit(rng, p.CTs[i])
	}
	for _, arm := range []struct {
		name   string
		ct     [][]byte
		expect int
	}{{proto.ArmValid, p.CTs, proto.E7Accepted}, {proto.ArmTampered, tampered, proto.E7Rejected}} {
		d, err := k.NewDecapsulatorReuse(p.SK, p.PK)
		if err != nil {
			set.close()
			return nil, err
		}
		set.closers = append(set.closers, d.Close)
		cts, name, expect := arm.ct, arm.name, arm.expect
		set.arms = append(set.arms, timing.Arm{
			Name: name,
			Call: func(i int) (int, error) {
				ss, err := d.Decaps(cts[i%n])
				switch {
				case err != nil:
					return outcome(name, i, proto.E7RejectedError, expect)
				case bytes.Equal(ss, p.SSs[i%n]):
					return outcome(name, i, proto.E7Accepted, expect)
				}
				return outcome(name, i, proto.E7Rejected, expect) // implicit rejection: a different secret
			},
		})
	}
	return set, nil
}
