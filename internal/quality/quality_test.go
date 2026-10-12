package quality

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiffStealOtherThrottle(t *testing.T) {
	dir := t.TempDir()
	ProcStatPath = filepath.Join(dir, "stat")
	CgroupDir = dir
	defer func() { ProcStatPath, CgroupDir = "/proc/stat", "/sys/fs/cgroup" }()
	// 2 CPUs. Between the snapshots: cpu1 has 100 jiffies of which 4 steal;
	// VM-wide busy rises by 120 jiffies (1.2 s); container used 1.0 s.
	writeFile(t, ProcStatPath, "cpu  1000 0 0 1000 0 0 0 0 0 0\ncpu0 500 0 0 500 0 0 0 0 0 0\ncpu1 500 0 0 500 0 0 0 0 0 0\n")
	writeFile(t, filepath.Join(dir, "cpu.stat"), "usage_usec 1000000\nnr_throttled 0\nthrottled_usec 0\n")
	a, err := Take()
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, ProcStatPath, "cpu  1120 0 0 1076 0 0 0 4 0 0\ncpu0 520 0 0 580 0 0 0 0 0 0\ncpu1 600 0 0 496 0 0 0 4 0 0\n")
	writeFile(t, filepath.Join(dir, "cpu.stat"), "usage_usec 2000000\nnr_throttled 3\nthrottled_usec 900\n")
	b, err := Take()
	if err != nil {
		t.Fatal(err)
	}
	b.mono = a.mono.Add(1 * time.Second)
	iv := Diff(a, b, []int{1})
	if math.Abs(iv.StealPct-4) > 1e-9 {
		t.Fatalf("steal %v, want 4%%", iv.StealPct)
	}
	// busy 1.2 s on 2 CPUs over 1 s = 60 %; other = (1.2-1.0)/2 = 10 %.
	if math.Abs(iv.VMBusyPct-60) > 1e-9 || math.Abs(iv.OtherPct-10) > 1e-9 {
		t.Fatalf("busy %v other %v", iv.VMBusyPct, iv.OtherPct)
	}
	if iv.NrThrottled != 3 || iv.ThrottledUS != 900 {
		t.Fatalf("throttle %+v", iv)
	}
}

func TestParseCPUList(t *testing.T) {
	got, err := ParseCPUList("0-2,5")
	if err != nil || len(got) != 4 || got[3] != 5 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestHostChunksAndOverlap(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "host-1.csv"), "utc_ns,other_pct,throttle,on_ac,alert,detail\n"+
		"1000,1.0,0,1,0,a\n6000,9.5,0,1,0,b\n11000,2.0,1,1,0,c\n")
	writeFile(t, filepath.Join(dir, "host-1.ok"), "")
	// An unmarked chunk must be ignored.
	writeFile(t, filepath.Join(dir, "host-2.csv"), "utc_ns,other_pct,throttle,on_ac,alert,detail\n16000,50,0,1,0,x\n")
	s, bad, err := ReadHostChunks(dir)
	if err != nil || bad != 0 || len(s) != 3 {
		t.Fatalf("samples %d bad %d err %v", len(s), bad, err)
	}
	// Block [2000, 7000] with zero offset and 5000 period: samples at 6000
	// (covers 1000..6000) and 11000 (covers 6000..11000) overlap.
	v := HostOver(s, 2000, 7000, 0, 5000)
	if v.Samples != 2 || v.MaxOtherPct != 9.5 || !v.Throttled || !v.Covered {
		t.Fatalf("verdict %+v", v)
	}
}

func TestCalibrationFormulas(t *testing.T) {
	_, lim := DriftFromTrial(map[string][]float64{"a": {100, 100, 100, 100, 100}})
	if lim != DefaultDriftFrac {
		t.Fatalf("quiet trial must keep the 3%% floor, got %v", lim)
	}
	d95, lim := DriftFromTrial(map[string][]float64{"a": {90, 100, 110}})
	if math.Abs(d95-0.1) > 1e-9 || math.Abs(lim-0.2) > 1e-9 {
		t.Fatalf("d95 %v lim %v", d95, lim)
	}
	_, c := CanaryFromTrial([]float64{100, 100, 100, 100})
	if c != DefaultCanaryFrac {
		t.Fatalf("canary %v", c)
	}
	f, m := AAFloor([]float64{100, 101, 99, 100, 100}, []float64{100, 100, 100, 101, 99})
	if !(f > 0 && m >= DefaultE10Margin) {
		t.Fatalf("A/A floor %v margin %v", f, m)
	}
	if Classify([]int64{500000, 600000, 2000000}) != "fast" || Classify([]int64{2e6, 3e6, 1e6}) != "slow" {
		t.Fatal("classification")
	}
}

// TestCommittedThresholdsFile checks config/thresholds.json: it parses, it
// carries the formulas implemented here, and every calibrated section is
// complete (a main run would refuse an incomplete one anyway).
func TestCommittedThresholdsFile(t *testing.T) {
	f, err := Load("../../config/thresholds.json")
	if err != nil {
		t.Fatal(err)
	}
	if f.Formulas != FormulaText {
		t.Fatal("config/thresholds.json formulas differ from quality.FormulaText")
	}
	for name, m := range f.Machines {
		if m.CalibratedFrom == "" || m.DriftFrac < DefaultDriftFrac || m.CanaryFrac < DefaultCanaryFrac ||
			m.StealPct != DefaultStealPct || m.OtherPct != DefaultOtherPct || len(m.E10Margin) != 3 {
			t.Fatalf("machine %s: incomplete or out-of-rule section", name)
		}
		// The 90 algorithm x operation classifications of the first trial; any
		// further entries (E11 cells) need the add-on calibration that made them.
		base := 0
		for k := range m.Class {
			alg, op, ok := strings.Cut(k, "/")
			if ok && !strings.Contains(alg, "+") && (op == "keygen" || op == "sign" || op == "verify" || op == "encaps" || op == "decaps") {
				base++
			}
		}
		if base != 90 {
			t.Fatalf("machine %s: %d base classifications, want 90", name, base)
		}
		if len(m.Class) > base && m.ClassCalibratedFrom["E11"] == "" {
			t.Fatalf("machine %s: E11 classifications without class_calibrated_from.E11", name)
		}
		for set, v := range m.E10Margin {
			if v < DefaultE10Margin {
				t.Fatalf("machine %s: E10 margin of %s below the 2%% floor", name, set)
			}
		}
	}
}
