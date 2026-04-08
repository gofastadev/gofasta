package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofastadev/gofasta/pkg/config"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *config.LogConfig
		verify func(t *testing.T, logger *slog.Logger)
	}{
		{
			name: "json format",
			cfg:  &config.LogConfig{Level: "info", Format: "json"},
			verify: func(t *testing.T, logger *slog.Logger) {
				require.NotNil(t, logger)
			},
		},
		{
			name: "text format",
			cfg:  &config.LogConfig{Level: "debug", Format: "text"},
			verify: func(t *testing.T, logger *slog.Logger) {
				require.NotNil(t, logger)
			},
		},
		{
			name: "default format (non-json falls back to text)",
			cfg:  &config.LogConfig{Level: "info", Format: ""},
			verify: func(t *testing.T, logger *slog.Logger) {
				require.NotNil(t, logger)
			},
		},
		{
			name: "uppercase JSON format",
			cfg:  &config.LogConfig{Level: "error", Format: "JSON"},
			verify: func(t *testing.T, logger *slog.Logger) {
				require.NotNil(t, logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.cfg)
			tt.verify(t, logger)
		})
	}
}

func TestNewLogger_SetsDefault(t *testing.T) {
	cfg := &config.LogConfig{Level: "warn", Format: "text"}
	logger := NewLogger(cfg)
	// NewLogger sets slog.SetDefault, so slog.Default() should be the same logger
	assert.Equal(t, logger, slog.Default())
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"Debug uppercase", "Debug", slog.LevelDebug},
		{"warn", "warn", slog.LevelWarn},
		{"warning", "warning", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"ERROR uppercase", "ERROR", slog.LevelError},
		{"info", "info", slog.LevelInfo},
		{"empty defaults to info", "", slog.LevelInfo},
		{"unknown defaults to info", "unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseLevel(tt.input))
		})
	}
}
