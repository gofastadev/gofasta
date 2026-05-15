// Package models provides the base model types for gofasta applications.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel is the interface every gofasta domain model should satisfy.
// It exposes the common framework fields (ID, timestamps, soft-delete,
// active/deletable flags, optimistic-concurrency version).
//
// GetDeletedAt returns gorm.DeletedAt — a sql.NullTime alias that
// GORM's soft-delete plugin recognizes. Callers wanting just the
// time can read .Time; callers wanting "is this deleted?" should
// check .Valid (true = soft-deleted).
type BaseModel interface {
	gorm.Model
	GetID() uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetDeletedAt() gorm.DeletedAt
	GetIsActive() bool
	GetIsDeletable() bool
	GetRecordVersion() int
}

// BaseModelImpl is a concrete implementation of BaseModel.
// Embed this in your domain models to get standard fields.
//
// DeletedAt is gorm.DeletedAt (not plain time.Time) so soft-delete
// works the way GORM intends: rows are inserted with deleted_at = NULL,
// SELECT queries automatically filter out soft-deleted rows via
// `WHERE deleted_at IS NULL`, and Delete() flips deleted_at = NOW()
// instead of removing the row. The `index` tag accelerates the
// auto-applied filter.
type BaseModelImpl struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;"`
	CreatedAt     time.Time      `gorm:"type:timestamp;not null;"`
	UpdatedAt     time.Time      `gorm:"type:timestamp;not null;"`
	DeletedAt     gorm.DeletedAt `gorm:"type:timestamp;index;"`
	RecordVersion int            `gorm:"type:int;not null;default:1"`
	IsActive      bool           `gorm:"type:bool;not null;default:true"`
	IsDeletable   bool           `gorm:"type:bool;not null;default:true"`
}

// GetID returns the model's UUID primary key.
func (b BaseModelImpl) GetID() uuid.UUID { return b.ID }

// GetCreatedAt returns the creation timestamp.
func (b BaseModelImpl) GetCreatedAt() time.Time { return b.CreatedAt }

// GetUpdatedAt returns the last-update timestamp.
func (b BaseModelImpl) GetUpdatedAt() time.Time { return b.UpdatedAt }

// GetIsActive reports whether the record is currently active.
func (b BaseModelImpl) GetIsActive() bool { return b.IsActive }

// GetIsDeletable reports whether the record may be deleted.
func (b BaseModelImpl) GetIsDeletable() bool { return b.IsDeletable }

// GetRecordVersion returns the optimistic-concurrency version counter.
func (b BaseModelImpl) GetRecordVersion() int { return b.RecordVersion }

// GetDeletedAt returns the soft-delete sentinel. The zero value
// (gorm.DeletedAt{Valid: false}) means the record has NOT been soft-
// deleted. When Valid is true, .Time holds the soft-delete timestamp.
func (b BaseModelImpl) GetDeletedAt() gorm.DeletedAt { return b.DeletedAt }

// BeforeCreate is a GORM hook that populates the UUID, timestamps, and
// defaults for a new record.
func (b *BaseModelImpl) BeforeCreate(_ *gorm.DB) error {
	b.ID = uuid.New()
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()
	b.IsActive = true
	b.IsDeletable = true
	b.RecordVersion = 1
	return nil
}
