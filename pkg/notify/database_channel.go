package notify

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DBNotification is the GORM model for stored notifications.
type DBNotification struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID    string     `gorm:"not null;index" json:"userId"`
	Subject   string     `gorm:"not null" json:"subject"`
	Body      string     `json:"body"`
	Data      string     `gorm:"type:text" json:"data"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `gorm:"not null" json:"createdAt"`
}

// TableName overrides GORM's default table name.
func (DBNotification) TableName() string { return "notifications" }

// DatabaseChannel stores notifications in the database for in-app notification center.
type DatabaseChannel struct {
	db *gorm.DB
}

// NewDatabaseChannel returns a DatabaseChannel that persists notifications via db.
func NewDatabaseChannel(db *gorm.DB) *DatabaseChannel {
	return &DatabaseChannel{db: db}
}

// Channel returns the channel identifier for routing.
func (c *DatabaseChannel) Channel() Channel { return ChannelDatabase }

// Send persists n as a notification record for recipient.
func (c *DatabaseChannel) Send(ctx context.Context, recipient Recipient, n Notification) error {
	dataJSON, _ := json.Marshal(n.Data)
	record := DBNotification{
		ID:        uuid.New(),
		UserID:    recipient.ID,
		Subject:   n.Subject,
		Body:      n.Body,
		Data:      string(dataJSON),
		CreatedAt: time.Now(),
	}
	return c.db.WithContext(ctx).Create(&record).Error
}
