package mailer

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gofastadev/gofasta/pkg/config"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

// BrevoSender sends emails via the Brevo (Sendinblue) HTTP API.
type BrevoSender struct {
	cfg      config.BrevoConfig
	from     brevoContact
	renderer *TemplateRenderer
	logger   *slog.Logger
	client   *http.Client
}

type brevoContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

type brevoRequest struct {
	Sender      brevoContact   `json:"sender"`
	To          []brevoContact `json:"to"`
	CC          []brevoContact `json:"cc,omitempty"`
	BCC         []brevoContact `json:"bcc,omitempty"`
	ReplyTo     *brevoContact  `json:"replyTo,omitempty"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
	TextContent string         `json:"textContent,omitempty"`
}

// NewBrevoSender returns a Brevo-backed EmailSender.
func NewBrevoSender(cfg config.BrevoConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger) *BrevoSender {
	return &BrevoSender{
		cfg:      cfg,
		from:     brevoContact{Name: fromName, Email: fromAddress},
		renderer: renderer,
		logger:   loggerOrDefault(logger),
		client:   &http.Client{},
	}
}

// Send delivers msg via the Brevo HTTP API.
func (b *BrevoSender) Send(ctx context.Context, msg EmailMessage) error {
	htmlBody, err := b.resolveBody(msg)
	if err != nil {
		return err
	}

	reqBody := brevoRequest{
		Sender:      b.from,
		Subject:     msg.Subject,
		HTMLContent: htmlBody,
		TextContent: msg.TextBody,
	}

	for _, to := range msg.To {
		reqBody.To = append(reqBody.To, brevoContact{Email: to})
	}
	for _, cc := range msg.CC {
		reqBody.CC = append(reqBody.CC, brevoContact{Email: cc})
	}
	for _, bcc := range msg.BCC {
		reqBody.BCC = append(reqBody.BCC, brevoContact{Email: bcc})
	}
	if msg.ReplyTo != "" {
		reqBody.ReplyTo = &brevoContact{Email: msg.ReplyTo}
	}

	if _, err := postJSON(ctx, b.client, "brevo", brevoAPIURL, map[string]string{
		"api-key": b.cfg.APIKey,
	}, reqBody); err != nil {
		return err
	}

	b.logger.Info("email sent via Brevo", "to", msg.To, "subject", msg.Subject)
	return nil
}

func (b *BrevoSender) resolveBody(msg EmailMessage) (string, error) {
	if msg.Template != "" {
		return b.renderer.Render(msg.Template, msg.TemplateData)
	}
	return msg.HTMLBody, nil
}
