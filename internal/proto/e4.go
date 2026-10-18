package proto

// E4: build sensitivity (internal/plan/e4.go). A block is E1's: one process
// per (algorithm, operation), the reuse arm, the first call cold; only the
// image's liboqs build differs. The baseline is E1 itself, measured on the
// production-like build (b1-prod), so E4 refuses to run on that build.

// E4BaselineVariant is the build E4 is compared against (E1's).
const E4BaselineVariant = "b1-prod"
