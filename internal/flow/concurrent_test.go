package flow

import (
	"testing"
	"time"

	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/conc"
)

// TestConcurrentStages (E12): several workers, each with its own prepared
// stage, run QChain's copied code at the same time and every call checks out,
// for a liboqs suite and the classical baseline (the OpenSSL twins). Run with
// -race in the image's test stage, it also proves the copied code and the
// twins share no unsynchronised state.
func TestConcurrentStages(t *testing.T) {
	for _, s := range [][2]string{{"ml-dsa-44", "ml-kem-768"}, {"ecdsa-p256", "dhkem-p256"}} {
		for _, c := range []cells.Cell{
			{Stage: cells.Verification, Sig: s[0], KEM: s[1], N: 10, K: 5, V: 16},
			{Stage: cells.Issuance, Sig: s[0], KEM: s[1], N: 10, V: 16},
		} {
			var ws []conc.Worker
			var setups []*Setup
			for k := 0; k < 3; k++ {
				st, err := Prepare(c, uint64(100+k))
				if err != nil {
					t.Fatalf("%s %s: %v", c.Alg(), c.Op(), err)
				}
				setups = append(setups, st)
				ws = append(ws, conc.Worker{Call: st.Arms[0].Call, Check: st.Arms[0].Check})
			}
			r, err := conc.Run(ws, conc.Config{Warmup: 50 * time.Millisecond, Window: 300 * time.Millisecond, CheckEvery: 1})
			if err != nil {
				t.Fatalf("%s %s: %v", c.Alg(), c.Op(), err)
			}
			for k, n := range r.PerWorker {
				if n == 0 {
					t.Fatalf("%s %s: worker %d completed no call in the window", c.Alg(), c.Op(), k)
				}
			}
			for _, st := range setups {
				if _, err := st.After(); err != nil {
					t.Fatalf("%s %s: %v", c.Alg(), c.Op(), err)
				}
			}
		}
	}
}
