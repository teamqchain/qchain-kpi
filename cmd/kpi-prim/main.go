// Command kpi-prim is the measuring process. Every invocation is one fresh
// process doing one thing:
//
//	kpi-prim prep   --alg ID --rep R --pool P --seed S --dir DIR [--e10] [--e6]
//	kpi-prim block  --spec JSON
//	kpi-prim canary [--windows 15] [--window-ms 100]
//
// prep generates the untimed inputs of one (algorithm, repetition); block
// measures one block of E1, E10 or the probe pass; canary runs the fixed
// SHA3-256 throughput check. Each prints exactly one JSON line on stdout.
// It is started by kpi-batch, never by hand during a run.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/canary"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

func main() {
	runtime.LockOSThread()
	if len(os.Args) < 2 {
		usage()
	}
	if err := procsetup.GoGate(); err != nil {
		fail(proto.ExitGate, err)
	}
	if err := pqc.CheckLibrary(); err != nil {
		fail(proto.ExitGate, err)
	}
	var err error
	switch os.Args[1] {
	case "prep":
		err = cmdPrep(os.Args[2:])
	case "block":
		err = cmdBlock(os.Args[2:])
	case "canary":
		err = cmdCanary(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		code := proto.ExitError
		if errors.Is(err, timing.ErrCheck) {
			code = proto.ExitCorrectness
		}
		fail(code, err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: kpi-prim prep|block|canary [flags]")
	os.Exit(proto.ExitUsage)
}

func fail(code int, err error) {
	fmt.Fprintf(os.Stderr, "kpi-prim: %v\n", err)
	os.Exit(code)
}

func emit(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "%s\n", b)
	return err
}

// newSignature / newKEM pick the implementation. Neither runs a crypto
// operation.
func newSignature(a algs.Alg) (scheme.Signature, error) {
	if a.Classical() {
		return classical.NewSignature(a)
	}
	return pqc.NewSignature(a)
}

func newKEM(a algs.Alg) (scheme.KEM, error) {
	if a.Classical() {
		return classical.NewKEM(a)
	}
	return pqc.NewKEM(a)
}

// ---------------------------------------------------------------------- prep

func cmdPrep(args []string) error {
	fs := flag.NewFlagSet("prep", flag.ExitOnError)
	id := fs.String("alg", "", "algorithm id")
	rep := fs.Int("rep", -1, "repetition (0 = the probe pass's small input set)")
	pool := fs.Int("pool", 0, "pool size P")
	seed := fs.Uint64("seed", 0, "message seed")
	dir := fs.String("dir", "", "prep directory")
	e10 := fs.Bool("e10", false, "also prepare the E10 pools (ML-DSA)")
	e6 := fs.Bool("e6", false, "also prepare the E6 pools (signature schemes: messages of every E6 size)")
	fs.Parse(args)
	a, ok := algs.ByID(*id)
	if !ok || *rep < 0 || *pool < 1 || *dir == "" {
		return fmt.Errorf("prep: need --alg, --rep >= 0, --pool >= 1 and --dir")
	}
	if *e10 && (a.Family != "ML-DSA" || *rep == proto.ProbeRep) {
		return fmt.Errorf("prep: E10 pools are defined for ML-DSA repetitions only")
	}
	if err := os.MkdirAll(*dir, 0o700); err != nil {
		return err
	}
	var p *inputs.Prep
	var err error
	if a.Track == algs.Sig {
		s, err2 := newSignature(a)
		if err2 != nil {
			return err2
		}
		p, err = inputs.GenerateSig(s, *rep, *pool, *seed, *e10)
		if err == nil && *e6 {
			err = inputs.AddE6(p, s, proto.E6Sizes, func(size int) int { return proto.E6PoolSize(*pool, size) }, *seed)
		}
	} else {
		if *e6 {
			return fmt.Errorf("prep: E6 pools are defined for signature schemes only")
		}
		k, err2 := newKEM(a)
		if err2 != nil {
			return err2
		}
		p, err = inputs.GenerateKEM(k, *rep, *pool, *seed)
	}
	if err != nil {
		return err
	}
	if err := inputs.Save(proto.PrepPath(*dir, a.ID, *rep), p); err != nil {
		return err
	}
	return emit(proto.PrepResult{Alg: a.ID, Rep: *rep, Pool: p.P, PrepNS: p.PrepNS, Fingerprint: p.Fingerprint, E10: p.HasE10(),
		E6: len(p.E6) > 0 && p.HasE6(proto.E6Sizes)})
}

// -------------------------------------------------------------------- canary

func cmdCanary(args []string) error {
	fs := flag.NewFlagSet("canary", flag.ExitOnError)
	windows := fs.Int("windows", 15, "number of windows")
	windowMS := fs.Int("window-ms", 100, "window length in ms")
	fs.Parse(args)
	cpus, _ := quality.CPUSet()
	start := time.Now().UnixNano()
	r := canary.Run(*windows, time.Duration(*windowMS)*time.Millisecond)
	return emit(proto.CanaryResult{Result: r, StartNS: start, EndNS: time.Now().UnixNano(), CPUs: cpus})
}
