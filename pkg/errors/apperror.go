package apperrors

import "fmt"

// ErrorType represents the category of an application error.
type ErrorType int

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

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Internal
}

func NewNotFound(msg string, internal error) *AppError {
	return &AppError{Type: NotFound, Message: msg, Internal: internal}
}

func NewValidation(msg string, details interface{}) *AppError {
	return &AppError{Type: Validation, Message: msg, Details: details}
}

func NewInternal(msg string, internal error) *AppError {
	return &AppError{Type: Internal, Message: msg, Internal: internal}
}

func NewConflict(msg string, internal error) *AppError {
	return &AppError{Type: Conflict, Message: msg, Internal: internal}
}

func NewUnauthorized(msg string, internal error) *AppError {
	return &AppError{Type: Unauthorized, Message: msg, Internal: internal}
}

func NewForbidden(msg string, internal error) *AppError {
	return &AppError{Type: Forbidden, Message: msg, Internal: internal}
}

func NewBadRequest(msg string, details interface{}) *AppError {
	return &AppError{Type: BadRequest, Message: msg, Details: details}
}
