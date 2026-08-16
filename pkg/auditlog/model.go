package auditlog

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Entry represents an audit log entry stored in each service's database.
type Entry struct {
	ID            string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventType     string          `gorm:"type:varchar(100);not null"`
	UserID        *string         `gorm:"type:uuid"`
	ServiceName   string          `gorm:"type:varchar(50);not null"`
	IPAddress     string          `gorm:"type:varchar(45)"`
	UserAgent     string          `gorm:"type:text"`
	Details       json.RawMessage `gorm:"type:jsonb"`
	ResourceType  string          `gorm:"type:varchar(100)"`
	ResourceID    string          `gorm:"type:varchar(255)"`
	CreatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt     gorm.DeletedAt  `gorm:"index"`
	RecordVersion int             `gorm:"not null;default:1"`
	IsActive      bool            `gorm:"not null;default:true"`
	IsDeletable   bool            `gorm:"not null;default:true"`
}

// TableName pins the table to audit_logs. The Go type was renamed when this
// package moved into the framework; the table it maps to was not, because rows
// already live there.
func (Entry) TableName() string {
	return "audit_logs"
}

// Filter provides filtering options for querying audit logs.
type Filter struct {
	EventType    string
	UserID       string
	ServiceName  string
	ResourceType string
	ResourceID   string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}
