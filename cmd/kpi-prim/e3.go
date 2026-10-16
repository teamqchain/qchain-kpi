package main

// E3: integration-pattern cost (internal/proto/e3.go names the layers). One
// block per (algorithm, operation): the C library timed inside C (l0), the
// same C call timed from Go (l0g), E1's reuse arm (l1) and, for the classical
// algorithms, Go's standard library (l2), interleaved, each with the full
// approved budget, on the same keys and the same input pool. Every call is
// checked; the l2 results are checked with OpenSSL, so the two libraries are
// shown to compute the same scheme on every call. The C jobs are E9's
// (internal/pqc and internal/classical MemJob), with the input pool copied
// into C memory once, so a timed call only selects a pointer.

import (
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/gostd"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/l0"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// e3Arms builds an E3 block's layers in proto.E3Arms order.
func e3Arms(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	var sets []*armSet
	fail := func(err error) (*armSet, error) {
		mergeSets(sets...).close()
		return nil, err
	}
	for _, selfTimed := range []bool{true, false} {
		s, err := e3CArm(a, op, p, seed, selfTimed)
		if err != nil {
			return fail(err)
		}
		sets = append(sets, s)
	}
	l1, err := e1Arms(a, op, p, seed)
	if err != nil {
		return fail(err)
	}
	l1.arms[0].Name = proto.ArmL1
	sets = append(sets, l1)
	if a.Classical() {
		l2, err := e3GoArm(a, op, p, seed)
		if err != nil {
			return fail(err)
		}
		sets = append(sets, l2)
	}
	set := mergeSets(sets...)
	if err := checkArmOrder(set, proto.E3Arms(a)); err != nil {
		set.close()
		return nil, err
	}
	return set, nil
}

// e3Pool is the input pool of an operation: messages (sign, verify),
// signatures (verify), ciphertexts (decaps).
func e3Pool(op string, p *inputs.Prep) (in1, in2 [][]byte) {
	switch op {
	case "sign":
		return p.Msgs, nil
	case "verify":
		return p.Msgs, p.Sigs
	case "decaps":
		return p.CTs, nil
	}
	return nil, nil
}

// e3CArm is the l0 arm (selfTimed: timed inside C) or the l0g arm (the same
// C call, timed by the engine from Go). Both check every call as E9 does.
func e3CArm(a algs.Alg, op string, p *inputs.Prep, seed uint64, selfTimed bool) (*armSet, error) {
	check, after, err := e9CChecks(a, op, p, seed)
	if err != nil {
		return nil, err
	}
	job, err := newMemJob(a, op, p)
	if err != nil {
		return nil, err
	}
	job.SetPool(e3Pool(op, p))
	fn, arg := job.Call()
	var rc int
	arm := timing.Arm{Name: proto.ArmL0g,
		Call: func(i int) (int, error) {
			job.Select(i)
			rc = l0.Call(fn, arg)
			return 0, nil
		},
		Check: func(i, _ int) error {
			out1, out2 := job.Outputs()
			return check(i, rc, out1, out2)
		},
	}
	if selfTimed {
		arm.Name = proto.ArmL0
		arm.Timed = func(i int) (int64, int, error) {
			job.Select(i)
			ns, r := l0.Time(fn, arg)
			rc = r
			return ns, 0, nil
		}
	}
	return &armSet{arms: []timing.Arm{arm}, after: after, closers: []func(){job.Close}}, nil
}

// e3GoArm is the l2 arm: Go's standard library, the key objects parsed once.
func e3GoArm(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return nil, err
		}
		return e3GoSig(a, s, op, p, seed)
	}
	k, err := newKEM(a)
	if err != nil {
		return nil, err
	}
	return e3GoKEM(a, k, op, p, seed)
}

func e3GoSig(a algs.Alg, s scheme.Signature, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	sizes := s.Sizes()
	arm := timing.Arm{Name: proto.ArmL2}
	set := &armSet{after: noAfter}
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		var pk, sk []byte
		set.hasBytes = sizes.SKVariable
		arm.Call = func(int) (int, error) {
			var err error
			pk, sk, err = gostd.SigKeygen(a)
			return len(pk) + len(sk), err
		}
		arm.Check = func(i, _ int) error {
			if !sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("Go key lengths %d/%d, want %+v", len(pk), len(sk), sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		}
		set.after = exerciseSig(s, sub) // Go's keys signed and verified with OpenSSL
	case "sign":
		sign, err := gostd.NewSigner(a, p.SK, p.PK)
		if err != nil {
			return nil, err
		}
		var last []byte
		set.hasBytes = sizes.SigVariable
		arm.Call = func(i int) (int, error) {
			var err error
			last, err = sign(p.Msgs[i%p.P])
			return len(last), err
		}
		arm.Check = func(i, _ int) error {
			if err := sigLengthOK(sizes, last); err != nil {
				return err
			}
			if ok, err := s.Verify(p.PK, p.Msgs[i%p.P], last); err != nil || !ok {
				return fmt.Errorf("OpenSSL rejects Go's signature %d (%v)", i, err)
			}
			return nil
		}
	case "verify":
		verify, err := gostd.NewVerifier(a, p.PK)
		if err != nil {
			return nil, err
		}
		var ok bool
		arm.Call = func(i int) (int, error) {
			var err error
			ok, err = verify(p.Msgs[i%p.P], p.Sigs[i%p.P])
			return 0, err
		}
		arm.Check = func(i, _ int) error {
			if !ok {
				return fmt.Errorf("Go rejects OpenSSL's valid signature %d", i%p.P)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("%s has no operation %q", a.ID, op)
	}
	set.arms = []timing.Arm{arm}
	return set, nil
}

func e3GoKEM(a algs.Alg, k scheme.KEM, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	sizes := k.Sizes()
	arm := timing.Arm{Name: proto.ArmL2}
	set := &armSet{after: noAfter}
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		var pk, sk []byte
		arm.Call = func(int) (int, error) {
			var err error
			pk, sk, err = gostd.KEMKeygen(a)
			return len(pk) + len(sk), err
		}
		arm.Check = func(i, _ int) error {
			if !sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("Go key lengths %d/%d, want %+v", len(pk), len(sk), sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		}
		set.after = exerciseKEM(k, sub) // Go's keys used by OpenSSL
	case "encaps":
		enc, err := gostd.NewEncapsulator(a, p.PK)
		if err != nil {
			return nil, err
		}
		var ct, ss []byte
		arm.Call = func(int) (int, error) {
			var err error
			ct, ss, err = enc()
			return len(ct), err
		}
		arm.Check = func(i, _ int) error {
			if len(ct) != sizes.Ciphertext || len(ss) != sizes.SharedSecret {
				return fmt.Errorf("Go ciphertext/secret lengths %d/%d", len(ct), len(ss))
			}
			ss2, err := k.Decaps(p.SK, p.PK, ct)
			if err != nil || !equalBytes(ss, ss2) {
				return fmt.Errorf("OpenSSL decapsulates Go's ciphertext %d differently (%v)", i, err)
			}
			return nil
		}
	case "decaps":
		dec, err := gostd.NewDecapsulator(a, p.SK, p.PK)
		if err != nil {
			return nil, err
		}
		var ss []byte
		arm.Call = func(i int) (int, error) {
			var err error
			ss, err = dec(p.CTs[i%p.P])
			return len(ss), err
		}
		arm.Check = func(i, _ int) error {
			if !equalBytes(ss, p.SSs[i%p.P]) {
				return fmt.Errorf("Go decapsulates OpenSSL's ciphertext %d to a different secret", i%p.P)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("%s has no operation %q", a.ID, op)
	}
	set.arms = []timing.Arm{arm}
	return set, nil
}

// e3NullArms: a cgo call to a C function that does nothing, batched; and the
// C timer's floor (two readings with nothing between them, timed in C).
func e3NullArms() *armSet {
	return &armSet{after: noAfter, arms: []timing.Arm{
		{Name: proto.ArmCgoNull, Batch: proto.E3NullBatch, Call: func(int) (int, error) { l0.Null(); return 0, nil }},
		{Name: proto.ArmCEmpty, Call: func(int) (int, error) { l0.Empty(); return 0, nil },
			Timed: func(int) (int64, int, error) { return l0.Empty(), 0, nil }},
	}}
}
