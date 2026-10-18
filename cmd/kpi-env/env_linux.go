//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// Linux facts (inside the container): /proc and /sys.

func kernel() Kernel {
	k := Kernel{Release: readTrim("/proc/sys/kernel/osrelease"), Version: readTrim("/proc/sys/kernel/version")}
	var u syscall.Utsname
	if syscall.Uname(&u) == nil {
		var sb strings.Builder
		for _, c := range u.Machine {
			if c == 0 {
				break
			}
			sb.WriteByte(byte(c))
		}
		k.Machine = sb.String() // only the machine field; never the node name
	}
	return k
}

// cpuInfo parses /proc/cpuinfo.
func cpuInfo() CPU {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return CPU{}
	}
	defer f.Close()
	var procs []map[string]string
	cur := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			if len(cur) > 0 {
				procs = append(procs, cur)
				cur = map[string]string{}
			}
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		cur[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(cur) > 0 {
		procs = append(procs, cur)
	}
	c := CPU{Fields: map[string]string{}, Distinct: map[string]string{}}
	for _, p := range procs {
		if _, ok := p["processor"]; ok {
			c.Logical++
		}
	}
	if len(procs) == 0 {
		return c
	}
	// Fields that vary per reading or per core and carry no configuration
	// information are skipped.
	skip := map[string]bool{"processor": true, "cpu MHz": true, "core id": true, "apicid": true, "initial apicid": true}
	p0 := procs[0]
	for k, v := range p0 {
		if skip[k] {
			continue
		}
		switch k {
		case "flags", "Features":
			c.Flags = strings.Fields(v)
		default:
			c.Fields[k] = v
		}
	}
	// Machine-wide lines (arm64 kernels print "Hardware" etc. after the last processor).
	for _, p := range procs[1:] {
		if _, ok := p["processor"]; !ok {
			for k, v := range p {
				c.Fields[k] = v
			}
		}
	}
	for _, p := range procs {
		if _, ok := p["processor"]; !ok {
			continue
		}
		for k, v := range p {
			if skip[k] || k == "flags" || k == "Features" || k == "bogomips" || k == "BogoMIPS" {
				continue
			}
			if v != p0[k] {
				c.Distinct[k] = "differs between processors"
			}
		}
	}
	for i := 0; i < 8; i++ {
		d := fmt.Sprintf("/sys/devices/system/cpu/cpu0/cache/index%d", i)
		lv := readTrim(d + "/level")
		if lv == "" {
			continue
		}
		c.Caches = append(c.Caches, fmt.Sprintf("L%s %s %s", lv, readTrim(d+"/type"), readTrim(d+"/size")))
	}
	return c
}

func memTotalKB() uint64 {
	for _, l := range strings.Split(readTrim("/proc/meminfo"), "\n") {
		if strings.HasPrefix(l, "MemTotal:") {
			var kb uint64
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(l, "MemTotal:")), "%d", &kb)
			return kb
		}
	}
	return 0
}
