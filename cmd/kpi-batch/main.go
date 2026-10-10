// Command kpi-batch is the orchestrator that runs inside the measurement
// container (it is the image's entry point):
//
//	kpi-batch run       run (or resume) a batch from a plan into /out
//	kpi-batch dry-run   print the block order a plan and seed produce
//	kpi-batch calibrate derive one machine's thresholds from a trial run
//	kpi-batch estimate  project run time and data size of plans from a trial run
//	kpi-batch verify    check a run folder against its SHA256SUMS
//	kpi-batch clock     print this container's wall clock (Unix ns), for the
//	                    launcher's host-to-container clock offset
//
// It never measures anything itself: every block, prep and canary is a fresh
// kpi-prim process (coding plan §2).
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	code := 0
	switch os.Args[1] {
	case "run":
		code, err = cmdRun(os.Args[2:])
	case "dry-run":
		err = cmdDryRun(os.Args[2:])
	case "calibrate":
		err = cmdCalibrate(os.Args[2:])
	case "estimate":
		err = cmdEstimate(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "clock":
		fmt.Println(time.Now().UnixNano())
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "kpi-batch: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: kpi-batch run|dry-run|calibrate|estimate|verify|clock [flags]")
	os.Exit(2)
}
