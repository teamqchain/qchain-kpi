package main

// E8: sizes. One block per (algorithm, keygen | sign | encaps) generates many
// outputs in one process and records the length of every item. Nothing is
// timed. Lengths are what the libraries return; QChain stores every key,
// signature, ciphertext and shared secret as a lowercase hex string inside
// JSON, so the encoded size of an item of n bytes is 2n hex characters, or
// 2n + 2 bytes as a JSON string (recorded in the raw header; derived in the
// analysis).

import (
	"bytes"
	"compress/gzip"
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
	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/record"
	"qchain-kpi/internal/stats"
	"qchain-kpi/internal/timing"
)

// e8Budget is the sampling rule of E8: produce Target items, but stop
// earlier once TimeBox has passed and at least Min items exist (slow key
// generations and SLH-DSA signatures).
type e8Budget struct {
	Target  int
	Min     int
	TimeBox time.Duration
}

func e8BudgetFor(profile, op string) (e8Budget, error) {
	switch profile {
	case plan.ProfileApproved:
		if op == "keygen" {
			return e8Budget{Target: 1000, Min: 100, TimeBox: 60 * time.Second}, nil
		}
		return e8Budget{Target: 10000, Min: 100, TimeBox: 60 * time.Second}, nil
	case plan.ProfileSmoke:
		if op == "keygen" {
			return e8Budget{Target: 20, Min: 5, TimeBox: 2 * time.Second}, nil
		}
		return e8Budget{Target: 50, Min: 5, TimeBox: 2 * time.Second}, nil
	}
	return e8Budget{}, fmt.Errorf("no E8 budget for profile %q", profile)
}

func e8Block(sp proto.Spec, a algs.Alg) error {
	b := sp.Block
	items := proto.E8Arms(a, b.Op)
	if items == nil {
		return fmt.Errorf("block: E8 has no operation %q (only keygen, sign, encaps)", b.Op)
	}
	if !a.HasOp(b.Op) {
		return fmt.Errorf("block: %s has no operation %q", a.ID, b.Op)
	}
	bud, err := e8BudgetFor(b.Profile, b.Op)
	if err != nil {
		return err
	}
	oomSet := procsetup.OOMAdj()
	rng := rand.New(rand.NewPCG(b.Seed, 0x6b70692d65382d6d)) // "kpi-e8-m"
	lengths := make([][]int, len(items))
	keysChecked := 0

	cpus, _ := quality.CPUSet()
	q0, _ := quality.Take()
	startNS := time.Now().UnixNano()
	t0 := time.Now()
	more := func(n int) bool {
		if n >= bud.Target {
			return false
		}
		return n < bud.Min || time.Since(t0) < bud.TimeBox
	}

	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return err
		}
		sizes := s.Sizes()
		switch b.Op {
		case "keygen":
			sub := newKeySubset(b.Seed)
			for i := 0; more(i); i++ {
				pk, sk, err := s.GenerateKey()
				if err != nil {
					return err
				}
				if !sizes.KeyLengthsOK(pk, sk) {
					return fmt.Errorf("%w: key lengths %d/%d, want %+v", timing.ErrCheck, len(pk), len(sk), sizes)
				}
				sub.observe(i, pk, sk)
				lengths[0] = append(lengths[0], len(pk))
				lengths[1] = append(lengths[1], len(sk))
			}
			if keysChecked, err = exerciseSig(s, sub)(); err != nil {
				return fmt.Errorf("%w: %v", timing.ErrCheck, err)
			}
		case "sign":
			pk, sk, err := s.GenerateKey()
			if err != nil {
				return err
			}
			for i := 0; more(i); i++ {
				m := inputs.HexMessage(rng)
				sg, err := s.Sign(sk, pk, m)
				if err != nil {
					return err
				}
				if err := sigLengthOK(sizes, sg); err != nil {
					return fmt.Errorf("%w: %v", timing.ErrCheck, err)
				}
				if ok, err := s.Verify(pk, m, sg); err != nil || !ok {
					return fmt.Errorf("%w: signature %d does not verify (%v)", timing.ErrCheck, i, err)
				}
				lengths[0] = append(lengths[0], len(sg))
			}
		}
	} else {
		k, err := newKEM(a)
		if err != nil {
			return err
		}
		sizes := k.Sizes()
		switch b.Op {
		case "keygen":
			sub := newKeySubset(b.Seed)
			for i := 0; more(i); i++ {
				pk, sk, err := k.GenerateKey()
				if err != nil {
					return err
				}
				if !sizes.KeyLengthsOK(pk, sk) {
					return fmt.Errorf("%w: key lengths %d/%d, want %+v", timing.ErrCheck, len(pk), len(sk), sizes)
				}
				sub.observe(i, pk, sk)
				lengths[0] = append(lengths[0], len(pk))
				lengths[1] = append(lengths[1], len(sk))
			}
			if keysChecked, err = exerciseKEM(k, sub)(); err != nil {
				return fmt.Errorf("%w: %v", timing.ErrCheck, err)
			}
		case "encaps":
			pk, sk, err := k.GenerateKey()
			if err != nil {
				return err
			}
			for i := 0; more(i); i++ {
				ct, ss, err := k.Encaps(pk)
				if err != nil {
					return err
				}
				if len(ct) != sizes.Ciphertext || len(ss) != sizes.SharedSecret {
					return fmt.Errorf("%w: ciphertext/secret lengths %d/%d", timing.ErrCheck, len(ct), len(ss))
				}
				ss2, err := k.Decaps(sk, pk, ct)
				if err != nil || !equalBytes(ss, ss2) {
					return fmt.Errorf("%w: encapsulation %d does not decapsulate (%v)", timing.ErrCheck, i, err)
				}
				lengths[0] = append(lengths[0], len(ct))
				lengths[1] = append(lengths[1], len(ss))
			}
		}
	}
	wall := time.Since(t0).Nanoseconds()
	endNS := time.Now().UnixNano()
	q1, _ := quality.Take()

	// Raw file: one row per item, in production order.
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	header := map[string]string{
		"key": b.Key(), "id": b.ID, "exp": b.Exp, "alg": b.Alg, "op": b.Op,
		"rep": strconv.Itoa(b.Rep), "attempt": strconv.Itoa(b.Attempt), "profile": b.Profile,
		"seed": strconv.FormatUint(b.Seed, 10), "items": strings.Join(items, ","),
		"budget":         fmt.Sprintf("target %d items, at least %d, stop after %s once the minimum is reached", bud.Target, bud.Min, bud.TimeBox),
		"columns":        "item = pk, sk, sig, ct or ss; bytes = its length as returned by the library (raw bytes)",
		"encoding":       "QChain stores these as lowercase hex strings in JSON: hex characters = 2 x bytes; JSON string = 2 x bytes + 2",
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
	fmt.Fprintln(gz, "item,bytes")
	rows := 0
	for i := range lengths[0] {
		for k, name := range items {
			fmt.Fprintf(gz, "%s,%d\n", name, lengths[k][i])
			rows++
		}
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

	out := proto.Result{
		Key: b.Key(), Block: b, StartNS: startNS, EndNS: endNS, WallNS: wall,
		Budget:  proto.BudgetInfo{Mode: "sizes", Samples: bud.Target, MinSamples: bud.Min, TimeBoxS: bud.TimeBox.Seconds()},
		Quality: quality.Diff(q0, q1, cpus), CPUs: cpus, GOMAXPROCS: runtime.GOMAXPROCS(0), OOMAdj: oomSet,
		KeysChecked: keysChecked, Rows: rows, RawSHA256: sum,
	}
	for k, name := range items {
		xs := make([]float64, len(lengths[k]))
		lo, hi := 0, 0
		for i, n := range lengths[k] {
			xs[i] = float64(n)
			if i == 0 || n < lo {
				lo = n
			}
			if n > hi {
				hi = n
			}
		}
		// For E8 the statistics are in bytes, not nanoseconds.
		out.Arms = append(out.Arms, proto.ArmSummary{Name: name, Batch: 1, Stats: stats.Summarize(xs, 0, 0),
			MinBytes: lo, MaxBytes: hi})
	}
	return emit(out)
}
