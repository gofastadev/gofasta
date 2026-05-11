package push

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	firebase "firebase.google.com/go/v4"
	fcm "firebase.google.com/go/v4/messaging"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// ─────────────────────────────────────────────────────────────────────
// Test infrastructure for pkg/push.
//
// The Firebase Admin SDK does authenticated HTTP from the moment any
// Send-style method is called, which makes black-box testing brittle.
// We test it black-box via two seams:
//
//   - fcmClient interface — FCMSender stores an interface so the four
//     methods we call (SendEachForMulticast, Send, SubscribeToTopic,
//     UnsubscribeFromTopic) can be replaced with an in-memory fake.
//
//   - firebaseNewAppFn / appMessagingFn — package vars wrapping the
//     SDK constructors so NewFCMSender's "SDK init failed" branches
//     can be exercised by swapping them to stubs.
//
// Every test that mutates the seams uses t.Cleanup to restore the
// originals, so test ordering doesn't matter.
// ─────────────────────────────────────────────────────────────────────

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeFCMClient is the in-memory fcmClient implementation used by the
// FCMSender tests. The Fn fields are nil by default; tests assign
// only the ones they need.
type fakeFCMClient struct {
	sendMulticastFn func(ctx context.Context, m *fcm.MulticastMessage) (*fcm.BatchResponse, error)
	sendFn          func(ctx context.Context, m *fcm.Message) (string, error)
	subscribeFn     func(ctx context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error)
	unsubscribeFn   func(ctx context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error)
}

func (f *fakeFCMClient) SendEachForMulticast(ctx context.Context, m *fcm.MulticastMessage) (*fcm.BatchResponse, error) {
	return f.sendMulticastFn(ctx, m)
}

func (f *fakeFCMClient) Send(ctx context.Context, m *fcm.Message) (string, error) {
	return f.sendFn(ctx, m)
}

func (f *fakeFCMClient) SubscribeToTopic(ctx context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error) {
	return f.subscribeFn(ctx, tokens, topic)
}

func (f *fakeFCMClient) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error) {
	return f.unsubscribeFn(ctx, tokens, topic)
}

func swapNewAppFn(t *testing.T, fn func(ctx context.Context, c *firebase.Config, opts ...option.ClientOption) (*firebase.App, error)) {
	t.Helper()
	orig := firebaseNewAppFn
	firebaseNewAppFn = fn
	t.Cleanup(func() { firebaseNewAppFn = orig })
}

func swapMessagingFn(t *testing.T, fn func(app *firebase.App, ctx context.Context) (fcmClient, error)) {
	t.Helper()
	orig := appMessagingFn
	appMessagingFn = fn
	t.Cleanup(func() { appMessagingFn = orig })
}

// A minimum-shape service-account JSON the firebase-admin SDK accepts
// at construction time. firebase.NewApp + app.Messaging only verify
// structural validity at this point; real authentication happens on
// the first network call (which our tests never make).
const fakeServiceAccountJSON = `{
  "type": "service_account",
  "project_id": "fake-project",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBg=\n-----END PRIVATE KEY-----\n",
  "client_email": "fake@fake-project.iam.gserviceaccount.com",
  "client_id": "0",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": ""
}`

// ── loadFCMCredentials ────────────────────────────────────────────────

func TestLoadFCMCredentials_InlineJSONWins(t *testing.T) {
	cfg := FCMConfig{CredentialsJSON: "INLINE", CredentialsFilePath: "/nope"}
	got, err := loadFCMCredentials(cfg)
	require.NoError(t, err)
	assert.Equal(t, []byte("INLINE"), got)
}

func TestLoadFCMCredentials_FromFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sa.json")
	require.NoError(t, os.WriteFile(p, []byte("FROMFILE"), 0o600))
	got, err := loadFCMCredentials(FCMConfig{CredentialsFilePath: p})
	require.NoError(t, err)
	assert.Equal(t, []byte("FROMFILE"), got)
}

func TestLoadFCMCredentials_FileMissing(t *testing.T) {
	_, err := loadFCMCredentials(FCMConfig{CredentialsFilePath: "/does/not/exist"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read credentials file")
}

func TestLoadFCMCredentials_NeitherSet(t *testing.T) {
	_, err := loadFCMCredentials(FCMConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide CredentialsJSON or CredentialsFilePath")
}

// ── NewFCMSender ──────────────────────────────────────────────────────

func TestNewFCMSender_NilLoggerDefaults(t *testing.T) {
	// The SDK path is mocked out, so this verifies only the logger
	// default-assignment branch executes.
	swapNewAppFn(t, func(_ context.Context, _ *firebase.Config, _ ...option.ClientOption) (*firebase.App, error) {
		return &firebase.App{}, nil
	})
	swapMessagingFn(t, func(_ *firebase.App, _ context.Context) (fcmClient, error) {
		return &fakeFCMClient{}, nil
	})
	s, err := NewFCMSender(FCMConfig{CredentialsJSON: "x"}, nil)
	require.NoError(t, err)
	assert.NotNil(t, s.logger)
}

func TestNewFCMSender_CredentialLoadFails(t *testing.T) {
	_, err := NewFCMSender(FCMConfig{}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CredentialsJSON")
}

func TestNewFCMSender_NewAppFails(t *testing.T) {
	swapNewAppFn(t, func(_ context.Context, _ *firebase.Config, _ ...option.ClientOption) (*firebase.App, error) {
		return nil, errors.New("boom")
	})
	_, err := NewFCMSender(FCMConfig{CredentialsJSON: "x"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "firebase.NewApp")
}

func TestNewFCMSender_MessagingFails(t *testing.T) {
	swapNewAppFn(t, func(_ context.Context, _ *firebase.Config, _ ...option.ClientOption) (*firebase.App, error) {
		return &firebase.App{}, nil
	})
	swapMessagingFn(t, func(_ *firebase.App, _ context.Context) (fcmClient, error) {
		return nil, errors.New("messaging-boom")
	})
	_, err := NewFCMSender(FCMConfig{CredentialsJSON: "x"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.Messaging")
}

func TestNewFCMSender_HappyPath(t *testing.T) {
	swapNewAppFn(t, func(_ context.Context, _ *firebase.Config, _ ...option.ClientOption) (*firebase.App, error) {
		return &firebase.App{}, nil
	})
	swapMessagingFn(t, func(_ *firebase.App, _ context.Context) (fcmClient, error) {
		return &fakeFCMClient{}, nil
	})
	s, err := NewFCMSender(FCMConfig{CredentialsJSON: fakeServiceAccountJSON, ProjectID: "p"}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "fcm", s.Name())
}

// TestNewFCMSender_RealSDKConstruction exercises the production
// appMessagingFn default — the closure that calls app.Messaging(ctx).
// We feed a structurally-valid service-account JSON so firebase.NewApp
// succeeds, then let the unmocked app.Messaging build the real
// *fcm.Client (which does not authenticate until first network call,
// so this stays a unit test). This is the only test that does NOT
// swap appMessagingFn.
func TestNewFCMSender_RealSDKConstruction(t *testing.T) {
	s, err := NewFCMSender(FCMConfig{
		CredentialsJSON: fakeServiceAccountJSON,
		ProjectID:       "fake",
	}, discardLogger())
	require.NoError(t, err)
	assert.NotNil(t, s.client)
}

// ── FCMSender.SendToTokens ───────────────────────────────────────────

func TestFCMSender_SendToTokens_Empty(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{}}
	got, err := s.SendToTokens(context.Background(), nil, Message{})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFCMSender_SendToTokens_HappyPath(t *testing.T) {
	fakeClient := &fakeFCMClient{
		sendMulticastFn: func(_ context.Context, m *fcm.MulticastMessage) (*fcm.BatchResponse, error) {
			assert.Equal(t, []string{"t1", "t2"}, m.Tokens)
			assert.NotNil(t, m.Notification)
			assert.NotNil(t, m.Android)
			assert.NotNil(t, m.APNS)
			return &fcm.BatchResponse{
				Responses: []*fcm.SendResponse{
					{Success: true, MessageID: "m1"},
					{Success: false, Error: errors.New("registration-token-not-registered")},
				},
			}, nil
		},
	}
	s := &FCMSender{client: fakeClient}
	badge := 5
	results, err := s.SendToTokens(context.Background(), []string{"t1", "t2"}, Message{
		Title:    "T",
		Body:     "B",
		Sound:    "default",
		Badge:    &badge,
		Priority: PriorityHigh,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.True(t, results[0].OK)
	assert.Equal(t, "m1", results[0].ProviderMsgID)
	assert.False(t, results[1].OK)
	assert.NotEmpty(t, results[1].Error)
	assert.NotEmpty(t, results[1].ErrorCode)
}

func TestFCMSender_SendToTokens_BatchError(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		sendMulticastFn: func(_ context.Context, _ *fcm.MulticastMessage) (*fcm.BatchResponse, error) {
			return nil, errors.New("network")
		},
	}}
	_, err := s.SendToTokens(context.Background(), []string{"t1"}, Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SendEachForMulticast")
}

// TestFCMSender_SendToTokens_ResponseShorterThanTokens — defensive
// branch where the SDK returns fewer responses than tokens; the token
// field falls back to "".
func TestFCMSender_SendToTokens_ResponseShorterThanTokens(t *testing.T) {
	// We can't easily construct a longer responses slice than tokens,
	// but the for-range index is bound by len(resp.Responses), so the
	// `i < len(tokens)` guard is dead code on the happy path. To
	// exercise it we use the dual case: SDK returns MORE responses
	// than tokens.
	s := &FCMSender{client: &fakeFCMClient{
		sendMulticastFn: func(_ context.Context, _ *fcm.MulticastMessage) (*fcm.BatchResponse, error) {
			return &fcm.BatchResponse{
				Responses: []*fcm.SendResponse{
					{Success: true, MessageID: "m1"},
					{Success: true, MessageID: "m2"},
				},
			}, nil
		},
	}}
	results, err := s.SendToTokens(context.Background(), []string{"t1"}, Message{})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "t1", results[0].Token)
	assert.Equal(t, "", results[1].Token, "second response has no matching token slot")
}

// ── FCMSender.SendToTopic ─────────────────────────────────────────────

func TestFCMSender_SendToTopic_EmptyTopic(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{}}
	_, err := s.SendToTopic(context.Background(), "", Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
}

func TestFCMSender_SendToTopic_HappyPath(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		sendFn: func(_ context.Context, m *fcm.Message) (string, error) {
			assert.Equal(t, "weather", m.Topic)
			return "msg-id", nil
		},
	}}
	res, err := s.SendToTopic(context.Background(), "weather", Message{Title: "rain"})
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "msg-id", res.ProviderMsgID)
	assert.Equal(t, "sent", res.Status)
	assert.Contains(t, res.RawResponseJSON, `"messageId":"msg-id"`)
}

func TestFCMSender_SendToTopic_SDKError(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		sendFn: func(_ context.Context, _ *fcm.Message) (string, error) {
			return "", errors.New("quota")
		},
	}}
	_, err := s.SendToTopic(context.Background(), "t", Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Send to topic")
}

// ── FCMSender.SubscribeToTopic / UnsubscribeFromTopic ────────────────

func TestFCMSender_SubscribeToTopic_EmptyInputs(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{}}
	res, err := s.SubscribeToTopic(context.Background(), "", []string{"t"})
	require.NoError(t, err)
	assert.NotNil(t, res)

	res, err = s.SubscribeToTopic(context.Background(), "topic", nil)
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestFCMSender_SubscribeToTopic_HappyPath(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		subscribeFn: func(_ context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error) {
			assert.Equal(t, "weather", topic)
			assert.Equal(t, []string{"t1"}, tokens)
			return &fcm.TopicManagementResponse{
				SuccessCount: 1,
				FailureCount: 0,
			}, nil
		},
	}}
	res, err := s.SubscribeToTopic(context.Background(), "weather", []string{"t1"})
	require.NoError(t, err)
	assert.Equal(t, 1, res.SuccessCount)
}

func TestFCMSender_SubscribeToTopic_SDKError(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		subscribeFn: func(_ context.Context, _ []string, _ string) (*fcm.TopicManagementResponse, error) {
			return nil, errors.New("boom")
		},
	}}
	_, err := s.SubscribeToTopic(context.Background(), "t", []string{"x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SubscribeToTopic")
}

func TestFCMSender_UnsubscribeFromTopic_EmptyInputs(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{}}
	res, err := s.UnsubscribeFromTopic(context.Background(), "", []string{"t"})
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestFCMSender_UnsubscribeFromTopic_HappyPath(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		unsubscribeFn: func(_ context.Context, _ []string, _ string) (*fcm.TopicManagementResponse, error) {
			return &fcm.TopicManagementResponse{SuccessCount: 2}, nil
		},
	}}
	res, err := s.UnsubscribeFromTopic(context.Background(), "t", []string{"x", "y"})
	require.NoError(t, err)
	assert.Equal(t, 2, res.SuccessCount)
}

func TestFCMSender_UnsubscribeFromTopic_SDKError(t *testing.T) {
	s := &FCMSender{client: &fakeFCMClient{
		unsubscribeFn: func(_ context.Context, _ []string, _ string) (*fcm.TopicManagementResponse, error) {
			return nil, errors.New("boom")
		},
	}}
	_, err := s.UnsubscribeFromTopic(context.Background(), "t", []string{"x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UnsubscribeFromTopic")
}

// ── helpers ─────────────────────────────────────────────────────────

func TestNotificationFromMessage_Empty(t *testing.T) {
	assert.Nil(t, notificationFromMessage(Message{}))
}

func TestNotificationFromMessage_NonEmpty(t *testing.T) {
	n := notificationFromMessage(Message{Title: "T", Body: "B", ImageURL: "u"})
	require.NotNil(t, n)
	assert.Equal(t, "T", n.Title)
	assert.Equal(t, "B", n.Body)
	assert.Equal(t, "u", n.ImageURL)
}

func TestAndroidConfigFromMessage_Empty(t *testing.T) {
	assert.Nil(t, androidConfigFromMessage(Message{}))
}

func TestAndroidConfigFromMessage_HighPriority(t *testing.T) {
	c := androidConfigFromMessage(Message{Priority: PriorityHigh})
	require.NotNil(t, c)
	assert.Equal(t, "high", c.Priority)
}

func TestAndroidConfigFromMessage_NormalPriority(t *testing.T) {
	c := androidConfigFromMessage(Message{Priority: PriorityNormal})
	require.NotNil(t, c)
	assert.Equal(t, "normal", c.Priority)
}

func TestAndroidConfigFromMessage_SoundAndClick(t *testing.T) {
	c := androidConfigFromMessage(Message{Sound: "default", ClickAction: "OPEN"})
	require.NotNil(t, c)
	require.NotNil(t, c.Notification)
	assert.Equal(t, "default", c.Notification.Sound)
	assert.Equal(t, "OPEN", c.Notification.ClickAction)
}

func TestAndroidConfigFromMessage_OverrideOnly(t *testing.T) {
	// AndroidOverride alone is enough to populate the config.
	c := androidConfigFromMessage(Message{AndroidOverride: map[string]any{"k": "v"}})
	require.NotNil(t, c)
	// No priority/sound/click → Notification remains nil
	assert.Nil(t, c.Notification)
}

func TestAPNSConfigFromMessage_Empty(t *testing.T) {
	assert.Nil(t, apnsConfigFromMessage(Message{}))
}

func TestAPNSConfigFromMessage_WithSoundAndBadge(t *testing.T) {
	badge := 7
	c := apnsConfigFromMessage(Message{Sound: "default", Badge: &badge})
	require.NotNil(t, c)
	require.NotNil(t, c.Payload)
	require.NotNil(t, c.Payload.Aps)
	assert.Equal(t, "default", c.Payload.Aps.Sound)
	require.NotNil(t, c.Payload.Aps.Badge)
	assert.Equal(t, 7, *c.Payload.Aps.Badge)
}

func TestTopicResultFromResponse(t *testing.T) {
	resp := &fcm.TopicManagementResponse{
		SuccessCount: 3,
		FailureCount: 2,
		Errors: []*fcm.ErrorInfo{
			{Index: 1, Reason: "INVALID_ARGUMENT"},
			{Index: 4, Reason: "NOT_FOUND"},
		},
	}
	out := topicResultFromResponse(resp)
	assert.Equal(t, 3, out.SuccessCount)
	assert.Equal(t, 2, out.FailureCount)
	require.Len(t, out.Errors, 2)
	assert.Contains(t, out.Errors[0], "INVALID_ARGUMENT")
	assert.Contains(t, out.Errors[1], "NOT_FOUND")
}

func TestRawJSONOrEmpty_Marshalable(t *testing.T) {
	assert.JSONEq(t, `{"a":1}`, rawJSONOrEmpty(map[string]int{"a": 1}))
}

func TestRawJSONOrEmpty_Unmarshalable(t *testing.T) {
	// channels can't be marshaled; json.Marshal returns an error.
	assert.Equal(t, "", rawJSONOrEmpty(make(chan int)))
}

// ── classifyFCMError ─────────────────────────────────────────────────

func TestClassifyFCMError_Nil(t *testing.T) {
	assert.Equal(t, "", classifyFCMError(nil))
}

// For the SDK-predicate branches we use the error strings that the
// firebase-admin-go predicates check for; the predicates are based on
// the wrapped error's code string, accessible via the format the
// SDK uses internally. We bypass the typed-predicate branches via
// the message-sniff fallback, which still gives us the canonical
// classification — and exercises the fallback path the predicates
// were designed to backstop.

func TestClassifyFCMError_MessageNotFound(t *testing.T) {
	assert.Equal(t, "UNREGISTERED", classifyFCMError(errors.New("registration token not_found")))
}

func TestClassifyFCMError_MessageNotRegistered(t *testing.T) {
	assert.Equal(t, "UNREGISTERED", classifyFCMError(errors.New("token not registered with FCM")))
}

func TestClassifyFCMError_MessageInvalidArgument(t *testing.T) {
	assert.Equal(t, "INVALID_ARGUMENT", classifyFCMError(errors.New("invalid_argument: bad data")))
}

func TestClassifyFCMError_Unknown(t *testing.T) {
	assert.Equal(t, "UNKNOWN", classifyFCMError(errors.New("totally novel error")))
}

// TestClassifyFCMError_PredicateBranches drives each typed-predicate
// branch of classifyFCMError. The SDK's Is* functions recognize only
// `*internal.FirebaseError`, which is unreachable from outside the
// SDK module — so we flip the package-level predicate seams to
// return true for a sentinel error, exercising every branch.
func TestClassifyFCMError_PredicateBranches(t *testing.T) {
	sentinel := errors.New("sentinel")

	cases := []struct {
		name string
		swap func(t *testing.T)
		want string
	}{
		{
			name: "Unregistered",
			swap: func(t *testing.T) {
				orig := isFCMUnregisteredFn
				isFCMUnregisteredFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMUnregisteredFn = orig })
			},
			want: "UNREGISTERED",
		},
		{
			name: "InvalidArgument",
			swap: func(t *testing.T) {
				orig := isFCMInvalidArgumentFn
				isFCMInvalidArgumentFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMInvalidArgumentFn = orig })
			},
			want: "INVALID_ARGUMENT",
		},
		{
			name: "QuotaExceeded",
			swap: func(t *testing.T) {
				orig := isFCMQuotaExceededFn
				isFCMQuotaExceededFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMQuotaExceededFn = orig })
			},
			want: "QUOTA_EXCEEDED",
		},
		{
			name: "SenderIDMismatch",
			swap: func(t *testing.T) {
				orig := isFCMSenderIDMismatchFn
				isFCMSenderIDMismatchFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMSenderIDMismatchFn = orig })
			},
			want: "SENDER_ID_MISMATCH",
		},
		{
			name: "ThirdPartyAuthError",
			swap: func(t *testing.T) {
				orig := isFCMThirdPartyAuthErrorFn
				isFCMThirdPartyAuthErrorFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMThirdPartyAuthErrorFn = orig })
			},
			want: "THIRD_PARTY_AUTH_ERROR",
		},
		{
			name: "Unavailable",
			swap: func(t *testing.T) {
				orig := isFCMUnavailableFn
				isFCMUnavailableFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMUnavailableFn = orig })
			},
			want: "UNAVAILABLE",
		},
		{
			name: "Internal",
			swap: func(t *testing.T) {
				orig := isFCMInternalFn
				isFCMInternalFn = func(_ error) bool { return true }
				t.Cleanup(func() { isFCMInternalFn = orig })
			},
			want: "INTERNAL",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.swap(t)
			assert.Equal(t, tc.want, classifyFCMError(sentinel))
		})
	}
}

// ── factory.NewSender ────────────────────────────────────────────────

func TestNewSender_NilConfig(t *testing.T) {
	s, err := NewSender(nil, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "noop", s.Name())
}

func TestNewSender_EmptyProvider(t *testing.T) {
	s, err := NewSender(&config.PushConfig{}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "noop", s.Name())
}

func TestNewSender_FCMMissingCredentials(t *testing.T) {
	_, err := NewSender(&config.PushConfig{Provider: "fcm"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FCM.CredentialsJSON")
}

func TestNewSender_FCMHappyPath(t *testing.T) {
	swapNewAppFn(t, func(_ context.Context, _ *firebase.Config, _ ...option.ClientOption) (*firebase.App, error) {
		return &firebase.App{}, nil
	})
	swapMessagingFn(t, func(_ *firebase.App, _ context.Context) (fcmClient, error) {
		return &fakeFCMClient{}, nil
	})
	s, err := NewSender(&config.PushConfig{
		Provider: "fcm",
		FCM: config.PushFCMConfig{
			CredentialsJSON: fakeServiceAccountJSON,
			ProjectID:       "fake",
		},
	}, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, "fcm", s.Name())
}

func TestNewSender_UnknownProvider(t *testing.T) {
	_, err := NewSender(&config.PushConfig{Provider: "telepathy"}, discardLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

// ── noopSender ───────────────────────────────────────────────────────

func TestNoopSender_AllMethods(t *testing.T) {
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	n := &noopSender{logger: logger}

	assert.Equal(t, "noop", n.Name())

	_, err := n.SendToTokens(context.Background(), []string{"a", "b"}, Message{Title: "hi", Priority: PriorityHigh})
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Contains(t, logBuf.String(), "send-to-tokens dropped")

	_, err = n.SendToTopic(context.Background(), "x", Message{Title: "hi"})
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Contains(t, logBuf.String(), "send-to-topic dropped")

	_, err = n.SubscribeToTopic(context.Background(), "x", []string{"y"})
	require.ErrorIs(t, err, ErrUnsupported)

	_, err = n.UnsubscribeFromTopic(context.Background(), "x", []string{"y"})
	require.ErrorIs(t, err, ErrUnsupported)
}

func TestNoopSender_NilLogger(t *testing.T) {
	// Exercises the `if s.logger != nil` short-circuit in
	// SendToTokens and SendToTopic.
	n := &noopSender{}
	_, err := n.SendToTokens(context.Background(), []string{"t"}, Message{})
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = n.SendToTopic(context.Background(), "t", Message{})
	require.ErrorIs(t, err, ErrNotConfigured)
}

// ── TokenResult.IsTokenInvalidated ──────────────────────────────────

func TestTokenResult_IsTokenInvalidated(t *testing.T) {
	for _, code := range []string{"UNREGISTERED", "INVALID_ARGUMENT", "NOT_REGISTERED"} {
		assert.True(t, TokenResult{ErrorCode: code}.IsTokenInvalidated(), code)
	}
	assert.False(t, TokenResult{}.IsTokenInvalidated())
	assert.False(t, TokenResult{ErrorCode: "UNKNOWN"}.IsTokenInvalidated())
}
