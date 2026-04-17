// Package featureflag wraps the OpenFeature Go SDK so callers can evaluate
// feature flags through a stable interface while remaining free to swap the
// underlying provider (in-memory, Flagd, LaunchDarkly, go-feature-flag,
// ConfigCat, or a custom implementation) at application startup via
// openfeature.SetProvider.
//
// OpenFeature is the CNCF-standard flag-evaluation contract; this package
// adds a thin, gofasta-shaped facade — a single FeatureFlagService type with
// an IsEnabled method — so application code does not have to import the
// OpenFeature SDK directly unless it needs advanced features.
package featureflag

import (
	"context"
	"log/slog"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// FeatureFlagService evaluates feature flags through whichever OpenFeature
// provider the application has registered. If no provider has been set, the
// SDK's default NoopProvider answers every evaluation with the supplied
// default, which makes this type safe to construct before any provider is
// wired up.
//
//nolint:revive // name kept for public-API stability; rename is a breaking change.
type FeatureFlagService struct {
	client *openfeature.Client
	logger *slog.Logger
}

// NewFeatureFlagService returns a service bound to the currently-registered
// OpenFeature provider. Callers are expected to register a provider via
// openfeature.SetProvider (or openfeature.SetProviderAndWait) during startup;
// if they do not, every IsEnabled call resolves to the caller-supplied
// default via the SDK's NoopProvider.
func NewFeatureFlagService(logger *slog.Logger) *FeatureFlagService {
	client := openfeature.NewClient("gofasta")
	logger.Info("feature flag service initialized")
	return &FeatureFlagService{client: client, logger: logger}
}

// NewInMemoryService bootstraps the OpenFeature SDK with an in-memory provider
// populated from the supplied flag map and returns a FeatureFlagService wired
// against it. Intended for local development, tests, and small applications
// that do not need a rule engine. Calling this replaces any previously
// registered global provider.
func NewInMemoryService(flags map[string]memprovider.InMemoryFlag, logger *slog.Logger) (*FeatureFlagService, error) {
	return newServiceWithProvider(memprovider.NewInMemoryProvider(flags), logger)
}

// newServiceWithProvider registers an arbitrary OpenFeature provider and
// returns a FeatureFlagService wired against it. Exposed as an unexported
// helper so tests can exercise the provider-init-error branch with a mock
// provider; callers outside the package should use NewInMemoryService or
// register their provider directly via openfeature.SetProvider.
func newServiceWithProvider(provider openfeature.FeatureProvider, logger *slog.Logger) (*FeatureFlagService, error) {
	if err := openfeature.SetProviderAndWait(provider); err != nil {
		return nil, err
	}
	return NewFeatureFlagService(logger), nil
}

// IsEnabled resolves a boolean flag for the supplied user context. Attributes
// become OpenFeature evaluation-context attributes that the underlying
// provider's targeting rules can read. If the provider returns an error the
// service falls back to false and logs the failure at debug level — feature
// flags must never take an application down.
func (s *FeatureFlagService) IsEnabled(ctx context.Context, flagKey, userID string, attributes map[string]any) bool {
	evalCtx := openfeature.NewEvaluationContext(userID, attributes)
	val, err := s.client.BooleanValue(ctx, flagKey, false, evalCtx)
	if err != nil {
		s.logger.Debug("feature flag evaluation failed, using default",
			"flag", flagKey, "error", err)
		return false
	}
	return val
}

// Close shuts down every provider the OpenFeature SDK has registered. Safe to
// call on a service that was constructed via NewFeatureFlagService without
// any provider wired up — Shutdown is a no-op in that case.
func (s *FeatureFlagService) Close() {
	openfeature.Shutdown()
}
