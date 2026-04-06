package i18n

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// I18nService handles internationalization / message translation.
type I18nService struct {
	bundle       *i18n.Bundle
	defaultLang  string
}

// NewI18nService loads locale files from the given directory.
// Files should be named like: en.yaml, fr.yaml, es.yaml
func NewI18nService(localesDir string, defaultLang string) *I18nService {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	// Load all locale files
	files, _ := filepath.Glob(filepath.Join(localesDir, "*.yaml"))
	for _, f := range files {
		bundle.LoadMessageFile(f)
	}

	return &I18nService{bundle: bundle, defaultLang: defaultLang}
}

// T translates a message ID using the given language tag.
func (s *I18nService) T(lang, messageID string, data map[string]interface{}) string {
	localizer := i18n.NewLocalizer(s.bundle, lang, s.defaultLang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// LangFromRequest extracts the preferred language from the Accept-Language header.
func (s *I18nService) LangFromRequest(r *http.Request) string {
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return s.defaultLang
	}
	// Simple parsing: take the first language tag
	parts := strings.SplitN(accept, ",", 2)
	lang := strings.TrimSpace(strings.SplitN(parts[0], ";", 2)[0])
	if lang == "" {
		return s.defaultLang
	}
	return lang
}

// CreateDefaultLocaleFile creates a default English locale file if none exists.
func CreateDefaultLocaleFile(localesDir string) {
	path := filepath.Join(localesDir, "en.yaml")
	if _, err := os.Stat(path); err == nil {
		return
	}
	os.MkdirAll(localesDir, 0755)
	content := `# English translations
welcome:
  other: "Welcome, {{.Name}}!"
not_found:
  other: "Resource not found"
unauthorized:
  other: "Authentication required"
forbidden:
  other: "You don't have permission to access this resource"
validation_failed:
  other: "Validation failed"
internal_error:
  other: "An internal error occurred"
`
	os.WriteFile(path, []byte(content), 0644)
}
