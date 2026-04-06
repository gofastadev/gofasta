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

	"github.com/gofastadev/gofasta/configs"
)

// SMTPSender sends emails via SMTP with optional STARTTLS.
type SMTPSender struct {
	cfg      configs.SMTPConfig
	from     string
	fromName string
	renderer *TemplateRenderer
	logger   *slog.Logger
}

func NewSMTPSender(cfg configs.SMTPConfig, fromName, fromAddress string, renderer *TemplateRenderer, logger *slog.Logger) *SMTPSender {
	return &SMTPSender{
		cfg:      cfg,
		from:     fromAddress,
		fromName: fromName,
		renderer: renderer,
		logger:   logger,
	}
}

func (s *SMTPSender) Send(ctx context.Context, msg EmailMessage) error {
	htmlBody, err := s.resolveBody(msg)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))

	var conn net.Conn
	if s.cfg.UseTLS {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host})
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
	defer client.Close()

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
	boundary := "gofasta-boundary-" + fmt.Sprintf("%d", ctx.Value("requestID"))
	if boundary == "gofasta-boundary-<nil>" {
		boundary = "gofasta-boundary-000"
	}

	var sb strings.Builder
	sb.WriteString("From: " + fromHeader + "\r\n")
	sb.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	if len(msg.CC) > 0 {
		sb.WriteString("CC: " + strings.Join(msg.CC, ", ") + "\r\n")
	}
	sb.WriteString("Subject: " + msg.Subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")

	if len(msg.Attachments) == 0 {
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(htmlBody)
	} else {
		sb.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		sb.WriteString("\r\n")

		mw := multipart.NewWriter(&sb)
		mw.SetBoundary(boundary)

		htmlPart, _ := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"text/html; charset=UTF-8"},
		})
		htmlPart.Write([]byte(htmlBody))

		for _, att := range msg.Attachments {
			ct := att.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			attPart, _ := mw.CreatePart(textproto.MIMEHeader{
				"Content-Type":              {ct},
				"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, mime.QEncoding.Encode("UTF-8", att.Filename))},
				"Content-Transfer-Encoding": {"base64"},
			})
			attPart.Write([]byte(base64.StdEncoding.EncodeToString(att.Content)))
		}
		mw.Close()
	}

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
