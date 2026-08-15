package apperrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
