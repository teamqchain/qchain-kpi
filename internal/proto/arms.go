package proto

import (
	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/plan"
)

// Experiment-independent views of blocks: their arms, and whether an experiment
// is a probe pass, measured, or sizes only.

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
	case "E7", "probe-E7":
		return E7Arms(a)
	case "E9":
		if _, arm, err := ParseE9Op(op); err == nil {
			return E9Quantities(arm)
		}
		return nil
	}
	return []string{ArmReuse}
}

// ArmsOfBlock is ArmsOf for any block, including E11 (whose algorithm field
// names a suite, not one algorithm).
func ArmsOfBlock(exp, alg, op string) []string {
	if exp == "E11" || exp == "probe-E11" {
		return E11Arms(op)
	}
	return ArmsOf(exp, algs.MustByID(alg), op)
}

// IsProbe reports whether an experiment is a probe pass (calibration only).
func IsProbe(exp string) bool { return plan.IsProbe(exp) }

// Measured reports whether an experiment's blocks are measurements that the
// drift rule and the headline selection apply to.
func Measured(exp string) bool {
	return exp == "E1" || exp == "E10" || exp == "E2" || exp == "E2c" || exp == "E11" || exp == "E6" || exp == "E7"
}

// SizeOnly reports whether an experiment measures sizes only (E8, E13):
// its blocks prepare their own inputs.
func SizeOnly(exp string) bool { return exp == "E8" || exp == "E13" }

// Untimed reports whether an experiment's blocks are not timed (sizes, and
// E9's memory): they need no fast/slow classification or batching, the
// run-quality flags do not apply to them, they are never repeated, and their
// first attempt is the result.
func Untimed(exp string) bool { return SizeOnly(exp) || exp == "E9" }
