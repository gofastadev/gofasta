package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMSChannel(t *testing.T) {
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	require.NotNil(t, ch)
	assert.NotNil(t, ch.client)
	assert.Equal(t, "+15551234567", ch.fromNumber)
}

func TestSMSChannel_Channel(t *testing.T) {
	ch := NewSMSChannel("AC_test_sid", "auth_token_123", "+15551234567")
	assert.Equal(t, ChannelSMS, ch.Channel())
}
