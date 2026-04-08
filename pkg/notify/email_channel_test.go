package notify

import (
	"context"
	"testing"

	"github.com/gofastadev/gofasta/pkg/mailer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmailSender implements mailer.EmailSender for testing.
type mockEmailSender struct {
	sentMessages []mailer.EmailMessage
	sendFunc     func(ctx context.Context, msg mailer.EmailMessage) error
}

func (m *mockEmailSender) Send(ctx context.Context, msg mailer.EmailMessage) error {
	m.sentMessages = append(m.sentMessages, msg)
	if m.sendFunc != nil {
		return m.sendFunc(ctx, msg)
	}
	return nil
}

func TestNewEmailChannel(t *testing.T) {
	sender := &mockEmailSender{}
	ch := NewEmailChannel(sender)
	assert.NotNil(t, ch)
	assert.Equal(t, ChannelEmail, ch.Channel())
}

func TestEmailChannel_Send(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
		recipient    Recipient
		expectTo     []string
		expectSubj   string
		expectHTML   string
		expectTpl    string
		expectData   map[string]any
	}{
		{
			name: "plain body wraps in paragraph",
			notification: Notification{
				Subject: "Test Subject",
				Body:    "Hello world",
			},
			recipient:  Recipient{Email: "user@example.com"},
			expectTo:   []string{"user@example.com"},
			expectSubj: "Test Subject",
			expectHTML: "<p>Hello world</p>",
		},
		{
			name: "HTML body used directly",
			notification: Notification{
				Subject:  "HTML Email",
				HTMLBody: "<h1>Hello</h1>",
			},
			recipient:  Recipient{Email: "user@example.com"},
			expectTo:   []string{"user@example.com"},
			expectSubj: "HTML Email",
			expectHTML: "<h1>Hello</h1>",
		},
		{
			name: "template takes precedence over HTML body",
			notification: Notification{
				Subject:  "Template Email",
				HTMLBody: "<p>ignored</p>",
				Template: "welcome",
				Data:     map[string]any{"name": "John"},
			},
			recipient:  Recipient{Email: "john@example.com"},
			expectTo:   []string{"john@example.com"},
			expectSubj: "Template Email",
			expectTpl:  "welcome",
			expectData: map[string]any{"name": "John"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &mockEmailSender{}
			ch := NewEmailChannel(sender)

			err := ch.Send(context.Background(), tt.recipient, tt.notification)
			require.NoError(t, err)
			require.Len(t, sender.sentMessages, 1)

			msg := sender.sentMessages[0]
			assert.Equal(t, tt.expectTo, msg.To)
			assert.Equal(t, tt.expectSubj, msg.Subject)

			if tt.expectTpl != "" {
				assert.Equal(t, tt.expectTpl, msg.Template)
				assert.Equal(t, tt.expectData, msg.TemplateData)
			} else {
				assert.Equal(t, tt.expectHTML, msg.HTMLBody)
			}
		})
	}
}

func TestEmailChannel_Send_Error(t *testing.T) {
	sender := &mockEmailSender{
		sendFunc: func(ctx context.Context, msg mailer.EmailMessage) error {
			return assert.AnError
		},
	}
	ch := NewEmailChannel(sender)

	err := ch.Send(context.Background(), Recipient{Email: "user@example.com"}, Notification{Subject: "Test"})
	assert.Error(t, err)
}
