package proto

// E10: ML-DSA issuer/holder equivalence: three interleaved sign arms, each
// with its own message pool.
const (
	ArmIssuer      = "issuer"       // E10: commitment hash, issuer key
	ArmHolder      = "holder"       // E10: presentation payload hash, holder key
	ArmIssuerAgain = "issuer-again" // E10: second commitment pool, issuer key (A/A)
)
