package plan

import (
	"strings"
	"testing"
)

// TestE12Expansion: 4 suites x 2 stages x 3 worker counts x 5 repetitions on
// 4 CPUs; the smoke plan 2 suites x 2 stages x 2 worker counts.
func TestE12Expansion(t *testing.T) {
	p, err := Load("../../plans/e12.json")
	if err != nil {
		t.Fatal(err)
	}
	bl, err := Expand(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	ws := map[string]int{}
	for _, b := range bl {
		if b.Exp != "E12" || !strings.Contains(b.Alg, "+") {
			t.Fatalf("block %+v", b)
		}
		ws[b.Op[strings.LastIndex(b.Op, "-w"):]]++
	}
	if len(bl) != 4*2*3*5 || ws["-w1"] != 40 || ws["-w2"] != 40 || ws["-w4"] != 40 || p.CPUSet != "0-3" {
		t.Fatalf("%d blocks %v cpuset %q", len(bl), ws, p.CPUSet)
	}
	if bl[0].ID != "E12."+bl[0].Alg+"."+bl[0].Op+".r1" {
		t.Fatalf("block id %q", bl[0].ID)
	}
	s, err := Load("../../plans/smoke-e12.json")
	if err != nil {
		t.Fatal(err)
	}
	if sb, err := Expand(s, 1); err != nil || len(sb) != 2*2*2 {
		t.Fatalf("smoke-e12: %d blocks (%v)", len(sb), err)
	}
}

// TestCPUSetRules: E12 needs its CPUs named and enough of them; several CPUs
// are for E12 plans only.
func TestCPUSetRules(t *testing.T) {
	e12 := Item{Exp: "E12", Algs: []string{"all"}, Workers: []int{1, 2, 4}}
	e1 := Item{Exp: "E1", Algs: []string{"ml-dsa-44"}}
	for _, c := range []struct {
		p  Plan
		ok bool
	}{
		{Plan{Items: []Item{e12}, CPUSet: "0-3"}, true},
		{Plan{Items: []Item{e12}}, false},                    // no cpuset
		{Plan{Items: []Item{e12}, CPUSet: "0-1"}, false},     // 4 workers, 2 CPUs
		{Plan{Items: []Item{e12, e1}, CPUSet: "0-3"}, false}, // E1 would run unpinned
		{Plan{Items: []Item{e1}}, true},                      // the default single CPU
		{Plan{Items: []Item{{Exp: "E12", Algs: []string{"all"}, Workers: []int{2, 2}}}, CPUSet: "0-3"}, false},
	} {
		if err := cpusetCheck(&c.p); (err == nil) != c.ok {
			t.Fatalf("%+v: %v", c.p, err)
		}
	}
	for _, c := range []struct {
		s string
		n int
	}{{"1", 1}, {"0-3", 4}, {"0,2", 2}, {"0-1,3", 3}} {
		if n, err := CPUCount(c.s); err != nil || n != c.n {
			t.Fatalf("CPUCount(%q) = %d, %v", c.s, n, err)
		}
	}
}
