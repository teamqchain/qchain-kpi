//go:build linux

package quality

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
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
