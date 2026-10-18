//go:build darwin

package quality

import (
	"runtime"
	"testing"
	"time"
)

// TestTakeDarwin: every CPU is read, a busy loop shows up as this process's
// own CPU time and as machine busy time, and nothing counts as "other" beyond
// what the machine really did.
func TestTakeDarwin(t *testing.T) {
	a, err := Take()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.PerCPU) != runtime.NumCPU() {
		t.Fatalf("%d CPUs read, NumCPU %d", len(a.PerCPU), runtime.NumCPU())
	}
	end := time.Now().Add(300 * time.Millisecond)
	x := 0
	for time.Now().Before(end) {
		x++
	}
	b, err := Take()
	if err != nil {
		t.Fatal(err)
	}
	if own := b.UsageUS - a.UsageUS; own < 200_000 {
		t.Fatalf("own CPU time %d µs over a 300 ms busy loop", own)
	}
	iv := Diff(a, b, nil)
	if iv.VMBusyPct <= 0 || iv.VMBusyPct > 100 || iv.OtherPct < 0 || iv.OtherPct > iv.VMBusyPct {
		t.Fatalf("interval %+v", iv)
	}
	cpus, _ := CPUSet()
	if len(cpus) != runtime.NumCPU() {
		t.Fatal("CPUSet")
	}
	_ = x
}
