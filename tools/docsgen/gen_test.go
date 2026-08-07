package main

import (
	"os"
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
