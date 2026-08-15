package mailer

import (
	"context"
	"log/slog"
)

// EmailSender is the interface all email providers implement.
// Inject this into services via DI — the provider is selected by config.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// EmailMessage holds everything needed to send an email.
// Use Template + TemplateData for rendered HTML templates,
// or HTMLBody for raw pre-built HTML.
type EmailMessage struct {
	To       []string
	Subject  string
	HTMLBody string // pre-rendered HTML (used if Template is empty)
	// TextBody is the plain-text alternative. Optional, and worth setting:
	// a message with only an HTML part is harder to read in a text-only
	// client, and spam filters score multipart/alternative more kindly than
	// HTML alone. Providers that cannot express an alternative ignore it.
	TextBody     string
	Template     string         // template name, e.g. "welcome" → templates/emails/welcome.html
	TemplateData map[string]any // data passed into the template
	CC           []string
	BCC          []string
	ReplyTo      string
	Attachments  []Attachment
}

// Attachment represents an email file attachment.
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

// loggerOrDefault returns logger, or slog's default when the caller passed
// nil. A nil *slog.Logger panics on first use, and every send path logs, so a
// constructor that accepted nil was handing back a sender that worked right
// up until it succeeded.
func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
