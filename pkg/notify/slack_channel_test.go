package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlackChannel(t *testing.T) {
	ch := NewSlackChannel("https://hooks.slack.com/services/test")
	require.NotNil(t, ch)
	assert.Equal(t, "https://hooks.slack.com/services/test", ch.webhookURL)
	assert.NotNil(t, ch.client)
}

func TestSlackChannel_Channel(t *testing.T) {
	ch := NewSlackChannel("https://hooks.slack.com/services/test")
	assert.Equal(t, ChannelSlack, ch.Channel())
}

func TestSlackChannel_Send_Success(t *testing.T) {
	var receivedContentType string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ch := NewSlackChannel(server.URL)

	recipient := Recipient{ID: "user-1", Email: "test@example.com"}
	notification := Notification{
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	err := ch.Send(context.Background(), recipient, notification)
	require.NoError(t, err)
	assert.Equal(t, "application/json", receivedContentType)
	assert.Contains(t, string(receivedBody), "Test Subject")
	assert.Contains(t, string(receivedBody), "Test Body")
}

func TestSlackChannel_Send_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	ch := NewSlackChannel(server.URL)

	recipient := Recipient{ID: "user-1"}
	notification := Notification{
		Subject: "Test",
		Body:    "Body",
	}

	err := ch.Send(context.Background(), recipient, notification)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slack webhook returned 400")
}

func TestSlackChannel_Send_ConnectionError(t *testing.T) {
	ch := NewSlackChannel("http://localhost:1") // unlikely to be listening

	err := ch.Send(context.Background(), Recipient{}, Notification{Subject: "Hi", Body: "There"})
	require.Error(t, err)
}

func TestSlackChannel_Send_VariousHTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"200 OK", http.StatusOK, false},
		{"201 Created", http.StatusCreated, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"403 Forbidden", http.StatusForbidden, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			ch := NewSlackChannel(server.URL)
			err := ch.Send(context.Background(), Recipient{}, Notification{Subject: "S", Body: "B"})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
