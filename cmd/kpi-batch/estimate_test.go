package main

import (
	"path/filepath"
	"strings"
	"testing"

	"qchain-kpi/internal/quality"
)

// Plans live in one folder per machine (plans/mac, plans/vm, plans/windows).
// An estimate over an empty or wrong folder must fail, not report zero hours.
func TestEstimateNeedsPlanFiles(t *testing.T) {
	dir := trialFolder(t)
	cal, err := calibrate(dir, "mac-m1", quality.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, fileCalib), cal); err != nil {
		t.Fatal(err)
	}
	_, err = estimate(dir, t.TempDir(), "mac-m1")
	if err == nil || !strings.Contains(err.Error(), "no plan files") {
		t.Fatalf("estimate over an empty plans folder: err = %v, want 'no plan files'", err)
	}
}
