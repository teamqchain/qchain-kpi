//go:build !darwin

package procsetup

// Interactive does nothing outside macOS (the container pins the process to
// one CPU instead).
func Interactive() bool { return false }
