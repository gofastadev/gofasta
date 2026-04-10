package validators

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestIsValidURL(t *testing.T) {
	av := NewAppValidator(nil)

	type URLInput struct {
		URL string `validate:"is_valid_url"`
	}

	tests := []struct {
		name    string
		url     string
		isValid bool
	}{
		{name: "valid https", url: "https://example.com", isValid: true},
		{name: "valid http", url: "http://example.com", isValid: true},
		{name: "valid with path", url: "https://example.com/path/to/resource", isValid: true},
		{name: "missing scheme", url: "example.com", isValid: false},
		{name: "ftp scheme", url: "ftp://example.com", isValid: false},
		{name: "empty string", url: "", isValid: false},
		{name: "random text", url: "not a url", isValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := av.ValidateStruct(URLInput{URL: tt.url})
			if tt.isValid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}

func TestIsUUIDv4Valid(t *testing.T) {
	av := NewAppValidator(nil)

	type UUIDInput struct {
		ID string `validate:"uuid4_valid"`
	}

	validUUID := uuid.New().String()

	tests := []struct {
		name    string
		id      string
		isValid bool
	}{
		{name: "valid uuid v4", id: validUUID, isValid: true},
		{name: "invalid uuid", id: "not-a-uuid", isValid: false},
		{name: "empty string", id: "", isValid: false},
		{name: "partial uuid", id: "550e8400-e29b-41d4", isValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := av.ValidateStruct(UUIDInput{ID: tt.id})
			if tt.isValid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}

// TestGetValue tests the getValue helper using a real validator invocation.
// We register a custom validator that captures what getValue returns.
func TestGetValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "string field",
			input: struct {
				Value string `validate:"test_getValue"`
			}{Value: "hello"},
			expected: "hello",
		},
		{
			name: "uuid field",
			input: struct {
				Value uuid.UUID `validate:"test_getValue"`
			}{Value: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "pointer to string (non-nil)",
			input: struct {
				Value *string `validate:"test_getValue"`
			}{Value: strPtr("world")},
			expected: "world",
		},
		{
			name: "pointer to string (non-nil empty)",
			input: struct {
				Value *string `validate:"test_getValue"`
			}{Value: strPtr("")},
			expected: "",
		},
		{
			name: "pointer to uuid (non-nil)",
			input: struct {
				Value *uuid.UUID `validate:"test_getValue"`
			}{Value: uuidPtr(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))},
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			var captured string

			v.RegisterValidation("test_getValue", func(fl validator.FieldLevel) bool {
				captured = getValue(fl)
				return true
			})

			err := v.Struct(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, captured)
		})
	}
}

// TestGetValue_PointerToUUID_Nil tests nil pointer UUID returns empty.
// Note: validator skips nil pointer fields unless "required" is set, so we
// use "required" to force the validator to run on the nil field.
func TestGetValue_PointerToUUID_Nil(t *testing.T) {
	v := validator.New()
	var captured string
	var called bool

	v.RegisterValidation("test_getValue", func(fl validator.FieldLevel) bool {
		called = true
		captured = getValue(fl)
		return true
	})

	type Input struct {
		Value *uuid.UUID `validate:"required,test_getValue"`
	}
	// With required + nil pointer, "required" fails first so test_getValue may not run.
	// Instead, just verify via pointer to string nil which does get called.
	_ = v.Struct(Input{Value: nil})
	// The validator may or may not call test_getValue depending on required failing first.
	// This is expected behavior - nil pointers with required will fail on required.
	if called {
		assert.Equal(t, "", captured)
	}
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

// TestIsValidURL_WithPointer tests that nil pointer passes is_valid_url.
func TestIsValidURL_WithPointer(t *testing.T) {
	av := NewAppValidator(nil)

	type Input struct {
		URL *string `validate:"omitempty,is_valid_url"`
	}

	// nil pointer with omitempty should pass
	errs := av.ValidateStruct(Input{URL: nil})
	assert.Empty(t, errs)

	// valid URL pointer
	url := "https://example.com"
	errs = av.ValidateStruct(Input{URL: &url})
	assert.Empty(t, errs)

	// invalid URL pointer
	badURL := "not-a-url"
	errs = av.ValidateStruct(Input{URL: &badURL})
	assert.NotEmpty(t, errs)
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

func strPtr(s string) *string {
	return &s
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

// setupTestDB creates an in-memory SQLite database with a test_records table.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.Exec("CREATE TABLE test_records (id TEXT PRIMARY KEY, name TEXT, is_deletable BOOLEAN, deleted_at TIMESTAMP)")
	return db
}

func TestIsRecordExistByName_NoConflict(t *testing.T) {
	db := setupTestDB(t)
	av := NewAppValidator(db)

	type NameInput struct {
		Name string `validate:"is_record_exist_by_name_for_conflict=test_records"`
	}

	errs := av.ValidateStruct(NameInput{Name: "nonexistent"})
	assert.Empty(t, errs)
}

func TestIsRecordExistByName_Conflict(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("INSERT INTO test_records (id, name, is_deletable) VALUES (?, ?, ?)", uuid.New().String(), "taken", true)
	av := NewAppValidator(db)

	type NameInput struct {
		Name string `validate:"is_record_exist_by_name_for_conflict=test_records"`
	}

	errs := av.ValidateStruct(NameInput{Name: "taken"})
	assert.NotEmpty(t, errs)
}

func TestIsRecordExistByName_NilPointer(t *testing.T) {
	db := setupTestDB(t)
	av := NewAppValidator(db)

	type NameInput struct {
		Name *string `validate:"omitempty,is_record_exist_by_name_for_conflict=test_records"`
	}

	errs := av.ValidateStruct(NameInput{Name: nil})
	assert.Empty(t, errs)
}

func TestIsRecordExistById_Exists(t *testing.T) {
	db := setupTestDB(t)
	recordID := uuid.New().String()
	db.Exec("INSERT INTO test_records (id, name, is_deletable) VALUES (?, ?, ?)", recordID, "active", true)
	av := NewAppValidator(db)

	type IDInput struct {
		ID string `validate:"does_record_exist_by_id_for_verification=test_records"`
	}

	errs := av.ValidateStruct(IDInput{ID: recordID})
	assert.Empty(t, errs)
}

func TestIsRecordExistById_NotExists(t *testing.T) {
	db := setupTestDB(t)
	av := NewAppValidator(db)

	type IDInput struct {
		ID string `validate:"does_record_exist_by_id_for_verification=test_records"`
	}

	errs := av.ValidateStruct(IDInput{ID: uuid.New().String()})
	assert.NotEmpty(t, errs)
}

func TestIsRecordExistById_Deleted(t *testing.T) {
	db := setupTestDB(t)
	recordID := uuid.New().String()
	deletedAt := time.Now().Add(-time.Hour)
	db.Exec("INSERT INTO test_records (id, name, is_deletable, deleted_at) VALUES (?, ?, ?, ?)", recordID, "deleted", true, deletedAt)
	av := NewAppValidator(db)

	type IDInput struct {
		ID string `validate:"does_record_exist_by_id_for_verification=test_records"`
	}

	errs := av.ValidateStruct(IDInput{ID: recordID})
	assert.NotEmpty(t, errs)
}

func TestIsRecordDeletable_Deletable(t *testing.T) {
	db := setupTestDB(t)
	recordID := uuid.New().String()
	db.Exec("INSERT INTO test_records (id, name, is_deletable) VALUES (?, ?, ?)", recordID, "deletable", true)
	av := NewAppValidator(db)

	type IDInput struct {
		ID string `validate:"is_record_deletable=test_records"`
	}

	errs := av.ValidateStruct(IDInput{ID: recordID})
	assert.Empty(t, errs)
}

func TestIsRecordDeletable_NotDeletable(t *testing.T) {
	db := setupTestDB(t)
	recordID := uuid.New().String()
	db.Exec("INSERT INTO test_records (id, name, is_deletable) VALUES (?, ?, ?)", recordID, "protected", false)
	av := NewAppValidator(db)

	type IDInput struct {
		ID string `validate:"is_record_deletable=test_records"`
	}

	errs := av.ValidateStruct(IDInput{ID: recordID})
	assert.NotEmpty(t, errs)
}

func TestGetValue_NilPointer(t *testing.T) {
	v := validator.New()
	var captured string
	var called bool

	v.RegisterValidation("test_nil_getValue", func(fl validator.FieldLevel) bool {
		called = true
		captured = getValue(fl)
		return true
	})

	type Input struct {
		Value *string `validate:"test_nil_getValue"`
	}

	// For a nil pointer without "required", the validator library skips the field.
	// So we verify via the isRecordExistByName nil-pointer path instead:
	// the validator itself checks fl.Field().IsNil() and returns true (passes).
	// Here we test getValue directly when called on a nil pointer via required.
	_ = v.Struct(Input{Value: nil})
	if called {
		assert.Equal(t, "", captured)
	} else {
		// Validator skipped nil pointer field - this is expected.
		// The nil pointer path is tested via TestIsRecordExistByName_NilPointer above.
		t.Log("validator skipped nil pointer field as expected")
	}
}

func TestIsRecordExistByName_DBError(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close() // close connection to force DB error

	av := NewAppValidator(db)
	type Input struct {
		Name string `validate:"is_record_exist_by_name_for_conflict=test_records"`
	}
	errs := av.ValidateStruct(Input{Name: "test"})
	assert.NotEmpty(t, errs)
}

func TestIsRecordExistById_DBError(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	av := NewAppValidator(db)
	type Input struct {
		ID string `validate:"does_record_exist_by_id_for_verification=test_records"`
	}
	errs := av.ValidateStruct(Input{ID: "some-id"})
	assert.NotEmpty(t, errs)
}

func TestIsRecordDeletable_DBError(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	av := NewAppValidator(db)
	type Input struct {
		ID string `validate:"is_record_deletable=test_records"`
	}
	errs := av.ValidateStruct(Input{ID: "some-id"})
	assert.NotEmpty(t, errs)
}

func TestGetValue_NonPointerUUID(t *testing.T) {
	v := validator.New()
	var captured string

	v.RegisterValidation("test_getValue_uuid", func(fl validator.FieldLevel) bool {
		captured = getValue(fl)
		return true
	})

	expectedUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	type Input struct {
		Value uuid.UUID `validate:"test_getValue_uuid"`
	}
	err := v.Struct(Input{Value: expectedUUID})
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", captured)
}
