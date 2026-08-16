package mailer

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/gofastadev/gofasta/pkg/config"
)

// SMTPSender sends emails via SMTP with optional STARTTLS.
type SMTPSender struct {
	cfg      config.SMTPConfig
	from     string
	fromName string
	renderer *TemplateRenderer
	logger   *slog.Logger
}

// NewSMTPSender returns an SMTP-backed EmailSender.
func NewSMTPSender(cfg config.SMTPConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger) *SMTPSender {
	return &SMTPSender{
		cfg:      cfg,
		from:     fromAddress,
		fromName: fromName,
		renderer: renderer,
		logger:   loggerOrDefault(logger),
	}
}

// Send delivers msg over SMTP, optionally upgrading to TLS via STARTTLS.
//
//nolint:gocyclo // linear branch coverage for SMTP connection + MIME assembly.
func (s *SMTPSender) Send(ctx context.Context, msg EmailMessage) error {
	htmlBody, err := s.resolveBody(msg)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))

	var conn net.Conn
	if s.cfg.UseTLS {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{
			ServerName:         s.cfg.Host,
			InsecureSkipVerify: s.cfg.InsecureSkipVerify,
		})
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		conn = tlsConn
	} else {
		plainConn, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
		conn = plainConn
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS if not already using TLS and server supports it
	if !s.cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	// Auth
	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	fromHeader := fmt.Sprintf("%s <%s>", s.fromName, s.from)
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	allRecipients := append(append(msg.To, msg.CC...), msg.BCC...)
	for _, rcpt := range allRecipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	// Build MIME message
	reqID := ctx.Value("requestID")
	if reqID == nil {
		reqID = "000"
	}
	boundary := fmt.Sprintf("gofasta-boundary-%v", reqID)

	var sb strings.Builder
	sb.WriteString("From: " + fromHeader + "\r\n")
	sb.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	if len(msg.CC) > 0 {
		sb.WriteString("CC: " + strings.Join(msg.CC, ", ") + "\r\n")
	}
	// Reply-To is what makes a no-reply sending address usable: mail goes out
	// from the platform's address but a recipient's reply reaches a human. The
	// API providers all honor EmailMessage.ReplyTo, so SMTP dropping it made
	// the field mean different things per provider.
	if msg.ReplyTo != "" {
		sb.WriteString("Reply-To: " + msg.ReplyTo + "\r\n")
	}
	sb.WriteString("Subject: " + msg.Subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")

	writeBody(&sb, msg, htmlBody, boundary)

	if _, err := wc.Write([]byte(sb.String())); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}

	s.logger.Info("email sent via SMTP", "to", msg.To, "subject", msg.Subject)
	return client.Quit()
}

func (s *SMTPSender) resolveBody(msg EmailMessage) (string, error) {
	if msg.Template != "" {
		return s.renderer.Render(msg.Template, msg.TemplateData)
	}
	return msg.HTMLBody, nil
}

// writeAlternative writes the plain-text and HTML parts of a
// multipart/alternative body, in RFC 2046 order: least-preferred first, so a
// client that understands both shows the HTML.
func writeAlternative(mw *multipart.Writer, htmlBody, textBody string) {
	textPart, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=UTF-8"},
	})
	_, _ = textPart.Write([]byte(textBody))

	htmlPart, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/html; charset=UTF-8"},
	})
	_, _ = htmlPart.Write([]byte(htmlBody))
}

// writeBody appends the message body to sb in the simplest MIME shape that
// carries everything msg holds:
//
//	html only                 → text/html
//	html + text               → multipart/alternative
//	html + attachments        → multipart/mixed
//	html + text + attachments → multipart/mixed wrapping an alternative
//
// Kept apart from Send so the SMTP conversation and the document assembly can
// each be read on their own.
func writeBody(sb *strings.Builder, msg EmailMessage, htmlBody, boundary string) {
	switch {
	case len(msg.Attachments) == 0 && msg.TextBody == "":
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(htmlBody)
	case len(msg.Attachments) == 0:
		// Both bodies, nothing to attach: multipart/alternative lets the
		// client pick. Plain text goes first — the parts are ordered
		// least-preferred to most-preferred per RFC 2046 §5.1.4.
		sb.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
		sb.WriteString("\r\n")

		mw := multipart.NewWriter(sb)
		_ = mw.SetBoundary(boundary)
		writeAlternative(mw, htmlBody, msg.TextBody)
		_ = mw.Close()
	default:
		sb.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		sb.WriteString("\r\n")

		mw := multipart.NewWriter(sb)
		_ = mw.SetBoundary(boundary)

		if msg.TextBody != "" {
			// The bodies nest one level down so the alternative applies to
			// them alone, leaving the attachments as siblings of the pair.
			altBoundary := boundary + "-alt"
			altPart, _ := mw.CreatePart(textproto.MIMEHeader{
				"Content-Type": {"multipart/alternative; boundary=" + altBoundary},
			})
			amw := multipart.NewWriter(altPart)
			_ = amw.SetBoundary(altBoundary)
			writeAlternative(amw, htmlBody, msg.TextBody)
			_ = amw.Close()
		} else {
			htmlPart, _ := mw.CreatePart(textproto.MIMEHeader{
				"Content-Type": {"text/html; charset=UTF-8"},
			})
			_, _ = htmlPart.Write([]byte(htmlBody))
		}

		for _, att := range msg.Attachments {
			ct := att.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			attPart, _ := mw.CreatePart(textproto.MIMEHeader{
				"Content-Type":              {ct},
				"Content-Disposition":       {fmt.Sprintf("attachment; filename=%q", mime.QEncoding.Encode("UTF-8", att.Filename))},
				"Content-Transfer-Encoding": {"base64"},
			})
			_, _ = attPart.Write([]byte(base64.StdEncoding.EncodeToString(att.Content)))
		}
		_ = mw.Close()
	}
}
