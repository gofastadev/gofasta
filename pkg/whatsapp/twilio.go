package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofastadev/gofasta/pkg/config"
)

// TwilioSender talks to Twilio's Programmable Messaging API. Auth is
// HTTP Basic with Account SID as username and Auth Token as password.
//
// WhatsApp addresses are namespaced with "whatsapp:" — both the
// from-number and the recipient. The from-number must be a Twilio
// WhatsApp sender that has cleared the WhatsApp template review.
//
// Endpoint:
//
//	POST https://api.twilio.com/2010-04-01/Accounts/{SID}/Messages.json
//
// Form fields: From, To, Body, MediaUrl (optional, supports multiple),
// StatusCallback (optional). Media must be reachable by Twilio over
// the public internet — there is no media-upload endpoint.
type TwilioSender struct {
	cfg    config.WhatsAppTwilioConfig
	client *http.Client
	logger *slog.Logger
}

// twilioBaseURL is the production Twilio Messages API root. Vars (not
// const) so tests can repoint at an httptest server. Production never
// reassigns; _test.go reassigns + restores via t.Cleanup.
var twilioBaseURL = "https://api.twilio.com"

// NewTwilioSender constructs the Twilio-backed Sender. Account SID +
// Auth Token authenticate every call (HTTP Basic).
func NewTwilioSender(cfg config.WhatsAppTwilioConfig, logger *slog.Logger) *TwilioSender {
	return &TwilioSender{
		cfg:    cfg,
		client: &http.Client{Timeout: defaultWhatsAppHTTPTimeout},
		logger: logger,
	}
}

// Name implements Sender.
func (s *TwilioSender) Name() string { return "whatsapp-twilio" }

// Send dispatches via Twilio's Messages API.
func (s *TwilioSender) Send(ctx context.Context, msg Message) (*SendResult, error) {
	endpoint := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", twilioBaseURL, s.cfg.AccountSID)
	form := url.Values{}
	form.Set("From", "whatsapp:"+strings.TrimPrefix(s.cfg.FromNumber, "whatsapp:"))
	form.Set("To", "whatsapp:"+normalizeTwilioTo(msg.To))
	form.Set("Body", msg.Body)
	if msg.Media != nil && msg.Media.URL != "" {
		// Twilio supports up to 10 MediaUrl entries; we send one.
		form.Set("MediaUrl", msg.Media.URL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.AccountSID, s.cfg.AuthToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twilio POST: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("twilio: status %d body=%s", resp.StatusCode, string(body))
	}

	var tr twilioMessageResponse
	if jsonErr := json.Unmarshal(body, &tr); jsonErr != nil {
		return nil, fmt.Errorf("twilio decode: %w (body: %s)", jsonErr, string(body))
	}
	return &SendResult{
		OK:              true,
		ProviderMsgID:   tr.SID,
		Status:          tr.Status,
		RawResponseJSON: string(body),
	}, nil
}

// DeleteMessage — Twilio doesn't expose a "delete sent message" API for
// WhatsApp. WhatsApp's revoke-for-everyone is enforced by the
// platform, not by Twilio. Surface ErrUnsupported so callers handle
// it gracefully.
func (s *TwilioSender) DeleteMessage(_ context.Context, _ string) error {
	return ErrUnsupported
}

// normalizeTwilioTo prepends "+" if the caller handed us bare digits.
// Twilio rejects an unprefixed number even when the country code is
// otherwise valid.
func normalizeTwilioTo(to string) string {
	to = strings.TrimSpace(to)
	if !strings.HasPrefix(to, "+") {
		return "+" + to
	}
	return to
}

type twilioMessageResponse struct {
	SID    string `json:"sid"`
	Status string `json:"status"`
}
