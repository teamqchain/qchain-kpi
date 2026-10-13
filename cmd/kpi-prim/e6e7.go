package main

// E6 (message size) and E7 (rejection) arms. Both use the libraries' reuse
// objects, as E1 does, so their results line up with E1's.

import (
	"bytes"
	"fmt"
	"math/rand/v2"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// e6Arms builds the single reuse arm of an E6 block: sign or verify over the
// prep's pool of messages of the block's size. Every signature is verified
// and every verification must succeed (checks run outside the timed region).
func e6Arms(a algs.Alg, opField string, p *inputs.Prep) (*armSet, error) {
	op, size, err := proto.ParseE6Op(opField)
	if err != nil {
		return nil, err
	}
	if a.Track != algs.Sig {
		return nil, fmt.Errorf("E6 is defined for signature schemes, not %s", a.ID)
	}
	pool, ok := p.E6Pool(size)
	if !ok || len(pool.Sigs) != pool.P {
		return nil, fmt.Errorf("prep %s r%d has no E6 pool of %d bytes", p.Alg, p.Rep, size)
	}
	s, err := newSignature(a)
	if err != nil {
		return nil, err
	}
	msgs := pool.Messages() // regenerated outside the timed region
	n := pool.P
	sizes := s.Sizes()
	if op == "sign" {
		r, err := s.NewSignerReuse(p.SK, p.PK)
		if err != nil {
			return nil, err
		}
		var last []byte
		var v scheme.VerifierReuse // created at the first check, after the cold call
		set := &armSet{hasBytes: sizes.SigVariable, after: noAfter, closers: []func(){r.Close}}
		set.closers = append(set.closers, func() {
			if v != nil {
				v.Close()
			}
		})
		set.arms = []timing.Arm{{
			Name: proto.ArmReuse,
			Call: func(i int) (int, error) {
				var err error
				last, err = r.Sign(msgs[i%n])
				return len(last), err
			},
			Check: func(i, _ int) error {
				if err := sigLengthOK(sizes, last); err != nil {
					return err
				}
				if v == nil {
					var err error
					if v, err = s.NewVerifierReuse(p.PK); err != nil {
						return err
					}
				}
				if ok, err := v.Verify(msgs[i%n], last); err != nil || !ok {
					return fmt.Errorf("signature %d over %d bytes does not verify (%v)", i, size, err)
				}
				return nil
			},
		}}
		return set, nil
	}
	r, err := s.NewVerifierReuse(p.PK)
	if err != nil {
		return nil, err
	}
	var okv bool
	set := &armSet{after: noAfter, closers: []func(){r.Close}}
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(i int) (int, error) {
			var err error
			okv, err = r.Verify(msgs[i%n], pool.Sigs[i%n])
			return 0, err
		},
		Check: func(i, _ int) error {
			if !okv {
				return fmt.Errorf("valid signature %d over %d bytes rejected", i%n, size)
			}
			return nil
		},
	}}
	return set, nil
}

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
