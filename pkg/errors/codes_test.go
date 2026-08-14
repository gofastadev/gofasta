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

func TestErrorTypeCode(t *testing.T) {
	tests := []struct {
		errType ErrorType
		want    string
	}{
		{NotFound, "NOT_FOUND"},
		{Validation, "VALIDATION_FAILED"},
		{Conflict, "CONFLICT"},
		{Unauthorized, "UNAUTHORIZED"},
		{Forbidden, "FORBIDDEN"},
		{BadRequest, "BAD_REQUEST"},
		{PreconditionFailed, "PRECONDITION_FAILED"},
		{PreconditionRequired, "PRECONDITION_REQUIRED"},
		{Internal, "INTERNAL"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.errType.Code())
			assert.Equal(t, tt.want, tt.errType.String())
		})
	}
}

// An ErrorType outside the declared set must not report an empty code: a client
// switching on it should land in its "something went wrong" branch rather than
// in no branch at all.
func TestErrorTypeCodeUnknown(t *testing.T) {
	assert.Equal(t, CodeInternal, ErrorType(999).Code())
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
			ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
			result := GraphQLErrorPresenter(ctx, tt.err)
			require.NotNil(t, result)
			assert.Equal(t, tt.want, result.Extensions["code"])
		})
	}
}

// The classification has to survive wrapping — services routinely add context
// with %w on the way out.
func TestGraphQLErrorPresenterWrappedKeepsCode(t *testing.T) {
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
	wrapped := fmt.Errorf("loading profile: %w", NewNotFound("user not found", nil))

	result := GraphQLErrorPresenter(ctx, wrapped)
	require.NotNil(t, result)
	assert.Equal(t, CodeNotFound, result.Extensions["code"])
	assert.Equal(t, "user not found", result.Message)
}

// Details are how field-level validation failures reach a client that has no
// in-band errors array to put them in.
func TestGraphQLErrorPresenterCarriesDetails(t *testing.T) {
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
	details := []map[string]string{{"fieldName": "email", "message": "must be a valid email"}}

	result := GraphQLErrorPresenter(ctx, NewValidation("validation failed", details))
	require.NotNil(t, result)
	assert.Equal(t, CodeValidationFailed, result.Extensions["code"])
	assert.Equal(t, details, result.Extensions["details"])
}

// An AppError without Details must not invent an empty `details` key — a client
// checking for its presence would read that as "there were field errors".
func TestGraphQLErrorPresenterOmitsAbsentDetails(t *testing.T) {
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)

	result := GraphQLErrorPresenter(ctx, NewNotFound("user not found", nil))
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
			ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
			result := GraphQLErrorPresenter(ctx, tt.err)
			require.NotNil(t, result)
			assert.Equal(t, CodeInternal, result.Extensions["code"])
			assert.Equal(t, tt.wantMsg, result.Message)
		})
	}
}

// Extensions a resolver already attached must survive: the presenter adds to
// the map, it does not replace it.
func TestGraphQLErrorPresenterPreservesExistingExtensions(t *testing.T) {
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
	gqlErr := &gqlerror.Error{
		Message:    "not found: resource missing",
		Extensions: map[string]any{"requestId": "abc-123"},
	}

	result := GraphQLErrorPresenter(ctx, gqlErr)
	require.NotNil(t, result)
	assert.Equal(t, "abc-123", result.Extensions["requestId"])
	assert.Equal(t, CodeInternal, result.Extensions["code"])
}
