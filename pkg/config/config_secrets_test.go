package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSecrets_RejectsPlaceholders(t *testing.T) {
	cfg := &AppConfig{}
	cfg.Auth.JWTSecret = PlaceholderJWTSecret
	cfg.Session.Secret = PlaceholderSessionSecret

	err := cfg.ValidateSecrets()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.jwt_secret")
	assert.Contains(t, err.Error(), "session.secret")
}

func TestValidateSecrets_RejectsEmpty(t *testing.T) {
	cfg := &AppConfig{}

	err := cfg.ValidateSecrets()
	require.Error(t, err)
}

func TestValidateSecrets_AcceptsRealSecrets(t *testing.T) {
	cfg := &AppConfig{}
	cfg.Auth.JWTSecret = "a-real-random-32-byte-secret-value"
	cfg.Session.Secret = "another-real-random-secret-value!!"

	require.NoError(t, cfg.ValidateSecrets())
}

func TestApplyDefaults_UsesPlaceholderConstants(t *testing.T) {
	cfg := &AppConfig{}
	applyDefaults(cfg)

	assert.Equal(t, PlaceholderJWTSecret, cfg.Auth.JWTSecret)
	assert.Equal(t, PlaceholderSessionSecret, cfg.Session.Secret)
	// Defaults are applied for local dev, but ValidateSecrets must still reject them.
	require.Error(t, cfg.ValidateSecrets())
}
