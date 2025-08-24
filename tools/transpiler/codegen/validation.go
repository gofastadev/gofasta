package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

// addValidationImportsIfNeeded adds validation imports if needed
func (g *CodeGenerator) addValidationImportsIfNeeded(file *GofaFile) {
	// Check if validation decorators are used
	hasValidation := g.hasValidationDecorators(file)
	if hasValidation {
		g.addImport("regexp")
		g.addImport("net/url")
		g.addImport("time")
		g.addImport("encoding/base64")
		g.addImport("encoding/json")
		g.addImport("net")
	}
}

// generateValidationCodeIfNeeded generates validation code if needed
func (g *CodeGenerator) generateValidationCodeIfNeeded(file *GofaFile) {
	dtoStructs := g.findDTOStructsWithValidation(file)
	if len(dtoStructs) == 0 {
		return
	}

	// Generate validation helper functions
	g.generateValidationHelperFunctions()
	g.writeLine("")

	// Generate validation functions for each DTO
	for _, dto := range dtoStructs {
		g.generateDTOValidationFunction(dto)
		g.writeLine("")
	}
}

// hasValidationDecorators checks if file uses validation decorators
func (g *CodeGenerator) hasValidationDecorators(file *GofaFile) bool {
	// Simple check - in real implementation would scan all decorators
	return false // Placeholder
}

// findDTOStructsWithValidation finds structs that need validation
func (g *CodeGenerator) findDTOStructsWithValidation(file *GofaFile) map[string]*ValidationStructInfo {
	structs := make(map[string]*ValidationStructInfo)
	// Placeholder implementation
	return structs
}

// generateValidationHelperFunctions generates helper functions for validation
func (g *CodeGenerator) generateValidationHelperFunctions() {
	g.writeLine("// ValidationError represents a validation error")
	g.writeLine("type ValidationError struct {")
	g.indent()
	g.writeLine("Field   string")
	g.writeLine("Message string")
	g.writeLine("Value   interface{}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("func (e ValidationError) Error() string {")
	g.indent()
	g.writeLine("return fmt.Sprintf(\"Validation failed for field '%s': %s\", e.Field, e.Message)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// ValidationErrors represents multiple validation errors")
	g.writeLine("type ValidationErrors []ValidationError")
	g.writeLine("")

	g.writeLine("func (e ValidationErrors) Error() string {")
	g.indent()
	g.writeLine("if len(e) == 0 {")
	g.indent()
	g.writeLine("return \"No validation errors\"")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return fmt.Sprintf(\"Validation failed: %d errors\", len(e))")
	g.unindent()
	g.writeLine("}")
}

// generateDTOValidationFunction generates validation function for a DTO
func (g *CodeGenerator) generateDTOValidationFunction(dto *ValidationStructInfo) {
	g.writeLine(fmt.Sprintf("// Validate%s validates the %s struct", dto.Name, dto.Name))
	g.writeLine(fmt.Sprintf("func Validate%s(obj *%s) ValidationErrors {", dto.Name, dto.Name))
	g.indent()
	g.writeLine("var errors ValidationErrors")
	g.writeLine("")

	// Generate validation for each field
	for _, field := range dto.Fields {
		g.generateFieldValidation(field)
	}

	g.writeLine("return errors")
	g.unindent()
	g.writeLine("}")
}

// generateFieldValidation generates validation for a field
func (g *CodeGenerator) generateFieldValidation(field *ValidationFieldInfo) {
	g.writeLine(fmt.Sprintf("// Validate field: %s", field.Name))
	
	for _, rule := range field.Validators {
		g.generateValidationRule(field, rule)
	}
	
	g.writeLine("")
}

// generateValidationRule generates a specific validation rule
func (g *CodeGenerator) generateValidationRule(field *ValidationFieldInfo, rule ValidationRule) {
	switch rule.Type {
	case "Required":
		g.writeLine(fmt.Sprintf("if obj.%s == nil || obj.%s == \"\" {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"Field is required\"})", field.Name))
		g.unindent()
		g.writeLine("}")
	case "Email":
		g.writeLine(fmt.Sprintf("if obj.%s != \"\" && !isValidEmail(obj.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"Invalid email format\"})", field.Name))
		g.unindent()
		g.writeLine("}")
	case "Min":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if obj.%s < %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"Value must be at least %v\"})", field.Name, rule.Args[0]))
			g.unindent()
			g.writeLine("}")
		}
	case "Max":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if obj.%s > %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"Value must be at most %v\"})", field.Name, rule.Args[0]))
			g.unindent()
			g.writeLine("}")
		}
	default:
		g.writeLine(fmt.Sprintf("// TODO: Implement %s validation", rule.Type))
	}
}

// parseValidationDecoratorsFromField parses validation decorators from field decorators
func (g *CodeGenerator) parseValidationDecoratorsFromField(field *FieldNode) []ValidationRule {
	var rules []ValidationRule

	for _, decorator := range field.Decorators {
		decoratorType := GetDecoratorType(decorator.Name)
		if IsValidationDecorator(decoratorType) {
			rule := g.parseValidationRuleFromDecorator(decorator)
			if rule != nil {
				rules = append(rules, *rule)
			}
		}
	}

	return rules
}

// parseValidationRuleFromDecorator parses a single validation rule from a decorator node
func (g *CodeGenerator) parseValidationRuleFromDecorator(decorator *DecoratorNode) *ValidationRule {
	return &ValidationRule{
		Type:    decorator.Name,
		Args:    g.convertDecoratorArgsToInterface(decorator.Args),
		Message: g.getValidationMessage(decorator.Name, g.convertDecoratorArgsToInterface(decorator.Args)),
		Code:    g.getValidationCode(decorator.Name),
	}
}

// IsValidationDecorator checks if a decorator is a validation decorator
func IsValidationDecorator(decoratorType interface{}) bool {
	if str, ok := decoratorType.(string); ok {
		validationDecorators := []string{
			"IsEmail", "IsOptional", "IsNotEmpty", "IsString", "IsBoolean", 
			"IsNumber", "IsInt", "IsArray", "IsObject", "IsUrl", "IsUUID",
			"MinLength", "MaxLength", "Length", "Min", "Max", "IsAlpha",
			"IsAlphanumeric", "IsDecimal", "IsHexColor", "IsIP", "IsJSON",
			"Matches", "Contains", "IsDate", "IsDateString", "IsCreditCard",
			"IsISBN", "IsPhoneNumber", "IsPostalCode", "IsLatitude", "IsLongitude",
		}
		for _, vd := range validationDecorators {
			if str == vd {
				return true
			}
		}
	}
	return false
}

// parseValidationDecorators parses validation decorators from struct tags
func (g *CodeGenerator) parseValidationDecorators(tag string) []ValidationRule {
	var rules []ValidationRule

	// Extract validate: content from struct tag
	validateContent := g.extractTagContent(tag, "validate")
	if validateContent == "" {
		return rules
	}

	// Parse individual decorators like @IsEmail() @Min(18) @Max(120)
	decorators := g.extractDecorators(validateContent)

	for _, decorator := range decorators {
		rule := g.parseValidationRule(decorator)
		if rule != nil {
			rules = append(rules, *rule)
		}
	}

	return rules
}

// parseValidationRule parses a validation rule from decorator string
func (g *CodeGenerator) parseValidationRule(decorator string) *ValidationRule {
	decorator = strings.TrimSpace(decorator)
	if !strings.HasPrefix(decorator, "@") {
		return nil
	}

	// Remove @
	decorator = decorator[1:]

	// Check if it has parentheses
	parenIndex := strings.Index(decorator, "(")
	if parenIndex == -1 {
		// Simple decorator like @IsEmail
		return &ValidationRule{
			Type:    decorator,
			Args:    []interface{}{},
			Message: g.getValidationMessage(decorator, []interface{}{}),
			Code:    g.getValidationCode(decorator),
		}
	}

	// Extract decorator type and arguments
	decoratorType := decorator[:parenIndex]
	argsStr := decorator[parenIndex+1 : len(decorator)-1] // Remove parentheses

	args := g.parseValidationArgs(argsStr)

	return &ValidationRule{
		Type:    decoratorType,
		Args:    args,
		Message: g.getValidationMessage(decoratorType, args),
		Code:    g.getValidationCode(decoratorType),
	}
}

// parseValidationArgs parses validation arguments from string
func (g *CodeGenerator) parseValidationArgs(argsStr string) []interface{} {
	var args []interface{}

	if strings.TrimSpace(argsStr) == "" {
		return args
	}

	// Split by comma
	parts := strings.Split(argsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to parse as number
		if intVal, err := strconv.Atoi(part); err == nil {
			args = append(args, intVal)
			continue
		}

		// Try to parse as float
		if floatVal, err := strconv.ParseFloat(part, 64); err == nil {
			args = append(args, floatVal)
			continue
		}

		// Try to parse as boolean
		if boolVal, err := strconv.ParseBool(part); err == nil {
			args = append(args, boolVal)
			continue
		}

		// Remove quotes if present and treat as string
		if strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"") {
			part = part[1 : len(part)-1]
		}
		args = append(args, part)
	}

	return args
}

// getValidationMessage gets validation message for a rule
func (g *CodeGenerator) getValidationMessage(ruleType string, args []interface{}) string {
	switch ruleType {
	case "IsEmail":
		return "Invalid email address"
	case "IsNotEmpty":
		return "Field cannot be empty"
	case "MinLength":
		if len(args) > 0 {
			return fmt.Sprintf("Must be at least %v characters long", args[0])
		}
		return "Must meet minimum length requirement"
	case "MaxLength":
		if len(args) > 0 {
			return fmt.Sprintf("Must be at most %v characters long", args[0])
		}
		return "Must not exceed maximum length"
	case "Min":
		if len(args) > 0 {
			return fmt.Sprintf("Must be at least %v", args[0])
		}
		return "Must meet minimum value requirement"
	case "Max":
		if len(args) > 0 {
			return fmt.Sprintf("Must be at most %v", args[0])
		}
		return "Must not exceed maximum value"
	default:
		return fmt.Sprintf("Validation failed for %s", ruleType)
	}
}

// getValidationCode gets validation code for a rule
func (g *CodeGenerator) getValidationCode(ruleType string) string {
	switch ruleType {
	case "IsEmail":
		return "INVALID_EMAIL"
	case "IsNotEmpty":
		return "EMPTY_FIELD"
	case "MinLength":
		return "MIN_LENGTH"
	case "MaxLength":
		return "MAX_LENGTH"
	case "Min":
		return "MIN_VALUE"
	case "Max":
		return "MAX_VALUE"
	default:
		return "VALIDATION_ERROR"
	}
}