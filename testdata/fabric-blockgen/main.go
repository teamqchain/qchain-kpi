// fabric-blockgen writes a synthetic Fabric block file with the real Fabric protobuf
// types (and real X.509 certificates), plus the sizes it expects a reader to find.
// It exists only to test scripts/e13-block-sizes.py. Every value is made up.
//
//	go run . OUT_DIR      -> OUT_DIR/synthetic.blockfile, OUT_DIR/synthetic.expected.jsonl
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func marshal(m proto.Message) []byte { return must(proto.Marshal(m)) }

func rnd(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func certPEM(cn, org string) []byte {
	k := must(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{org}, OrganizationalUnit: []string{"client"}},
		NotBefore:    time.Now(), NotAfter: time.Now().Add(24 * 365 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	der := must(x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k))
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

type expTx struct {
	Block, Idx int
	Fn         string
	Env, Payload, Pem, NEnd, Results, ValueBytes, KeyLen int
	Args       []int
}

func identity(mspid string, pemb []byte) []byte {
	return marshal(&msp.SerializedIdentity{Mspid: mspid, IdBytes: pemb})
}

func buildTx(cc, fn string, args [][]byte, writes []*kvrwset.KVWrite, creatorMsp string, endorsers []string, creatorPEM []byte, endPEM map[string][]byte) ([]byte, expTx) {
	kv := marshal(&kvrwset.KVRWSet{Writes: writes})
	results := marshal(&rwset.TxReadWriteSet{DataModel: rwset.TxReadWriteSet_KV,
		NsRwset: []*rwset.NsReadWriteSet{{Namespace: cc, Rwset: kv}}})
	ca := marshal(&peer.ChaincodeAction{Results: results, Response: &peer.Response{Status: 200, Payload: []byte(`{"success":true}`)},
		ChaincodeId: &peer.ChaincodeID{Name: cc, Version: "2.0"}})
	prp := marshal(&peer.ProposalResponsePayload{ProposalHash: rnd(32), Extension: ca})
	cis := marshal(&peer.ChaincodeInvocationSpec{ChaincodeSpec: &peer.ChaincodeSpec{Type: peer.ChaincodeSpec_NODE,
		ChaincodeId: &peer.ChaincodeID{Name: cc}, Input: &peer.ChaincodeInput{Args: append([][]byte{[]byte(fn)}, args...)}}})
	var ends []*peer.Endorsement
	for _, m := range endorsers {
		ends = append(ends, &peer.Endorsement{Endorser: identity(m, endPEM[m]), Signature: rnd(71)})
	}
	cap := marshal(&peer.ChaincodeActionPayload{ChaincodeProposalPayload: marshal(&peer.ChaincodeProposalPayload{Input: cis}),
		Action: &peer.ChaincodeEndorsedAction{ProposalResponsePayload: prp, Endorsements: ends}})
	shdr := marshal(&common.SignatureHeader{Creator: identity(creatorMsp, creatorPEM), Nonce: rnd(24)})
	tx := marshal(&peer.Transaction{Actions: []*peer.TransactionAction{{Header: shdr, Payload: cap}}})
	chdr := marshal(&common.ChannelHeader{Type: int32(common.HeaderType_ENDORSER_TRANSACTION), Version: 1, Timestamp: timestamppb.Now(),
		ChannelId: "mychannel", TxId: hex.EncodeToString(rnd(32)),
		Extension: marshal(&peer.ChaincodeHeaderExtension{ChaincodeId: &peer.ChaincodeID{Name: cc}})})
	payload := marshal(&common.Payload{Header: &common.Header{ChannelHeader: chdr, SignatureHeader: shdr}, Data: tx})
	env := marshal(&common.Envelope{Payload: payload, Signature: rnd(72)})
	e := expTx{Fn: fn, Env: len(env), Payload: len(payload), Pem: len(creatorPEM), NEnd: len(endorsers), Results: len(results)}
	for _, a := range args {
		e.Args = append(e.Args, len(a))
	}
	if len(writes) > 0 {
		e.ValueBytes, e.KeyLen = len(writes[0].Value), len(writes[0].Key)
	}
	return env, e
}

func block(num uint64, envs [][]byte, codes []byte, ordMsp string, ordPEM []byte) []byte {
	shdr := marshal(&common.SignatureHeader{Creator: identity(ordMsp, ordPEM), Nonce: rnd(24)})
	b := &common.Block{
		Header: &common.BlockHeader{Number: num, PreviousHash: rnd(32), DataHash: rnd(32)},
		Data:   &common.BlockData{Data: envs},
		Metadata: &common.BlockMetadata{Metadata: [][]byte{
			marshal(&common.Metadata{Value: nil, Signatures: []*common.MetadataSignature{{SignatureHeader: shdr, Signature: rnd(71)}}}),
			marshal(&common.Metadata{Value: marshal(&common.LastConfig{Index: 0})}),
			codes, nil, rnd(32),
		}},
	}
	return serialize(b)
}

// serialize is the layout a peer's block file uses (Fabric v2.5.15,
// common/ledger/blkstorage/block_serialization.go): number, data hash, previous
// hash, transaction count, each envelope, metadata count, each metadata entry,
// every byte string with its own length prefix. It is not the Block protobuf.
func serialize(b *common.Block) []byte {
	var o []byte
	raw := func(x []byte) { o = binary.AppendUvarint(o, uint64(len(x))); o = append(o, x...) }
	o = binary.AppendUvarint(o, b.Header.Number)
	raw(b.Header.DataHash)
	raw(b.Header.PreviousHash)
	o = binary.AppendUvarint(o, uint64(len(b.Data.Data)))
	for _, e := range b.Data.Data {
		raw(e)
	}
	o = binary.AppendUvarint(o, uint64(len(b.Metadata.Metadata)))
	for _, m := range b.Metadata.Metadata {
		raw(m)
	}
	return o
}

func main() {
	out := os.Args[1]
	gen, gov, ord := certPEM("user1", "general.uae.com"), certPEM("peer0", "government.uae.com"), certPEM("orderer0", "orderer.example.com")
	endPEM := map[string][]byte{"GeneralMSP": certPEM("peer0", "general.uae.com"), "GovernmentMSP": gov}
	hexOf := func(n int) string { return hex.EncodeToString(rnd(n)) }
	record := func(dt string, fields map[string]any) []byte {
		fields["DocType"] = dt
		return must(json.Marshal(fields)) // Go sorts map keys, as the chaincode's stored JSON does
	}
	var blob []byte
	var exp []expTx
	add := func(b []byte) { blob = binary.AppendUvarint(blob, uint64(len(b))); blob = append(blob, b...) }
	pushBlock := func(num int, items []struct {
		env []byte
		e   expTx
	}, codes []byte) {
		var envs [][]byte
		for i, it := range items {
			envs = append(envs, it.env)
			it.e.Block, it.e.Idx = num, i
			exp = append(exp, it.e)
		}
		add(block(uint64(num), envs, codes, "OrdererMSP", ord))
	}
	type item = struct {
		env []byte
		e   expTx
	}
	mk := func(fn string, args [][]byte, key string, val []byte) item {
		env, e := buildTx("qchaincode", fn, args, []*kvrwset.KVWrite{{Key: key, Value: val}}, "GeneralMSP", []string{"GeneralMSP", "GovernmentMSP"}, gen, endPEM)
		return item{env, e}
	}
	// block 0: a configuration transaction (type 1), no chaincode
	cfgPayload := marshal(&common.Payload{Header: &common.Header{ChannelHeader: marshal(&common.ChannelHeader{Type: int32(common.HeaderType_CONFIG), ChannelId: "mychannel"}),
		SignatureHeader: marshal(&common.SignatureHeader{Creator: identity("OrdererMSP", ord), Nonce: rnd(24)})}, Data: rnd(900)})
	add(block(0, [][]byte{marshal(&common.Envelope{Payload: cfgPayload, Signature: rnd(72)})}, []byte{0}, "OrdererMSP", ord))
	// block 1: a holder is registered, its keys are bound
	holder := record("holder", map[string]any{"ID": "H-0001", "FirstName": "Aaaa", "LastName": "Bbbbbb"})
	holder2 := record("holder", map[string]any{"ID": "H-0001", "FirstName": "Aaaa", "LastName": "Bbbbbb", "KemPublicKey": hexOf(1184), "DsaPublicKey": hexOf(1312)})
	pushBlock(1, []item{
		mk("registerHolder", [][]byte{[]byte("H-0001"), []byte("Aaaa"), []byte("Bbbbbb")}, "H-0001", holder),
		mk("bindHolderKeys", [][]byte{[]byte("H-0001"), []byte(hexOf(1184)), []byte(hexOf(1312))}, "H-0001", holder2),
	}, []byte{0, 0})
	// blocks 2 and 3: credentials with 10 attributes (ML-DSA-44 sized), one invalid transaction
	cred := func() item {
		fh := map[string]string{}
		for i := 1; i <= 10; i++ {
			fh[fmt.Sprintf("field%02d", i)] = hexOf(32)
		}
		sig, pk := hexOf(2420), hexOf(1312)
		val := record("credential", map[string]any{"CommitmentVersion": 2, "ID": "CRED-" + hexOf(32), "Holder": "H-0001",
			"Issuer": "x509::CN=issuer1,OU=client,O=Hyperledger::CN=ca.general.uae.com,O=general.uae.com", "IssuerOrgID": "GeneralMSP",
			"Status": "active", "CredentialType": "Bachelor of Science", "IssuedAt": "2026-10-12T10:00:00+04:00", "ExpiryDate": "2030-06-30",
			"CID": "Qm" + strings.Repeat("a", 44), "FieldHashes": fh, "CredentialHash": hexOf(32), "Signature": sig, "PublicKey": pk})
		fhj := must(json.Marshal(fh))
		return mk("issueCredential", [][]byte{[]byte("H-0001"), []byte("Bachelor of Science"), []byte("2026-10-12T10:00:00+04:00"), []byte("2030-06-30"),
			[]byte("GeneralMSP"), []byte("Qm" + strings.Repeat("a", 44)), fhj, []byte(hexOf(32)), []byte(sig), []byte(pk)}, "CRED-"+hexOf(32), val)
	}
	pushBlock(2, []item{cred(), cred(), cred()}, []byte{0, 0, 0})
	pushBlock(3, []item{cred(), cred()}, []byte{0, 11})
	must(0, os.WriteFile(filepath.Join(out, "synthetic.blockfile"), blob, 0o644))
	var sb strings.Builder
	for _, e := range exp {
		sb.Write(must(json.Marshal(e)))
		sb.WriteString("\n")
	}
	must(0, os.WriteFile(filepath.Join(out, "synthetic.expected.jsonl"), []byte(sb.String()), 0o644))
	fmt.Printf("wrote %d bytes, %d transactions\n", len(blob), len(exp))
}
