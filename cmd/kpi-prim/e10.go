package main

import (
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// E10: ML-DSA issuer/holder equivalence.

// e10Arms builds the three interleaved E10 arms. Each arm has its own signer
// object and its own message pool; issuer and issuer-again share the key
// value, so their difference is the A/A noise floor.
func e10Arms(a algs.Alg, p *inputs.Prep) (*armSet, error) {
	s, err := newSignature(a)
	if err != nil {
		return nil, err
	}
	sizes := s.Sizes()
	set := &armSet{after: noAfter}
	var issuerV, holderV scheme.VerifierReuse
	set.closers = append(set.closers, func() {
		if issuerV != nil {
			issuerV.Close()
		}
		if holderV != nil {
			holderV.Close()
		}
	})
	type armDef struct {
		name   string
		sk, pk []byte
		msgs   [][]byte
		holder bool
	}
	defs := []armDef{
		{proto.ArmIssuer, p.SK, p.PK, p.IssuerMsgs, false},
		{proto.ArmHolder, p.HolderSK, p.HolderPK, p.HolderMsgs, true},
		{proto.ArmIssuerAgain, p.SK, p.PK, p.IssuerAgainMsgs, false},
	}
	for _, d := range defs {
		d := d
		r, err := s.NewSignerReuse(d.sk, d.pk)
		if err != nil {
			set.close()
			return nil, err
		}
		set.closers = append(set.closers, r.Close)
		var last []byte
		set.arms = append(set.arms, timing.Arm{
			Name: d.name,
			Call: func(i int) (int, error) {
				var err error
				last, err = r.Sign(d.msgs[i%p.P])
				return len(last), err
			},
			Check: func(i, _ int) error {
				if err := sigLengthOK(sizes, last); err != nil {
					return err
				}
				v := &issuerV
				if d.holder {
					v = &holderV
				}
				if *v == nil {
					var err error
					if *v, err = s.NewVerifierReuse(d.pk); err != nil {
						return err
					}
				}
				ok, err := (*v).Verify(d.msgs[i%p.P], last)
				if err != nil || !ok {
					return fmt.Errorf("%s signature %d does not verify (%v)", d.name, i, err)
				}
				return nil
			},
		})
	}
	return set, nil
}
