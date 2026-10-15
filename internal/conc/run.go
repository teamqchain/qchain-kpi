package conc

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Worker is one worker's own operation (its own inputs and state; workers
// share nothing). Call performs call i; Check verifies the most recent call.
type Worker struct {
	Call  func(i int) (int, error)
	Check func(i, n int) error
}

// Config is the time plan of a run.
type Config struct {
	Warmup     time.Duration // calls before the window are not counted
	Window     time.Duration // calls completed inside it are counted
	CheckEvery int           // inside the window, check every n-th call of each worker (warm-up: every call)
}

// Result is a run's outcome.
type Result struct {
	Workers     int
	Hist        Hist    // latency of every call completed in the window
	PerSecond   []int64 // calls completed in each second of the window, all workers
	PerWorker   []int64 // calls completed in the window, per worker
	WarmupCalls int64   // calls completed before the window (all checked)
	Checked     int64   // calls checked in total
	WindowNS    int64   // the window's length
	WallNS      int64   // start to the end of the last call
}

// Throughput is the calls per second completed in the window.
func (r *Result) Throughput() float64 { return float64(r.Hist.N) / (float64(r.WindowNS) / 1e9) }

type workerOut struct {
	hist      Hist
	perSecond []int64
	warm, chk int64
	err       error
}

// Run starts every worker at once and lets each call its operation back to
// back: first for the warm-up, then for the window. A call counts when it
// completes inside the window; no call starts after the window ends, and the
// calls still running then are waited for but not counted. The first error
// stops every worker.
func Run(ws []Worker, cfg Config) (Result, error) {
	if len(ws) == 0 || cfg.Window <= 0 || cfg.Warmup < 0 || cfg.CheckEvery < 1 {
		return Result{}, fmt.Errorf("conc: bad run (%d workers, %+v)", len(ws), cfg)
	}
	secs := int((cfg.Window + time.Second - 1) / time.Second)
	outs := make([]*workerOut, len(ws))
	var stop atomic.Bool
	var ready, done sync.WaitGroup
	start := make(chan struct{})
	var t0, warmEnd, end time.Time
	ready.Add(len(ws))
	done.Add(len(ws))
	for k := range ws {
		o := &workerOut{perSecond: make([]int64, secs)}
		outs[k] = o
		go func(w Worker, o *workerOut) {
			defer done.Done()
			ready.Done()
			<-start
			for i := 0; !stop.Load(); i++ {
				s := time.Now()
				if !s.Before(end) {
					return
				}
				n, err := w.Call(i)
				e := time.Now()
				if err != nil {
					o.err = fmt.Errorf("call %d: %w", i, err)
					stop.Store(true)
					return
				}
				inWindow := !e.Before(warmEnd) && e.Before(end)
				if inWindow {
					d := e.Sub(s).Nanoseconds()
					o.hist.Add(d)
					o.perSecond[int(e.Sub(warmEnd)/time.Second)]++
				} else if e.Before(warmEnd) {
					o.warm++
				}
				if w.Check != nil && (!inWindow || i%cfg.CheckEvery == 0) {
					if err := w.Check(i, n); err != nil {
						o.err = fmt.Errorf("call %d: %w", i, err)
						stop.Store(true)
						return
					}
					o.chk++
				}
			}
		}(ws[k], o)
	}
	ready.Wait()
	t0 = time.Now()
	warmEnd = t0.Add(cfg.Warmup)
	end = warmEnd.Add(cfg.Window)
	close(start) // the times above happen before every worker's first read (channel close)
	done.Wait()

	r := Result{Workers: len(ws), PerSecond: make([]int64, secs), WindowNS: cfg.Window.Nanoseconds(),
		WallNS: time.Since(t0).Nanoseconds()}
	var errs []error
	for _, o := range outs {
		if o.err != nil {
			errs = append(errs, o.err)
		}
		r.Hist.Merge(&o.hist)
		for s, c := range o.perSecond {
			r.PerSecond[s] += c
		}
		r.PerWorker = append(r.PerWorker, o.hist.N)
		r.WarmupCalls += o.warm
		r.Checked += o.chk
	}
	if len(errs) > 0 {
		return r, errors.Join(errs...)
	}
	return r, nil
}
