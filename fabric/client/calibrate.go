package main

// Payload calibration: the payload length that makes a committed transaction's
// envelope as large as E13's envelope of QChain's issueCredential for one
// signature scheme. The payload travels as hex text twice (in the arguments
// and in the stored record), so the envelope grows by about 4 bytes per
// payload byte; the exact relation is fitted from two probe transactions and
// then refined with real transactions until the envelope is within 0.5 %.

import (
	"fmt"
	"math"
)

// CalStep is one calibration transaction.
type CalStep struct {
	Payload  int `json:"payload_bytes"`
	Envelope int `json:"envelope_bytes"`
}

// Calibration is the result for one point.
type Calibration struct {
	Point    string    `json:"point"`
	Target   int       `json:"target_envelope"` // 0: the smallest transaction
	Payload  int       `json:"payload_bytes"`
	Envelope int       `json:"envelope_bytes"` // the envelope the final payload gave
	ErrorPct float64   `json:"error_pct"`      // (envelope - target) / target, percent
	Slope    float64   `json:"slope_bytes_per_payload_byte,omitempty"`
	Steps    []CalStep `json:"steps"`
}

const (
	calTolPct  = 0.5 // accept within 0.5 %
	calFailPct = 1.0 // refuse a calibration that ends worse than 1 %
	calProbeLo = 1000
	calProbeHi = 8000
	calRounds  = 4
)

// Calibrate finds the payload for target. measure submits a transaction with
// a payload of p bytes and returns its envelope size.
func Calibrate(point string, target int, measure func(p int) (int, error)) (Calibration, error) {
	c := Calibration{Point: point, Target: target}
	step := func(p int) (int, error) {
		e, err := measure(p)
		if err == nil {
			c.Steps = append(c.Steps, CalStep{Payload: p, Envelope: e})
		}
		return e, err
	}
	if target == 0 { // the smallest transaction: a 1-byte payload
		e, err := step(1)
		if err != nil {
			return c, err
		}
		c.Payload, c.Envelope = 1, e
		return c, nil
	}
	e0, err := step(calProbeLo)
	if err != nil {
		return c, err
	}
	e1, err := step(calProbeHi)
	if err != nil {
		return c, err
	}
	b := float64(e1-e0) / float64(calProbeHi-calProbeLo)
	if b <= 0 {
		return c, fmt.Errorf("calibration of %s: the envelope did not grow with the payload (%d, %d)", point, e0, e1)
	}
	a := float64(e0) - b*calProbeLo
	c.Slope = b
	p := int(math.Round((float64(target) - a) / b))
	if p < 1 {
		return c, fmt.Errorf("calibration of %s: target %d is below the smallest envelope (about %.0f)", point, target, a+b)
	}
	for round := 0; round < calRounds; round++ {
		e, err := step(p)
		if err != nil {
			return c, err
		}
		c.Payload, c.Envelope = p, e
		c.ErrorPct = 100 * float64(e-target) / float64(target)
		if math.Abs(c.ErrorPct) <= calTolPct {
			return c, nil
		}
		adj := int(math.Round(float64(target-e) / b))
		if adj == 0 {
			adj = 1
			if e > target {
				adj = -1
			}
		}
		if p+adj < 1 {
			break
		}
		p += adj
	}
	if math.Abs(c.ErrorPct) > calFailPct {
		return c, fmt.Errorf("calibration of %s: envelope %d is %.2f%% from the target %d after %d rounds", point, c.Envelope, c.ErrorPct, target, calRounds)
	}
	return c, nil
}
