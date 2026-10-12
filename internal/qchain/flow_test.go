package qchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"qchain-kpi/internal/algs"
)

// testInfo builds the portal's info JSON: n attributes field01... with
// 16-character values, plus an expiry date.
func testInfo(rng *rand.Rand, n int) (string, map[string]string) {
	attrs := map[string]string{}
	m := map[string]string{"expiryDate": "2030-06-30"}
	for i := 1; i <= n; i++ {
		v := RandomValue(rng, 16)
		attrs[FieldName(i)] = v
		m[FieldName(i)] = v
	}
	b, _ := json.Marshal(m)
	return string(b), attrs
}

func newRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestLifecycleEveryFamily runs activation, issuance, retrieval, presentation
// and verification through the copied production code for one suite of every
// algorithm family, and checks that tampering is caught.
func TestLifecycleEveryFamily(t *testing.T) {
	defer SetSuite(algs.MustByID("ml-dsa-44"), algs.MustByID("ml-kem-768"))
	suites := [][2]string{
		{"ml-dsa-44", "ml-kem-768"}, {"falcon-512", "ml-kem-512"}, {"slh-dsa-sha2-128f", "hqc-1"},
		{"ml-dsa-65", "hqc-3"}, {"ecdsa-p256", "dhkem-p256"}, {"ed25519", "dhkem-x25519"}, {"rsa3072-pss", "dhkem-p256"},
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for _, su := range suites {
		sig, kem := algs.MustByID(su[0]), algs.MustByID(su[1])
		if err := SetSuite(sig, kem); err != nil {
			t.Fatal(err)
		}
		// Issuer key (once, at setup in production).
		issuerPub, issuerPriv, err := sigGenerateKeypair()
		if err != nil {
			t.Fatalf("%s: issuer key: %v", su, err)
		}
		// Activation: wallet key generation, then the backend's checks.
		kemPub, kemPriv, dsaPub, dsaPriv, err := WalletKeygen()
		if err != nil {
			t.Fatalf("%s: wallet keygen: %v", su, err)
		}
		act, _ := json.Marshal(map[string]string{"emiratesID": "784-1990-1234567-1", "kemPublicKey": kemPub, "dsaPublicKey": dsaPub})
		if err := ActivationCheck(newRequest(t, act)); err != nil {
			t.Fatalf("%s: activation check: %v", su, err)
		}
		bad, _ := json.Marshal(map[string]string{"emiratesID": "x", "kemPublicKey": kemPub + "00", "dsaPublicKey": dsaPub})
		if ActivationCheck(newRequest(t, bad)) == nil {
			t.Fatalf("%s: a wrong-length key passed the activation check", su)
		}

		// Issuance.
		info, attrs := testInfo(rng, 10)
		cid := FakeCIDv0([]byte("cid-seed-" + su[0]))
		iss, err := Issue(info, "H-0001", "Bachelor of Science", kemPub, cid, "GeneralMSP", issuerPriv)
		if err != nil {
			t.Fatalf("%s: issue: %v", su, err)
		}
		if iss.Commitment.ExpiryDate != "2030-06-30" || len(iss.Commitment.FieldHashes) != 10 {
			t.Fatalf("%s: commitment %+v", su, iss.Commitment)
		}

		// Retrieval: backend check, wallet decryption.
		if !EnvelopeCheck(iss.Envelope) {
			t.Fatalf("%s: envelope check failed", su)
		}
		opened, err := WalletDecrypt(iss.Envelope, kemPriv)
		if err != nil {
			t.Fatalf("%s: wallet decrypt: %v", su, err)
		}
		for k, v := range attrs {
			if opened[k] != v {
				t.Fatalf("%s: decrypted %s = %v, want %q", su, k, opened[k], v)
			}
		}
		saltsAny, ok := opened[saltsFieldKey].(map[string]any)
		if !ok || len(saltsAny) != 10 {
			t.Fatalf("%s: decrypted salts %v", su, opened[saltsFieldKey])
		}
		// The server's own reference opener agrees.
		var env Envelope
		if err := json.Unmarshal(iss.Envelope, &env); err != nil {
			t.Fatal(err)
		}
		if _, err := openAttributes(&env, "holder", kemPriv); err != nil {
			t.Fatalf("%s: openAttributes: %v", su, err)
		}

		// The chain record and holder as the chaincode stores them.
		fabricID := fmt.Sprintf("CRED-%064x", rng.Uint64())
		cred := &ChainCredential{DocType: "credential", CommitmentVersion: 2, ID: fabricID, Holder: "H-0001",
			Issuer: "x509::CN=issuer1::CN=ca", IssuerOrgID: "GeneralMSP", Status: "active",
			CredentialType: iss.Commitment.CredentialType, IssuedAt: iss.Commitment.IssuedAt, ExpiryDate: iss.Commitment.ExpiryDate,
			CID: cid, FieldHashes: iss.Commitment.FieldHashes, CredentialHash: iss.CredentialHash, Signature: iss.Signature, PublicKey: issuerPub}
		holder := &ChainHolder{DocType: "holder", ID: "H-0001", FirstName: "A", LastName: "B", KemPublicKey: kemPub, DsaPublicKey: dsaPub}
		res, err := ChainResponse(cred, holder)
		if err != nil {
			t.Fatal(err)
		}

		// Presentation: disclose 5 of 10.
		disclosed, salts := map[string]string{}, map[string]string{}
		for i := 1; i <= 5; i++ {
			k := FieldName(i)
			disclosed[k] = attrs[k]
			salts[k] = saltsAny[k].(string)
		}
		payload, err := BuildDisclosedPayload("CRED-0001", disclosed, salts, WalletTimestamp(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		holderSig, err := WalletSign(payload, dsaPriv)
		if err != nil {
			t.Fatalf("%s: wallet sign: %v", su, err)
		}
		body, err := PresentationRequest("CRED-0001", nil, payload, holderSig)
		if err != nil {
			t.Fatal(err)
		}
		if err := PresentationCheck(newRequest(t, body)); err != nil {
			t.Fatalf("%s: presentation check: %v", su, err)
		}

		// Verification.
		c2, h2, err := DecodeChainResponse(res, fabricID)
		if err != nil {
			t.Fatal(err)
		}
		out := Verify("CRED-0001", fabricID, c2, h2, payload, holderSig, issuerPub, time.Now())
		if !out.Verified || !out.SignatureValid || !out.HashMatches || !out.FieldHashesValid || !out.HolderSignatureValid {
			t.Fatalf("%s: verification failed: %+v", su, out)
		}

		// Tampering is caught: a changed value, and a foreign holder signature.
		tampered := map[string]string{}
		for k, v := range disclosed {
			tampered[k] = v
		}
		tampered[FieldName(1)] = "X" + tampered[FieldName(1)][1:]
		tp, _ := BuildDisclosedPayload("CRED-0001", tampered, salts, WalletTimestamp(time.Now()))
		tsig, _ := WalletSign(tp, dsaPriv)
		if o := Verify("CRED-0001", fabricID, c2, h2, tp, tsig, issuerPub, time.Now()); o.Verified || o.Reason != reasonFieldHashesInvalid {
			t.Fatalf("%s: tampered value: %+v", su, o)
		}
		_, otherPriv, _ := sigGenerateKeypair()
		fsig, _ := WalletSign(payload, otherPriv)
		if o := Verify("CRED-0001", fabricID, c2, h2, payload, fsig, issuerPub, time.Now()); o.Verified || o.Reason != reasonHolderSignatureInvalid {
			t.Fatalf("%s: foreign holder signature: %+v", su, o)
		}
	}
}

// TestChainResponseMatchesChaincode: sorted keys at every level, and the
// backend's decoding returns the record unchanged.
func TestChainResponseMatchesChaincode(t *testing.T) {
	cred := &ChainCredential{DocType: "credential", CommitmentVersion: 2, ID: "CRED-1", Holder: "H-0001", Issuer: "x", IssuerOrgID: "GeneralMSP",
		Status: "active", CredentialType: "T", IssuedAt: "2026-10-12T10:00:00+04:00", ExpiryDate: "2030-06-30", CID: "Qm",
		FieldHashes: map[string]string{"field02": "b", "field01": "a"}, CredentialHash: "h", Signature: "s", PublicKey: "p"}
	holder := &ChainHolder{DocType: "holder", ID: "H-0001", FirstName: "A", LastName: "B", KemPublicKey: "k", DsaPublicKey: "d"}
	res, err := ChainResponse(cred, holder)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"credentials":{"CRED-1":{"CID":"Qm","CommitmentVersion":2,"CredentialHash":"h","CredentialType":"T","DocType":"credential",` +
		`"ExpiryDate":"2030-06-30","FieldHashes":{"field01":"a","field02":"b"},"Holder":"H-0001","ID":"CRED-1","IssuedAt":"2026-10-12T10:00:00+04:00",` +
		`"Issuer":"x","IssuerOrgID":"GeneralMSP","PublicKey":"p","Signature":"s","Status":"active"}},` +
		`"holders":{"H-0001":{"DocType":"holder","DsaPublicKey":"d","FirstName":"A","ID":"H-0001","KemPublicKey":"k","LastName":"B"}}}`
	if string(res) != want {
		t.Fatalf("chain response\n got %s\nwant %s", res, want)
	}
	c, h, err := DecodeChainResponse(res, "CRED-1")
	if err != nil || c.FieldHashes["field02"] != "b" || h.DsaPublicKey != "d" {
		t.Fatalf("decode: %+v %+v %v", c, h, err)
	}
}

// TestE2PathUnchanged: E2's prod arm still calls the copied liboqs bodies,
// not the dispatcher (its measured path did not change).
func TestE2PathUnchanged(t *testing.T) {
	defer SetSuite(algs.MustByID("ml-dsa-44"), algs.MustByID("ml-kem-768"))
	if err := SetSuite(algs.MustByID("ed25519"), algs.MustByID("dhkem-x25519")); err != nil {
		t.Fatal(err)
	}
	SetSignatureAlgorithm("ML-DSA-44")
	SetKEMAlgorithm("ML-KEM-768")
	pub, priv, err := ProdSigKeygen()
	if err != nil {
		t.Fatal(err)
	}
	sig, err := ProdSign("abc", priv) // liboqs body even though the dispatcher is set to Ed25519
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := ProdVerify("abc", sig, pub); err != nil || !ok {
		t.Fatalf("ProdVerify: %v %v", ok, err)
	}
}
