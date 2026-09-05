package apperrors

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// presenterContext builds the response context gqlgen's DefaultErrorPresenter
// expects. Every test here needs one and none of them care about its contents.
func presenterContext() context.Context {
	return graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
}

// DefaultErrorPresenter returns nil for exactly one input — a nil error. The
// presenter has to hand that back untouched; dereferencing it to read .Message
// would panic on a resolver that returned no error at all.
func TestGraphQLErrorPresenter_NilError(t *testing.T) {
	assert.Nil(t, GraphQLErrorPresenter(presenterContext(), nil))
}

func TestGraphQLErrorPresenter_AppError(t *testing.T) {
	appErr := NewValidation("validation failed", nil)

	result := GraphQLErrorPresenter(presenterContext(), appErr)
	require.NotNil(t, result)
	assert.Equal(t, "validation failed", result.Message)
}

func TestGraphQLErrorPresenter_SafePrefixes(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
	}{
		{"not found prefix", "not found: user 123"},
		{"validation failed prefix", "validation failed: email required"},
		{"invalid prefix", "invalid token provided"},
		{"already exists prefix", "already exists: duplicate entry"},
		{"unauthorized prefix", "unauthorized access attempt"},
		{"forbidden prefix", "forbidden resource"},
		{"authentication required prefix", "authentication required to proceed"},
		{"permission denied prefix", "permission denied for this action"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("%s", tt.errMsg)
			result := GraphQLErrorPresenter(presenterContext(), err)
			require.NotNil(t, result)
			assert.Equal(t, tt.errMsg, result.Message)
		})
	}
}

func TestGraphQLErrorPresenter_UnsafeError(t *testing.T) {
	err := fmt.Errorf("database connection failed: dial tcp 127.0.0.1:5432")

	result := GraphQLErrorPresenter(presenterContext(), err)
	require.NotNil(t, result)
	assert.Equal(t, "an internal error occurred", result.Message)
}

func TestGraphQLErrorPresenter_WrappedAppError(t *testing.T) {
	appErr := NewNotFound("user not found", nil)
	wrapped := fmt.Errorf("service error: %w", appErr)

	result := GraphQLErrorPresenter(presenterContext(), wrapped)
	require.NotNil(t, result)
	assert.Equal(t, "user not found", result.Message)
}

func TestGraphQLErrorPresenter_GqlError(t *testing.T) {
	gqlErr := &gqlerror.Error{Message: "not found: resource missing"}

	result := GraphQLErrorPresenter(presenterContext(), gqlErr)
	require.NotNil(t, result)
	assert.Equal(t, "not found: resource missing", result.Message)
}

// Every constructor must produce an error the client can classify. This is the
// contract the scaffold's GraphQL schema documents.
func TestGraphQLErrorPresenterStampsCode(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		want string
	}{
		{"not found", NewNotFound("user not found", nil), CodeNotFound},
		{"validation", NewValidation("validation failed", nil), CodeValidationFailed},
		{"conflict", NewConflict("already exists", nil), CodeConflict},
		{"unauthorized", NewUnauthorized("authentication required", nil), CodeUnauthorized},
		{"forbidden", NewForbidden("permission denied", nil), CodeForbidden},
		{"bad request", NewBadRequest("invalid cursor", nil), CodeBadRequest},
		{"internal", NewInternal("boom", nil), CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GraphQLErrorPresenter(presenterContext(), tt.err)
			require.NotNil(t, result)
			assert.Equal(t, tt.want, result.Extensions["code"])
		})
	}
}

// The classification has to survive wrapping — services routinely add context
// with %w on the way out.
func TestGraphQLErrorPresenterWrappedKeepsCode(t *testing.T) {
	wrapped := fmt.Errorf("loading profile: %w", NewNotFound("user not found", nil))

	result := GraphQLErrorPresenter(presenterContext(), wrapped)
	require.NotNil(t, result)
	assert.Equal(t, CodeNotFound, result.Extensions["code"])
	assert.Equal(t, "user not found", result.Message)
}

// Details are how field-level validation failures reach a client that has no
// in-band errors array to put them in.
func TestGraphQLErrorPresenterCarriesDetails(t *testing.T) {
	details := []map[string]string{{"fieldName": "email", "message": "must be a valid email"}}

	result := GraphQLErrorPresenter(presenterContext(), NewValidation("validation failed", details))
	require.NotNil(t, result)
	assert.Equal(t, CodeValidationFailed, result.Extensions["code"])
	assert.Equal(t, details, result.Extensions["details"])
}

// An AppError without Details must not invent an empty `details` key — a client
// checking for its presence would read that as "there were field errors".
func TestGraphQLErrorPresenterOmitsAbsentDetails(t *testing.T) {
	result := GraphQLErrorPresenter(presenterContext(), NewNotFound("user not found", nil))
	require.NotNil(t, result)
	_, present := result.Extensions["details"]
	assert.False(t, present)
}

// A bare error never passed through the taxonomy, so there is nothing to
// classify it as. INTERNAL is the honest answer for both the sanitized and the
// safe-prefix path — inventing NOT_FOUND from the words "not found" would be
// guessing on the client's behalf.
func TestGraphQLErrorPresenterBareErrorsAreInternal(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"safe prefix passes message through", errors.New("not found: user 123"), "not found: user 123"},
		{"unsafe message is replaced", errors.New("dial tcp 127.0.0.1:5432: refused"), "an internal error occurred"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GraphQLErrorPresenter(presenterContext(), tt.err)
			require.NotNil(t, result)
			assert.Equal(t, CodeInternal, result.Extensions["code"])
			assert.Equal(t, tt.wantMsg, result.Message)
		})
	}
}

// Extensions a resolver already attached must survive: the presenter adds to
// the map, it does not replace it.
func TestGraphQLErrorPresenterPreservesExistingExtensions(t *testing.T) {
	gqlErr := &gqlerror.Error{
		Message:    "not found: resource missing",
		Extensions: map[string]any{"requestId": "abc-123"},
	}

	result := GraphQLErrorPresenter(presenterContext(), gqlErr)
	require.NotNil(t, result)
	assert.Equal(t, "abc-123", result.Extensions["requestId"])
	assert.Equal(t, CodeInternal, result.Extensions["code"])
}

// TestGraphQLErrorPresenterTranslatesGqlgenCodes covers the failures gqlgen
// classifies before a resolver runs.
//
// gqlgen sets its own code extension for a query it cannot parse and for
// variables that violate the schema. The presenter used to overwrite both
// with INTERNAL, which tells a client to retry or show an outage page for a
// request that will never succeed as written — the client has to change the
// query, and only the code says so.
func TestGraphQLErrorPresenterTranslatesGqlgenCodes(t *testing.T) {
	for name, tc := range map[string]struct {
		gqlgenCode string
		want       string
	}{
		"a variable that violates the schema": {"GRAPHQL_VALIDATION_FAILED", CodeValidationFailed},
		"a query that will not parse":         {"GRAPHQL_PARSE_FAILED", CodeBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			err := &gqlerror.Error{
				Message:    "must be defined",
				Extensions: map[string]any{"code": tc.gqlgenCode},
			}

			got := GraphQLErrorPresenter(presenterContext(), err)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.Extensions["code"])
			// The message is gqlgen's own and names the offending field, so
			// it survives rather than being replaced with the generic text.
			assert.Equal(t, "must be defined", got.Message)
		})
	}
}

// TestGraphQLErrorPresenterIgnoresUnknownGqlgenCodes pins the boundary.
//
// Only the two codes that mean something specific to a client are
// translated; anything else keeps the classification the presenter would
// have made on its own, because inventing a mapping would be guessing.
func TestGraphQLErrorPresenterIgnoresUnknownGqlgenCodes(t *testing.T) {
	err := &gqlerror.Error{
		Message:    "something went sideways",
		Extensions: map[string]any{"code": "SOME_FUTURE_GQLGEN_CODE"},
	}

	got := GraphQLErrorPresenter(presenterContext(), err)
	require.NotNil(t, got)
	assert.Equal(t, CodeInternal, got.Extensions["code"])
}

// TestGraphQLErrorPresenterPrefersAppErrorClassification pins the precedence.
//
// An AppError is the application's own classification and outranks gqlgen's:
// a resolver that returned NOT_FOUND must not have it rewritten because the
// error traveled inside a gqlerror carrying a code of its own.
func TestGraphQLErrorPresenterPrefersAppErrorClassification(t *testing.T) {
	got := GraphQLErrorPresenter(presenterContext(), NewNotFound("course not found", nil))
	require.NotNil(t, got)
	assert.Equal(t, CodeNotFound, got.Extensions["code"])
}
