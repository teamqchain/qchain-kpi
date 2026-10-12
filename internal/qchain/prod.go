package qchain

// Harness code (not copied): exported entry points to the verbatim production
// functions, for E2's prod arm, plus one mirror.

import (
	"encoding/hex"
	"fmt"

	"github.com/open-quantum-safe/liboqs-go/oqs"
)

// SetKEMAlgorithm selects the liboqs KEM used by the copied kemGenerateKeypair,
// kemEncap and kemDecap (production uses ML-KEM-768). One process measures one
// algorithm, so a package-level setting is sufficient.
func SetKEMAlgorithm(name string) { kemName = name }

// ProdSign is production's pqcSign (offchain/crypto.go).
func ProdSign(message, privateKeyHex string) (string, error) {
	return pqcSignLiboqs(message, privateKeyHex)
}

// ProdVerify is production's pqcVerify (offchain/crypto.go).
func ProdVerify(message, signatureHex, publicKeyHex string) (bool, error) {
	return pqcVerifyLiboqs(message, signatureHex, publicKeyHex)
}

// ProdKEMKeygen is production's kemGenerateKeypair (offchain/kem.go).
func ProdKEMKeygen() (pubHex, secHex string, err error) { return kemGenerateKeypairLiboqs() }

// ProdKEMEncap is production's kemEncap (offchain/kem.go).
func ProdKEMEncap(pubHex string) (ctHex, ssHex string, err error) { return kemEncapLiboqs(pubHex) }

// ProdKEMDecap is production's kemDecap (offchain/kem.go).
func ProdKEMDecap(ctHex, secHex string) (string, error) { return kemDecapLiboqs(ctHex, secHex) }

// ProdSigKeygen MIRRORS production's signature key generation. Production has
// no function for it: the steps are inline in the keygen tool
// (offchain/cmd/keygen/main.go lines 162-174 @ 9026aba, the "dsa" case, and the
// same steps at lines 222-238). They are restated here in the same order:
// Init with no key, GenerateKeyPair, ExportSecretKey, hex-encode both keys,
// and the deferred Clean. Only the algorithm name is a parameter (sigName),
// and errors are returned instead of log.Fatalf.
func ProdSigKeygen() (pubHex, privHex string, err error) {
	signer := oqs.Signature{}
	defer signer.Clean()
	if err := signer.Init(sigName, nil); err != nil {
		return "", "", fmt.Errorf("init %s signer: %w", sigName, err)
	}
	pubKey, err := signer.GenerateKeyPair()
	if err != nil {
		return "", "", fmt.Errorf("generate key pair: %w", err)
	}
	privKey := signer.ExportSecretKey()
	return hex.EncodeToString(pubKey), hex.EncodeToString(privKey), nil
}
