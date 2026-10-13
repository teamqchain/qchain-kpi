package main

import (
	"path/filepath"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// TestE10Block: three interleaved arms, each with its own pool.
func TestE10Block(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	if pr := prep(t, dir, "ml-dsa-44", 1, 8, true); !pr.E10 {
		t.Fatalf("prep %+v", pr)
	}
	b := plan.Block{ID: "E10.ml-dsa-44.sign.r1", Exp: "E10", Alg: "ml-dsa-44", Op: "sign", Rep: 1, Attempt: 1,
		Profile: plan.ProfileSmoke, Mode: "fast", Seed: 9}
	r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Arms) != 3 || r.Arms[0].Name != proto.ArmIssuer || r.Arms[1].Name != proto.ArmHolder ||
		r.Arms[2].Name != proto.ArmIssuerAgain || r.Arms[0].FirstLabel != "first" {
		t.Fatalf("E10 arms %+v", r.Arms)
	}
	_, rows := rawRows(t, filepath.Join(out, b.RawPath()))
	if len(rows) != 3*201 {
		t.Fatalf("E10 raw rows %d", len(rows))
	}
}
