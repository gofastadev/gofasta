package utils

import (
	"errors"
	"testing"
	"testing/iotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofastadev/gofasta/pkg/types"
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

// --- PreparePaginating ---

func TestPreparePaginating_GetLimit(t *testing.T) {
	tests := []struct {
		name     string
		pager    PreparePaginating
		expected int
	}{
		{"nil PageFilters returns default 10", PreparePaginating{}, 10},
		{"nil Limit returns default 10", PreparePaginating{PageFilters: &types.TPaginationInputDto{}}, 10},
		{"custom limit", PreparePaginating{PageFilters: &types.TPaginationInputDto{Limit: intPtr(25)}}, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetLimit())
		})
	}
}

func TestPreparePaginating_GetPage(t *testing.T) {
	tests := []struct {
		name     string
		pager    PreparePaginating
		expected int
	}{
		{"nil PageFilters returns default 1", PreparePaginating{}, 1},
		{"nil Page returns default 1", PreparePaginating{PageFilters: &types.TPaginationInputDto{}}, 1},
		{"custom page", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(5)}}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetPage())
		})
	}
}

func TestPreparePaginating_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		pager    PreparePaginating
		expected int
	}{
		{"default page 1, limit 10 => offset 0", PreparePaginating{}, 0},
		{"page 2, limit 10 => offset 10", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(2), Limit: intPtr(10)}}, 10},
		{"page 3, limit 5 => offset 10", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(3), Limit: intPtr(5)}}, 10},
		{"page 1, limit 20 => offset 0", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(1), Limit: intPtr(20)}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetOffset())
		})
	}
}

func TestPreparePaginating_GetSort(t *testing.T) {
	desc := types.SortOrientationDesc
	asc := types.SortOrientationAsc
	bogusOrientation := types.SortOrientation("DROP TABLE users")

	tests := []struct {
		name     string
		pager    PreparePaginating
		expected string
	}{
		{
			"nil sorting defaults to created_at DESC",
			PreparePaginating{},
			"created_at DESC",
		},
		{
			"empty sortByField defaults to created_at",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: ""}},
			"created_at DESC",
		},
		{
			"custom field and DESC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "updatedAt", SortOrientation: &desc}},
			"updated_at DESC",
		},
		{
			"custom field and ASC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "userName", SortOrientation: &asc}},
			"user_name ASC",
		},
		{
			"custom field, nil orientation defaults to DESC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "name"}},
			"name DESC",
		},
		// SQL-injection regression: any sortField that doesn't match a
		// plain identifier ([a-zA-Z_][a-zA-Z0-9_]*) is rejected and
		// replaced with "created_at" so the request still completes
		// safely. Without this, GORM's Order() would happily execute
		// arbitrary SQL embedded in the query string.
		{
			"injection via semicolon falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "id; DROP TABLE users;--"}},
			"created_at DESC",
		},
		{
			"injection via space falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "id ASC, name"}},
			"created_at DESC",
		},
		{
			"injection via quote falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: `id"--`}},
			"created_at DESC",
		},
		{
			"injection via leading digit falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "1=1"}},
			"created_at DESC",
		},
		// Invalid orientation (e.g. unmarshaled from a malicious JSON
		// payload) falls back to DESC; we only honor the canonical
		// ASC/DESC enum values.
		{
			"invalid orientation falls back to DESC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "name", SortOrientation: &bogusOrientation}},
			"name DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetSort())
		})
	}
}

// --- GeneratePassword ---

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
		{"length 1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := GeneratePassword(tt.length)
			require.NoError(t, err)
			assert.Len(t, pwd, tt.length)
		})
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	// Two generated passwords should almost certainly differ
	pwd1, err := GeneratePassword(32)
	require.NoError(t, err)
	pwd2, err := GeneratePassword(32)
	require.NoError(t, err)
	assert.NotEqual(t, pwd1, pwd2)
}

func TestGeneratePassword_UsesCharset(t *testing.T) {
	pwd, err := GeneratePassword(100)
	require.NoError(t, err)
	for _, c := range pwd {
		assert.Contains(t, Charset, string(c), "password contains character not in Charset: %c", c)
	}
}

func TestGeneratePassword_RandError(t *testing.T) {
	oldReader := cryptoRandReader
	cryptoRandReader = iotest.ErrReader(errors.New("entropy failure"))
	defer func() { cryptoRandReader = oldReader }()

	_, err := GeneratePassword(8)
	require.Error(t, err)
}

func TestRandomChar_Success(t *testing.T) {
	c, err := randomChar()
	require.NoError(t, err)
	assert.Contains(t, Charset, string(c))
}

// --- ConvertStructToMap ---

func TestConvertStructToMap(t *testing.T) {
	t.Run("struct with pointer fields", func(t *testing.T) {
		name := "test"
		age := 25
		type Sample struct {
			Name *string
			Age  *int
		}
		result := ConvertStructToMap(Sample{Name: &name, Age: &age})
		assert.Equal(t, &name, result["name"])
		assert.Equal(t, &age, result["age"])
	})

	t.Run("nil pointer fields are skipped", func(t *testing.T) {
		type Sample struct {
			Name *string
			Age  *int
		}
		result := ConvertStructToMap(Sample{})
		assert.Empty(t, result)
	})

	t.Run("pointer to struct", func(t *testing.T) {
		name := "test"
		type Sample struct {
			Name *string
		}
		result := ConvertStructToMap(&Sample{Name: &name})
		assert.Equal(t, &name, result["name"])
	})

	t.Run("non-struct returns nil", func(t *testing.T) {
		result := ConvertStructToMap("not a struct")
		assert.Nil(t, result)
	})

	t.Run("non-pointer non-nillable fields are skipped", func(t *testing.T) {
		type Sample struct {
			Name  string
			Count int
			Items *string
		}
		item := "hello"
		result := ConvertStructToMap(Sample{Name: "test", Count: 5, Items: &item})
		// Only pointer/interface/map/slice/chan/func fields are included
		assert.NotContains(t, result, "name")
		assert.NotContains(t, result, "count")
		assert.Contains(t, result, "items")
	})

	t.Run("camelCase field names are converted to snake_case", func(t *testing.T) {
		firstName := "John"
		type Sample struct {
			FirstName *string
		}
		result := ConvertStructToMap(Sample{FirstName: &firstName})
		assert.Contains(t, result, "first_name")
	})

	t.Run("slice fields", func(t *testing.T) {
		type Sample struct {
			Tags []string
		}
		result := ConvertStructToMap(Sample{Tags: []string{"a", "b"}})
		assert.Contains(t, result, "tags")
	})

	t.Run("nil slice is skipped", func(t *testing.T) {
		type Sample struct {
			Tags []string
		}
		result := ConvertStructToMap(Sample{})
		assert.NotContains(t, result, "tags")
	})

	t.Run("map fields", func(t *testing.T) {
		type Sample struct {
			Meta map[string]string
		}
		result := ConvertStructToMap(Sample{Meta: map[string]string{"key": "val"}})
		assert.Contains(t, result, "meta")
	})
}

// --- helpers ---

func intPtr(v int) *int {
	return &v
}
