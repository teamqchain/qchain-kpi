package algs

import "testing"

func TestRegistryCounts(t *testing.T) {
	all := All()
	if len(all) != 30 {
		t.Fatalf("registry has %d algorithms, want 30", len(all))
	}
	var pqSig, pqKEM, clSig, clKEM int
	ids := map[string]bool{}
	names := map[string]bool{}
	for _, a := range all {
		if ids[a.ID] {
			t.Fatalf("duplicate id %s", a.ID)
		}
		ids[a.ID] = true
		if !a.Classical() {
			if names[a.OQSName] {
				t.Fatalf("duplicate liboqs name %s", a.OQSName)
			}
			names[a.OQSName] = true
			if a.Category < 1 || a.Category > 5 {
				t.Fatalf("%s: category %d", a.ID, a.Category)
			}
		} else if a.Category != 0 || a.Tier != "C" {
			t.Fatalf("%s: classical baseline must have category 0 and tier C", a.ID)
		}
		switch {
		case a.Track == Sig && !a.Classical():
			pqSig++
		case a.Track == KEM && !a.Classical():
			pqKEM++
		case a.Track == Sig:
			clSig++
		default:
			clKEM++
		}
	}
	if pqSig != 19 || pqKEM != 6 || clSig != 3 || clKEM != 2 {
		t.Fatalf("counts pqSig=%d pqKEM=%d clSig=%d clKEM=%d, want 19/6/3/2", pqSig, pqKEM, clSig, clKEM)
	}
}

func TestExactLiboqsNames(t *testing.T) {
	// Spot-check the exact strings verified against liboqs 0.16.0's headers.
	want := map[string]string{
		"slh-dsa-sha2-128s":  "SLH_DSA_PURE_SHA2_128S",
		"slh-dsa-shake-256f": "SLH_DSA_PURE_SHAKE_256F",
		"falcon-padded-1024": "Falcon-padded-1024",
		"hqc-5":              "HQC-5",
		"ml-kem-768":         "ML-KEM-768",
		"ml-dsa-44":          "ML-DSA-44",
	}
	for id, name := range want {
		a, ok := ByID(id)
		if !ok || a.OQSName != name {
			t.Fatalf("%s: got %q, want %q", id, a.OQSName, name)
		}
	}
	if n := len(OQSNames(Sig)); n != 19 {
		t.Fatalf("OQSNames(Sig) = %d, want 19", n)
	}
	if n := len(OQSNames(KEM)); n != 6 {
		t.Fatalf("OQSNames(KEM) = %d, want 6", n)
	}
}

func TestExpand(t *testing.T) {
	got, err := Expand([]string{"SLH-DSA", "ml-dsa-44", "SLH-DSA"})
	if err != nil || len(got) != 13 {
		t.Fatalf("Expand = %d (%v), want 13", len(got), err)
	}
	if _, err := Expand([]string{"nope"}); err == nil {
		t.Fatal("unknown selector accepted")
	}
	all, _ := Expand([]string{"all"})
	if len(all) != 30 {
		t.Fatalf("all = %d", len(all))
	}
}
