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
)

func newTestResendSender(t *testing.T, transport http.RoundTripper) *ResendSender {
	t.Helper()
	renderer := NewTemplateRenderer(t.TempDir(), "TestApp")
	r := NewResendSender(
		config.ResendConfig{APIKey: "re_test_key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	if transport != nil {
		r.client = &http.Client{Transport: transport}
	}
	return r
}

func TestNewResendSender(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	r := NewResendSender(
		config.ResendConfig{APIKey: "key123"},
		"FromName", "from@example.com",
		renderer, slog.Default(),
	)
	if r == nil {
		t.Fatal("expected non-nil ResendSender")
	}
	if r.from != "FromName <from@example.com>" {
		t.Errorf("from = %q, want %q", r.from, "FromName <from@example.com>")
	}
	if r.client == nil {
		t.Error("expected non-nil http.Client")
	}
}

func TestNewResendSender_EmptyFromName(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	r := NewResendSender(
		config.ResendConfig{APIKey: "key123"},
		"", "from@example.com",
		renderer, slog.Default(),
	)
	if r.from != "from@example.com" {
		t.Errorf("from = %q, want bare address %q", r.from, "from@example.com")
	}
}

func TestResendSender_Send_Success(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		capturedReq = req
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"49a3999c-0ce1-4ea6"}`)),
		}, nil
	})

	r := newTestResendSender(t, transport)
	err := r.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi there</p>",
		TextBody: "Hi there",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq.URL.String() != resendAPIURL {
		t.Errorf("url = %q, want %q", capturedReq.URL.String(), resendAPIURL)
	}
	if got := capturedReq.Header.Get("Authorization"); got != "Bearer re_test_key" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer re_test_key")
	}

	var payload resendRequest
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if payload.From != "Sender <sender@example.com>" {
		t.Errorf("from = %q, want %q", payload.From, "Sender <sender@example.com>")
	}
	if len(payload.To) != 1 || payload.To[0] != "user@example.com" {
		t.Errorf("to = %v, want [user@example.com]", payload.To)
	}
	if payload.Subject != "Hello" {
		t.Errorf("subject = %q, want %q", payload.Subject, "Hello")
	}
	if payload.HTML != "<p>Hi there</p>" {
		t.Errorf("html = %q, want %q", payload.HTML, "<p>Hi there</p>")
	}
	if payload.Text != "Hi there" {
		t.Errorf("text = %q, want %q", payload.Text, "Hi there")
	}
}

func TestResendSender_Send_WithAllOptions(t *testing.T) {
	var capturedBody []byte

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg-1"}`)),
		}, nil
	})

	r := newTestResendSender(t, transport)
	err := r.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com", "user2@example.com"},
		CC:       []string{"cc@example.com"},
		BCC:      []string{"bcc@example.com"},
		ReplyTo:  "reply@example.com",
		Subject:  "Full Email",
		HTMLBody: "<p>Body</p>",
		Attachments: []Attachment{
			{
				Filename:    "report.pdf",
				Content:     []byte("file content"),
				ContentType: "application/pdf",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload resendRequest
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if len(payload.To) != 2 {
		t.Errorf("to count = %d, want 2", len(payload.To))
	}
	if len(payload.CC) != 1 || payload.CC[0] != "cc@example.com" {
		t.Errorf("cc = %v, want [cc@example.com]", payload.CC)
	}
	if len(payload.BCC) != 1 || payload.BCC[0] != "bcc@example.com" {
		t.Errorf("bcc = %v, want [bcc@example.com]", payload.BCC)
	}
	if len(payload.ReplyTo) != 1 || payload.ReplyTo[0] != "reply@example.com" {
		t.Errorf("reply_to = %v, want [reply@example.com]", payload.ReplyTo)
	}
	if len(payload.Attachments) != 1 {
		t.Fatalf("attachments count = %d, want 1", len(payload.Attachments))
	}
	att := payload.Attachments[0]
	if att.Filename != "report.pdf" {
		t.Errorf("attachment filename = %q, want %q", att.Filename, "report.pdf")
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("attachment content_type = %q, want %q", att.ContentType, "application/pdf")
	}
	// Attachment.Content carries raw bytes; the wire format must be base64.
	if want := base64.StdEncoding.EncodeToString([]byte("file content")); att.Content != want {
		t.Errorf("attachment content = %q, want base64 %q", att.Content, want)
	}
}

func TestResendSender_Send_TemplateRendering(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "welcome.html", "<h1>Welcome {{.Name}}!</h1>")

	var capturedBody []byte
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg-1"}`)),
		}, nil
	})

	renderer := NewTemplateRenderer(dir, "TestApp")
	r := NewResendSender(
		config.ResendConfig{APIKey: "re_test_key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	r.client = &http.Client{Transport: transport}

	err := r.Send(context.Background(), EmailMessage{
		To:           []string{"user@example.com"},
		Subject:      "Welcome",
		Template:     "welcome",
		TemplateData: map[string]any{"Name": "Ada"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload resendRequest
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if !strings.Contains(payload.HTML, "Welcome Ada!") {
		t.Errorf("html = %q, want it to contain %q", payload.HTML, "Welcome Ada!")
	}
}

func TestResendSender_Send_TemplateError(t *testing.T) {
	r := newTestResendSender(t, nil)
	err := r.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		Template: "does-not-exist",
	})
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestResendSender_Send_ClientError(t *testing.T) {
	clientErr := errors.New("connection refused")
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, clientErr
	})

	r := newTestResendSender(t, transport)
	err := r.Send(context.Background(), EmailMessage{
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

func TestResendSender_Send_APIError(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 403,
			Body:       io.NopCloser(strings.NewReader(`{"message":"invalid api key"}`)),
		}, nil
	})

	r := newTestResendSender(t, transport)
	err := r.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi</p>",
	})
	if err == nil {
		t.Fatal("expected error for 403 status")
	}
	for _, want := range []string{"resend error:", "status 403", "invalid api key"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}
