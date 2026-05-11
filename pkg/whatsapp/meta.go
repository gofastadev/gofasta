package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofastadev/gofasta/pkg/config"
)

const defaultMetaAPIVersion = "v20.0"

// metaGraphBaseURL is the production Meta Graph API root. Var (not
// const) so tests can repoint at an httptest server. Production never
// reassigns; _test.go reassigns + restores via t.Cleanup.
var metaGraphBaseURL = "https://graph.facebook.com"

// MetaSender talks directly to the Meta WhatsApp Cloud API (Graph
// version). Bearer auth, JSON payloads, per-WABA phone-number IDs.
//
// Endpoint shape:
//
//	POST https://graph.facebook.com/{version}/{phone-number-id}/messages
//
// The same endpoint accepts text, media, template, and interactive
// payloads — disambiguated by the `type` field in the JSON body.
//
// Media has TWO routes:
//   - reuse a hosted public URL (link or id) — single POST to /messages
//   - upload bytes first to /{phone-number-id}/media (multipart),
//     then reference the returned media_id from /messages
//
// This sender takes the simpler `link` path when a URL is available
// and the upload-then-reference path when only Content is provided.
type MetaSender struct {
	cfg    config.WhatsAppMetaConfig
	client *http.Client
	logger *slog.Logger
}

// NewMetaSender constructs the Meta-Cloud-API-backed Sender. APIVersion
// defaults to v20.0 when empty.
func NewMetaSender(cfg config.WhatsAppMetaConfig, logger *slog.Logger) *MetaSender {
	if cfg.APIVersion == "" {
		cfg.APIVersion = defaultMetaAPIVersion
	}
	return &MetaSender{
		cfg:    cfg,
		client: &http.Client{Timeout: defaultWhatsAppHTTPTimeout},
		logger: logger,
	}
}

// Name implements Sender.
func (s *MetaSender) Name() string { return "whatsapp-meta" }

// Send dispatches the Message via Meta's WhatsApp Cloud API.
//
// to Meta's per-type top-level keys (image/document/video/audio).
// Extracting a per-type helper would push the same branch one frame
// deeper without removing the work.
//
//nolint:gocyclo // payload assembly switch on media type is inherent
func (s *MetaSender) Send(ctx context.Context, msg Message) (*SendResult, error) {
	to := strings.TrimPrefix(strings.TrimSpace(msg.To), "+")
	if to == "" {
		return nil, fmt.Errorf("meta whatsapp: To is required")
	}
	endpoint := fmt.Sprintf("%s/%s/%s/messages",
		metaGraphBaseURL, s.cfg.APIVersion, s.cfg.PhoneNumberID)

	payload := metaSendPayload{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
	}
	if msg.Media == nil {
		payload.Type = "text"
		payload.Text = &metaTextPayload{Body: msg.Body}
		if msg.PreviewURL != nil {
			payload.Text.PreviewURL = *msg.PreviewURL
		}
	} else {
		mediaID, mediaURL, err := s.resolveMedia(ctx, msg.Media)
		if err != nil {
			return nil, err
		}
		mediaType := strings.ToLower(strings.TrimSpace(msg.Media.Type))
		body := metaMediaPayload{Caption: msg.Media.Caption}
		if mediaURL != "" {
			body.Link = mediaURL
		}
		if mediaID != "" {
			body.ID = mediaID
		}
		if mediaType == "document" && msg.Media.Filename != "" {
			body.Filename = msg.Media.Filename
		}
		switch mediaType {
		case "image":
			payload.Type = "image"
			payload.Image = &body
		case "document":
			payload.Type = "document"
			payload.Document = &body
		case "video":
			payload.Type = "video"
			payload.Video = &body
		case "audio":
			payload.Type = "audio"
			payload.Audio = &body
		default:
			return nil, fmt.Errorf("meta whatsapp: unknown media type %q", mediaType)
		}
	}
	if msg.ReplyToProviderMsgID != "" {
		payload.Context = &metaContextPayload{MessageID: msg.ReplyToProviderMsgID}
	}

	body, err := s.postJSON(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var resp metaSendResponse
	if jsonErr := json.Unmarshal(body, &resp); jsonErr != nil {
		return nil, fmt.Errorf("meta decode: %w (body: %s)", jsonErr, string(body))
	}
	if len(resp.Messages) == 0 {
		return nil, fmt.Errorf("meta: empty messages array (body: %s)", string(body))
	}
	return &SendResult{
		OK:              true,
		ProviderMsgID:   resp.Messages[0].ID,
		Status:          "queued",
		RawResponseJSON: string(body),
	}, nil
}

// DeleteMessage isn't a public Cloud API operation today; Meta hasn't
// shipped revoke-for-everyone for outbound messages from businesses.
// Returning ErrUnsupported keeps the contract honest.
func (s *MetaSender) DeleteMessage(_ context.Context, _ string) error {
	return ErrUnsupported
}

// resolveMedia returns either (mediaID, "", nil) for upload-then-reference
// or ("", url, nil) for the simpler link path.
func (s *MetaSender) resolveMedia(ctx context.Context, media *MediaAttachment) (mediaID, mediaURL string, err error) {
	if media.URL != "" {
		return "", media.URL, nil
	}
	if len(media.Content) == 0 {
		return "", "", fmt.Errorf("meta whatsapp: media must have URL or Content")
	}
	uploadEndpoint := fmt.Sprintf("%s/%s/%s/media",
		metaGraphBaseURL, s.cfg.APIVersion, s.cfg.PhoneNumberID)
	// Multipart upload: messaging_product, type, file (the bytes).
	mid, uerr := uploadMetaMedia(ctx, s.client, uploadEndpoint, s.cfg.AccessToken, media)
	if uerr != nil {
		return "", "", fmt.Errorf("meta media upload: %w", uerr)
	}
	return mid, "", nil
}

func (s *MetaSender) postJSON(ctx context.Context, endpoint string, payload any) ([]byte, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.AccessToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta POST: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("meta %d: %s", resp.StatusCode, string(respBytes))
	}
	return respBytes, nil
}

// — wire types —

type metaSendPayload struct {
	MessagingProduct string              `json:"messaging_product"`
	RecipientType    string              `json:"recipient_type"`
	To               string              `json:"to"`
	Type             string              `json:"type"`
	Text             *metaTextPayload    `json:"text,omitempty"`
	Image            *metaMediaPayload   `json:"image,omitempty"`
	Document         *metaMediaPayload   `json:"document,omitempty"`
	Video            *metaMediaPayload   `json:"video,omitempty"`
	Audio            *metaMediaPayload   `json:"audio,omitempty"`
	Context          *metaContextPayload `json:"context,omitempty"`
}

type metaTextPayload struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

type metaMediaPayload struct {
	Link     string `json:"link,omitempty"`
	ID       string `json:"id,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type metaContextPayload struct {
	MessageID string `json:"message_id"`
}

type metaSendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}
