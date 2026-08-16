package mailer

import (
	"log/slog"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
)

func TestNewEmailSender_SMTP(t *testing.T) {
	cfg := &config.EmailConfig{
		Provider:    "smtp",
		FromName:    "Test",
		FromAddress: "test@example.com",
	}
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	logger := slog.Default()

	sender, err := NewEmailSender(cfg, renderer, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sender.(*SMTPSender); !ok {
		t.Errorf("expected *SMTPSender, got %T", sender)
	}
}

func TestNewEmailSender_SendGrid(t *testing.T) {
	cfg := &config.EmailConfig{
		Provider:    "sendgrid",
		FromName:    "Test",
		FromAddress: "test@example.com",
		SendGrid:    config.SendGridConfig{APIKey: "test-key"},
	}
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	logger := slog.Default()

	sender, err := NewEmailSender(cfg, renderer, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sender.(*SendGridSender); !ok {
		t.Errorf("expected *SendGridSender, got %T", sender)
	}
}

func TestNewEmailSender_Brevo(t *testing.T) {
	cfg := &config.EmailConfig{
		Provider:    "brevo",
		FromName:    "Test",
		FromAddress: "test@example.com",
		Brevo:       config.BrevoConfig{APIKey: "test-key"},
	}
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	logger := slog.Default()

	sender, err := NewEmailSender(cfg, renderer, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sender.(*BrevoSender); !ok {
		t.Errorf("expected *BrevoSender, got %T", sender)
	}
}

func TestNewEmailSender_Unknown(t *testing.T) {
	cfg := &config.EmailConfig{
		Provider: "mailchimp",
	}
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	logger := slog.Default()

	_, err := NewEmailSender(cfg, renderer, logger)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewEmailSender_Resend(t *testing.T) {
	cfg := &config.EmailConfig{
		Provider:    "resend",
		FromName:    "Test",
		FromAddress: "test@example.com",
		Resend:      config.ResendConfig{APIKey: "test-key"},
	}
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	logger := slog.Default()

	sender, err := NewEmailSender(cfg, renderer, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sender.(*ResendSender); !ok {
		t.Errorf("expected *ResendSender, got %T", sender)
	}
}
