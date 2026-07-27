package models_test

import (
	"testing"

	"github.com/gofastadev/gofasta/pkg/models"
)

// TestBaseModelIsUsableAsAnInterface is a compile-time regression test.
//
// BaseModel previously embedded gorm.Model — a struct — which made it a
// constraint-only interface. Declaring a variable of that type failed to
// compile with "cannot use type models.BaseModel outside a type constraint",
// so the interface this package documents could not be used the way its doc
// comment describes.
//
// The assignment below is the whole test: if the embed comes back, this file
// stops compiling and the package's tests fail.
func TestBaseModelIsUsableAsAnInterface(t *testing.T) {
	var m models.BaseModel = models.BaseModelImpl{}

	if got := m.GetRecordVersion(); got != 0 {
		t.Errorf("zero-valued BaseModelImpl reported RecordVersion %d, want 0", got)
	}
	if m.GetDeletedAt().Valid {
		t.Error("zero-valued BaseModelImpl reports itself soft-deleted")
	}
}
