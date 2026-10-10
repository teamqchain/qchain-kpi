// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/crypto.go lines 29-80 (sigName, pqcSign, pqcVerify, sha3Hex); the IPFS helpers are not needed.
// Only the package clause changed (main -> qchain) and: the import block lists only what these functions use, and sigName is a var instead of a const.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain

import (
	"encoding/hex"
	"fmt"

	"github.com/open-quantum-safe/liboqs-go/oqs"
	"golang.org/x/crypto/sha3"
)

// sigName is the post-quantum signature algorithm used everywhere in this server.
var sigName = "ML-DSA-44" // const in production; a var here so the harness can select the algorithm (SetSignatureAlgorithm)

// pqcSign signs `message` with the provided hex-encoded private key and returns
// a hex-encoded signature.
func pqcSign(message, privateKeyHex string) (string, error) {
	privKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("decoding private key hex: %w", err)
	}
	signer := oqs.Signature{}
	// defer runs Clean() when this function returns, freeing the C-side memory
	// liboqs allocated — no matter which return path we take.
	defer signer.Clean()
	if err := signer.Init(sigName, privKeyBytes); err != nil {
		return "", fmt.Errorf("init PQC signer with key: %w", err)
	}
	sig, err := signer.Sign([]byte(message))
	if err != nil {
		return "", fmt.Errorf("signing: %w", err)
	}
	return hex.EncodeToString(sig), nil
}

// pqcVerify verifies a hex-encoded signature against a message and hex-encoded
// public key. Verification needs no private key, so Init is called with nil.
func pqcVerify(message, signatureHex, publicKeyHex string) (bool, error) {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, fmt.Errorf("decoding public key hex: %w", err)
	}
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("decoding signature hex: %w", err)
	}
	verifier := oqs.Signature{}
	defer verifier.Clean()
	if err := verifier.Init(sigName, nil); err != nil {
		return false, fmt.Errorf("init PQC verifier: %w", err)
	}
	valid, err := verifier.Verify([]byte(message), sigBytes, pubKeyBytes)
	if err != nil {
		return false, fmt.Errorf("verify: %w", err)
	}
	return valid, nil
}

// sha3Hex returns the SHA3-256 hex digest of data.
func sha3Hex(data string) string {
	digest := sha3.Sum256([]byte(data))
	return hex.EncodeToString(digest[:])
}
