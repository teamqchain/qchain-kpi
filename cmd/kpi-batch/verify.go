package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"qchain-kpi/internal/plan"
)

func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errors.New("verify: --run is required")
	}
	n, unlisted, err := verifySums(*dir)
	if err != nil {
		return err
	}
	fmt.Printf("%d file(s) match SHA256SUMS\n", n)
	for _, u := range unlisted {
		fmt.Printf("not listed: %s\n", u)
	}
	for _, u := range unlisted {
		// Only the launcher's own files and late monitor chunks may be unlisted.
		if !(u == "console.txt" || strings.HasPrefix(u, "launcher/") || strings.HasPrefix(u, dirMonitor+"/")) {
			return fmt.Errorf("verify: unexpected unlisted file %s", u)
		}
	}
	return nil
}

func cmdDryRun(args []string) error {
	fs := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	path := fs.String("plan", "", "plan file")
	seed := fs.Uint64("seed", 0, "batch seed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := plan.Load(*path)
	if err != nil {
		return err
	}
	blocks, err := plan.Expand(p, *seed)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "plan %s (%s), seed %d, sha256 %s: %d blocks\n", p.Name, p.Kind, *seed, p.Hash, len(blocks))
	for i, b := range blocks {
		fmt.Fprintf(os.Stdout, "%4d  %-40s %s\n", i+1, b.Key(), b.Profile)
	}
	return nil
}
