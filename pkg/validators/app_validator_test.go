package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAppValidator tests that NewAppValidator creates a valid instance.
// We pass nil for db since we skip database-dependent validators in these tests.
// NewAppValidator will register db-dependent validators with nil db, which is fine
// as long as we don't trigger those validators.
func TestNewAppValidator(t *testing.T) {
	av := NewAppValidator(nil)
	assert.NotNil(t, av)
	assert.NotNil(t, av.Validate)
	assert.NotNil(t, av.Trans)
}

func TestValidateStruct_Valid(t *testing.T) {
	av := NewAppValidator(nil)

	type Input struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	input := Input{Name: "John", Email: "john@example.com"}
	errs := av.ValidateStruct(input)
	assert.Empty(t, errs)
}

func TestValidateStruct_Invalid(t *testing.T) {
	av := NewAppValidator(nil)

	type Input struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	tests := []struct {
		name          string
		input         Input
		expectedCount int
	}{
		{
			name:          "missing all required fields",
			input:         Input{},
			expectedCount: 2,
		},
		{
			name:          "missing name only",
			input:         Input{Email: "test@example.com"},
			expectedCount: 1,
		},
		{
			name:          "invalid email format",
			input:         Input{Name: "John", Email: "not-an-email"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := av.ValidateStruct(tt.input)
			assert.Len(t, errs, tt.expectedCount)
			for _, e := range errs {
				assert.NotEmpty(t, e.Message)
				assert.NotNil(t, e.FieldName)
			}
		})
	}
}

func TestValidateStruct_NoValidationTags(t *testing.T) {
	av := NewAppValidator(nil)

	type NoTags struct {
		Foo string
		Bar int
	}

	errs := av.ValidateStruct(NoTags{})
	assert.Empty(t, errs)
}

func TestValidateStruct_FieldNameIsLowerCamel(t *testing.T) {
	av := NewAppValidator(nil)

	type Input struct {
		FirstName string `validate:"required"`
	}

	errs := av.ValidateStruct(Input{})
	require.Len(t, errs, 1)
	require.NotNil(t, errs[0].FieldName)

	// The field name should be lowerCamelCase (firstName, not FirstName)
	fieldName := *errs[0].FieldName
	assert.True(t, len(fieldName) > 0)
	// First character should be lowercase
	assert.Equal(t, string(fieldName[0]), string(rune(fieldName[0]|0x20)))
}
