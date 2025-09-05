// Package core provides error handling with fast line number mapping - test file.
// This implements tests for Phase 1.3e: Create error handling with fast line number mapping.
package core

import (
	"go/token"
	"strings"
	"testing"
	"time"
)

// TestNewErrorHandler tests the creation of a new error handler
func TestNewErrorHandler(t *testing.T) {
	tests := []struct {
		name   string
		config *ErrorHandlerConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name:   "with default config",
			config: DefaultErrorHandlerConfig(),
		},
		{
			name: "with custom config",
			config: &ErrorHandlerConfig{
				MaxErrors:       50,
				EnableRecovery:  false,
				ShowErrorCode:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := NewErrorHandler(tt.config)
			if eh == nil {
				t.Fatal("expected non-nil error handler")
			}
			if eh.fileSets == nil {
				t.Fatal("expected fileSets to be initialized")
			}
			if eh.lineMaps == nil {
				t.Fatal("expected lineMaps to be initialized")
			}
			if eh.errorLog == nil {
				t.Fatal("expected errorLog to be initialized")
			}
		})
	}
}

// TestDefaultErrorHandlerConfig tests the default configuration
func TestDefaultErrorHandlerConfig(t *testing.T) {
	config := DefaultErrorHandlerConfig()
	
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.MaxErrors != 100 {
		t.Errorf("expected MaxErrors to be 100, got %d", config.MaxErrors)
	}
	if !config.CollectStackTrace {
		t.Error("expected CollectStackTrace to be true")
	}
	if !config.IncludeContext {
		t.Error("expected IncludeContext to be true")
	}
	if config.ContextLines != 3 {
		t.Errorf("expected ContextLines to be 3, got %d", config.ContextLines)
	}
	if !config.EnableRecovery {
		t.Error("expected EnableRecovery to be true")
	}
}

// TestCreateLineMap tests line map creation
func TestCreateLineMap(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	content := []byte("line 1\nline 2\nline 3\n")
	lm := eh.CreateLineMap("/test/file.go", content)
	
	if lm == nil {
		t.Fatal("expected non-nil line map")
	}
	if lm.filePath != "/test/file.go" {
		t.Errorf("filePath = %q, want %q", lm.filePath, "/test/file.go")
	}
	if len(lm.lineOffsets) != 4 { // 0, 7, 14, 21
		t.Errorf("expected 4 line offsets, got %d", len(lm.lineOffsets))
	}
	if lm.lineOffsets[0] != 0 {
		t.Errorf("first offset should be 0, got %d", lm.lineOffsets[0])
	}
	if lm.lineOffsets[1] != 7 {
		t.Errorf("second offset should be 7, got %d", lm.lineOffsets[1])
	}
}

// TestGetLineMap tests line map retrieval
func TestGetLineMap(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	content := []byte("test content")
	filePath := "/test/file.go"
	
	// Create line map
	originalLM := eh.CreateLineMap(filePath, content)
	
	// Retrieve line map
	retrievedLM := eh.GetLineMap(filePath)
	
	if retrievedLM != originalLM {
		t.Error("expected same line map instance")
	}
	if retrievedLM.accessCount != 1 {
		t.Errorf("expected access count to be 1, got %d", retrievedLM.accessCount)
	}
	
	// Test non-existent file
	nonExistentLM := eh.GetLineMap("/nonexistent/file.go")
	if nonExistentLM != nil {
		t.Error("expected nil for non-existent file")
	}
}

// TestLineMapGetPosition tests position calculation from offset
func TestLineMapGetPosition(t *testing.T) {
	content := []byte("line 1\nline 2\nline 3")
	lm := &LineMap{
		filePath:    "/test/file.go",
		content:     content,
		lineOffsets: []int{0, 7, 14}, // offsets for each line start
	}
	
	tests := []struct {
		name   string
		offset int
		line   int
		column int
	}{
		{"start of file", 0, 1, 1},
		{"start of line 1", 1, 1, 2},
		{"start of line 2", 7, 2, 1},
		{"middle of line 2", 10, 2, 4},
		{"start of line 3", 14, 3, 1},
		{"invalid negative", -1, 0, 0},
		{"invalid too large", 1000, 0, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, column := lm.GetPosition(tt.offset)
			if line != tt.line || column != tt.column {
				t.Errorf("GetPosition(%d) = (%d, %d), want (%d, %d)",
					tt.offset, line, column, tt.line, tt.column)
			}
		})
	}
}

// TestLineMapGetOffset tests offset calculation from line and column
func TestLineMapGetOffset(t *testing.T) {
	content := []byte("line 1\nline 2\nline 3")
	lm := &LineMap{
		filePath:    "/test/file.go",
		content:     content,
		lineOffsets: []int{0, 7, 14},
	}
	
	tests := []struct {
		name   string
		line   int
		column int
		offset int
	}{
		{"start of line 1", 1, 1, 0},
		{"char 2 of line 1", 1, 2, 1},
		{"start of line 2", 2, 1, 7},
		{"char 4 of line 2", 2, 4, 10},
		{"start of line 3", 3, 1, 14},
		{"invalid line", 0, 1, -1},
		{"invalid line too large", 5, 1, -1},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := lm.GetOffset(tt.line, tt.column)
			if offset != tt.offset {
				t.Errorf("GetOffset(%d, %d) = %d, want %d",
					tt.line, tt.column, offset, tt.offset)
			}
		})
	}
}

// TestLineMapGetLine tests getting line content
func TestLineMapGetLine(t *testing.T) {
	content := []byte("first line\nsecond line\nthird line")
	lm := &LineMap{
		filePath:    "/test/file.go",
		content:     content,
		lineOffsets: []int{0, 11, 23},
	}
	
	tests := []struct {
		name    string
		lineNum int
		want    string
	}{
		{"line 1", 1, "first line"},
		{"line 2", 2, "second line"},
		{"line 3", 3, "third line"},
		{"invalid line 0", 0, ""},
		{"invalid line too large", 5, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := lm.GetLine(tt.lineNum)
			if line != tt.want {
				t.Errorf("GetLine(%d) = %q, want %q", tt.lineNum, line, tt.want)
			}
		})
	}
}

// TestLineMapGetContext tests getting context lines
func TestLineMapGetContext(t *testing.T) {
	content := []byte("line 1\nline 2\nline 3\nline 4\nline 5")
	lm := &LineMap{
		filePath:    "/test/file.go",
		content:     content,
		lineOffsets: []int{0, 7, 14, 21, 28},
	}
	
	// Test getting context around line 3 with 1 line context
	context := lm.GetContext(3, 1)
	expected := []string{"line 2", "line 3", "line 4"}
	
	if len(context) != len(expected) {
		t.Errorf("expected %d context lines, got %d", len(expected), len(context))
	}
	
	for i, line := range context {
		if line != expected[i] {
			t.Errorf("context line %d = %q, want %q", i, line, expected[i])
		}
	}
	
	// Test edge cases
	contextStart := lm.GetContext(1, 2) // Should not go below line 1
	if len(contextStart) != 3 {
		t.Errorf("expected 3 lines for context at start, got %d", len(contextStart))
	}
	
	contextEnd := lm.GetContext(5, 2) // Should not go beyond available lines
	if len(contextEnd) != 3 {
		t.Errorf("expected 3 lines for context at end, got %d", len(contextEnd))
	}
}

// TestReportError tests error reporting
func TestReportError(t *testing.T) {
	eh := NewErrorHandler(&ErrorHandlerConfig{
		CollectStackTrace: false,
		IncludeContext:    false,
	})
	
	location := &ErrorLocation{
		File:   "/test/file.go",
		Line:   10,
		Column: 5,
	}
	
	// Test reporting known error code
	err := eh.ReportError("E001", "", location)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Code != "E001" {
		t.Errorf("Code = %q, want E001", err.Code)
	}
	if err.Message != "Syntax error" {
		t.Errorf("Message = %q, want 'Syntax error'", err.Message)
	}
	if err.Severity != SeverityError {
		t.Errorf("Severity = %v, want SeverityError", err.Severity)
	}
	if err.Category != CategorySyntax {
		t.Errorf("Category = %v, want CategorySyntax", err.Category)
	}
	
	// Test reporting custom error
	customErr := eh.ReportError("CUSTOM", "Custom error message", location)
	if customErr.Code != "CUSTOM" {
		t.Errorf("Custom error code = %q, want CUSTOM", customErr.Code)
	}
	if customErr.Message != "Custom error message" {
		t.Errorf("Custom error message = %q, want 'Custom error message'", customErr.Message)
	}
}

// TestReportWarning tests warning reporting
func TestReportWarning(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	location := &ErrorLocation{
		File:   "/test/file.go",
		Line:   5,
		Column: 10,
	}
	
	warning := eh.ReportWarning("W001", "Test warning", location)
	if warning.Severity != SeverityWarning {
		t.Errorf("expected SeverityWarning, got %v", warning.Severity)
	}
	if warning.Code != "W001" {
		t.Errorf("Code = %q, want W001", warning.Code)
	}
}

// TestReportInfo tests info reporting
func TestReportInfo(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	location := &ErrorLocation{
		File:   "/test/file.go",
		Line:   1,
		Column: 1,
	}
	
	info := eh.ReportInfo("Test info message", location)
	if info.Severity != SeverityInfo {
		t.Errorf("expected SeverityInfo, got %v", info.Severity)
	}
	if info.Code != "I000" {
		t.Errorf("Code = %q, want I000", info.Code)
	}
	if info.Message != "Test info message" {
		t.Errorf("Message = %q, want 'Test info message'", info.Message)
	}
}

// TestReportFatal tests fatal error reporting
func TestReportFatal(t *testing.T) {
	eh := NewErrorHandler(&ErrorHandlerConfig{
		PanicOnFatal:      false, // Don't panic in tests
		CollectStackTrace: false,
	})
	
	location := &ErrorLocation{
		File:   "/test/file.go",
		Line:   1,
		Column: 1,
	}
	
	// Should not panic with PanicOnFatal = false
	eh.ReportFatal("Test fatal error", location)
	
	stats := eh.GetStatistics()
	fatalErrors := stats["fatal_errors"].(int64)
	if fatalErrors != 1 {
		t.Errorf("expected 1 fatal error, got %d", fatalErrors)
	}
}

// TestGetErrors tests error retrieval
func TestGetErrors(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	location := &ErrorLocation{File: "/test/file.go", Line: 1, Column: 1}
	
	// Report different types of messages
	eh.ReportError("E001", "Error 1", location)
	eh.ReportError("E002", "Error 2", location)
	eh.ReportWarning("W001", "Warning 1", location)
	eh.ReportInfo("Info message", location)
	
	errors := eh.GetErrors()
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}
	
	warnings := eh.GetWarnings()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

// TestGetErrorsByFile tests error retrieval by file
func TestGetErrorsByFile(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	file1 := "/test/file1.go"
	file2 := "/test/file2.go"
	
	// Report errors for different files
	eh.ReportError("E001", "Error in file1", &ErrorLocation{File: file1, Line: 1, Column: 1})
	eh.ReportError("E002", "Error in file1", &ErrorLocation{File: file1, Line: 2, Column: 1})
	eh.ReportError("E001", "Error in file2", &ErrorLocation{File: file2, Line: 1, Column: 1})
	
	file1Errors := eh.GetErrorsByFile(file1)
	if len(file1Errors) != 2 {
		t.Errorf("expected 2 errors for file1, got %d", len(file1Errors))
	}
	
	file2Errors := eh.GetErrorsByFile(file2)
	if len(file2Errors) != 1 {
		t.Errorf("expected 1 error for file2, got %d", len(file2Errors))
	}
	
	nonExistentErrors := eh.GetErrorsByFile("/nonexistent.go")
	if len(nonExistentErrors) != 0 {
		t.Errorf("expected 0 errors for nonexistent file, got %d", len(nonExistentErrors))
	}
}

// TestHasErrors tests error checking functions
func TestHasErrors(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	// Initially no errors
	if eh.HasErrors() {
		t.Error("expected no errors initially")
	}
	if eh.HasFatalErrors() {
		t.Error("expected no fatal errors initially")
	}
	
	// Report an error
	eh.ReportError("E001", "Test error", nil)
	if !eh.HasErrors() {
		t.Error("expected to have errors after reporting")
	}
	
	// Report a fatal error
	eh.ReportFatal("Fatal error", nil)
	if !eh.HasFatalErrors() {
		t.Error("expected to have fatal errors after reporting")
	}
}

// TestClearErrors tests clearing errors
func TestClearErrors(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	// Report some errors
	eh.ReportError("E001", "Error 1", nil)
	eh.ReportWarning("W001", "Warning 1", nil)
	eh.ReportFatal("Fatal error", nil)
	
	// Verify errors exist
	if !eh.HasErrors() {
		t.Fatal("expected to have errors before clearing")
	}
	
	// Clear errors
	eh.Clear()
	
	// Verify errors are cleared
	if eh.HasErrors() {
		t.Error("expected no errors after clearing")
	}
	if eh.HasFatalErrors() {
		t.Error("expected no fatal errors after clearing")
	}
	
	stats := eh.GetStatistics()
	if stats["total_errors"].(int64) != 0 {
		t.Error("expected total_errors to be 0 after clearing")
	}
}

// TestErrorSeverityString tests severity string representation
func TestErrorSeverityString(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		want     string
	}{
		{SeverityHint, "hint"},
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{SeverityFatal, "fatal"},
		{ErrorSeverity(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTranspilerErrorError tests error interface implementation
func TestTranspilerErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *TranspilerError
		expected string
	}{
		{
			name: "with location",
			err: &TranspilerError{
				Message: "test error",
				Location: &ErrorLocation{
					File:   "/test/file.go",
					Line:   10,
					Column: 5,
				},
			},
			expected: "/test/file.go:10:5: test error",
		},
		{
			name: "without location",
			err: &TranspilerError{
				Message: "test error without location",
			},
			expected: "test error without location",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFormatSeverity tests severity formatting
func TestFormatSeverity(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	tests := []struct {
		severity ErrorSeverity
		want     string
	}{
		{SeverityHint, "HINT"},
		{SeverityInfo, "INFO"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityFatal, "FATAL"},
		{ErrorSeverity(999), "UNKNOWN"},
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := eh.formatSeverity(tt.severity)
			if got != tt.want {
				t.Errorf("formatSeverity() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestColorize tests error colorization
func TestColorize(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	tests := []struct {
		name     string
		text     string
		severity ErrorSeverity
	}{
		{"hint", "HINT", SeverityHint},
		{"info", "INFO", SeverityInfo},
		{"warning", "WARNING", SeverityWarning},
		{"error", "ERROR", SeverityError},
		{"fatal", "FATAL", SeverityFatal},
		{"unknown", "UNKNOWN", ErrorSeverity(999)},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colored := eh.colorize(tt.text, tt.severity)
			
			// For unknown severity, text should be unchanged
			if tt.severity == ErrorSeverity(999) {
				if colored != tt.text {
					t.Errorf("colorize() should return unchanged text for unknown severity")
				}
			} else {
				// For known severities, text should be wrapped with ANSI codes
				if !strings.Contains(colored, tt.text) {
					t.Errorf("colorized text should contain original text")
				}
				if len(colored) <= len(tt.text) {
					t.Errorf("colorized text should be longer than original")
				}
			}
		})
	}
}

// TestFormatError tests error formatting
func TestFormatError(t *testing.T) {
	eh := NewErrorHandler(&ErrorHandlerConfig{
		ColorOutput:     false,
		ShowErrorCode:   true,
		ShowSuggestions: true,
	})
	
	err := &TranspilerError{
		Code:     "E001",
		Message:  "Test error",
		Severity: SeverityError,
		Location: &ErrorLocation{
			File:   "/test/file.go",
			Line:   10,
			Column: 5,
		},
		Suggestions: []string{"Fix the syntax", "Check imports"},
	}
	
	formatted := eh.FormatError(err)
	
	if !strings.Contains(formatted, "ERROR") {
		t.Error("formatted error should contain severity")
	}
	if !strings.Contains(formatted, "/test/file.go:10:5") {
		t.Error("formatted error should contain location")
	}
	if !strings.Contains(formatted, "[E001]") {
		t.Error("formatted error should contain error code")
	}
	if !strings.Contains(formatted, "Test error") {
		t.Error("formatted error should contain message")
	}
	if !strings.Contains(formatted, "Suggestions:") {
		t.Error("formatted error should contain suggestions")
	}
	if !strings.Contains(formatted, "Fix the syntax") {
		t.Error("formatted error should contain suggestion text")
	}
}

// TestParseErrorLocation tests error location parsing
func TestParseErrorLocation(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	tests := []struct {
		name        string
		errMsg      string
		defaultFile string
		expected    *ErrorLocation
	}{
		{
			name:        "standard format",
			errMsg:      "/test/file.go:10:5: syntax error",
			defaultFile: "/default.go",
			expected: &ErrorLocation{
				File:   "/test/file.go",
				Line:   10,
				Column: 5,
			},
		},
		{
			name:        "minimal format",
			errMsg:      "file.go:1:1: error",
			defaultFile: "/default.go",
			expected: &ErrorLocation{
				File:   "file.go",
				Line:   1,
				Column: 1,
			},
		},
		{
			name:        "no location info",
			errMsg:      "generic error message",
			defaultFile: "/default.go",
			expected: &ErrorLocation{
				File: "/default.go",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location := eh.parseErrorLocation(tt.errMsg, tt.defaultFile)
			if location.File != tt.expected.File {
				t.Errorf("File = %q, want %q", location.File, tt.expected.File)
			}
			if location.Line != tt.expected.Line {
				t.Errorf("Line = %d, want %d", location.Line, tt.expected.Line)
			}
			if location.Column != tt.expected.Column {
				t.Errorf("Column = %d, want %d", location.Column, tt.expected.Column)
			}
		})
	}
}

// TestReportSyntaxError tests syntax error reporting
func TestReportSyntaxError(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	// Create a simple error for testing
	testErr := &CustomError{msg: "syntax error: unexpected token"}
	
	err := eh.ReportSyntaxError(testErr, "/test/file.go")
	if err.Code != "E001" {
		t.Errorf("expected error code E001, got %q", err.Code)
	}
	if err.Category != CategorySyntax {
		t.Errorf("expected CategorySyntax, got %v", err.Category)
	}
}

// CustomError for testing
type CustomError struct {
	msg string
}

func (e *CustomError) Error() string {
	return e.msg
}

// TestReportTypeError tests type error reporting
func TestReportTypeError(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	// Create a simple token fileset
	fset := token.NewFileSet()
	
	// Test with nil node to test the nil handling
	err := eh.ReportTypeError("Type mismatch error", nil, fset)
	if err.Code != "E002" {
		t.Errorf("expected error code E002, got %q", err.Code)
	}
	if err.Category != CategoryType {
		t.Errorf("expected CategoryType, got %v", err.Category)
	}
	if err.Message != "Type mismatch error" {
		t.Errorf("Message = %q, want 'Type mismatch error'", err.Message)
	}
}

// TestErrorHandlerGetStatistics tests statistics collection
func TestErrorHandlerGetStatistics(t *testing.T) {
	eh := NewErrorHandler(nil)
	
	// Report various types of errors
	eh.ReportError("E001", "Error 1", nil)
	eh.ReportError("E002", "Error 2", nil)
	eh.ReportWarning("W001", "Warning 1", nil)
	eh.ReportFatal("Fatal error", nil)
	
	stats := eh.GetStatistics()
	
	totalErrors := stats["total_errors"].(int64)
	if totalErrors != 3 { // 2 errors + 1 fatal
		t.Errorf("expected 3 total errors, got %d", totalErrors)
	}
	
	totalWarnings := stats["total_warnings"].(int64)
	if totalWarnings != 1 {
		t.Errorf("expected 1 warning, got %d", totalWarnings)
	}
	
	fatalErrors := stats["fatal_errors"].(int64)
	if fatalErrors != 1 {
		t.Errorf("expected 1 fatal error, got %d", fatalErrors)
	}
}

// TestRecover tests panic recovery
func TestRecover(t *testing.T) {
	eh := NewErrorHandler(&ErrorHandlerConfig{
		EnableRecovery: true,
	})
	
	// Test recovery disabled
	ehDisabled := NewErrorHandler(&ErrorHandlerConfig{
		EnableRecovery: false,
	})
	
	// This should not panic
	func() {
		defer ehDisabled.Recover()
		// No panic here, so recovery should do nothing
	}()
	
	stats := ehDisabled.GetStatistics()
	recoveries := stats["recoveries"].(int64)
	if recoveries != 0 {
		t.Errorf("expected 0 recoveries when disabled, got %d", recoveries)
	}
	
	// Test with recovery enabled but no panic
	func() {
		defer eh.Recover()
		// No panic here either
	}()
	
	stats = eh.GetStatistics()
	recoveries = stats["recoveries"].(int64)
	if recoveries != 0 {
		t.Errorf("expected 0 recoveries when no panic, got %d", recoveries)
	}
}

// TestErrorCategories tests error category constants
func TestErrorCategories(t *testing.T) {
	categories := []ErrorCategory{
		CategorySyntax,
		CategoryType,
		CategoryDecorator,
		CategoryImport,
		CategoryCompilation,
		CategoryRuntime,
		CategoryInternal,
	}
	
	expected := []string{
		"syntax",
		"type",
		"decorator",
		"import",
		"compilation",
		"runtime",
		"internal",
	}
	
	for i, category := range categories {
		if string(category) != expected[i] {
			t.Errorf("category %d = %q, want %q", i, string(category), expected[i])
		}
	}
}

// TestErrorCodes tests predefined error codes
func TestErrorCodes(t *testing.T) {
	expectedCodes := []string{"E001", "E002", "E003", "E004", "W001", "W002", "I001"}
	
	for _, code := range expectedCodes {
		errorCode, exists := errorCodes[code]
		if !exists {
			t.Errorf("expected error code %s to exist", code)
			continue
		}
		if errorCode.Code != code {
			t.Errorf("error code mismatch: got %q, want %q", errorCode.Code, code)
		}
		if errorCode.Message == "" {
			t.Errorf("error code %s should have a message", code)
		}
		if errorCode.Category == "" {
			t.Errorf("error code %s should have a category", code)
		}
	}
	
	// Test specific error codes
	e001 := errorCodes["E001"]
	if e001.Category != CategorySyntax {
		t.Errorf("E001 category = %v, want CategorySyntax", e001.Category)
	}
	if e001.Severity != SeverityError {
		t.Errorf("E001 severity = %v, want SeverityError", e001.Severity)
	}
	
	w001 := errorCodes["W001"]
	if w001.Severity != SeverityWarning {
		t.Errorf("W001 severity = %v, want SeverityWarning", w001.Severity)
	}
}

// TestErrorEntry tests error entry structure
func TestErrorEntry(t *testing.T) {
	err := &TranspilerError{
		Code:     "TEST",
		Message:  "Test error",
		Severity: SeverityError,
	}
	
	entry := ErrorEntry{
		Error:     err,
		Timestamp: time.Now(),
		Handled:   false,
	}
	
	if entry.Error != err {
		t.Error("expected error to be set correctly")
	}
	if entry.Handled {
		t.Error("expected entry to be unhandled initially")
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

// TestStackFrame tests stack frame structure
func TestStackFrame(t *testing.T) {
	frame := StackFrame{
		Function: "main.testFunction",
		File:     "main.go",
		Line:     42,
	}
	
	if frame.Function != "main.testFunction" {
		t.Errorf("Function = %q, want 'main.testFunction'", frame.Function)
	}
	if frame.File != "main.go" {
		t.Errorf("File = %q, want 'main.go'", frame.File)
	}
	if frame.Line != 42 {
		t.Errorf("Line = %d, want 42", frame.Line)
	}
}

// TestErrorLocation tests error location structure
func TestErrorLocation(t *testing.T) {
	location := ErrorLocation{
		File:      "/test/file.go",
		Line:      10,
		Column:    5,
		EndLine:   10,
		EndColumn: 15,
		Offset:    100,
		Length:    10,
	}
	
	if location.File != "/test/file.go" {
		t.Errorf("File = %q, want '/test/file.go'", location.File)
	}
	if location.Line != 10 {
		t.Errorf("Line = %d, want 10", location.Line)
	}
	if location.Column != 5 {
		t.Errorf("Column = %d, want 5", location.Column)
	}
	if location.Length != 10 {
		t.Errorf("Length = %d, want 10", location.Length)
	}
}