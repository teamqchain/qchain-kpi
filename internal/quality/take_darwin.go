//go:build darwin

package quality

// Native macOS (E5): per-CPU tick counters from the Mach kernel
// (host_processor_info, PROCESSOR_CPU_LOAD_INFO: user, system, idle, nice)
// and this process's own CPU time from getrusage. macOS has no steal, iowait,
// irq or softirq counters and no cgroup: those stay 0. The ticks are scaled
// to /proc/stat's unit (USER_HZ = 100) using the kernel's clock rate
// (kern.clockrate: hz), so Diff works unchanged.

/*
#include <mach/mach.h>
#include <mach/processor_info.h>
#include <mach/mach_host.h>
#include <sys/resource.h>
#include <sys/sysctl.h>

// kpi_cpu_ticks fills user/system/idle/nice ticks for up to max CPUs and
// returns the number of CPUs (or -1).
static int kpi_cpu_ticks(unsigned long long *out, int max) {
	natural_t n = 0;
	processor_info_array_t info;
	mach_msg_type_number_t count;
	if (host_processor_info(mach_host_self(), PROCESSOR_CPU_LOAD_INFO, &n, &info, &count) != KERN_SUCCESS) {
		return -1;
	}
	int m = (int)n < max ? (int)n : max;
	for (int i = 0; i < m; i++) {
		processor_cpu_load_info_t l = (processor_cpu_load_info_t)info + i;
		out[4 * i + 0] = l->cpu_ticks[CPU_STATE_USER];
		out[4 * i + 1] = l->cpu_ticks[CPU_STATE_SYSTEM];
		out[4 * i + 2] = l->cpu_ticks[CPU_STATE_IDLE];
		out[4 * i + 3] = l->cpu_ticks[CPU_STATE_NICE];
	}
	vm_deallocate(mach_task_self(), (vm_address_t)info, (vm_size_t)count * sizeof(int));
	return (int)n;
}

static long long kpi_self_cpu_us(void) {
	struct rusage ru;
	if (getrusage(RUSAGE_SELF, &ru) != 0) return -1;
	return (long long)ru.ru_utime.tv_sec * 1000000LL + ru.ru_utime.tv_usec +
	       (long long)ru.ru_stime.tv_sec * 1000000LL + ru.ru_stime.tv_usec;
}

static int kpi_clock_hz(void) {
	struct clockinfo ci;
	size_t len = sizeof ci;
	if (sysctlbyname("kern.clockrate", &ci, &len, NULL, 0) != 0) return -1;
	return ci.hz;
}
*/
import "C"

import (
	"errors"
	"runtime"
	"time"
)

const maxCPUs = 256

// Take reads the per-CPU tick counters and this process's CPU time.
func Take() (Snapshot, error) {
	s := Snapshot{UnixNS: time.Now().UnixNano(), mono: time.Now(), PerCPU: map[int]CPUTimes{}}
	hz := int(C.kpi_clock_hz())
	if hz <= 0 {
		return s, errors.New("quality: kern.clockrate unavailable")
	}
	var buf [4 * maxCPUs]C.ulonglong
	n := int(C.kpi_cpu_ticks(&buf[0], maxCPUs))
	if n <= 0 {
		return s, errors.New("quality: host_processor_info failed")
	}
	scale := func(t C.ulonglong) uint64 { return uint64(t) * userHZ / uint64(hz) }
	for i := 0; i < n && i < maxCPUs; i++ {
		t := CPUTimes{User: scale(buf[4*i]), System: scale(buf[4*i+1]), Idle: scale(buf[4*i+2]), Nice: scale(buf[4*i+3])}
		s.PerCPU[i] = t
		s.All.User += t.User
		s.All.System += t.System
		s.All.Idle += t.Idle
		s.All.Nice += t.Nice
	}
	if us := int64(C.kpi_self_cpu_us()); us >= 0 {
		s.UsageUS = uint64(us)
	}
	return s, nil
}

// CPUSet returns every logical CPU: macOS cannot pin a process to CPUs.
func CPUSet() ([]int, error) {
	out := make([]int, runtime.NumCPU())
	for i := range out {
		out[i] = i
	}
	return out, nil
}
