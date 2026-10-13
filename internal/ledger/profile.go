package ledger

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// CertIdentity is a Fabric identity with its certificate's DER layout.
type CertIdentity struct {
	Identity
	Cert CertLayout
}

// Profile holds the Fabric constants of QChain's network, as measured in the
// capture (DeriveProfile proves Measured equals what the capture shows).
type Profile struct {
	ChannelHeader  int            // ChannelHeader message, median of the QChain transactions
	Nonce          int            // signature-header nonce
	ProposalHash   int            // SHA-256
	Creator        CertIdentity   // the issuer's enrolled client identity (submits every QChain transaction)
	Endorsers      []CertIdentity // peer0 of GeneralMSP and of GovernmentMSP, in endorsement order
	Orderer        CertIdentity   // orderer0, signs every block
	FabricSig      int            // ECDSA P-256 signature (DER), median over client, endorsement and block signatures
	CCNameLen      int            // "qchaincode"
	CCIDAction     int            // ChaincodeID (name and version) in the chaincode action
	MD0Value       int            // OrdererBlockMetadata in the SIGNATURES metadata
	CommitHash     int            // COMMIT_HASH metadata entry
	HashLen        int            // block data hash and previous-block hash
	HolderFirst    int            // registerHolder first-name length, median
	HolderLast     int            // registerHolder last-name length, median
	SuspendReason  int            // suspendCredential reason length, median
	IssuerIdentity int            // the record's Issuer field (the submitter's Fabric ID string)
}

// Measured is QChain's network as captured on 13 Oct 2026.
var Measured = Profile{
	ChannelHeader: 109, Nonce: 24, ProposalHash: 32,
	Creator: CertIdentity{Identity{MSPLen: 10, PEM: 993}, CertLayout{DER: 691, TBS: 601, SigAlg: 12, SigValue: 71, SPKI: 91}},
	Endorsers: []CertIdentity{
		{Identity{MSPLen: 10, PEM: 1013}, CertLayout{DER: 708, TBS: 619, SigAlg: 12, SigValue: 70, SPKI: 91}},
		{Identity{MSPLen: 13, PEM: 1034}, CertLayout{DER: 723, TBS: 634, SigAlg: 12, SigValue: 70, SPKI: 91}},
	},
	Orderer:   CertIdentity{Identity{MSPLen: 10, PEM: 1062}, CertLayout{DER: 743, TBS: 654, SigAlg: 12, SigValue: 70, SPKI: 91}},
	FabricSig: 71, CCNameLen: 10, CCIDAction: 17, MD0Value: 13, CommitHash: 34, HashLen: 32,
	HolderFirst: 6, HolderLast: 7, SuspendReason: 24, IssuerIdentity: 116,
}

// IssuerPlaceholder stands in for a credential record's Issuer field (the
// submitter's Fabric client ID, which contains an enrolled user's name): the
// captured length and shape, with made-up values.
const IssuerPlaceholder = "x509::/OU=client/OU=issuer/OU=general/CN=issuer1::/C=AE/ST=Emirate/L=Capital/O=general.uae.com/CN=ca.general.uae.com"

// MaskDN replaces every attribute value of a Fabric ID string with x (the
// capture's privacy rule for the Issuer field).
func MaskDN(s string) string {
	b := []byte(s)
	in := false
	for i, c := range b {
		switch {
		case c == '=':
			in = true
		case c == ',' || c == '/' || c == ':' || c == '+':
			in = false
		case in:
			b[i] = 'x'
		}
	}
	return string(b)
}

// DeriveProfile computes the profile from the capture.
func DeriveProfile() (Profile, error) {
	c, err := loadCapture()
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	var chdr, sigs, first, last, reason []int
	one := func(name string, have *CertIdentity, id capIdentity) error {
		ci := CertIdentity{id.Identity, id.CertLayout}
		if have.PEM == 0 {
			*have = ci
		} else if !reflect.DeepEqual(*have, ci) {
			return fmt.Errorf("profile: %s identity varies", name)
		}
		return nil
	}
	for _, t := range c.Txs {
		if t.CC != "qchaincode" {
			continue
		}
		chdr = append(chdr, t.ChannelHeader)
		sigs = append(sigs, t.ClientSig)
		p.Nonce, p.ProposalHash, p.CCIDAction, p.CCNameLen = t.Nonce, t.ProposalHash, t.CAChaincodeID, 10
		if err := one("creator", &p.Creator, t.Creator); err != nil {
			return p, err
		}
		if p.Endorsers == nil {
			p.Endorsers = make([]CertIdentity, len(t.Endorsements))
		}
		for k, e := range t.Endorsements {
			sigs = append(sigs, e.Sig)
			if err := one("endorser", &p.Endorsers[k], e); err != nil {
				return p, err
			}
		}
		switch t.Fn {
		case "registerHolder":
			first, last = append(first, t.Args[1]), append(last, t.Args[2])
		case "suspendCredential":
			reason = append(reason, t.Args[1])
		case "issueCredential":
			for _, r := range t.RWSets {
				for _, w := range r.Writes {
					var n int
					if err := json.Unmarshal(w.Value.Fields["Issuer"], &n); err != nil {
						return p, fmt.Errorf("profile: Issuer length: %w", err)
					}
					if p.IssuerIdentity != 0 && n != p.IssuerIdentity {
						return p, fmt.Errorf("profile: Issuer length varies")
					}
					p.IssuerIdentity = n
				}
			}
		}
	}
	for _, b := range c.Blocks {
		for _, o := range b.Orderer {
			sigs = append(sigs, o.Sig)
			if err := one("orderer", &p.Orderer, o); err != nil {
				return p, err
			}
			p.MD0Value, p.HashLen, p.CommitHash = b.MD0Value, b.DataHash, b.MDLens[4]
		}
	}
	p.ChannelHeader, p.FabricSig = medianLow(chdr), medianLow(sigs)
	p.HolderFirst, p.HolderLast, p.SuspendReason = medianLow(first), medianLow(last), medianLow(reason)
	return p, nil
}

// ---------------------------------------------------------------- Fabric scenarios

// Fabric is the Fabric side of a transaction and its block: identities and
// signature length.
type Fabric struct {
	Name      string
	Creator   Identity
	Endorsers []Identity
	Orderer   Identity
	Sig       int // every Fabric signature: client, endorsements, block
}

// MeasuredFabric is QChain's network as it runs: ECDSA P-256.
func MeasuredFabric(p Profile) Fabric {
	f := Fabric{Name: "measured", Creator: p.Creator.Identity, Orderer: p.Orderer.Identity, Sig: p.FabricSig}
	for _, e := range p.Endorsers {
		f.Endorsers = append(f.Endorsers, e.Identity)
	}
	return f
}

// ProjectedFabric is a PROJECTION, not a measurement: the same network with
// every certificate (same names, validity and extensions) carrying an ML-DSA
// key of pk bytes signed by an ML-DSA CA, and every Fabric signature an
// ML-DSA signature of sig bytes. Fabric v2.5 does not support this today.
func ProjectedFabric(name string, p Profile, pk, sig int) Fabric {
	conv := func(c CertIdentity) Identity {
		return Identity{MSPLen: c.MSPLen, PEM: PEMLen(c.Cert.WithMLDSA(pk, sig).DER)}
	}
	f := Fabric{Name: name, Creator: conv(p.Creator), Orderer: conv(p.Orderer), Sig: sig}
	for _, e := range p.Endorsers {
		f.Endorsers = append(f.Endorsers, conv(e))
	}
	return f
}

// MLDSA are the ML-DSA public-key and signature sizes (FIPS 204) used by the
// projection scenarios.
var MLDSA = []struct {
	Name    string
	PK, Sig int
}{{"ml-dsa-44", 1312, 2420}, {"ml-dsa-65", 1952, 3309}, {"ml-dsa-87", 2592, 4627}}

// Call is the QChain part of one chaincode call: the function, its arguments,
// the keys it reads and writes in the qchaincode namespace, and the response.
type Call struct {
	Fn          string
	Args        []int // after the function name
	Reads       []Read
	Writes      []Write
	RespPayload int
}

// Tx assembles a full transaction for the call on this network.
func (f Fabric) Tx(p Profile, c Call) Tx {
	t := Tx{ChannelHeader: p.ChannelHeader, Creator: f.Creator, Nonce: p.Nonce, ClientSig: f.Sig,
		CCNameLen: p.CCNameLen, Args: append([]int{len(c.Fn)}, c.Args...), ProposalHash: p.ProposalHash,
		RespStatus: 200, RespPayload: c.RespPayload, CCIDAction: p.CCIDAction}
	for _, e := range f.Endorsers {
		t.Endorsements = append(t.Endorsements, Endorsement{Endorser: e, Sig: f.Sig})
	}
	t.NS = []NsRWSet{
		{NameLen: 10, Reads: []Read{LifecycleRead}},
		{NameLen: p.CCNameLen, Reads: c.Reads, Writes: c.Writes},
	}
	return t
}

// BlockOf is a block of the given number holding the envelopes, with ordinary
// metadata signed by the network's orderer.
func (f Fabric) BlockOf(p Profile, number uint64, envs []int) Block {
	sig := []MetaSig{{Signer: f.Orderer, Nonce: p.Nonce, Sig: f.Sig}}
	return Block{Number: number, DataHash: p.HashLen, PrevHash: p.HashLen, Envs: envs,
		Metadata: StandardMetadata(p.MD0Value, sig, len(envs), p.CommitHash)}
}

// Footprint is one transaction's share of the block file.
type Footprint struct {
	Envelope    int
	Message     int // what the block cutter counts
	Frame1      int // block of this transaction alone (as observed: one transaction per block)
	PerBlockMax int // transactions of this size per block when they arrive faster than the batch timeout
	ShareMax    int // this transaction's share of such a block (rounded up)
}

// Footprint places the transaction in a block at the given height.
func (f Fabric) Footprint(p Profile, t Tx, height uint64) Footprint {
	s := t.Sizes()
	fp := Footprint{Envelope: s.Envelope, Message: s.Message, PerBlockMax: MaxPerBlock(s.Message)}
	fp.Frame1 = f.BlockOf(p, height, []int{s.Envelope}).Frame()
	envs := make([]int, fp.PerBlockMax)
	for i := range envs {
		envs[i] = s.Envelope
	}
	frame := f.BlockOf(p, height, envs).Frame()
	fp.ShareMax = (frame + fp.PerBlockMax - 1) / fp.PerBlockMax
	return fp
}
