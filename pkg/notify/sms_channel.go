package notify

import (
	"context"

	twilio "github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// twilioMessageCreator is a narrow interface satisfied by the real Twilio API
// client, enabling lightweight test doubles without heavyweight SDK mocks.
type twilioMessageCreator interface {
	CreateMessage(params *api.CreateMessageParams) (*api.ApiV2010Message, error)
}

// SMSChannel sends notifications via Twilio SMS.
type SMSChannel struct {
	api        twilioMessageCreator
	fromNumber string
}

// NewSMSChannel builds an SMSChannel backed by the Twilio REST API.
func NewSMSChannel(accountSID, authToken, fromNumber string) *SMSChannel {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &SMSChannel{api: client.Api, fromNumber: fromNumber}
}

// Channel returns the channel identifier for routing.
func (c *SMSChannel) Channel() Channel { return ChannelSMS }

// Send dispatches n to recipient as an SMS. The context is accepted for API
// symmetry with other channels but not forwarded to the Twilio client.
func (c *SMSChannel) Send(_ context.Context, recipient Recipient, n Notification) error {
	params := &api.CreateMessageParams{}
	params.SetTo(recipient.Phone)
	params.SetFrom(c.fromNumber)
	params.SetBody(n.Subject + ": " + n.Body)
	_, err := c.api.CreateMessage(params)
	return err
}
