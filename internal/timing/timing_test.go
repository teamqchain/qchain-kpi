package timing

import (
	"errors"
	"testing"
	"time"
)

func TestFastInterleavedBudgetAndOrder(t *testing.T) {
	var rows []string
	calls := map[string]int{}
	mk := func(name string) Arm {
		return Arm{Name: name, Call: func(i int) (int, error) { calls[name]++; return 7, nil }}
	}
	cfg := Config{Mode: Fast, Warmup: 3, Samples: 50, FirstLabel: "cold", Seed: 1,
		Sink: func(arm string, ns int64, nb int) { rows = append(rows, arm) }}
	res, err := Run([]Arm{mk("a"), mk("b"), mk("c")}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res.Arms {
		if len(r.PerOpNS) != 50 {
			t.Fatalf("arm %s: %d samples, want 50", r.Name, len(r.PerOpNS))
		}
	}
	// 3 first calls + 3*3 warm-up + 3*50 samples
	for _, n := range []string{"a", "b", "c"} {
		if calls[n] != 1+3+50 {
			t.Fatalf("arm %s called %d times", n, calls[n])
		}
	}
	if rows[0] != "cold:a" || rows[1] != "cold:b" || rows[2] != "cold:c" {
		t.Fatalf("first rows %v", rows[:3])
	}
	if len(rows) != 3+150 {
		t.Fatalf("rows %d", len(rows))
	}
}

func TestSeedDeterminesOrder(t *testing.T) {
	run := func(seed uint64) []string {
		var rows []string
		cfg := Config{Mode: Fast, Samples: 20, Seed: seed, Sink: func(a string, _ int64, _ int) { rows = append(rows, a) }}
		arms := []Arm{{Name: "a", Call: func(int) (int, error) { return 0, nil }},
			{Name: "b", Call: func(int) (int, error) { return 0, nil }}}
		if _, err := Run(arms, cfg); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	x, y := run(9), run(9)
	for i := range x {
		if x[i] != y[i] {
			t.Fatal("same seed gave a different order")
		}
	}
}

func TestSlowBudgetUsesAccumulatedTimeAndMinimum(t *testing.T) {
	a := Arm{Name: "slow", Call: func(int) (int, error) { time.Sleep(2 * time.Millisecond); return 0, nil }}
	cfg := Config{Mode: Slow, Warmup: 0, TimeBox: 20 * time.Millisecond, MinSamples: 30, Seed: 1}
	res, err := Run([]Arm{a}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	n := len(res.Arms[0].PerOpNS)
	if n < 30 {
		t.Fatalf("slow arm took %d samples, minimum is 30", n)
	}
}

func TestBatchDividesPerOp(t *testing.T) {
	var lastIdx int
	a := Arm{Name: "tiny", Batch: 10, Call: func(i int) (int, error) { lastIdx = i; return 0, nil }}
	res, err := Run([]Arm{a}, Config{Mode: Fast, Samples: 5, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Arms[0].PerOpNS) != 5 || res.Arms[0].Batch != 10 {
		t.Fatalf("result %+v", res.Arms[0])
	}
	// first call uses index 0, then 5 samples x 10 calls use indices 1..50.
	if lastIdx != 50 {
		t.Fatalf("last index %d, want 50", lastIdx)
	}
}

func TestCheckFailureAbortsBlock(t *testing.T) {
	a := Arm{Name: "bad",
		Call:  func(int) (int, error) { return 1, nil },
		Check: func(i, nb int) error { if i == 5 { return errors.New("wrong") }; return nil }}
	_, err := Run([]Arm{a}, Config{Mode: Fast, Samples: 100, Seed: 1})
	if !errors.Is(err, ErrCheck) {
		t.Fatalf("expected ErrCheck, got %v", err)
	}
}

// TestTimedArm: a self-timed arm's own durations are recorded (first call and
// every sample, a batch summing its calls), its warm-up goes through Call, and
// an ordinary arm interleaved with it is still timed by the engine.
func TestTimedArm(t *testing.T) {
	calls, timed := 0, 0
	self := Arm{Name: "self", Batch: 2,
		Call:  func(int) (int, error) { calls++; return 7, nil },
		Timed: func(int) (int64, int, error) { timed++; return 1000, 7, nil }}
	plain := Arm{Name: "plain", Call: func(int) (int, error) { time.Sleep(50 * time.Microsecond); return 0, nil }}
	var rows []int64
	cfg := Config{Mode: Fast, Warmup: 3, Samples: 20, FirstLabel: "first", Seed: 1,
		Sink: func(arm string, ns int64, _ int) {
			if arm == "self" {
				rows = append(rows, ns)
			}
		}}
	res, err := Run([]Arm{self, plain}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s, p := res.Arms[0], res.Arms[1]
	if s.FirstNS != 1000 || len(s.PerOpNS) != 20 || calls != 3 || timed != 1+20*2 {
		t.Fatalf("self-timed arm: first %d, %d samples, %d calls, %d timed calls", s.FirstNS, len(s.PerOpNS), calls, timed)
	}
	for _, v := range s.PerOpNS {
		if v != 1000 {
			t.Fatalf("self-timed per-op %v, want 1000", v)
		}
	}
	for _, ns := range rows {
		if ns != 2000 {
			t.Fatalf("batch total %d, want 2000", ns)
		}
	}
	if s.SumNS != 20*2000 || s.MinBytes != 7 {
		t.Fatalf("sum %d bytes %d", s.SumNS, s.MinBytes)
	}
	for _, v := range p.PerOpNS {
		if v < 50e3 {
			t.Fatalf("plain arm %v ns below its 50 µs sleep", v)
		}
	}
}
