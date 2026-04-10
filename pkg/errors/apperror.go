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
