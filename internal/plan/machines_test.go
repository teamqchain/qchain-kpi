package plan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Plans live in one folder per machine (plans/README.md). The same file name
// in several folders is a byte-identical copy; a plan whose coverage differs
// on a machine gets its own name.

// machineCPUs: the CPUs a container can be given (the VM's 2 vCPUs; the 8 CPUs
// of Docker's WSL2 VM on the Windows laptop).
var machineCPUs = map[string]int{"vm": 2, "windows": 8}

// vmWindowsMain: the experiments of the VM's and Windows' main plans (decision
// of 21 Oct, docs/methodology.md section 9). E3, E5 to E8, E13, E14 and E11's
// all-pairs sweep are the Mac's only.
var vmWindowsMain = []string{"E1", "E10", "E11", "E12", "E2", "E2c", "E4", "E9"}

func machinePlans(t *testing.T, machine string) map[string]*Plan {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "plans", machine, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("plans/%s: %d plans (%v)", machine, len(files), err)
	}
	out := map[string]*Plan{}
	for _, f := range files {
		p, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name+".json" != filepath.Base(f) {
			t.Fatalf("%s: name %q", f, p.Name)
		}
		if bl, err := Expand(p, 1); err != nil || len(bl) == 0 {
			t.Fatalf("%s: %d blocks (%v)", f, len(bl), err)
		}
		out[p.Name] = p
	}
	return out
}

// TestMachinePlanCopies: every plan loads and expands, and a plan found in
// several folders is the same file.
func TestMachinePlanCopies(t *testing.T) {
	machines := []string{"mac", "vm", "windows"}
	for _, m := range machines {
		machinePlans(t, m)
	}
	for i, a := range machines {
		for _, b := range machines[i+1:] {
			files, _ := filepath.Glob(filepath.Join("..", "..", "plans", a, "*.json"))
			for _, f := range files {
				g := filepath.Join("..", "..", "plans", b, filepath.Base(f))
				x, _ := os.ReadFile(f)
				y, err := os.ReadFile(g)
				if err == nil && !bytes.Equal(x, y) {
					t.Errorf("plans/%s/%s differs from plans/%s's copy: give a changed plan its own name",
						b, filepath.Base(f), a)
				}
			}
		}
	}
}

// TestVMWindowsScope: the VM's and Windows' plans hold the 21 Oct scope and
// no more: E2 without SLH-DSA, E11 on the curated suites, E12 on CPUs the
// machine has (in a plan named for the machine), and a main plan for every
// experiment of the scope. Each add-on trial probes exactly its main plan's
// cells, which the main run needs classified.
func TestVMWindowsScope(t *testing.T) {
	for _, m := range []string{"vm", "windows"} {
		plans := machinePlans(t, m)
		var main []string
		for _, p := range plans {
			for _, it := range p.Items {
				if !IsProbe(it.Exp) && !slices.Contains(vmWindowsMain, it.Exp) {
					t.Errorf("plans/%s/%s: %s is not in the machine's scope", m, p.Name, it.Exp)
				}
				if p.Kind == KindMain && !slices.Contains(main, it.Exp) {
					main = append(main, it.Exp)
				}
				switch it.Exp {
				case "probe-E6", "probe-E7":
					t.Errorf("plans/%s/%s: %s probes a Mac-only experiment", m, p.Name, it.Exp)
				case "E2", "E2c", "probe-E2":
					for _, a := range mustExpand(it.Algs) {
						if a.Family == "SLH-DSA" {
							t.Errorf("plans/%s/%s: %s includes %s", m, p.Name, it.Exp, a.ID)
						}
					}
				case "E11", "probe-E11":
					if it.Suites != SuitesCurated {
						t.Errorf("plans/%s/%s: %s is not on the curated suites", m, p.Name, it.Exp)
					}
				}
			}
			if p.CPUSet != "" {
				if top := maxCPU(p.CPUSet); top >= machineCPUs[m] || !strings.HasSuffix(p.Name, "-"+m) {
					t.Errorf("plans/%s/%s: cpuset %q (CPUs 0-%d; a plan with CPUs is named for its machine)",
						m, p.Name, p.CPUSet, machineCPUs[m]-1)
				}
			}
		}
		slices.Sort(main)
		if want := slices.Sorted(slices.Values(vmWindowsMain)); !slices.Equal(main, want) {
			t.Errorf("plans/%s: main plans cover %v, want %v", m, main, want)
		}
		for _, c := range []struct{ trial, probe, main, exp string }{
			{"trial-e2-noslh", "probe-E2", "e2-noslh", "E2"},
			{"trial-e2-noslh", "probe-E2", "e2-noslh", "E2c"},
			{"trial-e11-curated", "probe-E11", "e11-curated", "E11"},
		} {
			if got, want := planCells(t, plans[c.trial], c.probe), planCells(t, plans[c.main], c.exp); !slices.Equal(got, want) || len(got) == 0 {
				t.Errorf("plans/%s: %s probes %d cells, %s's %s has %d", m, c.trial, len(got), c.main, c.exp, len(want))
			}
		}
	}
}

// planCells: the sorted (algorithm, operation) cells of one experiment of a plan.
func planCells(t *testing.T, p *Plan, exp string) []string {
	t.Helper()
	if p == nil {
		t.Fatalf("missing plan for %s", exp)
	}
	bl, err := Expand(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, b := range bl {
		if b.Exp == exp && b.Rep == 1 {
			out = append(out, b.Alg+" "+b.Op)
		}
	}
	slices.Sort(out)
	return out
}

// maxCPU is the highest CPU in a docker CPU list (Load has checked its form).
func maxCPU(list string) int {
	top := -1
	for _, f := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == '-' }) {
		if n, err := strconv.Atoi(f); err == nil && n > top {
			top = n
		}
	}
	return top
}

// macPlans: the Mac's plans as they ran (all its runs are done): the first 16
// hex digits of the SHA-256 of the file and of its expansion (every block's
// identity, order and seed; batch seed 1).
var macPlans = map[string][2]string{
	"e1":          {"cb88aa8bd4702581", "a8f6afdb0f7cab16"}, // 450 blocks
	"e10":         {"59b261d41f4e15f5", "b90754b14f2b83e8"}, // 15 blocks
	"e11":         {"60f8fcce94ccca83", "17f0950a5cb2be16"}, // 4955 blocks
	"e12":         {"bccec22dbd94b186", "c4ff2d2371ca6693"}, // 120 blocks
	"e13":         {"dc70970224ef038b", "0fe4f9eb3c3cd23b"}, // 30 blocks
	"e2":          {"0b3ab8a0ee0fdd04", "2096cd11c5945b53"}, // 900 blocks
	"e3":          {"456561eac6ef3e5c", "25333ce136d49574"}, // 455 blocks
	"e4":          {"963d59af5f0edd63", "041ffce0103a5723"}, // 450 blocks
	"e5":          {"817f084b19a3e1e1", "3562416cb50473ce"}, // 450 blocks
	"e6":          {"0e603dc48dd2c85e", "8ecb605560b6da18"}, // 1100 blocks
	"e7":          {"481028be1ebada6a", "49172fbe865c9f9e"}, // 150 blocks
	"e8":          {"3ec32128b57d5810", "c41e3856da2969d6"}, // 60 blocks
	"e9":          {"ad361d51ba16996a", "101543e32a633a5e"}, // 1350 blocks
	"smoke-e11":   {"e8277a53f1935aa9", "99f6e3b43c9c3a5e"}, // 250 blocks
	"smoke-e12":   {"f12e22799586fbc3", "8b9c7f3454f400f2"}, // 8 blocks
	"smoke-e13":   {"f4278d643b605204", "fabdf99fcf42d56f"}, // 30 blocks
	"smoke-e2":    {"230038bea3d6f8e0", "bf61ec02f41817c7"}, // 360 blocks
	"smoke-e3":    {"5d00d0b67777ec4e", "2a16aa6ad0c4a3c5"}, // 61 blocks
	"smoke-e4":    {"a4d52f5ba315271e", "f6282095d641aa45"}, // 36 blocks
	"smoke-e5":    {"98a0748620734297", "dd261ec85a60262f"}, // 42 blocks
	"smoke-e6":    {"ad1d1b7358018b43", "70871ba2eadbf535"}, // 440 blocks
	"smoke-e7":    {"fedb023f8c7be3c8", "b5d94589f930b83c"}, // 60 blocks
	"smoke-e8":    {"9e8ad517a1b7c825", "2dcf29543d9ca143"}, // 60 blocks
	"smoke-e9":    {"1df5ee880c259abf", "f265d02596ec01b3"}, // 54 blocks
	"smoke-quick": {"936d20e03350961d", "482aaa507a8591da"}, // 56 blocks
	"smoke":       {"d7f55e146b1c6f21", "27c2dab35d5c899c"}, // 276 blocks
	"trial-e11":   {"becfa94ee4c97f13", "b371f80136d174e1"}, // 991 blocks
	"trial-e2":    {"aecf48db1d56ef56", "0cb6897f2c132f91"}, // 90 blocks
	"trial-e6":    {"f4a611d801a10e88", "c3ffe0f8adc57d8f"}, // 220 blocks
	"trial-e7":    {"ede75ca646c40c42", "62c2951177cdb36f"}, // 30 blocks
	"trial":       {"99c8955bf949d866", "74aaca33cdcda3f7"}, // 193 blocks
}

// TestMacPlansUnchanged: the Mac's plans stay as they ran. A changed Mac plan,
// or plan code that changes how one expands, fails here; plans for the other
// machines go in their own folders.
func TestMacPlansUnchanged(t *testing.T) {
	plans := machinePlans(t, "mac")
	if len(plans) != len(macPlans) {
		t.Errorf("plans/mac has %d plans, %d ran", len(plans), len(macPlans))
	}
	for name, p := range plans {
		bl, _ := Expand(p, 1)
		js, err := json.Marshal(bl)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(js)
		got := [2]string{p.Hash[:16], hex.EncodeToString(sum[:8])}
		if got != macPlans[name] {
			t.Errorf("plans/mac/%s.json: file %s, expansion %s; it ran as %v", name, got[0], got[1], macPlans[name])
		}
	}
}
