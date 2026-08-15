package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CamelToSnake ---

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple camelCase", "camelCase", "camel_case"},
		{"PascalCase", "PascalCase", "pascal_case"},
		{"multiple words", "createdAt", "created_at"},
		{"already snake", "already_snake", "already_snake"},
		{"single word lowercase", "hello", "hello"},
		{"single uppercase letter", "A", "a"},
		{"empty string", "", ""},
		{"all uppercase consecutive", "ID", "id"},
		{"mixed with numbers", "field2Name", "field2_name"},
		{"updatedAt", "updatedAt", "updated_at"},
		{"sortByField", "sortByField", "sort_by_field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CamelToSnake(tt.input))
		})
	}
}

// --- ParseIDStringIsValidUUID ---

func TestParseIDStringIsValidUUID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID generated", uuid.New().String(), false},
		{"invalid UUID", "not-a-uuid", true},
		{"empty string", "", true},
		{"partial UUID", "550e8400-e29b-41d4", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIDStringIsValidUUID(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, result.String())
			}
		})
	}
}

// --- ConvertUpperCamelToLowerCamel ---

func TestConvertUpperCamelToLowerCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PascalCase to camelCase", "CreatedAt", "createdAt"},
		{"single word", "Name", "name"},
		{"already lower camel", "createdAt", "createdAt"},
		{"empty string", "", ""},
		{"ID special case", "ID", "id"},
		{"single char", "A", "a"},
		{"multiple words", "SortByField", "sortByField"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ConvertUpperCamelToLowerCamel(tt.input))
		})
	}
}
