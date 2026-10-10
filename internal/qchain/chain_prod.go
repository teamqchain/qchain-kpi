// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/chain.go lines 42-76 (ChainCredential, ChainHolder) and 277-301 (dubaiDate, isExpired, effectiveStatus).
// Only the package clause changed (main -> qchain) and: the import block lists only what these use.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain

import "time"

type ChainCredential struct {
	DocType           string            `json:"DocType"`
	CommitmentVersion int               `json:"CommitmentVersion"`
	ID                string            `json:"ID"`
	Holder            string            `json:"Holder"`
	Issuer            string            `json:"Issuer"`
	IssuerOrgID       string            `json:"IssuerOrgID"`
	Status            string            `json:"Status"`
	CredentialType    string            `json:"CredentialType"`
	IssuedAt          string            `json:"IssuedAt"`
	ExpiryDate        string            `json:"ExpiryDate"`
	CID               string            `json:"CID"`
	FieldHashes       map[string]string `json:"FieldHashes,omitempty"`
	CredentialHash    string            `json:"CredentialHash,omitempty"`
	Signature         string            `json:"Signature,omitempty"`
	PublicKey         string            `json:"PublicKey,omitempty"`
	RevokedAt         string            `json:"RevokedAt,omitempty"`
	SuspendedAt       string            `json:"SuspendedAt,omitempty"`
	SuspendedReason   string            `json:"SuspendedReason,omitempty"`
	RestoredAt        string            `json:"RestoredAt,omitempty"`
	ExpiryUpdatedAt   string            `json:"ExpiryUpdatedAt,omitempty"`
}

// ChainHolder mirrors the chaincode's holder record. The summary view replaces
// the two public keys with the KemBound / DsaBound flags.
type ChainHolder struct {
	DocType      string `json:"DocType"`
	ID           string `json:"ID"`
	FirstName    string `json:"FirstName"`
	LastName     string `json:"LastName"`
	KemPublicKey string `json:"KemPublicKey,omitempty"`
	DsaPublicKey string `json:"DsaPublicKey,omitempty"`
	KemBound     bool   `json:"KemBound,omitempty"`
	DsaBound     bool   `json:"DsaBound,omitempty"`
}


// dubaiDate formats t as the UAE calendar date (YYYY-MM-DD).
func dubaiDate(t time.Time) string {
	return t.In(mustLoadLocation("Asia/Dubai")).Format("2006-01-02")
}

// isExpired reports whether a YYYY-MM-DD expiry has passed. A credential is
// valid through the END of its expiry day, UAE time.
func isExpired(expiry string, now time.Time) bool {
	return expiry != "" && dubaiDate(now) > expiry
}

// effectiveStatus is the status every API reports: the stored on-chain status,
// except that an active credential past its expiry is "expired". Precedence:
// revoked > suspended > expired > active.
func effectiveStatus(c *ChainCredential, now time.Time) string {
	switch c.Status {
	case "revoked", "suspended":
		return c.Status
	}
	if isExpired(c.ExpiryDate, now) {
		return "expired"
	}
	return c.Status
}
