package l0

import (
	"testing"
	"time"
)

// TestTime: a job sleeping 2 ms measures at least 2 ms inside C and agrees
// with Go's clock around the call; the result code comes back; the timer
// floor and a null cgo call are tiny.
func TestTime(t *testing.T) {
	fn, arg, free := testSleep(2000)
	defer free()
	t0 := time.Now()
	ns, rc := Time(fn, arg)
	outer := time.Since(t0).Nanoseconds()
	if rc != 7 || ns < 2e6 || ns > outer || outer-ns > 2e6 {
		t.Fatalf("C-timed %d ns, Go around it %d ns, rc %d", ns, outer, rc)
	}
	if rc := Call(fn, arg); rc != 7 {
		t.Fatalf("Call rc %d", rc)
	}
	if e := Empty(); e < 0 || e > 100e3 {
		t.Fatalf("empty C region %d ns", e)
	}
	const n = 100000
	t0 = time.Now()
	for i := 0; i < n; i++ {
		Null()
	}
	if per := time.Since(t0).Nanoseconds() / n; per <= 0 || per > 5000 {
		t.Fatalf("null cgo call %d ns", per)
	}
}
