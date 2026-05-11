package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	fcm "firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMConfig configures the Firebase Cloud Messaging provider.
//
// Credentials come from a Google service-account JSON. Pick ONE
// source — `CredentialsJSON` (the raw JSON inline in env, useful for
// containerized deployments) takes precedence over
// `CredentialsFilePath` (a file on disk).
//
// ProjectID is read from the credentials JSON when not set
// explicitly; the override exists so tests can use a fake project.
type FCMConfig struct {
	CredentialsJSON     string
	CredentialsFilePath string
	ProjectID           string
}

// fcmClient is the subset of *firebase.Messaging.Client that
// FCMSender depends on. Captured as an interface so tests can
// substitute an in-memory fake; production always passes a real
// *fcm.Client returned by app.Messaging(ctx).
type fcmClient interface {
	SendEachForMulticast(ctx context.Context, m *fcm.MulticastMessage) (*fcm.BatchResponse, error)
	Send(ctx context.Context, m *fcm.Message) (string, error)
	SubscribeToTopic(ctx context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error)
	UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) (*fcm.TopicManagementResponse, error)
}

// FCMSender implements Sender against Firebase Cloud Messaging.
//
// One client is constructed at boot and reused. The Firebase SDK
// handles HTTP/2 pooling + token refresh internally, so we don't
// need a per-request client.
type FCMSender struct {
	client fcmClient
	logger *slog.Logger
}

// firebaseNewAppFn and appMessagingFn are package-level seams over the
// firebase-admin-go SDK so tests can drive every NewFCMSender branch
// without an actual Google credential. Production assigns the real
// SDK functions; tests reassign and restore on cleanup.
var (
	firebaseNewAppFn = firebase.NewApp
	appMessagingFn   = func(app *firebase.App, ctx context.Context) (fcmClient, error) {
		return app.Messaging(ctx)
	}
)

// NewFCMSender builds the FCM client. It returns an error early if
// the credentials are unreadable or the SDK can't initialize — boot
// fails loudly rather than each push silently 500ing.
func NewFCMSender(cfg FCMConfig, logger *slog.Logger) (*FCMSender, error) {
	if logger == nil {
		logger = slog.Default()
	}
	creds, err := loadFCMCredentials(cfg)
	if err != nil {
		return nil, err
	}
	conf := &firebase.Config{ProjectID: cfg.ProjectID}
	app, err := firebaseNewAppFn(context.Background(), conf, option.WithCredentialsJSON(creds))
	if err != nil {
		return nil, fmt.Errorf("fcm: firebase.NewApp: %w", err)
	}
	client, err := appMessagingFn(app, context.Background())
	if err != nil {
		return nil, fmt.Errorf("fcm: app.Messaging: %w", err)
	}
	return &FCMSender{client: client, logger: logger}, nil
}

// Name reports the provider name embedded in logs and metrics.
func (s *FCMSender) Name() string { return "fcm" }

// SendToTokens — uses SendEachForMulticast under the hood, which
// returns one SendResponse per token. We translate that to our
// TokenResult shape and stamp ErrorCode for invalidated tokens so
// the caller can prune them from the registry.
func (s *FCMSender) SendToTokens(ctx context.Context, tokens []string, msg Message) ([]TokenResult, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	multi := &fcm.MulticastMessage{
		Tokens:       tokens,
		Data:         msg.Data,
		Notification: notificationFromMessage(msg),
		Android:      androidConfigFromMessage(msg),
		APNS:         apnsConfigFromMessage(msg),
	}
	resp, err := s.client.SendEachForMulticast(ctx, multi)
	if err != nil {
		return nil, fmt.Errorf("fcm: SendEachForMulticast: %w", err)
	}
	out := make([]TokenResult, 0, len(tokens))
	for i, r := range resp.Responses {
		// Defensive: the SDK guarantees one response per input token,
		// but bound the slice anyway in case that ever changes.
		token := ""
		if i < len(tokens) {
			token = tokens[i]
		}
		tr := TokenResult{Token: token, OK: r.Success, ProviderMsgID: r.MessageID}
		if r.Error != nil {
			tr.Error = r.Error.Error()
			tr.ErrorCode = classifyFCMError(r.Error)
		}
		out = append(out, tr)
	}
	return out, nil
}

// SendToTopic — single send against a topic name (FCM resolves the
// fan-out server-side). Per-device outcomes aren't visible.
func (s *FCMSender) SendToTopic(ctx context.Context, topic string, msg Message) (*SendResult, error) {
	if topic == "" {
		return nil, errors.New("fcm: topic is required")
	}
	m := &fcm.Message{
		Topic:        topic,
		Data:         msg.Data,
		Notification: notificationFromMessage(msg),
		Android:      androidConfigFromMessage(msg),
		APNS:         apnsConfigFromMessage(msg),
	}
	id, err := s.client.Send(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("fcm: Send to topic %q: %w", topic, err)
	}
	return &SendResult{
		OK:              true,
		ProviderMsgID:   id,
		Status:          "sent",
		RawResponseJSON: rawJSONOrEmpty(map[string]string{"messageId": id, "topic": topic}),
	}, nil
}

// SubscribeToTopic subscribes the given device tokens to a topic via
// FCM's IID API. Empty topic or empty token list is a no-op (returns
// an empty result, not an error) so callers can pass through filtered
// input without conditional plumbing.
func (s *FCMSender) SubscribeToTopic(ctx context.Context, topic string, tokens []string) (*TopicMembershipResult, error) {
	if topic == "" || len(tokens) == 0 {
		return &TopicMembershipResult{}, nil
	}
	resp, err := s.client.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		return nil, fmt.Errorf("fcm: SubscribeToTopic: %w", err)
	}
	return topicResultFromResponse(resp), nil
}

// UnsubscribeFromTopic removes the given device tokens from a topic.
// Empty inputs are treated the same as SubscribeToTopic.
func (s *FCMSender) UnsubscribeFromTopic(ctx context.Context, topic string, tokens []string) (*TopicMembershipResult, error) {
	if topic == "" || len(tokens) == 0 {
		return &TopicMembershipResult{}, nil
	}
	resp, err := s.client.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		return nil, fmt.Errorf("fcm: UnsubscribeFromTopic: %w", err)
	}
	return topicResultFromResponse(resp), nil
}

// ─── helpers ─────────────────────────────────────────────────────────

func loadFCMCredentials(cfg FCMConfig) ([]byte, error) {
	if cfg.CredentialsJSON != "" {
		return []byte(cfg.CredentialsJSON), nil
	}
	if cfg.CredentialsFilePath != "" {
		b, err := os.ReadFile(cfg.CredentialsFilePath)
		if err != nil {
			return nil, fmt.Errorf("fcm: read credentials file %q: %w", cfg.CredentialsFilePath, err)
		}
		return b, nil
	}
	return nil, errors.New("fcm: provide CredentialsJSON or CredentialsFilePath")
}

func notificationFromMessage(msg Message) *fcm.Notification {
	if msg.Title == "" && msg.Body == "" && msg.ImageURL == "" {
		return nil
	}
	return &fcm.Notification{
		Title:    msg.Title,
		Body:     msg.Body,
		ImageURL: msg.ImageURL,
	}
}

func androidConfigFromMessage(msg Message) *fcm.AndroidConfig {
	if msg.Priority == "" && msg.Sound == "" && msg.ClickAction == "" && msg.AndroidOverride == nil {
		return nil
	}
	cfg := &fcm.AndroidConfig{}
	switch msg.Priority {
	case PriorityHigh:
		cfg.Priority = "high"
	case PriorityNormal:
		cfg.Priority = "normal"
	}
	if msg.Sound != "" || msg.ClickAction != "" {
		cfg.Notification = &fcm.AndroidNotification{
			Sound:       msg.Sound,
			ClickAction: msg.ClickAction,
		}
	}
	return cfg
}

func apnsConfigFromMessage(msg Message) *fcm.APNSConfig {
	// Only populate APNS when we have something iOS-relevant. Otherwise
	// leave nil — FCM falls back to the top-level Notification + Data.
	if msg.Sound == "" && msg.Badge == nil {
		return nil
	}
	aps := &fcm.Aps{Sound: msg.Sound, Badge: msg.Badge}
	return &fcm.APNSConfig{Payload: &fcm.APNSPayload{Aps: aps}}
}

func topicResultFromResponse(resp *fcm.TopicManagementResponse) *TopicMembershipResult {
	out := &TopicMembershipResult{
		SuccessCount: resp.SuccessCount,
		FailureCount: resp.FailureCount,
	}
	for _, e := range resp.Errors {
		// fcm.ErrorInfo carries Index + Reason; collapse to one string
		// so the caller doesn't have to import the FCM types just to log.
		out.Errors = append(out.Errors, fmt.Sprintf("[%d] %s", e.Index, e.Reason))
	}
	return out
}

// SDK-predicate seams. The firebase-admin-go Is* functions only
// recognize the SDK's internal error type (`*internal.FirebaseError`),
// which is unreachable from outside the SDK module. Wrapping each
// predicate behind a package-level var lets unit tests substitute a
// stub that returns true for a synthesised error, exercising every
// branch of classifyFCMError without depending on SDK internals.
// Production assigns the real predicates; tests reassign and restore
// via t.Cleanup.
var (
	isFCMUnregisteredFn        = fcm.IsUnregistered
	isFCMInvalidArgumentFn     = fcm.IsInvalidArgument
	isFCMQuotaExceededFn       = fcm.IsQuotaExceeded
	isFCMSenderIDMismatchFn    = fcm.IsSenderIDMismatch
	isFCMThirdPartyAuthErrorFn = fcm.IsThirdPartyAuthError
	isFCMUnavailableFn         = fcm.IsUnavailable
	isFCMInternalFn            = fcm.IsInternal
)

// classifyFCMError maps the SDK's typed error to a machine-readable
// code we can stash in TokenResult.ErrorCode. The full mapping
// reference is at:
//
//	https://firebase.google.com/docs/reference/fcm/rest/v1/ErrorCode
//
// We surface the codes the registry-pruner cares about; everything
// else falls through as "UNKNOWN" with the original message preserved
// in Error.
func classifyFCMError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case isFCMUnregisteredFn(err):
		// IsUnregistered subsumes the legacy IsRegistrationTokenNotRegistered
		// per the firebase-admin-go deprecation note.
		return "UNREGISTERED"
	case isFCMInvalidArgumentFn(err):
		return "INVALID_ARGUMENT"
	case isFCMQuotaExceededFn(err):
		return "QUOTA_EXCEEDED"
	case isFCMSenderIDMismatchFn(err):
		return "SENDER_ID_MISMATCH"
	case isFCMThirdPartyAuthErrorFn(err):
		return "THIRD_PARTY_AUTH_ERROR"
	case isFCMUnavailableFn(err):
		return "UNAVAILABLE"
	case isFCMInternalFn(err):
		return "INTERNAL"
	}
	// Fall back to a sniff of the message — useful when the SDK
	// version doesn't yet have a typed predicate for a new error.
	msg := strings.ToUpper(err.Error())
	switch {
	case strings.Contains(msg, "NOT_FOUND"), strings.Contains(msg, "NOT REGISTERED"):
		return "UNREGISTERED"
	case strings.Contains(msg, "INVALID_ARGUMENT"):
		return "INVALID_ARGUMENT"
	}
	return "UNKNOWN"
}

func rawJSONOrEmpty(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
