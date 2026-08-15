package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- ClaimsFromContext ----------

func TestClaimsFromContext_WithClaims(t *testing.T) {
	claims := &Claims{UserID: "user-123", Role: "admin"}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	got, err := ClaimsFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "user-123", got.UserID)
	assert.Equal(t, "admin", got.Role)
}

func TestClaimsFromContext_NoClaims(t *testing.T) {
	ctx := context.Background()

	got, err := ClaimsFromContext(ctx)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "no claims in context")
}

func TestClaimsFromContext_NilClaims(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimsKey, (*Claims)(nil))

	got, err := ClaimsFromContext(ctx)
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestClaimsFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimsKey, "not-claims")

	got, err := ClaimsFromContext(ctx)
	require.Error(t, err)
	assert.Nil(t, got)
}

// ---------- Claims key ----------

func TestClaimsKey_Value(t *testing.T) {
	assert.Equal(t, contextKey("claims"), ClaimsKey)
}
