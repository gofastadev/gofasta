package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gofastadev/gofasta/configs"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

// BrevoSender sends emails via the Brevo (Sendinblue) HTTP API.
type BrevoSender struct {
	cfg      configs.BrevoConfig
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
}

func NewBrevoSender(cfg configs.BrevoConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger) *BrevoSender {
	return &BrevoSender{
		cfg:      cfg,
		from:     brevoContact{Name: fromName, Email: fromAddress},
		renderer: renderer,
		logger:   logger,
		client:   &http.Client{},
	}
}

func (b *BrevoSender) Send(ctx context.Context, msg EmailMessage) error {
	htmlBody, err := b.resolveBody(msg)
	if err != nil {
		return err
	}

	reqBody := brevoRequest{
		Sender:      b.from,
		Subject:     msg.Subject,
		HTMLContent: htmlBody,
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

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("brevo marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", brevoAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("brevo request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", b.cfg.APIKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo error: status %d, body: %s", resp.StatusCode, string(body))
	}

	b.logger.Info("email sent via Brevo", "to", msg.To, "subject", msg.Subject, "status", resp.StatusCode)
	return nil
}

func (b *BrevoSender) resolveBody(msg EmailMessage) (string, error) {
	if msg.Template != "" {
		return b.renderer.Render(msg.Template, msg.TemplateData)
	}
	return msg.HTMLBody, nil
}
