package decorators

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ValidationRule represents a validation rule for parameter validation
type ValidationRule struct {
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	Parameters interface{} `json:"parameters,omitempty"`
}

// ValidationResult holds the result of validation
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Common validation patterns
var (
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// ExtractValidationDecorators extracts validation decorators from comments or tags
func ExtractValidationDecorators(target interface{}) (map[string][]ValidationRule, error) {
	validationRules := make(map[string][]ValidationRule)
	
	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// Extract validation rules from struct fields
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldRules := extractFieldValidationRules(field)
		if len(fieldRules) > 0 {
			validationRules[field.Name] = fieldRules
		}
	}

	return validationRules, nil
}

// extractFieldValidationRules extracts validation rules from field tags
func extractFieldValidationRules(field reflect.StructField) []ValidationRule {
	var rules []ValidationRule

	// Parse validation tag
	if validateTag, ok := field.Tag.Lookup("validate"); ok {
		fieldRules := parseValidationRules(validateTag)
		rules = append(rules, fieldRules...)
	}

	// Parse json tag for required fields
	if jsonTag, ok := field.Tag.Lookup("json"); ok {
		if !strings.Contains(jsonTag, "omitempty") {
			rules = append(rules, ValidationRule{
				Type: "required",
				Name: field.Name,
			})
		}
	}

	return rules
}

// parseValidationRules parses validation rules from a validation tag
func parseValidationRules(tag string) []ValidationRule {
	var rules []ValidationRule
	
	if tag == "" {
		return rules
	}

	// Split by comma to get individual rules
	ruleStrings := strings.Split(tag, ",")
	
	for _, ruleString := range ruleStrings {
		rule := parseValidationRule(strings.TrimSpace(ruleString))
		if rule != nil {
			rules = append(rules, *rule)
		}
	}

	return rules
}

// parseValidationRule parses a single validation rule
func parseValidationRule(ruleString string) *ValidationRule {
	if ruleString == "" {
		return nil
	}

	// Handle rules with parameters (e.g., "min=5", "max=100")
	if strings.Contains(ruleString, "=") {
		parts := strings.SplitN(ruleString, "=", 2)
		ruleName := strings.TrimSpace(parts[0])
		ruleParam := strings.TrimSpace(parts[1])

		return &ValidationRule{
			Type:       ruleName,
			Parameters: ruleParam,
		}
	}

	// Handle simple rules (e.g., "required", "email")
	return &ValidationRule{
		Type: ruleString,
	}
}

// ValidateValue validates a value against validation rules
func ValidateValue(value interface{}, rules []ValidationRule) []string {
	var errors []string

	for _, rule := range rules {
		if err := validateSingleRule(value, rule); err != "" {
			errors = append(errors, err)
		}
	}

	return errors
}

// validateSingleRule validates a value against a single validation rule
func validateSingleRule(value interface{}, rule ValidationRule) string {
	switch rule.Type {
	case "required":
		return validateRequired(value)
	case "email":
		return validateEmail(value)
	case "uuid":
		return validateUUID(value)
	case "min":
		return validateMin(value, rule.Parameters)
	case "max":
		return validateMax(value, rule.Parameters)
	case "minlength":
		return validateMinLength(value, rule.Parameters)
	case "maxlength":
		return validateMaxLength(value, rule.Parameters)
	case "numeric":
		return validateNumeric(value)
	case "alpha":
		return validateAlpha(value)
	case "alphanumeric":
		return validateAlphanumeric(value)
	default:
		return fmt.Sprintf("unknown validation rule: %s", rule.Type)
	}
}

// Individual validation functions
func validateRequired(value interface{}) string {
	if value == nil {
		return "field is required"
	}

	// Check for empty strings
	if str, ok := value.(string); ok && str == "" {
		return "field is required"
	}

	// Check for zero values using reflection
	v := reflect.ValueOf(value)
	if v.IsZero() {
		return "field is required"
	}

	return ""
}

func validateEmail(value interface{}) string {
	str, ok := value.(string)
	if !ok {
		return "field must be a string"
	}

	if !emailPattern.MatchString(str) {
		return "field must be a valid email address"
	}

	return ""
}

func validateUUID(value interface{}) string {
	str, ok := value.(string)
	if !ok {
		return "field must be a string"
	}

	if !uuidPattern.MatchString(str) {
		return "field must be a valid UUID"
	}

	return ""
}

func validateMin(value interface{}, param interface{}) string {
	paramStr, ok := param.(string)
	if !ok {
		return "invalid min parameter"
	}

	minValue, err := strconv.ParseFloat(paramStr, 64)
	if err != nil {
		return "invalid min parameter"
	}

	// Try to convert value to float64
	var numValue float64
	switch v := value.(type) {
	case int:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	case float32:
		numValue = float64(v)
	case float64:
		numValue = v
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			numValue = parsed
		} else {
			return "field must be numeric"
		}
	default:
		return "field must be numeric"
	}

	if numValue < minValue {
		return fmt.Sprintf("field must be at least %g", minValue)
	}

	return ""
}

func validateMax(value interface{}, param interface{}) string {
	paramStr, ok := param.(string)
	if !ok {
		return "invalid max parameter"
	}

	maxValue, err := strconv.ParseFloat(paramStr, 64)
	if err != nil {
		return "invalid max parameter"
	}

	// Try to convert value to float64
	var numValue float64
	switch v := value.(type) {
	case int:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	case float32:
		numValue = float64(v)
	case float64:
		numValue = v
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			numValue = parsed
		} else {
			return "field must be numeric"
		}
	default:
		return "field must be numeric"
	}

	if numValue > maxValue {
		return fmt.Sprintf("field must be at most %g", maxValue)
	}

	return ""
}

func validateMinLength(value interface{}, param interface{}) string {
	paramStr, ok := param.(string)
	if !ok {
		return "invalid minlength parameter"
	}

	minLength, err := strconv.Atoi(paramStr)
	if err != nil {
		return "invalid minlength parameter"
	}

	str, ok := value.(string)
	if !ok {
		return "field must be a string"
	}

	if len(str) < minLength {
		return fmt.Sprintf("field must be at least %d characters long", minLength)
	}

	return ""
}

func validateMaxLength(value interface{}, param interface{}) string {
	paramStr, ok := param.(string)
	if !ok {
		return "invalid maxlength parameter"
	}

	maxLength, err := strconv.Atoi(paramStr)
	if err != nil {
		return "invalid maxlength parameter"
	}

	str, ok := value.(string)
	if !ok {
		return "field must be a string"
	}

	if len(str) > maxLength {
		return fmt.Sprintf("field must be at most %d characters long", maxLength)
	}

	return ""
}

func validateNumeric(value interface{}) string {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return ""
	case string:
		str := value.(string)
		if _, err := strconv.ParseFloat(str, 64); err != nil {
			return "field must be numeric"
		}
		return ""
	default:
		return "field must be numeric"
	}
}

func validateAlpha(value interface{}) string {
	str, ok := value.(string)
	if !ok {
		return "field must be a string"
	}

	for _, r := range str {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return "field must contain only alphabetic characters"
		}
	}

	return ""
}

func validateAlphanumeric(value interface{}) string {
	str, ok := value.(string)
	if !ok {
		return "field must be a string"
	}

	for _, r := range str {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return "field must contain only alphanumeric characters"
		}
	}

	return ""
}

// ValidateStruct validates an entire struct based on validation decorators
func ValidateStruct(target interface{}) (*ValidationResult, error) {
	validationRules, err := ExtractValidationDecorators(target)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	// Validate each field
	for fieldName, rules := range validationRules {
		field := targetValue.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		fieldValue := field.Interface()
		fieldErrors := ValidateValue(fieldValue, rules)

		if len(fieldErrors) > 0 {
			result.Valid = false
			for _, err := range fieldErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", fieldName, err))
			}
		}
	}

	return result, nil
}