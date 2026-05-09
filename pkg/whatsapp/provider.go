package whatsapp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gofastadev/gofasta/pkg/config"
)

// NewSender returns the Sender selected by WhatsAppConfig.Provider.
// Used by Wire DI; developers don't call this directly.
//
//   - ""         → noopSender (Send returns "not configured"; safe default)
//   - "ultramsg" → UltraMsgSender
//   - "twilio"   → TwilioSender
//   - "meta"     → MetaSender
//
// Adding a new provider (MessageBird, Vonage, GreenAPI, 360dialog,
// Bird, etc.):
//
//  1. Drop a `<provider>.go` file implementing Sender into this
//     package. Mirror the structure of ultramsg.go / meta.go.
//  2. Add a sub-config struct in pkg/config/config.go and reference it
//     from WhatsAppConfig.
//  3. Add a switch case here.
//
// No code outside pkg/whatsapp needs to change.
func NewSender(cfg *config.WhatsAppConfig, logger *slog.Logger) (Sender, error) {
	if cfg == nil {
		return &noopSender{}, nil
	}
	switch cfg.Provider {
	case "":
		return &noopSender{}, nil
	case "ultramsg":
		if cfg.UltraMsg.BaseURL == "" || cfg.UltraMsg.InstanceID == "" || cfg.UltraMsg.Token == "" {
			return nil, fmt.Errorf("whatsapp provider=ultramsg requires UltraMsg.BaseURL, InstanceID, Token")
		}
		return NewUltraMsgSender(cfg.UltraMsg, logger), nil
	case "twilio":
		if cfg.Twilio.AccountSID == "" || cfg.Twilio.AuthToken == "" || cfg.Twilio.FromNumber == "" {
			return nil, fmt.Errorf("whatsapp provider=twilio requires Twilio.AccountSID, AuthToken, FromNumber")
		}
		return NewTwilioSender(cfg.Twilio, logger), nil
	case "meta":
		if cfg.Meta.AccessToken == "" || cfg.Meta.PhoneNumberID == "" {
			return nil, fmt.Errorf("whatsapp provider=meta requires Meta.AccessToken, PhoneNumberID")
		}
		return NewMetaSender(cfg.Meta, logger), nil
	default:
		return nil, fmt.Errorf("unknown whatsapp provider: %q (supported: ultramsg, twilio, meta)", cfg.Provider)
	}
}

type noopSender struct{}

// Name implements Sender.
func (n *noopSender) Name() string { return "whatsapp-noop" }

// Send returns a "not configured" error so misconfiguration is loud.
func (n *noopSender) Send(_ context.Context, _ Message) (*SendResult, error) {
	return nil, fmt.Errorf("whatsapp: not configured (set whatsapp.provider in config.yaml to enable)")
}

// DeleteMessage returns ErrUnsupported, matching the rest of the
// disabled-state contract.
func (n *noopSender) DeleteMessage(_ context.Context, _ string) error {
	return ErrUnsupported
}
