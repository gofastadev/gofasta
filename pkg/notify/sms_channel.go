package notify

import (
	"context"

	twilio "github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSChannel sends notifications via Twilio SMS.
type SMSChannel struct {
	client   *twilio.RestClient
	fromNumber string
}

func NewSMSChannel(accountSID, authToken, fromNumber string) *SMSChannel {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &SMSChannel{client: client, fromNumber: fromNumber}
}

func (c *SMSChannel) Channel() Channel { return ChannelSMS }

func (c *SMSChannel) Send(ctx context.Context, recipient Recipient, n Notification) error {
	params := &api.CreateMessageParams{}
	params.SetTo(recipient.Phone)
	params.SetFrom(c.fromNumber)
	params.SetBody(n.Subject + ": " + n.Body)
	_, err := c.client.Api.CreateMessage(params)
	return err
}
