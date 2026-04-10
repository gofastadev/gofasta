package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackChannel sends notifications via Slack webhook.
type SlackChannel struct {
	webhookURL string
	client     *http.Client
}

// NewSlackChannel returns a SlackChannel targeting the given incoming-webhook URL.
func NewSlackChannel(webhookURL string) *SlackChannel {
	return &SlackChannel{webhookURL: webhookURL, client: &http.Client{}}
}

// Channel returns the channel identifier for routing.
func (c *SlackChannel) Channel() Channel { return ChannelSlack }

// Send posts n to Slack as a webhook payload.
func (c *SlackChannel) Send(ctx context.Context, _ Recipient, n Notification) error {
	payload := map[string]string{"text": fmt.Sprintf("*%s*\n%s", n.Subject, n.Body)}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
