package auth

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
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

// ---------- SubjectID ----------

func TestSubjectID_PrefersUserIDThenFallsBackToSub(t *testing.T) {
	// The failure this guards: an OIDC token carries the identity in `sub` and
	// nothing in `user_id`. Code reading UserID directly records every such
	// caller as anonymous, and an audit row loses the one field that names who
	// acted.
	tests := []struct {
		name   string
		claims *Claims
		want   string
	}{
		{"own token", &Claims{UserID: "user-1"}, "user-1"},
		{"oidc token", &Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "uuid-9"}}, "uuid-9"},
		{"both, user_id wins", &Claims{
			UserID:           "user-1",
			RegisteredClaims: jwt.RegisteredClaims{Subject: "uuid-9"},
		}, "user-1"},
		{"neither", &Claims{}, ""},
		{"nil claims", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.claims.SubjectID())
		})
	}
}

// ---------- roles ----------

func TestHasRole_ReadsBothTheSingularAndPluralClaim(t *testing.T) {
	assert.True(t, (&Claims{Role: "admin"}).HasRole("admin"))
	assert.True(t, (&Claims{Roles: []string{"learner", "facilitator"}}).HasRole("facilitator"))
	assert.False(t, (&Claims{Role: "admin"}).HasRole("super_admin"))
	assert.False(t, (*Claims)(nil).HasRole("admin"))

	// An empty role must never match, or a token with no role satisfies a
	// check that forgot to supply one.
	assert.False(t, (&Claims{}).HasRole(""))
}

func TestHasAnyRole_EmptyRequirementDenies(t *testing.T) {
	c := &Claims{Roles: []string{"learner"}}
	assert.True(t, c.HasAnyRole("admin", "learner"))
	assert.False(t, c.HasAnyRole("admin", "facilitator"))

	// An empty list is a bug at the call site far more often than it is an
	// intentional "anyone may pass"; denying makes it a failed request rather
	// than an open endpoint.
	assert.False(t, c.HasAnyRole())
}

func TestAllRoles_UnionsAndDeduplicates(t *testing.T) {
	c := &Claims{Role: "admin", Roles: []string{"admin", "learner", ""}}
	assert.Equal(t, []string{"admin", "learner"}, c.AllRoles())
	assert.Nil(t, (*Claims)(nil).AllRoles())
}

// ---------- RoleLadder ----------

var testLadder = RoleLadder{"super_admin", "admin", "facilitator", "learner", "normal_user"}

func TestRoleLadder_HighestPicksMostAuthoritative(t *testing.T) {
	// Order in the token must not matter — only the ladder's order does.
	assert.Equal(t, "admin", testLadder.Highest(&Claims{Roles: []string{"learner", "admin"}}))
	assert.Equal(t, "admin", testLadder.Highest(&Claims{Roles: []string{"admin", "learner"}}))
	assert.Equal(t, "", testLadder.Highest(&Claims{Roles: []string{"unlisted"}}))
	assert.Equal(t, "", testLadder.Highest(nil))
}

func TestRoleLadder_HasHigherOrEqual(t *testing.T) {
	admin := &Claims{Roles: []string{"admin"}}

	assert.True(t, testLadder.HasHigherOrEqual(admin, "learner"), "admin outranks learner")
	assert.True(t, testLadder.HasHigherOrEqual(admin, "admin"), "equal passes")
	assert.False(t, testLadder.HasHigherOrEqual(admin, "super_admin"))

	// A required role the ladder does not list is a typo at the call site.
	// Resolving a typo to "allow" turns a mistake into an open door.
	assert.False(t, testLadder.HasHigherOrEqual(admin, "amdin"))
	assert.False(t, testLadder.HasHigherOrEqual(&Claims{Roles: []string{"unlisted"}}, "learner"))
}

func TestRoleLadder_Precedence(t *testing.T) {
	assert.Equal(t, 0, testLadder.Precedence("super_admin"))
	assert.Equal(t, 4, testLadder.Precedence("normal_user"))
	assert.Equal(t, -1, testLadder.Precedence("nope"))
}

// ---------- token in context ----------

func TestTokenFromContext(t *testing.T) {
	assert.Equal(t, "", TokenFromContext(context.Background()))
	assert.Equal(t, "abc", TokenFromContext(WithToken(context.Background(), "abc")))

	// Storing an empty token would make a caller believe it has a credential
	// to forward, and it would forward "".
	assert.Equal(t, context.Background(), WithToken(context.Background(), ""))
}
