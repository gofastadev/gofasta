package featureflag

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// failingProvider implements openfeature.FeatureProvider + StateHandler and
// always errors out of Init so tests can exercise the provider-init error
// branch in newServiceWithProvider. Registered with
// openfeature.SetProviderAndWait, it drives the OpenFeature SDK to return a
// non-nil error which our helper must propagate unchanged.
type failingProvider struct{}

func (failingProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: "always-failing"}
}

func (failingProvider) BooleanEvaluation(_ context.Context, _ string, defaultValue bool, _ openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return openfeature.BoolResolutionDetail{Value: defaultValue}
}

func (failingProvider) StringEvaluation(_ context.Context, _ string, defaultValue string, _ openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{Value: defaultValue}
}

func (failingProvider) FloatEvaluation(_ context.Context, _ string, defaultValue float64, _ openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{Value: defaultValue}
}

func (failingProvider) IntEvaluation(_ context.Context, _ string, defaultValue int64, _ openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{Value: defaultValue}
}

func (failingProvider) ObjectEvaluation(_ context.Context, _ string, defaultValue any, _ openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{Value: defaultValue}
}

func (failingProvider) Hooks() []openfeature.Hook { return nil }

// Init satisfies openfeature.StateHandler and always returns an error —
// this is what SetProviderAndWait surfaces to our constructor's error branch.
func (failingProvider) Init(_ openfeature.EvaluationContext) error {
	return errors.New("synthetic provider init failure")
}

func (failingProvider) Shutdown() {}

// sampleFlags returns an in-memory flag map with one enabled flag
// ("dark-mode") and one disabled flag ("experimental-search") used as shared
// test fixtures.
func sampleFlags() map[string]memprovider.InMemoryFlag {
	return map[string]memprovider.InMemoryFlag{
		"dark-mode": {
			Key:            "dark-mode",
			State:          memprovider.Enabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true, "off": false},
		},
		"experimental-search": {
			Key:            "experimental-search",
			State:          memprovider.Enabled,
			DefaultVariant: "off",
			Variants:       map[string]any{"on": true, "off": false},
		},
	}
}

// resetProvider clears any globally-registered OpenFeature provider so tests
// do not leak state into one another. OpenFeature exposes provider
// registration as package-level global state, so each test sets up its own
// provider and this helper tears it down afterwards.
func resetProvider(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { openfeature.Shutdown() })
}

func TestNewFeatureFlagService(t *testing.T) {
	resetProvider(t)
	svc := NewFeatureFlagService(slog.Default())
	if svc == nil {
		t.Fatal("expected non-nil FeatureFlagService")
	}
}

func TestNewInMemoryService(t *testing.T) {
	resetProvider(t)
	svc, err := NewInMemoryService(sampleFlags(), slog.Default())
	if err != nil {
		t.Fatalf("NewInMemoryService() error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil FeatureFlagService")
	}
}

func TestIsEnabled_FlagEnabled(t *testing.T) {
	resetProvider(t)
	svc, err := NewInMemoryService(sampleFlags(), slog.Default())
	if err != nil {
		t.Fatalf("NewInMemoryService() error: %v", err)
	}

	if !svc.IsEnabled(context.Background(), "dark-mode", "user-123", nil) {
		t.Error("IsEnabled(dark-mode) = false, want true")
	}
}

func TestIsEnabled_FlagDisabledByDefaultVariant(t *testing.T) {
	resetProvider(t)
	svc, err := NewInMemoryService(sampleFlags(), slog.Default())
	if err != nil {
		t.Fatalf("NewInMemoryService() error: %v", err)
	}

	if svc.IsEnabled(context.Background(), "experimental-search", "user-123", nil) {
		t.Error("IsEnabled(experimental-search) = true, want false (default variant resolves to off)")
	}
}

func TestIsEnabled_WithAttributes(t *testing.T) {
	resetProvider(t)
	svc, err := NewInMemoryService(sampleFlags(), slog.Default())
	if err != nil {
		t.Fatalf("NewInMemoryService() error: %v", err)
	}

	got := svc.IsEnabled(context.Background(), "dark-mode", "user-123", map[string]any{"plan": "premium"})
	if !got {
		t.Error("IsEnabled(dark-mode, with attributes) = false, want true")
	}
}

func TestIsEnabled_FlagNotFound(t *testing.T) {
	resetProvider(t)
	svc, err := NewInMemoryService(sampleFlags(), slog.Default())
	if err != nil {
		t.Fatalf("NewInMemoryService() error: %v", err)
	}

	if svc.IsEnabled(context.Background(), "nonexistent-flag", "user-123", nil) {
		t.Error("IsEnabled(nonexistent-flag) = true, want false (missing flags must fall back to default)")
	}
}

func TestIsEnabled_NoProviderRegistered(t *testing.T) {
	resetProvider(t)
	// NewFeatureFlagService alone does not register a provider — the SDK's
	// default NoopProvider should resolve every flag to the caller-supplied
	// default (false).
	svc := NewFeatureFlagService(slog.Default())
	if svc.IsEnabled(context.Background(), "dark-mode", "user-123", nil) {
		t.Error("IsEnabled(dark-mode) with no provider = true, want false (NoopProvider must return the supplied default)")
	}
}

func TestClose(t *testing.T) {
	// Close must be safe to call even with a service that never had a
	// provider registered.
	svc := NewFeatureFlagService(slog.Default())
	svc.Close()
}

func TestNewServiceWithProvider_InitError(t *testing.T) {
	resetProvider(t)
	// Register a provider whose Init returns an error. The helper must
	// propagate the SDK's error unchanged rather than returning a broken
	// service with a nil client.
	svc, err := newServiceWithProvider(failingProvider{}, slog.Default())
	if err == nil {
		t.Fatal("expected error from failing provider init, got nil")
	}
	if svc != nil {
		t.Errorf("expected nil service on init failure, got %+v", svc)
	}
}
