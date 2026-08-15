package mailer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
)

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
