package proto

import "qchain-kpi/internal/algs"

// E8 measures sizes, not time. Its block for an operation produces many
// outputs (keys, signatures, ciphertexts) in one process and records the
// length of every item; the "arms" are the items:
//   - keygen: pk, sk;
//   - sign:   sig;
//   - encaps: ct, ss.
//
// Most lengths are fixed; unpadded Falcon and ECDSA (DER) signatures and RSA
// PKCS#8 private keys vary, and E8 records their real distribution.
const (
	ArmPK  = "pk"
	ArmSK  = "sk"
	ArmSig = "sig"
	ArmCT  = "ct"
	ArmSS  = "ss"
)

// E8Arms returns the size items of an E8 block, in output order.
func E8Arms(a algs.Alg, op string) []string {
	switch op {
	case "keygen":
		return []string{ArmPK, ArmSK}
	case "sign":
		return []string{ArmSig}
	case "encaps":
		return []string{ArmCT, ArmSS}
	}
	return nil
}

// SizeOnly reports whether an experiment measures sizes only. Its blocks are
// not timed: the run-quality flags do not apply to them, they are never
// repeated, and their first attempt is the result.
func SizeOnly(exp string) bool { return exp == "E8" }
