package notify

import (
	"context"

	"github.com/gofastadev/gofasta/pkg/mailer"
)

// EmailChannel sends notifications via email using the existing mailer.
type EmailChannel struct {
	sender mailer.EmailSender
}

// NewEmailChannel returns an EmailChannel delegating to sender.
func NewEmailChannel(sender mailer.EmailSender) *EmailChannel {
	return &EmailChannel{sender: sender}
}

// Channel returns the channel identifier for routing.
func (c *EmailChannel) Channel() Channel { return ChannelEmail }

// Send dispatches n to recipient as an email.
func (c *EmailChannel) Send(ctx context.Context, recipient Recipient, n Notification) error {
	msg := mailer.EmailMessage{
		To:      []string{recipient.Email},
		Subject: n.Subject,
	}
	switch {
	case n.Template != "":
		msg.Template = n.Template
		msg.TemplateData = n.Data
	case n.HTMLBody != "":
		msg.HTMLBody = n.HTMLBody
	default:
		msg.HTMLBody = "<p>" + n.Body + "</p>"
	}
	return c.sender.Send(ctx, msg)
}
