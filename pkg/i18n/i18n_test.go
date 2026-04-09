package i18n

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const testLocaleContent = `welcome:
  other: "Welcome, {{.Name}}!"
not_found:
  other: "Resource not found"
`

func setupLocaleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), []byte(testLocaleContent), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewI18nService(t *testing.T) {
	dir := setupLocaleDir(t)
	svc := NewI18nService(dir, "en")
	if svc == nil {
		t.Fatal("expected non-nil I18nService")
	}
	if svc.defaultLang != "en" {
		t.Errorf("defaultLang = %q, want %q", svc.defaultLang, "en")
	}
}

func TestT_Found(t *testing.T) {
	dir := setupLocaleDir(t)
	svc := NewI18nService(dir, "en")

	got := svc.T("en", "not_found", nil)
	if got != "Resource not found" {
		t.Errorf("T() = %q, want %q", got, "Resource not found")
	}
}

func TestT_NotFound(t *testing.T) {
	dir := setupLocaleDir(t)
	svc := NewI18nService(dir, "en")

	got := svc.T("en", "nonexistent_key", nil)
	if got != "nonexistent_key" {
		t.Errorf("T() = %q, want %q (fallback to messageID)", got, "nonexistent_key")
	}
}

func TestT_WithData(t *testing.T) {
	dir := setupLocaleDir(t)
	svc := NewI18nService(dir, "en")

	got := svc.T("en", "welcome", map[string]interface{}{"Name": "Alice"})
	want := "Welcome, Alice!"
	if got != want {
		t.Errorf("T() = %q, want %q", got, want)
	}
}

func TestLangFromRequest(t *testing.T) {
	dir := setupLocaleDir(t)
	svc := NewI18nService(dir, "en")

	tests := []struct {
		name       string
		acceptLang string
		want       string
	}{
		{"simple en", "en", "en"},
		{"with quality", "fr,en;q=0.5", "fr"},
		{"empty defaults", "", "en"},
		{"with quality only", "es;q=0.8", "es"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.acceptLang != "" {
				req.Header.Set("Accept-Language", tt.acceptLang)
			}
			got := svc.LangFromRequest(req)
			if got != tt.want {
				t.Errorf("LangFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLangFromRequest_EmptyLangAfterParsing(t *testing.T) {
	dir := setupLocaleDir(t)
	svc := NewI18nService(dir, "en")

	req, _ := http.NewRequest("GET", "/", nil)
	// ";q=0.8" has no language tag before the semicolon, so splitting produces ""
	req.Header.Set("Accept-Language", ";q=0.8")

	got := svc.LangFromRequest(req)
	if got != "en" {
		t.Errorf("LangFromRequest() = %q, want %q (default fallback)", got, "en")
	}
}

func TestCreateDefaultLocaleFile(t *testing.T) {
	dir := t.TempDir()
	localesDir := filepath.Join(dir, "locales")

	CreateDefaultLocaleFile(localesDir)

	path := filepath.Join(localesDir, "en.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected en.yaml to be created")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty en.yaml")
	}
}

func TestCreateDefaultLocaleFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	localesDir := filepath.Join(dir, "locales")
	os.MkdirAll(localesDir, 0755)

	existing := []byte("custom: content\n")
	path := filepath.Join(localesDir, "en.yaml")
	os.WriteFile(path, existing, 0644)

	CreateDefaultLocaleFile(localesDir)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(existing) {
		t.Error("CreateDefaultLocaleFile overwrote existing file")
	}
}
