package validators

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
