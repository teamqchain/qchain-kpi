package main

import (
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// E6: sign and verify time against message size (the reuse arm, as E1).

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
