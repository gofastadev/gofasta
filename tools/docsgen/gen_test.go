package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a tiny fixture helper.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPackageDocParagraph_JoinsWrappedLines(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "doc.go"),
		"// Package demo does one thing across a\n// wrapped line. Second sentence here.\n//\n// Second paragraph is ignored.\npackage demo\n")

	got, err := packageDocParagraph(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := "Package demo does one thing across a wrapped line. Second sentence here."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPackageDocParagraph_DirNameDiffersFromPackageName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "doc.go"),
		"// Package other is keyed by directory, not declared name.\npackage other\n")

	got, err := packageDocParagraph(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "Package other is keyed") {
		t.Fatalf("unexpected paragraph: %q", got)
	}
}

func TestPackageDocParagraph_MissingCommentFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "code.go"), "package bare\n")

	_, err := packageDocParagraph(dir)
	if err == nil || !strings.Contains(err.Error(), "no package doc comment") {
		t.Fatalf("expected missing-comment error, got %v", err)
	}
}

func TestReplaceBlock(t *testing.T) {
	src := "before\n<!-- gofasta:begin pkg-table:core -->\nold\n<!-- gofasta:end pkg-table:core -->\nafter\n"
	got, err := replaceBlock(src, "core", "new\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "before\n<!-- gofasta:begin pkg-table:core -->\nnew\n<!-- gofasta:end pkg-table:core -->\nafter\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// Replacing again with the same body is a no-op — sync idempotence.
	again, err := replaceBlock(got, "core", "new\n")
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatal("replaceBlock is not idempotent")
	}
}

func TestReplaceBlock_MissingMarkerFails(t *testing.T) {
	if _, err := replaceBlock("no markers here", "core", "x"); err == nil {
		t.Fatal("expected error for missing begin marker")
	}
	if _, err := replaceBlock("<!-- gofasta:begin pkg-table:core -->", "core", "x"); err == nil {
		t.Fatal("expected error for missing end marker")
	}
}

// TestRunAgainstRepo runs the real check against the actual repository —
// the same gate `make docs-check` runs. It fails when README.md, the doc
// comments, and the category table disagree.
func TestRunAgainstRepo(t *testing.T) {
	if err := Run("../..", true); err != nil {
		t.Fatalf("docsgen -check is red against the working tree: %v", err)
	}
}

// TestAuditCategories_FailureModes builds a fake repo layout to prove the
// audit rejects uncategorized packages and stale category entries.
func TestAuditCategories_FailureModes(t *testing.T) {
	repo := t.TempDir()
	// One real categorized dir per category so the on-disk check passes,
	// is overkill — auditCategories only needs pkg/ to exist. Create every
	// categorized dir, plus one stray.
	for _, cat := range categories {
		for _, d := range cat.Dirs {
			if err := os.MkdirAll(filepath.Join(repo, "pkg", d), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	// A plain file in pkg/ is not a package directory and must be ignored.
	writeFile(t, filepath.Join(repo, "pkg", "notes.txt"), "not a package\n")

	if err := auditCategories(repo); err != nil {
		t.Fatalf("baseline audit should pass: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(repo, "pkg", "newpkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := auditCategories(repo)
	if err == nil || !strings.Contains(err.Error(), "newpkg") {
		t.Fatalf("expected uncategorized-package error, got %v", err)
	}

	if err := os.RemoveAll(filepath.Join(repo, "pkg", "newpkg")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, "pkg", "cache")); err != nil {
		t.Fatal(err)
	}
	err = auditCategories(repo)
	if err == nil || !strings.Contains(err.Error(), "does not exist on disk") {
		t.Fatalf("expected stale-entry error, got %v", err)
	}
}

func TestAuditCategories_DuplicateEntryFails(t *testing.T) {
	orig := categories
	t.Cleanup(func() { categories = orig })
	categories = append(append([]category{}, orig...),
		category{Slug: "dup", Dirs: []string{orig[0].Dirs[0]}})

	err := auditCategories(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "more than one category") {
		t.Fatalf("expected duplicate-entry error, got %v", err)
	}
}

func TestAuditCategories_MissingPkgDirFails(t *testing.T) {
	if err := auditCategories(t.TempDir()); err == nil {
		t.Fatal("expected error when pkg/ does not exist")
	}
}

// buildFakeRepo lays out a repository that satisfies the full Run pipeline:
// every categorized directory exists, every doc source carries a package
// comment, and README.md holds all marker pairs around stale content.
func buildFakeRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for _, cat := range categories {
		for _, d := range cat.Dirs {
			if err := os.MkdirAll(filepath.Join(repo, "pkg", d), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, hasOverride := descriptionOverrides[d]; hasOverride {
				continue
			}
			src := d
			if redirect, hasRedirect := docSourceDir[d]; hasRedirect {
				src = redirect
			}
			name := path.Base(src)
			writeFile(t, filepath.Join(repo, "pkg", filepath.FromSlash(src), "doc.go"),
				fmt.Sprintf("// Package %s is the fake %s package.\npackage %s\n", name, d, name))
		}
	}

	var sb strings.Builder
	sb.WriteString("# Fake repo\n\n")
	for _, cat := range categories {
		fmt.Fprintf(&sb, "%s\nstale\n%s\n\n",
			fmt.Sprintf(markerBeginFmt, cat.Slug), fmt.Sprintf(markerEndFmt, cat.Slug))
	}
	writeFile(t, filepath.Join(repo, "README.md"), sb.String())
	return repo
}

func TestRun_RewritesOutOfSyncReadme(t *testing.T) {
	repo := buildFakeRepo(t)

	if err := Run(repo, false); err != nil {
		t.Fatalf("sync run failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "stale") {
		t.Fatal("stale table content survived the rewrite")
	}
	if !strings.Contains(string(got), "| Package | What it does |") {
		t.Fatal("rewritten README is missing the table header")
	}

	// The freshly written README must now pass check mode.
	if err := Run(repo, true); err != nil {
		t.Fatalf("check after sync should pass: %v", err)
	}
}

func TestRun_CheckFailsWhenOutOfSync(t *testing.T) {
	repo := buildFakeRepo(t)

	err := Run(repo, true)
	if err == nil || !strings.Contains(err.Error(), "out of sync") {
		t.Fatalf("expected out-of-sync error, got %v", err)
	}
}

func TestRun_AuditFailure(t *testing.T) {
	if err := Run(t.TempDir(), true); err == nil {
		t.Fatal("expected error when pkg/ is missing")
	}
}

func TestRun_MissingReadmeFails(t *testing.T) {
	repo := buildFakeRepo(t)
	if err := os.Remove(filepath.Join(repo, "README.md")); err != nil {
		t.Fatal(err)
	}
	if err := Run(repo, true); err == nil {
		t.Fatal("expected error when README.md is missing")
	}
}

func TestRun_RenderTableFailure(t *testing.T) {
	repo := buildFakeRepo(t)
	// Strip a doc source so packageDocParagraph fails inside renderTable.
	if err := os.Remove(filepath.Join(repo, "pkg", "config", "doc.go")); err != nil {
		t.Fatal(err)
	}
	err := Run(repo, true)
	if err == nil || !strings.Contains(err.Error(), "pkg/config") {
		t.Fatalf("expected pkg/config render error, got %v", err)
	}
}

func TestRun_MissingMarkerFails(t *testing.T) {
	repo := buildFakeRepo(t)
	writeFile(t, filepath.Join(repo, "README.md"), "# No markers here\n")
	err := Run(repo, true)
	if err == nil || !strings.Contains(err.Error(), "missing marker") {
		t.Fatalf("expected missing-marker error, got %v", err)
	}
}

func TestPackageDocParagraph_MissingDirFails(t *testing.T) {
	if _, err := packageDocParagraph(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestPackageDocParagraph_ParseErrorFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "broken.go"), "pkg broken // not a package clause\n")
	if _, err := packageDocParagraph(dir); err == nil {
		t.Fatal("expected parse error for malformed Go file")
	}
}

func TestPackageDocParagraph_NoGoFilesFails(t *testing.T) {
	dir := t.TempDir()
	// Subdirectories, test files, and non-Go files are all skipped.
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "only_test.go"), "package only\n")
	writeFile(t, filepath.Join(dir, "readme.txt"), "no code\n")

	_, err := packageDocParagraph(dir)
	if err == nil || !strings.Contains(err.Error(), "no Go package found") {
		t.Fatalf("expected no-package error, got %v", err)
	}
}

func TestReplaceBlock_EndBeforeBeginFails(t *testing.T) {
	src := fmt.Sprintf(markerEndFmt, "core") + "\n" + fmt.Sprintf(markerBeginFmt, "core") + "\n"
	_, err := replaceBlock(src, "core", "x")
	if err == nil || !strings.Contains(err.Error(), "before its begin marker") {
		t.Fatalf("expected marker-order error, got %v", err)
	}
}
