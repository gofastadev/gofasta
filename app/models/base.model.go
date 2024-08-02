// Base contains common columns for all tables.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel interface {
	GetID() uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// This is a concrete implementation of BaseModel
type BaseModelImpl struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (b BaseModelImpl) GetID() uuid.UUID        { return b.ID }
func (b BaseModelImpl) GetCreatedAt() time.Time { return b.CreatedAt }
func (b BaseModelImpl) GetUpdatedAt() time.Time { return b.UpdatedAt }

func (base *BaseModelImpl) BeforeCreate(txt *gorm.DB) error {
	base.ID = uuid.New()
	base.CreatedAt = time.Now()
	base.UpdatedAt = time.Now()
	return nil
}
