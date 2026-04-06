package mailer

import "context"

// EmailSender is the interface all email providers implement.
// Inject this into services via DI — the provider is selected by config.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// EmailMessage holds everything needed to send an email.
// Use Template + TemplateData for rendered HTML templates,
// or HTMLBody for raw pre-built HTML.
type EmailMessage struct {
	To           []string
	Subject      string
	HTMLBody     string         // pre-rendered HTML (used if Template is empty)
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
