//go:build darwin

package main

import (
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

// macOS facts (E5's native runs), from sysctl. Only hardware and OS facts
// are read; never the host name or any user information.

func sysctlString(name string) string {
	v, err := unix.Sysctl(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func sysctlInt(name string) (uint64, bool) {
	if v, err := unix.SysctlUint64(name); err == nil {
		return v, true
	}
	if v, err := unix.SysctlUint32(name); err == nil {
		return uint64(v), true
	}
	return 0, false
}

func kernel() Kernel {
	return Kernel{Release: sysctlString("kern.osrelease"), Version: sysctlString("kern.version"), Machine: sysctlString("hw.machine")}
}

// cpuInfo: the processor's name, its core clusters (performance levels) and
// caches, and the ARM features the kernel reports (hw.optional.arm.*).
func cpuInfo() CPU {
	c := CPU{Logical: runtime.NumCPU(), Fields: map[string]string{}, Distinct: map[string]string{}}
	c.Fields["brand_string"] = sysctlString("machdep.cpu.brand_string")
	c.Fields["model"] = sysctlString("hw.model")
	if n, ok := sysctlInt("hw.nperflevels"); ok {
		for l := uint64(0); l < n; l++ {
			p := fmt.Sprintf("hw.perflevel%d.", l)
			name := sysctlString(p + "name")
			cores, _ := sysctlInt(p + "physicalcpu")
			l1d, _ := sysctlInt(p + "l1dcachesize")
			l2, _ := sysctlInt(p + "l2cachesize")
			c.Fields[fmt.Sprintf("perflevel%d", l)] = fmt.Sprintf("%s: %d cores, L1d %d KiB, L2 %d KiB", name, cores, l1d/1024, l2/1024)
		}
		if n > 1 {
			c.Distinct["core_type"] = "performance and efficiency cores"
		}
	}
	if v, ok := sysctlInt("hw.cachelinesize"); ok {
		c.Fields["cachelinesize"] = fmt.Sprint(v)
	}
	if v, ok := sysctlInt("hw.pagesize"); ok {
		c.Fields["pagesize"] = fmt.Sprint(v)
	}
	for _, f := range []string{"FEAT_AES", "FEAT_PMULL", "FEAT_SHA1", "FEAT_SHA256", "FEAT_SHA512", "FEAT_SHA3",
		"FEAT_DotProd", "FEAT_FP16", "FEAT_LSE", "FEAT_CRC32", "AdvSIMD"} {
		if v, ok := sysctlInt("hw.optional.arm." + f); ok && v == 1 {
			c.Flags = append(c.Flags, f)
		} else if v, ok := sysctlInt("hw.optional." + strings.ToLower(f)); ok && v == 1 {
			c.Flags = append(c.Flags, f)
		}
	}
	return c
}

func memTotalKB() uint64 {
	v, _ := sysctlInt("hw.memsize")
	return v / 1024
}
