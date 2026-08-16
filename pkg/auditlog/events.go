package auditlog

import (
	"strings"
	"unicode"
)

// Auth events (originally from Solago)
const (
	LoginOTPSent           = "LOGIN_OTP_SENT"
	LoginOTPVerified       = "LOGIN_OTP_VERIFIED"
	LoginOTPFailed         = "LOGIN_OTP_FAILED"
	LoginSocial            = "LOGIN_SOCIAL"
	OAuth2Authorize        = "OAUTH2_AUTHORIZE"
	OAuth2TokenExchange    = "OAUTH2_TOKEN_EXCHANGE"
	TokenRefresh           = "TOKEN_REFRESH"
	PasswordResetRequested = "PASSWORD_RESET_REQUESTED"
	PasswordResetCompleted = "PASSWORD_RESET_COMPLETED"
	Logout                 = "LOGOUT"
	TokenRevoked           = "TOKEN_REVOKED"
	OTPRateLimited         = "OTP_RATE_LIMITED"
)

// CRUD events
const (
	ActionCreated = "CREATED"
	ActionUpdated = "UPDATED"
	ActionDeleted = "DELETED"
	ActionViewed  = "VIEWED"
)

// EventName builds an event type string from a resource and action.
// e.g., EventName("COURSE", "CREATED") → "COURSE_CREATED"
func EventName(resource, action string) string {
	return strings.ToUpper(resource) + "_" + strings.ToUpper(action)
}

// actionPrefixes maps camelCase mutation prefixes to audit actions.
var actionPrefixes = []struct {
	prefix string
	action string
}{
	{"create", ActionCreated},
	{"add", ActionCreated},
	{"assign", ActionCreated},
	{"register", ActionCreated},
	{"initialize", ActionCreated},
	{"update", ActionUpdated},
	{"edit", ActionUpdated},
	{"set", ActionUpdated},
	{"toggle", ActionUpdated},
	{"enable", ActionUpdated},
	{"disable", ActionUpdated},
	{"publish", ActionUpdated},
	{"unpublish", ActionUpdated},
	{"approve", ActionUpdated},
	{"reject", ActionUpdated},
	{"suspend", ActionUpdated},
	{"activate", ActionUpdated},
	{"reset", ActionUpdated},
	{"delete", ActionDeleted},
	{"remove", ActionDeleted},
	{"archive", ActionDeleted},
	{"revoke", ActionDeleted},
	{"cancel", ActionDeleted},
}

// ParseMutationName extracts the resource type and action from a camelCase
// mutation field name. For example:
//
//	"createCourse"       → resource="COURSE",        action="CREATED"
//	"updateCourseVersion" → resource="COURSE_VERSION", action="UPDATED"
//	"deleteLiveClass"    → resource="LIVE_CLASS",     action="DELETED"
//	"unknownThing"       → resource="UNKNOWN_THING",  action="" (no match)
func ParseMutationName(fieldName string) (resource, action string) {
	for _, ap := range actionPrefixes {
		if strings.HasPrefix(fieldName, ap.prefix) {
			remainder := fieldName[len(ap.prefix):]
			if remainder == "" {
				return strings.ToUpper(ap.prefix), ap.action
			}
			// Remainder starts with uppercase if it's a valid prefix match
			if remainder != "" && unicode.IsUpper(rune(remainder[0])) {
				return camelToUpperSnake(remainder), ap.action
			}
		}
	}
	// No prefix matched — use the full name as resource, no action
	return camelToUpperSnake(fieldName), ""
}

// camelToUpperSnake converts a PascalCase or camelCase string to UPPER_SNAKE_CASE.
// e.g., "CourseVersion" → "COURSE_VERSION", "liveClass" → "LIVE_CLASS"
func camelToUpperSnake(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}
