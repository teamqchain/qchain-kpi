// Code copied VERBATIM from the QChain keygen tool (main repo, commit 9026aba).
// Source: offchain/cmd/keygen/main.go lines 36-133 (decryptEnvelope, the Go mirror of
// QWallet's CryptoService.decryptEnvelope). Only the package clause changed (main -> qchain),
// the import block lists only what it uses, and ONE marked edit: the KEM decapsulation step
// is routed to the OpenSSL twin for classical KEMs (decision A); liboqs KEMs run the original
// lines, of which only `ss, err :=` became `ss, err =`.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/open-quantum-safe/liboqs-go/oqs"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

// decryptEnvelope opens a credential envelope as the holder. It is written
// independently of the server's envelope.go, mirroring QWallet's
// CryptoService.decryptEnvelope step by step, so the e2e test cross-checks the
// server's sealing against a second implementation.
func decryptEnvelope(envJSON []byte, kemPrivHex string) (map[string]any, error) {
	var env struct {
		KemAlg string `json:"kemAlg"`
		CredID string `json:"credId"`
		Wraps  []struct {
			Recipient string `json:"recipient"`
			KemCt     string `json:"kemCt"`
		} `json:"wraps"`
		Fields []struct {
			Key   string            `json:"key"`
			Nonce string            `json:"nonce"`
			Ct    string            `json:"ct"`
			Wrap  map[string]string `json:"wrap"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(envJSON, &env); err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}
	var kemCt []byte
	for _, w := range env.Wraps {
		if w.Recipient == "holder" {
			b, err := hex.DecodeString(w.KemCt)
			if err != nil {
				return nil, fmt.Errorf("decode kemCt: %w", err)
			}
			kemCt = b
		}
	}
	if kemCt == nil {
		return nil, fmt.Errorf("envelope has no holder wrap")
	}
	priv, err := hex.DecodeString(kemPrivHex)
	if err != nil {
		return nil, fmt.Errorf("decode KEM private key: %w", err)
	}
	// HARNESS EDIT (decision A, 11 Oct): classical KEMs (DHKEM) go to the
	// OpenSSL twin; liboqs KEMs run the original lines below unchanged.
	var ss []byte
	if c := classicalKEMFor(env.KemAlg); c != nil {
		if ss, err = c.decapRaw(priv, kemCt); err != nil {
			return nil, fmt.Errorf("KEM decapsulate: %w", err)
		}
	} else {
	kem := oqs.KeyEncapsulation{}
	defer kem.Clean()
	if err := kem.Init(env.KemAlg, priv); err != nil {
		return nil, fmt.Errorf("init %s: %w", env.KemAlg, err)
	}
	ss, err = kem.DecapSecret(kemCt)
	if err != nil {
		return nil, fmt.Errorf("KEM decapsulate: %w", err)
	}
	} // end of HARNESS EDIT

	gcmFor := func(key []byte) (cipher.AEAD, error) {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	}
	out := map[string]any{}
	for _, f := range env.Fields {
		// Key-wrapping key: HKDF-SHA3-256(ss, info="qchain/trackB/v1|<credId>|<key>").
		kwk := make([]byte, 32)
		info := "qchain/trackB/v1|" + env.CredID + "|" + f.Key
		if _, err := io.ReadFull(hkdf.New(sha3.New256, ss, nil, []byte(info)), kwk); err != nil {
			return nil, err
		}
		wrapped, err := hex.DecodeString(f.Wrap["holder"])
		if err != nil {
			return nil, fmt.Errorf("field %q: decode wrap: %w", f.Key, err)
		}
		kwkAEAD, err := gcmFor(kwk)
		if err != nil {
			return nil, err
		}
		dataKey, err := kwkAEAD.Open(nil, make([]byte, 12), wrapped, nil)
		if err != nil {
			return nil, fmt.Errorf("field %q: unwrap data key: %w", f.Key, err)
		}
		nonce, err1 := hex.DecodeString(f.Nonce)
		ct, err2 := hex.DecodeString(f.Ct)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("field %q: bad nonce/ciphertext hex", f.Key)
		}
		dataAEAD, err := gcmFor(dataKey)
		if err != nil {
			return nil, err
		}
		plain, err := dataAEAD.Open(nil, nonce, ct, nil)
		if err != nil {
			return nil, fmt.Errorf("field %q: decrypt: %w", f.Key, err)
		}
		var v any
		if err := json.Unmarshal(plain, &v); err != nil {
			v = string(plain)
		}
		out[f.Key] = v
	}
	return out, nil
}
