package notify

import (
	"context"
	"log/slog"
)

// Channel represents a notification delivery channel.
type Channel string

// Supported notification channels.
const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelSlack    Channel = "slack"
	ChannelDatabase Channel = "database"
)

// Notification holds the content to be sent across channels.
type Notification struct {
	Subject  string         // email subject / SMS prefix
	Body     string         // plain text body
	HTMLBody string         // HTML body (for email)
	Template string         // email template name (optional)
	Data     map[string]any // template data
	Channels []Channel      // which channels to use
}

// Recipient represents who receives the notification.
type Recipient struct {
	ID    string // user ID (for database channel)
	Email string // for email channel
	Phone string // for SMS channel
	Name  string // display name
}

// ChannelSender is implemented by each channel (email, SMS, slack, etc.)
type ChannelSender interface {
	Channel() Channel
	Send(ctx context.Context, recipient Recipient, notification Notification) error
}

// Notifier dispatches notifications to multiple channels.
type Notifier struct {
	channels map[Channel]ChannelSender
	logger   *slog.Logger
}

// NewNotifier creates a notifier with the given channel senders.
func NewNotifier(logger *slog.Logger, senders ...ChannelSender) *Notifier {
	channels := make(map[Channel]ChannelSender)
	for _, s := range senders {
		channels[s.Channel()] = s
	}
	return &Notifier{channels: channels, logger: logger}
}

// Send dispatches a notification to all specified channels.
// If no channels are specified on the notification, sends to all registered channels.
func (n *Notifier) Send(ctx context.Context, recipient Recipient, notification Notification) error {
	targets := notification.Channels
	if len(targets) == 0 {
		// Default: send to all registered channels
		for ch := range n.channels {
			targets = append(targets, ch)
		}
	}

	var lastErr error
	for _, ch := range targets {
		sender, ok := n.channels[ch]
		if !ok {
			n.logger.Warn("notification channel not configured", "channel", ch)
			continue
		}
		if err := sender.Send(ctx, recipient, notification); err != nil {
			n.logger.Error("notification send failed", "channel", ch, "recipient", recipient.Email, "error", err)
			lastErr = err
		} else {
			n.logger.Info("notification sent", "channel", ch, "recipient", recipient.Email)
		}
	}
	return lastErr
}

// RegisterChannel adds a channel sender at runtime.
func (n *Notifier) RegisterChannel(sender ChannelSender) {
	n.channels[sender.Channel()] = sender
}
