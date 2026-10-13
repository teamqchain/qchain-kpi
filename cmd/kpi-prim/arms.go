package main

import (
	"bytes"
	"fmt"

	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// armSet is the measured arms of one block plus their bookkeeping.
type armSet struct {
	arms     []timing.Arm
	hasBytes bool                // the output size varies, so the raw file gets a bytes column
	after    func() (int, error) // deferred correctness checks; returns the number of keys fully exercised
	closers  []func()
	// resets[k] (optional) restores arm k's state between calls outside the
	// timed region (E2c init/clean, E2 reinit with a secret key). Check
	// already does it during sampling; the allocation count needs it too.
	resets []func() error
}

func (s *armSet) resetOf(k int) func() error {
	if k < len(s.resets) {
		return s.resets[k]
	}
	return nil
}

func (s *armSet) close() {
	for i := len(s.closers) - 1; i >= 0; i-- {
		s.closers[i]()
	}
}

func noAfter() (int, error) { return 0, nil }

// --------------------------------------------------------------- key subset

// keySubset keeps copies of a seeded subset of generated keys (every 50th
// call, starting at a seeded offset, and at least 3 per block) so they can be
// fully exercised after the timed phase. Exercising every key would take
// hours for the slow SLH-DSA sets; every key's lengths are checked on the spot.
type keySubset struct {
	offset int
	saved  []savedKey
	ring   [3]savedKey // the most recent unsaved keys, kept while fewer than 3 are saved
	nring  int
}

type savedKey struct {
	pk, sk []byte
}

func newKeySubset(seed uint64) *keySubset { return &keySubset{offset: int(seed % 50)} }

func (t *keySubset) observe(i int, pk, sk []byte) {
	if i%50 == t.offset {
		t.saved = append(t.saved, savedKey{bytes.Clone(pk), bytes.Clone(sk)})
		return
	}
	if len(t.saved) < 3 {
		t.ring[t.nring%3] = savedKey{bytes.Clone(pk), bytes.Clone(sk)}
		t.nring++
	}
}

func (t *keySubset) final() []savedKey {
	out := append([]savedKey(nil), t.saved...)
	for _, k := range t.ring {
		if len(out) >= 3 {
			break
		}
		if k.pk != nil {
			out = append(out, k)
		}
	}
	return out
}

// ------------------------------------------------------------------ signatures

func sigLengthOK(sizes scheme.Sizes, sg []byte) error {
	if sizes.SigVariable {
		if len(sg) == 0 || len(sg) > sizes.Signature {
			return fmt.Errorf("signature length %d outside 1..%d", len(sg), sizes.Signature)
		}
		return nil
	}
	if len(sg) != sizes.Signature {
		return fmt.Errorf("signature length %d, want %d", len(sg), sizes.Signature)
	}
	return nil
}

// ------------------------------------------------------------------------ KEMs

// ------------------------------------------------------------------------ E10

// checkArmOrder makes sure a builder produced exactly the arms (and order)
// that proto declares, which the orchestrator relies on for batch sizes.
func checkArmOrder(set *armSet, want []string) error {
	if len(set.arms) != len(want) {
		return fmt.Errorf("built %d arms, want %v", len(set.arms), want)
	}
	for k, a := range set.arms {
		if a.Name != want[k] {
			return fmt.Errorf("arm %d is %q, want %q (%v)", k, a.Name, want[k], want)
		}
	}
	return nil
}

// exerciseSig fully checks a seeded subset of generated signature keys.
func exerciseSig(s scheme.Signature, sub *keySubset) func() (int, error) {
	return func() (int, error) {
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
}

// exerciseKEM fully checks a seeded subset of generated KEM keys.
func exerciseKEM(k scheme.KEM, sub *keySubset) func() (int, error) {
	return func() (int, error) {
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
}
