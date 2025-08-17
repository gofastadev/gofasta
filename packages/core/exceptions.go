package core

import "fmt"

// GofastaException is the base exception type for the framework
type GofastaException struct {
	Message    string
	StatusCode int
	Cause      error
}

func (e *GofastaException) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// NewGofastaException creates a new Gofasta exception
func NewGofastaException(message string, statusCode int, cause error) *GofastaException {
	return &GofastaException{
		Message:    message,
		StatusCode: statusCode,
		Cause:      cause,
	}
}

// ValidationException represents validation errors
type ValidationException struct {
	*GofastaException
	Errors map[string][]string
}

// NewValidationException creates a new validation exception
func NewValidationException(errors map[string][]string) *ValidationException {
	return &ValidationException{
		GofastaException: NewGofastaException("Validation failed", 400, nil),
		Errors:           errors,
	}
}

// UnauthorizedException represents authentication errors
type UnauthorizedException struct {
	*GofastaException
}

// NewUnauthorizedException creates a new unauthorized exception
func NewUnauthorizedException(message string) *UnauthorizedException {
	if message == "" {
		message = "Unauthorized"
	}
	return &UnauthorizedException{
		GofastaException: NewGofastaException(message, 401, nil),
	}
}

// ForbiddenException represents authorization errors
type ForbiddenException struct {
	*GofastaException
}

// NewForbiddenException creates a new forbidden exception
func NewForbiddenException(message string) *ForbiddenException {
	if message == "" {
		message = "Forbidden"
	}
	return &ForbiddenException{
		GofastaException: NewGofastaException(message, 403, nil),
	}
}

// NotFoundException represents not found errors
type NotFoundException struct {
	*GofastaException
}

// NewNotFoundException creates a new not found exception
func NewNotFoundException(message string) *NotFoundException {
	if message == "" {
		message = "Not Found"
	}
	return &NotFoundException{
		GofastaException: NewGofastaException(message, 404, nil),
	}
}

// BadRequestException represents bad request errors
type BadRequestException struct {
	*GofastaException
}

// NewBadRequestException creates a new bad request exception
func NewBadRequestException(message string) *BadRequestException {
	if message == "" {
		message = "Bad Request"
	}
	return &BadRequestException{
		GofastaException: NewGofastaException(message, 400, nil),
	}
}

// InternalServerException represents internal server errors
type InternalServerException struct {
	*GofastaException
}

// NewInternalServerException creates a new internal server exception
func NewInternalServerException(message string, cause error) *InternalServerException {
	if message == "" {
		message = "Internal Server Error"
	}
	return &InternalServerException{
		GofastaException: NewGofastaException(message, 500, cause),
	}
}

// ConflictException represents conflict errors
type ConflictException struct {
	*GofastaException
}

// NewConflictException creates a new conflict exception
func NewConflictException(message string) *ConflictException {
	if message == "" {
		message = "Conflict"
	}
	return &ConflictException{
		GofastaException: NewGofastaException(message, 409, nil),
	}
}

// UnprocessableEntityException represents validation errors with 422 status
type UnprocessableEntityException struct {
	*GofastaException
	Errors map[string][]string
}

// NewUnprocessableEntityException creates a new unprocessable entity exception
func NewUnprocessableEntityException(errors map[string][]string) *UnprocessableEntityException {
	return &UnprocessableEntityException{
		GofastaException: NewGofastaException("Unprocessable Entity", 422, nil),
		Errors:           errors,
	}
}

// TooManyRequestsException represents rate limit errors
type TooManyRequestsException struct {
	*GofastaException
	RetryAfter int
}

// NewTooManyRequestsException creates a new too many requests exception
func NewTooManyRequestsException(retryAfter int) *TooManyRequestsException {
	return &TooManyRequestsException{
		GofastaException: NewGofastaException("Too Many Requests", 429, nil),
		RetryAfter:       retryAfter,
	}
}

// ServiceUnavailableException represents service unavailable errors
type ServiceUnavailableException struct {
	*GofastaException
}

// NewServiceUnavailableException creates a new service unavailable exception
func NewServiceUnavailableException(message string) *ServiceUnavailableException {
	if message == "" {
		message = "Service Unavailable"
	}
	return &ServiceUnavailableException{
		GofastaException: NewGofastaException(message, 503, nil),
	}
}

// ExceptionHandler handles exceptions and converts them to HTTP responses
type ExceptionHandler struct{}

// NewExceptionHandler creates a new exception handler
func NewExceptionHandler() *ExceptionHandler {
	return &ExceptionHandler{}
}

// Handle handles an exception and returns an appropriate HTTP response
func (h *ExceptionHandler) Handle(exception interface{}) *Response {
	switch e := exception.(type) {
	case *ValidationException:
		return &Response{
			StatusCode: e.StatusCode,
			Body: map[string]interface{}{
				"error":   e.Message,
				"details": e.Errors,
			},
		}
	case *UnprocessableEntityException:
		return &Response{
			StatusCode: e.StatusCode,
			Body: map[string]interface{}{
				"error":   e.Message,
				"details": e.Errors,
			},
		}
	case *TooManyRequestsException:
		headers := make(map[string]string)
		if e.RetryAfter > 0 {
			headers["Retry-After"] = fmt.Sprintf("%d", e.RetryAfter)
		}
		return &Response{
			StatusCode: e.StatusCode,
			Headers:    headers,
			Body: map[string]interface{}{
				"error": e.Message,
			},
		}
	case *GofastaException:
		return &Response{
			StatusCode: e.StatusCode,
			Body: map[string]interface{}{
				"error": e.Message,
			},
		}
	case error:
		// Generic error handling
		return &Response{
			StatusCode: 500,
			Body: map[string]interface{}{
				"error": "Internal Server Error",
			},
		}
	default:
		// Unknown exception type
		return &Response{
			StatusCode: 500,
			Body: map[string]interface{}{
				"error": "Internal Server Error",
			},
		}
	}
}
