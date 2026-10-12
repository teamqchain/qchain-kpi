// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/chain.go lines 93-106 (ChainSnapshot, holderOf): the shape of the chaincode's batch-read response that the backend decodes.
// Only the package clause changed (main -> qchain).
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain


// ChainSnapshot is the result of one batch credential read: the requested
// credentials (missing IDs are simply absent) plus the holders they reference.
type ChainSnapshot struct {
	Credentials map[string]*ChainCredential `json:"credentials"`
	Holders     map[string]*ChainHolder     `json:"holders"`
}

// holderOf returns the on-chain holder record of a credential (nil if absent).
func (s *ChainSnapshot) holderOf(c *ChainCredential) *ChainHolder {
	if s == nil || c == nil {
		return nil
	}
	return s.Holders[c.Holder]
}
