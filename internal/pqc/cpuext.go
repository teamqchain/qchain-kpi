package pqc

/*
#cgo pkg-config: liboqs-go
#include <oqs/oqs.h>
*/
import "C"

// CPUExtensions reports liboqs's runtime CPU-feature detection (E0). With
// OQS_DIST_BUILD=OFF (variants b2/b3) liboqs always reports false here; the
// compiled-in implementations are then documented by oqsconfig.h instead.
func CPUExtensions() map[string]bool {
	exts := map[string]C.OQS_CPU_EXT{
		"ADX": C.OQS_CPU_EXT_ADX, "AES": C.OQS_CPU_EXT_AES, "AVX": C.OQS_CPU_EXT_AVX,
		"AVX2": C.OQS_CPU_EXT_AVX2, "AVX512": C.OQS_CPU_EXT_AVX512, "BMI1": C.OQS_CPU_EXT_BMI1,
		"BMI2": C.OQS_CPU_EXT_BMI2, "PCLMULQDQ": C.OQS_CPU_EXT_PCLMULQDQ,
		"VPCLMULQDQ": C.OQS_CPU_EXT_VPCLMULQDQ, "POPCNT": C.OQS_CPU_EXT_POPCNT,
		"SSE": C.OQS_CPU_EXT_SSE, "SSE2": C.OQS_CPU_EXT_SSE2, "SSE3": C.OQS_CPU_EXT_SSE3,
		"ARM_AES": C.OQS_CPU_EXT_ARM_AES, "ARM_SHA2": C.OQS_CPU_EXT_ARM_SHA2,
		"ARM_SHA3": C.OQS_CPU_EXT_ARM_SHA3, "ARM_NEON": C.OQS_CPU_EXT_ARM_NEON,
	}
	out := make(map[string]bool, len(exts))
	for name, e := range exts {
		out[name] = C.OQS_CPU_has_extension(e) == 1
	}
	return out
}
