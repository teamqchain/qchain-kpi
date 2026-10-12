// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/verification.go lines 33-221 (failure reasons, DisclosedPayload, stringMap, parseDisclosedPayload, VerifyInput, VerifyOutcome, verifyPresentation, presentationRequest, validatePresentationRequest); the verification-log handlers are not needed.
// Only the package clause changed (main -> qchain) and: the import block lists only what these use.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Failure reasons: returned as `reason` by /resolveSession and stored in
// verification_logs.failure_reason. The portal maps each of them.
const (
	reasonNotFound               = "NOT_FOUND"
	reasonRevoked                = "REVOKED"
	reasonSuspended              = "SUSPENDED"
	reasonExpired                = "EXPIRED"
	reasonSignatureInvalid       = "SIGNATURE_INVALID"
	reasonHashMismatch           = "HASH_MISMATCH"
	reasonFieldHashesInvalid     = "FIELD_HASHES_INVALID"
	reasonHolderSignatureInvalid = "HOLDER_SIGNATURE_INVALID"
)

// DisclosedPayload is the presentation the holder's wallet signs:
// {"credentialID","disclosedFields":{key:value},"salts":{key:salt},"timestamp"}.
type DisclosedPayload struct {
	CredentialID    string
	DisclosedFields map[string]string
	Salts           map[string]string
	Timestamp       string
}

// stringMap decodes a JSON object whose values must all be strings.
func stringMap(name string, raw map[string]json.RawMessage) (map[string]string, error) {
	out := make(map[string]string, len(raw))
	for _, key := range sortedKeys(raw) {
		var v string
		if err := json.Unmarshal(raw[key], &v); err != nil || bytes.Equal(bytes.TrimSpace(raw[key]), []byte("null")) {
			return nil, fmt.Errorf("%s[%q] must be a string", name, key)
		}
		out[key] = v
	}
	return out, nil
}

// parseDisclosedPayload parses a signed presentation. Values in
// disclosedFields and salts must be strings (every issued value is one).
func parseDisclosedPayload(raw string) (DisclosedPayload, error) {
	var loose struct {
		CredentialID    string                     `json:"credentialID"`
		DisclosedFields map[string]json.RawMessage `json:"disclosedFields"`
		Salts           map[string]json.RawMessage `json:"salts"`
		Timestamp       string                     `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(raw), &loose); err != nil {
		return DisclosedPayload{}, fmt.Errorf("invalid JSON in disclosedPayload: %v", err)
	}
	fields, err := stringMap("disclosedFields", loose.DisclosedFields)
	if err != nil {
		return DisclosedPayload{}, err
	}
	salts, err := stringMap("salts", loose.Salts)
	if err != nil {
		return DisclosedPayload{}, err
	}
	return DisclosedPayload{
		CredentialID:    loose.CredentialID,
		DisclosedFields: fields,
		Salts:           salts,
		Timestamp:       loose.Timestamp,
	}, nil
}

// VerifyInput is everything verifyPresentation needs, already fetched.
type VerifyInput struct {
	DisplayID        string           // credential ID the session was created for (CRED-0001)
	FabricID         string           // its on-chain key
	Cred             *ChainCredential // nil when the credential is not on-chain
	Holder           *ChainHolder     // the credential's on-chain holder (full view)
	RawPayload       string           // disclosedPayload exactly as the wallet signed it
	HolderSignature  string           // hex ML-DSA-44 signature over sha3Hex(RawPayload)
	TrustedIssuerKey string           // ISSUER_PUBLIC_KEY_HEX from .env
	Now              time.Time
}

// VerifyOutcome is the result of every check plus the overall verdict.
type VerifyOutcome struct {
	Verified             bool
	Reason               string // "" when verified
	Status               string // effective status (active/revoked/suspended/expired)
	ExistsOnChain        bool
	NotRevoked           bool
	SignatureValid       bool
	HashMatches          bool
	FieldHashesValid     bool
	HolderSignatureValid bool
	Disclosed            map[string]string // disclosed fields (no salts); nil if unparseable
}

// verifyPresentation runs every check. All checks run even after one fails,
// so the portal can show each result; Reason reports the most important
// failure: NOT_FOUND > REVOKED > SUSPENDED > EXPIRED > SIGNATURE_INVALID >
// HASH_MISMATCH > FIELD_HASHES_INVALID > HOLDER_SIGNATURE_INVALID.
func verifyPresentation(in VerifyInput) VerifyOutcome {
	var out VerifyOutcome
	payload, payloadErr := parseDisclosedPayload(in.RawPayload)
	if payloadErr == nil {
		out.Disclosed = payload.DisclosedFields
	}
	if in.Cred == nil {
		out.Reason = reasonNotFound
		return out
	}
	cred := in.Cred
	out.ExistsOnChain = true
	out.Status = effectiveStatus(cred, in.Now)
	out.NotRevoked = cred.Status == "active"

	// Issuer signature over the commitment rebuilt from the chain — never the
	// stored hash — and only from the trusted issuer key.
	if hash, err := commitmentHash(commitmentFromChain(cred)); err == nil {
		out.HashMatches = hash == cred.CredentialHash
		if in.TrustedIssuerKey != "" && strings.EqualFold(cred.PublicKey, in.TrustedIssuerKey) {
			out.SignatureValid, _ = pqcVerify(hash, cred.Signature, cred.PublicKey)
		}
	}

	if payloadErr == nil {
		out.FieldHashesValid = verifyDisclosedFields(payload.DisclosedFields, payload.Salts, cred.FieldHashes) == nil

		payloadID := strings.TrimSpace(payload.CredentialID)
		boundToCredential := payloadID != "" && (payloadID == in.DisplayID || payloadID == in.FabricID)
		if boundToCredential && in.Holder != nil && in.Holder.DsaPublicKey != "" && in.HolderSignature != "" {
			out.HolderSignatureValid, _ = pqcVerify(sha3Hex(in.RawPayload), in.HolderSignature, in.Holder.DsaPublicKey)
		}
	}

	switch {
	case out.Status == "revoked":
		out.Reason = reasonRevoked
	case out.Status == "suspended":
		out.Reason = reasonSuspended
	case out.Status == "expired":
		out.Reason = reasonExpired
	case !out.SignatureValid:
		out.Reason = reasonSignatureInvalid
	case !out.HashMatches:
		out.Reason = reasonHashMismatch
	case !out.FieldHashesValid:
		out.Reason = reasonFieldHashesInvalid
	case !out.HolderSignatureValid:
		out.Reason = reasonHolderSignatureInvalid
	}
	out.Verified = out.Reason == ""
	return out
}

// presentationRequest is the body of /mobile/generateOTP and
// /mobile/generatePresentation.
type presentationRequest struct {
	CredentialID     string   `json:"credentialID"`
	HiddenFields     []string `json:"hiddenFields"`
	DisclosedPayload string   `json:"disclosedPayload"`
	HolderSignature  string   `json:"holderSignature"`
}

// validatePresentationRequest decodes and checks a presentation request before
// any database access. The returned error text is sent to the wallet as-is.
func validatePresentationRequest(r *http.Request) (presentationRequest, error) {
	var req presentationRequest
	if err := decodeBody(r, &req); err != nil {
		return req, errors.New("invalid JSON")
	}
	req.CredentialID = strings.TrimSpace(req.CredentialID)
	if req.CredentialID == "" {
		return req, errors.New("missing credentialID")
	}
	if strings.TrimSpace(req.DisclosedPayload) == "" {
		return req, errors.New("missing disclosedPayload")
	}
	if strings.TrimSpace(req.HolderSignature) == "" {
		return req, errors.New("missing holderSignature")
	}
	payload, err := parseDisclosedPayload(req.DisclosedPayload)
	if err != nil {
		return req, err
	}
	payloadCredID := strings.TrimSpace(payload.CredentialID)
	if payloadCredID == "" {
		return req, errors.New("disclosedPayload missing credentialID")
	}
	if payloadCredID != req.CredentialID {
		return req, fmt.Errorf("disclosedPayload credentialID %q does not match request credentialID %q", payloadCredID, req.CredentialID)
	}
	if err := validateDisclosureSalts(payload.DisclosedFields, payload.Salts); err != nil {
		return req, err
	}
	return req, nil
}
