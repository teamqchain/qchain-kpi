package qchain

// Harness MIRRORS (not copies) of production steps that are written inline in
// HTTP handlers, so they cannot be called as functions. Each mirror restates
// the handler's lines in the same order and with the same calls, minus the
// I/O the study excludes (HTTP response writing, MySQL, IPFS, Fabric); every
// left-out line is named in the comment. docs/provenance.md lists them.

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// ActivationCheck MIRRORS handleRegisterHolderKeys (offchain/mobile.go lines
// 86-123 @ 9026aba): decode the request body, then the same checks in the
// same order, with the key lengths of the measured suite (decision B). Left
// out: the HTTP error responses (returned as errors here), the log line for a
// private key sent by mistake (none is sent), the MySQL holder lookup, the
// chain write and the MySQL key update.
func ActivationCheck(r *http.Request) error {
	var req struct {
		EmiratesID    string `json:"emiratesID"`
		KemPublicKey  string `json:"kemPublicKey"`
		DsaPublicKey  string `json:"dsaPublicKey"`
		KemPrivateKey string `json:"kemPrivateKey"`
		DsaPrivateKey string `json:"dsaPrivateKey"`
	}
	if err := decodeBody(r, &req); err != nil {
		return errors.New("invalid JSON body")
	}
	if req.KemPrivateKey != "" || req.DsaPrivateKey != "" {
		return errors.New("private key sent") // production logs a warning and continues; never happens here
	}
	if req.EmiratesID == "" {
		return errors.New("missing emiratesID")
	}
	if req.KemPublicKey == "" || req.DsaPublicKey == "" {
		return errors.New("missing kemPublicKey or dsaPublicKey")
	}
	if _, err := hex.DecodeString(req.KemPublicKey); err != nil {
		return errors.New("invalid hex for kemPublicKey: " + err.Error())
	}
	if _, err := hex.DecodeString(req.DsaPublicKey); err != nil {
		return errors.New("invalid hex for dsaPublicKey: " + err.Error())
	}
	if len(req.KemPublicKey) != kemPublicKeyHexLen {
		return fmt.Errorf("kemPublicKey must be %d hex characters", kemPublicKeyHexLen)
	}
	if len(req.DsaPublicKey) != dsaPublicKeyHexLen {
		return fmt.Errorf("dsaPublicKey must be %d hex characters", dsaPublicKeyHexLen)
	}
	return nil
}

// IssueResult is what the issuance mirror produces: the envelope (uploaded to
// IPFS in production), the commitment and its hash and signature, and the
// field-hashes JSON (the chain write's arguments).
type IssueResult struct {
	Envelope        []byte
	Salts           map[string]string
	Commitment      CredentialCommitment
	CredentialHash  string
	Signature       string
	FieldHashesJSON string
}

// Issue MIRRORS handleIssueCredential (offchain/credentials.go lines 140-209
// @ 9026aba): normalizeIssueAttributes, commitAttributes,
// sealCredentialEnvelope, the Dubai issuedAt, the commitment, signCommitment
// and json.Marshal of the field hashes, in that order. Left out: the request
// body decoding of the outer request (its only content is the info string
// passed here), the MySQL holder lookup and the chain read of the holder's
// KEM key (passed in), the IPFS upload (the CID it returns is passed in: IPFS
// computes it, not the backend), the chain write and the MySQL copy.
func Issue(info, holderID, credentialType, holderKemPubHex, cid, issuerOrgID, issuerPrivHex string) (*IssueResult, error) {
	attrs, expiry, err := normalizeIssueAttributes(info)
	if err != nil {
		return nil, fmt.Errorf("invalid info: %w", err)
	}
	salts, fieldHashes, err := commitAttributes(attrs)
	if err != nil {
		return nil, fmt.Errorf("field hashing failed: %w", err)
	}
	envelope, err := sealCredentialEnvelope(attrs, salts, holderKemPubHex)
	if err != nil {
		return nil, fmt.Errorf("off-chain encryption failed: %w", err)
	}
	issuedAtTime := time.Now().In(mustLoadLocation("Asia/Dubai")).Truncate(time.Second)
	commitment := CredentialCommitment{
		HolderID:       holderID,
		CredentialType: credentialType,
		IssuedAt:       issuedAtTime.Format(time.RFC3339),
		IssuerOrgID:    issuerOrgID,
		ExpiryDate:     expiry,
		CID:            cid,
		FieldHashes:    fieldHashes,
	}
	credentialHash, signature, err := signCommitment(commitment, issuerPrivHex)
	if err != nil {
		return nil, fmt.Errorf("PQC sign failed: %w", err)
	}
	fieldHashesJSON, err := json.Marshal(fieldHashes)
	if err != nil {
		return nil, fmt.Errorf("field hash marshal failed: %w", err)
	}
	return &IssueResult{Envelope: envelope, Salts: salts, Commitment: commitment, CredentialHash: credentialHash,
		Signature: signature, FieldHashesJSON: string(fieldHashesJSON)}, nil
}

// EnvelopeCheck is the backend's step of handleGetEnvelope (offchain/
// mobile.go line 178 @ 9026aba): looksLikeEnvelope over the bytes read from
// IPFS. Left out: the MySQL and chain reads, the IPFS read, the HTTP response.
func EnvelopeCheck(envelope []byte) bool { return looksLikeEnvelope(envelope) }

// WalletDecrypt is the holder opening the envelope: the keygen tool's
// decryptEnvelope, itself the Go mirror of QWallet's decryptEnvelope.
func WalletDecrypt(envelope []byte, kemPrivHex string) (map[string]any, error) {
	return decryptEnvelope(envelope, kemPrivHex)
}

// WalletKeygen MIRRORS the wallet's activation key generation
// (UI_App/lib/services/crypto_service.dart, generateKemKeyPair and
// generateSigningKeyPair @ 9026aba; the same steps as the keygen tool's kem
// and dsa cases): create the library object, generate, hex-encode both keys,
// dispose. It returns the KEM and the signature key pairs as hex.
func WalletKeygen() (kemPub, kemPriv, dsaPub, dsaPriv string, err error) {
	if kemPub, kemPriv, err = kemGenerateKeypair(); err != nil {
		return
	}
	dsaPub, dsaPriv, err = sigGenerateKeypair()
	return
}

// WalletSign MIRRORS the wallet's presentation signature
// (crypto_service.dart, buildSignedDisclosedPayload and signPayload @
// 9026aba): SHA3-256 of the canonical payload JSON as hex, then a fresh
// signer with the hex-decoded private key signs the UTF-8 of that hex, and the
// signature is hex-encoded. Those are the steps of production's pqcSign, which
// is therefore used for them.
func WalletSign(payloadJSON, dsaPrivHex string) (string, error) {
	return pqcSign(sha3Hex(payloadJSON), dsaPrivHex)
}

// PresentationRequest builds the body of /mobile/generatePresentation the
// wallet sends (harness: input preparation, not timed).
func PresentationRequest(credentialID string, hidden []string, payloadJSON, holderSig string) ([]byte, error) {
	if hidden == nil {
		hidden = []string{}
	}
	return json.Marshal(presentationRequest{CredentialID: credentialID, HiddenFields: hidden,
		DisclosedPayload: payloadJSON, HolderSignature: holderSig})
}

// PresentationCheck is the backend's step of handleGeneratePresentation
// (offchain/mobile.go line 483 @ 9026aba): validatePresentationRequest. Left
// out: the MySQL lookups, the session ID and insert, the HTTP response.
func PresentationCheck(r *http.Request) error {
	_, err := validatePresentationRequest(r)
	return err
}

// ChainResponse builds the chaincode's getCredentials response for one
// credential and its holder ("full" view), byte for byte as the chaincode
// returns it: records stored with sorted keys (sortKeysRecursive) and
// returned with JSON.stringify({credentials, holders}). Harness: input
// preparation, not timed.
func ChainResponse(cred *ChainCredential, holder *ChainHolder) ([]byte, error) {
	toMap := func(v any) (map[string]any, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		return m, json.Unmarshal(b, &m)
	}
	c, err := toMap(cred)
	if err != nil {
		return nil, err
	}
	h, err := toMap(holder)
	if err != nil {
		return nil, err
	}
	// Go marshals map keys sorted, which is the chaincode's stored order; the
	// outer object's keys ("credentials", "holders") are sorted too.
	var buf []byte
	buf, err = json.Marshal(map[string]any{
		"credentials": map[string]any{cred.ID: c},
		"holders":     map[string]any{holder.ID: h},
	})
	return buf, err
}

// DecodeChainResponse MIRRORS the decoding step of chainReadCredential /
// chainReadCredentials (offchain/chain.go lines 200-204 and 216-225 @
// 9026aba) for one ID: json.Unmarshal into ChainSnapshot, then the
// credential and its holder. Left out: the Fabric gateway call itself.
func DecodeChainResponse(res []byte, id string) (*ChainCredential, *ChainHolder, error) {
	var part ChainSnapshot
	if err := json.Unmarshal(res, &part); err != nil {
		return nil, nil, fmt.Errorf("decode getCredentials response: %w", err)
	}
	cred := part.Credentials[id]
	if cred == nil {
		return nil, nil, errors.New("not found on chain")
	}
	return cred, part.holderOf(cred), nil
}

// Verify is the backend's verification step of handleResolveSession
// (offchain/mobile.go lines 769-778 @ 9026aba): verifyPresentation with the
// trusted issuer key. Left out: the session and display-ID lookups (MySQL),
// the chain read (its decoding is DecodeChainResponse), logging and the HTTP
// response.
func Verify(displayID, fabricID string, cred *ChainCredential, holder *ChainHolder, rawPayload, holderSig, trustedIssuerKey string, now time.Time) VerifyOutcome {
	return verifyPresentation(VerifyInput{DisplayID: displayID, FabricID: fabricID, Cred: cred, Holder: holder,
		RawPayload: rawPayload, HolderSignature: holderSig, TrustedIssuerKey: trustedIssuerKey, Now: now})
}

// sortedFieldNames is a small helper for the tests and the flow driver.
func sortedFieldNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ------------------------------------------------- exported entry points (E11)

// IssuerKeygen generates the issuer's key pair as hex (once per block, untimed):
// the keygen tool's steps for liboqs, the OpenSSL twin for classical.
func IssuerKeygen() (pubHex, privHex string, err error) { return sigGenerateKeypair() }

// SigVerify is production's pqcVerify through the dispatcher (correctness
// checks, untimed).
func SigVerify(message, signatureHex, publicKeyHex string) (bool, error) {
	return pqcVerify(message, signatureHex, publicKeyHex)
}

// CommitAttributes is production's commitAttributes (input preparation).
func CommitAttributes(attrs map[string]string) (salts, fieldHashes map[string]string, err error) {
	return commitAttributes(attrs)
}

// SealEnvelope is production's sealCredentialEnvelope (input preparation for
// retrieval).
func SealEnvelope(attrs, salts map[string]string, holderKemPubHex string) ([]byte, error) {
	return sealCredentialEnvelope(attrs, salts, holderKemPubHex)
}
