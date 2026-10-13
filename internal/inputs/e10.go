package inputs

import (
	"bytes"
	"fmt"
	"math/rand/v2"

	"qchain-kpi/internal/qchain"
	"qchain-kpi/internal/scheme"
)

// E10 (ML-DSA): a separate holder key and three message pools.

// addE10 adds the E10 pools and holder key to a signature prep, continuing
// the prep's message generator: issuer = commitment hashes (N = 10
// attributes), holder = presentation payload hashes (N = 10, k = 5 disclosed),
// issuer-again = a second, independent pool of commitment hashes (the A/A arm).
func addE10(out *Prep, s scheme.Signature, rng *rand.Rand, p int) error {
	var err error
	if out.HolderPK, out.HolderSK, err = s.GenerateKey(); err != nil {
		return err
	}
	if bytes.Equal(out.HolderPK, out.PK) {
		return fmt.Errorf("prep %s: holder key equals issuer key", s.Alg().ID)
	}
	out.HolderFingerprint = Fingerprint(out.HolderPK)
	for i := 0; i < p; i++ {
		im, err1 := qchain.IssuerMessage(rng, 10)
		hm, err2 := qchain.HolderMessage(rng, 10, 5)
		am, err3 := qchain.IssuerMessage(rng, 10)
		if err1 != nil || err2 != nil || err3 != nil {
			return fmt.Errorf("prep E10 messages: %v %v %v", err1, err2, err3)
		}
		for _, m := range []string{im, hm, am} {
			if len(m) != 64 {
				return fmt.Errorf("prep E10: message %q is not 64 hex characters", m)
			}
		}
		out.IssuerMsgs = append(out.IssuerMsgs, []byte(im))
		out.HolderMsgs = append(out.HolderMsgs, []byte(hm))
		out.IssuerAgainMsgs = append(out.IssuerAgainMsgs, []byte(am))
	}
	return nil
}

// HasE10 reports whether the prep carries the E10 pools.
func (p *Prep) HasE10() bool {
	return len(p.HolderPK) > 0 && len(p.IssuerMsgs) == p.P && len(p.HolderMsgs) == p.P && len(p.IssuerAgainMsgs) == p.P
}
