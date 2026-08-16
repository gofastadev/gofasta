package mailer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// mockSendGridClient implements sendgridClient for testing.
type mockSendGridClient struct {
	sendFunc func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error)
}

func (m *mockSendGridClient) SendWithContext(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
	return m.sendFunc(ctx, email)
}

func newTestSendGridSender(t *testing.T, mock *mockSendGridClient) *SendGridSender {
	t.Helper()
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "sg-key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	s.client = mock
	return s
}

func TestNewSendGridSender(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "sg-key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	if s == nil {
		t.Fatal("expected non-nil SendGridSender")
	}
	if s.from.Name != "Sender" {
		t.Errorf("from.Name = %q, want %q", s.from.Name, "Sender")
	}
	if s.from.Address != "sender@example.com" {
		t.Errorf("from.Address = %q, want %q", s.from.Address, "sender@example.com")
	}
}

func TestSendGridSender_ResolveBody_HTMLBody(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "key"},
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

func TestSendGridSender_ResolveBody_Template(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "notify.html", "<p>Notification for {{.User}}</p>")
	renderer := NewTemplateRenderer(dir, "App")

	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		Template:     "notify",
		TemplateData: map[string]any{"User": "Charlie"},
	}
	body, err := s.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "Notification for Charlie") {
		t.Errorf("body = %q, want it to contain 'Notification for Charlie'", body)
	}
}

func TestSendGridSender_Send_Success(t *testing.T) {
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			return &rest.Response{StatusCode: 202, Body: ""}, nil
		},
	}
	s := newTestSendGridSender(t, mock)

	err := s.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendGridSender_Send_APIError(t *testing.T) {
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			return &rest.Response{StatusCode: 400, Body: "bad request"}, nil
		},
	}
	s := newTestSendGridSender(t, mock)

	err := s.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi</p>",
	})
	if err == nil {
		t.Fatal("expected error for 400 status")
	}
	if !strings.Contains(err.Error(), "sendgrid error") {
		t.Errorf("error = %q, want it to contain 'sendgrid error'", err.Error())
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want it to contain '400'", err.Error())
	}
}

func TestSendGridSender_Send_ClientError(t *testing.T) {
	clientErr := errors.New("connection refused")
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			return nil, clientErr
		},
	}
	s := newTestSendGridSender(t, mock)

	err := s.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi</p>",
	})
	if err == nil {
		t.Fatal("expected error when client returns error")
	}
	if !errors.Is(err, clientErr) {
		t.Errorf("error = %q, want it to wrap %q", err.Error(), clientErr.Error())
	}
}

func TestSendGridSender_Send_WithAllOptions(t *testing.T) {
	var capturedEmail *mail.SGMailV3
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			capturedEmail = email
			return &rest.Response{StatusCode: 202, Body: ""}, nil
		},
	}
	s := newTestSendGridSender(t, mock)

	attachmentContent := base64.StdEncoding.EncodeToString([]byte("file content"))
	err := s.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com", "user2@example.com"},
		Subject:  "Full Email",
		HTMLBody: "<p>Body</p>",
		CC:       []string{"cc@example.com"},
		BCC:      []string{"bcc@example.com"},
		ReplyTo:  "reply@example.com",
		Attachments: []Attachment{
			{
				Filename:    "report.pdf",
				Content:     []byte(attachmentContent),
				ContentType: "application/pdf",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedEmail == nil {
		t.Fatal("expected email to be captured by mock")
	}
	if capturedEmail.Subject != "Full Email" {
		t.Errorf("subject = %q, want %q", capturedEmail.Subject, "Full Email")
	}
	if capturedEmail.ReplyTo.Address != "reply@example.com" {
		t.Errorf("replyTo = %q, want %q", capturedEmail.ReplyTo.Address, "reply@example.com")
	}

	p := capturedEmail.Personalizations
	if len(p) != 1 {
		t.Fatalf("personalizations count = %d, want 1", len(p))
	}
	if len(p[0].To) != 2 {
		t.Errorf("To count = %d, want 2", len(p[0].To))
	}
	if len(p[0].CC) != 1 {
		t.Errorf("CC count = %d, want 1", len(p[0].CC))
	}
	if len(p[0].BCC) != 1 {
		t.Errorf("BCC count = %d, want 1", len(p[0].BCC))
	}
	if len(capturedEmail.Attachments) != 1 {
		t.Fatalf("attachments count = %d, want 1", len(capturedEmail.Attachments))
	}
	if capturedEmail.Attachments[0].Filename != "report.pdf" {
		t.Errorf("attachment filename = %q, want %q", capturedEmail.Attachments[0].Filename, "report.pdf")
	}
}

func TestSendGridSender_Send_TemplateRendering(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "welcome.html", "<h1>Welcome {{.Name}}!</h1>")

	var capturedEmail *mail.SGMailV3
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			capturedEmail = email
			return &rest.Response{StatusCode: 202, Body: ""}, nil
		},
	}

	renderer := NewTemplateRenderer(dir, "App")
	s := NewSendGridSender(
		config.SendGridConfig{APIKey: "sg-key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	s.client = mock

	err := s.Send(context.Background(), EmailMessage{
		To:           []string{"user@example.com"},
		Subject:      "Welcome",
		Template:     "welcome",
		TemplateData: map[string]any{"Name": "Alice"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedEmail == nil {
		t.Fatal("expected email to be captured")
	}
	// Check that the rendered template content is in the email body
	found := false
	for _, c := range capturedEmail.Content {
		if strings.Contains(c.Value, "Welcome Alice!") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected email content to contain rendered template 'Welcome Alice!'")
	}
}

func TestSendGridSender_Send_TemplateError(t *testing.T) {
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			t.Fatal("SendWithContext should not be called when template rendering fails")
			return nil, nil
		},
	}
	s := newTestSendGridSender(t, mock)

	err := s.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Test",
		Template: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for non-existent template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to contain 'not found'", err.Error())
	}
}

func TestSendGridSender_Send_TextBodyListedBeforeHTML(t *testing.T) {
	var capturedEmail *mail.SGMailV3
	mock := &mockSendGridClient{
		sendFunc: func(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
			capturedEmail = email
			return &rest.Response{StatusCode: 202, Body: ""}, nil
		},
	}
	s := newTestSendGridSender(t, mock)

	err := s.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Both bodies",
		HTMLBody: "<p>Hello</p>",
		TextBody: "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedEmail == nil {
		t.Fatal("expected email to be captured by mock")
	}
	if len(capturedEmail.Content) != 2 {
		t.Fatalf("content parts = %d, want 2", len(capturedEmail.Content))
	}
	if capturedEmail.Content[0].Type != "text/plain" || capturedEmail.Content[0].Value != "Hello" {
		t.Errorf("first content part = %q %q, want text/plain %q", capturedEmail.Content[0].Type, capturedEmail.Content[0].Value, "Hello")
	}
	if capturedEmail.Content[1].Type != "text/html" || capturedEmail.Content[1].Value != "<p>Hello</p>" {
		t.Errorf("second content part = %q %q, want text/html %q", capturedEmail.Content[1].Type, capturedEmail.Content[1].Value, "<p>Hello</p>")
	}
}

func TestSendGridSender_Send_AttachmentContentIsBase64(t *testing.T) {
	// SendGrid requires attachment content base64-encoded. Sending raw bytes
	// produced a request the API accepted and a file the recipient could not
	// open, with success reported either way.
	var captured []byte
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		captured, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: 202,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Header:     make(http.Header),
		}, nil
	})

	s := NewSendGridSender(config.SendGridConfig{APIKey: "k"}, "From", "from@example.com",
		nil, slog.Default(), WithHTTPClient(&http.Client{Transport: transport}))

	pdf := []byte("%PDF-1.4 binary\x00\xff bytes")
	if err := s.Send(context.Background(), EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "cert",
		HTMLBody: "<p>hi</p>",
		Attachments: []Attachment{
			{Filename: "certificate.pdf", Content: pdf, ContentType: "application/pdf"},
		},
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	var body struct {
		Attachments []struct {
			Content  string `json:"content"`
			Filename string `json:"filename"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(captured, &body); err != nil {
		t.Fatalf("payload is not JSON: %v\n%s", err, captured)
	}
	if len(body.Attachments) != 1 {
		t.Fatalf("attachment count = %d, want 1", len(body.Attachments))
	}

	decoded, err := base64.StdEncoding.DecodeString(body.Attachments[0].Content)
	if err != nil {
		t.Fatalf("attachment content is not base64: %v (%q)", err, body.Attachments[0].Content)
	}
	if string(decoded) != string(pdf) {
		t.Errorf("decoded attachment = %q, want the original bytes", decoded)
	}
}
