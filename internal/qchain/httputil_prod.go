// Code copied VERBATIM from the QChain backend (main repo, commit 9026aba).
// Source: offchain/httputil.go lines 37-40 (decodeBody).
// Only the package clause changed (main -> qchain) and: the import block lists only what it uses.
// Do not edit function bodies here; docs/provenance.md lists every copy and
// scripts/check-sync.sh compares them with the main repository.
package qchain

import (
	"encoding/json"
	"net/http"
)

// decodeBody parses the JSON request body into `v` (pass a pointer, e.g. &req).
func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
