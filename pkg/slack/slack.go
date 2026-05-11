// Package slack provides outbound Slack messaging primitives. It is the
// counterpart to pkg/mailer for chat: a SlackSender interface and one
// or more concrete implementations selected by configuration.
//
// Two delivery modes are supported in the standard build:
//
//   - "webhook" — the simplest path. A single incoming-webhook URL is
//     configured per app; PostMessage POSTs the payload to that URL.
//     Channel/icon/username are determined by the webhook owner; the
//     `Channel` field on Message is ignored. Files cannot be uploaded.
//
//   - "api" — uses bot tokens against api.slack.com. PostMessage hits
//     chat.postMessage, UploadFile hits files.uploadV2 (the modern
//     2-step external upload + complete flow). Supports threading,
//     blocks, attachments, and per-call channel overrides.
//
// Inbound interactivity (Slack POSTing to your service when a user
// clicks a button) is intentionally NOT in this package. That webhook
// belongs in the service that owns the domain action — only signing-
// secret verification and action_id routing live there. This package
// stays focused on outbound traffic so swapping providers is trivial.
package slack

import "context"

// Sender is the interface every Slack provider implements. Inject
// via DI; the concrete sender is selected by config.
//
// Named `Sender` (not `SlackSender`) because callers import the
// package as `slack` — `slack.Sender` reads naturally; `slack.SlackSender`
// stutters.
type Sender interface {
	Name() string
	PostMessage(ctx context.Context, msg Message) (*PostResult, error)
	UploadFile(ctx context.Context, file FileUpload) (*PostResult, error)
}

// Message is the outbound payload accepted by PostMessage. Only one of
// (Text, Blocks, Attachments) is required; passing several is allowed —
// Slack uses Text as the notification fallback when blocks are present.
//
// BlocksJSON / AttachmentsJSON are passed through as raw JSON strings
// so callers can construct rich Block Kit messages without taking a
// dependency on a Slack SDK. Callers are responsible for valid JSON.
type Message struct {
	Channel         string // channel ID (e.g. "C04JB471TQU"); ignored by webhook senders
	Text            string
	BlocksJSON      string // raw block-kit JSON (e.g. `[{"type":"section",...}]`)
	AttachmentsJSON string // raw attachments JSON (legacy; prefer Blocks)
	ThreadTimestamp string // optional — reply-in-thread
	IconEmoji       string // optional — overrides default icon
	IconURL         string
	Username        string // optional — overrides bot display name
	UnfurlLinks     *bool  // tri-state — nil = provider default
	UnfurlMedia     *bool
}

// FileUpload is the payload for UploadFile. Content carries the raw
// bytes; provider-specific impls handle the upload protocol (Slack's
// files.uploadV2 is a 2-step "get URL → PUT bytes → complete" flow).
type FileUpload struct {
	Channels        []string // channel IDs to share the upload into
	Filename        string
	Title           string
	InitialComment  string
	Content         []byte
	ContentType     string // MIME type; informs the upload's content-type
	ThreadTimestamp string
}

// PostResult identifies a delivered message so the caller can store the
// id for threading or later edits/deletes. Fields are nullable strings
// because not every provider returns them — webhook posts return only
// "ok".
type PostResult struct {
	OK              bool
	Channel         string // canonical channel ID Slack chose
	Timestamp       string // message ts — unique per channel
	ProviderMsgID   string // alias for Timestamp; included for parity with pkg/whatsapp
	RawResponseJSON string // entire response body, for debugging
}
