// Base contains common columns for all tables.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel interface {
	gorm.Model
	GetID() uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetIsActive() bool
	GetIsDeletable() bool
	GetRecordVersion() int
}

// This is a concrete implementation of BaseModel
type BaseModelImpl struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt     time.Time `gorm:"type:timestamp;not null;"`
	UpdatedAt     time.Time `gorm:"type:timestamp;not null;"`
	RecordVersion int       `gorm:"type:int;not null;default:1"`
	IsActive      bool      `gorm:"type:bool;not null;default:true"`
	IsDeletable   bool      `gorm:"type:bool;not null;default:true"`
}

func (b BaseModelImpl) GetID() uuid.UUID        { return b.ID }
func (b BaseModelImpl) GetCreatedAt() time.Time { return b.CreatedAt }
func (b BaseModelImpl) GetUpdatedAt() time.Time { return b.UpdatedAt }
func (b BaseModelImpl) GetIsActive() bool       { return b.IsActive }
func (b BaseModelImpl) GetIsDeletable() bool    { return b.IsDeletable }
func (b BaseModelImpl) GetRecordVersion() int   { return b.RecordVersion }

func (base *BaseModelImpl) BeforeCreate(txt *gorm.DB) error {
	base.ID = uuid.New()
	base.CreatedAt = time.Now()
	base.UpdatedAt = time.Now()
	base.IsActive = true
	base.IsDeletable = true
	base.RecordVersion = 1
	return nil
}
