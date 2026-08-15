package apperrors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "message only",
			err:      &AppError{Message: "something went wrong"},
			expected: "something went wrong",
		},
		{
			name:     "message with internal error",
			err:      &AppError{Message: "not found", Internal: errors.New("db error")},
			expected: "not found: db error",
		},
		{
			name:     "empty message with internal error",
			err:      &AppError{Message: "", Internal: errors.New("oops")},
			expected: ": oops",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	appErr := &AppError{Message: "outer", Internal: inner}

	assert.Equal(t, inner, appErr.Unwrap())
	assert.True(t, errors.Is(appErr, inner))

	appErrNoInternal := &AppError{Message: "no internal"}
	assert.Nil(t, appErrNoInternal.Unwrap())
}

func TestErrorConstructors(t *testing.T) {
	inner := errors.New("cause")

	tests := []struct {
		name        string
		constructor func() *AppError
		errType     ErrorType
		message     string
		hasInternal bool
		hasDetails  bool
	}{
		{
			name:        "NewNotFound",
			constructor: func() *AppError { return NewNotFound("item not found", inner) },
			errType:     NotFound,
			message:     "item not found",
			hasInternal: true,
		},
		{
			name:        "NewNotFound nil internal",
			constructor: func() *AppError { return NewNotFound("item not found", nil) },
			errType:     NotFound,
			message:     "item not found",
			hasInternal: false,
		},
		{
			name:        "NewValidation",
			constructor: func() *AppError { return NewValidation("invalid input", map[string]string{"field": "required"}) },
			errType:     Validation,
			message:     "invalid input",
			hasDetails:  true,
		},
		{
			name:        "NewInternal",
			constructor: func() *AppError { return NewInternal("server error", inner) },
			errType:     Internal,
			message:     "server error",
			hasInternal: true,
		},
		{
			name:        "NewConflict",
			constructor: func() *AppError { return NewConflict("duplicate", inner) },
			errType:     Conflict,
			message:     "duplicate",
			hasInternal: true,
		},
		{
			name:        "NewUnauthorized",
			constructor: func() *AppError { return NewUnauthorized("not authed", inner) },
			errType:     Unauthorized,
			message:     "not authed",
			hasInternal: true,
		},
		{
			name:        "NewForbidden",
			constructor: func() *AppError { return NewForbidden("access denied", inner) },
			errType:     Forbidden,
			message:     "access denied",
			hasInternal: true,
		},
		{
			name:        "NewBadRequest",
			constructor: func() *AppError { return NewBadRequest("bad input", []string{"error1"}) },
			errType:     BadRequest,
			message:     "bad input",
			hasDetails:  true,
		},
		{
			name:        "NewPreconditionFailed",
			constructor: func() *AppError { return NewPreconditionFailed("etag mismatch", inner) },
			errType:     PreconditionFailed,
			message:     "etag mismatch",
			hasInternal: true,
		},
		{
			name:        "NewPreconditionRequired",
			constructor: func() *AppError { return NewPreconditionRequired("If-Match required", []string{"missing header"}) },
			errType:     PreconditionRequired,
			message:     "If-Match required",
			hasDetails:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			require.NotNil(t, err)
			assert.Equal(t, tt.errType, err.Type)
			assert.Equal(t, tt.message, err.Message)

			if tt.hasInternal {
				assert.NotNil(t, err.Internal)
			}
			if tt.hasDetails {
				assert.NotNil(t, err.Details)
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected int
	}{
		{"NotFound", &AppError{Type: NotFound}, http.StatusNotFound},
		{"Validation", &AppError{Type: Validation}, http.StatusUnprocessableEntity},
		{"Conflict", &AppError{Type: Conflict}, http.StatusConflict},
		{"Unauthorized", &AppError{Type: Unauthorized}, http.StatusUnauthorized},
		{"Forbidden", &AppError{Type: Forbidden}, http.StatusForbidden},
		{"BadRequest", &AppError{Type: BadRequest}, http.StatusBadRequest},
		{"PreconditionFailed", &AppError{Type: PreconditionFailed}, http.StatusPreconditionFailed},
		{"PreconditionRequired", &AppError{Type: PreconditionRequired}, http.StatusPreconditionRequired},
		{"Internal", &AppError{Type: Internal}, http.StatusInternalServerError},
		{"Unknown type defaults to 500", &AppError{Type: ErrorType(999)}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HTTPStatus(tt.err))
		})
	}
}

func TestErrorType_Constants(t *testing.T) {
	// Verify iota ordering. New entries MUST be appended at the end
	// because existing values are stable API (clierr.Error JSON
	// payloads bake these into integration tests downstream).
	assert.Equal(t, ErrorType(0), NotFound)
	assert.Equal(t, ErrorType(1), Validation)
	assert.Equal(t, ErrorType(2), Conflict)
	assert.Equal(t, ErrorType(3), Internal)
	assert.Equal(t, ErrorType(4), Unauthorized)
	assert.Equal(t, ErrorType(5), Forbidden)
	assert.Equal(t, ErrorType(6), BadRequest)
	assert.Equal(t, ErrorType(7), PreconditionFailed)
	assert.Equal(t, ErrorType(8), PreconditionRequired)
}

func TestAppError_ImplementsErrorInterface(t *testing.T) {
	var err error = &AppError{Message: "test"}
	assert.NotNil(t, err)
	assert.Equal(t, "test", err.Error())
}

func TestErrorsAs(t *testing.T) {
	appErr := NewInternal("wrapped", errors.New("cause"))
	var target *AppError
	assert.True(t, errors.As(appErr, &target))
	assert.Equal(t, Internal, target.Type)
}
