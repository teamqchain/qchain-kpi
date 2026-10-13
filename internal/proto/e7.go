package proto

import (
	"qchain-kpi/internal/algs"
)

// E7: rejection time. One block per algorithm interleaves a valid call with
// invalid ones (coding plan §7): for a signature scheme, verify with a valid
// signature, with the same signature with one bit flipped ("badsig"), and
// with a valid signature over a different message of the same length
// ("wrongmsg"); for a KEM, decaps of a valid ciphertext and of the same
// ciphertext with one bit flipped ("tampered").

// E7 arm names.
const (
	ArmValid    = "valid"    // the operation on valid input (the baseline)
	ArmBadSig   = "badsig"   // verify: the signature with one bit flipped
	ArmWrongMsg = "wrongmsg" // verify: a valid signature over a different message of the same length
	ArmTampered = "tampered" // decaps: the ciphertext with one bit flipped
)

// E7Op is E7's operation for an algorithm: verify or decaps.
func E7Op(a algs.Alg) string {
	if a.Track == algs.Sig {
		return "verify"
	}
	return "decaps"
}

// E7Arms are an E7 block's arms, in build order.
func E7Arms(a algs.Alg) []string {
	if a.Track == algs.Sig {
		return []string{ArmValid, ArmBadSig, ArmWrongMsg}
	}
	return []string{ArmValid, ArmTampered}
}

// E7 outcome codes, recorded per sample in the raw file's bytes column.
const (
	E7Rejected      = 0 // verify returned false, or decaps returned a different secret (implicit rejection)
	E7RejectedError = 1 // the library returned an error (malformed input)
	E7Accepted      = 2 // verify returned true, or decaps returned the encapsulated secret
)
