package main

import (
	"flag"
	"os"
	"testing"
)

// callMain invokes main() with a fresh flag set and the given argv, so the
// flags declared inside main do not collide across invocations.
func callMain(t *testing.T, args ...string) {
	t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = oldArgs, oldFlags })
	flag.CommandLine = flag.NewFlagSet("docsgen", flag.ExitOnError)
	os.Args = append([]string{"docsgen"}, args...)
	main()
}

func TestMain_CheckAgainstRepoSucceeds(t *testing.T) {
	exitCode := -1
	osExit = func(code int) { exitCode = code }
	t.Cleanup(func() { osExit = os.Exit })

	callMain(t, "-check", "-repo", "../..")
	if exitCode != -1 {
		t.Fatalf("expected no exit, got os.Exit(%d)", exitCode)
	}
}

func TestMain_FailureExitsOne(t *testing.T) {
	exitCode := -1
	osExit = func(code int) { exitCode = code }
	t.Cleanup(func() { osExit = os.Exit })

	// An empty repo root has no pkg/ directory, so Run fails.
	callMain(t, "-repo", t.TempDir())
	if exitCode != 1 {
		t.Fatalf("expected os.Exit(1), got %d", exitCode)
	}
}
