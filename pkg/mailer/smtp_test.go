package mailer

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestSMTPSender_Send_AttachmentNoContentType(t *testing.T) {
	host, port := mockSMTPServer(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "With Attachment",
		HTMLBody: "<p>See attached</p>",
		Attachments: []Attachment{
			{Filename: "data.bin", Content: []byte("binary data")},
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

// mockSMTPServerWithFailure starts a mock SMTP server that returns a failure
// response when it receives the specified command prefix.
func mockSMTPServerWithFailure(t *testing.T, failCmd string, failResp string) (string, int) {
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
			go handleSMTPConnWithFailure(conn, failCmd, failResp)
		}
	}()

	return addr.IP.String(), addr.Port
}

func handleSMTPConnWithFailure(conn net.Conn, failCmd, failResp string) {
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
			fmt.Fprintf(writer, "250-localhost\r\n250 OK\r\n")
		case strings.HasPrefix(cmd, failCmd):
			fmt.Fprintf(writer, "%s\r\n", failResp)
			writer.Flush()
			// Read QUIT if the client sends it
			for {
				qLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				qCmd := strings.ToUpper(strings.TrimSpace(qLine))
				if qCmd == "QUIT" {
					fmt.Fprintf(writer, "221 Bye\r\n")
					writer.Flush()
					return
				}
			}
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

func TestSMTPSender_Send_MailFromError(t *testing.T) {
	host, port := mockSMTPServerWithFailure(t, "MAIL FROM", "550 Sender rejected")
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>Hello</p>",
	}

	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for MAIL FROM rejection")
	}
	if !strings.Contains(err.Error(), "smtp mail") {
		t.Errorf("error = %q, want it to contain 'smtp mail'", err.Error())
	}
}

func TestSMTPSender_Send_RcptToError(t *testing.T) {
	host, port := mockSMTPServerWithFailure(t, "RCPT TO", "550 Recipient rejected")
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>Hello</p>",
	}

	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for RCPT TO rejection")
	}
	if !strings.Contains(err.Error(), "smtp rcpt") {
		t.Errorf("error = %q, want it to contain 'smtp rcpt'", err.Error())
	}
}

func TestSMTPSender_Send_DataError(t *testing.T) {
	host, port := mockSMTPServerWithFailure(t, "DATA", "554 Transaction failed")
	renderer := NewTemplateRenderer(t.TempDir(), "App")

	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: false},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>Hello</p>",
	}

	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for DATA rejection")
	}
	if !strings.Contains(err.Error(), "smtp data") {
		t.Errorf("error = %q, want it to contain 'smtp data'", err.Error())
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

func generateSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

func mockTLSSMTPServer(t *testing.T) (string, int) {
	t.Helper()
	cert := generateSelfSignedCert(t)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("tls listen: %v", err)
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

func TestSMTPSender_Send_TLSSuccess(t *testing.T) {
	host, port := mockTLSSMTPServer(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port, UseTLS: true, InsecureSkipVerify: true},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)
	msg := EmailMessage{
		To:       []string{"r@example.com"},
		Subject:  "TLS Test",
		HTMLBody: "<p>Hello TLS</p>",
	}
	err := s.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mockSMTPServerNoGreeting(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	addr := ln.Addr().(*net.TCPAddr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return addr.IP.String(), addr.Port
}

func TestSMTPSender_Send_NoGreeting(t *testing.T) {
	host, port := mockSMTPServerNoGreeting(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port},
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
		t.Fatal("expected error when server sends no greeting")
	}
	if !strings.Contains(err.Error(), "smtp client") {
		t.Errorf("error = %q, want it to contain 'smtp client'", err.Error())
	}
}

func mockSMTPServerCloseAfterData(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	addr := ln.Addr().(*net.TCPAddr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSMTPConnCloseAfterData(conn)
		}
	}()
	return addr.IP.String(), addr.Port
}

func handleSMTPConnCloseAfterData(conn net.Conn) {
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
			fmt.Fprintf(writer, "250-localhost\r\n250 OK\r\n")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case strings.HasPrefix(cmd, "RCPT TO"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case cmd == "DATA":
			fmt.Fprintf(writer, "354 Start mail input\r\n")
			writer.Flush()
			// Read a few bytes of the data, then RST the connection.
			// Setting TCP linger to 0 causes a RST instead of FIN,
			// which makes the client's Write fail with "broken pipe".
			buf := make([]byte, 64)
			conn.Read(buf)
			if tc, ok := conn.(*net.TCPConn); ok {
				tc.SetLinger(0)
			}
			return
		default:
			fmt.Fprintf(writer, "250 OK\r\n")
		}
		writer.Flush()
	}
}

func TestSMTPSender_Send_WriteCloseError(t *testing.T) {
	host, port := mockSMTPServerCloseAfterData(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port},
		"Test", "test@example.com",
		renderer, slog.Default(),
	)

	// The body must exceed the combined kernel socket buffers so the client
	// cannot finish Write before the server's RST arrives: the mock reads only
	// 64 bytes, so the client blocks mid-Write until the RST unblocks it with
	// a broken pipe. A smaller body can fit entirely in the loopback buffers,
	// letting Write return nil and deferring the error to Close.
	largeBody := strings.Repeat("X", 16*1024*1024)
	msg := EmailMessage{
		To:       []string{"r@example.com"},
		Subject:  "Test",
		HTMLBody: largeBody,
	}
	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error when server closes after DATA")
	}
	if !strings.Contains(err.Error(), "smtp write") {
		t.Errorf("error = %q, want 'smtp write'", err.Error())
	}
}

// mockSMTPServerRejectAfterDot creates a server that accepts DATA, reads the full body,
// but returns a 554 error after the terminating dot. This causes wc.Close() to fail.
func mockSMTPServerRejectAfterDot(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	addr := ln.Addr().(*net.TCPAddr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSMTPConnRejectAfterDot(conn)
		}
	}()
	return addr.IP.String(), addr.Port
}

func handleSMTPConnRejectAfterDot(conn net.Conn) {
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
			fmt.Fprintf(writer, "250-localhost\r\n250 OK\r\n")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case strings.HasPrefix(cmd, "RCPT TO"):
			fmt.Fprintf(writer, "250 OK\r\n")
		case cmd == "DATA":
			fmt.Fprintf(writer, "354 Start mail input\r\n")
			writer.Flush()
			// Read the full body until the terminating "."
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(dataLine) == "." {
					break
				}
			}
			// Reject the message after the dot — this causes wc.Close() to return error
			fmt.Fprintf(writer, "554 Transaction failed\r\n")
			writer.Flush()
			// Read QUIT
			reader.ReadString('\n')
			fmt.Fprintf(writer, "221 Bye\r\n")
			writer.Flush()
			return
		default:
			fmt.Fprintf(writer, "250 OK\r\n")
		}
		writer.Flush()
	}
}

func TestSMTPSender_Send_CloseError(t *testing.T) {
	host, port := mockSMTPServerRejectAfterDot(t)
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSMTPSender(
		config.SMTPConfig{Host: host, Port: port},
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
		t.Fatal("expected error when server rejects after dot")
	}
	if !strings.Contains(err.Error(), "smtp close") {
		t.Errorf("error = %q, want 'smtp close'", err.Error())
	}
}

// capturingSMTPServer is mockSMTPServer that keeps the DATA payload, so a
// test can assert on the MIME document the sender actually composed rather
// than only on whether the exchange succeeded.
func capturingSMTPServer(t *testing.T) (host string, port int, body *string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	var mu sync.Mutex
	captured := ""

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				w := bufio.NewWriter(conn)
				r := bufio.NewReader(conn)
				fmt.Fprint(w, "220 localhost SMTP mock\r\n")
				_ = w.Flush()
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(line))
					switch cmd {
					case "DATA":
						fmt.Fprint(w, "354 Start mail input\r\n")
						_ = w.Flush()
						var sb strings.Builder
						for {
							dl, err := r.ReadString('\n')
							if err != nil {
								return
							}
							if strings.TrimSpace(dl) == "." {
								break
							}
							sb.WriteString(dl)
						}
						mu.Lock()
						captured = sb.String()
						mu.Unlock()
						fmt.Fprint(w, "250 OK\r\n")
					case "QUIT":
						fmt.Fprint(w, "221 Bye\r\n")
						_ = w.Flush()
						return
					default:
						fmt.Fprint(w, "250 OK\r\n")
					}
					_ = w.Flush()
				}
			}(conn)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, &captured
}

// smtpBodyFor sends msg through a real SMTPSender and returns the MIME
// document the server received.
func smtpBodyFor(t *testing.T, msg EmailMessage) string {
	t.Helper()
	host, port, captured := capturingSMTPServer(t)

	s := NewSMTPSender(config.SMTPConfig{Host: host, Port: port}, "From", "from@example.com", nil, nil)
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	return *captured
}

// A message with only HTML keeps the single-part form — no gratuitous
// multipart wrapper for clients that never asked for one.
func TestSMTPHTMLOnlyStaysSinglePart(t *testing.T) {
	body := smtpBodyFor(t, EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "html only",
		HTMLBody: "<p>hi</p>",
	})
	if !strings.Contains(body, "Content-Type: text/html; charset=UTF-8") {
		t.Fatalf("expected a plain text/html message, got:\n%s", body)
	}
	if strings.Contains(body, "multipart/") {
		t.Fatalf("did not expect a multipart wrapper, got:\n%s", body)
	}
}

// Both bodies: multipart/alternative, plain text before HTML so a client that
// understands both renders the HTML (RFC 2046 §5.1.4).
func TestSMTPBothBodiesUseAlternativeInOrder(t *testing.T) {
	body := smtpBodyFor(t, EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "both",
		HTMLBody: "<p>rich</p>",
		TextBody: "plain",
	})
	if !strings.Contains(body, "multipart/alternative") {
		t.Fatalf("expected multipart/alternative, got:\n%s", body)
	}
	textAt := strings.Index(body, "text/plain")
	htmlAt := strings.Index(body, "text/html")
	if textAt < 0 || htmlAt < 0 {
		t.Fatalf("expected both parts, got:\n%s", body)
	}
	if textAt > htmlAt {
		t.Fatal("text/plain must precede text/html: clients pick the LAST part they understand")
	}
}

// With attachments the two bodies nest inside an alternative, so the
// attachments stay siblings of the pair rather than of the HTML alone.
func TestSMTPAttachmentsNestTheAlternative(t *testing.T) {
	body := smtpBodyFor(t, EmailMessage{
		To:          []string{"to@example.com"},
		Subject:     "with attachment",
		HTMLBody:    "<p>rich</p>",
		TextBody:    "plain",
		Attachments: []Attachment{{Filename: "a.pdf", Content: []byte("x"), ContentType: "application/pdf"}},
	})
	if !strings.Contains(body, "multipart/mixed") {
		t.Fatalf("expected multipart/mixed at the top, got:\n%s", body)
	}
	if !strings.Contains(body, "multipart/alternative") {
		t.Fatalf("expected a nested alternative, got:\n%s", body)
	}
	if !strings.Contains(body, "a.pdf") {
		t.Fatalf("expected the attachment, got:\n%s", body)
	}
}

// The pre-existing shape — attachments, HTML only — must not change.
func TestSMTPAttachmentsWithoutTextBodyUnchanged(t *testing.T) {
	body := smtpBodyFor(t, EmailMessage{
		To:          []string{"to@example.com"},
		Subject:     "attachment only",
		HTMLBody:    "<p>rich</p>",
		Attachments: []Attachment{{Filename: "a.pdf", Content: []byte("x")}},
	})
	if strings.Contains(body, "multipart/alternative") {
		t.Fatalf("no TextBody, so no alternative expected:\n%s", body)
	}
	if !strings.Contains(body, "multipart/mixed") || !strings.Contains(body, "a.pdf") {
		t.Fatalf("expected mixed with the attachment, got:\n%s", body)
	}
}

// A nil logger must not panic. The constructors accept one, and every send
// path logs on success — so a sender built with nil worked right up until it
// worked, which is the worst possible time to find out.
func TestSendersTolerateNilLogger(t *testing.T) {
	host, port, _ := capturingSMTPServer(t)

	s := NewSMTPSender(config.SMTPConfig{Host: host, Port: port}, "From", "from@example.com", nil, nil)
	if err := s.Send(context.Background(), EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "nil logger",
		HTMLBody: "<p>hi</p>",
	}); err != nil {
		t.Fatalf("Send with nil logger: %v", err)
	}

	// The API senders take the same argument and must be built safely too.
	_ = NewBrevoSender(config.BrevoConfig{}, "From", "from@example.com", nil, nil)
	_ = NewSendGridSender(config.SendGridConfig{}, "From", "from@example.com", nil, nil)
}

// Reply-To is what makes a no-reply From usable: the platform sends, a human
// receives the reply. Brevo, SendGrid and Resend all honor
// EmailMessage.ReplyTo, so SMTP dropping it made one field mean two different
// things depending on which provider a tenant had configured.
//
// These reuse smtpBodyFor / capturingSMTPServer from smtp_test.go so the
// assertions are on the MIME document that actually went over the wire.

func TestSMTPWritesReplyTo(t *testing.T) {
	body := smtpBodyFor(t, EmailMessage{
		To:       []string{"learner@example.com"},
		Subject:  "Your certificate",
		HTMLBody: "<p>Congratulations</p>",
		ReplyTo:  "support@example.com",
	})

	if !strings.Contains(body, "Reply-To: support@example.com\r\n") {
		t.Fatalf("Reply-To header missing from the wire payload:\n%s", body)
	}
}

func TestSMTPOmitsEmptyReplyTo(t *testing.T) {
	// An empty field must not produce a bare "Reply-To:" header — a malformed
	// header is grounds for some receivers to reject the message outright.
	body := smtpBodyFor(t, EmailMessage{
		To:       []string{"learner@example.com"},
		Subject:  "No reply address",
		HTMLBody: "<p>Body</p>",
	})

	if strings.Contains(body, "Reply-To:") {
		t.Fatalf("Reply-To header written with no address:\n%s", body)
	}
}

func TestSMTPReplyToSitsWithTheOtherHeaders(t *testing.T) {
	body := smtpBodyFor(t, EmailMessage{
		To:       []string{"learner@example.com"},
		CC:       []string{"facilitator@example.com"},
		Subject:  "Ordering check",
		HTMLBody: "<p>Body</p>",
		ReplyTo:  "support@example.com",
	})

	for _, header := range []string{
		"To: learner@example.com",
		"CC: facilitator@example.com",
		"Reply-To: support@example.com",
		"Subject: Ordering check",
		"MIME-Version: 1.0",
	} {
		if !strings.Contains(body, header+"\r\n") {
			t.Errorf("missing header %q in:\n%s", header, body)
		}
	}

	// Headers precede the body — a misplaced write would land Reply-To inside
	// the message text, where it is inert.
	if strings.Index(body, "Reply-To:") > strings.Index(body, "<p>Body</p>") {
		t.Error("Reply-To was written after the message body")
	}
}
