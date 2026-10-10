// Package procsetup holds the start-up steps every measuring process takes.
package procsetup

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/metrics"

	"qchain-kpi/internal/buildinfo"
)

// GoGate refuses to run under any toolchain other than the pinned one, so all
// three machines measure the same Go runtime and code generator.
func GoGate() error {
	if v := runtime.Version(); v != buildinfo.ExpectedGoVersion {
		return fmt.Errorf("built with %s, expected %s", v, buildinfo.ExpectedGoVersion)
	}
	return nil
}

// MicroGC applies the micro-benchmark GC policy (coding plan §6): automatic
// garbage collection off, with a 1 GiB soft memory limit as a safety net only.
// The timing package forces untimed collections before sampling and every
// GCEvery samples, so the heap never approaches the limit.
func MicroGC() {
	debug.SetGCPercent(-1)
	debug.SetMemoryLimit(1 << 30)
}

// OOMAdj raises this process's oom_score_adj to the maximum (Linux only;
// raising it needs no privilege). If the container runs out of memory, the
// kernel then kills the measuring process, never the orchestrator.
func OOMAdj() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return os.WriteFile("/proc/self/oom_score_adj", []byte("1000"), 0) == nil
}

var autoGC = []metrics.Sample{{Name: "/gc/cycles/automatic:gc-cycles"}}

// AutoGCCycles returns how many GC cycles the runtime has started by itself
// (forced runtime.GC() calls are not counted).
func AutoGCCycles() uint64 {
	metrics.Read(autoGC)
	if autoGC[0].Value.Kind() != metrics.KindUint64 {
		return 0
	}
	return autoGC[0].Value.Uint64()
}
