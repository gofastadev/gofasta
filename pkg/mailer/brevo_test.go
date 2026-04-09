package mailer

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
)

// roundTripFunc is an adapter to use ordinary functions as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestBrevoSender(t *testing.T, transport http.RoundTripper) *BrevoSender {
	t.Helper()
	renderer := NewTemplateRenderer(t.TempDir(), "TestApp")
	b := NewBrevoSender(
		config.BrevoConfig{APIKey: "test-api-key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)
	if transport != nil {
		b.client = &http.Client{Transport: transport}
	}
	return b
}

func TestNewBrevoSender(t *testing.T) {
	renderer := NewTemplateRenderer(t.TempDir(), "App")
	b := NewBrevoSender(
		config.BrevoConfig{APIKey: "key123"},
		"FromName", "from@example.com",
		renderer, slog.Default(),
	)
	if b == nil {
		t.Fatal("expected non-nil BrevoSender")
	}
	if b.from.Email != "from@example.com" {
		t.Errorf("from.Email = %q, want %q", b.from.Email, "from@example.com")
	}
	if b.from.Name != "FromName" {
		t.Errorf("from.Name = %q, want %q", b.from.Name, "FromName")
	}
	if b.client == nil {
		t.Error("expected non-nil http.Client")
	}
}

func TestBrevoSender_Send_Success(t *testing.T) {
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
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"messageId":"abc123"}`)),
		}, nil
	})

	b := newTestBrevoSender(t, transport)
	err := b.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi there</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq.Header.Get("api-key") != "test-api-key" {
		t.Errorf("api-key header = %q, want %q", capturedReq.Header.Get("api-key"), "test-api-key")
	}

	var body brevoRequest
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if body.Subject != "Hello" {
		t.Errorf("subject = %q, want %q", body.Subject, "Hello")
	}
	if len(body.To) != 1 || body.To[0].Email != "user@example.com" {
		t.Errorf("to = %+v, want [{Email:user@example.com}]", body.To)
	}
}

func TestBrevoSender_Send_Error(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(strings.NewReader(`{"message":"bad request"}`)),
		}, nil
	})

	b := newTestBrevoSender(t, transport)
	err := b.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Hello",
		HTMLBody: "<p>Hi</p>",
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want it to contain '400'", err.Error())
	}
}

func TestBrevoSender_Send_WithReplyTo(t *testing.T) {
	var capturedBody []byte

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	b := newTestBrevoSender(t, transport)
	err := b.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Reply Test",
		HTMLBody: "<p>body</p>",
		ReplyTo:  "reply@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body brevoRequest
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if body.ReplyTo == nil {
		t.Fatal("expected ReplyTo to be set")
	}
	if body.ReplyTo.Email != "reply@example.com" {
		t.Errorf("ReplyTo.Email = %q, want %q", body.ReplyTo.Email, "reply@example.com")
	}
}

func TestBrevoSender_Send_WithCCAndBCC(t *testing.T) {
	var capturedBody []byte

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"messageId":"abc123"}`)),
		}, nil
	})

	b := newTestBrevoSender(t, transport)
	err := b.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		CC:       []string{"cc1@example.com", "cc2@example.com"},
		BCC:      []string{"bcc@example.com"},
		Subject:  "CC/BCC Test",
		HTMLBody: "<p>body</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body brevoRequest
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(body.CC) != 2 {
		t.Errorf("CC count = %d, want 2", len(body.CC))
	}
	if body.CC[0].Email != "cc1@example.com" {
		t.Errorf("CC[0].Email = %q, want %q", body.CC[0].Email, "cc1@example.com")
	}
	if len(body.BCC) != 1 {
		t.Errorf("BCC count = %d, want 1", len(body.BCC))
	}
	if body.BCC[0].Email != "bcc@example.com" {
		t.Errorf("BCC[0].Email = %q, want %q", body.BCC[0].Email, "bcc@example.com")
	}
}

func TestBrevoSender_Send_TemplateError(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	b := newTestBrevoSender(t, transport)
	err := b.Send(context.Background(), EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "Template Error",
		Template: "nonexistent_template",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestBrevoSender_ResolveBody_HTMLBody(t *testing.T) {
	b := newTestBrevoSender(t, nil)
	msg := EmailMessage{HTMLBody: "<p>Direct HTML</p>"}
	body, err := b.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "<p>Direct HTML</p>" {
		t.Errorf("body = %q, want %q", body, "<p>Direct HTML</p>")
	}
}

func TestBrevoSender_ResolveBody_Template(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "greeting.html", "<p>Hello {{.Name}}</p>")
	renderer := NewTemplateRenderer(dir, "App")

	b := NewBrevoSender(
		config.BrevoConfig{APIKey: "key"},
		"Sender", "sender@example.com",
		renderer, slog.Default(),
	)

	msg := EmailMessage{
		Template:     "greeting",
		TemplateData: map[string]any{"Name": "Bob"},
	}
	body, err := b.resolveBody(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "Hello Bob") {
		t.Errorf("body = %q, want it to contain 'Hello Bob'", body)
	}
}
