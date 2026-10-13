package ledger

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The read-only capture of the live QChain ledger and IPFS store (13 Oct 2026;
// provenance and privacy rules in vmcapture/README.md).
var (
	//go:embed vmcapture/capture-export.json
	captureJSON []byte
	//go:embed vmcapture/ipfs-link.txt
	ipfsLinkTxt []byte
	//go:embed vmcapture/ipfs-pins.txt
	ipfsPinsTxt []byte
)

// CaptureSHA256 is the SHA-256 of vmcapture/capture-export.json as made on the VM.
const CaptureSHA256 = "cd4d6d1490216b5f7e5de3e7c736886a1b6bbc757e4877d01a622e656fbbb45c"

type capIdentity struct {
	Identity
	CertLayout
	MSP         string `json:"msp"`
	Ser         int    `json:"ser"`
	Sig         int    `json:"sig"`
	Endorsement int    `json:"endorsement"`
	SigHeader   int    `json:"sig_header"`
	IDHeader    int    `json:"id_header"`
	Nonce       int    `json:"nonce"`
}

type capWrite struct {
	Key    int `json:"key"`
	Delete int `json:"delete"`
	Value  struct {
		Bytes  int                        `json:"bytes"`
		Fields map[string]json.RawMessage `json:"fields"`
	} `json:"value"`
}

type capNS struct {
	NS     string     `json:"ns"`
	Reads  int        `json:"reads"`
	RWSet  int        `json:"rwset"`
	Writes []capWrite `json:"writes"`
}

type capTx struct {
	Block           int           `json:"block"`
	Type            int           `json:"type"`
	Valid           int           `json:"valid"`
	CC              string        `json:"cc"`
	Fn              string        `json:"fn"`
	Env             int           `json:"env"`
	Payload         int           `json:"payload"`
	ClientSig       int           `json:"client_sig"`
	ChannelHeader   int           `json:"channel_header"`
	SigHeader       int           `json:"sig_header"`
	Nonce           int           `json:"nonce"`
	Data            int           `json:"data"`
	ActionPayload   int           `json:"action_payload"`
	ProposalPayload int           `json:"proposal_payload"`
	InputSpec       int           `json:"input_spec"`
	PRP             int           `json:"prp"`
	ProposalHash    int           `json:"proposal_hash"`
	Extension       int           `json:"extension"`
	Results         int           `json:"results"`
	Events          int           `json:"events"`
	RespStatus      uint64        `json:"resp_status"`
	RespMessage     int           `json:"resp_message"`
	RespPayload     int           `json:"resp_payload"`
	CAChaincodeID   int           `json:"ca_chaincode_id"`
	Args            []int         `json:"args"`
	Creator         capIdentity   `json:"creator"`
	Endorsements    []capIdentity `json:"endorsements"`
	RWSets          []capNS       `json:"rwsets"`
}

type capBlock struct {
	Number       uint64        `json:"number"`
	Frame        int           `json:"frame"`
	Block        int           `json:"block"`
	NTx          int           `json:"ntx"`
	MDLens       []int         `json:"md_lens"`
	PreviousHash int           `json:"previous_hash"`
	DataHash     int           `json:"data_hash"`
	MD0Value     int           `json:"md0_value"`
	Orderer      []capIdentity `json:"orderer_signatures"`
}

type capture struct {
	FileBytes   int        `json:"file_bytes"`
	ParsedBytes int        `json:"parsed_bytes"`
	Blocks      []capBlock `json:"blocks"`
	Txs         []capTx    `json:"txs"`
}

func loadCapture() (*capture, error) {
	var c capture
	if err := json.Unmarshal(captureJSON, &c); err != nil {
		return nil, fmt.Errorf("ledger: capture: %w", err)
	}
	return &c, nil
}

// Function-name lengths: the capture records the name, the model needs its length.
func fnLen(fn string) int { return len(fn) }

// nsNameLen is the length of a namespace name in the capture (the parser
// prints only the known public names).
func nsNameLen(ns string) (int, error) {
	switch ns {
	case "qchaincode", "_lifecycle":
		return 10, nil
	case "lscc":
		return 4, nil
	}
	return 0, fmt.Errorf("unknown namespace %q", ns)
}

func ident(c capIdentity) Identity { return c.Identity }

// componentTx rebuilds a captured transaction from its measured parts, with
// each namespace's read/write set size taken as measured (calibration level 1:
// the framing of every enclosing message).
func componentTx(t capTx) (Tx, error) {
	tx := Tx{ChannelHeader: t.ChannelHeader, Creator: ident(t.Creator), Nonce: t.Nonce, ClientSig: t.ClientSig,
		CCNameLen: 10, Args: append([]int{fnLen(t.Fn)}, t.Args...), ProposalHash: t.ProposalHash, Events: t.Events,
		RespStatus: t.RespStatus, RespMessage: t.RespMessage, RespPayload: t.RespPayload, CCIDAction: t.CAChaincodeID}
	for _, e := range t.Endorsements {
		tx.Endorsements = append(tx.Endorsements, Endorsement{Endorser: ident(e), Sig: e.Sig})
	}
	if t.CC == "_lifecycle" {
		tx.ResultsBytes = t.Results // hashed private-data writes (see Tx.ResultsBytes)
	}
	for _, r := range t.RWSets {
		n, err := nsNameLen(r.NS)
		if err != nil {
			return tx, err
		}
		tx.NS = append(tx.NS, NsRWSet{NameLen: n, KVBytes: r.RWSet})
	}
	return tx, nil
}

// LifecycleRead is the read every QChain transaction makes in the _lifecycle
// namespace: the chaincode definition's sequence key
// ("namespaces/fields/qchaincode/Sequence", 37 bytes), committed in block 3.
var LifecycleRead = Read{Key: 37, Exists: true, Block: 3}

// recordTx rebuilds a QChain transaction with its read/write sets computed
// from the keys QChaincode.js reads and writes (calibration level 2): every
// call reads the chaincode definition; registerHolder checks that the holder
// is absent; bindHolderKeys reads the holder; issueCredential reads the holder
// and checks that the new credential ID is absent; the state changes read the
// credential. A present key's version is (block, 0): every captured block has
// one transaction and every block number is below 128, so its size does not
// depend on which earlier block wrote the key.
func recordTx(t capTx) (Tx, error) {
	tx, err := componentTx(t)
	if err != nil {
		return tx, err
	}
	var q capNS
	for _, r := range t.RWSets {
		if r.NS == "qchaincode" {
			q = r
		}
	}
	if len(q.Writes) != 1 {
		return tx, fmt.Errorf("%s at block %d: %d writes", t.Fn, t.Block, len(q.Writes))
	}
	w := Write{Key: q.Writes[0].Key, Value: q.Writes[0].Value.Bytes}
	prior := uint64(t.Block - 1)
	var reads []Read
	switch t.Fn {
	case "registerHolder":
		reads = []Read{{Key: w.Key}}
	case "bindHolderKeys":
		reads = []Read{{Key: w.Key, Exists: true, Block: prior}}
	case "issueCredential":
		reads = []Read{{Key: t.Args[0], Exists: true, Block: prior}, {Key: w.Key}}
	case "updateExpiry", "suspendCredential", "restoreCredential", "revokeCredential":
		reads = []Read{{Key: w.Key, Exists: true, Block: prior}}
	default:
		return tx, fmt.Errorf("no read model for %s", t.Fn)
	}
	if len(reads) != q.Reads {
		return tx, fmt.Errorf("%s at block %d: model reads %d keys, the ledger %d", t.Fn, t.Block, len(reads), q.Reads)
	}
	tx.NS = []NsRWSet{
		{NameLen: 10, Reads: []Read{LifecycleRead}},
		{NameLen: 10, Reads: reads, Writes: []Write{w}},
	}
	return tx, nil
}

// Report is the outcome of Validate.
type Report struct {
	Component int // transactions reproduced from their measured parts (all endorser transactions)
	Record    int // QChain transactions reproduced from keys and record lengths
	Blocks    int // blocks reproduced in the block file's layout
	IPFS      int // IPFS objects reproduced (single blocks and chunked files)
	Certs     int // certificates whose DER parts and PEM size agree
}

func (r Report) String() string {
	return fmt.Sprintf("%d transactions from their parts, %d QChain transactions from keys and record lengths, %d blocks, "+
		"%d IPFS objects and %d certificates reproduced to the byte", r.Component, r.Record, r.Blocks, r.IPFS, r.Certs)
}

// Validate reproduces the capture with the model: every endorser transaction
// from its parts, every QChain transaction from its keys and record length,
// every block's size in the block file, every IPFS object, and every
// certificate's DER and PEM size. It fails on the first byte of difference.
func Validate() (Report, error) {
	var rep Report
	c, err := loadCapture()
	if err != nil {
		return rep, err
	}
	if c.ParsedBytes != c.FileBytes {
		return rep, fmt.Errorf("capture: %d of %d bytes parsed", c.ParsedBytes, c.FileBytes)
	}
	check := func(what string, got, want int) error {
		if got != want {
			return fmt.Errorf("ledger: %s: model %d bytes, ledger %d bytes", what, got, want)
		}
		return nil
	}
	envs := map[int][]int{}
	for _, t := range c.Txs {
		envs[t.Block] = append(envs[t.Block], t.Env)
		if t.Type != 3 {
			continue
		}
		tx, err := componentTx(t)
		if err != nil {
			return rep, err
		}
		s := tx.Sizes()
		what := fmt.Sprintf("%s/%s at block %d", t.CC, t.Fn, t.Block)
		for _, p := range [][3]any{{"envelope", s.Envelope, t.Env}, {"payload", s.Payload, t.Payload},
			{"signature header", s.SigHeader, t.SigHeader}, {"transaction", s.Data, t.Data},
			{"action payload", s.ActionPayload, t.ActionPayload}, {"proposal payload", s.ProposalPayload, t.ProposalPayload},
			{"invocation spec", s.InputSpec, t.InputSpec}, {"proposal response payload", s.PRP, t.PRP},
			{"chaincode action", s.Extension, t.Extension}, {"read/write sets", s.Results, t.Results}} {
			if err := check(what+" "+p[0].(string), p[1].(int), p[2].(int)); err != nil {
				return rep, err
			}
		}
		rep.Component++
		if t.CC != "qchaincode" {
			continue
		}
		rt, err := recordTx(t)
		if err != nil {
			return rep, err
		}
		for k, ns := range rt.NS {
			if err := check(fmt.Sprintf("%s namespace %d", what, k), ns.KV(), t.RWSets[k].RWSet); err != nil {
				return rep, err
			}
		}
		if err := check(what+" envelope from records", rt.Sizes().Envelope, t.Env); err != nil {
			return rep, err
		}
		rep.Record++
	}
	for _, b := range c.Blocks {
		blk := Block{Number: b.Number, DataHash: b.DataHash, PrevHash: b.PreviousHash, Envs: envs[int(b.Number)], Metadata: b.MDLens}
		what := fmt.Sprintf("block %d", b.Number)
		if err := check(what, blk.Frame(), b.Frame); err != nil {
			return rep, err
		}
		if len(b.Orderer) > 0 {
			var sigs []MetaSig
			for _, o := range b.Orderer {
				sigs = append(sigs, MetaSig{Signer: ident(o), Nonce: o.Nonce, Sig: o.Sig, IDHeader: o.IDHeader})
			}
			std := StandardMetadata(b.MD0Value, sigs, b.NTx, Measured.CommitHash)
			if err := check(what+" metadata", sum(std), sum(b.MDLens)); err != nil {
				return rep, err
			}
		}
		rep.Blocks++
	}
	// Certificates: the DER parts add up, and the PEM size follows from the DER size.
	var certs []capIdentity
	for _, t := range c.Txs {
		if t.Type == 3 {
			certs = append(certs, t.Creator)
			certs = append(certs, t.Endorsements...)
		}
	}
	for _, b := range c.Blocks {
		certs = append(certs, b.Orderer...)
	}
	for _, id := range certs {
		if !id.CertLayout.Consistent() {
			return rep, fmt.Errorf("ledger: certificate %+v: DER parts do not add up", id.CertLayout)
		}
		if err := check("PEM of a certificate", PEMLen(id.DER), id.PEM); err != nil {
			return rep, err
		}
		rep.Certs++
	}
	n, err := validateIPFS()
	rep.IPFS = n
	return rep, err
}

// StandardMetadata is the metadata of an ordinary block: SIGNATURES (the
// OrdererBlockMetadata value and the orderer signatures), an empty LAST_CONFIG
// entry, TRANSACTIONS_FILTER (one validation code per transaction), an empty
// ORDERER entry and COMMIT_HASH.
func StandardMetadata(value int, sigs []MetaSig, ntx, commitHash int) []int {
	return []int{SignaturesMetadata(value, sigs), 0, ntx, 0, commitHash}
}

func sum(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

func validateIPFS() (int, error) {
	n := 0
	rows := func(b []byte) ([][]int, error) {
		var out [][]int
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			l := strings.TrimSpace(sc.Text())
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			var r []int
			for _, f := range strings.Fields(l) {
				v, err := strconv.Atoi(f)
				if err != nil {
					return nil, err
				}
				r = append(r, v)
			}
			out = append(out, r)
		}
		return out, sc.Err()
	}
	pins, err := rows(ipfsPinsTxt)
	if err != nil {
		return n, err
	}
	for _, r := range pins { // count, file bytes, root block bytes, child blocks
		file, root, children := r[1], r[2], r[3]
		if file == 0 {
			continue // the empty directory
		}
		bl, err := IPFSBlocks(file)
		if err != nil {
			return n, err
		}
		if bl[0] != root || len(bl)-1 != children {
			return n, fmt.Errorf("ledger: IPFS object of %d bytes: model root %d and %d children, store %d and %d",
				file, bl[0], len(bl)-1, root, children)
		}
		n++
	}
	links, err := rows(ipfsLinkTxt)
	if err != nil {
		return n, err
	}
	for _, r := range links { // block, attributes, file bytes, root block bytes, child blocks
		bl, err := IPFSBlocks(r[2])
		if err != nil {
			return n, err
		}
		if bl[0] != r[3] || len(bl)-1 != r[4] {
			return n, fmt.Errorf("ledger: envelope of block %d: model %d, store %d", r[0], bl[0], r[3])
		}
		n++
	}
	return n, nil
}

// medianLow is the lower median (an element of xs).
func medianLow(xs []int) int {
	s := append([]int(nil), xs...)
	sort.Ints(s)
	return s[(len(s)-1)/2]
}
