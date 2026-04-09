package mailer

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
)

func TestNewSMTPSender(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: "smtp.example.com", Port: 587},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	if s == nil {
		t.Fatal("expected non-nil SMTPSender")
	}
	if s.from != "sender@example.com" {
		t.Errorf("from = %q, want %q", s.from, "sender@example.com")
	}
	if s.fromName != "Sender" {
		t.Errorf("fromName = %q, want %q", s.fromName, "Sender")
	}
	if s.cfg.Host != "smtp.example.com" {
		t.Errorf("cfg.Host = %q, want %q", s.cfg.Host, "smtp.example.com")
	}
	if s.cfg.Port != 587 {
		t.Errorf("cfg.Port = %d, want %d", s.cfg.Port, 587)
	}
}

func TestSMTPSender_ResolveBody_HTMLBody(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{HTMLBody: "<p>Direct HTML</p>"}
	body, err := s.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "<p>Direct HTML</p>" {
		t.Errorf("body = %q, want %q", body, "<p>Direct HTML</p>")
	}
}

func TestSMTPSender_ResolveBody_Template(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "reset.html", "<p>Reset link: {{.Link}}</p>")
	renderer := NewTemplateRenderer(dir, "App")

	s := NewSMTPSender(
		config.SMTPConfig{},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		Template:     "reset",
		TemplateData: map[string]any{"Link": "https://example.com/reset"},
	}
	body, err := s.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "https://example.com/reset") {
		t.Errorf("body = %q, want it to contain the reset link", body)
	}
}
