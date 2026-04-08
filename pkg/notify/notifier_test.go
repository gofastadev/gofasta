package notify

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChannelSender implements ChannelSender for testing.
type mockChannelSender struct {
	channel  Channel
	sendFunc func(ctx context.Context, recipient Recipient, notification Notification) error
	calls    []sendCall
}

type sendCall struct {
	Recipient    Recipient
	Notification Notification
}

func (m *mockChannelSender) Channel() Channel { return m.channel }

func (m *mockChannelSender) Send(ctx context.Context, recipient Recipient, notification Notification) error {
	m.calls = append(m.calls, sendCall{Recipient: recipient, Notification: notification})
	if m.sendFunc != nil {
		return m.sendFunc(ctx, recipient, notification)
	}
	return nil
}

func testNotifyLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewNotifier(t *testing.T) {
	emailSender := &mockChannelSender{channel: ChannelEmail}
	smsSender := &mockChannelSender{channel: ChannelSMS}

	n := NewNotifier(testNotifyLogger(), emailSender, smsSender)
	assert.NotNil(t, n)
	assert.Len(t, n.channels, 2)
}

func TestNewNotifier_NoDuplicateChannels(t *testing.T) {
	sender1 := &mockChannelSender{channel: ChannelEmail}
	sender2 := &mockChannelSender{channel: ChannelEmail}

	n := NewNotifier(testNotifyLogger(), sender1, sender2)
	assert.Len(t, n.channels, 1, "duplicate channel should be overwritten")
}

func TestNotifier_Send_SpecificChannels(t *testing.T) {
	emailSender := &mockChannelSender{channel: ChannelEmail}
	smsSender := &mockChannelSender{channel: ChannelSMS}

	n := NewNotifier(testNotifyLogger(), emailSender, smsSender)

	recipient := Recipient{Email: "test@example.com", Name: "Test"}
	notification := Notification{
		Subject:  "Hello",
		Body:     "World",
		Channels: []Channel{ChannelEmail},
	}

	err := n.Send(context.Background(), recipient, notification)
	require.NoError(t, err)

	assert.Len(t, emailSender.calls, 1)
	assert.Len(t, smsSender.calls, 0, "SMS should not have been called")
	assert.Equal(t, "test@example.com", emailSender.calls[0].Recipient.Email)
}

func TestNotifier_Send_AllChannelsWhenNoneSpecified(t *testing.T) {
	emailSender := &mockChannelSender{channel: ChannelEmail}
	smsSender := &mockChannelSender{channel: ChannelSMS}

	n := NewNotifier(testNotifyLogger(), emailSender, smsSender)

	recipient := Recipient{Email: "test@example.com", Phone: "+1234567890"}
	notification := Notification{
		Subject:  "Hello",
		Body:     "World",
		Channels: nil, // no channels specified
	}

	err := n.Send(context.Background(), recipient, notification)
	require.NoError(t, err)

	assert.Len(t, emailSender.calls, 1)
	assert.Len(t, smsSender.calls, 1)
}

func TestNotifier_Send_UnconfiguredChannel(t *testing.T) {
	emailSender := &mockChannelSender{channel: ChannelEmail}
	n := NewNotifier(testNotifyLogger(), emailSender)

	notification := Notification{
		Subject:  "Hello",
		Channels: []Channel{ChannelSlack}, // not registered
	}

	err := n.Send(context.Background(), Recipient{}, notification)
	assert.NoError(t, err, "unconfigured channel should be skipped, not error")
	assert.Len(t, emailSender.calls, 0)
}

func TestNotifier_Send_ChannelError(t *testing.T) {
	failSender := &mockChannelSender{
		channel: ChannelEmail,
		sendFunc: func(ctx context.Context, recipient Recipient, notification Notification) error {
			return errors.New("email failed")
		},
	}

	n := NewNotifier(testNotifyLogger(), failSender)

	notification := Notification{
		Subject:  "Hello",
		Channels: []Channel{ChannelEmail},
	}

	err := n.Send(context.Background(), Recipient{Email: "test@example.com"}, notification)
	assert.Error(t, err)
	assert.Equal(t, "email failed", err.Error())
}

func TestNotifier_Send_PartialFailure(t *testing.T) {
	emailSender := &mockChannelSender{
		channel: ChannelEmail,
		sendFunc: func(ctx context.Context, recipient Recipient, notification Notification) error {
			return errors.New("email failed")
		},
	}
	smsSender := &mockChannelSender{channel: ChannelSMS}

	n := NewNotifier(testNotifyLogger(), emailSender, smsSender)

	notification := Notification{
		Subject:  "Hello",
		Channels: []Channel{ChannelEmail, ChannelSMS},
	}

	err := n.Send(context.Background(), Recipient{}, notification)
	// lastErr is returned, which is from email (first in the list)
	// but SMS should still have been attempted
	assert.Error(t, err)
	assert.Len(t, smsSender.calls, 1, "SMS should still be attempted even if email fails")
}

func TestNotifier_RegisterChannel(t *testing.T) {
	n := NewNotifier(testNotifyLogger())
	assert.Empty(t, n.channels)

	sender := &mockChannelSender{channel: ChannelSlack}
	n.RegisterChannel(sender)

	assert.Len(t, n.channels, 1)
	assert.NotNil(t, n.channels[ChannelSlack])
}
