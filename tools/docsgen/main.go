// Command docsgen regenerates the package tables in README.md from the
// pkg/* package comments, so the documented package list can never drift
// from the code. Run with no flags to rewrite README.md in place; run with
// -check to verify the README is current (exit 1 when it is not) — the mode
// `make docs-check` and CI use.
package main

import (
	"flag"
	"fmt"
	"os"
)

// osExit is a seam so tests can observe the failure exit without killing
// the test process.
var osExit = os.Exit

func main() {
	check := flag.Bool("check", false, "verify README.md is in sync instead of rewriting it")
	repo := flag.String("repo", ".", "repository root containing README.md and pkg/")
	flag.Parse()

	if err := Run(*repo, *check); err != nil {
		fmt.Fprintln(os.Stderr, "docsgen:", err)
		osExit(1)
	}
}
