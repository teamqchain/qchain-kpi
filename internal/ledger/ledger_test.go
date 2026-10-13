package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

func TestCaptureIsTheVerifiedExport(t *testing.T) {
	sum := sha256.Sum256(captureJSON)
	if got := hex.EncodeToString(sum[:]); got != CaptureSHA256 {
		t.Fatalf("vmcapture/capture-export.json changed: SHA-256 %s, want %s", got, CaptureSHA256)
	}
}

// TestValidate: the model reproduces the live ledger and IPFS store to the
// byte (the calibration E13 requires before any other algorithm is computed).
func TestValidate(t *testing.T) {
	rep, err := Validate()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rep)
	// 60 endorser transactions (57 QChain + 3 chaincode lifecycle), 61 blocks,
	// 55 non-empty pinned objects (one of them a six-chunk file) plus the 16
	// envelopes linked to the ledger, and 238 certificates (submitter and
	// endorsers of every transaction, the orderer of every block).
	want := Report{Component: 60, Record: 57, Blocks: 61, IPFS: 55 + 16, Certs: 238}
	if rep != want {
		t.Fatalf("report %+v, want %+v", rep, want)
	}
}

func TestProfileMatchesCapture(t *testing.T) {
	p, err := DeriveProfile()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p, Measured) {
		t.Fatalf("Measured does not match the capture:\n got  %+v\n want %+v", Measured, p)
	}
}

// TestMeasuredScenarioReproducesIssuance: a credential call assembled from the
// profile (not from per-transaction measurements) lands on a real envelope
// size when given that transaction's arguments, record and channel header.
func TestMeasuredScenarioReproducesIssuance(t *testing.T) {
	c, err := loadCapture()
	if err != nil {
		t.Fatal(err)
	}
	f := MeasuredFabric(Measured)
	n := 0
	for _, x := range c.Txs {
		if x.Fn != "issueCredential" {
			continue
		}
		p := Measured
		p.ChannelHeader = x.ChannelHeader
		w := x.RWSets[1].Writes[0]
		call := Call{Fn: x.Fn, Args: x.Args, RespPayload: x.RespPayload,
			Reads:  []Read{{Key: x.Args[0], Exists: true, Block: uint64(x.Block - 1)}, {Key: w.Key}},
			Writes: []Write{{Key: w.Key, Value: w.Value.Bytes}}}
		tx := f.Tx(p, call)
		tx.ClientSig = x.ClientSig
		for k := range tx.Endorsements {
			tx.Endorsements[k].Sig = x.Endorsements[k].Sig
		}
		if got := tx.Sizes().Envelope; got != x.Env {
			t.Fatalf("issueCredential at block %d: scenario %d, ledger %d", x.Block, got, x.Env)
		}
		n++
	}
	if n != 16 {
		t.Fatalf("%d credentials", n)
	}
}

func TestProjection(t *testing.T) {
	for _, m := range MLDSA {
		for _, c := range []CertIdentity{Measured.Creator, Measured.Orderer, Measured.Endorsers[0], Measured.Endorsers[1]} {
			pc := c.Cert.WithMLDSA(m.PK, m.Sig)
			if !pc.Consistent() {
				t.Fatalf("%s: projected certificate %+v does not add up", m.Name, pc)
			}
			// The key and signature replace ECDSA's 65-byte point and ~71-byte signature.
			grow := pc.DER - c.Cert.DER
			if grow < m.PK+m.Sig-65-72 || grow > m.PK+m.Sig-65-70+20 {
				t.Fatalf("%s: certificate grows by %d bytes", m.Name, grow)
			}
		}
	}
	f := ProjectedFabric("ml-dsa-44", Measured, 1312, 2420)
	if f.Sig != 2420 || f.Creator.PEM <= Measured.Creator.PEM+4000 {
		t.Fatalf("projection %+v", f)
	}
}

func TestIPFSBlocks(t *testing.T) {
	for _, c := range []struct{ file, root int }{{14, 22}, {120, 128}, {128, 139}, {7564, 7575}, {146201, 146215}} {
		bl, _ := IPFSBlocks(c.file)
		if len(bl) != 1 || bl[0] != c.root {
			t.Fatalf("%d bytes: %v, want one block of %d", c.file, bl, c.root)
		}
	}
	bl, err := IPFSBlocks(1335135)
	if err != nil || bl[0] != 296 || len(bl) != 7 {
		t.Fatalf("six-chunk file: %v %v", bl, err)
	}
	if OnDisk([]int{4097, 10}, 4096) != 3*4096 {
		t.Fatal("OnDisk")
	}
}

func TestMaxPerBlock(t *testing.T) {
	for _, c := range []struct{ msg, n int }{{21000, 10}, {60000, 8}, {130000, 4}, {600000, 1}} {
		if got := MaxPerBlock(c.msg); got != c.n {
			t.Fatalf("MaxPerBlock(%d) = %d, want %d", c.msg, got, c.n)
		}
	}
}
