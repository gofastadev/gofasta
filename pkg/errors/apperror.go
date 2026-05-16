package apperrors

import "fmt"

// ErrorType represents the category of an application error.
type ErrorType int

// Error-type constants used by AppError.Type to classify a failure.
const (
	NotFound ErrorType = iota
	Validation
	Conflict
	Internal
	Unauthorized
	Forbidden
	BadRequest
	// PreconditionFailed maps to HTTP 412 (RFC 7232 §4.2). Use when an
	// If-Match / If-Unmodified-Since precondition the client supplied
	// does not hold against current server state — the canonical
	// optimistic-concurrency-control rejection.
	PreconditionFailed
	// PreconditionRequired maps to HTTP 428 (RFC 6585 §3). Use when the
	// server requires the request to be conditional (e.g., must include
	// If-Match) but the client did not provide one — protects against
	// the lost-update problem when callers forget the precondition.
	PreconditionRequired
)

// AppError is a structured error type for the application.
type AppError struct {
	Type     ErrorType
	Message  string
	Details  interface{}
	Internal error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// Unwrap returns the wrapped internal error, if any.
func (e *AppError) Unwrap() error {
	return e.Internal
}

// NewNotFound builds an AppError of type NotFound.
func NewNotFound(msg string, internal error) *AppError {
	return &AppError{Type: NotFound, Message: msg, Internal: internal}
}

// NewValidation builds an AppError of type Validation.
func NewValidation(msg string, details interface{}) *AppError {
	return &AppError{Type: Validation, Message: msg, Details: details}
}

// NewInternal builds an AppError of type Internal.
func NewInternal(msg string, internal error) *AppError {
	return &AppError{Type: Internal, Message: msg, Internal: internal}
}

// NewConflict builds an AppError of type Conflict.
func NewConflict(msg string, internal error) *AppError {
	return &AppError{Type: Conflict, Message: msg, Internal: internal}
}

// NewUnauthorized builds an AppError of type Unauthorized.
func NewUnauthorized(msg string, internal error) *AppError {
	return &AppError{Type: Unauthorized, Message: msg, Internal: internal}
}

// NewForbidden builds an AppError of type Forbidden.
func NewForbidden(msg string, internal error) *AppError {
	return &AppError{Type: Forbidden, Message: msg, Internal: internal}
}

// NewBadRequest builds an AppError of type BadRequest.
func NewBadRequest(msg string, details interface{}) *AppError {
	return &AppError{Type: BadRequest, Message: msg, Details: details}
}

// NewPreconditionFailed builds an AppError of type PreconditionFailed
// (HTTP 412). Use when the client's If-Match (or other conditional
// header) does not match current server state — the optimistic-
// concurrency-control rejection. Pass the wrapped GORM/sql error in
// internal so debugging keeps the original cause.
func NewPreconditionFailed(msg string, internal error) *AppError {
	return &AppError{Type: PreconditionFailed, Message: msg, Internal: internal}
}

// NewPreconditionRequired builds an AppError of type
// PreconditionRequired (HTTP 428). Use when an endpoint requires the
// caller to send an If-Match header and they did not — surfaces the
// lost-update risk explicitly so agents and clients know to fetch
// the current ETag before retrying.
func NewPreconditionRequired(msg string, details interface{}) *AppError {
	return &AppError{Type: PreconditionRequired, Message: msg, Details: details}
}
