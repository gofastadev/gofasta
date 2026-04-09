package mailer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewTemplateRenderer(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "hello.html", "<p>hi</p>")

	r := NewTemplateRenderer(dir, "TestApp")
	if r == nil {
		t.Fatal("expected non-nil TemplateRenderer")
	}
	if r.appName != "TestApp" {
		t.Errorf("appName = %q, want %q", r.appName, "TestApp")
	}
}

func TestRender_Success(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "welcome.html", "<h1>Hello {{.Name}}</h1>")

	r := NewTemplateRenderer(dir, "MyApp")
	out, err := r.Render("welcome", map[string]any{"Name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Hello Alice") {
		t.Errorf("output = %q, want it to contain %q", out, "Hello Alice")
	}
}

func TestRender_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := NewTemplateRenderer(dir, "App")

	_, err := r.Render("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for non-existent template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestRender_InjectsCommonVars(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "info.html", "App={{.AppName}} Year={{.Year}}")

	r := NewTemplateRenderer(dir, "GoFasta")
	out, err := r.Render("info", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "App=GoFasta") {
		t.Errorf("output = %q, want AppName injected", out)
	}
	year := strconv.Itoa(time.Now().Year())
	if !strings.Contains(out, "Year="+year) {
		t.Errorf("output = %q, want Year injected", out)
	}
}

func TestRender_NilData(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "nildata.html", "App={{.AppName}} Year={{.Year}}")

	r := NewTemplateRenderer(dir, "GoFasta")
	out, err := r.Render("nildata", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "App=GoFasta") {
		t.Errorf("output = %q, want AppName injected even with nil data", out)
	}
	year := strconv.Itoa(time.Now().Year())
	if !strings.Contains(out, "Year="+year) {
		t.Errorf("output = %q, want Year injected even with nil data", out)
	}
}

func TestReload(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "page.html", "Version1")

	r := NewTemplateRenderer(dir, "App")
	out, err := r.Render("page", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Version1") {
		t.Fatalf("output = %q, want 'Version1'", out)
	}

	// Overwrite template on disk
	writeTemplate(t, dir, "page.html", "Version2")
	r.Reload()

	out, err = r.Render("page", nil)
	if err != nil {
		t.Fatalf("unexpected error after reload: %v", err)
	}
	if !strings.Contains(out, "Version2") {
		t.Errorf("output = %q, want 'Version2' after reload", out)
	}
}

func TestFileNameWithoutExt(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"welcome.html", "welcome"},
		{"/path/to/file.txt", "file"},
		{"noext", "noext"},
		{"multi.dots.html", "multi.dots"},
	}
	for _, tc := range tests {
		got := fileNameWithoutExt(tc.input)
		if got != tc.want {
			t.Errorf("fileNameWithoutExt(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// writeTemplate is a helper that writes a template file into the given directory.
func writeTemplate(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write template %s: %v", name, err)
	}
}
