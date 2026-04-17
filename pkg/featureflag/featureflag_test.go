package featureflag

import (
	"context"
	"log/slog"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

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
