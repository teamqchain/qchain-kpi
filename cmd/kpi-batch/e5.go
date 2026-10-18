package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"syscall"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E5 (platform and runtime comparison): E1's cells run natively on the host
// (scripts/native-mac.sh on the Mac). The container's experiments and E5 must
// not mix: in the container (Linux) E5 is refused, and natively only E5 (and
// the probe pass of its smoke plan) may run, because every other experiment's
// rules assume the container (one pinned CPU, cgroup limits, no network).
func e5PlatformCheck(blocks []plan.Block, goos string) error {
	native := goos != "linux"
	for _, b := range blocks {
		e5 := b.Exp == "E5"
		switch {
		case e5 && !native:
			return fmt.Errorf("run: E5 runs natively on the host (scripts/native-mac.sh), not in the container")
		case native && !e5 && !proto.IsProbe(b.Exp):
			return fmt.Errorf("run: natively only E5 may run (block %s); every other experiment runs in the container", b.ID)
		}
	}
	return nil
}

// preflightNative checks a native run (E5): no container exists, so only the
// process's own settings are checked here; the machine itself (power,
// thermal pressure, quiet CPU, closed apps, Colima stopped) is checked by
// the native launcher before it starts the batch.
func preflightNative(out string) (map[string]string, []string) {
	facts := map[string]string{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "num_cpu": strconv.Itoa(runtime.NumCPU())}
	var problems []string
	facts["gomaxprocs"] = strconv.Itoa(runtime.GOMAXPROCS(0))
	if runtime.GOMAXPROCS(0) != 1 {
		problems = append(problems, fmt.Sprintf("GOMAXPROCS is %d, want 1", runtime.GOMAXPROCS(0)))
	}
	if os.Getuid() == 0 {
		problems = append(problems, "running as root")
	}
	facts["tz"] = os.Getenv("TZ")
	if os.Getenv("TZ") != "Asia/Dubai" {
		problems = append(problems, fmt.Sprintf("TZ is %q, want Asia/Dubai (production parity)", os.Getenv("TZ")))
	}
	var sfs syscall.Statfs_t
	if syscall.Statfs(out, &sfs) == nil {
		free := sfs.Bavail * uint64(sfs.Bsize)
		facts["out_free_gb"] = strconv.FormatFloat(float64(free)/1e9, 'f', 1, 64)
		if free < 2e9 {
			problems = append(problems, "less than 2 GB free for the run folder")
		}
	}
	return facts, problems
}
