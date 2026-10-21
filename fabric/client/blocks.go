package main

// Reading the ledger's blocks (qscc GetBlockByNumber): how many transactions
// each block holds, the size of every transaction's envelope (the bytes the
// orderer counts against PreferredMaxBytes) and of the whole block, and each
// transaction's validation code.

import (
	"fmt"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"google.golang.org/protobuf/proto"
)

// LedgerBlock is one block as read back from a peer.
type LedgerBlock struct {
	Number uint64
	Bytes  int // the marshalled block
	Txs    []BlockTx
}

// BlockTx is one transaction in a block.
type BlockTx struct {
	TxID          string
	EnvelopeBytes int   // the marshalled envelope, as stored in the block
	OrdererBytes  int   // len(payload) + len(signature): what the orderer's block cutter counts
	Code          int32 // validation code from the block metadata (0 = VALID)
}

// ParseBlock decodes a marshalled common.Block.
func ParseBlock(raw []byte) (*LedgerBlock, error) {
	var b common.Block
	if err := proto.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("block: %w", err)
	}
	if b.Header == nil || b.Data == nil {
		return nil, fmt.Errorf("block without header or data")
	}
	var filter []byte
	if b.Metadata != nil && len(b.Metadata.Metadata) > int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		filter = b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	}
	out := &LedgerBlock{Number: b.Header.Number, Bytes: len(raw)}
	for i, envBytes := range b.Data.Data {
		var env common.Envelope
		if err := proto.Unmarshal(envBytes, &env); err != nil {
			return nil, fmt.Errorf("block %d envelope %d: %w", b.Header.Number, i, err)
		}
		var pl common.Payload
		if err := proto.Unmarshal(env.Payload, &pl); err != nil {
			return nil, fmt.Errorf("block %d payload %d: %w", b.Header.Number, i, err)
		}
		var ch common.ChannelHeader
		if pl.Header != nil {
			if err := proto.Unmarshal(pl.Header.ChannelHeader, &ch); err != nil {
				return nil, fmt.Errorf("block %d channel header %d: %w", b.Header.Number, i, err)
			}
		}
		code := int32(-1)
		if i < len(filter) {
			code = int32(filter[i])
		}
		out.Txs = append(out.Txs, BlockTx{TxID: ch.TxId, EnvelopeBytes: len(envBytes), OrdererBytes: len(env.Payload) + len(env.Signature), Code: code})
	}
	return out, nil
}

// OrdererSum is the size the orderer's block cutter counted for the block
// (fabric orderer/common/blockcutter: messageSizeBytes = payload + signature).
func (b *LedgerBlock) OrdererSum() int {
	s := 0
	for _, t := range b.Txs {
		s += t.OrdererBytes
	}
	return s
}

// Orderer batch settings of QChain's channel (configtx.yaml).
const (
	MaxMessageCount   = 10
	PreferredMaxBytes = 512 * 1024
)

// CutReason infers why the orderer closed a block of this run's transactions:
// "count" (MaxMessageCount reached), "size" (one more transaction of the next
// one's size would pass PreferredMaxBytes) or "timeout" (the 2 s batch
// timer). next is the orderer size of the transaction that followed in the
// same burst, 0 if none did.
func CutReason(b *LedgerBlock, next int) string {
	switch {
	case len(b.Txs) >= MaxMessageCount:
		return "count"
	case next > 0 && b.OrdererSum()+next > PreferredMaxBytes:
		return "size"
	default:
		return "timeout"
	}
}
