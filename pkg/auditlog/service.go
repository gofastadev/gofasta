// Package auditlog records who did what, from where, and when.
//
// The write path is deliberately forgiving: LogEvent is asynchronous and
// swallows failures after logging them, because an audit write must never be
// the reason a user's action fails. LogEventSync exists for the events where
// the opposite is true and the caller wants to know.
package auditlog

import (
	"context"
	"encoding/json"
	"log/slog"

	"gorm.io/gorm"

	"github.com/gofastadev/gofasta/pkg/auth"
	"github.com/gofastadev/gofasta/pkg/httpcontext"
)

// SubjectFunc identifies the acting user from a request context.
//
// It exists because "who is the subject" is the one part of an audit record a
// framework cannot know: the claim carrying the user's identity differs per
// issuer. The default reads gofasta's own Claims; a project whose tokens carry
// the identity elsewhere supplies its own.
type SubjectFunc func(ctx context.Context) *string

// defaultSubject reads the acting user from gofasta's Claims.
//
// Through [auth.Claims.SubjectID], not the UserID field: tokens minted by an
// OAuth 2.0 / OIDC provider carry the identity in the registered `sub` claim
// and leave `user_id` empty. Reading the field directly writes a subject-less
// row for every such caller — and "who did this" is the first question anyone
// asks of an audit log.
func defaultSubject(ctx context.Context) *string {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return nil
	}
	id := claims.SubjectID()
	if id == "" {
		return nil
	}
	return &id
}

// Option configures an Service.
type Option func(*Service)

// WithSubjectFunc replaces how the acting user is identified.
func WithSubjectFunc(f SubjectFunc) Option {
	return func(s *Service) {
		if f != nil {
			s.subject = f
		}
	}
}

// Service provides methods for creating and querying audit log entries.
type Service struct {
	DB          *gorm.DB
	ServiceName string

	subject SubjectFunc
}

// NewAuditService creates a new Service for the given service.
func NewAuditService(db *gorm.DB, serviceName string, opts ...Option) *Service {
	s := &Service{DB: db, ServiceName: serviceName, subject: defaultSubject}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// LogEvent creates an audit log entry asynchronously.
func (s *Service) LogEvent(eventType string, userID *string, ipAddress, userAgent string, details map[string]interface{}) {
	go func() { _ = s.logEventInternal(eventType, userID, ipAddress, userAgent, details, "", "") }()
}

// LogEventWithResource creates an audit log entry with resource info asynchronously.
func (s *Service) LogEventWithResource(eventType string, userID *string, ipAddress, userAgent string, details map[string]interface{}, resourceType, resourceID string) {
	go func() {
		_ = s.logEventInternal(eventType, userID, ipAddress, userAgent, details, resourceType, resourceID)
	}()
}

// LogEventSync creates an audit log entry synchronously for critical events.
func (s *Service) LogEventSync(eventType string, userID *string, ipAddress, userAgent string, details map[string]interface{}) error {
	return s.logEventInternal(eventType, userID, ipAddress, userAgent, details, "", "")
}

func (s *Service) logEventInternal(eventType string, userID *string, ipAddress, userAgent string, details map[string]interface{}, resourceType, resourceID string) error {
	var detailsJSON json.RawMessage
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			slog.Error("auditlog: marshaling details", "error", err, "event_type", eventType)
			return err
		}
		detailsJSON = b
	}

	entry := Entry{
		EventType:    eventType,
		UserID:       userID,
		ServiceName:  s.ServiceName,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Details:      detailsJSON,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	if err := s.DB.Create(&entry).Error; err != nil {
		slog.Error("auditlog: creating entry", "error", err, "event_type", eventType)
		return err
	}
	return nil
}

// LogFromContext extracts user info from context and logs an event asynchronously.
func (s *Service) LogFromContext(ctx context.Context, eventType string, details map[string]interface{}) {
	userID, ipAddress, userAgent := s.FromContext(ctx)
	s.LogEvent(eventType, userID, ipAddress, userAgent, details)
}

// LogFromContextWithResource extracts user info from context and logs an event with resource info.
func (s *Service) LogFromContextWithResource(ctx context.Context, eventType, resourceType, resourceID string, details map[string]interface{}) {
	userID, ipAddress, userAgent := s.FromContext(ctx)
	s.LogEventWithResource(eventType, userID, ipAddress, userAgent, details, resourceType, resourceID)
}

// FromContext extracts the acting user, client IP, and user agent.
//
// The IP and user agent come from the request that httpcontext.Middleware put
// in the context. Without that middleware installed they are empty — and an
// audit trail with no client address is close to useless, so the omission is
// worth catching in a test rather than discovering during an investigation.
//
// X-Forwarded-For wins over RemoteAddr because every deployment behind a proxy
// or load balancer would otherwise record the proxy's address for every user.
func (s *Service) FromContext(ctx context.Context) (userID *string, ipAddress, userAgent string) {
	if s.subject != nil {
		userID = s.subject(ctx)
	}

	if r := httpcontext.Request(ctx); r != nil {
		ipAddress = r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ipAddress = forwarded
		}
		userAgent = r.UserAgent()
	}

	return userID, ipAddress, userAgent
}

// QueryLogs retrieves audit logs matching the given filters.
func (s *Service) QueryLogs(filters Filter) ([]Entry, int64, error) {
	var logs []Entry
	var total int64

	query := s.DB.Model(&Entry{})

	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.ServiceName != "" {
		query = query.Where("service_name = ?", filters.ServiceName)
	}
	if filters.ResourceType != "" {
		query = query.Where("resource_type = ?", filters.ResourceType)
	}
	if filters.ResourceID != "" {
		query = query.Where("resource_id = ?", filters.ResourceID)
	}
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", filters.EndDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	if err := query.Order("created_at DESC").Offset(filters.Offset).Limit(filters.Limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
