package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeOrderBy_AllowedColumn(t *testing.T) {
	allowed := []string{"created_at", "email", "name"}

	got := SafeOrderBy("email", true, allowed, "created_at")
	assert.Equal(t, "email", got.Column.Name)
	assert.True(t, got.Desc)
}

func TestSafeOrderBy_CamelCaseIsSnakeCased(t *testing.T) {
	allowed := []string{"created_at", "first_name"}

	got := SafeOrderBy("firstName", false, allowed, "created_at")
	assert.Equal(t, "first_name", got.Column.Name)
	assert.False(t, got.Desc)
}

func TestSafeOrderBy_UnknownColumnFallsBack(t *testing.T) {
	allowed := []string{"created_at", "email"}

	// A column not in the allowlist must fall back, never reach SQL.
	got := SafeOrderBy("password", true, allowed, "created_at")
	assert.Equal(t, "created_at", got.Column.Name)
}

func TestSafeOrderBy_InjectionPayloadFallsBack(t *testing.T) {
	allowed := []string{"created_at", "email"}

	got := SafeOrderBy("email); DROP TABLE users;--", false, allowed, "created_at")
	assert.Equal(t, "created_at", got.Column.Name)
}
