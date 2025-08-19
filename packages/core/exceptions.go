package core

import (
	"fmt"
	"strings"
	"time"
)

// GofastaError is the base error type for the framework
// It provides HTTP-aware error handling with modern Go error patterns
type GofastaError struct {
	Code       string                 `json:"code"`            // Error code identifier
	Message    string                 `json:"message"`         // Human-readable error message
	StatusCode int                    `json:"statusCode"`      // HTTP status code
	Cause      error                  `json:"-"`               // Underlying cause (supports error wrapping)
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Additional error context
	Timestamp  string                 `json:"timestamp,omitempty"` // Error occurrence timestamp
	Path       string                 `json:"path,omitempty"`     // Request path where error occurred
}

// Error implements the error interface
func (e *GofastaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error, supporting Go 1.13+ error wrapping
func (e *GofastaError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for Go 1.13+ errors.Is functionality
// Matches based on status code or error code
func (e *GofastaError) Is(target error) bool {
	if gofastaErr, ok := target.(*GofastaError); ok {
		return e.StatusCode == gofastaErr.StatusCode || e.Code == gofastaErr.Code
	}
	return false
}

// As implements error casting for Go 1.13+ errors.As functionality  
func (e *GofastaError) As(target any) bool {
	if gofastaErr, ok := target.(**GofastaError); ok {
		*gofastaErr = e
		return true
	}
	return false
}

// WithCause adds an underlying cause to the error (error wrapping)
func (e *GofastaError) WithCause(cause error) *GofastaError {
	e.Cause = cause
	return e
}

// WithMetadata adds additional metadata to the error
func (e *GofastaError) WithMetadata(key string, value interface{}) *GofastaError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithAllMetadata sets all metadata at once
func (e *GofastaError) WithAllMetadata(metadata map[string]interface{}) *GofastaError {
	e.Metadata = metadata
	return e
}

// WithPath sets the request path where the error occurred
func (e *GofastaError) WithPath(path string) *GofastaError {
	e.Path = path
	return e
}

// WithTimestamp sets the error occurrence timestamp
func (e *GofastaError) WithTimestamp(timestamp string) *GofastaError {
	e.Timestamp = timestamp
	return e
}

// WithCurrentTimestamp sets the error occurrence timestamp to current time
func (e *GofastaError) WithCurrentTimestamp() *GofastaError {
	e.Timestamp = time.Now().Format(time.RFC3339)
	return e
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

// BadRequestError represents bad request errors (400)
// Extends GofastaError with specific functionality for invalid request data
type BadRequestError struct {
	*GofastaError
	InvalidFields []string                 `json:"invalidFields,omitempty"` // List of invalid field names
	Suggestions   []string                 `json:"suggestions,omitempty"`   // Suggested corrections
	RequestInfo   map[string]interface{}   `json:"requestInfo,omitempty"`   // Additional request context
}

// NewBadRequestError creates a new bad request error
func NewBadRequestError(message string) *BadRequestError {
	if message == "" {
		message = "Bad Request"
	}
	return &BadRequestError{
		GofastaError: NewGofastaError("BAD_REQUEST", message, 400),
	}
}

// NewBadRequestErrorf creates a new bad request error with formatted message
func NewBadRequestErrorf(format string, args ...interface{}) *BadRequestError {
	return &BadRequestError{
		GofastaError: NewGofastaError("BAD_REQUEST", fmt.Sprintf(format, args...), 400),
	}
}

// NewBadRequestErrorWithCause creates a new bad request error with underlying cause
func NewBadRequestErrorWithCause(message string, cause error) *BadRequestError {
	if message == "" {
		message = "Bad Request"
	}
	badReqErr := &BadRequestError{
		GofastaError: NewGofastaError("BAD_REQUEST", message, 400),
	}
	badReqErr.GofastaError.Cause = cause
	return badReqErr
}

// WithInvalidFields adds invalid field names to the error
func (e *BadRequestError) WithInvalidFields(fields ...string) *BadRequestError {
	e.InvalidFields = append(e.InvalidFields, fields...)
	return e
}

// WithSuggestions adds suggested corrections to the error
func (e *BadRequestError) WithSuggestions(suggestions ...string) *BadRequestError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// WithRequestInfo adds additional request context information
func (e *BadRequestError) WithRequestInfo(key string, value interface{}) *BadRequestError {
	if e.RequestInfo == nil {
		e.RequestInfo = make(map[string]interface{})
	}
	e.RequestInfo[key] = value
	return e
}

// WithAllRequestInfo sets all request info at once
func (e *BadRequestError) WithAllRequestInfo(info map[string]interface{}) *BadRequestError {
	e.RequestInfo = info
	return e
}

// WithCause adds an underlying cause to the error (overrides GofastaError method for type consistency)
func (e *BadRequestError) WithCause(cause error) *BadRequestError {
	e.GofastaError.Cause = cause
	return e
}

// WithMetadata adds additional metadata to the error (overrides GofastaError method for type consistency)
func (e *BadRequestError) WithMetadata(key string, value interface{}) *BadRequestError {
	if e.GofastaError.Metadata == nil {
		e.GofastaError.Metadata = make(map[string]interface{})
	}
	e.GofastaError.Metadata[key] = value
	return e
}

// WithAllMetadata sets all metadata at once (overrides GofastaError method for type consistency)
func (e *BadRequestError) WithAllMetadata(metadata map[string]interface{}) *BadRequestError {
	e.GofastaError.Metadata = metadata
	return e
}

// WithPath sets the request path where the error occurred (overrides GofastaError method for type consistency)
func (e *BadRequestError) WithPath(path string) *BadRequestError {
	e.GofastaError.Path = path
	return e
}

// WithTimestamp sets the error occurrence timestamp (overrides GofastaError method for type consistency)
func (e *BadRequestError) WithTimestamp(timestamp string) *BadRequestError {
	e.GofastaError.Timestamp = timestamp
	return e
}

// WithCurrentTimestamp sets the error occurrence timestamp to current time (overrides GofastaError method for type consistency)
func (e *BadRequestError) WithCurrentTimestamp() *BadRequestError {
	e.GofastaError.Timestamp = time.Now().Format(time.RFC3339)
	return e
}

// UnauthorizedError represents unauthorized access errors (401)
// Extends GofastaError with specific functionality for authentication failures
type UnauthorizedError struct {
	*GofastaError
	AuthScheme     string                 `json:"authScheme,omitempty"`     // Authentication scheme (Bearer, Basic, etc.)
	Realm          string                 `json:"realm,omitempty"`          // Authentication realm
	Challenges     []string               `json:"challenges,omitempty"`     // Authentication challenges
	LoginUrl       string                 `json:"loginUrl,omitempty"`       // Login URL for redirection
	AuthContext    map[string]interface{} `json:"authContext,omitempty"`    // Additional authentication context
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *UnauthorizedError {
	if message == "" {
		message = "Unauthorized"
	}
	return &UnauthorizedError{
		GofastaError: NewGofastaError("UNAUTHORIZED", message, 401),
	}
}

// NewUnauthorizedErrorf creates a new unauthorized error with formatted message
func NewUnauthorizedErrorf(format string, args ...interface{}) *UnauthorizedError {
	return &UnauthorizedError{
		GofastaError: NewGofastaError("UNAUTHORIZED", fmt.Sprintf(format, args...), 401),
	}
}

// NewUnauthorizedErrorWithCause creates a new unauthorized error with underlying cause
func NewUnauthorizedErrorWithCause(message string, cause error) *UnauthorizedError {
	if message == "" {
		message = "Unauthorized"
	}
	unauthorizedErr := &UnauthorizedError{
		GofastaError: NewGofastaError("UNAUTHORIZED", message, 401),
	}
	unauthorizedErr.GofastaError.Cause = cause
	return unauthorizedErr
}

// WithAuthScheme sets the authentication scheme (Bearer, Basic, etc.)
func (e *UnauthorizedError) WithAuthScheme(scheme string) *UnauthorizedError {
	e.AuthScheme = scheme
	return e
}

// WithRealm sets the authentication realm
func (e *UnauthorizedError) WithRealm(realm string) *UnauthorizedError {
	e.Realm = realm
	return e
}

// WithChallenges adds authentication challenges
func (e *UnauthorizedError) WithChallenges(challenges ...string) *UnauthorizedError {
	e.Challenges = append(e.Challenges, challenges...)
	return e
}

// WithLoginUrl sets a login URL for redirection
func (e *UnauthorizedError) WithLoginUrl(url string) *UnauthorizedError {
	e.LoginUrl = url
	return e
}

// WithAuthContext adds authentication context information
func (e *UnauthorizedError) WithAuthContext(key string, value interface{}) *UnauthorizedError {
	if e.AuthContext == nil {
		e.AuthContext = make(map[string]interface{})
	}
	e.AuthContext[key] = value
	return e
}

// WithAllAuthContext sets all authentication context at once
func (e *UnauthorizedError) WithAllAuthContext(context map[string]interface{}) *UnauthorizedError {
	e.AuthContext = context
	return e
}

// WithCause adds an underlying cause to the error (overrides GofastaError method for type consistency)
func (e *UnauthorizedError) WithCause(cause error) *UnauthorizedError {
	e.GofastaError.Cause = cause
	return e
}

// WithMetadata adds additional metadata to the error (overrides GofastaError method for type consistency)
func (e *UnauthorizedError) WithMetadata(key string, value interface{}) *UnauthorizedError {
	if e.GofastaError.Metadata == nil {
		e.GofastaError.Metadata = make(map[string]interface{})
	}
	e.GofastaError.Metadata[key] = value
	return e
}

// WithAllMetadata sets all metadata at once (overrides GofastaError method for type consistency)
func (e *UnauthorizedError) WithAllMetadata(metadata map[string]interface{}) *UnauthorizedError {
	e.GofastaError.Metadata = metadata
	return e
}

// WithPath sets the request path where the error occurred (overrides GofastaError method for type consistency)
func (e *UnauthorizedError) WithPath(path string) *UnauthorizedError {
	e.GofastaError.Path = path
	return e
}

// WithTimestamp sets the error occurrence timestamp (overrides GofastaError method for type consistency)
func (e *UnauthorizedError) WithTimestamp(timestamp string) *UnauthorizedError {
	e.GofastaError.Timestamp = timestamp
	return e
}

// WithCurrentTimestamp sets the error occurrence timestamp to current time (overrides GofastaError method for type consistency)
func (e *UnauthorizedError) WithCurrentTimestamp() *UnauthorizedError {
	e.GofastaError.Timestamp = time.Now().Format(time.RFC3339)
	return e
}

// ForbiddenError represents forbidden access errors (403)
// Extends GofastaError with specific functionality for authorization failures
type ForbiddenError struct {
	*GofastaError
	RequiredPermissions []string               `json:"requiredPermissions,omitempty"` // Required permissions for access
	UserPermissions     []string               `json:"userPermissions,omitempty"`     // User's current permissions
	Resource            string                 `json:"resource,omitempty"`            // Protected resource identifier
	Action              string                 `json:"action,omitempty"`              // Attempted action
	AccessContext       map[string]interface{} `json:"accessContext,omitempty"`       // Additional access control context
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string) *ForbiddenError {
	if message == "" {
		message = "Forbidden"
	}
	return &ForbiddenError{
		GofastaError: NewGofastaError("FORBIDDEN", message, 403),
	}
}

// NewForbiddenErrorf creates a new forbidden error with formatted message
func NewForbiddenErrorf(format string, args ...interface{}) *ForbiddenError {
	return &ForbiddenError{
		GofastaError: NewGofastaError("FORBIDDEN", fmt.Sprintf(format, args...), 403),
	}
}

// NewForbiddenErrorWithCause creates a new forbidden error with underlying cause
func NewForbiddenErrorWithCause(message string, cause error) *ForbiddenError {
	if message == "" {
		message = "Forbidden"
	}
	forbiddenErr := &ForbiddenError{
		GofastaError: NewGofastaError("FORBIDDEN", message, 403),
	}
	forbiddenErr.GofastaError.Cause = cause
	return forbiddenErr
}

// WithRequiredPermissions sets the required permissions for access
func (e *ForbiddenError) WithRequiredPermissions(permissions ...string) *ForbiddenError {
	e.RequiredPermissions = append(e.RequiredPermissions, permissions...)
	return e
}

// WithUserPermissions sets the user's current permissions
func (e *ForbiddenError) WithUserPermissions(permissions ...string) *ForbiddenError {
	e.UserPermissions = append(e.UserPermissions, permissions...)
	return e
}

// WithResource sets the protected resource identifier
func (e *ForbiddenError) WithResource(resource string) *ForbiddenError {
	e.Resource = resource
	return e
}

// WithAction sets the attempted action
func (e *ForbiddenError) WithAction(action string) *ForbiddenError {
	e.Action = action
	return e
}

// WithAccessContext adds access control context information
func (e *ForbiddenError) WithAccessContext(key string, value interface{}) *ForbiddenError {
	if e.AccessContext == nil {
		e.AccessContext = make(map[string]interface{})
	}
	e.AccessContext[key] = value
	return e
}

// WithAllAccessContext sets all access context at once
func (e *ForbiddenError) WithAllAccessContext(context map[string]interface{}) *ForbiddenError {
	e.AccessContext = context
	return e
}

// WithCause adds an underlying cause to the error (overrides GofastaError method for type consistency)
func (e *ForbiddenError) WithCause(cause error) *ForbiddenError {
	e.GofastaError.Cause = cause
	return e
}

// WithMetadata adds additional metadata to the error (overrides GofastaError method for type consistency)
func (e *ForbiddenError) WithMetadata(key string, value interface{}) *ForbiddenError {
	if e.GofastaError.Metadata == nil {
		e.GofastaError.Metadata = make(map[string]interface{})
	}
	e.GofastaError.Metadata[key] = value
	return e
}

// WithAllMetadata sets all metadata at once (overrides GofastaError method for type consistency)
func (e *ForbiddenError) WithAllMetadata(metadata map[string]interface{}) *ForbiddenError {
	e.GofastaError.Metadata = metadata
	return e
}

// WithPath sets the request path where the error occurred (overrides GofastaError method for type consistency)
func (e *ForbiddenError) WithPath(path string) *ForbiddenError {
	e.GofastaError.Path = path
	return e
}

// WithTimestamp sets the error occurrence timestamp (overrides GofastaError method for type consistency)
func (e *ForbiddenError) WithTimestamp(timestamp string) *ForbiddenError {
	e.GofastaError.Timestamp = timestamp
	return e
}

// WithCurrentTimestamp sets the error occurrence timestamp to current time (overrides GofastaError method for type consistency)
func (e *ForbiddenError) WithCurrentTimestamp() *ForbiddenError {
	e.GofastaError.Timestamp = time.Now().Format(time.RFC3339)
	return e
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, id string) *GofastaError {
	return NewGofastaError("NOT_FOUND", fmt.Sprintf("%s with ID %s not found", resource, id), 404)
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

// Additional HTTP error creators following the roadmap

// NewMethodNotAllowedError creates a new method not allowed error
func NewMethodNotAllowedError(message string) *GofastaError {
	if message == "" {
		message = "Method Not Allowed"
	}
	return NewGofastaError("METHOD_NOT_ALLOWED", message, 405)
}

// NewNotAcceptableError creates a new not acceptable error
func NewNotAcceptableError(message string) *GofastaError {
	if message == "" {
		message = "Not Acceptable"
	}
	return NewGofastaError("NOT_ACCEPTABLE", message, 406)
}

// NewRequestTimeoutError creates a new request timeout error
func NewRequestTimeoutError(message string) *GofastaError {
	if message == "" {
		message = "Request Timeout"
	}
	return NewGofastaError("REQUEST_TIMEOUT", message, 408)
}

// NewGoneError creates a new gone error
func NewGoneError(message string) *GofastaError {
	if message == "" {
		message = "Gone"
	}
	return NewGofastaError("GONE", message, 410)
}

// NewPayloadTooLargeError creates a new payload too large error
func NewPayloadTooLargeError(message string) *GofastaError {
	if message == "" {
		message = "Payload Too Large"
	}
	return NewGofastaError("PAYLOAD_TOO_LARGE", message, 413)
}

// NewUnsupportedMediaTypeError creates a new unsupported media type error
func NewUnsupportedMediaTypeError(message string) *GofastaError {
	if message == "" {
		message = "Unsupported Media Type"
	}
	return NewGofastaError("UNSUPPORTED_MEDIA_TYPE", message, 415)
}

// NewUnprocessableEntityError creates a new unprocessable entity error
func NewUnprocessableEntityError(message string) *GofastaError {
	if message == "" {
		message = "Unprocessable Entity"
	}
	return NewGofastaError("UNPROCESSABLE_ENTITY", message, 422)
}

// NewTooManyRequestsError creates a new too many requests error
func NewTooManyRequestsError(message string) *GofastaError {
	if message == "" {
		message = "Too Many Requests"
	}
	return NewGofastaError("TOO_MANY_REQUESTS", message, 429)
}

// NewNotImplementedError creates a new not implemented error
func NewNotImplementedError(message string) *GofastaError {
	if message == "" {
		message = "Not Implemented"
	}
	return NewGofastaError("NOT_IMPLEMENTED", message, 501)
}

// NewBadGatewayError creates a new bad gateway error
func NewBadGatewayError(message string) *GofastaError {
	if message == "" {
		message = "Bad Gateway"
	}
	return NewGofastaError("BAD_GATEWAY", message, 502)
}

// NewServiceUnavailableError creates a new service unavailable error
func NewServiceUnavailableError(message string) *GofastaError {
	if message == "" {
		message = "Service Unavailable"
	}
	return NewGofastaError("SERVICE_UNAVAILABLE", message, 503)
}

// NewGatewayTimeoutError creates a new gateway timeout error
func NewGatewayTimeoutError(message string) *GofastaError {
	if message == "" {
		message = "Gateway Timeout"
	}
	return NewGofastaError("GATEWAY_TIMEOUT", message, 504)
}

// Error helper functions

// IsGofastaError checks if an error is a GofastaError and optionally matches status code or code
func IsGofastaError(err error, codes ...string) bool {
	var gofastaErr *GofastaError
	if !AsGofastaError(err, &gofastaErr) {
		return false
	}
	
	if len(codes) == 0 {
		return true
	}
	
	for _, code := range codes {
		if gofastaErr.Code == code {
			return true
		}
	}
	return false
}

// IsGofastaErrorWithStatus checks if an error is a GofastaError with specific status codes
func IsGofastaErrorWithStatus(err error, statusCodes ...int) bool {
	var gofastaErr *GofastaError
	if !AsGofastaError(err, &gofastaErr) {
		return false
	}
	
	if len(statusCodes) == 0 {
		return true
	}
	
	for _, statusCode := range statusCodes {
		if gofastaErr.StatusCode == statusCode {
			return true
		}
	}
	return false
}

// GetGofastaError extracts GofastaError from an error chain, returns nil if not found
func GetGofastaError(err error) *GofastaError {
	var gofastaErr *GofastaError
	if AsGofastaError(err, &gofastaErr) {
		return gofastaErr
	}
	return nil
}

// AsGofastaError is a helper function for error type assertion
func AsGofastaError(err error, target **GofastaError) bool {
	if err == nil {
		return false
	}
	
	// Direct GofastaError
	if gofastaErr, ok := err.(*GofastaError); ok {
		*target = gofastaErr
		return true
	}
	
	// ValidationError embeds *GofastaError
	if validationErr, ok := err.(*ValidationError); ok {
		*target = validationErr.GofastaError
		return true
	}
	
	// BadRequestError embeds *GofastaError
	if badRequestErr, ok := err.(*BadRequestError); ok {
		*target = badRequestErr.GofastaError
		return true
	}
	
	// UnauthorizedError embeds *GofastaError
	if unauthorizedErr, ok := err.(*UnauthorizedError); ok {
		*target = unauthorizedErr.GofastaError
		return true
	}
	
	// ForbiddenError embeds *GofastaError
	if forbiddenErr, ok := err.(*ForbiddenError); ok {
		*target = forbiddenErr.GofastaError
		return true
	}
	
	return false
}

// GetStatusText returns the standard HTTP status text for a status code
func GetStatusText(statusCode int) string {
	statusTexts := map[int]string{
		// 2xx Success
		200: "OK",
		201: "Created",
		202: "Accepted",
		204: "No Content",

		// 3xx Redirection
		301: "Moved Permanently",
		302: "Found",
		304: "Not Modified",

		// 4xx Client Errors
		400: "Bad Request",
		401: "Unauthorized",
		402: "Payment Required",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		406: "Not Acceptable",
		407: "Proxy Authentication Required",
		408: "Request Timeout",
		409: "Conflict",
		410: "Gone",
		411: "Length Required",
		412: "Precondition Failed",
		413: "Payload Too Large",
		414: "URI Too Long",
		415: "Unsupported Media Type",
		416: "Range Not Satisfiable",
		417: "Expectation Failed",
		418: "I'm a teapot",
		421: "Misdirected Request",
		422: "Unprocessable Entity",
		423: "Locked",
		424: "Failed Dependency",
		425: "Too Early",
		426: "Upgrade Required",
		428: "Precondition Required",
		429: "Too Many Requests",
		431: "Request Header Fields Too Large",
		451: "Unavailable For Legal Reasons",

		// 5xx Server Errors
		500: "Internal Server Error",
		501: "Not Implemented",
		502: "Bad Gateway",
		503: "Service Unavailable",
		504: "Gateway Timeout",
		505: "HTTP Version Not Supported",
		506: "Variant Also Negotiates",
		507: "Insufficient Storage",
		508: "Loop Detected",
		510: "Not Extended",
		511: "Network Authentication Required",
	}

	if text, exists := statusTexts[statusCode]; exists {
		return text
	}
	return "Unknown Status"
}

// JoinErrorMessages joins multiple error messages into a single string
func JoinErrorMessages(errors []error, separator string) string {
	if len(errors) == 0 {
		return ""
	}
	
	messages := make([]string, 0, len(errors))
	for _, err := range errors {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	
	return strings.Join(messages, separator)
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