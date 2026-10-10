// Package quality measures how quiet the machine was during each block and
// decides which blocks are flagged (coding plan §8).
//
// In-container signals (Linux kernel of the VM, or of the Colima/WSL2 VM):
//   - steal: % of the pinned vCPU's time taken by the hypervisor for others;
//   - other CPU: VM-wide busy time minus this container's own CPU time;
//   - CPU-quota throttling of this container (cgroup v2 cpu.stat).
//
// Busy = user + nice + system + irq + softirq (guest time is already counted
// in user/nice by the kernel; iowait and steal are not "busy").
package quality

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// userHZ is the unit of /proc/stat (USER_HZ is 100 on every Linux arm64/x86-64).
const userHZ = 100

// CPUTimes are the /proc/stat counters of one CPU line (jiffies).
type CPUTimes struct {
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

// Busy returns user+nice+system+irq+softirq.
func (c CPUTimes) Busy() uint64 { return c.User + c.Nice + c.System + c.IRQ + c.SoftIRQ }

// Total returns all accounted time (guest is inside user/nice).
func (c CPUTimes) Total() uint64 {
	return c.User + c.Nice + c.System + c.Idle + c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

// Snapshot is one reading of the counters.
type Snapshot struct {
	UnixNS      int64
	mono        time.Time
	All         CPUTimes
	PerCPU      map[int]CPUTimes
	UsageUS     uint64 // own cgroup cpu.stat usage_usec
	NrThrottled uint64
	ThrottledUS uint64
}

// Paths can be overridden in tests.
var (
	ProcStatPath   = "/proc/stat"
	CgroupDir      = "/sys/fs/cgroup"
)

// Take reads /proc/stat and the container's cgroup cpu.stat.
func Take() (Snapshot, error) {
	s := Snapshot{UnixNS: time.Now().UnixNano(), mono: time.Now(), PerCPU: map[int]CPUTimes{}}
	f, err := os.Open(ProcStatPath)
	if err != nil {
		return s, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		var v [8]uint64
		for i := 0; i < 8; i++ {
			v[i], _ = strconv.ParseUint(fields[i+1], 10, 64)
		}
		t := CPUTimes{v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7]}
		if fields[0] == "cpu" {
			s.All = t
		} else if n, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu")); err == nil {
			s.PerCPU[n] = t
		}
	}
	if err := sc.Err(); err != nil {
		return s, err
	}
	if b, err := os.ReadFile(CgroupDir + "/cpu.stat"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			kv := strings.Fields(l)
			if len(kv) != 2 {
				continue
			}
			n, _ := strconv.ParseUint(kv[1], 10, 64)
			switch kv[0] {
			case "usage_usec":
				s.UsageUS = n
			case "nr_throttled":
				s.NrThrottled = n
			case "throttled_usec":
				s.ThrottledUS = n
			}
		}
	}
	return s, nil
}

// CPUSet returns the CPUs this container may run on (cpuset.cpus.effective),
// e.g. "1" or "0-1,3".
func CPUSet() ([]int, error) {
	b, err := os.ReadFile(CgroupDir + "/cpuset.cpus.effective")
	if err != nil {
		return nil, err
	}
	return ParseCPUList(strings.TrimSpace(string(b)))
}

// ParseCPUList parses the kernel's CPU list format.
func ParseCPUList(s string) ([]int, error) {
	var out []int
	if s == "" {
		return out, nil
	}
	for _, part := range strings.Split(s, ",") {
		if lo, hi, ok := strings.Cut(part, "-"); ok {
			a, err1 := strconv.Atoi(lo)
			b, err2 := strconv.Atoi(hi)
			if err1 != nil || err2 != nil || b < a {
				return nil, fmt.Errorf("bad cpu list %q", s)
			}
			for i := a; i <= b; i++ {
				out = append(out, i)
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("bad cpu list %q", s)
			}
			out = append(out, n)
		}
	}
	return out, nil
}

// Interval is the quality of the time between two snapshots.
type Interval struct {
	WallNS        int64   `json:"wall_ns"`
	StealPct      float64 `json:"steal_pct"`       // on the pinned CPUs
	VMBusyPct     float64 `json:"vm_busy_pct"`     // whole VM
	OtherPct      float64 `json:"other_pct"`       // VM busy minus own container, % of VM capacity
	NrThrottled   uint64  `json:"nr_throttled"`    // CPU-quota throttling events of this container
	ThrottledUS   uint64  `json:"throttled_usec"`
}

// Diff computes the interval between a and b. pinned are the CPUs the
// container runs on.
func Diff(a, b Snapshot, pinned []int) Interval {
	iv := Interval{WallNS: b.mono.Sub(a.mono).Nanoseconds()}
	if iv.WallNS <= 0 {
		return iv
	}
	var dSteal, dTot uint64
	for _, c := range pinned {
		x, ok1 := a.PerCPU[c]
		y, ok2 := b.PerCPU[c]
		if !ok1 || !ok2 {
			continue
		}
		dSteal += y.Steal - x.Steal
		dTot += y.Total() - x.Total()
	}
	if dTot > 0 {
		iv.StealPct = 100 * float64(dSteal) / float64(dTot)
	}
	ncpu := len(b.PerCPU)
	if ncpu > 0 {
		busyUS := float64(b.All.Busy()-a.All.Busy()) * (1e6 / userHZ)
		capUS := float64(iv.WallNS) / 1e3 * float64(ncpu)
		iv.VMBusyPct = 100 * busyUS / capUS
		own := float64(b.UsageUS - a.UsageUS)
		other := busyUS - own
		if other < 0 {
			other = 0
		}
		iv.OtherPct = 100 * other / capUS
	}
	iv.NrThrottled = b.NrThrottled - a.NrThrottled
	iv.ThrottledUS = b.ThrottledUS - a.ThrottledUS
	return iv
}
