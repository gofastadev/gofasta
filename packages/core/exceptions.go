package core

import "fmt"

// GofastaError is the base error type for the framework
type GofastaError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"statusCode"`
	Cause      error                  `json:"cause,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func (e *GofastaError) Error() string {
	return e.Message
}

// NewGofastaError creates a new Gofasta error
func NewGofastaError(code, message string, statusCode int) *GofastaError {
	return &GofastaError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Metadata:   make(map[string]interface{}),
	}
}

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value"`
	Tag     string      `json:"tag"`
}

// ValidationError represents validation errors
type ValidationError struct {
	*GofastaError
	Errors []FieldError `json:"errors"`
}

// NewValidationError creates a new validation error
func NewValidationError(message string, errors []FieldError) *ValidationError {
	return &ValidationError{
		GofastaError: NewGofastaError("VALIDATION_ERROR", message, 422),
		Errors:       errors,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, id string) *GofastaError {
	return NewGofastaError("NOT_FOUND", fmt.Sprintf("%s with ID %s not found", resource, id), 404)
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *GofastaError {
	if message == "" {
		message = "Unauthorized"
	}
	return NewGofastaError("UNAUTHORIZED", message, 401)
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string) *GofastaError {
	if message == "" {
		message = "Forbidden"
	}
	return NewGofastaError("FORBIDDEN", message, 403)
}

// NewBadRequestError creates a new bad request error
func NewBadRequestError(message string) *GofastaError {
	if message == "" {
		message = "Bad Request"
	}
	return NewGofastaError("BAD_REQUEST", message, 400)
}

// NewInternalServerError creates a new internal server error
func NewInternalServerError(message string) *GofastaError {
	if message == "" {
		message = "Internal Server Error"
	}
	return NewGofastaError("INTERNAL_SERVER_ERROR", message, 500)
}

// NewConflictError creates a new conflict error
func NewConflictError(message string) *GofastaError {
	if message == "" {
		message = "Conflict"
	}
	return NewGofastaError("CONFLICT", message, 409)
}

// Legacy types for backward compatibility
type GofastaException = GofastaError

// NewGofastaException creates a new Gofasta exception (legacy)
func NewGofastaException(message string, statusCode int, cause error) *GofastaException {
	err := NewGofastaError("GENERIC_ERROR", message, statusCode)
	err.Cause = cause
	return err
}

// ValidationException represents validation errors (legacy)
type ValidationException struct {
	*GofastaException
	Errors map[string][]string
}

// NewValidationException creates a new validation exception (legacy)
func NewValidationException(errors map[string][]string) *ValidationException {
	return &ValidationException{
		GofastaException: NewGofastaException("Validation failed", 400, nil),
		Errors:           errors,
	}
}

// UnauthorizedException represents authentication errors (legacy)
type UnauthorizedException struct {
	*GofastaException
}

// NewUnauthorizedException creates a new unauthorized exception (legacy)
func NewUnauthorizedException(message string) *UnauthorizedException {
	if message == "" {
		message = "Unauthorized"
	}
	return &UnauthorizedException{
		GofastaException: NewGofastaException(message, 401, nil),
	}
}

// ForbiddenException represents authorization errors (legacy)
type ForbiddenException struct {
	*GofastaException
}

// NewForbiddenException creates a new forbidden exception (legacy)
func NewForbiddenException(message string) *ForbiddenException {
	if message == "" {
		message = "Forbidden"
	}
	return &ForbiddenException{
		GofastaException: NewGofastaException(message, 403, nil),
	}
}

// NotFoundException represents not found errors (legacy)
type NotFoundException struct {
	*GofastaException
	Resource string
	ID       string
}

// NewNotFoundException creates a new not found exception (legacy)
func NewNotFoundException(resource, id string) *NotFoundException {
	message := fmt.Sprintf("%s with ID %s not found", resource, id)
	return &NotFoundException{
		GofastaException: NewGofastaException(message, 404, nil),
		Resource:         resource,
		ID:               id,
	}
}

// BadRequestException represents bad request errors (legacy)
type BadRequestException struct {
	*GofastaException
}

// NewBadRequestException creates a new bad request exception (legacy)
func NewBadRequestException(message string) *BadRequestException {
	if message == "" {
		message = "Bad Request"
	}
	return &BadRequestException{
		GofastaException: NewGofastaException(message, 400, nil),
	}
}

// InternalServerException represents internal server errors (legacy)
type InternalServerException struct {
	*GofastaException
}

// NewInternalServerException creates a new internal server exception (legacy)
func NewInternalServerException(message string, cause error) *InternalServerException {
	if message == "" {
		message = "Internal Server Error"
	}
	return &InternalServerException{
		GofastaException: NewGofastaException(message, 500, cause),
	}
}

// ConflictException represents conflict errors (legacy)
type ConflictException struct {
	*GofastaException
}

// NewConflictException creates a new conflict exception (legacy)
func NewConflictException(message string) *ConflictException {
	if message == "" {
		message = "Conflict"
	}
	return &ConflictException{
		GofastaException: NewGofastaException(message, 409, nil),
	}
}

// ServiceUnavailableException represents service unavailable errors (legacy)
type ServiceUnavailableException struct {
	*GofastaException
}

// NewServiceUnavailableException creates a new service unavailable exception (legacy)
func NewServiceUnavailableException(message string) *ServiceUnavailableException {
	if message == "" {
		message = "Service Unavailable"
	}
	return &ServiceUnavailableException{
		GofastaException: NewGofastaException(message, 503, nil),
	}
}

// TooManyRequestsException represents rate limiting errors (legacy)
type TooManyRequestsException struct {
	*GofastaException
	RetryAfter int
}

// NewTooManyRequestsException creates a new too many requests exception (legacy)
func NewTooManyRequestsException(message string, retryAfter int) *TooManyRequestsException {
	if message == "" {
		message = "Too Many Requests"
	}
	return &TooManyRequestsException{
		GofastaException: NewGofastaException(message, 429, nil),
		RetryAfter:       retryAfter,
	}
}