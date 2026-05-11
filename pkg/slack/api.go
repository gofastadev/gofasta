package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Endpoint URLs are vars (not consts) so tests can point them at an
// httptest server. Production code never reassigns them; only
// _test.go files do, and they restore the originals on cleanup.
var (
	slackPostMessageURL   = "https://slack.com/api/chat.postMessage"
	slackFilesUploadV2URL = "https://slack.com/api/files.getUploadURLExternal"
	slackFilesCompleteURL = "https://slack.com/api/files.completeUploadExternal"
)

const defaultSlackHTTPTimeout = 30 * time.Second

// APISender posts to api.slack.com using a bot token.
//
// Threading model: every Send takes a fresh ctx; the http.Client is
// shared across calls (HTTP keep-alive amortizes TLS). The bot token
// is set once at construction; rotating tokens means rebuilding the
// sender. Most apps tolerate that — token rotation already requires
// a redeploy.
type APISender struct {
	token  string
	client *http.Client
	logger *slog.Logger
}

// NewAPISender constructs a sender backed by the standard slack.com
// HTTP API. The bot token must have at least chat:write to call
// PostMessage and files:write to call UploadFile.
func NewAPISender(token string, logger *slog.Logger) *APISender {
	return &APISender{
		token:  token,
		client: &http.Client{Timeout: defaultSlackHTTPTimeout},
		logger: logger,
	}
}

// Name reports the provider identifier; persisted on log rows so a
// row identifies its delivery path.
func (s *APISender) Name() string { return "slack-api" }

// PostMessage hits chat.postMessage. Returns a PostResult with the
// canonical ts so callers can thread replies. A 200 with `ok:false`
// in the body is surfaced as an error — Slack's API uses HTTP 200
// for application-level failures (token expired, channel_not_found),
// so we MUST inspect the body.
func (s *APISender) PostMessage(ctx context.Context, msg Message) (*PostResult, error) {
	if msg.Channel == "" {
		return nil, fmt.Errorf("slack: PostMessage requires Channel")
	}
	body := postMessagePayload{Channel: msg.Channel}
	if msg.Text != "" {
		body.Text = msg.Text
	}
	if msg.BlocksJSON != "" {
		body.Blocks = json.RawMessage(msg.BlocksJSON)
	}
	if msg.AttachmentsJSON != "" {
		body.Attachments = json.RawMessage(msg.AttachmentsJSON)
	}
	if msg.ThreadTimestamp != "" {
		body.ThreadTS = msg.ThreadTimestamp
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
	if msg.UnfurlLinks != nil {
		v := *msg.UnfurlLinks
		body.UnfurlLinks = &v
	}
	if msg.UnfurlMedia != nil {
		v := *msg.UnfurlMedia
		body.UnfurlMedia = &v
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackPostMessageURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack chat.postMessage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var pm postMessageResponse
	if jsonErr := json.Unmarshal(respBytes, &pm); jsonErr != nil {
		return nil, fmt.Errorf("slack chat.postMessage: decode: %w (body: %s)", jsonErr, string(respBytes))
	}
	if !pm.OK {
		return nil, fmt.Errorf("slack chat.postMessage: %s", pm.Error)
	}
	return &PostResult{
		OK:              true,
		Channel:         pm.Channel,
		Timestamp:       pm.TS,
		ProviderMsgID:   pm.TS,
		RawResponseJSON: string(respBytes),
	}, nil
}

// UploadFile uses Slack's files.uploadV2 protocol (the legacy
// files.upload was deprecated 2025-05). Two API calls + one PUT:
//
//  1. POST files.getUploadURLExternal — Slack mints a one-shot S3 URL
//  2. PUT  <upload_url>                 — the bytes go up directly
//  3. POST files.completeUploadExternal — finalizes and shares
//
// All three are bundled into a single call from the caller's view.
//
// validation lives inline for readability. Splitting would just push
// branching one frame deeper.
//
//nolint:gocyclo // protocol-driven 3-step sequence; each step's
func (s *APISender) UploadFile(ctx context.Context, file FileUpload) (*PostResult, error) {
	if file.Filename == "" {
		return nil, fmt.Errorf("slack: UploadFile requires Filename")
	}
	if len(file.Content) == 0 {
		return nil, fmt.Errorf("slack: UploadFile requires Content")
	}

	// Step 1 — get the upload URL.
	step1Form := strings.NewReader(fmt.Sprintf(
		"filename=%s&length=%d", file.Filename, len(file.Content),
	))
	req1, err := http.NewRequestWithContext(ctx, http.MethodPost, slackFilesUploadV2URL, step1Form)
	if err != nil {
		return nil, err
	}
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req1.Header.Set("Authorization", "Bearer "+s.token)
	resp1, err := s.client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("slack files.getUploadURLExternal: %w", err)
	}
	defer func() { _ = resp1.Body.Close() }()
	body1, _ := io.ReadAll(io.LimitReader(resp1.Body, 1<<16))
	var u uploadURLResponse
	if jsonErr := json.Unmarshal(body1, &u); jsonErr != nil {
		return nil, fmt.Errorf("slack getUploadURL decode: %w (body: %s)", jsonErr, string(body1))
	}
	if !u.OK {
		return nil, fmt.Errorf("slack files.getUploadURLExternal: %s", u.Error)
	}

	// Step 2 — PUT the bytes to the URL Slack returned.
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.UploadURL, bytes.NewReader(file.Content))
	if err != nil {
		return nil, err
	}
	if file.ContentType != "" {
		uploadReq.Header.Set("Content-Type", file.ContentType)
	}
	uploadResp, err := s.client.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("slack file PUT: %w", err)
	}
	_ = uploadResp.Body.Close()
	if uploadResp.StatusCode >= 400 {
		return nil, fmt.Errorf("slack file PUT: status %d", uploadResp.StatusCode)
	}

	// Step 3 — complete the upload, sharing into the requested channels.
	step3Body := completeUploadPayload{
		Files: []completeUploadFile{{ID: u.FileID, Title: file.Title}},
	}
	if len(file.Channels) > 0 {
		step3Body.ChannelID = file.Channels[0]
	}
	if file.InitialComment != "" {
		step3Body.InitialComment = file.InitialComment
	}
	if file.ThreadTimestamp != "" {
		step3Body.ThreadTS = file.ThreadTimestamp
	}
	step3JSON, _ := json.Marshal(step3Body)
	req3, err := http.NewRequestWithContext(ctx, http.MethodPost, slackFilesCompleteURL, bytes.NewReader(step3JSON))
	if err != nil {
		return nil, err
	}
	req3.Header.Set("Content-Type", "application/json; charset=utf-8")
	req3.Header.Set("Authorization", "Bearer "+s.token)
	resp3, err := s.client.Do(req3)
	if err != nil {
		return nil, fmt.Errorf("slack files.completeUploadExternal: %w", err)
	}
	defer func() { _ = resp3.Body.Close() }()
	body3, _ := io.ReadAll(io.LimitReader(resp3.Body, 1<<16))
	var c completeUploadResponse
	if jsonErr := json.Unmarshal(body3, &c); jsonErr != nil {
		return nil, fmt.Errorf("slack completeUpload decode: %w (body: %s)", jsonErr, string(body3))
	}
	if !c.OK {
		return nil, fmt.Errorf("slack files.completeUploadExternal: %s", c.Error)
	}
	return &PostResult{
		OK:              true,
		Channel:         step3Body.ChannelID,
		ProviderMsgID:   u.FileID,
		RawResponseJSON: string(body3),
	}, nil
}

// keep multipart import live for future direct multipart upload paths.
var _ = multipart.NewWriter

// — wire types —

type postMessagePayload struct {
	Channel     string          `json:"channel"`
	Text        string          `json:"text,omitempty"`
	Blocks      json.RawMessage `json:"blocks,omitempty"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
	ThreadTS    string          `json:"thread_ts,omitempty"`
	Username    string          `json:"username,omitempty"`
	IconEmoji   string          `json:"icon_emoji,omitempty"`
	IconURL     string          `json:"icon_url,omitempty"`
	UnfurlLinks *bool           `json:"unfurl_links,omitempty"`
	UnfurlMedia *bool           `json:"unfurl_media,omitempty"`
}

type postMessageResponse struct {
	OK      bool   `json:"ok"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Error   string `json:"error,omitempty"`
}

type uploadURLResponse struct {
	OK        bool   `json:"ok"`
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
	Error     string `json:"error,omitempty"`
}

type completeUploadPayload struct {
	Files          []completeUploadFile `json:"files"`
	ChannelID      string               `json:"channel_id,omitempty"`
	InitialComment string               `json:"initial_comment,omitempty"`
	ThreadTS       string               `json:"thread_ts,omitempty"`
}

type completeUploadFile struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

type completeUploadResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}
