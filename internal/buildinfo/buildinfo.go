// Package buildinfo carries build-time facts that every binary reports and
// checks at start-up.
package buildinfo

// HarnessCommit is the git commit of the harness sources the binary was built
// from. It is stamped at build time:
//
//	go build -ldflags "-X qchain-kpi/internal/buildinfo.HarnessCommit=<sha>"
var HarnessCommit = "unknown"

const (
	// ExpectedGoVersion is the Go toolchain production builds with
	// (golang:1.24-bookworm resolves to Go 1.24.13). Binaries refuse to run
	// under any other toolchain so all three machines measure the same Go.
	ExpectedGoVersion = "go1.24.13"

	// ExpectedLiboqsVersion is the pinned liboqs release.
	ExpectedLiboqsVersion = "0.16.0"
)
