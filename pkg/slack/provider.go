package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gofastadev/gofasta/pkg/config"
)

// NewSlackSender returns the Sender selected by SlackConfig.Provider.
// Used by Wire DI; developers don't call this directly.
//
//   - "webhook" → WebhookSender (uses cfg.WebhookURL)
//   - "api"     → APISender     (uses cfg.BotToken)
//   - "" (empty) → a no-op stub that returns "not configured" errors,
//     so a service that doesn't need Slack today doesn't fail boot.
func NewSlackSender(cfg *config.SlackConfig, logger *slog.Logger) (Sender, error) {
	if cfg == nil {
		return &noopSender{}, nil
	}
	switch cfg.Provider {
	case "":
		return &noopSender{}, nil
	case "webhook":
		if cfg.WebhookURL == "" {
			return nil, fmt.Errorf("slack provider=webhook requires SlackConfig.WebhookURL")
		}
		return NewWebhookSender(cfg.WebhookURL, logger), nil
	case "api":
		if cfg.BotToken == "" {
			return nil, fmt.Errorf("slack provider=api requires SlackConfig.BotToken")
		}
		return NewAPISender(cfg.BotToken, logger), nil
	default:
		return nil, fmt.Errorf("unknown slack provider: %q (supported: webhook, api)", cfg.Provider)
	}
}

// noopSender is the disabled-by-default sender. Surfaces a clear error
// rather than silently dropping messages so test failures and dev
// onboarding don't regress invisibly.
type noopSender struct{}

// Name implements Sender.
func (n *noopSender) Name() string { return "slack-noop" }

// PostMessage always returns "not configured" so misconfiguration is
// loud rather than silent.
func (n *noopSender) PostMessage(_ context.Context, _ Message) (*PostResult, error) {
	return nil, fmt.Errorf("slack: not configured (set slack.provider in config.yaml to enable)")
}

// UploadFile always returns "not configured" — same rationale as
// PostMessage.
func (n *noopSender) UploadFile(_ context.Context, _ FileUpload) (*PostResult, error) {
	return nil, fmt.Errorf("slack: not configured (set slack.provider in config.yaml to enable)")
}
