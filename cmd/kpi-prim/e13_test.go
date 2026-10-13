package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// e13Value returns the p50 column of the first row matching the prefix
// "scenario,tx,kem,n,v,quantity,".
func e13Value(t *testing.T, rows []string, prefix string) (lo, p50, hi int) {
	t.Helper()
	for _, r := range rows {
		if strings.HasPrefix(r, prefix) {
			f := strings.Split(r[len(prefix):], ",")
			lo, _ = strconv.Atoi(f[0])
			p50, _ = strconv.Atoi(f[1])
			hi, _ = strconv.Atoi(f[2])
			return
		}
	}
	t.Fatalf("no row %q", prefix)
	return
}

// TestE13Blocks: real credentials and envelopes go through the validated
// ledger model; the production suite lands on the captured sizes, the
// scenarios move in the expected direction, variable signatures are sampled,
// and envelopes are stored as IPFS would store them.
func TestE13Blocks(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	run := func(alg, op string) (proto.Result, map[string]string, []string) {
		b := plan.Block{ID: "E13." + alg + "." + op + ".r1", Exp: "E13", Alg: alg, Op: op, Rep: 1, Attempt: 1,
			Profile: plan.ProfileSmoke, Mode: "sizes", Seed: 7}
		raw := filepath.Join(out, b.RawPath())
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		h, rows := rawRows(t, raw)
		if r.Rows != len(rows) || !strings.Contains(h["validation"], "61 blocks") {
			t.Fatalf("%s: %d rows, header %v", b.Key(), len(rows), h["validation"])
		}
		if got, _ := fileSHA256(raw); got != r.RawSHA256 {
			t.Fatalf("%s: raw checksum mismatch", b.Key())
		}
		return r, h, rows
	}

	// Production signature scheme: fixed lengths, five attribute counts.
	r, _, rows := run("ml-dsa-44", "ledger")
	if len(r.Arms) != 5 || r.Arms[0].MinBytes != r.Arms[0].MaxBytes {
		t.Fatalf("ml-dsa-44 arms %+v", r.Arms)
	}
	_, issue5, _ := e13Value(t, rows, "measured,issueCredential,,5,16,envelope,")
	// The 14 real five-attribute credentials: 21,497-21,570 bytes. E11's
	// credential differs slightly (type, expiry set, field names, height).
	if issue5 < 21300 || issue5 > 21800 {
		t.Fatalf("issueCredential, 5 attributes: %d bytes", issue5)
	}
	_, frame, _ := e13Value(t, rows, "measured,issueCredential,,5,16,frame_1,")
	if frame-issue5 < 1300 || frame-issue5 > 1400 {
		t.Fatalf("block overhead %d bytes", frame-issue5)
	}
	_, perBlock, _ := e13Value(t, rows, "measured,issueCredential,,5,16,per_block_max,")
	if perBlock != 10 {
		t.Fatalf("%d credentials per block", perBlock)
	}
	_, proj, _ := e13Value(t, rows, "projected-ml-dsa-44,issueCredential,,5,16,envelope,")
	_, b64, _ := e13Value(t, rows, "whatif-base64,issueCredential,,5,16,envelope,")
	_, ref, _ := e13Value(t, rows, "whatif-keyref,issueCredential,,5,16,envelope,")
	_, both, _ := e13Value(t, rows, "whatif-both,issueCredential,,5,16,envelope,")
	if !(proj > issue5+15000 && b64 < issue5 && ref < issue5 && both < b64 && both < ref) {
		t.Fatalf("scenarios: measured %d projected %d base64 %d keyref %d both %d", issue5, proj, b64, ref, both)
	}
	_, bind, _ := e13Value(t, rows, "measured,bindHolderKeys,ml-kem-768,0,16,envelope,")
	if bind < 14700 || bind > 15100 { // eight real bindings: 14,879-14,885 bytes
		t.Fatalf("bindHolderKeys ML-KEM-768 + ML-DSA-44: %d bytes", bind)
	}
	_, reg, _ := e13Value(t, rows, "measured,registerHolder,,0,16,envelope,")
	if reg < 4880 || reg > 4930 { // ten real registrations: 4,892-4,910 bytes
		t.Fatalf("registerHolder: %d bytes", reg)
	}

	// Variable-length signatures are sampled.
	r, _, rows = run("falcon-512", "ledger")
	lo, _, hi := e13Value(t, rows, "measured,issueCredential,,10,16,envelope,")
	if !(lo < hi) || r.KeysChecked < 50 {
		t.Fatalf("falcon-512: envelopes %d..%d, %d signatures checked", lo, hi, r.KeysChecked)
	}

	// IPFS: the production KEM at five attributes (real envelopes: 4,274-4,424 bytes).
	_, _, rows = run("ml-kem-768", "ipfs")
	_, file, _ := e13Value(t, rows, "measured,envelope,ml-kem-768,5,16,file,")
	_, stored, _ := e13Value(t, rows, "measured,envelope,ml-kem-768,5,16,stored,")
	if file < 4000 || file > 4700 || stored != file+11 {
		t.Fatalf("ML-KEM-768 envelope, 5 attributes: file %d, stored %d", file, stored)
	}
	_, _, rows = run("hqc-5", "ipfs")
	_, blocks, _ := e13Value(t, rows, "measured,envelope,hqc-5,64,4096,blocks,")
	if blocks < 2 {
		t.Fatalf("HQC-5, 64 attributes of 4 KB: %d IPFS blocks", blocks)
	}

	b := plan.Block{ID: "E13.ml-dsa-44.ipfs.r1", Exp: "E13", Alg: "ml-dsa-44", Op: "ipfs", Rep: 1, Attempt: 1, Profile: plan.ProfileSmoke, Seed: 7}
	if _, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())}); err == nil {
		t.Fatal("E13 ipfs for a signature scheme must be refused")
	}
}
