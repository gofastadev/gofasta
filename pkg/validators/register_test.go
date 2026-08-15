package validators

import (
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterCommonValidators(t *testing.T) {
	v := validator.New()

	// Should not panic with nil db
	RegisterCommonValidators(v, nil)

	// Verify the validators are registered by checking uuid4_valid and is_valid_url
	type Input struct {
		URL string `validate:"is_valid_url"`
	}

	err := v.Struct(Input{URL: "https://example.com"})
	assert.NoError(t, err)

	err = v.Struct(Input{URL: "not-a-url"})
	assert.Error(t, err)
}

func TestRegisterTranslation(t *testing.T) {
	av := NewAppValidator(nil)

	av.Validate.RegisterValidation("custom_tag", func(fl validator.FieldLevel) bool {
		return false // always fails
	})
	RegisterTranslation(av.Validate, av.Trans, "custom_tag", "{0} failed custom validation")

	type Input struct {
		Field string `validate:"custom_tag"`
	}

	errs := av.ValidateStruct(Input{Field: "test"})
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "failed custom validation")
}

// TestValidatorHasRegisteredTags_NonDB verifies that non-database-dependent
// validator tags are registered and can be invoked without panicking.
// Database-dependent tags (is_record_deletable, is_record_exist_by_name_for_conflict,
// does_record_exist_by_id_for_verification) are skipped because they require a real
// *gorm.DB instance to execute.
func TestValidatorHasRegisteredTags_NonDB(t *testing.T) {
	av := NewAppValidator(nil)
	v := av.Validate

	// Only test tags that do NOT require a database connection.
	tags := []string{"uuid4_valid", "is_valid_url"}

	for _, tag := range tags {
		t.Run(tag, func(t *testing.T) {
			assert.NotNil(t, v, "validator should not be nil")

			typ := reflect.StructOf([]reflect.StructField{
				{
					Name: "Field",
					Type: reflect.TypeOf(""),
					Tag:  reflect.StructTag(`validate:"` + tag + `"`),
				},
			})
			val := reflect.New(typ).Elem()
			val.Field(0).SetString("test-value")

			// Should not panic - the tag is recognized
			_ = v.Struct(val.Addr().Interface())
		})
	}
}
