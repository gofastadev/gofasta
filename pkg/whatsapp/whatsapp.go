// Package whatsapp provides outbound WhatsApp messaging primitives. It
// is the chat counterpart to pkg/mailer and pkg/slack.
//
// Three providers ship in the standard build:
//
//   - "ultramsg" — third-party UltraMsg instance API. Simple
//     form-encoded POSTs against an instance-scoped base URL. Good for
//     dev / small deployments where the operator already runs a paid
//     UltraMsg instance.
//
//   - "twilio" — Twilio Programmable Messaging WhatsApp. HTTP Basic
//     auth (Account SID + Auth Token), addresses formatted as
//     `whatsapp:+E164`, sender numbers must be approved in the Twilio
//     console. Production-grade.
//
//   - "meta" — Meta WhatsApp Cloud API (Graph). Direct from Meta;
//     requires an approved WhatsApp Business Account (WABA) and a
//     phone-number-id. Bearer auth, JSON payloads, supports template
//     messages and interactive components.
//
// All three implement the same Sender interface. Switching providers
// is a config-only change. To add a fourth (MessageBird, Vonage,
// 360dialog, Bird, GreenAPI, etc.): write one new file
// implementing Sender, register it in NewSender, ship.
//
// Inbound webhook handlers (for incoming messages, delivery receipts,
// read receipts) are intentionally NOT in this package. Each provider
// has its own callback shape and signing scheme; the consumer should
// own that route directly.
package whatsapp

import "context"

// Sender is implemented by every WhatsApp provider. Inject via DI; the
// concrete sender is selected at boot by config.
type Sender interface {
	Name() string
	Send(ctx context.Context, msg Message) (*SendResult, error)
	// DeleteMessage is best-effort — only some providers (UltraMsg,
	// Meta) expose it. Twilio does not. Implementations that don't
	// support delete should return ErrUnsupported so callers can log
	// and move on.
	DeleteMessage(ctx context.Context, providerMsgID string) error
}

// Message is the outbound payload. Construction style is intentional:
// To and Body are required; everything else is optional and tagged
// with the provider features it lights up.
//
// To is an E.164 phone number (e.g. "+250788123456"). The provider
// adapts to its own wire format — Twilio's "whatsapp:+250..." and
// UltraMsg's bare "250..." are computed inside the impl.
type Message struct {
	To   string
	Body string

	// Media is an optional attachment. When set, providers that
	// support it (all three default providers) attach it to the
	// outgoing message. Unsupported providers fall back to sending
	// just the body and surfacing a debug-level log.
	Media *MediaAttachment

	// ReplyToProviderMsgID — quote-reply / threaded reply. Provider-
	// specific IDs:
	//   ultramsg: the response's `id` field
	//   twilio:   the SID of the original message
	//   meta:     the `wamid.…` returned by /messages
	ReplyToProviderMsgID string

	// PreviewURL toggles link unfurls. nil = provider default.
	// Only Meta and Twilio support it; UltraMsg ignores.
	PreviewURL *bool
}

// MediaAttachment is one image/document/video/audio file. Two delivery
// modes:
//   - URL — provider downloads the asset from a public URL. Fast, no
//     upload step. The URL must be reachable from the provider's
//     network and stable for at least the duration of the call.
//   - Content — raw bytes. The provider handles the upload (Meta:
//     POST /media; Twilio: pre-uploaded MediaUrl. UltraMsg: a separate
//     media-upload endpoint then attaches the returned URL).
//
// At least ONE of URL or Content must be set; if both are present,
// providers prefer URL (cheaper).
type MediaAttachment struct {
	Type        string // image | document | video | audio | sticker
	URL         string
	Content     []byte
	Filename    string // required for documents; ignored for image/video
	ContentType string // MIME type; required when sending via Content
	Caption     string // image/video/document caption
}

// SendResult identifies the delivered message so the caller can store
// its id for threading or later edits/deletes. Status reflects the
// provider's reported state at SEND time — not delivery confirmation
// (which arrives via webhook later).
type SendResult struct {
	OK              bool
	ProviderMsgID   string
	Status          string
	RawResponseJSON string // entire response body for debugging / support tickets
}
