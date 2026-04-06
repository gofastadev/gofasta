package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TemplateRenderer loads and renders HTML email templates.
// Templates are parsed once and cached for performance.
type TemplateRenderer struct {
	dir       string
	templates map[string]*template.Template
	mu        sync.RWMutex
	appName   string
}

// NewTemplateRenderer creates a renderer that loads templates from the given directory.
// Templates in layouts/ are used as base layouts.
func NewTemplateRenderer(templatesDir string, appName string) *TemplateRenderer {
	r := &TemplateRenderer{
		dir:       templatesDir,
		templates: make(map[string]*template.Template),
		appName:   appName,
	}
	r.loadAll()
	return r
}

// Render executes a named template with the given data and returns the HTML string.
func (r *TemplateRenderer) Render(name string, data map[string]any) (string, error) {
	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("email template %q not found", name)
	}

	// Inject common variables
	if data == nil {
		data = make(map[string]any)
	}
	if _, exists := data["AppName"]; !exists {
		data["AppName"] = r.appName
	}
	if _, exists := data["Year"]; !exists {
		data["Year"] = time.Now().Year()
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render email template %q: %w", name, err)
	}
	return buf.String(), nil
}

// Reload re-reads all templates from disk. Useful for development.
func (r *TemplateRenderer) Reload() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates = make(map[string]*template.Template)
	r.loadAll()
}

func (r *TemplateRenderer) loadAll() {
	layoutPattern := filepath.Join(r.dir, "layouts", "*.html")
	layouts, _ := filepath.Glob(layoutPattern)

	emailPattern := filepath.Join(r.dir, "*.html")
	emails, _ := filepath.Glob(emailPattern)

	for _, emailFile := range emails {
		name := fileNameWithoutExt(emailFile)
		files := append(layouts, emailFile)
		tmpl, err := template.New(filepath.Base(emailFile)).ParseFiles(files...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse email template %s: %v\n", name, err)
			continue
		}
		r.templates[name] = tmpl
	}
}

func fileNameWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}
