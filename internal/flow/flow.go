// Package flow is E11: the QChain credential lifecycle measured stage by
// stage through the copied production code (internal/qchain). Cells and the
// grid are defined in internal/cells; this package prepares a cell's inputs
// (untimed) and builds its measured arms. Everything a stage needs that
// production gets from I/O (the IPFS CID, the chain record) is prepared
// before timing (decision C).
package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/cells"
	"qchain-kpi/internal/qchain"
	"qchain-kpi/internal/timing"
)

// Defaults used where a stage does not depend on an algorithm (the library
// still needs one selected).
const (
	defaultSig = "ml-dsa-44"
	defaultKEM = "ml-kem-768"
)

// Fixed inputs (decision of 10 Oct): attribute names field01..., an expiry date.
const (
	expiryDate     = "2030-06-30"
	credentialType = "Bachelor of Science in Computer Science"
	holderID       = "H-0001"
	issuerOrgID    = "GeneralMSP"
	// issuerIdentity stands in for the Fabric client identity the chaincode
	// stores as Issuer (x509::<subject>::<issuer>); E13 replaces it with the
	// real value captured on the VM.
	issuerIdentity = "x509::CN=issuer1,OU=client,O=Hyperledger,ST=North Carolina,C=US::CN=ca.org1.example.com,O=org1.example.com,L=Durham,ST=North Carolina,C=US"
)

// Setup holds one block's prepared inputs and its arms.
type Setup struct {
	Cell  cells.Cell
	P     int
	Arms  []timing.Arm
	After func() (int, error) // deferred checks (issuance envelopes); returns the number fully checked
}

// reusable is an in-memory request whose body is replaced for every call:
// the HTTP transport is excluded (decision C), only its body is decoded.
type reusable struct{ r *http.Request }

func newReusable() *reusable {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	return &reusable{r: r}
}

func (u *reusable) with(body []byte) *http.Request {
	u.r.Body = io.NopCloser(bytes.NewReader(body))
	return u.r
}

func algOrDefault(id, def string) algs.Alg {
	if id == "" {
		id = def
	}
	return algs.MustByID(id)
}

// attributes returns n attributes of v bytes and the portal's info JSON.
func attributes(rng *rand.Rand, n, v int) (map[string]string, string) {
	attrs := make(map[string]string, n)
	m := map[string]string{"expiryDate": expiryDate}
	for i := 1; i <= n; i++ {
		val := qchain.RandomValue(rng, v)
		attrs[qchain.FieldName(i)] = val
		m[qchain.FieldName(i)] = val
	}
	b, _ := json.Marshal(m)
	return attrs, string(b)
}

func cidFor(rng *rand.Rand) string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(rng.Uint32())
	}
	return qchain.FakeCIDv0(b)
}

// Prepare builds a block's inputs (untimed) and its arms.
func Prepare(c cells.Cell, seed uint64) (*Setup, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := qchain.SetSuite(algOrDefault(c.Sig, defaultSig), algOrDefault(c.KEM, defaultKEM)); err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewPCG(seed, 0x6b70692d65313121)) // "kpi-e11!"
	st := &Setup{Cell: c, P: cells.PoolSize(c), After: func() (int, error) { return 0, nil }}
	var err error
	switch c.Stage {
	case cells.Activation:
		err = st.activation()
	case cells.Issuance:
		err = st.issuance(rng)
	case cells.Retrieval:
		err = st.retrieval(rng)
	case cells.Presentation:
		err = st.presentation(rng)
	case cells.Verification:
		err = st.verification(rng)
	case cells.Chained:
		err = st.chained(rng)
	}
	return st, err
}

// ----------------------------------------------------------------- stages

func (st *Setup) activation() error {
	arm, err := activationArm()
	if err != nil {
		return err
	}
	st.Arms = []timing.Arm{arm}
	return nil
}

// activationArm: the wallet generates both key pairs and sends their public
// halves; the backend decodes the request and checks the keys.
func activationArm() (timing.Arm, error) {
	req := newReusable()
	var kemPub, dsaPub string
	var checkErr error
	return timing.Arm{
		Name: cells.ArmStage,
		Call: func(int) (int, error) {
			var err error
			kemPub, _, dsaPub, _, err = qchain.WalletKeygen()
			if err != nil {
				return 0, err
			}
			body, err := json.Marshal(map[string]string{"emiratesID": "784-1990-1234567-1", "kemPublicKey": kemPub, "dsaPublicKey": dsaPub})
			if err != nil {
				return 0, err
			}
			checkErr = qchain.ActivationCheck(req.with(body))
			return len(body), nil
		},
		Check: func(int, int) error {
			if checkErr != nil {
				return fmt.Errorf("activation check rejected the wallet's keys: %v", checkErr)
			}
			return nil
		},
	}, nil
}

// issuer is the issuer's key pair (generated once, untimed).
type issuer struct{ pub, priv string }

func newIssuer() (*issuer, error) {
	pub, priv, err := qchain.IssuerKeygen()
	if err != nil {
		return nil, err
	}
	return &issuer{pub: pub, priv: priv}, nil
}

type holderKeys struct{ kemPub, kemPriv, dsaPub, dsaPriv string }

func newHolder() (*holderKeys, error) {
	var h holderKeys
	var err error
	h.kemPub, h.kemPriv, h.dsaPub, h.dsaPriv, err = qchain.WalletKeygen()
	return &h, err
}

func (st *Setup) issuance(rng *rand.Rand) error {
	iss, err := newIssuer()
	if err != nil {
		return err
	}
	h, err := newHolder()
	if err != nil {
		return err
	}
	arm, after, err := issuanceArm(rng, st.Cell, st.P, iss, h)
	if err != nil {
		return err
	}
	st.Arms, st.After = []timing.Arm{arm}, after
	return nil
}

// issuanceArm: the issuance handler's computation for one credential. Every
// commitment signature is verified (untimed); the envelopes of a seeded
// subset (every 50th, at least 3) are fully decrypted and compared after the
// timed phase, as for key generation in E1 (a full decryption of every
// envelope would double the block's wall time).
func issuanceArm(rng *rand.Rand, c cells.Cell, p int, iss *issuer, h *holderKeys) (timing.Arm, func() (int, error), error) {
	infos := make([]string, p)
	attrs := make([]map[string]string, p)
	cids := make([]string, p)
	for i := range infos {
		attrs[i], infos[i] = attributes(rng, c.N, c.V)
		cids[i] = cidFor(rng)
	}
	offset := int(rng.Uint64() % 50)
	type saved struct {
		env   []byte
		attrs map[string]string
	}
	var keep []saved
	var last *qchain.IssueResult
	arm := timing.Arm{
		Name: cells.ArmStage,
		Call: func(i int) (int, error) {
			var err error
			last, err = qchain.Issue(infos[i%p], holderID, credentialType, h.kemPub, cids[i%p], issuerOrgID, iss.priv)
			if err != nil {
				return 0, err
			}
			return len(last.Envelope), nil
		},
		Check: func(i, _ int) error {
			if len(last.Commitment.FieldHashes) != c.N {
				return fmt.Errorf("issuance: %d field hashes, want %d", len(last.Commitment.FieldHashes), c.N)
			}
			ok, err := qchain.SigVerify(last.CredentialHash, last.Signature, iss.pub)
			if err != nil || !ok {
				return fmt.Errorf("issuance %d: commitment signature does not verify (%v)", i, err)
			}
			if i%50 == offset || len(keep) < 3 {
				keep = append(keep, saved{env: bytes.Clone(last.Envelope), attrs: attrs[i%p]})
			}
			return nil
		},
	}
	after := func() (int, error) {
		for n, s := range keep {
			fields, err := qchain.WalletDecrypt(s.env, h.kemPriv)
			if err != nil {
				return n, fmt.Errorf("issued envelope %d does not decrypt: %v", n, err)
			}
			for k, v := range s.attrs {
				if fields[k] != v {
					return n, fmt.Errorf("issued envelope %d: field %s decrypts to %v", n, k, fields[k])
				}
			}
		}
		return len(keep), nil
	}
	return arm, after, nil
}

func (st *Setup) retrieval(rng *rand.Rand) error {
	h, err := newHolder()
	if err != nil {
		return err
	}
	arm, err := retrievalArm(rng, st.Cell, st.P, h)
	if err != nil {
		return err
	}
	st.Arms = []timing.Arm{arm}
	return nil
}

// retrievalArm: the backend checks the envelope read from IPFS, the wallet
// decrypts every field.
func retrievalArm(rng *rand.Rand, c cells.Cell, p int, h *holderKeys) (timing.Arm, error) {
	envs := make([][]byte, p)
	want := make([]map[string]string, p)
	for i := range envs {
		var err error
		want[i], _ = attributes(rng, c.N, c.V)
		salts, _, err := qchain.CommitAttributes(want[i])
		if err != nil {
			return timing.Arm{}, err
		}
		if envs[i], err = qchain.SealEnvelope(want[i], salts, h.kemPub); err != nil {
			return timing.Arm{}, err
		}
	}
	var fields map[string]any
	var looks bool
	return timing.Arm{
		Name: cells.ArmStage,
		Call: func(i int) (int, error) {
			looks = qchain.EnvelopeCheck(envs[i%p])
			var err error
			fields, err = qchain.WalletDecrypt(envs[i%p], h.kemPriv)
			return len(fields), err
		},
		Check: func(i, _ int) error {
			if !looks {
				return fmt.Errorf("retrieval %d: envelope check failed", i)
			}
			w := want[i%p]
			if len(fields) != len(w)+1 { // + "_salts"
				return fmt.Errorf("retrieval %d: %d fields, want %d", i, len(fields), len(w)+1)
			}
			for k, v := range w {
				if fields[k] != v {
					return fmt.Errorf("retrieval %d: field %s decrypts to %v", i, k, fields[k])
				}
			}
			return nil
		},
	}, nil
}

func (st *Setup) presentation(rng *rand.Rand) error {
	h, err := newHolder()
	if err != nil {
		return err
	}
	arm, err := presentationArm(rng, st.Cell, st.P, h)
	if err != nil {
		return err
	}
	st.Arms = []timing.Arm{arm}
	return nil
}

// disclosure is k disclosed attributes of one credential with their salts.
type disclosure struct {
	credID           string
	disclosed, salts map[string]string
}

func disclosures(rng *rand.Rand, p, k, v int) []disclosure {
	out := make([]disclosure, p)
	for i := range out {
		d := disclosure{credID: fmt.Sprintf("CRED-%04d", 1+rng.IntN(9999)), disclosed: map[string]string{}, salts: map[string]string{}}
		for j := 1; j <= k; j++ {
			f := qchain.FieldName(j)
			d.disclosed[f] = qchain.RandomValue(rng, v)
			d.salts[f] = qchain.RandomSalt(rng)
		}
		out[i] = d
	}
	return out
}

// presentationArm: the wallet builds the canonical payload, hashes and signs
// it and sends the request; the backend decodes and checks it. The request's
// hiddenFields list is sent empty: the wallet lists the names of the N-k
// hidden attributes there, but decision D measures presentation without N,
// and decoding up to 63 short names costs well under a microsecond.
func presentationArm(rng *rand.Rand, c cells.Cell, p int, h *holderKeys) (timing.Arm, error) {
	ds := disclosures(rng, p, c.K, c.V)
	req := newReusable()
	var payload, sig string
	var checkErr error
	return timing.Arm{
		Name: cells.ArmStage,
		Call: func(i int) (int, error) {
			d := ds[i%p]
			var err error
			payload, err = qchain.BuildDisclosedPayload(d.credID, d.disclosed, d.salts, qchain.WalletTimestamp(time.Now()))
			if err != nil {
				return 0, err
			}
			if sig, err = qchain.WalletSign(payload, h.dsaPriv); err != nil {
				return 0, err
			}
			body, err := qchain.PresentationRequest(d.credID, nil, payload, sig)
			if err != nil {
				return 0, err
			}
			checkErr = qchain.PresentationCheck(req.with(body))
			return len(body), nil
		},
		Check: func(i, _ int) error {
			if checkErr != nil {
				return fmt.Errorf("presentation %d rejected: %v", i, checkErr)
			}
			ok, err := qchain.SigVerify(qchain.PayloadHash(payload), sig, h.dsaPub)
			if err != nil || !ok {
				return fmt.Errorf("presentation %d: holder signature does not verify (%v)", i, err)
			}
			return nil
		},
	}, nil
}

func (st *Setup) verification(rng *rand.Rand) error {
	iss, err := newIssuer()
	if err != nil {
		return err
	}
	h, err := newHolder()
	if err != nil {
		return err
	}
	arms, err := verificationArms(rng, st.Cell, st.P, iss, h)
	if err != nil {
		return err
	}
	st.Arms = arms
	return nil
}

// presented is one issued credential as the chain returns it, plus a signed
// presentation of k of its attributes.
type presented struct {
	displayID, fabricID string
	response            []byte
	payload, sig        string
}

func presentations(rng *rand.Rand, c cells.Cell, p int, iss *issuer, h *holderKeys) ([]presented, error) {
	out := make([]presented, p)
	holder := &qchain.ChainHolder{DocType: "holder", ID: holderID, FirstName: "Mohammed", LastName: "Ali",
		KemPublicKey: h.kemPub, DsaPublicKey: h.dsaPub}
	for i := range out {
		attrs, info := attributes(rng, c.N, c.V)
		cid := cidFor(rng)
		res, err := qchain.Issue(info, holderID, credentialType, h.kemPub, cid, issuerOrgID, iss.priv)
		if err != nil {
			return nil, err
		}
		fabricID := fmt.Sprintf("CRED-%016x%016x%016x%016x", rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64())
		cred := &qchain.ChainCredential{DocType: "credential", CommitmentVersion: 2, ID: fabricID, Holder: holderID,
			Issuer: issuerIdentity, IssuerOrgID: issuerOrgID, Status: "active", CredentialType: credentialType,
			IssuedAt: res.Commitment.IssuedAt, ExpiryDate: res.Commitment.ExpiryDate, CID: cid,
			FieldHashes: res.Commitment.FieldHashes, CredentialHash: res.CredentialHash, Signature: res.Signature, PublicKey: iss.pub}
		resp, err := qchain.ChainResponse(cred, holder)
		if err != nil {
			return nil, err
		}
		disclosed, salts := map[string]string{}, map[string]string{}
		for j := 1; j <= c.K; j++ {
			f := qchain.FieldName(j)
			disclosed[f], salts[f] = attrs[f], res.Salts[f]
		}
		displayID := fmt.Sprintf("CRED-%04d", 1+i)
		payload, err := qchain.BuildDisclosedPayload(displayID, disclosed, salts, qchain.WalletTimestamp(time.Now()))
		if err != nil {
			return nil, err
		}
		sig, err := qchain.WalletSign(payload, h.dsaPriv)
		if err != nil {
			return nil, err
		}
		out[i] = presented{displayID: displayID, fabricID: fabricID, response: resp, payload: payload, sig: sig}
	}
	return out, nil
}

// verificationArms: the backend decodes the chain's response and runs
// verifyPresentation ("stage"); the decoding alone is a second arm
// ("chain-decode"), interleaved in the same process.
func verificationArms(rng *rand.Rand, c cells.Cell, p int, iss *issuer, h *holderKeys) ([]timing.Arm, error) {
	ps, err := presentations(rng, c, p, iss, h)
	if err != nil {
		return nil, err
	}
	var out qchain.VerifyOutcome
	var cred *qchain.ChainCredential
	stage := timing.Arm{
		Name: cells.ArmStage,
		Call: func(i int) (int, error) {
			x := ps[i%p]
			cr, holder, err := qchain.DecodeChainResponse(x.response, x.fabricID)
			if err != nil {
				return 0, err
			}
			out = qchain.Verify(x.displayID, x.fabricID, cr, holder, x.payload, x.sig, iss.pub, time.Now())
			return 0, nil
		},
		Check: func(i, _ int) error {
			if !out.Verified {
				return fmt.Errorf("verification %d failed: %s", i, out.Reason)
			}
			return nil
		},
	}
	decode := timing.Arm{
		Name: cells.ArmChainDecode,
		Call: func(i int) (int, error) {
			x := ps[i%p]
			var err error
			cred, _, err = qchain.DecodeChainResponse(x.response, x.fabricID)
			return len(x.response), err
		},
		Check: func(i, _ int) error {
			if cred == nil || len(cred.FieldHashes) != c.N {
				return fmt.Errorf("chain decode %d: record not decoded", i)
			}
			return nil
		},
	}
	return []timing.Arm{stage, decode}, nil
}

// chained runs the five stages back to back on their own prepared inputs
// (the same work their own blocks measure), so the sum of the stage results
// can be compared with the sequence.
func (st *Setup) chained(rng *rand.Rand) error {
	c := st.Cell
	iss, err := newIssuer()
	if err != nil {
		return err
	}
	h, err := newHolder()
	if err != nil {
		return err
	}
	act, err := activationArm()
	if err != nil {
		return err
	}
	isu, after, err := issuanceArm(rng, cells.Cell{Stage: cells.Issuance, Sig: c.Sig, KEM: c.KEM, N: c.N, V: c.V}, st.P, iss, h)
	if err != nil {
		return err
	}
	ret, err := retrievalArm(rng, cells.Cell{Stage: cells.Retrieval, KEM: c.KEM, N: c.N, V: c.V}, st.P, h)
	if err != nil {
		return err
	}
	pre, err := presentationArm(rng, cells.Cell{Stage: cells.Presentation, Sig: c.Sig, K: c.K, V: c.V}, st.P, h)
	if err != nil {
		return err
	}
	ver, err := verificationArms(rng, c, st.P, iss, h)
	if err != nil {
		return err
	}
	steps := []timing.Arm{act, isu, ret, pre, ver[0]}
	st.Arms = []timing.Arm{{
		Name: cells.ArmStage,
		Call: func(i int) (int, error) {
			for _, s := range steps {
				if _, err := s.Call(i); err != nil {
					return 0, fmt.Errorf("%s: %w", s.Name, err)
				}
			}
			return 0, nil
		},
		Check: func(i, n int) error {
			for k, s := range steps {
				if err := s.Check(i, n); err != nil {
					return fmt.Errorf("chained step %d: %w", k+1, err)
				}
			}
			return nil
		},
	}}
	st.After = after
	return nil
}
