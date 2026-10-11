package qchain

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The production golden commitment (offchain/commitment_test.go:21-52, main
// repo commit 9026aba): the verbatim copy must reproduce it byte for byte.
func TestGoldenCommitment(t *testing.T) {
	hashA := strings.Repeat("a", 64)
	hashB := strings.Repeat("b", 64)
	c := CredentialCommitment{
		HolderID:       "H-0001",
		CredentialType: "BSc Computer Science & AI <Hons>",
		IssuedAt:       "2026-09-24T06:15:30Z",
		IssuerOrgID:    "GeneralMSP",
		ExpiryDate:     "2030-06-30",
		CID:            "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG",
		FieldHashes:    map[string]string{"GPA": hashA, "College": hashB},
	}
	want := `{"cid":"QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG",` +
		`"credentialType":"BSc Computer Science & AI <Hons>",` +
		`"expiryDate":"2030-06-30",` +
		`"fieldHashes":{"College":"` + hashB + `","GPA":"` + hashA + `"},` +
		`"holderID":"H-0001","issuedAt":"2026-09-24T06:15:30Z","issuerOrgID":"GeneralMSP","v":2}`
	got, err := commitmentCanonicalJSON(c)
	if err != nil || got != want {
		t.Fatalf("canonical commitment differs:\n got  %s\n want %s", got, want)
	}
	h, _ := commitmentHash(c)
	if h != sha3Hex(want) || len(h) != 64 {
		t.Fatalf("commitmentHash %s", h)
	}
}

func TestPayloadIsCanonical(t *testing.T) {
	raw, err := BuildDisclosedPayload("CRED-0007",
		map[string]string{"field02": "B", "field01": "A"},
		map[string]string{"field02": "s2", "field01": "s1"},
		"2026-10-10T11:15:00.123456")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"credentialID":"CRED-0007","disclosedFields":{"field01":"A","field02":"B"},` +
		`"salts":{"field01":"s1","field02":"s2"},"timestamp":"2026-10-10T11:15:00.123456"}`
	if raw != want {
		t.Fatalf("payload\n got  %s\n want %s", raw, want)
	}
}

func TestWalletTimestampMatchesDart(t *testing.T) {
	loc := time.FixedZone("UAE", 4*3600)
	if got := WalletTimestamp(time.Date(2026, 10, 10, 11, 15, 0, 123456000, loc)); got != "2026-10-10T11:15:00.123456" {
		t.Fatalf("got %s", got)
	}
	// Dart drops the microsecond digits when they are zero.
	if got := WalletTimestamp(time.Date(2026, 10, 10, 11, 15, 0, 123000000, loc)); got != "2026-10-10T11:15:00.123" {
		t.Fatalf("got %s", got)
	}
}

func TestMessagesAndCID(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	hexRe := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for i := 0; i < 50; i++ {
		m, err := IssuerMessage(rng, 10)
		if err != nil || !hexRe.MatchString(m) {
			t.Fatalf("issuer message %q %v", m, err)
		}
		h, err := HolderMessage(rng, 10, 5)
		if err != nil || !hexRe.MatchString(h) {
			t.Fatalf("holder message %q %v", h, err)
		}
	}
	cid := FakeCIDv0([]byte("x"))
	// Production's CID pattern (offchain/crypto.go:92 @ 9026aba) and the CIDv0 length.
	cidPattern := regexp.MustCompile(`^(Qm[1-9A-HJ-NP-Za-km-z]{44}|b[a-z2-7]{20,100})$`)
	if len(cid) != 46 || !cidPattern.MatchString(cid) || !strings.HasPrefix(cid, "Qm") {
		t.Fatalf("fake CID %q", cid)
	}
}

func TestEffectiveStatusUsesDubaiCalendar(t *testing.T) {
	c := &ChainCredential{Status: "active", ExpiryDate: "2026-10-10"}
	// 2026-10-10T21:30Z is already 2026-10-11 in Dubai (UTC+4): expired.
	if s := effectiveStatus(c, time.Date(2026, 10, 10, 21, 30, 0, 0, time.UTC)); s != "expired" {
		t.Fatalf("status %s", s)
	}
}

// TestProdFunctionsRoundTrip runs the copied production functions (and the
// signature key-generation mirror) end to end for a non-default algorithm,
// so the parameterisation through sigName and kemName is exercised.
func TestProdFunctionsRoundTrip(t *testing.T) {
	SetSignatureAlgorithm("ML-DSA-65")
	SetKEMAlgorithm("ML-KEM-1024")
	defer SetSignatureAlgorithm("ML-DSA-44")
	defer SetKEMAlgorithm("ML-KEM-768")
	pkHex, skHex, err := ProdSigKeygen()
	if err != nil || len(pkHex) != 2*1952 || len(skHex) != 2*4032 {
		t.Fatalf("ML-DSA-65 key pair: %d/%d hex characters (%v)", len(pkHex), len(skHex), err)
	}
	msg := sha3Hex("payload")
	sigHex, err := ProdSign(msg, skHex)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := ProdVerify(msg, sigHex, pkHex); err != nil || !ok {
		t.Fatalf("production verify rejected its own signature (%v)", err)
	}
	if ok, _ := ProdVerify(sha3Hex("other"), sigHex, pkHex); ok {
		t.Fatal("production verify accepted a signature over another message")
	}
	kpk, ksk, err := ProdKEMKeygen()
	if err != nil || len(kpk) != 2*1568 {
		t.Fatalf("ML-KEM-1024 key pair: %d hex characters (%v)", len(kpk), err)
	}
	ctHex, ssHex, err := ProdKEMEncap(kpk)
	if err != nil {
		t.Fatal(err)
	}
	ss2, err := ProdKEMDecap(ctHex, ksk)
	if err != nil || ss2 != ssHex {
		t.Fatalf("production KEM round trip failed (%v)", err)
	}
}
