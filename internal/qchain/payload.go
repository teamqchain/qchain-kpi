package qchain

// Harness code (not copied): a Go mirror of the wallet's presentation payload,
// which production builds in Dart (UI_App/lib/services/crypto_service.dart,
// buildSignedDisclosedPayload, lines 619-673, main repo commit 9026aba).

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

// BuildDisclosedPayload returns the exact JSON the wallet signs:
// canonicalJsonEncode({credentialID, disclosedFields, salts, timestamp}), i.e.
// keys sorted at every level, no whitespace, no HTML escaping (Dart's
// jsonEncode never HTML-escapes). For the ASCII values the harness uses,
// Go's encoder with SetEscapeHTML(false) produces identical bytes.
func BuildDisclosedPayload(credentialID string, disclosed, salts map[string]string, timestamp string) (string, error) {
	payload := map[string]any{
		"credentialID":    credentialID,
		"disclosedFields": disclosed,
		"salts":           salts,
		"timestamp":       timestamp,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// WalletTimestamp formats t like Dart's DateTime.now().toIso8601String() for
// a local (non-UTC) time: "yyyy-MM-ddTHH:mm:ss.mmmuuu", where Dart omits the
// microsecond digits when they are zero.
func WalletTimestamp(t time.Time) string {
	if (t.Nanosecond()/1000)%1000 == 0 {
		return t.Format("2006-01-02T15:04:05.000")
	}
	return t.Format("2006-01-02T15:04:05.000000")
}

// PayloadHash is what the holder signs and the verifier recomputes:
// sha3Hex(rawPayload) (verification.go:156).
func PayloadHash(rawPayload string) string { return sha3Hex(rawPayload) }
