// Package push provides outbound mobile push notification primitives.
// It is the mobile counterpart to pkg/mailer, pkg/slack and pkg/whatsapp.
//
// One provider ships in the standard build:
//
//   - "fcm" — Firebase Cloud Messaging via the official Admin SDK
//     (firebase.google.com/go/v4). Targets one or more device tokens
//     (sendEachForMulticast under the hood) or a topic. Service-account
//     JSON credentials are loaded from a file path or inline JSON in
//     the PushConfig.
//
// All providers implement the same Sender interface. Switching is a
// config-only change. To add another (Expo, OneSignal, APNs-direct,
// AWS SNS, etc.):
//
//  1. Drop a `<provider>.go` file implementing Sender into this
//     package. Mirror the structure of fcm.go.
//  2. Add a sub-config struct in pkg/config/config.go and reference it
//     from PushConfig.
//  3. Add a switch case in factory.go.
//
// Inbound webhooks (delivery receipts, click events, token-refresh
// events from APNs) are intentionally NOT in this package. Each
// provider has its own callback shape and signing scheme; the
// consumer should own that route directly.
package push

import "context"

// Sender is implemented by every push provider. Inject via DI; the
// concrete sender is selected at boot by config.
//
// Why three send verbs instead of one polymorphic Send? Mobile push
// providers strictly distinguish *targeting modes*:
//
//   - Tokens — addressed, multicast. The provider returns one
//     per-token result so the caller can prune dead tokens.
//   - Topic  — fan-out by subscription. The provider returns one
//     aggregate result; per-recipient outcomes are not visible.
//
// Subscribe / Unsubscribe round out the topic story. Providers that
// don't support a verb return ErrUnsupported so callers can log and
// move on rather than treat it as a hard failure.
type Sender interface {
	Name() string

	// SendToTokens delivers `msg` to every token in `tokens`. The
	// returned slice has one entry per input token, in input order.
	// A non-nil error means the provider call itself failed (network,
	// auth) — per-token failures show up inside TokenResult.Error.
	SendToTokens(ctx context.Context, tokens []string, msg Message) ([]TokenResult, error)

	// SendToTopic delivers `msg` to a single topic. Provider fans out
	// to every device subscribed to the topic. There is no per-device
	// outcome — the provider returns one message id.
	SendToTopic(ctx context.Context, topic string, msg Message) (*SendResult, error)

	// SubscribeToTopic adds the given device tokens to a topic on the
	// provider side. Implementations that don't expose this should
	// return ErrUnsupported (clients then need to subscribe via the
	// mobile SDK directly).
	SubscribeToTopic(ctx context.Context, topic string, tokens []string) (*TopicMembershipResult, error)

	// UnsubscribeFromTopic — inverse of SubscribeToTopic.
	UnsubscribeFromTopic(ctx context.Context, topic string, tokens []string) (*TopicMembershipResult, error)
}

// Priority is the message-priority hint surfaced to the platform-
// specific push gateway. Providers map this to their own enum:
//   - FCM Android: PriorityNormal → "normal", PriorityHigh → "high"
//   - APNs (via FCM): PriorityNormal → "5", PriorityHigh → "10"
//
// PriorityHigh wakes the app immediately and shows the notification;
// PriorityNormal can be deferred by the OS to save battery.
type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

// Message is the outbound payload. Title and Body are the visible
// notification content; Data is structured payload the app reads on
// tap (e.g. {"screen": "shipment", "id": "abc"}). Provider-specific
// overrides go on the *Override fields when the caller needs to bend
// the platform default.
//
// Data values are constrained to strings — that's the FCM data-message
// contract, and Apple/Expo follow similar shapes. Encode complex
// values as JSON if needed.
type Message struct {
	Title    string
	Body     string
	Data     map[string]string
	Priority Priority

	// ImageURL is shown in the rich notification on supported OSes
	// (Android 5+, iOS 10+). Optional.
	ImageURL string

	// Sound — APNs and FCM both honor this; "default" is the safe
	// choice. Empty = silent.
	Sound string

	// Badge sets the iOS badge count. Nil = leave unchanged.
	Badge *int

	// ClickAction — Android only. Names an intent the app handles
	// when the user taps the notification.
	ClickAction string

	// Android / Apple / WebPush escape hatches. Each provider casts to
	// its own shape; types stay map[string]any to avoid pulling
	// firebase types into pkg/push consumers. Most senders won't need
	// these.
	AndroidOverride map[string]any
	AppleOverride   map[string]any
	WebPushOverride map[string]any
}

// SendResult is the provider-side acknowledgment of one accepted
// send. Status reflects what the provider returned at SEND time, NOT
// delivery confirmation (which arrives later via webhook).
type SendResult struct {
	OK              bool
	ProviderMsgID   string
	Status          string
	RawResponseJSON string
}

// TokenResult is the per-token outcome of a multicast send. Tokens
// that the provider rejected (UNREGISTERED, INVALID_ARGUMENT, etc.)
// surface as `Error` strings — callers should treat UNREGISTERED as
// a signal to revoke the token from the registry.
type TokenResult struct {
	Token         string
	OK            bool
	ProviderMsgID string
	Error         string

	// ErrorCode is the provider's machine-readable error code
	// ("UNREGISTERED", "INVALID_ARGUMENT", "QUOTA_EXCEEDED", …) when
	// available. Empty for ok results or for providers that don't
	// surface a code.
	ErrorCode string
}

// IsTokenInvalidated returns true if the per-token error indicates
// the token should be removed from the registry. Use this to drive
// automatic cleanup after a multicast.
func (r TokenResult) IsTokenInvalidated() bool {
	switch r.ErrorCode {
	case "UNREGISTERED", "INVALID_ARGUMENT", "NOT_REGISTERED":
		return true
	}
	return false
}

// TopicMembershipResult is the aggregate outcome of a topic
// (un)subscribe call. Providers usually return per-token success/
// failure; we collapse to two counters because the only common
// caller use case is "did the bulk op succeed".
type TopicMembershipResult struct {
	SuccessCount int
	FailureCount int
	Errors       []string // human-readable per-token failures (provider-specific)
}
