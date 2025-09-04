// Package core provides error handling with fast line number mapping.
// This implements Phase 1.3e: Create error handling with fast line number mapping.
package core

import (
	"fmt"
	"go/ast"
	"go/scanner"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorHandler provides error handling with fast line number mapping
type ErrorHandler struct {
	config       *ErrorHandlerConfig
	fileSets     map[string]*token.FileSet
	lineMaps     map[string]*LineMap
	errorLog     []ErrorEntry
	errorGroups  map[string][]*TranspilerError
	mu           sync.RWMutex

	// Metrics
	totalErrors   int64
	totalWarnings int64
	fatalErrors   int64
	recoveries    int64
}

// ErrorHandlerConfig contains configuration for error handler
type ErrorHandlerConfig struct {
	// Error handling settings
	MaxErrors          int
	MaxErrorsPerFile   int
	CollectStackTrace  bool
	IncludeContext     bool
	ContextLines       int
	
	// Display settings
	ColorOutput        bool
	ShowErrorCode      bool
	ShowSuggestions    bool
	GroupSimilarErrors bool
	
	// Recovery settings
	EnableRecovery     bool
	PanicOnFatal       bool
	LogToFile          bool
	LogFilePath        string
}

// TranspilerError represents a transpiler error with location information
type TranspilerError struct {
	Code        string
	Message     string
	Severity    ErrorSeverity
	Location    *ErrorLocation
	Context     []string
	Suggestions []string
	StackTrace  []StackFrame
	Timestamp   time.Time
	Category    ErrorCategory
}

// ErrorLocation represents the location of an error
type ErrorLocation struct {
	File       string
	Line       int
	Column     int
	EndLine    int
	EndColumn  int
	Offset     int
	Length     int
}

// ErrorSeverity represents the severity of an error
type ErrorSeverity int

const (
	SeverityHint ErrorSeverity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
	SeverityFatal
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategorySyntax      ErrorCategory = "syntax"
	CategoryType        ErrorCategory = "type"
	CategoryDecorator   ErrorCategory = "decorator"
	CategoryImport      ErrorCategory = "import"
	CategoryCompilation ErrorCategory = "compilation"
	CategoryRuntime     ErrorCategory = "runtime"
	CategoryInternal    ErrorCategory = "internal"
)

// ErrorEntry represents an entry in the error log
type ErrorEntry struct {
	Error     *TranspilerError
	Timestamp time.Time
	Handled   bool
}

// StackFrame represents a stack frame
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// LineMap provides fast line number mapping
type LineMap struct {
	filePath    string
	lineOffsets []int
	content     []byte
	lastAccess  time.Time
	accessCount int64
}

// ErrorCode represents a predefined error code
type ErrorCode struct {
	Code        string
	Message     string
	Category    ErrorCategory
	Severity    ErrorSeverity
	Suggestions []string
}

// Predefined error codes
var errorCodes = map[string]ErrorCode{
	"E001": {
		Code:     "E001",
		Message:  "Syntax error",
		Category: CategorySyntax,
		Severity: SeverityError,
	},
	"E002": {
		Code:     "E002",
		Message:  "Type mismatch",
		Category: CategoryType,
		Severity: SeverityError,
	},
	"E003": {
		Code:        "E003",
		Message:     "Unknown decorator",
		Category:    CategoryDecorator,
		Severity:    SeverityError,
		Suggestions: []string{"Check decorator name", "Ensure decorator is imported"},
	},
	"E004": {
		Code:     "E004",
		Message:  "Import not found",
		Category: CategoryImport,
		Severity: SeverityError,
	},
	"W001": {
		Code:     "W001",
		Message:  "Unused variable",
		Category: CategoryCompilation,
		Severity: SeverityWarning,
	},
	"W002": {
		Code:        "W002",
		Message:     "Deprecated decorator",
		Category:    CategoryDecorator,
		Severity:    SeverityWarning,
		Suggestions: []string{"Use alternative decorator"},
	},
	"I001": {
		Code:     "I001",
		Message:  "Consider using more specific type",
		Category: CategoryType,
		Severity: SeverityInfo,
	},
}

// DefaultErrorHandlerConfig returns the default configuration
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		MaxErrors:          100,
		MaxErrorsPerFile:   10,
		CollectStackTrace:  true,
		IncludeContext:     true,
		ContextLines:       3,
		ColorOutput:        true,
		ShowErrorCode:      true,
		ShowSuggestions:    true,
		GroupSimilarErrors: true,
		EnableRecovery:     true,
		PanicOnFatal:       false,
		LogToFile:          false,
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *ErrorHandlerConfig) *ErrorHandler {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}

	return &ErrorHandler{
		config:       config,
		fileSets:     make(map[string]*token.FileSet),
		lineMaps:     make(map[string]*LineMap),
		errorLog:     make([]ErrorEntry, 0),
		errorGroups:  make(map[string][]*TranspilerError),
	}
}

// CreateLineMap creates a line map for fast line number lookup
func (eh *ErrorHandler) CreateLineMap(filePath string, content []byte) *LineMap {
	lm := &LineMap{
		filePath:    filePath,
		content:     content,
		lineOffsets: []int{0},
		lastAccess:  time.Now(),
	}

	// Build line offset map
	for i, b := range content {
		if b == '\n' {
			lm.lineOffsets = append(lm.lineOffsets, i+1)
		}
	}

	eh.mu.Lock()
	eh.lineMaps[filePath] = lm
	eh.mu.Unlock()

	return lm
}

// GetLineMap retrieves a line map
func (eh *ErrorHandler) GetLineMap(filePath string) *LineMap {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	lm := eh.lineMaps[filePath]
	if lm != nil {
		atomic.AddInt64(&lm.accessCount, 1)
		lm.lastAccess = time.Now()
	}
	return lm
}

// GetPosition gets line and column from offset
func (lm *LineMap) GetPosition(offset int) (line, column int) {
	if offset < 0 || offset > len(lm.content) {
		return 0, 0
	}

	// Binary search for line
	line = sort.Search(len(lm.lineOffsets), func(i int) bool {
		return lm.lineOffsets[i] > offset
	})

	if line > 0 {
		column = offset - lm.lineOffsets[line-1] + 1
	} else {
		column = offset + 1
		line = 1
	}

	return line, column
}

// GetOffset gets offset from line and column
func (lm *LineMap) GetOffset(line, column int) int {
	if line <= 0 || line > len(lm.lineOffsets) {
		return -1
	}

	lineStart := lm.lineOffsets[line-1]
	return lineStart + column - 1
}

// GetLine gets the content of a specific line
func (lm *LineMap) GetLine(lineNum int) string {
	if lineNum <= 0 || lineNum > len(lm.lineOffsets) {
		return ""
	}

	start := lm.lineOffsets[lineNum-1]
	end := len(lm.content)
	if lineNum < len(lm.lineOffsets) {
		end = lm.lineOffsets[lineNum] - 1
	}

	return string(lm.content[start:end])
}

// GetContext gets surrounding context lines
func (lm *LineMap) GetContext(line, contextLines int) []string {
	var lines []string

	start := line - contextLines
	if start < 1 {
		start = 1
	}

	end := line + contextLines
	if end > len(lm.lineOffsets) {
		end = len(lm.lineOffsets)
	}

	for i := start; i <= end; i++ {
		lines = append(lines, lm.GetLine(i))
	}

	return lines
}

// ReportError reports an error
func (eh *ErrorHandler) ReportError(code string, message string, location *ErrorLocation) *TranspilerError {
	errorCode, exists := errorCodes[code]
	if !exists {
		errorCode = ErrorCode{
			Code:     code,
			Message:  message,
			Category: CategoryInternal,
			Severity: SeverityError,
		}
	}

	if message == "" {
		message = errorCode.Message
	}

	err := &TranspilerError{
		Code:        code,
		Message:     message,
		Severity:    errorCode.Severity,
		Location:    location,
		Category:    errorCode.Category,
		Suggestions: errorCode.Suggestions,
		Timestamp:   time.Now(),
	}

	// Add context if configured
	if eh.config.IncludeContext && location != nil {
		if lm := eh.GetLineMap(location.File); lm != nil {
			err.Context = lm.GetContext(location.Line, eh.config.ContextLines)
		}
	}

	// Add stack trace if configured
	if eh.config.CollectStackTrace {
		err.StackTrace = eh.captureStackTrace()
	}

	// Update metrics
	atomic.AddInt64(&eh.totalErrors, 1)
	if errorCode.Severity == SeverityWarning {
		atomic.AddInt64(&eh.totalWarnings, 1)
	} else if errorCode.Severity == SeverityFatal {
		atomic.AddInt64(&eh.fatalErrors, 1)
	}

	// Add to log
	eh.mu.Lock()
	eh.errorLog = append(eh.errorLog, ErrorEntry{
		Error:     err,
		Timestamp: time.Now(),
		Handled:   false,
	})

	// Group similar errors if configured
	if eh.config.GroupSimilarErrors {
		groupKey := fmt.Sprintf("%s:%s", err.Category, err.Code)
		eh.errorGroups[groupKey] = append(eh.errorGroups[groupKey], err)
	}
	eh.mu.Unlock()

	// Check limits
	if eh.config.MaxErrors > 0 && int(atomic.LoadInt64(&eh.totalErrors)) > eh.config.MaxErrors {
		eh.ReportFatal("Too many errors", nil)
	}

	return err
}

// ReportSyntaxError reports a syntax error from Go parser
func (eh *ErrorHandler) ReportSyntaxError(err error, file string) *TranspilerError {
	var location *ErrorLocation

	// Parse scanner error
	if scanErr, ok := err.(scanner.Error); ok {
		location = &ErrorLocation{
			File:   file,
			Line:   scanErr.Pos.Line,
			Column: scanErr.Pos.Column,
		}
	} else {
		// Try to extract location from error message
		location = eh.parseErrorLocation(err.Error(), file)
	}

	return eh.ReportError("E001", err.Error(), location)
}

// ReportTypeError reports a type error
func (eh *ErrorHandler) ReportTypeError(message string, node ast.Node, fset *token.FileSet) *TranspilerError {
	var location *ErrorLocation

	if node != nil && fset != nil {
		pos := fset.Position(node.Pos())
		endPos := fset.Position(node.End())
		location = &ErrorLocation{
			File:      pos.Filename,
			Line:      pos.Line,
			Column:    pos.Column,
			EndLine:   endPos.Line,
			EndColumn: endPos.Column,
			Offset:    pos.Offset,
		}
	}

	return eh.ReportError("E002", message, location)
}

// ReportWarning reports a warning
func (eh *ErrorHandler) ReportWarning(code string, message string, location *ErrorLocation) *TranspilerError {
	err := eh.ReportError(code, message, location)
	err.Severity = SeverityWarning
	return err
}

// ReportInfo reports an informational message
func (eh *ErrorHandler) ReportInfo(message string, location *ErrorLocation) *TranspilerError {
	err := &TranspilerError{
		Code:      "I000",
		Message:   message,
		Severity:  SeverityInfo,
		Location:  location,
		Category:  CategoryInternal,
		Timestamp: time.Now(),
	}

	eh.mu.Lock()
	eh.errorLog = append(eh.errorLog, ErrorEntry{
		Error:     err,
		Timestamp: time.Now(),
		Handled:   false,
	})
	eh.mu.Unlock()

	return err
}

// ReportFatal reports a fatal error
func (eh *ErrorHandler) ReportFatal(message string, location *ErrorLocation) {
	err := &TranspilerError{
		Code:      "F001",
		Message:   message,
		Severity:  SeverityFatal,
		Location:  location,
		Category:  CategoryInternal,
		Timestamp: time.Now(),
	}

	if eh.config.CollectStackTrace {
		err.StackTrace = eh.captureStackTrace()
	}

	atomic.AddInt64(&eh.fatalErrors, 1)

	eh.mu.Lock()
	eh.errorLog = append(eh.errorLog, ErrorEntry{
		Error:     err,
		Timestamp: time.Now(),
		Handled:   false,
	})
	eh.mu.Unlock()

	if eh.config.PanicOnFatal {
		panic(fmt.Sprintf("Fatal error: %s", message))
	}
}

// Recover recovers from a panic
func (eh *ErrorHandler) Recover() {
	if !eh.config.EnableRecovery {
		return
	}

	if r := recover(); r != nil {
		atomic.AddInt64(&eh.recoveries, 1)

		message := fmt.Sprintf("Panic recovered: %v", r)
		err := &TranspilerError{
			Code:       "P001",
			Message:    message,
			Severity:   SeverityFatal,
			Category:   CategoryRuntime,
			Timestamp:  time.Now(),
			StackTrace: eh.captureStackTrace(),
		}

		eh.mu.Lock()
		eh.errorLog = append(eh.errorLog, ErrorEntry{
			Error:     err,
			Timestamp: time.Now(),
			Handled:   true,
		})
		eh.mu.Unlock()
	}
}

// GetErrors returns all errors
func (eh *ErrorHandler) GetErrors() []*TranspilerError {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	var errors []*TranspilerError
	for _, entry := range eh.errorLog {
		if entry.Error.Severity >= SeverityError {
			errors = append(errors, entry.Error)
		}
	}
	return errors
}

// GetWarnings returns all warnings
func (eh *ErrorHandler) GetWarnings() []*TranspilerError {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	var warnings []*TranspilerError
	for _, entry := range eh.errorLog {
		if entry.Error.Severity == SeverityWarning {
			warnings = append(warnings, entry.Error)
		}
	}
	return warnings
}

// GetErrorsByFile returns errors for a specific file
func (eh *ErrorHandler) GetErrorsByFile(file string) []*TranspilerError {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	var errors []*TranspilerError
	for _, entry := range eh.errorLog {
		if entry.Error.Location != nil && entry.Error.Location.File == file {
			errors = append(errors, entry.Error)
		}
	}
	return errors
}

// HasErrors returns true if there are any errors
func (eh *ErrorHandler) HasErrors() bool {
	return atomic.LoadInt64(&eh.totalErrors) > 0
}

// HasFatalErrors returns true if there are any fatal errors
func (eh *ErrorHandler) HasFatalErrors() bool {
	return atomic.LoadInt64(&eh.fatalErrors) > 0
}

// Clear clears all errors
func (eh *ErrorHandler) Clear() {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.errorLog = make([]ErrorEntry, 0)
	eh.errorGroups = make(map[string][]*TranspilerError)
	atomic.StoreInt64(&eh.totalErrors, 0)
	atomic.StoreInt64(&eh.totalWarnings, 0)
	atomic.StoreInt64(&eh.fatalErrors, 0)
}

// FormatError formats an error for display
func (eh *ErrorHandler) FormatError(err *TranspilerError) string {
	var builder strings.Builder

	// Add severity
	severityStr := eh.formatSeverity(err.Severity)
	if eh.config.ColorOutput {
		severityStr = eh.colorize(severityStr, err.Severity)
	}
	builder.WriteString(severityStr)
	builder.WriteString(": ")

	// Add location
	if err.Location != nil {
		locationStr := fmt.Sprintf("%s:%d:%d",
			err.Location.File,
			err.Location.Line,
			err.Location.Column)
		builder.WriteString(locationStr)
		builder.WriteString(": ")
	}

	// Add error code if configured
	if eh.config.ShowErrorCode && err.Code != "" {
		builder.WriteString("[")
		builder.WriteString(err.Code)
		builder.WriteString("] ")
	}

	// Add message
	builder.WriteString(err.Message)
	builder.WriteString("\n")

	// Add context
	if len(err.Context) > 0 {
		for i, line := range err.Context {
			if err.Location != nil {
				lineNum := err.Location.Line - eh.config.ContextLines + i
				if lineNum == err.Location.Line {
					builder.WriteString(fmt.Sprintf("%4d | ", lineNum))
					builder.WriteString(line)
					builder.WriteString("\n")
					// Add pointer to error location
					builder.WriteString("     | ")
					builder.WriteString(strings.Repeat(" ", err.Location.Column-1))
					builder.WriteString("^\n")
				} else {
					builder.WriteString(fmt.Sprintf("%4d | %s\n", lineNum, line))
				}
			}
		}
	}

	// Add suggestions
	if eh.config.ShowSuggestions && len(err.Suggestions) > 0 {
		builder.WriteString("\nSuggestions:\n")
		for _, suggestion := range err.Suggestions {
			builder.WriteString("  - ")
			builder.WriteString(suggestion)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// PrintErrors prints all errors
func (eh *ErrorHandler) PrintErrors() {
	errors := eh.GetErrors()
	for _, err := range errors {
		fmt.Print(eh.FormatError(err))
	}
}

// GetStatistics returns error handler statistics
func (eh *ErrorHandler) GetStatistics() map[string]interface{} {
	eh.mu.RLock()
	lineMapCount := len(eh.lineMaps)
	errorGroupCount := len(eh.errorGroups)
	eh.mu.RUnlock()

	return map[string]interface{}{
		"total_errors":    atomic.LoadInt64(&eh.totalErrors),
		"total_warnings":  atomic.LoadInt64(&eh.totalWarnings),
		"fatal_errors":    atomic.LoadInt64(&eh.fatalErrors),
		"recoveries":      atomic.LoadInt64(&eh.recoveries),
		"line_maps":       lineMapCount,
		"error_groups":    errorGroupCount,
		"error_log_size":  len(eh.errorLog),
	}
}

// Helper methods

func (eh *ErrorHandler) captureStackTrace() []StackFrame {
	var frames []StackFrame

	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		frame := StackFrame{
			Function: fn.Name(),
			File:     filepath.Base(file),
			Line:     line,
		}
		frames = append(frames, frame)
	}

	return frames
}

func (eh *ErrorHandler) parseErrorLocation(errMsg string, defaultFile string) *ErrorLocation {
	// Try to parse location from error message
	// Format: file:line:column: message
	parts := strings.SplitN(errMsg, ":", 4)
	if len(parts) >= 3 {
		var line, column int
		fmt.Sscanf(parts[1], "%d", &line)
		fmt.Sscanf(parts[2], "%d", &column)
		return &ErrorLocation{
			File:   parts[0],
			Line:   line,
			Column: column,
		}
	}

	return &ErrorLocation{
		File: defaultFile,
	}
}

func (eh *ErrorHandler) formatSeverity(severity ErrorSeverity) string {
	switch severity {
	case SeverityHint:
		return "HINT"
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (eh *ErrorHandler) colorize(text string, severity ErrorSeverity) string {
	// ANSI color codes
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorGray   = "\033[90m"
		colorBold   = "\033[1m"
	)

	var color string
	switch severity {
	case SeverityHint:
		color = colorGray
	case SeverityInfo:
		color = colorBlue
	case SeverityWarning:
		color = colorYellow
	case SeverityError:
		color = colorRed
	case SeverityFatal:
		color = colorBold + colorRed
	default:
		return text
	}

	return color + text + colorReset
}

// Error implements the error interface
func (e *TranspilerError) Error() string {
	if e.Location != nil {
		return fmt.Sprintf("%s:%d:%d: %s",
			e.Location.File,
			e.Location.Line,
			e.Location.Column,
			e.Message)
	}
	return e.Message
}

// String returns the string representation of severity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityHint:
		return "hint"
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityFatal:
		return "fatal"
	default:
		return "unknown"
	}
}
