package mailer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gofastadev/gofasta/pkg/config"
)

const resendAPIURL = "https://api.resend.com/emails"

// ResendSender sends emails via the Resend HTTP API.
type ResendSender struct {
	cfg      config.ResendConfig
	from     string
	renderer *TemplateRenderer
	logger   *slog.Logger
	client   *http.Client
}

type resendAttachment struct {
	Content     string `json:"content"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}

type resendRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	CC          []string           `json:"cc,omitempty"`
	BCC         []string           `json:"bcc,omitempty"`
	ReplyTo     []string           `json:"reply_to,omitempty"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Text        string             `json:"text,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

// NewResendSender returns a Resend-backed EmailSender.
func NewResendSender(cfg config.ResendConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger) *ResendSender {
	from := fromAddress
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddress)
	}
	return &ResendSender{
		cfg:      cfg,
		from:     from,
		renderer: renderer,
		logger:   loggerOrDefault(logger),
		client:   &http.Client{},
	}
}

// Send delivers msg via the Resend HTTP API.
func (r *ResendSender) Send(ctx context.Context, msg EmailMessage) error {
	htmlBody, err := r.resolveBody(msg)
	if err != nil {
		return err
	}

	reqBody := resendRequest{
		From:    r.from,
		To:      msg.To,
		CC:      msg.CC,
		BCC:     msg.BCC,
		Subject: msg.Subject,
		HTML:    htmlBody,
		Text:    msg.TextBody,
	}
	if msg.ReplyTo != "" {
		reqBody.ReplyTo = []string{msg.ReplyTo}
	}
	for _, att := range msg.Attachments {
		reqBody.Attachments = append(reqBody.Attachments, resendAttachment{
			// Attachment.Content is raw bytes package-wide; Resend's API
			// expects base64.
			Content:     base64.StdEncoding.EncodeToString(att.Content),
			Filename:    att.Filename,
			ContentType: att.ContentType,
		})
	}

	body, err := postJSON(ctx, r.client, "resend", resendAPIURL, map[string]string{
		"Authorization": "Bearer " + r.cfg.APIKey,
	}, reqBody)
	if err != nil {
		return err
	}

	var res struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &res)

	r.logger.Info("email sent via Resend", "to", msg.To, "subject", msg.Subject, "id", res.ID)
	return nil
}

func (r *ResendSender) resolveBody(msg EmailMessage) (string, error) {
	if msg.Template != "" {
		return r.renderer.Render(msg.Template, msg.TemplateData)
	}
	return msg.HTMLBody, nil
}
