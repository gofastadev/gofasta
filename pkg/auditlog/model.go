package auditlog

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Entry represents an audit log entry stored in each service's database.
//
// SubjectID is the authenticated principal that performed the action, taken
// from the token's `sub` claim. It is deliberately not called UserID: under the
// client-credentials grant the subject is a client id, and a column named for
// users while holding client ids misleads exactly the person reading an audit
// trail during an incident.
//
// It is nullable because an action can be genuinely unattributed — a scheduled
// job, a system event, a request that failed before authentication. NULL and
// "the middleware did not run" are indistinguishable here, so the middleware
// wiring is worth checking when subjects come back empty.
type Entry struct {
	ID            string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventType     string          `gorm:"type:varchar(100);not null"`
	SubjectID     *string         `gorm:"type:uuid"`
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
//
// The subject column is a different story: GORM derives it from the field name,
// so SubjectID maps to subject_id and a database still carrying user_id needs
// an ALTER TABLE ... RENAME COLUMN. There is no column tag holding the old name
// on purpose — a field called SubjectID silently writing to user_id is the kind
// of drift that misleads whoever reads the schema next.
func (Entry) TableName() string {
	return "audit_logs"
}

// Filter provides filtering options for querying audit logs.
type Filter struct {
	EventType    string
	SubjectID    string
	ServiceName  string
	ResourceType string
	ResourceID   string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}
