package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
)

// TestE12Blocks: real E12 blocks (smoke budgets) report the four quantities,
// and the raw file's per-second series, per-worker counts and histogram all
// add up to the calls counted in the window.
func TestE12Blocks(t *testing.T) {
	out := t.TempDir()
	dir := t.TempDir()
	for _, c := range []struct {
		alg, stage string
		w          int
	}{
		{"ml-dsa-44+ml-kem-768", "verification-n10-k5-v16", 2},
		{"ecdsa-p256+dhkem-p256", "issuance-n10-v16", 1},
	} {
		op := plan.E12Op(c.stage, c.w)
		b := plan.Block{ID: "E12." + c.alg + "." + op + ".r1", Exp: "E12", Alg: c.alg, Op: op, Rep: 1, Attempt: 1,
			Profile: plan.ProfileSmoke, Mode: "window", Seed: 12}
		raw := filepath.Join(out, b.RawPath())
		r, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: raw})
		if err != nil {
			t.Fatalf("%s: %v", b.Key(), err)
		}
		if r.Budget.Mode != "window" || r.GOMAXPROCS != c.w || len(r.Arms) != len(proto.E12Arms) {
			t.Fatalf("%s: budget %+v, GOMAXPROCS %d, arms %+v", b.Key(), r.Budget, r.GOMAXPROCS, r.Arms)
		}
		for k, a := range r.Arms {
			if a.Name != proto.E12Arms[k] {
				t.Fatalf("%s: quantity %d is %q", b.Key(), k, a.Name)
			}
		}
		lat, tp := r.Arms[0].Stats, r.Arms[1].Stats
		if lat.N < 10 || !(lat.Min <= lat.P50 && lat.P50 <= lat.P99 && lat.P99 <= lat.Max) || tp.N != 2 {
			t.Fatalf("%s: latency %+v throughput %+v", b.Key(), lat, tp)
		}
		h, rows := rawRows(t, raw)
		sums := map[string]int64{}
		workers := 0
		for _, row := range rows {
			f := strings.Split(row, ",")
			n, _ := strconv.ParseInt(f[2], 10, 64)
			sums[f[0]] += n
			if f[0] == "worker" {
				workers++
				if n == 0 {
					t.Fatalf("%s: worker %s counted no call", b.Key(), f[1])
				}
			}
		}
		calls, _ := strconv.ParseInt(h["calls_window"], 10, 64)
		if calls != int64(lat.N) || sums["second"] != calls || sums["worker"] != calls || sums["bucket"] != calls || workers != c.w {
			t.Fatalf("%s: calls %d, sums %v, %d workers", b.Key(), calls, sums, workers)
		}
		if h["workers"] != strconv.Itoa(c.w) || h["stage"] != c.stage {
			t.Fatalf("%s: header %v", b.Key(), h)
		}
	}
	// More workers than the container's CPUs are refused.
	if cpus, err := quality.CPUSet(); err == nil {
		op := plan.E12Op("verification-n10-k5-v16", len(cpus)+1)
		b := plan.Block{ID: "E12.ml-dsa-44+ml-kem-768." + op + ".r1", Exp: "E12", Alg: "ml-dsa-44+ml-kem-768", Op: op,
			Rep: 1, Attempt: 1, Profile: plan.ProfileSmoke, Mode: "window", Seed: 12}
		if _, err := block(t, proto.Spec{Block: b, PrepDir: dir, RawFile: filepath.Join(out, b.RawPath())}); err == nil ||
			!strings.Contains(err.Error(), fmt.Sprintf("%d workers", len(cpus)+1)) {
			t.Fatalf("%d workers on %d CPUs: %v", len(cpus)+1, len(cpus), err)
		}
	}
}
