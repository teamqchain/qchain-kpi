// Command kpi-canary-native runs the batch canary natively on the host (macOS
// only). The launcher runs it twice per session, once plainly and once under
// `taskpolicy -b` (background policy, which macOS keeps on the efficiency
// cores), to measure the performance-core and efficiency-core reference
// speeds. Every canary inside the container is then classed as P-like or
// E-like against these references (coding plan §8, core-type check).
//
// It is pure Go and built with the host's Go 1.24.13 toolchain:
//
//	GOTOOLCHAIN=local CGO_ENABLED=0 go build -o bin/kpi-canary-native ./cmd/kpi-canary-native
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"qchain-kpi/internal/canary"
	"qchain-kpi/internal/procsetup"
)

type out struct {
	Label  string        `json:"label"`
	GOOS   string        `json:"goos"`
	GOARCH string        `json:"goarch"`
	Go     string        `json:"go"`
	UTC    string        `json:"utc"`
	Canary canary.Result `json:"canary"`
}

func main() {
	label := flag.String("label", "plain", "what this run measures (plain or background)")
	windows := flag.Int("windows", 15, "number of windows")
	windowMS := flag.Int("window-ms", 100, "window length in ms")
	flag.Parse()
	if err := procsetup.GoGate(); err != nil {
		fmt.Fprintf(os.Stderr, "kpi-canary-native: %v\n", err)
		os.Exit(3)
	}
	runtime.LockOSThread()
	r := canary.Run(*windows, time.Duration(*windowMS)*time.Millisecond)
	b, _ := json.Marshal(out{Label: *label, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Go: runtime.Version(),
		UTC: time.Now().UTC().Format(time.RFC3339), Canary: r})
	fmt.Println(string(b))
}
