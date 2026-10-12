package flow

import (
	"testing"

	"qchain-kpi/internal/cells"

	"qchain-kpi/internal/timing"
)

// TestStages runs every stage through the timing loop with tiny budgets for a
// spread of suites, including the classical ones and large values.
func TestStages(t *testing.T) {
	cfg := timing.Config{Mode: timing.Fast, Warmup: 2, Samples: 12, FirstLabel: "first", Seed: 1}
	cs := []cells.Cell{
		{Stage: cells.Activation, Sig: "ml-dsa-44", KEM: "ml-kem-768"},
		{Stage: cells.Activation, Sig: "ecdsa-p256", KEM: "dhkem-p256"},
		{Stage: cells.Activation, Sig: "falcon-padded-512", KEM: "hqc-1"},
		{Stage: cells.Issuance, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 10, V: 16},
		{Stage: cells.Issuance, Sig: "rsa3072-pss", KEM: "dhkem-p256", N: 64, V: 16},
		{Stage: cells.Issuance, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 5, V: 4096},
		{Stage: cells.Retrieval, KEM: "hqc-3", N: 20, V: 16},
		{Stage: cells.Retrieval, KEM: "dhkem-x25519", N: 1, V: 256},
		{Stage: cells.Presentation, Sig: "falcon-1024", K: 32, V: 16},
		{Stage: cells.Presentation, Sig: "ed25519", K: 1, V: 16},
		{Stage: cells.Verification, Sig: "slh-dsa-sha2-128f", KEM: "ml-kem-512", N: 10, K: 5, V: 16},
		{Stage: cells.Verification, Sig: "ecdsa-p256", KEM: "dhkem-p256", N: 64, K: 64, V: 16},
		{Stage: cells.Verification, Sig: "ml-dsa-87", KEM: "hqc-5", N: 1, K: 1, V: 16},
		{Stage: cells.Chained, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 10, K: 5, V: 16},
	}
	for _, c := range cs {
		st, err := Prepare(c, 42)
		if err != nil {
			t.Fatalf("%s %s: prepare: %v", c.Alg(), c.Op(), err)
		}
		res, err := timing.Run(st.Arms, cfg)
		if err != nil {
			t.Fatalf("%s %s: %v", c.Alg(), c.Op(), err)
		}
		wantArms := 1
		if c.Stage == cells.Verification {
			wantArms = 2
		}
		if len(res.Arms) != wantArms || len(res.Arms[0].PerOpNS) != cfg.Samples {
			t.Fatalf("%s %s: result %+v", c.Alg(), c.Op(), res.Arms)
		}
		n, err := st.After()
		if err != nil {
			t.Fatalf("%s %s: deferred checks: %v", c.Alg(), c.Op(), err)
		}
		if (c.Stage == cells.Issuance || c.Stage == cells.Chained) && n < 3 {
			t.Fatalf("%s %s: only %d envelopes fully checked", c.Alg(), c.Op(), n)
		}
	}
}
