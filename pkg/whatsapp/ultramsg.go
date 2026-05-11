package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
)

const defaultWhatsAppHTTPTimeout = 30 * time.Second

// UltraMsgSender talks to api.ultramsg.com against a single instance.
// UltraMsg uses simple form-encoded POSTs:
//
//	{base}/instance{ID}/messages/chat      — text + optional media URL
//	{base}/instance{ID}/messages/image     — image
//	{base}/instance{ID}/messages/document  — document
//	{base}/instance{ID}/messages/delete    — delete by id
//	{base}/instance{ID}/media/upload       — upload bytes → returns URL
//
// All take `token` as a form field, not a header.
//
// `to` is the bare digits — no `+`, no `whatsapp:` prefix.
type UltraMsgSender struct {
	cfg    config.WhatsAppUltraMsgConfig
	client *http.Client
	logger *slog.Logger
}

// NewUltraMsgSender constructs the sender. Trims trailing slash on
// BaseURL so callers can pass either form.
func NewUltraMsgSender(cfg config.WhatsAppUltraMsgConfig, logger *slog.Logger) *UltraMsgSender {
	cfg.BaseURL = strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	return &UltraMsgSender{
		cfg:    cfg,
		client: &http.Client{Timeout: defaultWhatsAppHTTPTimeout},
		logger: logger,
	}
}

// Name implements Sender.
func (s *UltraMsgSender) Name() string { return "whatsapp-ultramsg" }

// Send dispatches the Message to UltraMsg, choosing the chat or media
// endpoint based on whether msg.Media is populated.
func (s *UltraMsgSender) Send(ctx context.Context, msg Message) (*SendResult, error) {
	to := normalizeUltraMsgTo(msg.To)
	if msg.Media == nil {
		return s.sendChat(ctx, to, msg.Body)
	}
	return s.sendMedia(ctx, to, msg)
}

func (s *UltraMsgSender) sendChat(ctx context.Context, to, body string) (*SendResult, error) {
	endpoint := fmt.Sprintf("%s/%s/messages/chat", s.cfg.BaseURL, s.cfg.InstanceID)
	form := url.Values{}
	form.Set("token", s.cfg.Token)
	form.Set("to", to)
	form.Set("body", body)
	return s.postForm(ctx, endpoint, form)
}

// sendMedia handles the "media" branch. UltraMsg lets you pass a public
// URL directly via the `image` / `document` field — much simpler than
// the upload-then-attach 2-step. We only fall back to media/upload when
// the caller hands us raw bytes.
func (s *UltraMsgSender) sendMedia(ctx context.Context, to string, msg Message) (*SendResult, error) {
	mediaURL := msg.Media.URL
	if mediaURL == "" {
		// Step 1 — upload bytes to UltraMsg, get back a URL.
		uploaded, err := s.uploadBytes(ctx, msg.Media)
		if err != nil {
			return nil, fmt.Errorf("ultramsg media upload: %w", err)
		}
		mediaURL = uploaded
	}

	mediaType := strings.ToLower(strings.TrimSpace(msg.Media.Type))
	endpoint := fmt.Sprintf("%s/%s/messages/%s", s.cfg.BaseURL, s.cfg.InstanceID, mediaType)
	form := url.Values{}
	form.Set("token", s.cfg.Token)
	form.Set("to", to)
	switch mediaType {
	case "image":
		form.Set("image", mediaURL)
		form.Set("caption", msg.Media.Caption)
	case "document":
		form.Set("document", mediaURL)
		form.Set("filename", msg.Media.Filename)
		form.Set("caption", msg.Media.Caption)
	case "video":
		form.Set("video", mediaURL)
		form.Set("caption", msg.Media.Caption)
	case "audio":
		form.Set("audio", mediaURL)
	default:
		return nil, fmt.Errorf("ultramsg: unknown media type %q", mediaType)
	}
	return s.postForm(ctx, endpoint, form)
}

func (s *UltraMsgSender) uploadBytes(ctx context.Context, media *MediaAttachment) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/media/upload", s.cfg.BaseURL, s.cfg.InstanceID)
	// UltraMsg accepts media-upload via a base64 field (file=base64string).
	// This keeps the protocol form-encoded and avoids multipart streaming.
	form := url.Values{}
	form.Set("token", s.cfg.Token)
	form.Set("file", base64String(media.Content))
	res, err := s.postFormRaw(ctx, endpoint, form)
	if err != nil {
		return "", err
	}
	var u uploadResponse
	if jsonErr := json.Unmarshal(res, &u); jsonErr != nil {
		return "", fmt.Errorf("ultramsg upload decode: %w (body: %s)", jsonErr, string(res))
	}
	if !u.Success || u.URL == "" {
		return "", fmt.Errorf("ultramsg upload failed: %s", u.Error)
	}
	return u.URL, nil
}

// DeleteMessage hits messages/delete with the provider message id.
// Implements Sender.
func (s *UltraMsgSender) DeleteMessage(ctx context.Context, providerMsgID string) error {
	if providerMsgID == "" {
		return fmt.Errorf("ultramsg delete: providerMsgID is required")
	}
	endpoint := fmt.Sprintf("%s/%s/messages/delete", s.cfg.BaseURL, s.cfg.InstanceID)
	form := url.Values{}
	form.Set("token", s.cfg.Token)
	form.Set("id", providerMsgID)
	_, err := s.postForm(ctx, endpoint, form)
	return err
}

func (s *UltraMsgSender) postForm(ctx context.Context, endpoint string, form url.Values) (*SendResult, error) {
	body, err := s.postFormRaw(ctx, endpoint, form)
	if err != nil {
		return nil, err
	}
	var resp ultraMsgResponse
	if jsonErr := json.Unmarshal(body, &resp); jsonErr != nil {
		return nil, fmt.Errorf("ultramsg decode: %w (body: %s)", jsonErr, string(body))
	}
	if resp.Sent != "true" && !resp.Success {
		return nil, fmt.Errorf("ultramsg: %s", resp.Message)
	}
	return &SendResult{
		OK:              true,
		ProviderMsgID:   resp.ID,
		Status:          "sent",
		RawResponseJSON: string(body),
	}, nil
}

func (s *UltraMsgSender) postFormRaw(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ultramsg POST %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ultramsg %s: status %d body=%s", endpoint, resp.StatusCode, string(body))
	}
	return body, nil
}

// normalizeUltraMsgTo strips a leading "+" since UltraMsg expects bare
// digits. Defensive — most callers already follow E.164 with "+".
func normalizeUltraMsgTo(to string) string {
	to = strings.TrimSpace(to)
	to = strings.TrimPrefix(to, "+")
	return to
}

func base64String(b []byte) string {
	// inlined to avoid pulling in encoding/base64 across this file's
	// import block when the rest only needs json/net/http/url.
	return stdBase64.EncodeToString(b)
}

// — wire types —

type ultraMsgResponse struct {
	ID      string `json:"id"`
	Sent    string `json:"sent"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type uploadResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url"`
	Error   string `json:"error,omitempty"`
}

// keep readability — bytes Buffer used in the test/upload path.
var _ = bytes.NewBuffer
