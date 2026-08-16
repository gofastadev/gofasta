package mailer

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
)

func TestResolveSenderOptions_DefaultsToABoundedClient(t *testing.T) {
	// The defect this replaced: a zero-value http.Client has no timeout, so a
	// provider that accepted the connection and then went quiet held the
	// calling goroutine for the life of the process.
	got := resolveSenderOptions(nil)
	if got.httpClient == nil {
		t.Fatal("no default client")
	}
	if got.httpClient.Timeout == 0 {
		t.Error("default client has no timeout")
	}
	if got.httpClient.Timeout != defaultHTTPTimeout {
		t.Errorf("timeout = %v, want %v", got.httpClient.Timeout, defaultHTTPTimeout)
	}
}

func TestWithHTTPClient_IsUsedAsGiven(t *testing.T) {
	custom := &http.Client{Timeout: 3 * time.Second}
	got := resolveSenderOptions([]SenderOption{WithHTTPClient(custom)})
	if got.httpClient != custom {
		t.Error("supplied client was not used")
	}
}

func TestWithHTTPClient_IgnoresNil(t *testing.T) {
	// A nil client would disable the default and leave the sender with no
	// timeout at all — the exact failure the default exists to prevent.
	got := resolveSenderOptions([]SenderOption{WithHTTPClient(nil)})
	if got.httpClient == nil || got.httpClient.Timeout != defaultHTTPTimeout {
		t.Error("nil client displaced the bounded default")
	}
}

func TestBrevoSender_UsesTheSuppliedClient(t *testing.T) {
	called := false
	custom := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, context.Canceled
	})}

	b := NewBrevoSender(config.BrevoConfig{APIKey: "k"}, "From", "from@example.com",
		nil, slog.Default(), WithHTTPClient(custom))
	_ = b.Send(context.Background(), EmailMessage{
		To: []string{"to@example.com"}, Subject: "s", HTMLBody: "<p>b</p>",
	})

	if !called {
		t.Error("Brevo did not send through the supplied client")
	}
}

func TestSendGridSender_UsesTheSuppliedClient(t *testing.T) {
	// Also guards the reason this is not done by reassigning
	// rest.DefaultClient: that global must stay untouched.
	called := false
	custom := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, context.Canceled
	})}

	s := NewSendGridSender(config.SendGridConfig{APIKey: "k"}, "From", "from@example.com",
		nil, slog.Default(), WithHTTPClient(custom))
	_ = s.Send(context.Background(), EmailMessage{
		To: []string{"to@example.com"}, Subject: "s", HTMLBody: "<p>b</p>",
	})

	if !called {
		t.Error("SendGrid did not send through the supplied client")
	}
}
