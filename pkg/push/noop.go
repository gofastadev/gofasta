package push

import (
	"context"
	"log/slog"
)

// noopSender is the safe default returned by NewSender when no
// provider is configured. Its Send* methods return ErrNotConfigured —
// loud at runtime so a misconfigured deployment doesn't silently lose
// every push. Subscribe/Unsubscribe return ErrUnsupported for
// symmetry with real providers that lack the verb.
//
// Useful for local dev when you want a service to boot but don't have
// FCM credentials handy.
type noopSender struct {
	logger *slog.Logger
}

func (s *noopSender) Name() string { return "noop" }

func (s *noopSender) SendToTokens(_ context.Context, tokens []string, msg Message) ([]TokenResult, error) {
	if s.logger != nil {
		s.logger.Debug("push noop: send-to-tokens dropped",
			"tokenCount", len(tokens),
			"title", msg.Title,
			"priority", msg.Priority,
		)
	}
	return nil, ErrNotConfigured
}

func (s *noopSender) SendToTopic(_ context.Context, topic string, msg Message) (*SendResult, error) {
	if s.logger != nil {
		s.logger.Debug("push noop: send-to-topic dropped",
			"topic", topic,
			"title", msg.Title,
		)
	}
	return nil, ErrNotConfigured
}

func (s *noopSender) SubscribeToTopic(_ context.Context, _ string, _ []string) (*TopicMembershipResult, error) {
	return nil, ErrUnsupported
}

func (s *noopSender) UnsubscribeFromTopic(_ context.Context, _ string, _ []string) (*TopicMembershipResult, error) {
	return nil, ErrUnsupported
}
