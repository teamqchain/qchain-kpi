// Package ledger is E13's size model of QChain on Hyperledger Fabric v2.5 and
// IPFS. It computes, to the byte, how large a transaction envelope, a stored
// block and an IPFS object are, from the sizes of their parts. Every rule here
// is checked against the read-only capture of the live ledger in vmcapture/
// (Validate): no size is estimated where Fabric's encoding determines it.
//
// The model has no cryptography and no cgo. The parts that depend on the
// algorithm (signatures, keys, records, envelopes) are produced by QChain's
// real code in cmd/kpi-prim and passed in as lengths.
package ledger

// ---------------------------------------------------------------- protobuf framing

func varintLen(x uint64) int {
	n := 1
	for x >= 0x80 {
		x >>= 7
		n++
	}
	return n
}

func vlen(n int) int { return varintLen(uint64(n)) }

// ld is a length-delimited field (bytes, string or message) that is always
// encoded: tag, length, content. Repeated elements and set messages are
// always encoded, even when empty.
func ld(field, n int) int { return varintLen(uint64(field<<3)) + vlen(n) + n }

// opt is a proto3 scalar bytes or string field, omitted when empty.
func opt(field, n int) int {
	if n == 0 {
		return 0
	}
	return ld(field, n)
}

// vf is a proto3 varint field, omitted when zero.
func vf(field int, x uint64) int {
	if x == 0 {
		return 0
	}
	return varintLen(uint64(field<<3)) + varintLen(x)
}

// ---------------------------------------------------------------- messages

// Identity is a Fabric msp.SerializedIdentity: the MSP ID and the PEM
// certificate.
type Identity struct {
	MSPLen int `json:"msp_len"`
	PEM    int `json:"pem"`
}

// Serialized is the identity's encoded size.
func (id Identity) Serialized() int { return opt(1, id.MSPLen) + opt(2, id.PEM) }

// Endorsement is one peer's endorsement: its identity and signature.
type Endorsement struct {
	Endorser Identity
	Sig      int
}

func (e Endorsement) size() int { return ld(1, e.Endorser.Serialized()) + opt(2, e.Sig) }

// Read is one key read by a transaction. Exists marks a key that was present:
// Fabric then records its version (block and transaction number); a read of
// an absent key has no version.
type Read struct {
	Key    int
	Exists bool
	Block  uint64
	TxNum  uint64
}

func (r Read) size() int {
	s := ld(1, r.Key)
	if r.Exists {
		s += ld(2, vf(1, r.Block)+vf(2, r.TxNum))
	}
	return s
}

// Write is one key written (Value is the stored record's length).
type Write struct {
	Key    int
	Value  int
	Delete bool
}

func (w Write) size() int {
	s := ld(1, w.Key) + opt(3, w.Value)
	if w.Delete {
		s += 2
	}
	return s
}

// NsRWSet is the read/write set of one namespace (chaincode). KVBytes, when
// positive, is the namespace's measured read/write set size used as it is
// (calibration at the component level); otherwise it is computed from the
// reads and writes.
type NsRWSet struct {
	NameLen int
	Reads   []Read
	Writes  []Write
	KVBytes int
}

// KV is the size of the namespace's KVRWSet message.
func (n NsRWSet) KV() int {
	if n.KVBytes > 0 {
		return n.KVBytes
	}
	s := 0
	for _, r := range n.Reads {
		s += ld(1, r.size())
	}
	for _, w := range n.Writes {
		s += ld(3, w.size())
	}
	return s
}

func (n NsRWSet) size() int { return opt(1, n.NameLen) + ld(2, n.KV()) }

// Tx is an endorser transaction as Fabric v2.5 stores it in a block: an
// Envelope{Payload{Header{ChannelHeader, SignatureHeader}, Transaction{
// TransactionAction{SignatureHeader, ChaincodeActionPayload{
// ChaincodeProposalPayload{ChaincodeInvocationSpec{ChaincodeSpec{ChaincodeID,
// ChaincodeInput{args}}}}, ChaincodeEndorsedAction{ProposalResponsePayload{
// proposal hash, ChaincodeAction{TxReadWriteSet, events, Response,
// ChaincodeID}}, Endorsements}}}}}, client signature}.
type Tx struct {
	ChannelHeader int // the ChannelHeader message (type, timestamp, channel, tx ID, extension)
	Creator       Identity
	Nonce         int
	ClientSig     int
	CCNameLen     int   // chaincode name in the invocation
	Args          []int // chaincode arguments, the function name first
	Endorsements  []Endorsement
	ProposalHash  int
	NS            []NsRWSet
	Events        int
	RespStatus    uint64
	RespMessage   int
	RespPayload   int
	CCIDAction    int // the ChaincodeID message in the action (name and version)
	// ResultsBytes, when positive, is the measured TxReadWriteSet size used as
	// it is. Only the chaincode-lifecycle transactions need it: they also write
	// hashed private data (each organisation's approval collection), which the
	// capture does not break down. QChain's own calls have no private data.
	ResultsBytes int
}

// TxSizes are the encoded sizes of a transaction and its main parts.
type TxSizes struct {
	Envelope        int
	Payload         int
	SigHeader       int
	Data            int
	ActionPayload   int
	ProposalPayload int
	InputSpec       int
	PRP             int
	Extension       int
	Results         int
	Endorsements    int // all endorsements, as encoded inside the action
	Message         int // payload + client signature: what the orderer's block cutter counts
}

// Sizes encodes the transaction's layout.
func (t Tx) Sizes() TxSizes {
	var s TxSizes
	results := 0
	for _, n := range t.NS {
		results += ld(2, n.size())
	}
	if t.ResultsBytes > 0 {
		results = t.ResultsBytes
	}
	resp := vf(1, t.RespStatus) + opt(2, t.RespMessage) + opt(3, t.RespPayload)
	s.Results = results
	s.Extension = opt(1, results) + opt(2, t.Events) + ld(3, resp) + opt(4, t.CCIDAction)
	s.PRP = opt(1, t.ProposalHash) + opt(2, s.Extension)
	cea := ld(1, s.PRP)
	for _, e := range t.Endorsements {
		cea += ld(2, e.size())
		s.Endorsements += ld(2, e.size())
	}
	input := 0
	for _, a := range t.Args {
		input += ld(1, a)
	}
	spec := ld(2, opt(2, t.CCNameLen)) + ld(3, input)
	s.InputSpec = ld(1, spec)
	s.ProposalPayload = ld(1, s.InputSpec)
	s.ActionPayload = ld(1, s.ProposalPayload) + ld(2, cea)
	s.SigHeader = ld(1, t.Creator.Serialized()) + opt(2, t.Nonce)
	ta := ld(1, s.SigHeader) + ld(2, s.ActionPayload)
	s.Data = ld(1, ta)
	header := ld(1, t.ChannelHeader) + ld(2, s.SigHeader)
	s.Payload = ld(1, header) + ld(2, s.Data)
	s.Envelope = ld(1, s.Payload) + opt(2, t.ClientSig)
	s.Message = s.Payload + t.ClientSig
	return s
}

// ---------------------------------------------------------------- blocks

// MetaSig is one orderer signature in a block's SIGNATURES metadata.
type MetaSig struct {
	Signer   Identity
	Nonce    int
	Sig      int
	IDHeader int
}

// SignaturesMetadata is the size of the SIGNATURES metadata entry: the
// OrdererBlockMetadata value and the orderer signatures.
func SignaturesMetadata(value int, sigs []MetaSig) int {
	s := opt(1, value)
	for _, m := range sigs {
		shdr := ld(1, m.Signer.Serialized()) + opt(2, m.Nonce)
		s += ld(2, ld(1, shdr)+opt(2, m.Sig)+opt(3, m.IDHeader))
	}
	return s
}

// Block is a block as a peer stores it in its block file (Fabric v2.5.15,
// common/ledger/blkstorage/block_serialization.go): block number, data hash,
// previous hash, transaction count, each envelope, metadata count, each
// metadata entry, every byte string with its own length prefix, the whole
// block preceded by its length. It is not the Block protobuf.
type Block struct {
	Number   uint64
	DataHash int
	PrevHash int
	Envs     []int
	Metadata []int
}

// Stored is the block's serialized length (without the length prefix).
func (b Block) Stored() int {
	s := varintLen(b.Number) + vlen(b.DataHash) + b.DataHash + vlen(b.PrevHash) + b.PrevHash + vlen(len(b.Envs))
	for _, e := range b.Envs {
		s += vlen(e) + e
	}
	s += vlen(len(b.Metadata))
	for _, m := range b.Metadata {
		s += vlen(m) + m
	}
	return s
}

// Frame is the block's size in the block file, length prefix included.
func (b Block) Frame() int {
	n := b.Stored()
	return vlen(n) + n
}

// ---------------------------------------------------------------- batching

// Fabric's orderer batch settings for QChain's channel (qchain-network/config/
// configtx.yaml @ 9026aba): BatchTimeout 2 s, MaxMessageCount 10,
// PreferredMaxBytes 512 KB.
const (
	MaxMessageCount   = 10
	PreferredMaxBytes = 512 * 1024
)

// MaxPerBlock is how many identical transactions of message size msg the
// block cutter puts in one block when they arrive faster than the batch
// timeout: at most MaxMessageCount, and no more than fit in PreferredMaxBytes
// (a single larger message gets a block of its own).
func MaxPerBlock(msg int) int {
	n := PreferredMaxBytes / msg
	if n < 1 {
		return 1
	}
	if n > MaxMessageCount {
		return MaxMessageCount
	}
	return n
}
