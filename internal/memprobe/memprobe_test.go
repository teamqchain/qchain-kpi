//go:build linux

package memprobe

import (
	"os"
	"runtime"
	"testing"
)

// TestMain re-runs the test binary with the heap counter preloaded (the image
// sets KPI_MALLOC_LIB to the library it builds), the way kpi-batch starts
// E9's blocks.
func TestMain(m *testing.M) { os.Exit(Preloaded(m.Run)) }

func TestCounterLoaded(t *testing.T) {
	if !Counting() {
		t.Fatalf("the heap counter is not loaded (KPI_MALLOC_LIB=%q)", os.Getenv(LibEnv))
	}
}

// TestStackMeasure: a job that writes a 64 KiB local array measures 64 KiB
// plus its own small frame above the baseline of an empty job.
func TestStackMeasure(t *testing.T) {
	r, err := NewRunner(StackSize)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	base := r.Run(Empty)
	if base.Stack <= 0 || base.Stack > 1024 {
		t.Fatalf("baseline %d bytes", base.Stack)
	}
	for i := 0; i < 3; i++ {
		got := r.Run(testStackJob()).Stack - base.Stack
		if got < 65536 || got > 65536+512 {
			t.Fatalf("64 KiB array measured %d bytes (baseline %d)", got, base.Stack)
		}
		if again := r.Run(Empty).Stack; again != base.Stack {
			t.Fatalf("baseline moved from %d to %d after a deep job (painting does not reset)", base.Stack, again)
		}
	}
}

// TestHeapMeasure: a job that keeps a 1,000-byte block and frees a
// 100,000-byte one is counted exactly on the runner thread.
func TestHeapMeasure(t *testing.T) {
	r, err := NewRunner(StackSize)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	j, slot := testMallocJob()
	defer testFreeSlot(slot)
	res := r.Run(j)
	h := res.Heap
	if res.RC != 0 || h.Requested != 101000 || h.Allocs != 2 || h.Frees != 1 {
		t.Fatalf("counted %+v (rc %d)", h, res.RC)
	}
	if h.Retained < 1000 || h.Retained > 1040 {
		t.Fatalf("retained %d, want the kept block's usable size", h.Retained)
	}
	if h.Peak < 101000 || h.Peak > h.Bytes || h.Bytes > 101100 {
		t.Fatalf("peak %d, bytes %d", h.Peak, h.Bytes)
	}
}

// TestThreadHeap: the Go side reads the counters of its own (locked) thread.
func TestThreadHeap(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ResetPeak()
	before := ThreadHeap()
	p := testMallocHere(777)
	after := ThreadHeap()
	testFree(p)
	end := ThreadHeap()
	u := Delta(before, after)
	if u.Requested != 777 || u.Allocs != 1 || u.Retained < 777 || u.Peak != u.Retained {
		t.Fatalf("malloc(777) counted %+v", u)
	}
	if d := Delta(before, end); d.Retained != 0 || d.Frees != 1 {
		t.Fatalf("after free: %+v", d)
	}
}

func TestRSS(t *testing.T) {
	if err := ResetPeakRSS(); err != nil {
		t.Fatal(err)
	}
	rss, hwm, err := RSS()
	if err != nil || rss <= 0 || hwm < rss {
		t.Fatalf("rss %d hwm %d err %v", rss, hwm, err)
	}
	buf := make([]byte, 64<<20)
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}
	_, hwm2, err := RSS()
	if err != nil || hwm2-rss < 60<<20 {
		t.Fatalf("touching 64 MiB raised the peak by %d bytes (%v)", hwm2-rss, err)
	}
	runtime.KeepAlive(buf)
}
