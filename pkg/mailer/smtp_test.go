package mailer

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
)

func TestNewSMTPSender(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: "smtp.example.com", Port: 587},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	if s == nil {
		t.Fatal("expected non-nil SMTPSender")
	}
	if s.from != "sender@example.com" {
		t.Errorf("from = %q, want %q", s.from, "sender@example.com")
	}
	if s.fromName != "Sender" {
		t.Errorf("fromName = %q, want %q", s.fromName, "Sender")
	}
	if s.cfg.Host != "smtp.example.com" {
		t.Errorf("cfg.Host = %q, want %q", s.cfg.Host, "smtp.example.com")
	}
	if s.cfg.Port != 587 {
		t.Errorf("cfg.Port = %d, want %d", s.cfg.Port, 587)
	}
}

func TestSMTPSender_ResolveBody_HTMLBody(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{},
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

func TestSMTPSender_ResolveBody_Template(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "reset.html", "<p>Reset link: {{.Link}}</p>")
	renderer := NewTemplateRenderer(dir, "App")

	s := NewSMTPSender(
		config.SMTPConfig{},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		Template:     "reset",
		TemplateData: map[string]any{"Link": "https://example.com/reset"},
	}
	body, err := s.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "https://example.com/reset") {
		t.Errorf("body = %q, want it to contain the reset link", body)
	}
}

// mockSMTPServer starts a minimal SMTP server on a random port for testing.
func mockSMTPServer(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	addr := ln.Addr().(*net.TCPAddr)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSMTPConn(conn)
		}
	}()

	return addr.IP.String(), addr.Port
}

func handleSMTPConn(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Greeting
	fmt.Fprintf(writer, "220 localhost SMTP mock\r\n")
	writer.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))

		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			fmt.Fprintf(writer, "250-localhost\r\n250 OK\r\n")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case strings.HasPrefix(cmd, "RCPT TO"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case cmd == "DATA":
			fmt.Fprintf(writer, "354 Start mail input\r\n")
			writer.Flush()
			// Read until lone "."
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(dataLine) == "." {
					break
				}
			}
			fmt.Fprintf(writer, "250 OK\r\n")
		case cmd == "QUIT":
			fmt.Fprintf(writer, "221 Bye\r\n")
			writer.Flush()
			return
		default:
			fmt.Fprintf(writer, "250 OK\r\n")
		}
		writer.Flush()
	}
}

func TestSMTPSender_Send_Success(t *testing.T) {
	host, port := mockSMTPServer(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test Subject",
		HTMLBody: "<p>Hello</p>",
	}

	err := s.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSMTPSender_Send_WithCCAndAttachments(t *testing.T) {
	host, port := mockSMTPServer(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"to@example.com"},
		CC:       []string{"cc@example.com"},
		BCC:      []string{"bcc@example.com"},
		Subject:  "With Attachments",
		HTMLBody: "<p>See attached</p>",
		Attachments: []Attachment{
			{Filename: "test.txt", Content: []byte("file content"), ContentType: "text/plain"},
		},
	}

	err := s.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSMTPSender_Send_ConnectionRefused(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: "127.0.0.1", Port: 59999, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>test</p>",
	}

	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for refused connection")
	}
}

func TestSMTPSender_Send_TLSConnectionRefused(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: "127.0.0.1", Port: 59999, UseTLS: true},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)
	msg := EmailMessage{
		To:       []string{"r@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>test</p>",
	}
	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for TLS connection refused")
	}
	if !strings.Contains(err.Error(), "smtp tls dial") {
		t.Errorf("error = %q, want it to contain 'smtp tls dial'", err.Error())
	}
}

func TestSMTPSender_Send_ResolveBodyError(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: "127.0.0.1", Port: 59999},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)
	msg := EmailMessage{
		To:       []string{"r@example.com"},
		Subject:  "Test",
		Template: "nonexistent",
	}
	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestSMTPSender_Send_WithAuth(t *testing.T) {
	host, port := mockSMTPServer(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false, Username: "user", Password: "pass"},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)
	msg := EmailMessage{
		To:       []string{"r@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>test</p>",
	}
	err := s.Send(context.Background(), msg)
	// The mock SMTP server doesn't support proper AUTH protocol (needs 235 response),
	// so we expect an auth error. This covers the auth code path (lines 77-81).
	if err == nil {
		t.Fatal("expected error for auth with basic mock server")
	}
	if !strings.Contains(err.Error(), "smtp auth") {
		t.Errorf("error = %q, want it to contain 'smtp auth'", err.Error())
	}
}

// mockSMTPServerWithSTARTTLS starts a mock SMTP server that advertises STARTTLS
// but cannot actually perform the TLS handshake, exercising the STARTTLS error path.
func mockSMTPServerWithSTARTTLS(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	addr := ln.Addr().(*net.TCPAddr)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSMTPConnWithSTARTTLS(conn)
		}
	}()

	return addr.IP.String(), addr.Port
}

func handleSMTPConnWithSTARTTLS(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	fmt.Fprintf(writer, "220 localhost SMTP mock\r\n")
	writer.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))

		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			fmt.Fprintf(writer, "250-localhost\r\n250-STARTTLS\r\n250 OK\r\n")
		case cmd == "STARTTLS":
			fmt.Fprintf(writer, "220 Ready to start TLS\r\n")
			writer.Flush()
			// Close connection immediately — client TLS handshake will fail
			return
		case strings.HasPrefix(cmd, "MAIL FROM"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case strings.HasPrefix(cmd, "RCPT TO"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case cmd == "DATA":
			fmt.Fprintf(writer, "354 Start mail input\r\n")
			writer.Flush()
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(dataLine) == "." {
					break
				}
			}
			fmt.Fprintf(writer, "250 OK\r\n")
		case cmd == "QUIT":
			fmt.Fprintf(writer, "221 Bye\r\n")
			writer.Flush()
			return
		default:
			fmt.Fprintf(writer, "250 OK\r\n")
		}
		writer.Flush()
	}
}

func TestSMTPSender_Send_STARTTLSError(t *testing.T) {
	host, port := mockSMTPServerWithSTARTTLS(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test STARTTLS",
		HTMLBody: "<p>Hello</p>",
	}

	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for STARTTLS failure")
	}
	if !strings.Contains(err.Error(), "smtp starttls") {
		t.Errorf("error = %q, want it to contain 'smtp starttls'", err.Error())
	}
}

func TestSMTPSender_Send_WithTemplate(t *testing.T) {
	host, port := mockSMTPServer(t)
	dir := t.TempDir()
	writeTemplate(t, dir, "welcome.html", "<h1>Welcome {{.Name}}</h1>")
	renderer := NewTemplateRenderer(dir, "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:           []string{"recipient@example.com"},
		Subject:      "Welcome",
		Template:     "welcome",
		TemplateData: map[string]any{"Name": "Alice"},
	}

	err := s.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
