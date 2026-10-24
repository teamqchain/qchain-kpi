package main

import (
	"fmt"

	"qchain-kpi/internal/plan"
	"qchain-kpi/internal/proto"
)

// E4 (build sensitivity): its baseline is E1 on the production-like build, so
// a trial or main E4 run on that same build would only repeat E1. Smoke runs
// may use any build (they test the pipeline).
func e4VariantCheck(p *plan.Plan, blocks []plan.Block, build map[string]string) error {
	if p.Kind == plan.KindSmoke {
		return nil
	}
	hasE4 := false
	for _, b := range blocks {
		hasE4 = hasE4 || b.Exp == "E4"
	}
	if !hasE4 {
		return nil
	}
	v := build["variant"]
	if v == "" {
		return fmt.Errorf("run: E4 needs the image's liboqs build variant (oqs-build.txt), which is missing")
	}
	if v == proto.E4BaselineVariant {
		return fmt.Errorf("run: E4 compares other liboqs builds with E1's %s; start it with the launcher's --variant (b2-portable, b3-native or b4-noossl)", v)
	}
	return nil
}
