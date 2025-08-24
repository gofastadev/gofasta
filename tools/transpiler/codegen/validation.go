package codegen

import "fmt"

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