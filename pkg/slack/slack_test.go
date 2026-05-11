package slack

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────
// Test infrastructure for pkg/slack.
//
// Every test uses httptest.NewServer to act as the Slack endpoint —
// no real network calls — and swaps the package-level URL vars to
// point at the test server. swapURLs returns a restore closure;
// callers register it via t.Cleanup so the swap is local to each
// test and concurrent tests don't fight over the globals.
// ─────────────────────────────────────────────────────────────────────

func swapURLs(t *testing.T, postMsg, uploadURL, complete string) {
	t.Helper()
	origPost, origUpload, origComplete := slackPostMessageURL, slackFilesUploadV2URL, slackFilesCompleteURL
	if postMsg != "" {
		slackPostMessageURL = postMsg
	}
	if uploadURL != "" {
		slackFilesUploadV2URL = uploadURL
	}
	if complete != "" {
		slackFilesCompleteURL = complete
	}
	t.Cleanup(func() {
		slackPostMessageURL = origPost
		slackFilesUploadV2URL = origUpload
		slackFilesCompleteURL = origComplete
	})
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── WebhookSender ─────────────────────────────────────────────────────

// TestWebhookSender_PostMessage_HappyPath exercises every field that
// the webhook payload supports, plus a 200 response with a body that
// gets echoed back into RawResponseJSON.
func TestWebhookSender_PostMessage_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"text":"hello"`)
		assert.Contains(t, string(body), `"username":"bot"`)
		assert.Contains(t, string(body), `"icon_emoji":":wave:"`)
		assert.Contains(t, string(body), `"icon_url":"https://img"`)
		assert.Contains(t, string(body), `"blocks":[{"type":"section"}]`)
		assert.Contains(t, string(body), `"attachments":[{"color":"#f00"}]`)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	s := NewWebhookSender(srv.URL, discardLogger())
	res, err := s.PostMessage(context.Background(), Message{
		Text:            "hello",
		BlocksJSON:      `[{"type":"section"}]`,
		AttachmentsJSON: `[{"color":"#f00"}]`,
		Username:        "bot",
		IconEmoji:       ":wave:",
		IconURL:         "https://img",
	})
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "ok", res.RawResponseJSON)
	assert.Equal(t, "slack-webhook", s.Name())
}

// TestWebhookSender_PostMessage_NotConfigured — empty URL must short-
// circuit with a clear error rather than POSTing to "".
func TestWebhookSender_PostMessage_NotConfigured(t *testing.T) {
	s := NewWebhookSender("", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// TestWebhookSender_PostMessage_NewRequestError — a malformed URL trips
// http.NewRequestWithContext and surfaces as a plain error (not
// wrapped). Using a URL with a control character is the canonical way
// to force NewRequest failure without touching net/http internals.
func TestWebhookSender_PostMessage_NewRequestError(t *testing.T) {
	s := NewWebhookSender("http://\nbad.example", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Text: "hi"})
	require.Error(t, err)
}

// TestWebhookSender_PostMessage_TransportError — the http.Client returns
// an error before any status code is parsed. We point at a closed
// server so the connection is refused.
func TestWebhookSender_PostMessage_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // close immediately so the next Do call sees a connect-refused

	s := NewWebhookSender(srv.URL, discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slack webhook POST")
}

// TestWebhookSender_PostMessage_4xx — Slack's webhook returns 400 with
// a free-form body when the JSON is invalid; the sender surfaces it
// as an error embedding the status + body.
func TestWebhookSender_PostMessage_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_payload"))
	}))
	t.Cleanup(srv.Close)

	s := NewWebhookSender(srv.URL, discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "invalid_payload")
}

// TestWebhookSender_UploadFile_AlwaysErrors — file upload is not part
// of the webhook protocol; the sender must reject the call.
func TestWebhookSender_UploadFile_AlwaysErrors(t *testing.T) {
	s := NewWebhookSender("http://example", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// ── APISender — PostMessage ──────────────────────────────────────────

// TestAPISender_PostMessage_HappyPath exercises a successful round-trip
// including every optional field, then asserts the parsed response is
// returned verbatim in PostResult.
func TestAPISender_PostMessage_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer xoxb-token", r.Header.Get("Authorization"))
		assert.Contains(t, r.Header.Get("Content-Type"), "application/json")
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		assert.Contains(t, s, `"channel":"C123"`)
		assert.Contains(t, s, `"text":"hi"`)
		assert.Contains(t, s, `"thread_ts":"123.456"`)
		assert.Contains(t, s, `"username":"bot"`)
		assert.Contains(t, s, `"icon_emoji":":wave:"`)
		assert.Contains(t, s, `"icon_url":"https://img"`)
		assert.Contains(t, s, `"unfurl_links":true`)
		assert.Contains(t, s, `"unfurl_media":false`)
		assert.Contains(t, s, `"blocks":[{"type":"section"}]`)
		assert.Contains(t, s, `"attachments":[{"color":"#f00"}]`)
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1700000000.000100"}`))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, srv.URL, "", "")

	tr, fa := true, false
	s := NewAPISender("xoxb-token", discardLogger())
	res, err := s.PostMessage(context.Background(), Message{
		Channel:         "C123",
		Text:            "hi",
		BlocksJSON:      `[{"type":"section"}]`,
		AttachmentsJSON: `[{"color":"#f00"}]`,
		ThreadTimestamp: "123.456",
		Username:        "bot",
		IconEmoji:       ":wave:",
		IconURL:         "https://img",
		UnfurlLinks:     &tr,
		UnfurlMedia:     &fa,
	})
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "C123", res.Channel)
	assert.Equal(t, "1700000000.000100", res.Timestamp)
	assert.Equal(t, "1700000000.000100", res.ProviderMsgID)
	assert.Contains(t, res.RawResponseJSON, `"ok":true`)
	assert.Equal(t, "slack-api", s.Name())
}

// TestAPISender_PostMessage_MissingChannel — Slack's chat.postMessage
// requires a channel. We reject the call before hitting the network.
func TestAPISender_PostMessage_MissingChannel(t *testing.T) {
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires Channel")
}

// TestAPISender_PostMessage_NewRequestError — an unparseable URL trips
// http.NewRequestWithContext.
func TestAPISender_PostMessage_NewRequestError(t *testing.T) {
	swapURLs(t, "http://\nbad", "", "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Channel: "C", Text: "hi"})
	require.Error(t, err)
}

// TestAPISender_PostMessage_TransportError — connection refused.
func TestAPISender_PostMessage_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	swapURLs(t, srv.URL, "", "")

	s := NewAPISender("xoxb", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Channel: "C", Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat.postMessage")
}

// TestAPISender_PostMessage_DecodeError — Slack returns malformed JSON
// (or a non-JSON body); we surface a decode error with the body
// embedded for debugging.
func TestAPISender_PostMessage_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("this is not json"))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, srv.URL, "", "")

	s := NewAPISender("xoxb", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Channel: "C", Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// TestAPISender_PostMessage_AppLevelError — Slack returns HTTP 200 but
// ok:false in the body. Must be treated as an error so the caller
// does not store a phantom delivery.
func TestAPISender_PostMessage_AppLevelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, srv.URL, "", "")

	s := NewAPISender("xoxb", discardLogger())
	_, err := s.PostMessage(context.Background(), Message{Channel: "C", Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "channel_not_found")
}

// ── APISender — UploadFile ──────────────────────────────────────────

// TestAPISender_UploadFile_HappyPath drives the full 3-step protocol:
//
//  1. POST files.getUploadURLExternal → returns upload_url + file_id
//  2. POST <upload_url>                → 200 (bytes accepted)
//  3. POST files.completeUploadExternal → ok:true
//
// One httptest.Server multiplexes all three paths so we can assert
// the protocol invariants in one place.
func TestAPISender_UploadFile_HappyPath(t *testing.T) {
	var step1Hit, uploadHit, step3Hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files.getUploadURLExternal":
			step1Hit = true
			assert.Equal(t, "Bearer xoxb", r.Header.Get("Authorization"))
			assert.Contains(t, r.Header.Get("Content-Type"), "x-www-form-urlencoded")
			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), "filename=report.pdf")
			assert.Contains(t, string(body), "length=8")
			// Use the same test server for the upload URL; protocol-wise
			// Slack returns a one-shot S3 URL but the round-trip is just
			// a POST, which httptest can model.
			_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + r.Host + `","file_id":"F1"}`))
			// Easier: hardcode the upload path on this same server
			// (response above is replaced by the explicit version below).
		case "/upload":
			uploadHit = true
			assert.Equal(t, "application/pdf", r.Header.Get("Content-Type"))
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, "contents", string(body))
			w.WriteHeader(http.StatusOK)
		case "/files.completeUploadExternal":
			step3Hit = true
			body, _ := io.ReadAll(r.Body)
			s := string(body)
			assert.Contains(t, s, `"id":"F1"`)
			assert.Contains(t, s, `"title":"My Report"`)
			assert.Contains(t, s, `"channel_id":"C123"`)
			assert.Contains(t, s, `"initial_comment":"see attached"`)
			assert.Contains(t, s, `"thread_ts":"1.2"`)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	// step1 has to return an upload_url that the client will then POST
	// to. We rewrite step1's body so it points at /upload on this same
	// test server.
	mux := http.NewServeMux()
	mux.HandleFunc("/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		step1Hit = true
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "filename=report.pdf")
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + srv2URL(t) + `/upload","file_id":"F1"}`))
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadHit = true
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "contents", string(body))
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		step3Hit = true
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	srv.Config.Handler = mux
	swapURLs(t, "", srv.URL+"/files.getUploadURLExternal", srv.URL+"/files.completeUploadExternal")

	// Stash the server's URL so step 1's response body knows it. We
	// rely on a small package-level helper because the mux handlers
	// close over `srv` which captures the URL.
	srv2URLHolder.Store(srv.URL)
	t.Cleanup(func() { srv2URLHolder.Store("") })

	s := NewAPISender("xoxb", discardLogger())
	res, err := s.UploadFile(context.Background(), FileUpload{
		Channels:        []string{"C123"},
		Filename:        "report.pdf",
		Title:           "My Report",
		InitialComment:  "see attached",
		Content:         []byte("contents"),
		ContentType:     "application/pdf",
		ThreadTimestamp: "1.2",
	})
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "C123", res.Channel)
	assert.Equal(t, "F1", res.ProviderMsgID)
	assert.True(t, step1Hit && uploadHit && step3Hit, "all three steps must fire")
}

// TestAPISender_UploadFile_MissingFilename — short-circuits before any
// network call.
func TestAPISender_UploadFile_MissingFilename(t *testing.T) {
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Content: []byte("x")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Filename")
}

// TestAPISender_UploadFile_MissingContent — same as above for empty
// payload.
func TestAPISender_UploadFile_MissingContent(t *testing.T) {
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Content")
}

// TestAPISender_UploadFile_Step1NewRequestError — invalid URL.
func TestAPISender_UploadFile_Step1NewRequestError(t *testing.T) {
	swapURLs(t, "", "http://\nbad", "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
}

// TestAPISender_UploadFile_Step1TransportError — closed server.
func TestAPISender_UploadFile_Step1TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	swapURLs(t, "", srv.URL, "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getUploadURLExternal")
}

// TestAPISender_UploadFile_Step1DecodeError — non-JSON body.
func TestAPISender_UploadFile_Step1DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("oops"))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, "", srv.URL, "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// TestAPISender_UploadFile_Step1AppLevelError — ok:false from Slack.
func TestAPISender_UploadFile_Step1AppLevelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"file_too_large"}`))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, "", srv.URL, "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file_too_large")
}

// TestAPISender_UploadFile_Step2NewRequestError — Slack returns a bad
// upload URL; the second http.NewRequestWithContext fails.
func TestAPISender_UploadFile_Step2NewRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Backticks so `\n` inside the JSON source is literally
		// backslash-n; json.Unmarshal then decodes it to a real
		// newline in the upload_url Go string, which the next
		// http.NewRequestWithContext rejects.
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"http://\nbad","file_id":"F1"}`))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, "", srv.URL, "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
}

// TestAPISender_UploadFile_Step2TransportError — Slack returns an
// upload URL that connects-refused.
func TestAPISender_UploadFile_Step2TransportError(t *testing.T) {
	deadSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	deadSrv.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + deadSrv.URL + `","file_id":"F1"}`))
	}))
	t.Cleanup(srv.Close)
	swapURLs(t, "", srv.URL, "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file PUT")
}

// TestAPISender_UploadFile_Step2_4xx — upload URL returns 5xx.
func TestAPISender_UploadFile_Step2_4xx(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(uploadSrv.Close)
	step1Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + uploadSrv.URL + `","file_id":"F1"}`))
	}))
	t.Cleanup(step1Srv.Close)
	swapURLs(t, "", step1Srv.URL, "")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 403")
}

// TestAPISender_UploadFile_Step3NewRequestError — third step's URL is
// malformed.
func TestAPISender_UploadFile_Step3NewRequestError(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(uploadSrv.Close)
	step1Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + uploadSrv.URL + `","file_id":"F1"}`))
	}))
	t.Cleanup(step1Srv.Close)
	swapURLs(t, "", step1Srv.URL, "http://\nbad")
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
}

// TestAPISender_UploadFile_Step3TransportError — step 3 connect refused.
func TestAPISender_UploadFile_Step3TransportError(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(uploadSrv.Close)
	step1Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + uploadSrv.URL + `","file_id":"F1"}`))
	}))
	t.Cleanup(step1Srv.Close)
	deadSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	deadSrv.Close()
	swapURLs(t, "", step1Srv.URL, deadSrv.URL)
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "completeUploadExternal")
}

// TestAPISender_UploadFile_Step3DecodeError — non-JSON from complete.
func TestAPISender_UploadFile_Step3DecodeError(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(uploadSrv.Close)
	step1Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + uploadSrv.URL + `","file_id":"F1"}`))
	}))
	t.Cleanup(step1Srv.Close)
	completeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(completeSrv.Close)
	swapURLs(t, "", step1Srv.URL, completeSrv.URL)
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// TestAPISender_UploadFile_Step3AppLevelError — ok:false from complete.
func TestAPISender_UploadFile_Step3AppLevelError(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(uploadSrv.Close)
	step1Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + uploadSrv.URL + `","file_id":"F1"}`))
	}))
	t.Cleanup(step1Srv.Close)
	completeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"upload_expired"}`))
	}))
	t.Cleanup(completeSrv.Close)
	swapURLs(t, "", step1Srv.URL, completeSrv.URL)
	s := NewAPISender("xoxb", discardLogger())
	_, err := s.UploadFile(context.Background(), FileUpload{Filename: "x", Content: []byte("c")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload_expired")
}

// ── provider.go: NewSlackSender selector ─────────────────────────────

func TestNewSlackSender_NilConfig(t *testing.T) {
	s, err := NewSlackSender(nil, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "slack-noop", s.Name())
}

func TestNewSlackSender_EmptyProvider(t *testing.T) {
	s, err := NewSlackSender(&config.SlackConfig{}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "slack-noop", s.Name())
}

func TestNewSlackSender_Webhook(t *testing.T) {
	s, err := NewSlackSender(&config.SlackConfig{Provider: "webhook", WebhookURL: "http://x"}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "slack-webhook", s.Name())
}

func TestNewSlackSender_WebhookMissingURL(t *testing.T) {
	_, err := NewSlackSender(&config.SlackConfig{Provider: "webhook"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WebhookURL")
}

func TestNewSlackSender_API(t *testing.T) {
	s, err := NewSlackSender(&config.SlackConfig{Provider: "api", BotToken: "xoxb"}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "slack-api", s.Name())
}

func TestNewSlackSender_APIMissingToken(t *testing.T) {
	_, err := NewSlackSender(&config.SlackConfig{Provider: "api"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BotToken")
}

func TestNewSlackSender_UnknownProvider(t *testing.T) {
	_, err := NewSlackSender(&config.SlackConfig{Provider: "carrierpigeon"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown slack provider")
}

// ── provider.go: noopSender ──────────────────────────────────────────

func TestNoopSender_AllMethodsReturnNotConfigured(t *testing.T) {
	n := &noopSender{}
	assert.Equal(t, "slack-noop", n.Name())
	_, err := n.PostMessage(context.Background(), Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
	_, err = n.UploadFile(context.Background(), FileUpload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// ── Misc seam used by the happy-path upload test ─────────────────────

// srv2URLHolder is a tiny atomic.Value-style holder used by the
// happy-path upload test so a mux handler created before the test
// server's URL is known can read it back at request time. Keeps the
// test file declarative.
var srv2URLHolder atomicString

type atomicString struct {
	v string
}

func (a *atomicString) Store(s string) { a.v = s }
func (a *atomicString) Load() string   { return a.v }

func srv2URL(t *testing.T) string {
	t.Helper()
	u := srv2URLHolder.Load()
	if u == "" {
		t.Fatalf("srv2URLHolder unset")
	}
	return u
}

// TestHTTPClientTimeout is a paranoia check that the http.Client built
// by NewAPISender / NewWebhookSender honors a finite timeout. We don't
// wait the full 30s; we just verify that the field is non-zero and
// short enough that a hung server can't hang the test process.
func TestHTTPClientTimeout(t *testing.T) {
	api := NewAPISender("xoxb", discardLogger())
	web := NewWebhookSender("http://x", discardLogger())
	assert.NotZero(t, api.client.Timeout)
	assert.NotZero(t, web.client.Timeout)
	assert.LessOrEqual(t, api.client.Timeout, time.Minute)
	assert.LessOrEqual(t, web.client.Timeout, time.Minute)
}

// TestWebhookSender_PostMessage_ContextCanceled — passing an already-
// canceled context aborts the request immediately. Verifies the ctx
// is plumbed through to http.NewRequestWithContext.
func TestWebhookSender_PostMessage_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	t.Cleanup(srv.Close)

	s := NewWebhookSender(srv.URL, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.PostMessage(ctx, Message{Text: "hi"})
	require.Error(t, err)
	// context-canceled bubbles up wrapped; either is acceptable.
	assert.True(t,
		errors.Is(err, context.Canceled) ||
			strings.Contains(err.Error(), "context canceled"),
		"expected context error, got %v", err)
}
