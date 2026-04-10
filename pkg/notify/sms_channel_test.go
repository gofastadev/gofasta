package notify

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// mockTwilioAPI implements twilioMessageCreator for testing.
type mockTwilioAPI struct {
	createFunc func(params *api.CreateMessageParams) (*api.ApiV2010Message, error)
}

func (m *mockTwilioAPI) CreateMessage(params *api.CreateMessageParams) (*api.ApiV2010Message, error) {
	return m.createFunc(params)
}

func TestNewSMSChannel(t *testing.T) {
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	require.NotNil(t, ch)
	assert.NotNil(t, ch.api)
	assert.Equal(t, "+15551234567", ch.fromNumber)
}

func TestSMSChannel_Channel(t *testing.T) {
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	assert.Equal(t, ChannelSMS, ch.Channel())
}

func TestSMSChannel_Send_Success(t *testing.T) {
	mock := &mockTwilioAPI{
		createFunc: func(params *api.CreateMessageParams) (*api.ApiV2010Message, error) {
			return &api.ApiV2010Message{}, nil
		},
	}
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	ch.api = mock

	err := ch.Send(context.Background(), Recipient{Phone: "+15559876543"}, Notification{
		Subject: "Alert",
		Body:    "Server is down",
	})
	assert.NoError(t, err)
}

func TestSMSChannel_Send_Error(t *testing.T) {
	expectedErr := errors.New("twilio: service unavailable")
	mock := &mockTwilioAPI{
		createFunc: func(params *api.CreateMessageParams) (*api.ApiV2010Message, error) {
			return nil, expectedErr
		},
	}
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	ch.api = mock

	err := ch.Send(context.Background(), Recipient{Phone: "+15559876543"}, Notification{
		Subject: "Alert",
		Body:    "Server is down",
	})
	assert.ErrorIs(t, err, expectedErr)
}

func TestSMSChannel_Send_FormatsBody(t *testing.T) {
	var captured *api.CreateMessageParams
	mock := &mockTwilioAPI{
		createFunc: func(params *api.CreateMessageParams) (*api.ApiV2010Message, error) {
			captured = params
			return &api.ApiV2010Message{}, nil
		},
	}
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	ch.api = mock

	err := ch.Send(context.Background(), Recipient{Phone: "+15559876543"}, Notification{
		Subject: "Outage",
		Body:    "DB unreachable",
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "+15559876543", *captured.To)
	assert.Equal(t, "+15551234567", *captured.From)
	assert.Equal(t, "Outage: DB unreachable", *captured.Body)
}
