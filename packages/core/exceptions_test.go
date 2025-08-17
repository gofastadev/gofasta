package core

import (
	"errors"
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
	
	if err.Error() != "This is a test error" {
		t.Errorf("Expected Error() to return 'This is a test error', got %s", err.Error())
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
	if gofastaErr, ok := err.(*GofastaError); ok {
		return gofastaErr.Code == "VALIDATION_ERROR"
	}
	return false
}

func IsNotFoundError(err error) bool {
	if gofastaErr, ok := err.(*GofastaError); ok {
		return gofastaErr.Code == "NOT_FOUND"
	}
	return false
}

func IsUnauthorizedError(err error) bool {
	if gofastaErr, ok := err.(*GofastaError); ok {
		return gofastaErr.Code == "UNAUTHORIZED"
	}
	return false
}