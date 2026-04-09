package mailer

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
)

func TestNewSendGridSender(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "sg-key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	if s == nil {
		t.Fatal("expected non-nil SendGridSender")
	}
	if s.from.Name != "Sender" {
		t.Errorf("from.Name = %q, want %q", s.from.Name, "Sender")
	}
	if s.from.Address != "sender@example.com" {
		t.Errorf("from.Address = %q, want %q", s.from.Address, "sender@example.com")
	}
}

func TestSendGridSender_ResolveBody_HTMLBody(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "key"},
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

func TestSendGridSender_ResolveBody_Template(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "notify.html", "<p>Notification for {{.User}}</p>")
	renderer := NewTemplateRenderer(dir, "App")

	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		Template:     "notify",
		TemplateData: map[string]any{"User": "Charlie"},
	}
	body, err := s.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "Notification for Charlie") {
		t.Errorf("body = %q, want it to contain 'Notification for Charlie'", body)
	}
}
