package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBaseModelImpl_Getters(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	deletedAt := gorm.DeletedAt{Time: now.Add(-time.Hour), Valid: true}

	model := BaseModelImpl{
		ID:            id,
		CreatedAt:     now,
		UpdatedAt:     now,
		DeletedAt:     deletedAt,
		RecordVersion: 3,
		IsActive:      true,
		IsDeletable:   false,
	}

	assert.Equal(t, id, model.GetID())
	assert.Equal(t, now, model.GetCreatedAt())
	assert.Equal(t, now, model.GetUpdatedAt())
	got := model.GetDeletedAt()
	assert.True(t, got.Valid, "GetDeletedAt should report Valid for a soft-deleted record")
	assert.Equal(t, deletedAt.Time, got.Time)
	assert.Equal(t, 3, model.GetRecordVersion())
	assert.True(t, model.GetIsActive())
	assert.False(t, model.GetIsDeletable())
}

func TestBaseModelImpl_GettersDefaults(t *testing.T) {
	model := BaseModelImpl{}

	assert.Equal(t, uuid.Nil, model.GetID())
	assert.True(t, model.GetCreatedAt().IsZero())
	assert.True(t, model.GetUpdatedAt().IsZero())
	// Zero-value gorm.DeletedAt has Valid=false (i.e. NOT soft-deleted).
	assert.False(t, model.GetDeletedAt().Valid)
	assert.Equal(t, 0, model.GetRecordVersion())
	assert.False(t, model.GetIsActive())
	assert.False(t, model.GetIsDeletable())
}

func TestBaseModelImpl_BeforeCreate(t *testing.T) {
	model := &BaseModelImpl{}

	before := time.Now()
	err := model.BeforeCreate(nil)
	after := time.Now()

	require.NoError(t, err)

	// ID should be a valid non-nil UUID
	assert.NotEqual(t, uuid.Nil, model.ID)
	_, err = uuid.Parse(model.ID.String())
	require.NoError(t, err)

	// CreatedAt and UpdatedAt should be set to approximately now
	assert.True(t, !model.CreatedAt.Before(before) && !model.CreatedAt.After(after),
		"CreatedAt should be between before and after")
	assert.True(t, !model.UpdatedAt.Before(before) && !model.UpdatedAt.After(after),
		"UpdatedAt should be between before and after")

	// Defaults
	assert.True(t, model.IsActive)
	assert.True(t, model.IsDeletable)
	assert.Equal(t, 1, model.RecordVersion)
}

func TestBaseModelImpl_BeforeCreate_GeneratesUniqueIDs(t *testing.T) {
	model1 := &BaseModelImpl{}
	model2 := &BaseModelImpl{}

	err := model1.BeforeCreate(nil)
	require.NoError(t, err)
	err = model2.BeforeCreate(nil)
	require.NoError(t, err)

	assert.NotEqual(t, model1.ID, model2.ID)
}

func TestBaseModelImpl_BeforeCreate_OverwritesExistingValues(t *testing.T) {
	oldID := uuid.New()
	model := &BaseModelImpl{
		ID:            oldID,
		IsActive:      false,
		IsDeletable:   false,
		RecordVersion: 99,
	}

	err := model.BeforeCreate(nil)
	require.NoError(t, err)

	// BeforeCreate always overwrites
	assert.NotEqual(t, oldID, model.ID)
	assert.True(t, model.IsActive)
	assert.True(t, model.IsDeletable)
	assert.Equal(t, 1, model.RecordVersion)
}
