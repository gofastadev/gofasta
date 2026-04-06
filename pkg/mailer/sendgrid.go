package mailer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/healtronlabs/gofasta/configs"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridSender sends emails via the SendGrid HTTP API.
type SendGridSender struct {
	cfg      configs.SendGridConfig
	from     *mail.Email
	renderer *TemplateRenderer
	logger   *slog.Logger
}

func NewSendGridSender(cfg configs.SendGridConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger) *SendGridSender {
	return &SendGridSender{
		cfg:      cfg,
		from:     mail.NewEmail(fromName, fromAddress),
		renderer: renderer,
		logger:   logger,
	}
}

func (s *SendGridSender) Send(ctx context.Context, msg EmailMessage) error {
	htmlBody, err := s.resolveBody(msg)
	if err != nil {
		return err
	}

	m := mail.NewV3Mail()
	m.SetFrom(s.from)
	m.Subject = msg.Subject

	p := mail.NewPersonalization()
	for _, to := range msg.To {
		p.AddTos(mail.NewEmail("", to))
	}
	for _, cc := range msg.CC {
		p.AddCCs(mail.NewEmail("", cc))
	}
	for _, bcc := range msg.BCC {
		p.AddBCCs(mail.NewEmail("", bcc))
	}
	m.AddPersonalizations(p)

	m.AddContent(mail.NewContent("text/html", htmlBody))

	if msg.ReplyTo != "" {
		m.SetReplyTo(mail.NewEmail("", msg.ReplyTo))
	}

	for _, att := range msg.Attachments {
		a := mail.NewAttachment()
		a.SetFilename(att.Filename)
		a.SetType(att.ContentType)
		a.SetContent(string(att.Content))
		m.AddAttachment(a)
	}

	client := sendgrid.NewSendClient(s.cfg.APIKey)
	resp, err := client.SendWithContext(ctx, m)
	if err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: status %d, body: %s", resp.StatusCode, resp.Body)
	}

	s.logger.Info("email sent via SendGrid", "to", msg.To, "subject", msg.Subject, "status", resp.StatusCode)
	return nil
}

func (s *SendGridSender) resolveBody(msg EmailMessage) (string, error) {
	if msg.Template != "" {
		return s.renderer.Render(msg.Template, msg.TemplateData)
	}
	return msg.HTMLBody, nil
}
