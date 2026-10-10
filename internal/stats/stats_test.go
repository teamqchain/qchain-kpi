package stats

import (
	"math"
	"testing"
)

func approx(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestQuantileMatchesNumPyLinear(t *testing.T) {
	s := []float64{1, 2, 3, 4}
	// numpy.quantile([1,2,3,4], q) with the default "linear" method.
	cases := map[float64]float64{0: 1, 0.25: 1.75, 0.5: 2.5, 0.9: 3.7, 1: 4}
	for q, want := range cases {
		if got := Quantile(s, q); !approx(got, want, 1e-12) {
			t.Fatalf("Quantile(%v) = %v, want %v", q, got, want)
		}
	}
	if got := Quantile([]float64{7}, 0.99); got != 7 {
		t.Fatalf("single element: %v", got)
	}
}

func TestSummarize(t *testing.T) {
	xs := []float64{10, 20, 30, 40, 50}
	s := Summarize(xs, 0, 1)
	if s.N != 5 || s.Mean != 30 || s.P50 != 30 || s.Min != 10 || s.Max != 50 {
		t.Fatalf("summary %+v", s)
	}
	if !approx(s.SD, 15.811388300841896, 1e-9) {
		t.Fatalf("sd %v", s.SD)
	}
}

func TestBootstrapDeterministic(t *testing.T) {
	xs := make([]float64, 500)
	for i := range xs {
		xs[i] = float64(i % 37)
	}
	a := BootstrapSEMedian(xs, 200, 42)
	b := BootstrapSEMedian(xs, 200, 42)
	if a != b || a <= 0 {
		t.Fatalf("bootstrap not deterministic or zero: %v %v", a, b)
	}
}

func TestRobustSD(t *testing.T) {
	// MAD of {1,2,3,4,100} = median(|x-3|) = median{2,1,0,1,97} = 1.
	if got := RobustSD([]float64{1, 2, 3, 4, 100}); !approx(got, 1.4826, 1e-12) {
		t.Fatalf("RobustSD = %v", got)
	}
}

func TestMeanCI90(t *testing.T) {
	m, h := MeanCI90([]float64{1, 2, 3, 4, 5})
	// sd = 1.5811, t(0.95, 4) = 2.132, half = 2.132*1.5811/sqrt(5) = 1.5075.
	if m != 3 || !approx(h, 1.50749, 1e-4) {
		t.Fatalf("MeanCI90 = %v ± %v", m, h)
	}
}
