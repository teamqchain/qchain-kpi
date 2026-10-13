package main

// E13: ledger and IPFS bytes. Nothing is timed.
//
// A ledger block (one per signature scheme) issues real credentials with
// QChain's copied production code (qchain.Issue) for every attribute count,
// builds the chaincode records the way QChaincode.js stores them, and places
// every QChain call in Fabric blocks with the size model of internal/ledger,
// which reproduces the captured live ledger to the byte (ledger.Validate runs
// first in every block). Variable-length signatures (unpadded Falcon, ECDSA
// DER) are sampled, and every size is reported over the samples.
//
// An IPFS block (one per KEM) seals real envelopes (qchain.Issue) for every
// attribute count and value size, opens them with the wallet's code, and
// gives their IPFS storage.
//
// Scenarios: "measured" is QChain's network as captured; "projected-ml-dsa-*"
// is a PROJECTION with Fabric's certificates and signatures switched to ML-DSA;
// "whatif-*" are hypothetical QChain encodings (signature and keys in base64
// instead of hex, the issuer key stored as a 64-hex fingerprint).

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/ledger"
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/qchain"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/record"
	"qchain-kpi/internal/stats"
	"qchain-kpi/internal/timing"
)

// e13Height is the block height assumed for a transaction's reads: the middle
// of a million single-transaction blocks. A read's version costs 4 bytes at
// this height (3 below block 16,384); the capture's blocks were all below 128.
const e13Height = 500_000

var (
	e13Ns = []int{1, 5, 10, 20, 64}
	e13Vs = []int{16, 256, 4096}
)

// E11's credential (internal/flow) and the capture's typical lengths.
const (
	e13CredType = "Bachelor of Science in Computer Science"
	e13Holder   = "H-0001"
	e13OrgID    = "GeneralMSP"
	e13Expiry   = "2030-06-30"
	e13TxTime   = "2026-10-13T10:00:00Z" // QChaincode.js _txISO: UTC, second precision
)

func e13Samples(profile string) (int, error) {
	switch profile {
	case plan.ProfileApproved:
		return 1000, nil
	case plan.ProfileSmoke:
		return 50, nil
	}
	return 0, fmt.Errorf("no E13 budget for profile %q", profile)
}

// e13Row is one output row.
type e13Row struct {
	scenario, tx, kem string
	n, v              int
	quantity          string
	vals              []int
}

func (r e13Row) csv() string {
	s := append([]int(nil), r.vals...)
	sort.Ints(s)
	return fmt.Sprintf("%s,%s,%s,%d,%d,%s,%d,%d,%d", r.scenario, r.tx, r.kem, r.n, r.v, r.quantity,
		s[0], s[(len(s)-1)/2], s[len(s)-1])
}

func e13Block(sp proto.Spec, a algs.Alg) error {
	b := sp.Block
	if want := proto.E13Op(a); b.Op != want {
		return fmt.Errorf("block: E13 op of %s is %q, not %q", a.ID, want, b.Op)
	}
	samples, err := e13Samples(b.Profile)
	if err != nil {
		return err
	}
	rep, err := ledger.Validate()
	if err != nil {
		return fmt.Errorf("%w: size model does not reproduce the captured ledger: %v", timing.ErrCheck, err)
	}
	oomSet := procsetup.OOMAdj()
	rng := rand.New(rand.NewPCG(b.Seed, 0x6b70692d6531332d)) // "kpi-e13-"
	cpus, _ := quality.CPUSet()
	q0, _ := quality.Take()
	startNS := time.Now().UnixNano()
	t0 := time.Now()

	var rows []e13Row
	var arms []proto.ArmSummary
	checked := 0
	if a.Track == algs.Sig {
		rows, arms, checked, err = e13Ledger(a, samples, rng)
	} else {
		rows, arms, checked, err = e13IPFS(a, rng)
	}
	if err != nil {
		return err
	}
	wall := time.Since(t0).Nanoseconds()
	endNS := time.Now().UnixNano()
	q1, _ := quality.Take()

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	prof, _ := json.Marshal(ledger.Measured)
	header := map[string]string{
		"key": b.Key(), "id": b.ID, "exp": b.Exp, "alg": b.Alg, "op": b.Op,
		"rep": strconv.Itoa(b.Rep), "attempt": strconv.Itoa(b.Attempt), "profile": b.Profile,
		"seed": strconv.FormatUint(b.Seed, 10), "signature_samples": strconv.Itoa(samples),
		"validation":     rep.String() + " (capture SHA-256 " + ledger.CaptureSHA256 + ")",
		"fabric_profile": string(prof),
		"height":         strconv.Itoa(e13Height),
		"columns": "scenario, tx, kem, n = attributes, v = attribute value bytes, quantity, then min, median, max over the " +
			"signature samples (one value when the signature length is fixed); quantities are bytes, except blocks and per_block_max (counts)",
		"scenarios": "measured = QChain's network as captured; projected-ml-dsa-* = PROJECTION: Fabric certificates and " +
			"signatures switched to ML-DSA; whatif-base64 = signature and keys in base64 instead of hex; whatif-keyref = " +
			"issuer public key stored as a 64-hex fingerprint; whatif-both = both",
		"harness_commit": buildinfo.HarnessCommit, "go": runtime.Version(), "liboqs": pqc.Version(),
		"openssl": classical.OpenSSLVersion(), "start_unix_ns": strconv.FormatInt(startNS, 10),
	}
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(gz, "# %s=%s\n", k, strings.ReplaceAll(header[k], "\n", " "))
	}
	fmt.Fprintln(gz, "scenario,tx,kem,n,v,quantity,min,p50,max")
	for _, r := range rows {
		fmt.Fprintln(gz, r.csv())
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := record.WriteFileAtomic(sp.RawFile, buf.Bytes(), 0o644); err != nil {
		return err
	}
	sum, err := fileSHA256(sp.RawFile)
	if err != nil {
		return err
	}
	return emit(proto.Result{
		Key: b.Key(), Block: b, StartNS: startNS, EndNS: endNS, WallNS: wall,
		Budget:  proto.BudgetInfo{Mode: "sizes", Samples: samples},
		Quality: quality.Diff(q0, q1, cpus), CPUs: cpus, GOMAXPROCS: runtime.GOMAXPROCS(0), OOMAdj: oomSet,
		KeysChecked: checked, Rows: len(rows), RawSHA256: sum, Arms: arms,
	})
}

// e13Attributes is E11's credential content: n attributes field01.. of v
// random alphanumeric bytes and an expiry date, as the portal's info JSON.
func e13Attributes(rng *rand.Rand, n, v int) (map[string]string, string, error) {
	attrs := map[string]string{}
	m := map[string]string{"expiryDate": e13Expiry}
	for i := 1; i <= n; i++ {
		val := qchain.RandomValue(rng, v)
		attrs[qchain.FieldName(i)] = val
		m[qchain.FieldName(i)] = val
	}
	info, err := json.Marshal(m)
	return attrs, string(info), err
}

// sortedJSONLen is the length of a record as QChaincode.js stores it:
// recursively sorted keys (sortKeysRecursive), compact JSON.stringify. Go's
// encoder sorts map keys; the records hold no characters it would escape.
func sortedJSONLen(v any) (int, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return 0, err
	}
	out, err := json.Marshal(m)
	if bytes.ContainsAny(out, "<>&") {
		return 0, fmt.Errorf("record would be HTML-escaped")
	}
	return len(out), err
}

// encodings of a QChain scenario: lengths of a signature, the issuer's public
// key and a holder key of n bytes as stored.
type e13Scenario struct {
	name   string
	fab    ledger.Fabric
	sig    func(n int) int
	issuer func(n int) int
	holder func(n int) int
}

func hexLen(n int) int    { return 2 * n }
func base64Len(n int) int { return 4 * ((n + 2) / 3) }
func keyRef(int) int      { return 64 }

func e13Scenarios() []e13Scenario {
	p := ledger.Measured
	m := ledger.MeasuredFabric(p)
	out := []e13Scenario{{"measured", m, hexLen, hexLen, hexLen}}
	for _, d := range ledger.MLDSA {
		out = append(out, e13Scenario{"projected-" + d.Name, ledger.ProjectedFabric("projected-"+d.Name, p, d.PK, d.Sig), hexLen, hexLen, hexLen})
	}
	return append(out,
		e13Scenario{"whatif-base64", m, base64Len, base64Len, base64Len},
		e13Scenario{"whatif-keyref", m, hexLen, keyRef, hexLen},
		e13Scenario{"whatif-both", m, base64Len, keyRef, base64Len})
}

// e13Cred is one issued credential's lengths as measured (hex encodings).
type e13Cred struct {
	issueArgs  []int // after the function name; signature at 8, public key at 9
	updateArgs []int // credID, expiry, hash, signature, public key
	records    map[string]int
	sigBytes   int // raw signature length in this credential
	pkBytes    int
}

func e13Ledger(a algs.Alg, samples int, rng *rand.Rand) ([]e13Row, []proto.ArmSummary, int, error) {
	kemDefault := algs.MustByID("ml-kem-768")
	if err := qchain.SetSuite(a, kemDefault); err != nil {
		return nil, nil, 0, err
	}
	defer qchain.SetSuite(algs.MustByID("ml-dsa-44"), kemDefault)
	issuerPub, issuerPriv, err := qchain.IssuerKeygen()
	if err != nil {
		return nil, nil, 0, err
	}
	kemPub, _, dsaPub, _, err := qchain.WalletKeygen()
	if err != nil {
		return nil, nil, 0, err
	}
	checked := 0

	// Signature lengths: sampled when they vary (E8's path: the library object,
	// every signature verified), otherwise the one fixed length.
	s, err := newSignature(a)
	if err != nil {
		return nil, nil, 0, err
	}
	sizes := s.Sizes()
	if sizes.SigVariable != proto.E13SigVariable(a) {
		return nil, nil, 0, fmt.Errorf("block: %s signature variability is %v, the estimate assumes %v", a.ID, sizes.SigVariable, proto.E13SigVariable(a))
	}
	sigLens := []int{sizes.Signature}
	if sizes.SigVariable {
		pk, sk, err := s.GenerateKey()
		if err != nil {
			return nil, nil, 0, err
		}
		sigLens = sigLens[:0]
		for i := 0; i < samples; i++ {
			m := inputs.HexMessage(rng)
			sg, err := s.Sign(sk, pk, m)
			if err != nil {
				return nil, nil, 0, err
			}
			if ok, err := s.Verify(pk, m, sg); err != nil || !ok {
				return nil, nil, 0, fmt.Errorf("%w: sampled signature %d does not verify", timing.ErrCheck, i)
			}
			sigLens = append(sigLens, len(sg))
			checked++
		}
	}

	creds := map[int]e13Cred{}
	for _, n := range e13Ns {
		_, info, err := e13Attributes(rng, n, 16)
		if err != nil {
			return nil, nil, 0, err
		}
		cid := qchain.FakeCIDv0([]byte(info))
		res, err := qchain.Issue(info, e13Holder, e13CredType, kemPub, cid, e13OrgID, issuerPriv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("block: issue %s n=%d: %w", a.ID, n, err)
		}
		if ok, err := qchain.SigVerify(res.CredentialHash, res.Signature, issuerPub); err != nil || !ok {
			return nil, nil, 0, fmt.Errorf("%w: issued credential n=%d does not verify (%v)", timing.ErrCheck, n, err)
		}
		checked++
		credID := "CRED-" + strings.Repeat("0", 64) // "CRED-" + Fabric transaction ID
		rec := qchain.ChainCredential{DocType: "credential", CommitmentVersion: 2, ID: credID, Holder: e13Holder,
			Issuer: ledger.IssuerPlaceholder, IssuerOrgID: e13OrgID, Status: "active", CredentialType: e13CredType,
			IssuedAt: res.Commitment.IssuedAt, ExpiryDate: res.Commitment.ExpiryDate, CID: cid,
			FieldHashes: res.Commitment.FieldHashes, CredentialHash: res.CredentialHash, Signature: res.Signature,
			PublicKey: issuerPub}
		c := e13Cred{
			issueArgs: []int{len(e13Holder), len(e13CredType), len(res.Commitment.IssuedAt), len(res.Commitment.ExpiryDate),
				len(e13OrgID), len(cid), len(res.FieldHashesJSON), len(res.CredentialHash), len(res.Signature), len(issuerPub)},
			updateArgs: []int{len(credID), len(e13Expiry), len(res.CredentialHash), len(res.Signature), len(issuerPub)},
			records:    map[string]int{}, sigBytes: len(res.Signature) / 2, pkBytes: len(issuerPub) / 2,
		}
		variants := map[string]qchain.ChainCredential{"issueCredential": rec}
		up := rec
		up.ExpiryUpdatedAt = e13TxTime
		variants["updateExpiry"] = up
		su := rec
		su.Status, su.SuspendedAt, su.SuspendedReason = "suspended", e13TxTime, strings.Repeat("r", ledger.Measured.SuspendReason)
		variants["suspendCredential"] = su
		re := rec
		re.RestoredAt = e13TxTime
		variants["restoreCredential"] = re
		rv := rec
		rv.Status, rv.RevokedAt = "revoked", e13TxTime
		variants["revokeCredential"] = rv
		for k, v := range variants {
			if c.records[k], err = sortedJSONLen(v); err != nil {
				return nil, nil, 0, err
			}
		}
		if !sizes.SigVariable && c.sigBytes != sizes.Signature {
			return nil, nil, 0, fmt.Errorf("%w: %s signature of %d bytes, the library reports %d", timing.ErrCheck, a.ID, c.sigBytes, sizes.Signature)
		}
		creds[n] = c
	}

	// Holder records: registration, then each KEM's keys bound (the holder's
	// signature key is the scheme measured).
	p := ledger.Measured
	holder := map[string]string{"DocType": "holder", "ID": e13Holder,
		"FirstName": strings.Repeat("a", p.HolderFirst), "LastName": strings.Repeat("b", p.HolderLast)}
	regRecord, err := sortedJSONLen(holder)
	if err != nil {
		return nil, nil, 0, err
	}
	dsaBytes := len(dsaPub) / 2
	type kemKey struct {
		id    string
		bytes int
	}
	var kems []kemKey
	for _, k := range algs.All() {
		if k.Track != algs.KEM {
			continue
		}
		obj, err := newKEM(k)
		if err != nil {
			return nil, nil, 0, err
		}
		kems = append(kems, kemKey{k.ID, obj.Sizes().PublicKey})
	}

	var rows []e13Row
	var arms []proto.ArmSummary
	add := func(sc, tx, kem string, n int, fps []ledger.Footprint, args, recs []int) {
		col := func(f func(ledger.Footprint) int) []int {
			out := make([]int, len(fps))
			for i, x := range fps {
				out[i] = f(x)
			}
			return out
		}
		rows = append(rows,
			e13Row{sc, tx, kem, n, 16, "envelope", col(func(x ledger.Footprint) int { return x.Envelope })},
			e13Row{sc, tx, kem, n, 16, "message", col(func(x ledger.Footprint) int { return x.Message })},
			e13Row{sc, tx, kem, n, 16, "args", args},
			e13Row{sc, tx, kem, n, 16, "record", recs},
			e13Row{sc, tx, kem, n, 16, "frame_1", col(func(x ledger.Footprint) int { return x.Frame1 })},
			e13Row{sc, tx, kem, n, 16, "per_block_max", col(func(x ledger.Footprint) int { return x.PerBlockMax })},
			e13Row{sc, tx, kem, n, 16, "share_max", col(func(x ledger.Footprint) int { return x.ShareMax })})
		if sc == "measured" && tx == "issueCredential" {
			env := col(func(x ledger.Footprint) int { return x.Envelope })
			xs := make([]float64, len(env))
			lo, hi := env[0], env[0]
			for i, e := range env {
				xs[i] = float64(e)
				lo, hi = min(lo, e), max(hi, e)
			}
			arms = append(arms, proto.ArmSummary{Name: fmt.Sprintf("issue-n%d", n), Batch: 1,
				Stats: stats.Summarize(xs, 0, 0), MinBytes: lo, MaxBytes: hi})
		}
	}
	argSum := func(fn string, args []int) int { return len(fn) + sum(args) }

	for _, sc := range e13Scenarios() {
		for _, n := range e13Ns {
			c := creds[n]
			for _, tx := range []string{"issueCredential", "updateExpiry", "suspendCredential", "restoreCredential", "revokeCredential"} {
				var fps []ledger.Footprint
				var argT, recT []int
				for _, l := range sigLens {
					// The record and arguments with this signature length, in the scenario's encodings.
					delta := sc.sig(l) - 2*c.sigBytes + sc.issuer(c.pkBytes) - 2*c.pkBytes
					rec := c.records[tx] + delta
					credKey := len("CRED-") + 64
					var call ledger.Call
					switch tx {
					case "issueCredential":
						args := append([]int(nil), c.issueArgs...)
						args[8], args[9] = sc.sig(l), sc.issuer(c.pkBytes)
						call = ledger.IssueCall(args, credKey, rec, e13Height)
					case "updateExpiry":
						args := append([]int(nil), c.updateArgs...)
						args[3], args[4] = sc.sig(l), sc.issuer(c.pkBytes)
						call = ledger.UpdateExpiryCall(args, rec, e13Height)
					case "suspendCredential":
						call = ledger.SuspendCall(credKey, p.SuspendReason, rec, e13Height)
					case "restoreCredential":
						call = ledger.RestoreCall(credKey, rec, e13Height)
					case "revokeCredential":
						call = ledger.RevokeCall(credKey, rec, e13Height)
					}
					fps = append(fps, sc.fab.Footprint(p, sc.fab.Tx(p, call), e13Height))
					argT, recT = append(argT, argSum(call.Fn, call.Args)), append(recT, rec)
				}
				add(sc.name, tx, "", n, fps, argT, recT)
			}
		}
		// Holder registration (no algorithm involved) and key binding for every KEM.
		reg := ledger.RegisterCall(len(e13Holder), p.HolderFirst, p.HolderLast, regRecord)
		add(sc.name, "registerHolder", "", 0, []ledger.Footprint{sc.fab.Footprint(p, sc.fab.Tx(p, reg), e13Height)},
			[]int{argSum(reg.Fn, reg.Args)}, []int{regRecord})
		for _, k := range kems {
			hk := map[string]string{}
			for kk, vv := range holder {
				hk[kk] = vv
			}
			hk["KemPublicKey"] = strings.Repeat("0", sc.holder(k.bytes))
			hk["DsaPublicKey"] = strings.Repeat("0", sc.holder(dsaBytes))
			rec, err := sortedJSONLen(hk)
			if err != nil {
				return nil, nil, 0, err
			}
			call := ledger.BindCall(len(e13Holder), sc.holder(k.bytes), sc.holder(dsaBytes), rec, e13Height)
			add(sc.name, "bindHolderKeys", k.id, 0, []ledger.Footprint{sc.fab.Footprint(p, sc.fab.Tx(p, call), e13Height)},
				[]int{argSum(call.Fn, call.Args)}, []int{rec})
		}
	}
	return rows, arms, checked, nil
}

func sum(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

func e13IPFS(a algs.Alg, rng *rand.Rand) ([]e13Row, []proto.ArmSummary, int, error) {
	sigDefault := algs.MustByID("ml-dsa-44")
	if err := qchain.SetSuite(sigDefault, a); err != nil {
		return nil, nil, 0, err
	}
	defer qchain.SetSuite(sigDefault, algs.MustByID("ml-kem-768"))
	_, issuerPriv, err := qchain.IssuerKeygen()
	if err != nil {
		return nil, nil, 0, err
	}
	kemPub, kemPriv, _, _, err := qchain.WalletKeygen()
	if err != nil {
		return nil, nil, 0, err
	}
	var rows []e13Row
	var arms []proto.ArmSummary
	checked := 0
	for _, n := range e13Ns {
		for _, v := range e13Vs {
			// Two envelopes per setting: their lengths must agree (the format has
			// no variable-length part for fixed attribute lengths).
			var lens []int
			for i := 0; i < 2; i++ {
				attrs, info, err := e13Attributes(rng, n, v)
				if err != nil {
					return nil, nil, 0, err
				}
				res, err := qchain.Issue(info, e13Holder, e13CredType, kemPub, qchain.FakeCIDv0([]byte(info)), e13OrgID, issuerPriv)
				if err != nil {
					return nil, nil, 0, fmt.Errorf("block: seal %s n=%d v=%d: %w", a.ID, n, v, err)
				}
				opened, err := qchain.WalletDecrypt(res.Envelope, kemPriv)
				if err != nil {
					return nil, nil, 0, fmt.Errorf("%w: envelope n=%d v=%d does not open: %v", timing.ErrCheck, n, v, err)
				}
				for k, want := range attrs {
					if opened[k] != want {
						return nil, nil, 0, fmt.Errorf("%w: envelope n=%d v=%d: %s does not decrypt", timing.ErrCheck, n, v, k)
					}
				}
				checked++
				lens = append(lens, len(res.Envelope))
			}
			if lens[0] != lens[1] {
				return nil, nil, 0, fmt.Errorf("%w: envelope length varies (%d, %d) for n=%d v=%d", timing.ErrCheck, lens[0], lens[1], n, v)
			}
			bl, err := ledger.IPFSBlocks(lens[0])
			if err != nil {
				return nil, nil, 0, err
			}
			rows = append(rows,
				e13Row{"measured", "envelope", a.ID, n, v, "file", lens},
				e13Row{"measured", "envelope", a.ID, n, v, "stored", []int{sum(bl)}},
				e13Row{"measured", "envelope", a.ID, n, v, "blocks", []int{len(bl)}},
				e13Row{"measured", "envelope", a.ID, n, v, "on_disk", []int{ledger.OnDisk(bl, 4096)}})
			arms = append(arms, proto.ArmSummary{Name: fmt.Sprintf("env-n%d-v%d", n, v), Batch: 1,
				Stats: stats.Summarize([]float64{float64(lens[0])}, 0, 0), MinBytes: lens[0], MaxBytes: lens[0]})
		}
	}
	return rows, arms, checked, nil
}
