package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// WebhookSender posts to a single incoming-webhook URL. The simplest
// integration path — no bot token, no scopes, channel determined by
// who created the webhook.
//
// Limitations baked into the protocol (not into this code): no file
// uploads, no per-message channel routing, no chat:write API. Use
// APISender for those.
type WebhookSender struct {
	webhookURL string
	client     *http.Client
	logger     *slog.Logger
}

// NewWebhookSender constructs a webhook-only sender. Pass an empty
// URL to disable — Send becomes a no-op that returns an explanatory
// error so the operator sees it in logs without the app crashing.
func NewWebhookSender(webhookURL string, logger *slog.Logger) *WebhookSender {
	return &WebhookSender{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: defaultSlackHTTPTimeout},
		logger:     logger,
	}
}

// Name implements Sender.
func (s *WebhookSender) Name() string { return "slack-webhook" }

// PostMessage POSTs to the configured webhook. The Channel/Username/
// IconEmoji/IconURL fields ARE honored — Slack's webhook protocol
// accepts them at the JSON level even though they cannot override
// the channel the webhook is bound to.
func (s *WebhookSender) PostMessage(ctx context.Context, msg Message) (*PostResult, error) {
	if s.webhookURL == "" {
		return nil, fmt.Errorf("slack webhook: not configured")
	}

	body := postMessagePayload{}
	if msg.Text != "" {
		body.Text = msg.Text
	}
	if msg.BlocksJSON != "" {
		body.Blocks = json.RawMessage(msg.BlocksJSON)
	}
	if msg.AttachmentsJSON != "" {
		body.Attachments = json.RawMessage(msg.AttachmentsJSON)
	}
	if msg.Username != "" {
		body.Username = msg.Username
	}
	if msg.IconEmoji != "" {
		body.IconEmoji = msg.IconEmoji
	}
	if msg.IconURL != "" {
		body.IconURL = msg.IconURL
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack webhook POST: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("slack webhook %d: %s", resp.StatusCode, string(respBytes))
	}
	return &PostResult{
		OK:              true,
		RawResponseJSON: string(respBytes),
	}, nil
}

// UploadFile is unsupported for webhook senders — the protocol has no
// file primitive. Use APISender if you need file uploads.
func (s *WebhookSender) UploadFile(_ context.Context, _ FileUpload) (*PostResult, error) {
	return nil, fmt.Errorf("slack webhook: file upload not supported (use api provider)")
}
