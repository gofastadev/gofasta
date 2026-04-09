package featureflag

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	ffclient "github.com/thomaspoignant/go-feature-flag"
)

const testFlagConfig = `dark-mode:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: enabled
`

func writeConfigFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "features.yaml")
	if err := os.WriteFile(path, []byte(testFlagConfig), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// initService creates a new service and registers cleanup to close it.
// It also ensures any prior global ffclient state is closed first.
func initService(t *testing.T) *FeatureFlagService {
	t.Helper()
	ffclient.Close()
	path := writeConfigFile(t)
	svc, err := NewFeatureFlagService(path, slog.Default())
	if err != nil {
		t.Fatalf("NewFeatureFlagService() error: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func TestFeatureFlagService(t *testing.T) {
	t.Run("NewFeatureFlagService_Success", func(t *testing.T) {
		ffclient.Close()
		path := writeConfigFile(t)
		svc, err := NewFeatureFlagService(path, slog.Default())
		if err != nil {
			t.Fatalf("NewFeatureFlagService() error: %v", err)
		}
		if svc == nil {
			t.Fatal("expected non-nil FeatureFlagService")
		}
		svc.Close()
	})

	t.Run("NewFeatureFlagService_InvalidPath", func(t *testing.T) {
		ffclient.Close()
		svc, err := NewFeatureFlagService("/nonexistent/path/features.yaml", slog.Default())
		if err == nil {
			svc.Close()
			t.Fatal("expected error for non-existent config path")
		}
		if svc != nil {
			t.Error("expected nil service on error")
		}
	})

	t.Run("IsEnabled_FlagEnabled", func(t *testing.T) {
		svc := initService(t)
		got := svc.IsEnabled("dark-mode", "user-123", nil)
		if !got {
			t.Error("IsEnabled(dark-mode) = false, want true")
		}
	})

	t.Run("IsEnabled_WithAttributes", func(t *testing.T) {
		svc := initService(t)
		got := svc.IsEnabled("dark-mode", "user-123", map[string]interface{}{"plan": "premium"})
		if !got {
			t.Error("IsEnabled(dark-mode, with attributes) = false, want true")
		}
	})

	t.Run("IsEnabled_FlagNotFound", func(t *testing.T) {
		svc := initService(t)
		got := svc.IsEnabled("nonexistent-flag", "user-123", nil)
		if got {
			t.Error("IsEnabled(nonexistent-flag) = true, want false")
		}
	})

	t.Run("Close", func(t *testing.T) {
		ffclient.Close()
		path := writeConfigFile(t)
		svc, err := NewFeatureFlagService(path, slog.Default())
		if err != nil {
			t.Fatalf("NewFeatureFlagService() error: %v", err)
		}
		// Should not panic
		svc.Close()
	})
}
