package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
