// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/config.go line 22 (time/tzdata import) and lines 102-110 (mustLoadLocation).
// Only the package clause changed (main -> qchain) and: the import block lists only what this function uses.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain

import (
	"time"

	// Same as production: the zone database is embedded so Asia/Dubai
	// resolves inside a minimal container.
	_ "time/tzdata"
)

// mustLoadLocation loads a timezone (e.g. "Asia/Dubai"), falling back to UTC if
// the zone database is unavailable. Used so issuance timestamps are local time.
func mustLoadLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}
