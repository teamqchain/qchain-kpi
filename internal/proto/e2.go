package proto

import (
	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
)

// E2 arm names (coding plan §4 and §7).
//
// An E2 block measures one operation three ways, interleaved in one process:
//   - reuse:  the library's long-lived object (the same code as E1);
//   - reinit: a fresh library object for every call (create, operate,
//     destroy), the way QChain's backend calls liboqs;
//   - prod:   QChain's production function itself (hex in, hex out), copied
//     verbatim from the backend for the post-quantum algorithms. The classical
//     algorithms have no production code; their prod arm applies the same
//     pattern (hex-decode, fresh object, operate, hex-encode) to OpenSSL.
//
// An E2c block measures the pieces that separate prod from reuse, each on its
// own: hex decoding of the inputs, creating the library object ("init": liboqs
// Init, or loading the key into an OpenSSL key object), destroying it
// ("clean": liboqs Clean, or freeing the OpenSSL key object) and hex encoding
// of the outputs. These are short, so they are batched where needed.
const (
	ArmReinit = "reinit"
	ArmProd   = "prod"

	ArmHexDecodeSK  = "hex-decode-sk"
	ArmHexDecodePK  = "hex-decode-pk"
	ArmHexDecodeSig = "hex-decode-sig"
	ArmHexDecodeCT  = "hex-decode-ct"
	ArmHexEncodePK  = "hex-encode-pk"
	ArmHexEncodeSK  = "hex-encode-sk"
	ArmHexEncodeSig = "hex-encode-sig"
	ArmHexEncodeCT  = "hex-encode-ct"
	ArmHexEncodeSS  = "hex-encode-ss"
	ArmInit         = "init"
	ArmClean        = "clean"
)

// E2Arms returns the arms of an E2 block, in the order kpi-prim builds them.
// A classical key generation already creates a fresh key object on every
// call, so its reuse and reinit arms would be the same code; it has no reinit
// arm.
func E2Arms(a algs.Alg, op string) []string {
	if a.Classical() && op == "keygen" {
		return []string{ArmReuse, ArmProd}
	}
	return []string{ArmReuse, ArmReinit, ArmProd}
}

// E2cArms returns the component arms of an E2c block, in build order. They are
// the steps the prod arm adds to the bare operation:
//   - keygen: create/destroy the object (post-quantum only), hex-encode both keys;
//   - sign:   hex-decode the secret key (ECDSA also its public point, which
//     OpenSSL imports together with the scalar), create/destroy the
//     key object, hex-encode the signature;
//   - verify: hex-decode the public key and the signature, create/destroy;
//   - encaps: hex-decode the public key, create/destroy, hex-encode the
//     ciphertext and the shared secret;
//   - decaps: hex-decode the ciphertext and the secret key (DHKEM also the
//     public key, which RFC 9180 decapsulation needs), create/destroy,
//     hex-encode the shared secret.
func E2cArms(a algs.Alg, op string) []string {
	switch op {
	case "keygen":
		if a.Classical() {
			return []string{ArmHexEncodePK, ArmHexEncodeSK}
		}
		return []string{ArmInit, ArmClean, ArmHexEncodePK, ArmHexEncodeSK}
	case "sign":
		if a.ID == "ecdsa-p256" {
			return []string{ArmHexDecodeSK, ArmHexDecodePK, ArmInit, ArmClean, ArmHexEncodeSig}
		}
		return []string{ArmHexDecodeSK, ArmInit, ArmClean, ArmHexEncodeSig}
	case "verify":
		return []string{ArmHexDecodePK, ArmHexDecodeSig, ArmInit, ArmClean}
	case "encaps":
		return []string{ArmHexDecodePK, ArmInit, ArmClean, ArmHexEncodeCT, ArmHexEncodeSS}
	case "decaps":
		if a.Classical() {
			return []string{ArmHexDecodeCT, ArmHexDecodeSK, ArmHexDecodePK, ArmInit, ArmClean, ArmHexEncodeSS}
		}
		return []string{ArmHexDecodeCT, ArmHexDecodeSK, ArmInit, ArmClean, ArmHexEncodeSS}
	}
	return nil
}

// ArmsOf returns the arm names of a block of any experiment (the probe-E2
// pass probes every E2 and E2c arm of the operation).
func ArmsOf(exp string, a algs.Alg, op string) []string {
	switch exp {
	case "E10":
		return []string{ArmIssuer, ArmHolder, ArmIssuerAgain}
	case "E2":
		return E2Arms(a, op)
	case "E2c":
		return E2cArms(a, op)
	case "probe-E2":
		return append(E2Arms(a, op), E2cArms(a, op)...)
	case "E8":
		return E8Arms(a, op)
	}
	return []string{ArmReuse}
}

// IsProbe reports whether an experiment is a probe pass (calibration only).
func IsProbe(exp string) bool { return plan.IsProbe(exp) }

// Measured reports whether an experiment's blocks are measurements that the
// drift rule and the headline selection apply to.
func Measured(exp string) bool { return exp == "E1" || exp == "E10" || exp == "E2" || exp == "E2c" }
