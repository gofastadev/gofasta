package mailer

import (
	"fmt"
	"log/slog"

	"github.com/gofastadev/gofasta/configs"
)

// NewEmailSender creates the appropriate email sender based on the configured provider.
// This is the factory used by Wire DI — the developer never calls this directly.
func NewEmailSender(cfg *configs.EmailConfig, renderer *TemplateRenderer, logger *slog.Logger) (EmailSender, error) {
	switch cfg.Provider {
	case "smtp":
		return NewSMTPSender(cfg.SMTP, cfg.FromName, cfg.FromAddress, renderer, logger), nil
	case "sendgrid":
		return NewSendGridSender(cfg.SendGrid, cfg.FromName, cfg.FromAddress, renderer, logger), nil
	case "brevo":
		return NewBrevoSender(cfg.Brevo, cfg.FromName, cfg.FromAddress, renderer, logger), nil
	default:
		return nil, fmt.Errorf("unknown email provider: %q (supported: smtp, sendgrid, brevo)", cfg.Provider)
	}
}
