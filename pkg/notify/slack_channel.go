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

func NewSlackChannel(webhookURL string) *SlackChannel {
	return &SlackChannel{webhookURL: webhookURL, client: &http.Client{}}
}

func (c *SlackChannel) Channel() Channel { return ChannelSlack }

func (c *SlackChannel) Send(ctx context.Context, recipient Recipient, n Notification) error {
	payload := map[string]string{"text": fmt.Sprintf("*%s*\n%s", n.Subject, n.Body)}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
