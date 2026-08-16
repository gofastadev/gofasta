package mailer

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// sendgridClient abstracts the SendGrid API client for testability.
type sendgridClient interface {
	SendWithContext(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error)
}

// SendGridSender sends emails via the SendGrid HTTP API.
type SendGridSender struct {
	client   sendgridClient
	from     *mail.Email
	renderer *TemplateRenderer
	logger   *slog.Logger
}

// NewSendGridSender returns a SendGrid-backed EmailSender.
func NewSendGridSender(cfg config.SendGridConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger, opts ...SenderOption) *SendGridSender {
	return &SendGridSender{
		client: &sendgridRESTClient{
			request: sendgrid.GetRequest(cfg.APIKey, sendGridMailPath, ""),
			rest:    &rest.Client{HTTPClient: resolveSenderOptions(opts).httpClient},
		},
		from:     mail.NewEmail(fromName, fromAddress),
		renderer: renderer,
		logger:   loggerOrDefault(logger),
	}
}

// sendGridMailPath is the v3 send endpoint, the same one sendgrid.NewSendClient
// targets.
const sendGridMailPath = "/v3/mail/send"

// sendgridRESTClient sends through a rest.Client the caller owns.
//
// sendgrid.NewSendClient routes every request through rest.DefaultClient, a
// package-level global. Honoring WithHTTPClient by reassigning that global
// would change the transport for every SendGrid client in the process — and
// race with any other goroutine constructing one. Holding our own rest.Client
// keeps the choice local to this sender.
type sendgridRESTClient struct {
	request rest.Request
	rest    *rest.Client
}

// SendWithContext implements sendgridClient.
func (c *sendgridRESTClient) SendWithContext(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
	req := c.request
	req.Body = mail.GetRequestBody(email)
	return c.rest.SendWithContext(ctx, req)
}

// Send delivers msg via the SendGrid HTTP API.
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

	// Plain text first: SendGrid requires content parts in increasing order
	// of preference, and rejects a request that lists text/html before
	// text/plain.
	if msg.TextBody != "" {
		m.AddContent(mail.NewContent("text/plain", msg.TextBody))
	}
	m.AddContent(mail.NewContent("text/html", htmlBody))

	if msg.ReplyTo != "" {
		m.SetReplyTo(mail.NewEmail("", msg.ReplyTo))
	}

	for _, att := range msg.Attachments {
		a := mail.NewAttachment()
		a.SetFilename(att.Filename)
		a.SetType(att.ContentType)
		// SendGrid requires attachment content base64-encoded. Passing the raw
		// bytes produced a request the API accepted and a file the recipient
		// could not open — the send reported success either way, so the only
		// symptom was a corrupt attachment.
		a.SetContent(base64.StdEncoding.EncodeToString(att.Content))
		m.AddAttachment(a)
	}

	resp, err := s.client.SendWithContext(ctx, m)
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
