package ledger

// X.509 certificate sizes (DER and PEM), and the projection of a captured
// ECDSA P-256 certificate onto ML-DSA (E13's labelled projection scenario:
// Fabric's identities and signatures switched to ML-DSA, which Fabric v2.5
// does not support today).

func derLenLen(n int) int {
	switch {
	case n < 0x80:
		return 1
	case n < 0x100:
		return 2
	case n < 0x10000:
		return 3
	case n < 0x1000000:
		return 4
	}
	return 5
}

// tlv is the full size of a DER element with content length n.
func tlv(n int) int { return 1 + derLenLen(n) + n }

// contentOf is the content length of a DER element of full size full.
func contentOf(full int) int {
	for _, h := range []int{2, 3, 4, 5} {
		if tlv(full-h) == full {
			return full - h
		}
	}
	return -1
}

// PEMLen is the size of a PEM "CERTIFICATE" block for a DER certificate of
// der bytes: base64 in 64-character lines, each ending in a newline, between
// the BEGIN and END lines.
func PEMLen(der int) int {
	b := 4 * ((der + 2) / 3)
	return len("-----BEGIN CERTIFICATE-----\n") + b + (b+63)/64 + len("-----END CERTIFICATE-----\n")
}

// CertLayout is a certificate's DER layout as captured: the full sizes of the
// certificate, its TBSCertificate, the signature AlgorithmIdentifier (which
// also appears inside the TBSCertificate) and the SubjectPublicKeyInfo, and
// the length of the signature value.
type CertLayout struct {
	DER      int `json:"der"`
	TBS      int `json:"tbs"`
	SigAlg   int `json:"sigalg"`
	SigValue int `json:"sig_value"`
	SPKI     int `json:"spki"`
}

// Consistent reports whether the parts add up to the certificate
// (SEQUENCE{TBSCertificate, AlgorithmIdentifier, BIT STRING{0, signature}}).
func (c CertLayout) Consistent() bool {
	return tlv(c.TBS+c.SigAlg+tlv(c.SigValue+1)) == c.DER
}

// mldsaAlgID is an ML-DSA AlgorithmIdentifier: SEQUENCE{OID
// 2.16.840.1.101.3.4.3.17/18/19} with absent parameters (RFC 9881).
const mldsaAlgID = 2 + 2 + 9

// WithMLDSA is the same certificate (same names, validity and extensions)
// with an ML-DSA public key of pk bytes, signed by an ML-DSA CA (signature of
// sig bytes).
func (c CertLayout) WithMLDSA(pk, sig int) CertLayout {
	spki := tlv(mldsaAlgID + tlv(pk+1))
	tbs := tlv(contentOf(c.TBS) - c.SPKI - c.SigAlg + spki + mldsaAlgID)
	return CertLayout{
		DER:      tlv(tbs + mldsaAlgID + tlv(sig+1)),
		TBS:      tbs,
		SigAlg:   mldsaAlgID,
		SigValue: sig,
		SPKI:     spki,
	}
}
