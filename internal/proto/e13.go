package proto

import (
	"strings"

	"qchain-kpi/internal/algs"
)

// E13: ledger and IPFS bytes (model in internal/ledger).

// E13Op is E13's single operation of an algorithm: "ledger" for a signature
// scheme (Fabric transactions and blocks), "ipfs" for a KEM (envelopes).
func E13Op(a algs.Alg) string {
	if a.Track == algs.Sig {
		return "ledger"
	}
	return "ipfs"
}

// E13SigVariable reports whether a scheme's signature length varies (E8):
// unpadded Falcon and ECDSA (DER). E13 samples their signatures; kpi-prim
// checks this against the library.
func E13SigVariable(a algs.Alg) bool {
	return (a.Family == "Falcon" && !strings.Contains(a.ID, "padded")) || a.ID == "ecdsa-p256"
}
