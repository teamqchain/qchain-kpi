package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/stats"
)

// maxAttempts is the number of non-diagnostic attempts of a block: the first
// one plus at most 2 repeats for external causes (research plan §9).
const maxAttempts = 3

// canaryRec is one canary run as stored in canary.jsonl.
type canaryRec struct {
	Seq    int    `json:"seq"`
	Reason string `json:"reason"` // start, periodic, resume, end
	proto.CanaryResult
	Ratio float64 `json:"ratio_to_first"`
	Core  string  `json:"core_class,omitempty"` // Mac: "P-like" or "E-like"
}

// Session is one start or resume of the batch.
type Session struct {
	N            int     `json:"n"`
	StartedUTC   string  `json:"started_utc"`
	StartNS      int64   `json:"start_unix_ns"`
	HostOffsetNS int64   `json:"host_offset_ns"` // host clock minus container clock
	NativeP      float64 `json:"native_p_rate,omitempty"`
	NativeE      float64 `json:"native_e_rate,omitempty"`
}

// evalInput is everything the quality rules look at.
type evalInput struct {
	Kind      string // plan kind
	Results   []proto.Result
	Canaries  []canaryRec
	Host      []quality.HostSample
	Sessions  []Session
	TH        *quality.Machine
	CellReps  map[string]int // expected first attempts per cell (E1/E10)
	Final     bool           // no more canaries or host samples will come for finished blocks
	HostCPUs  int            // logical CPUs of the host
	VMCPUs    int            // CPUs of the kernel the container runs on
	PeriodNS  int64          // host monitor sampling period
	HostCheck bool           // host coverage is required (trial/main)
}

// loadEval reads a run folder into the input of the quality rules.
func loadEval(dir string, st *State, th *quality.Machine, final bool) (evalInput, error) {
	results, err := readJSONL[proto.Result](filepath.Join(dir, fileBlocksJL))
	if err != nil {
		return evalInput{}, err
	}
	canaries, err := readJSONL[canaryRec](filepath.Join(dir, fileCanaryJL))
	if err != nil {
		return evalInput{}, err
	}
	host, _, err := quality.ReadHostChunks(filepath.Join(dir, dirMonitor))
	if err != nil {
		return evalInput{}, err
	}
	return evalInput{Kind: st.PlanKind, Results: results, Canaries: canaries, Host: host, Sessions: st.Sessions,
		TH: th, CellReps: st.CellReps, Final: final, HostCPUs: st.HostCPUs, VMCPUs: st.VMCPUs,
		PeriodNS: st.HostPeriodNS, HostCheck: st.PlanKind != plan.KindSmoke}, nil
}

// Verdict is the quality decision for one attempt.
type Verdict struct {
	Res           proto.Result
	ExtPending    bool     // waiting for host coverage or the following canary
	External      []string // external-cause flags
	DriftPending  bool
	Drift         bool
	DriftDev      float64
	DriftLimit    float64
	Host          quality.HostVerdict
	CombinedOther float64
	CanaryBefore  float64 // ratio to the batch's first canary (NaN if none)
	CanaryAfter   float64
	CoreBefore    string
	CoreAfter     string
	Headline      bool
	Action        string // "", repeat, diagnostic, exhausted, record
}

func (v *Verdict) flagged() bool { return len(v.External) > 0 }

// sessionFor returns the session a block started in.
func sessionFor(ss []Session, startNS int64) (Session, bool) {
	var out Session
	ok := false
	for _, s := range ss {
		if s.StartNS <= startNS {
			out, ok = s, true
		}
	}
	return out, ok
}

func refKey(cell, arm string) string { return cell + "|" + arm }

// evaluate applies the flag rules of coding plan §8 to every attempt.
func evaluate(in evalInput) []*Verdict {
	// Drift references: median of the first attempts' block medians, per
	// cell and arm, frozen once all first attempts of the cell exist.
	firsts := map[string]map[string][]float64{}
	nFirst := map[string]int{}
	for _, r := range in.Results {
		b := r.Block
		if b.Attempt != 1 || b.Diagnostic || !proto.Measured(b.Exp) {
			continue
		}
		c := b.Cell()
		nFirst[c]++
		if firsts[c] == nil {
			firsts[c] = map[string][]float64{}
		}
		for _, a := range r.Arms {
			firsts[c][a.Name] = append(firsts[c][a.Name], a.Stats.P50)
		}
	}
	ref := map[string]float64{}
	for c, arms := range firsts {
		want := in.CellReps[c]
		if nFirst[c] < want && !(in.Final && nFirst[c] >= 3) {
			continue
		}
		for arm, meds := range arms {
			ref[refKey(c, arm)] = stats.Median(meds)
		}
	}

	out := make([]*Verdict, 0, len(in.Results))
	for _, r := range in.Results {
		v := &Verdict{Res: r, CanaryBefore: math.NaN(), CanaryAfter: math.NaN()}
		// Host monitor.
		sess, _ := sessionFor(in.Sessions, r.StartNS)
		v.Host = quality.HostOver(in.Host, r.StartNS, r.EndNS, sess.HostOffsetNS, in.PeriodNS)
		hostOther := 0.0
		if v.Host.MaxOtherPct > 0 {
			hostOther = v.Host.MaxOtherPct
		}
		vmShare := r.Quality.OtherPct
		if in.HostCPUs > 0 && in.VMCPUs > 0 {
			vmShare = r.Quality.OtherPct * float64(in.VMCPUs) / float64(in.HostCPUs)
		}
		v.CombinedOther = hostOther + vmShare
		if in.HostCheck && !v.Host.Covered {
			if in.Final {
				v.External = append(v.External, "host_missing")
			} else {
				v.ExtPending = true
			}
		}
		if v.Host.Throttled {
			v.External = append(v.External, "thermal")
		}
		if v.Host.OnBattery {
			v.External = append(v.External, "battery")
		}
		if v.Host.Alert {
			v.External = append(v.External, "alert")
		}
		if r.Quality.StealPct > in.TH.StealPct {
			v.External = append(v.External, "steal")
		}
		if v.CombinedOther > in.TH.OtherPct {
			v.External = append(v.External, "other")
		}
		if r.Quality.NrThrottled > 0 {
			v.External = append(v.External, "quota")
		}
		// Bracketing canaries.
		var before, after *canaryRec
		for i := range in.Canaries {
			c := &in.Canaries[i]
			if c.EndNS <= r.StartNS {
				before = c
			}
			if after == nil && c.StartNS >= r.EndNS {
				after = c
			}
		}
		if after == nil && !in.Final {
			v.ExtPending = true
		}
		canaryBad, coreBad := false, false
		if before != nil {
			v.CanaryBefore, v.CoreBefore = before.Ratio, before.Core
			canaryBad = canaryBad || math.Abs(before.Ratio-1) > in.TH.CanaryFrac
			coreBad = coreBad || before.Core == "E-like"
		}
		if after != nil {
			v.CanaryAfter, v.CoreAfter = after.Ratio, after.Core
			canaryBad = canaryBad || math.Abs(after.Ratio-1) > in.TH.CanaryFrac
			coreBad = coreBad || after.Core == "E-like"
		}
		if canaryBad {
			v.External = append(v.External, "canary")
		}
		if coreBad {
			v.External = append(v.External, "core")
		}
		if proto.SizeOnly(r.Block.Exp) {
			// Sizes do not depend on machine noise: no run-quality flags,
			// no repeats; the first attempt is the result.
			v.External, v.ExtPending = nil, false
			out = append(out, v)
			continue
		}
		// Drift against the frozen reference.
		if proto.Measured(r.Block.Exp) {
			for _, a := range r.Arms {
				rv, ok := ref[refKey(r.Block.Cell(), a.Name)]
				if !ok {
					v.DriftPending = !in.Final
					continue
				}
				if rv <= 0 {
					continue
				}
				dev := math.Abs(a.Stats.P50/rv - 1)
				lim := math.Max(in.TH.DriftFrac, 3*a.Stats.BootSE50/rv)
				if dev-lim > v.DriftDev-v.DriftLimit || v.DriftLimit == 0 {
					v.DriftDev, v.DriftLimit = dev, lim
				}
				if dev > lim {
					v.Drift = true
				}
			}
		}
		out = append(out, v)
	}
	return out
}

// decision is a block to add to the queue.
type decision struct {
	Block  plan.Block
	Reason string // "repeat:<flags>" or "diagnostic:drift"
}

// decide marks headlines and returns the repeats and diagnostic re-runs to
// queue (coding plan §8 attempt policy). requeued/diagnosed record what was
// already queued, so a decision is never taken twice.
func decide(kind string, vs []*Verdict, batchSeed uint64, requeued, diagnosed map[string]bool) []decision {
	return decideBase(kind, vs, batchSeed, requeued, diagnosed, nil)
}

// decideBase is decide for runs that were extended: base[id] is the number of
// attempts a block already had when it was extended, so its cap of
// maxAttempts counts only the attempts made since.
func decideBase(kind string, vs []*Verdict, batchSeed uint64, requeued, diagnosed map[string]bool, base map[string]int) []decision {
	byID := map[string][]*Verdict{}
	maxAtt := map[string]int{}
	var ids []string
	for _, v := range vs {
		b := v.Res.Block
		if proto.IsProbe(b.Exp) {
			continue
		}
		if _, ok := byID[b.ID]; !ok {
			ids = append(ids, b.ID)
		}
		if b.Attempt > maxAtt[b.ID] {
			maxAtt[b.ID] = b.Attempt
		}
		if !b.Diagnostic {
			byID[b.ID] = append(byID[b.ID], v)
		} else if _, ok := byID[b.ID]; !ok {
			byID[b.ID] = nil
		}
	}
	sort.Strings(ids)
	var out []decision
	for _, id := range ids {
		att := byID[id]
		sort.Slice(att, func(i, j int) bool { return att[i].Res.Block.Attempt < att[j].Res.Block.Attempt })
		var head *Verdict
		undecided := false
		for _, v := range att {
			if v.ExtPending {
				undecided = true
				break
			}
			if !v.flagged() {
				head = v
				break
			}
		}
		if head != nil {
			head.Headline = true
		}
		if len(att) == 0 {
			continue
		}
		last := att[len(att)-1]
		for _, v := range att {
			if v.flagged() && v.Action == "" {
				v.Action = "record"
			}
		}
		if kind == plan.KindSmoke {
			continue
		}
		switch {
		case head == nil && !undecided && last.flagged() && len(att)-base[id] < maxAttempts:
			if !requeued[last.Res.Key] {
				nb := last.Res.Block
				nb.Attempt = maxAtt[id] + 1
				nb.Diagnostic = false
				nb.Seed = plan.DeriveSeed(batchSeed, nb.Key())
				out = append(out, decision{Block: nb, Reason: "repeat:" + strings.Join(last.External, ";")})
				requeued[last.Res.Key] = true
				maxAtt[id] = nb.Attempt
			}
			last.Action = "repeat"
		case head == nil && !undecided && last.flagged() && len(att)-base[id] >= maxAttempts:
			last.Action = "exhausted"
		case head != nil && kind == plan.KindMain && head.Drift && !head.DriftPending:
			if !diagnosed[id] {
				nb := head.Res.Block
				nb.Attempt = maxAtt[id] + 1
				nb.Diagnostic = true
				nb.Seed = plan.DeriveSeed(batchSeed, nb.Key())
				out = append(out, decision{Block: nb, Reason: fmt.Sprintf("diagnostic:drift %.4f > %.4f", head.DriftDev, head.DriftLimit)})
				diagnosed[id] = true
				maxAtt[id] = nb.Attempt
			}
			head.Action = "diagnostic"
		}
		if head != nil && head.Drift && head.Action == "" {
			head.Action = "record" // trial: drift is recorded, never re-run
		}
	}
	return out
}

// insertAt returns the seeded random queue position for a new attempt: any
// position from the head (run next) to the tail.
func insertAt(batchSeed uint64, key string, queueLen int) int {
	if queueLen == 0 {
		return 0
	}
	rng := rand.New(rand.NewPCG(plan.DeriveSeed(batchSeed, "requeue."+key), 0x6b70692d72657175)) // "kpi-requ"
	return rng.IntN(queueLen + 1)
}

var qualityHeader = []string{"key", "id", "exp", "alg", "op", "rep", "attempt", "diagnostic",
	"status", "external_flags", "drift", "drift_dev", "drift_limit",
	"steal_pct", "vm_other_pct", "host_max_other_pct", "combined_other_pct", "nr_throttled",
	"host_samples", "host_covered", "host_throttled", "host_on_battery", "host_alert",
	"canary_before_ratio", "canary_after_ratio", "core_before", "core_after", "headline", "action"}

func f4(x float64) string {
	if math.IsNaN(x) {
		return ""
	}
	return strconv.FormatFloat(x, 'f', 4, 64)
}

func qualityRows(vs []*Verdict) [][]string {
	rows := make([][]string, 0, len(vs))
	for _, v := range vs {
		b := v.Res.Block
		status := "ok"
		switch {
		case v.ExtPending:
			status = "pending"
		case v.flagged():
			status = "flagged"
		}
		drift := "no"
		if v.DriftPending {
			drift = "pending"
		} else if v.Drift {
			drift = "yes"
		}
		if proto.IsProbe(b.Exp) || proto.SizeOnly(b.Exp) {
			drift = "n/a"
		}
		rows = append(rows, []string{v.Res.Key, b.ID, b.Exp, b.Alg, b.Op, strconv.Itoa(b.Rep), strconv.Itoa(b.Attempt),
			strconv.FormatBool(b.Diagnostic), status, strings.Join(v.External, ";"), drift, f4(v.DriftDev), f4(v.DriftLimit),
			f4(v.Res.Quality.StealPct), f4(v.Res.Quality.OtherPct), f4(v.Host.MaxOtherPct), f4(v.CombinedOther),
			strconv.FormatUint(v.Res.Quality.NrThrottled, 10), strconv.Itoa(v.Host.Samples), strconv.FormatBool(v.Host.Covered),
			strconv.FormatBool(v.Host.Throttled), strconv.FormatBool(v.Host.OnBattery), strconv.FormatBool(v.Host.Alert),
			f4(v.CanaryBefore), f4(v.CanaryAfter), v.CoreBefore, v.CoreAfter, strconv.FormatBool(v.Headline), v.Action})
	}
	return rows
}
