package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/plan"
)

// Extending a finished run (decision 11 Oct, user): blocks that ended without
// a clean attempt are run again inside the same run folder instead of in a new
// plan. Nothing about this is hidden: the journal, the manifest (extensions and
// sessions) and the state record it, the raw files continue the attempt
// numbering (a4, a5, ...), and each extended block gets a fresh budget of
// maxAttempts attempts. The measuring sources must be identical to the ones the
// run was made with; the launcher checks that with git before it starts, and
// the image and harness commit are then allowed to differ (both are recorded).

var blockIDPattern = regexp.MustCompile(`^(E1|E10|E2|E2c)\.([a-z0-9-]+)\.([a-z]+)\.r([0-9]+)$`)

// withoutBuild returns an identity whose image and harness commit are those of
// the saved batch, so that only the other differences count.
func withoutBuild(now, saved identity) identity {
	now.ImageID, now.HarnessCommit = saved.ImageID, saved.HarnessCommit
	return now
}

// extendBlocks queues one more round of attempts for each listed block of a
// finished batch. Every block must exist and must have no headline attempt.
func (b *batch) extendBlocks(ids []string, oldImage, oldCommit string) error {
	in, err := loadEval(b.c.Out, b.st, b.th, true)
	if err != nil {
		return err
	}
	vs := evaluate(in)
	decideBase(b.st.PlanKind, vs, b.st.Seed, copyMap(b.st.Requeued), copyMap(b.st.Diagnosed), b.st.Base)
	type info struct {
		last     plan.Block
		attempts int
		maxAtt   int
		headline bool
	}
	seen := map[string]*info{}
	for _, v := range vs {
		bl := v.Res.Block
		x := seen[bl.ID]
		if x == nil {
			x = &info{}
			seen[bl.ID] = x
		}
		if bl.Attempt >= x.maxAtt {
			x.maxAtt, x.last = bl.Attempt, bl
		}
		if !bl.Diagnostic {
			x.attempts++
		}
		if v.Headline {
			x.headline = true
		}
	}
	if b.st.Base == nil {
		b.st.Base = map[string]int{}
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !blockIDPattern.MatchString(id) {
			return fmt.Errorf("extend: %q is not a block id like E1.ml-dsa-44.sign.r3", id)
		}
		x := seen[id]
		switch {
		case x == nil:
			return fmt.Errorf("extend: block %s is not in this run", id)
		case x.headline:
			return fmt.Errorf("extend: block %s already has a clean headline attempt", id)
		}
		nb := x.last
		nb.Attempt = x.maxAtt + 1
		nb.Diagnostic = false
		nb.Mode = ""
		nb.Seed = plan.DeriveSeed(b.c.Seed, nb.Key())
		b.st.Queue = append(b.st.Queue, nb)
		b.st.Base[id] = x.attempts
		if err := b.journal("extension_queued", nb.Key(), fmt.Sprintf("block had %d attempt(s), none clean", x.attempts)); err != nil {
			return err
		}
	}
	b.st.Planned += len(ids)
	b.st.Extensions = append(b.st.Extensions, Extension{Session: b.sess.N, Blocks: ids,
		OldImageID: oldImage, OldHarnessCmt: oldCommit, NewImageID: b.c.ImageID, NewHarnessCmt: buildinfo.HarnessCommit,
		OrchestratorChanged: parseBlockList(b.c.OrchChanged)})
	return nil
}

// parseBlockList splits the --extend argument.
func parseBlockList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// extensionNotes lists the extensions for the manifest.
func extensionNotes(st *State) []string {
	var out []string
	for _, e := range st.Extensions {
		out = append(out, "session "+strconv.Itoa(e.Session)+" extended this run: blocks "+strings.Join(e.Blocks, ", ")+
			" had no clean attempt and were run again (attempt numbers continue); input sets were regenerated (same message seeds, new keys)")
		if e.OldImageID != e.NewImageID || e.OldHarnessCmt != e.NewHarnessCmt {
			out = append(out, "image "+shortID(e.OldImageID)+" -> "+shortID(e.NewImageID)+", harness commit "+shortID(e.OldHarnessCmt)+" -> "+shortID(e.NewHarnessCmt)+
				" (measuring sources verified identical by the launcher)")
		}
		if len(e.OrchestratorChanged) > 0 {
			out = append(out, "orchestrator (cmd/kpi-batch) files that differ from the original run's commit: "+strings.Join(e.OrchestratorChanged, ", ")+
				" (they implement the extension; decisions for runs that are not extended are unchanged)")
		}
	}
	return out
}

func shortID(s string) string {
	s = strings.TrimPrefix(s, "sha256:")
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// extendFile is where the launcher's build records of an extension go.
func extendBuildInfoDir(out string, n int) string {
	return filepath.Join(out, dirEnv, fmt.Sprintf("build-info-s%d", n))
}
