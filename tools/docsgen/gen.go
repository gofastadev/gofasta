package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// category is one README sub-table under "What's in the Box". Slug feeds the
// marker id (pkg-table:<slug>); dirs lists pkg/ directory names in the exact
// row order the table renders.
type category struct {
	Slug string
	Dirs []string
}

// categories mirrors the README's sub-table order. Every pkg/ directory that
// contains Go files MUST appear in exactly one category — Run fails loudly on
// any uncategorized or stale entry, which is the whole point: adding a new
// package without deciding where it is documented breaks the build.
var categories = []category{
	{Slug: "core", Dirs: []string{"config", "logger", "errors", "models", "types", "database"}},
	{Slug: "http", Dirs: []string{"httputil", "middleware", "health", "graphql"}},
	{Slug: "security", Dirs: []string{"auth", "encryption", "session"}},
	{Slug: "data", Dirs: []string{"cache", "storage", "seeds"}},
	{Slug: "communication", Dirs: []string{"mailer", "slack", "whatsapp", "push", "notify", "websocket"}},
	{Slug: "background", Dirs: []string{"scheduler", "queue", "resilience"}},
	{Slug: "validation", Dirs: []string{"validators", "i18n", "utils"}},
	{Slug: "observability", Dirs: []string{"observability", "featureflag"}},
	{Slug: "testing", Dirs: []string{"testutil"}},
}

// overrides supplies table rows that cannot come from a package comment, and
// custom row labels where the directory name is not the import path users see.
var descriptionOverrides = map[string]string{
	// pkg/graphql ships .gql schema fragments only — there is no Go package
	// to carry a doc comment.
	"graphql": "Shared GraphQL schema fragments (`.gql` files) for pagination, sorting, errors, and a default health Query/Mutation. Consumed by gqlgen at build time, not imported as Go code — keeps REST and GraphQL response shapes aligned.",
}

// docSourceDir redirects doc extraction when the row documents a subpackage.
var docSourceDir = map[string]string{
	"testutil": "testutil/testdb",
}

// rowLabels overrides the `pkg/<dir>` label rendered in the first column.
var rowLabels = map[string]string{
	"testutil": "pkg/testutil/testdb",
}

const (
	markerBeginFmt = "<!-- gofasta:begin pkg-table:%s -->"
	markerEndFmt   = "<!-- gofasta:end pkg-table:%s -->"
)

// Run regenerates (or, with check=true, verifies) the README package tables.
func Run(repoRoot string, check bool) error {
	if err := auditCategories(repoRoot); err != nil {
		return err
	}

	readmePath := filepath.Join(repoRoot, "README.md")
	src, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}

	updated := string(src)
	for _, cat := range categories {
		table, err := renderTable(repoRoot, cat)
		if err != nil {
			return err
		}
		updated, err = replaceBlock(updated, cat.Slug, table)
		if err != nil {
			return err
		}
	}

	if updated == string(src) {
		return nil
	}
	if check {
		return fmt.Errorf("README.md package tables are out of sync with pkg/* doc comments — run `make docs-sync`")
	}
	return os.WriteFile(readmePath, []byte(updated), 0o644)
}

// auditCategories enforces the 1:1 mapping between pkg/ directories with Go
// files and category entries.
func auditCategories(repoRoot string) error {
	categorized := map[string]bool{}
	for _, cat := range categories {
		for _, d := range cat.Dirs {
			if categorized[d] {
				return fmt.Errorf("pkg/%s appears in more than one category", d)
			}
			categorized[d] = true
		}
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "pkg"))
	if err != nil {
		return err
	}
	var missing []string
	onDisk := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		onDisk[e.Name()] = true
		if !categorized[e.Name()] {
			missing = append(missing, e.Name())
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("pkg/{%s} not assigned to any docsgen category — add it to the categories table in tools/docsgen/gen.go", strings.Join(missing, ", "))
	}
	for d := range categorized {
		if !onDisk[d] {
			return fmt.Errorf("docsgen category entry pkg/%s does not exist on disk — remove it from tools/docsgen/gen.go", d)
		}
	}
	return nil
}

// renderTable builds one category's markdown table.
func renderTable(repoRoot string, cat category) (string, error) {
	var b strings.Builder
	b.WriteString("| Package | What it does |\n")
	b.WriteString("|---------|-------------|\n")
	for _, dir := range cat.Dirs {
		label, ok := rowLabels[dir]
		if !ok {
			label = "pkg/" + dir
		}
		desc, ok := descriptionOverrides[dir]
		if !ok {
			src := dir
			if redirect, hasRedirect := docSourceDir[dir]; hasRedirect {
				src = redirect
			}
			var err error
			desc, err = packageDocParagraph(filepath.Join(repoRoot, "pkg", src))
			if err != nil {
				return "", fmt.Errorf("pkg/%s: %w", dir, err)
			}
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", label, strings.ReplaceAll(desc, "|", `\|`))
	}
	return b.String(), nil
}

// packageDocParagraph extracts the first paragraph of a package's doc
// comment, re-joining wrapped lines into a single line of table-cell text.
// Only package clauses are parsed — the declared name may differ from the
// directory (pkg/errors declares apperrors), and whichever non-test file
// carries the doc comment wins.
func packageDocParagraph(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	pkgName := ""
	docText := ""
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			return "", err
		}
		pkgName = f.Name.Name
		if f.Doc != nil {
			docText = f.Doc.Text()
			break
		}
	}
	if pkgName == "" {
		return "", fmt.Errorf("no Go package found in %s", dir)
	}
	if strings.TrimSpace(docText) == "" {
		return "", fmt.Errorf("package %s has no package doc comment — add a doc.go (revive's package-comments rule enforces this)", pkgName)
	}
	paragraph, _, _ := strings.Cut(strings.TrimSpace(docText), "\n\n")
	return strings.Join(strings.Fields(paragraph), " "), nil
}

// replaceBlock swaps the content between a table's begin/end markers.
func replaceBlock(src, slug, body string) (string, error) {
	begin := fmt.Sprintf(markerBeginFmt, slug)
	end := fmt.Sprintf(markerEndFmt, slug)

	beginIdx := strings.Index(src, begin)
	if beginIdx == -1 {
		return "", fmt.Errorf("README.md is missing marker %q", begin)
	}
	endIdx := strings.Index(src, end)
	if endIdx == -1 {
		return "", fmt.Errorf("README.md is missing marker %q", end)
	}
	if endIdx < beginIdx {
		return "", fmt.Errorf("marker %q appears before its begin marker", end)
	}
	contentStart := beginIdx + len(begin)
	return src[:contentStart] + "\n" + body + src[endIdx:], nil
}
