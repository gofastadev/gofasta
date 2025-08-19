package core

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGofastaError(t *testing.T) {
	err := NewGofastaError("TEST_ERROR", "This is a test error", 400)
	
	if err == nil {
		t.Fatal("NewGofastaError() returned nil")
	}
	
	if err.Code != "TEST_ERROR" {
		t.Errorf("Expected error code 'TEST_ERROR', got %s", err.Code)
	}
	
	if err.Message != "This is a test error" {
		t.Errorf("Expected error message 'This is a test error', got %s", err.Message)
	}
	
	if err.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", err.StatusCode)
	}
	
	expectedError := "TEST_ERROR: This is a test error"
	if err.Error() != expectedError {
		t.Errorf("Expected Error() to return '%s', got %s", expectedError, err.Error())
	}
}

func TestGofastaError_WithCause(t *testing.T) {
	cause := errors.New("original error")
	err := NewGofastaError("WRAPPED_ERROR", "Wrapped error", 500)
	err.Cause = cause
	
	if err.Cause != cause {
		t.Error("Cause not set correctly")
	}
}

func TestGofastaError_WithMetadata(t *testing.T) {
	err := NewGofastaError("METADATA_ERROR", "Error with metadata", 422)
	err.Metadata = map[string]interface{}{
		"field":  "email",
		"reason": "invalid format",
	}
	
	if err.Metadata["field"] != "email" {
		t.Errorf("Expected metadata field 'email', got %v", err.Metadata["field"])
	}
	
	if err.Metadata["reason"] != "invalid format" {
		t.Errorf("Expected metadata reason 'invalid format', got %v", err.Metadata["reason"])
	}
}

func TestValidationError(t *testing.T) {
	validationErrors := []FieldError{
		{Field: "email", Message: "Email is required"},
		{Field: "password", Message: "Password must be at least 8 characters"},
	}
	
	err := NewValidationError("Validation failed", validationErrors)
	
	if err == nil {
		t.Fatal("NewValidationError() returned nil")
	}
	
	if err.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected error code 'VALIDATION_ERROR', got %s", err.Code)
	}
	
	if err.Message != "Validation failed" {
		t.Errorf("Expected error message 'Validation failed', got %s", err.Message)
	}
	
	if err.StatusCode != 422 {
		t.Errorf("Expected status code 422, got %d", err.StatusCode)
	}
	
	if len(err.Errors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(err.Errors))
	}
	
	if err.Errors[0].Field != "email" {
		t.Errorf("Expected first error field 'email', got %s", err.Errors[0].Field)
	}
	
	if err.Errors[1].Field != "password" {
		t.Errorf("Expected second error field 'password', got %s", err.Errors[1].Field)
	}
}

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("User", "123")
	
	if err == nil {
		t.Fatal("NewNotFoundError() returned nil")
	}
	
	if err.Code != "NOT_FOUND" {
		t.Errorf("Expected error code 'NOT_FOUND', got %s", err.Code)
	}
	
	expectedMessage := "User with ID 123 not found"
	if err.Message != expectedMessage {
		t.Errorf("Expected error message '%s', got %s", expectedMessage, err.Message)
	}
	
	if err.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", err.StatusCode)
	}
}

func TestUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("Invalid credentials")
	
	if err == nil {
		t.Fatal("NewUnauthorizedError() returned nil")
	}
	
	if err.Code != "UNAUTHORIZED" {
		t.Errorf("Expected error code 'UNAUTHORIZED', got %s", err.Code)
	}
	
	if err.Message != "Invalid credentials" {
		t.Errorf("Expected error message 'Invalid credentials', got %s", err.Message)
	}
	
	if err.StatusCode != 401 {
		t.Errorf("Expected status code 401, got %d", err.StatusCode)
	}
}

func TestForbiddenError(t *testing.T) {
	err := NewForbiddenError("Access denied")
	
	if err == nil {
		t.Fatal("NewForbiddenError() returned nil")
	}
	
	if err.Code != "FORBIDDEN" {
		t.Errorf("Expected error code 'FORBIDDEN', got %s", err.Code)
	}
	
	if err.Message != "Access denied" {
		t.Errorf("Expected error message 'Access denied', got %s", err.Message)
	}
	
	if err.StatusCode != 403 {
		t.Errorf("Expected status code 403, got %d", err.StatusCode)
	}
}

func TestBadRequestError(t *testing.T) {
	err := NewBadRequestError("Invalid request format")
	
	if err == nil {
		t.Fatal("NewBadRequestError() returned nil")
	}
	
	if err.Code != "BAD_REQUEST" {
		t.Errorf("Expected error code 'BAD_REQUEST', got %s", err.Code)
	}
	
	if err.Message != "Invalid request format" {
		t.Errorf("Expected error message 'Invalid request format', got %s", err.Message)
	}
	
	if err.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", err.StatusCode)
	}
}

func TestInternalServerError(t *testing.T) {
	err := NewInternalServerError("Database connection failed")
	
	if err == nil {
		t.Fatal("NewInternalServerError() returned nil")
	}
	
	if err.Code != "INTERNAL_SERVER_ERROR" {
		t.Errorf("Expected error code 'INTERNAL_SERVER_ERROR', got %s", err.Code)
	}
	
	if err.Message != "Database connection failed" {
		t.Errorf("Expected error message 'Database connection failed', got %s", err.Message)
	}
	
	if err.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", err.StatusCode)
	}
}

func TestConflictError(t *testing.T) {
	err := NewConflictError("Email already exists")
	
	if err == nil {
		t.Fatal("NewConflictError() returned nil")
	}
	
	if err.Code != "CONFLICT" {
		t.Errorf("Expected error code 'CONFLICT', got %s", err.Code)
	}
	
	if err.Message != "Email already exists" {
		t.Errorf("Expected error message 'Email already exists', got %s", err.Message)
	}
	
	if err.StatusCode != 409 {
		t.Errorf("Expected status code 409, got %d", err.StatusCode)
	}
}

func TestFieldError(t *testing.T) {
	fieldError := FieldError{
		Field:   "username",
		Message: "Username is required",
		Value:   "",
		Tag:     "required",
	}

	if fieldError.Field != "username" {
		t.Errorf("Expected field 'username', got %s", fieldError.Field)
	}

	if fieldError.Message != "Username is required" {
		t.Errorf("Expected message 'Username is required', got %s", fieldError.Message)
	}

	if fieldError.Value != "" {
		t.Errorf("Expected value '', got %v", fieldError.Value)
	}

	if fieldError.Tag != "required" {
		t.Errorf("Expected tag 'required', got %s", fieldError.Tag)
	}
}

func TestGofastaError_IsType(t *testing.T) {
	validationErr := NewValidationError("Validation failed", []FieldError{})
	notFoundErr := NewNotFoundError("User", "123")
	unauthorizedErr := NewUnauthorizedError("Invalid token")

	// Test validation error - ValidationError wraps GofastaError
	if !IsValidationError(validationErr) {
		t.Error("IsValidationError() should return true for validation error")
	}

	if IsNotFoundError(validationErr) {
		t.Error("IsNotFoundError() should return false for validation error")
	}

	// Test not found error
	if IsNotFoundError(notFoundErr) == false {
		t.Error("IsNotFoundError() should return true for not found error")
	}

	if IsValidationError(notFoundErr) {
		t.Error("IsValidationError() should return false for not found error")
	}

	// Test unauthorized error
	if IsUnauthorizedError(unauthorizedErr) == false {
		t.Error("IsUnauthorizedError() should return true for unauthorized error")
	}

	if IsNotFoundError(unauthorizedErr) {
		t.Error("IsNotFoundError() should return false for unauthorized error")
	}
}

func TestGofastaError_IsType_WithNonGofastaError(t *testing.T) {
	regularErr := errors.New("regular error")
	
	if IsValidationError(regularErr) {
		t.Error("IsValidationError() should return false for regular error")
	}
	
	if IsNotFoundError(regularErr) {
		t.Error("IsNotFoundError() should return false for regular error")
	}
	
	if IsUnauthorizedError(regularErr) {
		t.Error("IsUnauthorizedError() should return false for regular error")
	}
}

// Helper functions for error type checking
func IsValidationError(err error) bool {
	return IsGofastaError(err, "VALIDATION_ERROR")
}

func IsNotFoundError(err error) bool {
	return IsGofastaError(err, "NOT_FOUND")
}

func IsUnauthorizedError(err error) bool {
	return IsGofastaError(err, "UNAUTHORIZED")
}

// Tests for enhanced error functionality

func TestGofastaError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewGofastaError("WRAPPED_ERROR", "Wrapped error", 500)
	err.Cause = cause

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, expected %v", unwrapped, cause)
	}

	// Test with no cause
	errNoCause := NewGofastaError("NO_CAUSE", "Error without cause", 404)
	unwrappedNil := errNoCause.Unwrap()
	if unwrappedNil != nil {
		t.Errorf("Unwrap() = %v, expected nil", unwrappedNil)
	}
}

func TestGofastaError_Is(t *testing.T) {
	err400 := NewGofastaError("BAD_REQUEST", "Bad request", 400)
	err404 := NewGofastaError("NOT_FOUND", "Not found", 404)
	anotherErr400 := NewGofastaError("ANOTHER_BAD_REQUEST", "Different message", 400)
	sameCodeErr := NewGofastaError("BAD_REQUEST", "Different message", 500)

	// Same status code should match
	if !err400.Is(anotherErr400) {
		t.Error("Expected errors with same status code to match")
	}

	// Same code should match
	if !err400.Is(sameCodeErr) {
		t.Error("Expected errors with same code to match")
	}

	// Different status codes and codes should not match
	if err400.Is(err404) {
		t.Error("Expected errors with different status codes and codes not to match")
	}

	// Non-GofastaError should not match
	regularErr := errors.New("regular error")
	if err400.Is(regularErr) {
		t.Error("Expected GofastaError not to match regular error")
	}
}

func TestGofastaError_As(t *testing.T) {
	err := NewGofastaError("TEST_ERROR", "Test error", 400)
	
	var target *GofastaError
	if !err.As(&target) {
		t.Error("Expected As to return true for GofastaError target")
	}
	
	if target != err {
		t.Error("Expected target to be set to the original error")
	}

	// Test with wrong type
	var wrongTarget *string
	if err.As(&wrongTarget) {
		t.Error("Expected As to return false for wrong target type")
	}
}

func TestGofastaError_MethodChaining(t *testing.T) {
	cause := errors.New("underlying cause")
	metadata := map[string]interface{}{
		"field": "value",
		"count": 123,
	}
	path := "/api/test"
	timestamp := "2024-01-01T00:00:00Z"

	// Test comprehensive method chaining
	err := NewGofastaError("CHAIN_TEST", "Chain test error", 400).
		WithCause(cause).
		WithAllMetadata(metadata).
		WithPath(path).
		WithTimestamp(timestamp)

	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}

	if err.Path != path {
		t.Errorf("Expected path %v, got %v", path, err.Path)
	}

	if err.Timestamp != timestamp {
		t.Errorf("Expected timestamp %v, got %v", timestamp, err.Timestamp)
	}

	if err.Metadata["field"] != "value" {
		t.Error("Metadata not properly set")
	}

	if err.Metadata["count"] != 123 {
		t.Error("Metadata count not properly set")
	}
}

func TestGofastaError_WithMetadataChaining(t *testing.T) {
	err := NewGofastaError("METADATA_TEST", "Metadata test", 400)
	
	// Test individual metadata setting
	err.WithMetadata("key1", "value1").
		WithMetadata("key2", 42).
		WithMetadata("key3", true)

	if err.Metadata["key1"] != "value1" {
		t.Error("Metadata key1 not set correctly")
	}

	if err.Metadata["key2"] != 42 {
		t.Error("Metadata key2 not set correctly")
	}

	if err.Metadata["key3"] != true {
		t.Error("Metadata key3 not set correctly")
	}
}

func TestGofastaError_WithCurrentTimestamp(t *testing.T) {
	err := NewGofastaError("TIMESTAMP_TEST", "Timestamp test", 400)
	err.WithCurrentTimestamp()

	if err.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}

	// Basic validation that it's in RFC3339 format (just check length)
	if len(err.Timestamp) < 19 {
		t.Error("Timestamp appears to be in wrong format")
	}
}

func TestNewErrorTypes(t *testing.T) {
	tests := []struct {
		name       string
		creator    func(string) *GofastaError
		message    string
		statusCode int
		errorCode  string
	}{
		{"MethodNotAllowed", NewMethodNotAllowedError, "Custom message", 405, "METHOD_NOT_ALLOWED"},
		{"MethodNotAllowed default", NewMethodNotAllowedError, "", 405, "METHOD_NOT_ALLOWED"},
		{"NotAcceptable", NewNotAcceptableError, "Custom message", 406, "NOT_ACCEPTABLE"},
		{"RequestTimeout", NewRequestTimeoutError, "Timeout occurred", 408, "REQUEST_TIMEOUT"},
		{"Gone", NewGoneError, "Resource gone", 410, "GONE"},
		{"PayloadTooLarge", NewPayloadTooLargeError, "Too big", 413, "PAYLOAD_TOO_LARGE"},
		{"UnsupportedMediaType", NewUnsupportedMediaTypeError, "Wrong type", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"UnprocessableEntity", NewUnprocessableEntityError, "Cannot process", 422, "UNPROCESSABLE_ENTITY"},
		{"TooManyRequests", NewTooManyRequestsError, "Rate limited", 429, "TOO_MANY_REQUESTS"},
		{"NotImplemented", NewNotImplementedError, "Not done yet", 501, "NOT_IMPLEMENTED"},
		{"BadGateway", NewBadGatewayError, "Upstream failed", 502, "BAD_GATEWAY"},
		{"ServiceUnavailable", NewServiceUnavailableError, "Service down", 503, "SERVICE_UNAVAILABLE"},
		{"GatewayTimeout", NewGatewayTimeoutError, "Gateway slow", 504, "GATEWAY_TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creator(tt.message)

			if err.StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, err.StatusCode)
			}

			if err.Code != tt.errorCode {
				t.Errorf("Expected error code %s, got %s", tt.errorCode, err.Code)
			}

			expectedMessage := tt.message
			if expectedMessage == "" {
				expectedMessage = GetStatusText(tt.statusCode)
			}

			if err.Message != expectedMessage {
				t.Errorf("Expected message %s, got %s", expectedMessage, err.Message)
			}
		})
	}
}

func TestIsGofastaError(t *testing.T) {
	gofastaErr := NewGofastaError("TEST_ERROR", "Test error", 400)
	regularErr := errors.New("regular error")

	// Test with GofastaError
	if !IsGofastaError(gofastaErr) {
		t.Error("Expected IsGofastaError to return true for GofastaError")
	}

	// Test with code matching
	if !IsGofastaError(gofastaErr, "TEST_ERROR") {
		t.Error("Expected IsGofastaError to return true for matching code")
	}

	// Test with non-matching code
	if IsGofastaError(gofastaErr, "DIFFERENT_ERROR") {
		t.Error("Expected IsGofastaError to return false for non-matching code")
	}

	// Test with multiple codes
	if !IsGofastaError(gofastaErr, "WRONG_ERROR", "TEST_ERROR", "ANOTHER_ERROR") {
		t.Error("Expected IsGofastaError to return true when code matches one of multiple")
	}

	// Test with regular error
	if IsGofastaError(regularErr) {
		t.Error("Expected IsGofastaError to return false for regular error")
	}
}

func TestIsGofastaErrorWithStatus(t *testing.T) {
	gofastaErr := NewGofastaError("TEST_ERROR", "Test error", 400)
	regularErr := errors.New("regular error")

	// Test with status code matching
	if !IsGofastaErrorWithStatus(gofastaErr, 400) {
		t.Error("Expected IsGofastaErrorWithStatus to return true for matching status code")
	}

	// Test with non-matching status code
	if IsGofastaErrorWithStatus(gofastaErr, 404) {
		t.Error("Expected IsGofastaErrorWithStatus to return false for non-matching status code")
	}

	// Test with multiple status codes
	if !IsGofastaErrorWithStatus(gofastaErr, 404, 400, 500) {
		t.Error("Expected IsGofastaErrorWithStatus to return true when status matches one of multiple")
	}

	// Test with regular error
	if IsGofastaErrorWithStatus(regularErr, 400) {
		t.Error("Expected IsGofastaErrorWithStatus to return false for regular error")
	}
}

func TestGetGofastaError(t *testing.T) {
	gofastaErr := NewGofastaError("TEST_ERROR", "Test error", 400)
	regularErr := errors.New("regular error")

	// Test with GofastaError
	result := GetGofastaError(gofastaErr)
	if result != gofastaErr {
		t.Error("Expected GetGofastaError to return the GofastaError")
	}

	// Test with regular error
	result = GetGofastaError(regularErr)
	if result != nil {
		t.Error("Expected GetGofastaError to return nil for regular error")
	}

	// Test with nil error
	result = GetGofastaError(nil)
	if result != nil {
		t.Error("Expected GetGofastaError to return nil for nil error")
	}
}

func TestGetStatusText(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   string
	}{
		{200, "OK"},
		{201, "Created"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{404, "Not Found"},
		{405, "Method Not Allowed"},
		{422, "Unprocessable Entity"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{503, "Service Unavailable"},
		{999, "Unknown Status"}, // Unknown status code
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := GetStatusText(tt.statusCode)
			if result != tt.expected {
				t.Errorf("GetStatusText(%d) = %s, expected %s", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestJoinErrorMessages(t *testing.T) {
	tests := []struct {
		name      string
		errors    []error
		separator string
		expected  string
	}{
		{
			name:      "empty slice",
			errors:    []error{},
			separator: "; ",
			expected:  "",
		},
		{
			name:      "single error",
			errors:    []error{errors.New("error 1")},
			separator: "; ",
			expected:  "error 1",
		},
		{
			name: "multiple errors",
			errors: []error{
				errors.New("error 1"),
				errors.New("error 2"),
				errors.New("error 3"),
			},
			separator: "; ",
			expected:  "error 1; error 2; error 3",
		},
		{
			name: "errors with nil values",
			errors: []error{
				errors.New("error 1"),
				nil,
				errors.New("error 3"),
			},
			separator: " | ",
			expected:  "error 1 | error 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinErrorMessages(tt.errors, tt.separator)
			if result != tt.expected {
				t.Errorf("JoinErrorMessages() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

// Tests for BadRequestError

func TestBadRequestError_Basic(t *testing.T) {
	err := NewBadRequestError("Invalid input data")

	if err == nil {
		t.Fatal("NewBadRequestError() returned nil")
	}

	if err.Code != "BAD_REQUEST" {
		t.Errorf("Expected error code 'BAD_REQUEST', got %s", err.Code)
	}

	if err.Message != "Invalid input data" {
		t.Errorf("Expected error message 'Invalid input data', got %s", err.Message)
	}

	if err.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", err.StatusCode)
	}
}

func TestBadRequestError_DefaultMessage(t *testing.T) {
	err := NewBadRequestError("")

	expectedMessage := "Bad Request"
	if err.Message != expectedMessage {
		t.Errorf("Expected default message '%s', got %s", expectedMessage, err.Message)
	}
}

func TestBadRequestErrorf(t *testing.T) {
	err := NewBadRequestErrorf("Invalid %s field: %d", "age", 150)

	expectedMessage := "Invalid age field: 150"
	if err.Message != expectedMessage {
		t.Errorf("Expected formatted message '%s', got %s", expectedMessage, err.Message)
	}
}

func TestBadRequestErrorWithCause(t *testing.T) {
	cause := errors.New("validation failed")
	err := NewBadRequestErrorWithCause("Request validation error", cause)

	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}

	if err.Message != "Request validation error" {
		t.Errorf("Expected message 'Request validation error', got %s", err.Message)
	}
}

func TestBadRequestError_WithInvalidFields(t *testing.T) {
	err := NewBadRequestError("Field validation failed").
		WithInvalidFields("email", "age").
		WithInvalidFields("name")

	expectedFields := []string{"email", "age", "name"}
	if len(err.InvalidFields) != len(expectedFields) {
		t.Errorf("Expected %d invalid fields, got %d", len(expectedFields), len(err.InvalidFields))
	}

	for i, field := range expectedFields {
		if i >= len(err.InvalidFields) || err.InvalidFields[i] != field {
			t.Errorf("Expected invalid field %s at index %d", field, i)
		}
	}
}

func TestBadRequestError_WithSuggestions(t *testing.T) {
	err := NewBadRequestError("Invalid format").
		WithSuggestions("Use ISO 8601 format", "Include timezone").
		WithSuggestions("Check documentation")

	expectedSuggestions := []string{"Use ISO 8601 format", "Include timezone", "Check documentation"}
	if len(err.Suggestions) != len(expectedSuggestions) {
		t.Errorf("Expected %d suggestions, got %d", len(expectedSuggestions), len(err.Suggestions))
	}

	for i, suggestion := range expectedSuggestions {
		if i >= len(err.Suggestions) || err.Suggestions[i] != suggestion {
			t.Errorf("Expected suggestion %s at index %d", suggestion, i)
		}
	}
}

func TestBadRequestError_WithRequestInfo(t *testing.T) {
	err := NewBadRequestError("Request processing failed").
		WithRequestInfo("contentType", "application/json").
		WithRequestInfo("contentLength", 1024).
		WithRequestInfo("method", "POST")

	if err.RequestInfo["contentType"] != "application/json" {
		t.Error("Request info contentType not set correctly")
	}

	if err.RequestInfo["contentLength"] != 1024 {
		t.Error("Request info contentLength not set correctly")
	}

	if err.RequestInfo["method"] != "POST" {
		t.Error("Request info method not set correctly")
	}
}

func TestBadRequestError_WithAllRequestInfo(t *testing.T) {
	requestInfo := map[string]interface{}{
		"userAgent": "Mozilla/5.0",
		"ipAddress": "192.168.1.1",
		"referer":   "https://example.com",
	}

	err := NewBadRequestError("Request analysis failed").
		WithAllRequestInfo(requestInfo)

	if err.RequestInfo["userAgent"] != "Mozilla/5.0" {
		t.Error("Request info userAgent not set correctly")
	}

	if err.RequestInfo["ipAddress"] != "192.168.1.1" {
		t.Error("Request info ipAddress not set correctly")
	}

	if err.RequestInfo["referer"] != "https://example.com" {
		t.Error("Request info referer not set correctly")
	}
}

func TestBadRequestError_MethodChaining(t *testing.T) {
	cause := errors.New("parsing failed")
	
	err := NewBadRequestError("Comprehensive bad request error").
		WithCause(cause).
		WithInvalidFields("email", "phone").
		WithSuggestions("Check email format", "Use international format for phone").
		WithRequestInfo("contentType", "application/json").
		WithMetadata("validationRules", "strict").
		WithPath("/api/users").
		WithCurrentTimestamp()

	// Test cause
	if err.Cause != cause {
		t.Error("Cause not set correctly")
	}

	// Test invalid fields
	if len(err.InvalidFields) != 2 || err.InvalidFields[0] != "email" || err.InvalidFields[1] != "phone" {
		t.Error("Invalid fields not set correctly")
	}

	// Test suggestions
	if len(err.Suggestions) != 2 || err.Suggestions[0] != "Check email format" {
		t.Error("Suggestions not set correctly")
	}

	// Test request info
	if err.RequestInfo["contentType"] != "application/json" {
		t.Error("Request info not set correctly")
	}

	// Test metadata (from embedded GofastaError)
	if err.Metadata["validationRules"] != "strict" {
		t.Error("Metadata not set correctly")
	}

	// Test path
	if err.Path != "/api/users" {
		t.Error("Path not set correctly")
	}

	// Test timestamp
	if err.Timestamp == "" {
		t.Error("Timestamp not set")
	}
}

func TestBadRequestError_ErrorMessage(t *testing.T) {
	// Test error message without cause
	err := NewBadRequestError("Simple bad request")
	expectedMsg := "BAD_REQUEST: Simple bad request"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test error message with cause
	cause := errors.New("validation failure")
	err = NewBadRequestErrorWithCause("Bad request with cause", cause)
	expectedMsgWithCause := "BAD_REQUEST: Bad request with cause (caused by: validation failure)"
	if err.Error() != expectedMsgWithCause {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsgWithCause, err.Error())
	}
}

func TestBadRequestError_AsGofastaError(t *testing.T) {
	badReqErr := NewBadRequestError("Test bad request")
	
	var gofastaErr *GofastaError
	if !AsGofastaError(badReqErr, &gofastaErr) {
		t.Error("Expected AsGofastaError to return true for BadRequestError")
	}

	if gofastaErr == nil {
		t.Error("Expected GofastaError to be extracted from BadRequestError")
	}

	if gofastaErr.Code != "BAD_REQUEST" {
		t.Errorf("Expected extracted error code 'BAD_REQUEST', got %s", gofastaErr.Code)
	}
}

func TestBadRequestError_IsGofastaError(t *testing.T) {
	badReqErr := NewBadRequestError("Test bad request")

	// Test with IsGofastaError
	if !IsGofastaError(badReqErr) {
		t.Error("Expected IsGofastaError to return true for BadRequestError")
	}

	// Test with specific code matching
	if !IsGofastaError(badReqErr, "BAD_REQUEST") {
		t.Error("Expected IsGofastaError to return true for matching BAD_REQUEST code")
	}

	// Test with wrong code
	if IsGofastaError(badReqErr, "NOT_FOUND") {
		t.Error("Expected IsGofastaError to return false for non-matching code")
	}

	// Test with status code matching
	if !IsGofastaErrorWithStatus(badReqErr, 400) {
		t.Error("Expected IsGofastaErrorWithStatus to return true for matching status code 400")
	}

	// Test with wrong status code
	if IsGofastaErrorWithStatus(badReqErr, 404) {
		t.Error("Expected IsGofastaErrorWithStatus to return false for non-matching status code")
	}
}

func TestBadRequestError_GetGofastaError(t *testing.T) {
	badReqErr := NewBadRequestError("Test bad request")
	
	extractedErr := GetGofastaError(badReqErr)
	if extractedErr == nil {
		t.Error("Expected GetGofastaError to extract GofastaError from BadRequestError")
	}

	if extractedErr.Code != "BAD_REQUEST" {
		t.Errorf("Expected extracted error code 'BAD_REQUEST', got %s", extractedErr.Code)
	}

	if extractedErr.StatusCode != 400 {
		t.Errorf("Expected extracted status code 400, got %d", extractedErr.StatusCode)
	}
}

func TestBadRequestError_ComplexScenario(t *testing.T) {
	// Simulate a complex validation scenario
	parseErr := errors.New("JSON parsing failed at line 15")
	
	err := NewBadRequestErrorf("Request body validation failed: %d errors found", 3).
		WithCause(parseErr).
		WithInvalidFields("user.email", "user.age", "preferences.theme").
		WithSuggestions(
			"Ensure email follows RFC 5322 format",
			"Age must be between 13 and 120",
			"Theme must be one of: light, dark, auto",
		).
		WithRequestInfo("contentType", "application/json").
		WithRequestInfo("contentLength", 2048).
		WithRequestInfo("userAgent", "MyApp/1.0").
		WithMetadata("validationEngine", "JSONSchema").
		WithMetadata("schemaVersion", "v2.1").
		WithPath("/api/v1/users/profile").
		WithCurrentTimestamp()

	// Validate all properties are set correctly
	if err.Code != "BAD_REQUEST" {
		t.Error("Error code not set correctly")
	}

	if err.StatusCode != 400 {
		t.Error("Status code not set correctly")
	}

	if err.Cause != parseErr {
		t.Error("Cause not set correctly")
	}

	if len(err.InvalidFields) != 3 {
		t.Error("Invalid fields not set correctly")
	}

	if len(err.Suggestions) != 3 {
		t.Error("Suggestions not set correctly")
	}

	if len(err.RequestInfo) != 3 {
		t.Error("Request info not set correctly")
	}

	if len(err.Metadata) != 2 {
		t.Error("Metadata not set correctly")
	}

	if err.Path != "/api/v1/users/profile" {
		t.Error("Path not set correctly")
	}

	if err.Timestamp == "" {
		t.Error("Timestamp not set")
	}

	// Test error message includes cause
	if !strings.Contains(err.Error(), "JSON parsing failed") {
		t.Error("Error message should include cause details")
	}
}

// Tests for UnauthorizedError

func TestUnauthorizedError_Basic(t *testing.T) {
	err := NewUnauthorizedError("Authentication required")

	if err == nil {
		t.Fatal("NewUnauthorizedError() returned nil")
	}

	if err.Code != "UNAUTHORIZED" {
		t.Errorf("Expected error code 'UNAUTHORIZED', got %s", err.Code)
	}

	if err.Message != "Authentication required" {
		t.Errorf("Expected error message 'Authentication required', got %s", err.Message)
	}

	if err.StatusCode != 401 {
		t.Errorf("Expected status code 401, got %d", err.StatusCode)
	}
}

func TestUnauthorizedError_DefaultMessage(t *testing.T) {
	err := NewUnauthorizedError("")

	expectedMessage := "Unauthorized"
	if err.Message != expectedMessage {
		t.Errorf("Expected default message '%s', got %s", expectedMessage, err.Message)
	}
}

func TestUnauthorizedErrorf(t *testing.T) {
	err := NewUnauthorizedErrorf("Invalid %s token: %s", "JWT", "expired")

	expectedMessage := "Invalid JWT token: expired"
	if err.Message != expectedMessage {
		t.Errorf("Expected formatted message '%s', got %s", expectedMessage, err.Message)
	}
}

func TestUnauthorizedErrorWithCause(t *testing.T) {
	cause := errors.New("token validation failed")
	err := NewUnauthorizedErrorWithCause("Authentication failed", cause)

	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}

	if err.Message != "Authentication failed" {
		t.Errorf("Expected message 'Authentication failed', got %s", err.Message)
	}
}

func TestUnauthorizedError_WithAuthScheme(t *testing.T) {
	err := NewUnauthorizedError("Token required").
		WithAuthScheme("Bearer").
		WithAuthScheme("Basic")

	// Should use the last set value
	if err.AuthScheme != "Basic" {
		t.Errorf("Expected auth scheme 'Basic', got %s", err.AuthScheme)
	}
}

func TestUnauthorizedError_WithRealm(t *testing.T) {
	err := NewUnauthorizedError("Access denied").
		WithRealm("Protected Area")

	if err.Realm != "Protected Area" {
		t.Errorf("Expected realm 'Protected Area', got %s", err.Realm)
	}
}

func TestUnauthorizedError_WithChallenges(t *testing.T) {
	err := NewUnauthorizedError("Authentication required").
		WithChallenges("Bearer", "Basic realm=\"API\"").
		WithChallenges("Digest realm=\"API\"")

	expectedChallenges := []string{"Bearer", "Basic realm=\"API\"", "Digest realm=\"API\""}
	if len(err.Challenges) != len(expectedChallenges) {
		t.Errorf("Expected %d challenges, got %d", len(expectedChallenges), len(err.Challenges))
	}

	for i, challenge := range expectedChallenges {
		if i >= len(err.Challenges) || err.Challenges[i] != challenge {
			t.Errorf("Expected challenge %s at index %d", challenge, i)
		}
	}
}

func TestUnauthorizedError_WithLoginUrl(t *testing.T) {
	loginUrl := "https://api.example.com/auth/login"
	err := NewUnauthorizedError("Login required").
		WithLoginUrl(loginUrl)

	if err.LoginUrl != loginUrl {
		t.Errorf("Expected login URL '%s', got %s", loginUrl, err.LoginUrl)
	}
}

func TestUnauthorizedError_WithAuthContext(t *testing.T) {
	err := NewUnauthorizedError("Token expired").
		WithAuthContext("tokenType", "JWT").
		WithAuthContext("expirationTime", "2024-01-01T00:00:00Z").
		WithAuthContext("userId", 12345)

	if err.AuthContext["tokenType"] != "JWT" {
		t.Error("Auth context tokenType not set correctly")
	}

	if err.AuthContext["expirationTime"] != "2024-01-01T00:00:00Z" {
		t.Error("Auth context expirationTime not set correctly")
	}

	if err.AuthContext["userId"] != 12345 {
		t.Error("Auth context userId not set correctly")
	}
}

func TestUnauthorizedError_WithAllAuthContext(t *testing.T) {
	authContext := map[string]interface{}{
		"provider":    "OAuth2",
		"clientId":    "client123",
		"scope":       "read write",
		"grantType":   "authorization_code",
	}

	err := NewUnauthorizedError("OAuth authentication failed").
		WithAllAuthContext(authContext)

	if err.AuthContext["provider"] != "OAuth2" {
		t.Error("Auth context provider not set correctly")
	}

	if err.AuthContext["clientId"] != "client123" {
		t.Error("Auth context clientId not set correctly")
	}

	if err.AuthContext["scope"] != "read write" {
		t.Error("Auth context scope not set correctly")
	}

	if err.AuthContext["grantType"] != "authorization_code" {
		t.Error("Auth context grantType not set correctly")
	}
}

func TestUnauthorizedError_MethodChaining(t *testing.T) {
	cause := errors.New("JWT token expired")
	
	err := NewUnauthorizedError("Comprehensive authentication error").
		WithCause(cause).
		WithAuthScheme("Bearer").
		WithRealm("API Gateway").
		WithChallenges("Bearer realm=\"API Gateway\"", "Basic").
		WithLoginUrl("https://auth.example.com/login").
		WithAuthContext("tokenType", "JWT").
		WithAuthContext("expired", true).
		WithMetadata("endpoint", "/api/secure").
		WithPath("/api/users/profile").
		WithCurrentTimestamp()

	// Test cause
	if err.Cause != cause {
		t.Error("Cause not set correctly")
	}

	// Test auth scheme
	if err.AuthScheme != "Bearer" {
		t.Error("Auth scheme not set correctly")
	}

	// Test realm
	if err.Realm != "API Gateway" {
		t.Error("Realm not set correctly")
	}

	// Test challenges
	if len(err.Challenges) != 2 || err.Challenges[0] != "Bearer realm=\"API Gateway\"" || err.Challenges[1] != "Basic" {
		t.Error("Challenges not set correctly")
	}

	// Test login URL
	if err.LoginUrl != "https://auth.example.com/login" {
		t.Error("Login URL not set correctly")
	}

	// Test auth context
	if err.AuthContext["tokenType"] != "JWT" || err.AuthContext["expired"] != true {
		t.Error("Auth context not set correctly")
	}

	// Test metadata (from embedded GofastaError)
	if err.Metadata["endpoint"] != "/api/secure" {
		t.Error("Metadata not set correctly")
	}

	// Test path
	if err.Path != "/api/users/profile" {
		t.Error("Path not set correctly")
	}

	// Test timestamp
	if err.Timestamp == "" {
		t.Error("Timestamp not set")
	}
}

func TestUnauthorizedError_ErrorMessage(t *testing.T) {
	// Test error message without cause
	err := NewUnauthorizedError("Simple authentication failure")
	expectedMsg := "UNAUTHORIZED: Simple authentication failure"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test error message with cause
	cause := errors.New("token signature invalid")
	err = NewUnauthorizedErrorWithCause("Authentication failed", cause)
	expectedMsgWithCause := "UNAUTHORIZED: Authentication failed (caused by: token signature invalid)"
	if err.Error() != expectedMsgWithCause {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsgWithCause, err.Error())
	}
}

func TestUnauthorizedError_AsGofastaError(t *testing.T) {
	unauthorizedErr := NewUnauthorizedError("Test unauthorized")
	
	var gofastaErr *GofastaError
	if !AsGofastaError(unauthorizedErr, &gofastaErr) {
		t.Error("Expected AsGofastaError to return true for UnauthorizedError")
	}

	if gofastaErr == nil {
		t.Error("Expected GofastaError to be extracted from UnauthorizedError")
	}

	if gofastaErr.Code != "UNAUTHORIZED" {
		t.Errorf("Expected extracted error code 'UNAUTHORIZED', got %s", gofastaErr.Code)
	}
}

func TestUnauthorizedError_IsGofastaError(t *testing.T) {
	unauthorizedErr := NewUnauthorizedError("Test unauthorized")

	// Test with IsGofastaError
	if !IsGofastaError(unauthorizedErr) {
		t.Error("Expected IsGofastaError to return true for UnauthorizedError")
	}

	// Test with specific code matching
	if !IsGofastaError(unauthorizedErr, "UNAUTHORIZED") {
		t.Error("Expected IsGofastaError to return true for matching UNAUTHORIZED code")
	}

	// Test with wrong code
	if IsGofastaError(unauthorizedErr, "NOT_FOUND") {
		t.Error("Expected IsGofastaError to return false for non-matching code")
	}

	// Test with status code matching
	if !IsGofastaErrorWithStatus(unauthorizedErr, 401) {
		t.Error("Expected IsGofastaErrorWithStatus to return true for matching status code 401")
	}

	// Test with wrong status code
	if IsGofastaErrorWithStatus(unauthorizedErr, 404) {
		t.Error("Expected IsGofastaErrorWithStatus to return false for non-matching status code")
	}
}

func TestUnauthorizedError_GetGofastaError(t *testing.T) {
	unauthorizedErr := NewUnauthorizedError("Test unauthorized")
	
	extractedErr := GetGofastaError(unauthorizedErr)
	if extractedErr == nil {
		t.Error("Expected GetGofastaError to extract GofastaError from UnauthorizedError")
	}

	if extractedErr.Code != "UNAUTHORIZED" {
		t.Errorf("Expected extracted error code 'UNAUTHORIZED', got %s", extractedErr.Code)
	}

	if extractedErr.StatusCode != 401 {
		t.Errorf("Expected extracted status code 401, got %d", extractedErr.StatusCode)
	}
}

func TestUnauthorizedError_ComplexScenario(t *testing.T) {
	// Simulate a complex OAuth2 authentication failure scenario
	tokenErr := errors.New("JWT signature verification failed: invalid key")
	
	err := NewUnauthorizedErrorf("OAuth2 authentication failed: %s", "token validation error").
		WithCause(tokenErr).
		WithAuthScheme("Bearer").
		WithRealm("OAuth2 API").
		WithChallenges(
			"Bearer realm=\"OAuth2 API\"",
			"Bearer error=\"invalid_token\"",
			"Bearer error_description=\"The access token is invalid or expired\"",
		).
		WithLoginUrl("https://auth.example.com/oauth/authorize?client_id=123&response_type=code").
		WithAuthContext("tokenType", "JWT").
		WithAuthContext("clientId", "client_123").
		WithAuthContext("scope", "read write admin").
		WithAuthContext("expiresIn", 3600).
		WithAuthContext("tokenExpired", true).
		WithMetadata("authProvider", "OAuth2").
		WithMetadata("authVersion", "2.0").
		WithMetadata("requestId", "req-12345").
		WithPath("/api/v1/oauth/userinfo").
		WithCurrentTimestamp()

	// Validate all properties are set correctly
	if err.Code != "UNAUTHORIZED" {
		t.Error("Error code not set correctly")
	}

	if err.StatusCode != 401 {
		t.Error("Status code not set correctly")
	}

	if err.Cause != tokenErr {
		t.Error("Cause not set correctly")
	}

	if err.AuthScheme != "Bearer" {
		t.Error("Auth scheme not set correctly")
	}

	if err.Realm != "OAuth2 API" {
		t.Error("Realm not set correctly")
	}

	if len(err.Challenges) != 3 {
		t.Error("Challenges not set correctly")
	}

	if err.LoginUrl != "https://auth.example.com/oauth/authorize?client_id=123&response_type=code" {
		t.Error("Login URL not set correctly")
	}

	if len(err.AuthContext) != 5 {
		t.Error("Auth context not set correctly")
	}

	if len(err.Metadata) != 3 {
		t.Error("Metadata not set correctly")
	}

	if err.Path != "/api/v1/oauth/userinfo" {
		t.Error("Path not set correctly")
	}

	if err.Timestamp == "" {
		t.Error("Timestamp not set")
	}

	// Test error message includes cause
	if !strings.Contains(err.Error(), "JWT signature verification failed") {
		t.Error("Error message should include cause details")
	}
}

// Tests for ForbiddenError

func TestForbiddenError_Basic(t *testing.T) {
	err := NewForbiddenError("Access denied to protected resource")

	if err == nil {
		t.Fatal("NewForbiddenError() returned nil")
	}

	if err.Code != "FORBIDDEN" {
		t.Errorf("Expected error code 'FORBIDDEN', got %s", err.Code)
	}

	if err.Message != "Access denied to protected resource" {
		t.Errorf("Expected error message 'Access denied to protected resource', got %s", err.Message)
	}

	if err.StatusCode != 403 {
		t.Errorf("Expected status code 403, got %d", err.StatusCode)
	}
}

func TestForbiddenError_DefaultMessage(t *testing.T) {
	err := NewForbiddenError("")

	expectedMessage := "Forbidden"
	if err.Message != expectedMessage {
		t.Errorf("Expected default message '%s', got %s", expectedMessage, err.Message)
	}
}

func TestForbiddenErrorf(t *testing.T) {
	err := NewForbiddenErrorf("Access denied to %s resource with ID %d", "user", 12345)

	expectedMessage := "Access denied to user resource with ID 12345"
	if err.Message != expectedMessage {
		t.Errorf("Expected formatted message '%s', got %s", expectedMessage, err.Message)
	}
}

func TestForbiddenErrorWithCause(t *testing.T) {
	cause := errors.New("permission check failed")
	err := NewForbiddenErrorWithCause("Authorization failed", cause)

	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}

	if err.Message != "Authorization failed" {
		t.Errorf("Expected message 'Authorization failed', got %s", err.Message)
	}
}

func TestForbiddenError_WithRequiredPermissions(t *testing.T) {
	err := NewForbiddenError("Insufficient permissions").
		WithRequiredPermissions("read", "write").
		WithRequiredPermissions("admin")

	expectedPermissions := []string{"read", "write", "admin"}
	if len(err.RequiredPermissions) != len(expectedPermissions) {
		t.Errorf("Expected %d required permissions, got %d", len(expectedPermissions), len(err.RequiredPermissions))
	}

	for i, permission := range expectedPermissions {
		if i >= len(err.RequiredPermissions) || err.RequiredPermissions[i] != permission {
			t.Errorf("Expected required permission %s at index %d", permission, i)
		}
	}
}

func TestForbiddenError_WithUserPermissions(t *testing.T) {
	err := NewForbiddenError("Permission mismatch").
		WithUserPermissions("read", "basic").
		WithUserPermissions("guest")

	expectedPermissions := []string{"read", "basic", "guest"}
	if len(err.UserPermissions) != len(expectedPermissions) {
		t.Errorf("Expected %d user permissions, got %d", len(expectedPermissions), len(err.UserPermissions))
	}

	for i, permission := range expectedPermissions {
		if i >= len(err.UserPermissions) || err.UserPermissions[i] != permission {
			t.Errorf("Expected user permission %s at index %d", permission, i)
		}
	}
}

func TestForbiddenError_WithResource(t *testing.T) {
	resource := "/api/admin/users"
	err := NewForbiddenError("Resource access denied").
		WithResource(resource)

	if err.Resource != resource {
		t.Errorf("Expected resource '%s', got %s", resource, err.Resource)
	}
}

func TestForbiddenError_WithAction(t *testing.T) {
	action := "DELETE"
	err := NewForbiddenError("Action not allowed").
		WithAction(action)

	if err.Action != action {
		t.Errorf("Expected action '%s', got %s", action, err.Action)
	}
}

func TestForbiddenError_WithAccessContext(t *testing.T) {
	err := NewForbiddenError("Access control failure").
		WithAccessContext("userId", 12345).
		WithAccessContext("role", "user").
		WithAccessContext("department", "engineering")

	if err.AccessContext["userId"] != 12345 {
		t.Error("Access context userId not set correctly")
	}

	if err.AccessContext["role"] != "user" {
		t.Error("Access context role not set correctly")
	}

	if err.AccessContext["department"] != "engineering" {
		t.Error("Access context department not set correctly")
	}
}

func TestForbiddenError_WithAllAccessContext(t *testing.T) {
	accessContext := map[string]interface{}{
		"organizationId": "org_123",
		"teamId":         "team_456",
		"accessLevel":    "standard",
		"ipAddress":      "192.168.1.100",
	}

	err := NewForbiddenError("Access control policy violation").
		WithAllAccessContext(accessContext)

	if err.AccessContext["organizationId"] != "org_123" {
		t.Error("Access context organizationId not set correctly")
	}

	if err.AccessContext["teamId"] != "team_456" {
		t.Error("Access context teamId not set correctly")
	}

	if err.AccessContext["accessLevel"] != "standard" {
		t.Error("Access context accessLevel not set correctly")
	}

	if err.AccessContext["ipAddress"] != "192.168.1.100" {
		t.Error("Access context ipAddress not set correctly")
	}
}

func TestForbiddenError_MethodChaining(t *testing.T) {
	cause := errors.New("RBAC policy evaluation failed")
	
	err := NewForbiddenError("Comprehensive access control error").
		WithCause(cause).
		WithRequiredPermissions("admin", "write", "delete").
		WithUserPermissions("read", "basic").
		WithResource("/api/admin/system/config").
		WithAction("DELETE").
		WithAccessContext("userId", 67890).
		WithAccessContext("role", "standard_user").
		WithAccessContext("organization", "acme_corp").
		WithMetadata("policyEngine", "RBAC").
		WithMetadata("policyVersion", "v2.1").
		WithPath("/api/admin/system/config").
		WithCurrentTimestamp()

	// Test cause
	if err.Cause != cause {
		t.Error("Cause not set correctly")
	}

	// Test required permissions
	if len(err.RequiredPermissions) != 3 || err.RequiredPermissions[0] != "admin" {
		t.Error("Required permissions not set correctly")
	}

	// Test user permissions
	if len(err.UserPermissions) != 2 || err.UserPermissions[0] != "read" {
		t.Error("User permissions not set correctly")
	}

	// Test resource
	if err.Resource != "/api/admin/system/config" {
		t.Error("Resource not set correctly")
	}

	// Test action
	if err.Action != "DELETE" {
		t.Error("Action not set correctly")
	}

	// Test access context
	if err.AccessContext["userId"] != 67890 || err.AccessContext["role"] != "standard_user" {
		t.Error("Access context not set correctly")
	}

	// Test metadata (from embedded GofastaError)
	if err.Metadata["policyEngine"] != "RBAC" {
		t.Error("Metadata not set correctly")
	}

	// Test path
	if err.Path != "/api/admin/system/config" {
		t.Error("Path not set correctly")
	}

	// Test timestamp
	if err.Timestamp == "" {
		t.Error("Timestamp not set")
	}
}

func TestForbiddenError_ErrorMessage(t *testing.T) {
	// Test error message without cause
	err := NewForbiddenError("Simple access denied")
	expectedMsg := "FORBIDDEN: Simple access denied"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test error message with cause
	cause := errors.New("ACL check failed")
	err = NewForbiddenErrorWithCause("Access control failure", cause)
	expectedMsgWithCause := "FORBIDDEN: Access control failure (caused by: ACL check failed)"
	if err.Error() != expectedMsgWithCause {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsgWithCause, err.Error())
	}
}

func TestForbiddenError_AsGofastaError(t *testing.T) {
	forbiddenErr := NewForbiddenError("Test forbidden")
	
	var gofastaErr *GofastaError
	if !AsGofastaError(forbiddenErr, &gofastaErr) {
		t.Error("Expected AsGofastaError to return true for ForbiddenError")
	}

	if gofastaErr == nil {
		t.Error("Expected GofastaError to be extracted from ForbiddenError")
	}

	if gofastaErr.Code != "FORBIDDEN" {
		t.Errorf("Expected extracted error code 'FORBIDDEN', got %s", gofastaErr.Code)
	}
}

func TestForbiddenError_IsGofastaError(t *testing.T) {
	forbiddenErr := NewForbiddenError("Test forbidden")

	// Test with IsGofastaError
	if !IsGofastaError(forbiddenErr) {
		t.Error("Expected IsGofastaError to return true for ForbiddenError")
	}

	// Test with specific code matching
	if !IsGofastaError(forbiddenErr, "FORBIDDEN") {
		t.Error("Expected IsGofastaError to return true for matching FORBIDDEN code")
	}

	// Test with wrong code
	if IsGofastaError(forbiddenErr, "NOT_FOUND") {
		t.Error("Expected IsGofastaError to return false for non-matching code")
	}

	// Test with status code matching
	if !IsGofastaErrorWithStatus(forbiddenErr, 403) {
		t.Error("Expected IsGofastaErrorWithStatus to return true for matching status code 403")
	}

	// Test with wrong status code
	if IsGofastaErrorWithStatus(forbiddenErr, 404) {
		t.Error("Expected IsGofastaErrorWithStatus to return false for non-matching status code")
	}
}

func TestForbiddenError_GetGofastaError(t *testing.T) {
	forbiddenErr := NewForbiddenError("Test forbidden")
	
	extractedErr := GetGofastaError(forbiddenErr)
	if extractedErr == nil {
		t.Error("Expected GetGofastaError to extract GofastaError from ForbiddenError")
	}

	if extractedErr.Code != "FORBIDDEN" {
		t.Errorf("Expected extracted error code 'FORBIDDEN', got %s", extractedErr.Code)
	}

	if extractedErr.StatusCode != 403 {
		t.Errorf("Expected extracted status code 403, got %d", extractedErr.StatusCode)
	}
}

func TestForbiddenError_ComplexScenario(t *testing.T) {
	// Simulate a complex RBAC (Role-Based Access Control) scenario
	rbacErr := errors.New("role hierarchy evaluation failed: insufficient role level")
	
	err := NewForbiddenErrorf("RBAC policy violation: %s access denied", "administrative").
		WithCause(rbacErr).
		WithRequiredPermissions("admin", "system:write", "users:delete", "config:modify").
		WithUserPermissions("user", "basic:read", "profile:write").
		WithResource("/api/v1/admin/system/users/bulk-delete").
		WithAction("DELETE").
		WithAccessContext("userId", "user_98765").
		WithAccessContext("userRole", "standard_user").
		WithAccessContext("requiredRole", "system_admin").
		WithAccessContext("organizationId", "org_acme").
		WithAccessContext("teamId", "team_engineering").
		WithAccessContext("accessAttemptTime", "2024-01-15T14:30:00Z").
		WithAccessContext("clientIp", "10.0.1.50").
		WithAccessContext("userAgent", "Mozilla/5.0 API Client").
		WithMetadata("rbacEngine", "Casbin").
		WithMetadata("policyVersion", "v3.2").
		WithMetadata("evaluationTime", "45ms").
		WithMetadata("securityLevel", "high").
		WithPath("/api/v1/admin/system/users/bulk-delete").
		WithCurrentTimestamp()

	// Validate all properties are set correctly
	if err.Code != "FORBIDDEN" {
		t.Error("Error code not set correctly")
	}

	if err.StatusCode != 403 {
		t.Error("Status code not set correctly")
	}

	if err.Cause != rbacErr {
		t.Error("Cause not set correctly")
	}

	if len(err.RequiredPermissions) != 4 {
		t.Error("Required permissions not set correctly")
	}

	if len(err.UserPermissions) != 3 {
		t.Error("User permissions not set correctly")
	}

	if err.Resource != "/api/v1/admin/system/users/bulk-delete" {
		t.Error("Resource not set correctly")
	}

	if err.Action != "DELETE" {
		t.Error("Action not set correctly")
	}

	if len(err.AccessContext) != 8 {
		t.Error("Access context not set correctly")
	}

	if len(err.Metadata) != 4 {
		t.Error("Metadata not set correctly")
	}

	if err.Path != "/api/v1/admin/system/users/bulk-delete" {
		t.Error("Path not set correctly")
	}

	if err.Timestamp == "" {
		t.Error("Timestamp not set")
	}

	// Test error message includes cause
	if !strings.Contains(err.Error(), "role hierarchy evaluation failed") {
		t.Error("Error message should include cause details")
	}

	// Test specific context values
	if err.AccessContext["userId"] != "user_98765" {
		t.Error("Access context userId not set correctly")
	}

	if err.AccessContext["userRole"] != "standard_user" {
		t.Error("Access context userRole not set correctly")
	}

	if err.AccessContext["requiredRole"] != "system_admin" {
		t.Error("Access context requiredRole not set correctly")
	}
}