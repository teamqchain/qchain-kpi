// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/mobile.go lines 33-37 (the expected hex lengths of the wallet's public keys).
// Only the package clause changed (main -> qchain) and (decision B, 11 Oct) `const` became `var`:
// production hard-codes ML-KEM-768 and ML-DSA-44; SetSuite sets the lengths of the measured suite.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain
// Expected hex lengths of the wallet's public keys.
var ( // decision B (11 Oct): const in production; set per suite by SetSuite
	kemPublicKeyHexLen = 2368 // ML-KEM-768 public key, 1184 bytes
	dsaPublicKeyHexLen = 2624 // ML-DSA-44 public key, 1312 bytes
)
