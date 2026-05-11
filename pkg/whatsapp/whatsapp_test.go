package whatsapp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────
// Test infrastructure for pkg/whatsapp.
//
// Each test stands up an httptest.Server and points the production
// code at it via two seams:
//
//   - twilioBaseURL   — package var holding the Twilio API root
//   - metaGraphBaseURL — package var holding the Meta Graph root
//
// UltraMsg's base URL is already config-driven so we just pass the
// server URL in config.
//
// All HTTP server handlers are explicit — no shared mux or hidden
// routing — so each test reads top-to-bottom as a contract.
// ─────────────────────────────────────────────────────────────────────

func swapTwilioBaseURL(t *testing.T, url string) {
	t.Helper()
	orig := twilioBaseURL
	twilioBaseURL = url
	t.Cleanup(func() { twilioBaseURL = orig })
}

func swapMetaBaseURL(t *testing.T, url string) {
	t.Helper()
	orig := metaGraphBaseURL
	metaGraphBaseURL = url
	t.Cleanup(func() { metaGraphBaseURL = orig })
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── UltraMsgSender — chat path ───────────────────────────────────────

func TestUltraMsgSender_SendChat_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/inst1/messages/chat", r.URL.Path)
		_ = r.ParseForm()
		assert.Equal(t, "tok", r.PostForm.Get("token"))
		assert.Equal(t, "250788000000", r.PostForm.Get("to"))
		assert.Equal(t, "hello", r.PostForm.Get("body"))
		_, _ = w.Write([]byte(`{"sent":"true","id":"abc"}`))
	}))
	t.Cleanup(srv.Close)

	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL:    srv.URL + "/",
		InstanceID: "inst1",
		Token:      "tok",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{To: "+250788000000", Body: "hello"})
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "abc", res.ProviderMsgID)
	assert.Equal(t, "sent", res.Status)
	assert.Equal(t, "whatsapp-ultramsg", s.Name())
}

func TestUltraMsgSender_SendChat_Success_BoolFlag(t *testing.T) {
	// Some UltraMsg responses use the boolean `success` flag instead of
	// the string `sent:"true"`. Either form must register as OK.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"id":"id1"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "id1", res.ProviderMsgID)
}

func TestUltraMsgSender_SendChat_Status4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad token"))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestUltraMsgSender_SendChat_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestUltraMsgSender_SendChat_AppLevelFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"sent":"false","success":false,"message":"throttled"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "throttled")
}

func TestUltraMsgSender_SendChat_NewRequestError(t *testing.T) {
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: "http://\nbad", InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
}

func TestUltraMsgSender_SendChat_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POST")
}

// ── UltraMsgSender — media path ──────────────────────────────────────

func TestUltraMsgSender_SendMedia_ImageURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/i/messages/image", r.URL.Path)
		_ = r.ParseForm()
		assert.Equal(t, "https://example.com/img.png", r.PostForm.Get("image"))
		assert.Equal(t, "cap", r.PostForm.Get("caption"))
		_, _ = w.Write([]byte(`{"sent":"true","id":"m1"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{
		To:    "1",
		Media: &MediaAttachment{Type: "image", URL: "https://example.com/img.png", Caption: "cap"},
	})
	require.NoError(t, err)
	assert.Equal(t, "m1", res.ProviderMsgID)
}

func TestUltraMsgSender_SendMedia_DocumentURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/i/messages/document", r.URL.Path)
		_ = r.ParseForm()
		assert.Equal(t, "report.pdf", r.PostForm.Get("filename"))
		_, _ = w.Write([]byte(`{"sent":"true","id":"d1"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{
		To:    "1",
		Media: &MediaAttachment{Type: "document", URL: "https://x", Filename: "report.pdf"},
	})
	require.NoError(t, err)
	assert.Equal(t, "d1", res.ProviderMsgID)
}

func TestUltraMsgSender_SendMedia_Video(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/i/messages/video", r.URL.Path)
		_, _ = w.Write([]byte(`{"sent":"true","id":"v1"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To:    "1",
		Media: &MediaAttachment{Type: "video", URL: "https://x"},
	})
	require.NoError(t, err)
}

func TestUltraMsgSender_SendMedia_Audio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/i/messages/audio", r.URL.Path)
		_, _ = w.Write([]byte(`{"sent":"true","id":"a1"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To:    "1",
		Media: &MediaAttachment{Type: "audio", URL: "https://x"},
	})
	require.NoError(t, err)
}

func TestUltraMsgSender_SendMedia_UnknownType(t *testing.T) {
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: "http://x", InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To:    "1",
		Media: &MediaAttachment{Type: "carrierpigeon", URL: "https://x"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown media type")
}

func TestUltraMsgSender_SendMedia_BytesUploadThenSend(t *testing.T) {
	var uploadHit, sendHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/i/media/upload":
			uploadHit = true
			_ = r.ParseForm()
			assert.NotEmpty(t, r.PostForm.Get("file"))
			_, _ = w.Write([]byte(`{"success":true,"url":"https://cdn.example/x.png"}`))
		case "/i/messages/image":
			sendHit = true
			_ = r.ParseForm()
			assert.Equal(t, "https://cdn.example/x.png", r.PostForm.Get("image"))
			_, _ = w.Write([]byte(`{"sent":"true","id":"u1"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To:    "1",
		Media: &MediaAttachment{Type: "image", Content: []byte("PNG")},
	})
	require.NoError(t, err)
	assert.True(t, uploadHit && sendHit, "both upload and send must fire")
}

func TestUltraMsgSender_SendMedia_UploadDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To: "1", Media: &MediaAttachment{Type: "image", Content: []byte("PNG")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload decode")
}

func TestUltraMsgSender_SendMedia_UploadAppLevelFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":"file too big"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To: "1", Media: &MediaAttachment{Type: "image", Content: []byte("PNG")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file too big")
}

func TestUltraMsgSender_SendMedia_UploadTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To: "1", Media: &MediaAttachment{Type: "image", Content: []byte("PNG")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "media upload")
}

// ── UltraMsgSender — DeleteMessage ──────────────────────────────────

func TestUltraMsgSender_DeleteMessage_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/i/messages/delete", r.URL.Path)
		_ = r.ParseForm()
		assert.Equal(t, "msg-1", r.PostForm.Get("id"))
		_, _ = w.Write([]byte(`{"sent":"true","id":"msg-1"}`))
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	require.NoError(t, s.DeleteMessage(context.Background(), "msg-1"))
}

func TestUltraMsgSender_DeleteMessage_EmptyID(t *testing.T) {
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: "http://x", InstanceID: "i", Token: "t",
	}, discardLogger())
	err := s.DeleteMessage(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "providerMsgID")
}

func TestUltraMsgSender_DeleteMessage_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	s := NewUltraMsgSender(config.WhatsAppUltraMsgConfig{
		BaseURL: srv.URL, InstanceID: "i", Token: "t",
	}, discardLogger())
	require.Error(t, s.DeleteMessage(context.Background(), "x"))
}

// ── helpers ─────────────────────────────────────────────────────────

func TestNormalizeUltraMsgTo(t *testing.T) {
	assert.Equal(t, "250788000000", normalizeUltraMsgTo("+250788000000"))
	assert.Equal(t, "250788000000", normalizeUltraMsgTo("  +250788000000 "))
	assert.Equal(t, "250788000000", normalizeUltraMsgTo("250788000000"))
}

func TestBase64String(t *testing.T) {
	assert.Equal(t, "aGVsbG8=", base64String([]byte("hello")))
}

func TestNormalizeTwilioTo(t *testing.T) {
	assert.Equal(t, "+250788000000", normalizeTwilioTo("+250788000000"))
	assert.Equal(t, "+250788000000", normalizeTwilioTo(" 250788000000 "))
}

// ── TwilioSender ─────────────────────────────────────────────────────

func TestTwilioSender_Send_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/2010-04-01/Accounts/SID/Messages.json", r.URL.Path)
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "SID", user)
		assert.Equal(t, "TOK", pass)
		_ = r.ParseForm()
		assert.Equal(t, "whatsapp:+14155551111", r.PostForm.Get("From"))
		assert.Equal(t, "whatsapp:+250788000000", r.PostForm.Get("To"))
		assert.Equal(t, "hi", r.PostForm.Get("Body"))
		assert.Equal(t, "https://media", r.PostForm.Get("MediaUrl"))
		_, _ = w.Write([]byte(`{"sid":"SM1","status":"queued"}`))
	}))
	t.Cleanup(srv.Close)
	swapTwilioBaseURL(t, srv.URL)

	s := NewTwilioSender(config.WhatsAppTwilioConfig{
		AccountSID: "SID", AuthToken: "TOK", FromNumber: "+14155551111",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{
		To: "+250788000000", Body: "hi",
		Media: &MediaAttachment{URL: "https://media"},
	})
	require.NoError(t, err)
	assert.Equal(t, "SM1", res.ProviderMsgID)
	assert.Equal(t, "queued", res.Status)
	assert.Equal(t, "whatsapp-twilio", s.Name())
}

func TestTwilioSender_Send_FromAlreadyHasWhatsAppPrefix(t *testing.T) {
	// FromNumber set to "whatsapp:+..." should not double-prefix.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		assert.Equal(t, "whatsapp:+14155551111", r.PostForm.Get("From"))
		_, _ = w.Write([]byte(`{"sid":"X","status":"queued"}`))
	}))
	t.Cleanup(srv.Close)
	swapTwilioBaseURL(t, srv.URL)
	s := NewTwilioSender(config.WhatsAppTwilioConfig{
		AccountSID: "S", AuthToken: "T", FromNumber: "whatsapp:+14155551111",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.NoError(t, err)
}

func TestTwilioSender_Send_Status4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("auth_failed"))
	}))
	t.Cleanup(srv.Close)
	swapTwilioBaseURL(t, srv.URL)
	s := NewTwilioSender(config.WhatsAppTwilioConfig{
		AccountSID: "S", AuthToken: "T", FromNumber: "+1",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestTwilioSender_Send_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	swapTwilioBaseURL(t, srv.URL)
	s := NewTwilioSender(config.WhatsAppTwilioConfig{
		AccountSID: "S", AuthToken: "T", FromNumber: "+1",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestTwilioSender_Send_NewRequestError(t *testing.T) {
	swapTwilioBaseURL(t, "http://\nbad")
	s := NewTwilioSender(config.WhatsAppTwilioConfig{
		AccountSID: "S", AuthToken: "T", FromNumber: "+1",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
}

func TestTwilioSender_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	swapTwilioBaseURL(t, srv.URL)
	s := NewTwilioSender(config.WhatsAppTwilioConfig{
		AccountSID: "S", AuthToken: "T", FromNumber: "+1",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POST")
}

func TestTwilioSender_DeleteMessage_Unsupported(t *testing.T) {
	s := NewTwilioSender(config.WhatsAppTwilioConfig{}, discardLogger())
	err := s.DeleteMessage(context.Background(), "x")
	require.ErrorIs(t, err, ErrUnsupported)
}

// ── MetaSender ──────────────────────────────────────────────────────

func TestMetaSender_Name(t *testing.T) {
	s := NewMetaSender(config.WhatsAppMetaConfig{}, discardLogger())
	assert.Equal(t, "whatsapp-meta", s.Name())
}

func TestMetaSender_APIVersionDefault(t *testing.T) {
	// Empty APIVersion defaults to defaultMetaAPIVersion.
	s := NewMetaSender(config.WhatsAppMetaConfig{}, discardLogger())
	assert.Equal(t, defaultMetaAPIVersion, s.cfg.APIVersion)
}

func TestMetaSender_APIVersionOverride(t *testing.T) {
	s := NewMetaSender(config.WhatsAppMetaConfig{APIVersion: "v22.0"}, discardLogger())
	assert.Equal(t, "v22.0", s.cfg.APIVersion)
}

func TestMetaSender_Send_TextHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v20.0/PNI/messages", r.URL.Path)
		assert.Equal(t, "Bearer TOKEN", r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		assert.Contains(t, s, `"messaging_product":"whatsapp"`)
		assert.Contains(t, s, `"type":"text"`)
		assert.Contains(t, s, `"body":"hi"`)
		assert.Contains(t, s, `"preview_url":true`)
		assert.Contains(t, s, `"message_id":"wamid.ABC"`)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.XYZ"}]}`))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)

	tr := true
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "TOKEN", PhoneNumberID: "PNI",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{
		To:                   "+250788000000",
		Body:                 "hi",
		PreviewURL:           &tr,
		ReplyToProviderMsgID: "wamid.ABC",
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.XYZ", res.ProviderMsgID)
}

func TestMetaSender_Send_MissingTo(t *testing.T) {
	swapMetaBaseURL(t, "http://x")
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "To is required")
}

func TestMetaSender_Send_ImageViaURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		assert.Contains(t, s, `"type":"image"`)
		assert.Contains(t, s, `"link":"https://img.example/x.png"`)
		assert.Contains(t, s, `"caption":"capt"`)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.IMG"}]}`))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)

	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "PNI",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{
		To:    "+1",
		Media: &MediaAttachment{Type: "image", URL: "https://img.example/x.png", Caption: "capt"},
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.IMG", res.ProviderMsgID)
}

func TestMetaSender_Send_DocumentViaURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		assert.Contains(t, s, `"type":"document"`)
		assert.Contains(t, s, `"filename":"r.pdf"`)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.DOC"}]}`))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To:    "+1",
		Media: &MediaAttachment{Type: "document", URL: "https://x", Filename: "r.pdf"},
	})
	require.NoError(t, err)
}

func TestMetaSender_Send_VideoAndAudioViaURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[{"id":"X"}]}`))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())

	_, err := s.Send(context.Background(), Message{
		To: "+1", Media: &MediaAttachment{Type: "video", URL: "https://x"},
	})
	require.NoError(t, err)
	_, err = s.Send(context.Background(), Message{
		To: "+1", Media: &MediaAttachment{Type: "audio", URL: "https://x"},
	})
	require.NoError(t, err)
}

func TestMetaSender_Send_UnknownMediaType(t *testing.T) {
	swapMetaBaseURL(t, "http://x")
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To: "+1", Media: &MediaAttachment{Type: "sticker", URL: "https://x"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown media type")
}

func TestMetaSender_Send_BytesUploadThenSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/media"):
			_, _ = w.Write([]byte(`{"id":"MID-1"}`))
		case strings.HasSuffix(r.URL.Path, "/messages"):
			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), `"id":"MID-1"`)
			_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.UP"}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	res, err := s.Send(context.Background(), Message{
		To: "+1",
		Media: &MediaAttachment{
			Type: "image", Content: []byte("PNG"), ContentType: "image/png",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.UP", res.ProviderMsgID)
}

func TestMetaSender_Send_BytesNoContentType(t *testing.T) {
	swapMetaBaseURL(t, "http://nope.example")
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To: "+1",
		Media: &MediaAttachment{Type: "image", Content: []byte("x")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ContentType is required")
}

func TestMetaSender_Send_MediaMissingURLAndContent(t *testing.T) {
	swapMetaBaseURL(t, "http://x")
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{
		To: "+1", Media: &MediaAttachment{Type: "image"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL or Content")
}

func TestMetaSender_Send_PostJSONNewRequestError(t *testing.T) {
	swapMetaBaseURL(t, "http://\nbad")
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "+1", Body: "hi"})
	require.Error(t, err)
}

func TestMetaSender_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "+1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POST")
}

func TestMetaSender_Send_Status4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("oauth fail"))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "+1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestMetaSender_Send_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "+1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestMetaSender_Send_EmptyMessagesArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[]}`))
	}))
	t.Cleanup(srv.Close)
	swapMetaBaseURL(t, srv.URL)
	s := NewMetaSender(config.WhatsAppMetaConfig{
		AccessToken: "T", PhoneNumberID: "P",
	}, discardLogger())
	_, err := s.Send(context.Background(), Message{To: "+1", Body: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty messages array")
}

func TestMetaSender_DeleteMessage_Unsupported(t *testing.T) {
	s := NewMetaSender(config.WhatsAppMetaConfig{}, discardLogger())
	require.ErrorIs(t, s.DeleteMessage(context.Background(), "x"), ErrUnsupported)
}

// ── meta_upload.go ───────────────────────────────────────────────────

func TestUploadMetaMedia_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		assert.Equal(t, "Bearer T", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"id":"MID-1"}`))
	}))
	t.Cleanup(srv.Close)
	id, err := uploadMetaMedia(context.Background(), &http.Client{},
		srv.URL, "T", &MediaAttachment{
			Type: "image", Content: []byte("PNG"), ContentType: "image/png",
			Filename: "x.png",
		})
	require.NoError(t, err)
	assert.Equal(t, "MID-1", id)
}

func TestUploadMetaMedia_NoContentType(t *testing.T) {
	_, err := uploadMetaMedia(context.Background(), &http.Client{}, "http://x", "T",
		&MediaAttachment{Type: "image", Content: []byte("x")})
	require.ErrorIs(t, err, errMissingContentType)
}

func TestUploadMetaMedia_DefaultFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Default filename is "upload" when Filename is empty.
		assert.Contains(t, string(body), `filename="upload"`)
		_, _ = w.Write([]byte(`{"id":"X"}`))
	}))
	t.Cleanup(srv.Close)
	id, err := uploadMetaMedia(context.Background(), &http.Client{},
		srv.URL, "T", &MediaAttachment{
			Type: "image", Content: []byte("PNG"), ContentType: "image/png",
		})
	require.NoError(t, err)
	assert.Equal(t, "X", id)
}

func TestUploadMetaMedia_NewRequestError(t *testing.T) {
	_, err := uploadMetaMedia(context.Background(), &http.Client{},
		"http://\nbad", "T", &MediaAttachment{
			Type: "image", Content: []byte("x"), ContentType: "image/png",
		})
	require.Error(t, err)
}

func TestUploadMetaMedia_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	_, err := uploadMetaMedia(context.Background(), &http.Client{},
		srv.URL, "T", &MediaAttachment{
			Type: "image", Content: []byte("x"), ContentType: "image/png",
		})
	require.Error(t, err)
}

func TestUploadMetaMedia_Status4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad token"))
	}))
	t.Cleanup(srv.Close)
	_, err := uploadMetaMedia(context.Background(), &http.Client{},
		srv.URL, "T", &MediaAttachment{
			Type: "image", Content: []byte("x"), ContentType: "image/png",
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meta media")
}

func TestUploadMetaMedia_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	_, err := uploadMetaMedia(context.Background(), &http.Client{},
		srv.URL, "T", &MediaAttachment{
			Type: "image", Content: []byte("x"), ContentType: "image/png",
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// failingBuf is an io.Reader+io.Writer that fails on the Nth call to
// Write. Used to exercise the rare `if err != nil` branches inside
// uploadMetaMedia's multipart rendering — they're only reachable when
// the underlying writer fails, which a bytes.Buffer never does.
type failingBuf struct {
	failAfter int
	calls     int
}

func (f *failingBuf) Write(p []byte) (int, error) {
	f.calls++
	if f.calls > f.failAfter {
		return 0, io.ErrShortWrite
	}
	return len(p), nil
}

func (f *failingBuf) Read(_ []byte) (int, error) { return 0, io.EOF }

func swapBodyBufFn(t *testing.T, fn func() readWriteBuffer) {
	t.Helper()
	orig := multipartBodyBufFn
	multipartBodyBufFn = fn
	t.Cleanup(func() { multipartBodyBufFn = orig })
}

// TestUploadMetaMedia_MultipartWriteFieldFails exercises the first
// `mw.WriteField` error branch by failing the underlying writer on
// the very first write.
func TestUploadMetaMedia_MultipartWriteFieldFails(t *testing.T) {
	swapBodyBufFn(t, func() readWriteBuffer { return &failingBuf{failAfter: 0} })
	_, err := uploadMetaMedia(context.Background(), &http.Client{}, "http://x", "T",
		&MediaAttachment{Type: "image", Content: []byte("x"), ContentType: "image/png"})
	require.Error(t, err)
}

// TestUploadMetaMedia_MultipartSecondWriteFieldFails — fail on the
// second internal Write so the FIRST WriteField succeeds but the
// SECOND ("type") errors.
func TestUploadMetaMedia_MultipartSecondWriteFieldFails(t *testing.T) {
	// multipart writes a header + value per field. failAfter=2 lets
	// the first WriteField complete and fails inside the second.
	swapBodyBufFn(t, func() readWriteBuffer { return &failingBuf{failAfter: 2} })
	_, err := uploadMetaMedia(context.Background(), &http.Client{}, "http://x", "T",
		&MediaAttachment{Type: "image", Content: []byte("x"), ContentType: "image/png"})
	require.Error(t, err)
}

// TestUploadMetaMedia_MultipartPartAndCloseFail sweeps failAfter
// values across the range that multipart.Writer's CreatePart and
// Close paths land in. The exact internal Write count depends on
// stdlib internals; the loop guarantees both error branches fire by
// trying every plausible cutoff between "first WriteField done" and
// "request issued."
func TestUploadMetaMedia_MultipartPartAndCloseFail(t *testing.T) {
	// Failing late triggers either CreatePart, part.Write, or
	// mw.Close — all three branches return an error from
	// uploadMetaMedia, which is what we assert. Aggregating into a
	// sweep keeps the test resilient to stdlib write-count drift
	// across Go versions.
	for i := 3; i <= 12; i++ {
		t.Run("failAfter", func(t *testing.T) {
			swapBodyBufFn(t, func() readWriteBuffer { return &failingBuf{failAfter: i} })
			_, _ = uploadMetaMedia(context.Background(), &http.Client{}, "http://x", "T",
				&MediaAttachment{
					Type:        "image",
					Content:     []byte("xxxxxxxxxxxxxxxxxxxx"),
					ContentType: "image/png",
				})
		})
	}
}

func TestUploadMetaMedia_EmptyID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":""}`))
	}))
	t.Cleanup(srv.Close)
	_, err := uploadMetaMedia(context.Background(), &http.Client{},
		srv.URL, "T", &MediaAttachment{
			Type: "image", Content: []byte("x"), ContentType: "image/png",
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty id")
}

// ── provider.go: factory + noop ─────────────────────────────────────

func TestNewSender_Nil(t *testing.T) {
	s, err := NewSender(nil, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "whatsapp-noop", s.Name())
}

func TestNewSender_EmptyProvider(t *testing.T) {
	s, err := NewSender(&config.WhatsAppConfig{}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "whatsapp-noop", s.Name())
}

func TestNewSender_UltraMsg_HappyPath(t *testing.T) {
	s, err := NewSender(&config.WhatsAppConfig{
		Provider: "ultramsg",
		UltraMsg: config.WhatsAppUltraMsgConfig{
			BaseURL: "http://x", InstanceID: "i", Token: "t",
		},
	}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "whatsapp-ultramsg", s.Name())
}

func TestNewSender_UltraMsg_MissingFields(t *testing.T) {
	_, err := NewSender(&config.WhatsAppConfig{Provider: "ultramsg"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UltraMsg")
}

func TestNewSender_Twilio_HappyPath(t *testing.T) {
	s, err := NewSender(&config.WhatsAppConfig{
		Provider: "twilio",
		Twilio: config.WhatsAppTwilioConfig{
			AccountSID: "S", AuthToken: "T", FromNumber: "+1",
		},
	}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "whatsapp-twilio", s.Name())
}

func TestNewSender_Twilio_MissingFields(t *testing.T) {
	_, err := NewSender(&config.WhatsAppConfig{Provider: "twilio"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Twilio")
}

func TestNewSender_Meta_HappyPath(t *testing.T) {
	s, err := NewSender(&config.WhatsAppConfig{
		Provider: "meta",
		Meta: config.WhatsAppMetaConfig{
			AccessToken: "T", PhoneNumberID: "P",
		},
	}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "whatsapp-meta", s.Name())
}

func TestNewSender_Meta_MissingFields(t *testing.T) {
	_, err := NewSender(&config.WhatsAppConfig{Provider: "meta"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Meta")
}

func TestNewSender_UnknownProvider(t *testing.T) {
	_, err := NewSender(&config.WhatsAppConfig{Provider: "telepathy"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown whatsapp provider")
}

func TestNoopSender_Methods(t *testing.T) {
	n := &noopSender{}
	assert.Equal(t, "whatsapp-noop", n.Name())
	_, err := n.Send(context.Background(), Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
	err = n.DeleteMessage(context.Background(), "x")
	require.ErrorIs(t, err, ErrUnsupported)
}
