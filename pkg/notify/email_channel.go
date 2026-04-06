package notify

import (
	"context"

	"github.com/healtronlabs/gofasta/pkg/mailer"
)

// EmailChannel sends notifications via email using the existing mailer.
type EmailChannel struct {
	sender mailer.EmailSender
}

func NewEmailChannel(sender mailer.EmailSender) *EmailChannel {
	return &EmailChannel{sender: sender}
}

func (c *EmailChannel) Channel() Channel { return ChannelEmail }

func (c *EmailChannel) Send(ctx context.Context, recipient Recipient, n Notification) error {
	msg := mailer.EmailMessage{
		To:      []string{recipient.Email},
		Subject: n.Subject,
	}
	if n.Template != "" {
		msg.Template = n.Template
		msg.TemplateData = n.Data
	} else if n.HTMLBody != "" {
		msg.HTMLBody = n.HTMLBody
	} else {
		msg.HTMLBody = "<p>" + n.Body + "</p>"
	}
	return c.sender.Send(ctx, msg)
}
