package main

import (
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// E1: primitive latency. Each operation's single "reuse" arm: the library's
// long-lived object, one process per operation (also used by the probe pass).

// e1Arms builds the single "reuse" arm of an E1 (or probe) block.
func e1Arms(a algs.Alg, op string, p *inputs.Prep, seed uint64) (*armSet, error) {
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return nil, err
		}
		if len(p.Msgs) != p.P {
			return nil, fmt.Errorf("prep %s has no signature pool", a.ID)
		}
		switch op {
		case "keygen":
			return sigKeygenArm(s, seed)
		case "sign":
			return signArm(s, p)
		case "verify":
			return verifyArm(s, p)
		}
	} else {
		k, err := newKEM(a)
		if err != nil {
			return nil, err
		}
		if len(p.CTs) != p.P {
			return nil, fmt.Errorf("prep %s has no KEM pool", a.ID)
		}
		switch op {
		case "keygen":
			return kemKeygenArm(k, seed)
		case "encaps":
			return encapsArm(k, p)
		case "decaps":
			return decapsArm(k, p)
		}
	}
	return nil, fmt.Errorf("%s has no operation %q", a.ID, op)
}

func sigKeygenArm(s scheme.Signature, seed uint64) (*armSet, error) {
	r, err := s.NewKeygenReuse()
	if err != nil {
		return nil, err
	}
	sizes := s.Sizes()
	sub := newKeySubset(seed)
	var pk, sk []byte
	set := &armSet{hasBytes: sizes.SKVariable, closers: []func(){r.Close}}
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(int) (int, error) {
			var err error
			pk, sk, err = r.Keygen()
			return len(pk) + len(sk), err
		},
		Check: func(i, _ int) error {
			if !sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("key lengths %d/%d, want %+v", len(pk), len(sk), sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		},
	}}
	set.after = func() (int, error) {
		keys := sub.final()
		for n, k := range keys {
			sg, err := s.Sign(k.sk, k.pk, checkMessage)
			if err != nil {
				return n, fmt.Errorf("generated key %d cannot sign: %v", n, err)
			}
			if ok, err := s.Verify(k.pk, checkMessage, sg); err != nil || !ok {
				return n, fmt.Errorf("generated key %d: signature does not verify (%v)", n, err)
			}
		}
		return len(keys), nil
	}
	return set, nil
}

func signArm(s scheme.Signature, p *inputs.Prep) (*armSet, error) {
	r, err := s.NewSignerReuse(p.SK, p.PK)
	if err != nil {
		return nil, err
	}
	sizes := s.Sizes()
	set := &armSet{hasBytes: sizes.SigVariable, after: noAfter, closers: []func(){r.Close}}
	var last []byte
	var v scheme.VerifierReuse // created at the first check, after the cold call
	set.closers = append(set.closers, func() {
		if v != nil {
			v.Close()
		}
	})
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(i int) (int, error) {
			var err error
			last, err = r.Sign(p.Msgs[i%p.P])
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
			ok, err := v.Verify(p.Msgs[i%p.P], last)
			if err != nil || !ok {
				return fmt.Errorf("signature %d does not verify (%v)", i, err)
			}
			return nil
		},
	}}
	return set, nil
}

func verifyArm(s scheme.Signature, p *inputs.Prep) (*armSet, error) {
	r, err := s.NewVerifierReuse(p.PK)
	if err != nil {
		return nil, err
	}
	var ok bool
	set := &armSet{after: noAfter, closers: []func(){r.Close}}
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(i int) (int, error) {
			var err error
			ok, err = r.Verify(p.Msgs[i%p.P], p.Sigs[i%p.P])
			return 0, err
		},
		Check: func(i, _ int) error {
			if !ok {
				return fmt.Errorf("valid signature %d rejected", i%p.P)
			}
			return nil
		},
	}}
	return set, nil
}

func kemKeygenArm(k scheme.KEM, seed uint64) (*armSet, error) {
	r, err := k.NewKeygenReuse()
	if err != nil {
		return nil, err
	}
	sizes := k.Sizes()
	sub := newKeySubset(seed)
	var pk, sk []byte
	set := &armSet{closers: []func(){r.Close}}
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(int) (int, error) {
			var err error
			pk, sk, err = r.Keygen()
			return len(pk) + len(sk), err
		},
		Check: func(i, _ int) error {
			if !sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("key lengths %d/%d, want %+v", len(pk), len(sk), sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		},
	}}
	set.after = func() (int, error) {
		keys := sub.final()
		for n, kp := range keys {
			ct, ss, err := k.Encaps(kp.pk)
			if err != nil {
				return n, fmt.Errorf("generated key %d cannot encapsulate: %v", n, err)
			}
			ss2, err := k.Decaps(kp.sk, kp.pk, ct)
			if err != nil || !equalBytes(ss, ss2) {
				return n, fmt.Errorf("generated key %d: decapsulation mismatch (%v)", n, err)
			}
		}
		return len(keys), nil
	}
	return set, nil
}

func encapsArm(k scheme.KEM, p *inputs.Prep) (*armSet, error) {
	r, err := k.NewEncapsulatorReuse(p.PK)
	if err != nil {
		return nil, err
	}
	sizes := k.Sizes()
	var ct, ss []byte
	var d scheme.DecapsulatorReuse // created at the first check, after the cold call
	set := &armSet{after: noAfter, closers: []func(){r.Close}}
	set.closers = append(set.closers, func() {
		if d != nil {
			d.Close()
		}
	})
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(int) (int, error) {
			var err error
			ct, ss, err = r.Encaps()
			return len(ct), err
		},
		Check: func(i, _ int) error {
			if len(ct) != sizes.Ciphertext || len(ss) != sizes.SharedSecret {
				return fmt.Errorf("ciphertext/secret lengths %d/%d, want %d/%d", len(ct), len(ss), sizes.Ciphertext, sizes.SharedSecret)
			}
			if d == nil {
				var err error
				if d, err = k.NewDecapsulatorReuse(p.SK, p.PK); err != nil {
					return err
				}
			}
			ss2, err := d.Decaps(ct)
			if err != nil || !equalBytes(ss, ss2) {
				return fmt.Errorf("encapsulation %d does not decapsulate to the same secret (%v)", i, err)
			}
			return nil
		},
	}}
	return set, nil
}

func decapsArm(k scheme.KEM, p *inputs.Prep) (*armSet, error) {
	r, err := k.NewDecapsulatorReuse(p.SK, p.PK)
	if err != nil {
		return nil, err
	}
	var ss []byte
	set := &armSet{after: noAfter, closers: []func(){r.Close}}
	set.arms = []timing.Arm{{
		Name: proto.ArmReuse,
		Call: func(i int) (int, error) {
			var err error
			ss, err = r.Decaps(p.CTs[i%p.P])
			return len(ss), err
		},
		Check: func(i, _ int) error {
			if !equalBytes(ss, p.SSs[i%p.P]) {
				return fmt.Errorf("decapsulation %d returned the wrong secret", i%p.P)
			}
			return nil
		},
	}}
	return set, nil
}
