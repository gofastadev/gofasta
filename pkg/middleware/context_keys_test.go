package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextKeys_Values(t *testing.T) {
	assert.Equal(t, contextKey("requestID"), RequestIDKey)
	assert.Equal(t, contextKey("userClaims"), UserClaimsKey)
}

func TestContextKey_TypeSafety(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-id")

	val, ok := ctx.Value(RequestIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "test-id", val)

	// Plain string key should not match the typed contextKey
	val2, ok2 := ctx.Value("requestID").(string)
	assert.False(t, ok2)
	assert.Empty(t, val2)
}
