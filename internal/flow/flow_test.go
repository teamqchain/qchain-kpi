package flow

import (
	"testing"

	"qchain-kpi/internal/timing"
)

func TestGrid(t *testing.T) {
	g := Grid()
	count := map[string]int{}
	for _, c := range g {
		if err := c.Validate(); err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
		back, err := ParseCell(c.Alg(), c.Op())
		if err != nil || back != c {
			t.Fatalf("round trip %+v -> %s / %s -> %+v (%v)", c, c.Alg(), c.Op(), back, err)
		}
		count[c.Stage]++
	}
	want := map[string]int{Activation: 176, Issuance: 250, Retrieval: 50, Presentation: 108, Verification: 394, Chained: 13}
	for s, n := range want {
		if count[s] != n {
			t.Fatalf("%s: %d cells, want %d (all: %v)", s, count[s], n, count)
		}
	}
	if len(g) != 991 {
		t.Fatalf("%d cells", len(g))
	}
	if ks := Ks(1); len(ks) != 1 || ks[0] != 1 {
		t.Fatalf("Ks(1) = %v", ks)
	}
	if ks := Ks(64); len(ks) != 3 || ks[1] != 32 {
		t.Fatalf("Ks(64) = %v", ks)
	}
}

// TestStages runs every stage through the timing loop with tiny budgets for a
// spread of suites, including the classical ones and large values.
func TestStages(t *testing.T) {
	cfg := timing.Config{Mode: timing.Fast, Warmup: 2, Samples: 12, FirstLabel: "first", Seed: 1}
	cells := []Cell{
		{Stage: Activation, Sig: "ml-dsa-44", KEM: "ml-kem-768"},
		{Stage: Activation, Sig: "ecdsa-p256", KEM: "dhkem-p256"},
		{Stage: Activation, Sig: "falcon-padded-512", KEM: "hqc-1"},
		{Stage: Issuance, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 10, V: 16},
		{Stage: Issuance, Sig: "rsa3072-pss", KEM: "dhkem-p256", N: 64, V: 16},
		{Stage: Issuance, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 5, V: 4096},
		{Stage: Retrieval, KEM: "hqc-3", N: 20, V: 16},
		{Stage: Retrieval, KEM: "dhkem-x25519", N: 1, V: 256},
		{Stage: Presentation, Sig: "falcon-1024", K: 32, V: 16},
		{Stage: Presentation, Sig: "ed25519", K: 1, V: 16},
		{Stage: Verification, Sig: "slh-dsa-sha2-128f", KEM: "ml-kem-512", N: 10, K: 5, V: 16},
		{Stage: Verification, Sig: "ecdsa-p256", KEM: "dhkem-p256", N: 64, K: 64, V: 16},
		{Stage: Verification, Sig: "ml-dsa-87", KEM: "hqc-5", N: 1, K: 1, V: 16},
		{Stage: Chained, Sig: "ml-dsa-44", KEM: "ml-kem-768", N: 10, K: 5, V: 16},
	}
	for _, c := range cells {
		st, err := Prepare(c, 42)
		if err != nil {
			t.Fatalf("%s %s: prepare: %v", c.Alg(), c.Op(), err)
		}
		res, err := timing.Run(st.Arms, cfg)
		if err != nil {
			t.Fatalf("%s %s: %v", c.Alg(), c.Op(), err)
		}
		wantArms := 1
		if c.Stage == Verification {
			wantArms = 2
		}
		if len(res.Arms) != wantArms || len(res.Arms[0].PerOpNS) != cfg.Samples {
			t.Fatalf("%s %s: result %+v", c.Alg(), c.Op(), res.Arms)
		}
		n, err := st.After()
		if err != nil {
			t.Fatalf("%s %s: deferred checks: %v", c.Alg(), c.Op(), err)
		}
		if (c.Stage == Issuance || c.Stage == Chained) && n < 3 {
			t.Fatalf("%s %s: only %d envelopes fully checked", c.Alg(), c.Op(), n)
		}
	}
}

func TestParseCellRejects(t *testing.T) {
	for _, x := range [][2]string{
		{"ml-dsa-44+ml-kem-768", "issuance-n10"},            // V missing
		{"ml-kem-768", "presentation-k5-v16"},               // presentation needs a signature
		{"ml-dsa-44+ml-kem-768", "verification-n5-k10-v16"}, // k > N
		{"ml-dsa-44+ml-kem-768", "teleport"},
		{"ml-kem-768+ml-dsa-44", "activation"}, // order swapped
	} {
		if _, err := ParseCell(x[0], x[1]); err == nil {
			t.Fatalf("%v accepted", x)
		}
	}
}
