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
		g.addImport("reflect")
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
	for _, decl := range file.Declarations {
		if service, ok := decl.(*ServiceDeclaration); ok {
			// Check if this is a DTO-like service with validation decorators
			for _, field := range service.Fields {
				for _, decorator := range field.Decorators {
					if IsValidationDecorator(decorator.Name) {
						return true
					}
				}
			}
		}
	}
	return false
}

// findDTOStructsWithValidation finds structs that need validation
func (g *CodeGenerator) findDTOStructsWithValidation(file *GofaFile) map[string]*ValidationStructInfo {
	structs := make(map[string]*ValidationStructInfo)
	
	for _, decl := range file.Declarations {
		if service, ok := decl.(*ServiceDeclaration); ok {
			// Check if this service has validation decorators on its fields
			hasValidationDecorators := false
			var validationFields []*ValidationFieldInfo
			
			for _, field := range service.Fields {
				validators := g.parseValidationDecoratorsFromField(field)
				if len(validators) > 0 {
					hasValidationDecorators = true
					validationFields = append(validationFields, &ValidationFieldInfo{
						Name:       field.Name,
						Type:       field.Type,
						Validators: validators,
					})
				}
			}
			
			// If this service has validation decorators, add it to structs map
			if hasValidationDecorators {
				structs[service.Name] = &ValidationStructInfo{
					Name:   service.Name,
					Fields: validationFields,
				}
			}
		}
	}
	
	return structs
}

// generateValidationHelperFunctions generates helper functions for validation
func (g *CodeGenerator) generateValidationHelperFunctions() {
	g.writeLine("// ValidationError represents a validation error")
	g.writeLine("type ValidationError struct {")
	g.indent()
	g.writeLine("Field   string      `json:\"field\"`")
	g.writeLine("Message string      `json:\"message\"`")
	g.writeLine("Value   interface{} `json:\"value\"`")
	g.writeLine("Code    string      `json:\"code\"`")
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
	g.writeLine("")

	g.writeLine("// ValidationResult represents the result of a validation operation")
	g.writeLine("type ValidationResult struct {")
	g.indent()
	g.writeLine("IsValid bool              `json:\"isValid\"`")
	g.writeLine("Errors  []ValidationError `json:\"errors,omitempty\"`")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Add specific validation helper functions
	g.generateEmailValidationHelper()
	g.generateURLValidationHelper()
	g.generateNumericValidationHelper()
	g.generateAlphaValidationHelper()
	g.generateAlphanumericValidationHelper()
	g.generateIntValidationHelper()
	g.generateFloatValidationHelper()
	g.generateBooleanValidationHelper()
	g.generateDateValidationHelper()
	g.generateIPValidationHelper()
	g.generateJSONValidationHelper()
	g.generateBase64ValidationHelper()
	g.generateHexColorValidationHelper()
	g.generatePhoneNumberValidationHelper()
	g.generateCreditCardValidationHelper()
	g.generateISBNValidationHelper()
	g.generateArrayMinSizeValidationHelper()
}

// generateDTOValidationFunction generates validation function for a DTO
func (g *CodeGenerator) generateDTOValidationFunction(dto *ValidationStructInfo) {
	g.writeLine(fmt.Sprintf("// Validate%s validates the %s struct", dto.Name, dto.Name))
	g.writeLine(fmt.Sprintf("func Validate%s(dto *%s) ValidationErrors {", dto.Name, dto.Name))
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
	
	// Check if field has IsOptional or ValidateIf decorator
	hasOptional := false
	var validateIfCondition string
	var nonConditionalRules []ValidationRule
	
	for _, rule := range field.Validators {
		if rule.Type == "IsOptional" {
			hasOptional = true
		} else if rule.Type == "ValidateIf" {
			if len(rule.Args) > 0 {
				validateIfCondition = rule.Args[0].(string)
			}
		} else {
			nonConditionalRules = append(nonConditionalRules, rule)
		}
	}
	
	// Handle ValidateIf conditional validation
	if validateIfCondition != "" {
		g.writeLine("// ValidateIf validation")
		g.writeLine(fmt.Sprintf("if %s {", validateIfCondition))
		g.indent()
		for _, rule := range nonConditionalRules {
			g.generateValidationRule(field, rule)
		}
		g.unindent()
		g.writeLine("}")
	} else if hasOptional {
		// For optional fields, wrap the validations in a non-empty check
		if len(nonConditionalRules) > 0 {
			// Check if it's an array field
			if strings.Contains(field.Type, "[]") {
				g.writeLine(fmt.Sprintf("if dto.%s != nil && len(dto.%s) > 0 {", field.Name, field.Name))
			} else {
				g.writeLine(fmt.Sprintf("if strings.TrimSpace(dto.%s) != \"\" {", field.Name))
			}
			g.indent()
			for _, rule := range nonConditionalRules {
				g.generateValidationRule(field, rule)
			}
			g.unindent()
			g.writeLine("}")
		}
	} else {
		for _, rule := range field.Validators {
			g.generateValidationRule(field, rule)
		}
	}
	
	g.writeLine("")
}

// generateValidationRule generates a specific validation rule
func (g *CodeGenerator) generateValidationRule(field *ValidationFieldInfo, rule ValidationRule) {
	switch rule.Type {
	case "Required":
		g.writeLine(fmt.Sprintf("if dto.%s == nil || dto.%s == \"\" {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsEmail":
		g.writeLine("// IsEmail validation")
		g.writeLine(fmt.Sprintf("if !isValidEmail(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsNotEmpty":
		g.writeLine("// IsNotEmpty validation")
		g.writeLine(fmt.Sprintf("if dto.%s == \"\" {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "Min":
		g.writeLine("// Min validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if dto.%s < %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "Max":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if dto.%s > %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "Length":
		if len(rule.Args) >= 2 {
			g.writeLine(fmt.Sprintf("if len(dto.%s) < %v || len(dto.%s) > %v {", field.Name, rule.Args[0], field.Name, rule.Args[1]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "IsArray":
		g.writeLine(fmt.Sprintf("if reflect.TypeOf(dto.%s).Kind() != reflect.Slice {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "ArrayMinSize":
		g.writeLine("// ArrayMinSize validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if dto.%s != nil && len(dto.%s) < %v {", field.Name, field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "ArrayMaxSize":
		g.writeLine("// ArrayMaxSize validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if dto.%s != nil && len(dto.%s) > %v {", field.Name, field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "ArrayNotEmpty":
		g.writeLine("// ArrayNotEmpty validation")
		g.writeLine(fmt.Sprintf("if dto.%s == nil || len(dto.%s) == 0 {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsURL":
		g.writeLine(fmt.Sprintf("if dto.%s != \"\" && !isValidURL(dto.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsNumeric":
		g.writeLine(fmt.Sprintf("if dto.%s != \"\" && !isNumeric(dto.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsAlphanumeric":
		g.writeLine(fmt.Sprintf("if dto.%s != \"\" && !isAlphanumeric(dto.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsAlpha":
		g.writeLine(fmt.Sprintf("if dto.%s != \"\" && !isAlpha(dto.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsPositive":
		g.writeLine("// IsPositive validation")
		g.writeLine(fmt.Sprintf("if dto.%s <= 0 {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsFloat":
		g.writeLine("// IsFloat validation")
		g.writeLine(fmt.Sprintf("if !isFloat(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsInt":
		g.writeLine("// IsInt validation")
		g.writeLine(fmt.Sprintf("if !isInt(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsBoolean":
		g.writeLine("// IsBoolean validation")
		g.writeLine(fmt.Sprintf("if !isBoolean(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsDate":
		g.writeLine("// IsDate validation")
		g.writeLine(fmt.Sprintf("if !isDate(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsIP":
		g.writeLine("// IsIP validation")
		g.writeLine(fmt.Sprintf("if !isIP(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsJSON":
		g.writeLine("// IsJSON validation")
		g.writeLine(fmt.Sprintf("if !isJSON(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsBase64":
		g.writeLine("// IsBase64 validation")
		g.writeLine(fmt.Sprintf("if !isBase64(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsHexColor":
		g.writeLine("// IsHexColor validation")
		g.writeLine(fmt.Sprintf("if !isHexColor(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsPhoneNumber":
		g.writeLine("// IsPhoneNumber validation")
		g.writeLine(fmt.Sprintf("if !isPhoneNumber(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsCreditCard":
		g.writeLine("// IsCreditCard validation")
		g.writeLine(fmt.Sprintf("if !isCreditCard(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsISBN":
		g.writeLine("// IsISBN validation")
		g.writeLine(fmt.Sprintf("if !isISBN(dto.%s) {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsEmpty":
		g.writeLine("// IsEmpty validation")
		if strings.Contains(field.Type, "[]") {
			g.writeLine(fmt.Sprintf("if dto.%s != nil && len(dto.%s) > 0 {", field.Name, field.Name))
		} else {
			g.writeLine(fmt.Sprintf("if strings.TrimSpace(dto.%s) != \"\" {", field.Name))
		}
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "MinLength":
		g.writeLine("// MinLength validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if len(dto.%s) < %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "MaxLength":
		g.writeLine("// MaxLength validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if len(dto.%s) > %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "IsDefined":
		g.writeLine("// IsDefined validation")
		g.writeLine(fmt.Sprintf("if dto.%s == \"\" {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "NotEquals":
		g.writeLine("// NotEquals validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if dto.%s == \"%v\" {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "Equals":
		g.writeLine("// Equals validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if dto.%s != \"%v\" {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "Contains":
		g.writeLine("// Contains validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if !strings.Contains(dto.%s, \"%v\") {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "NotContains":
		g.writeLine("// NotContains validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if strings.Contains(dto.%s, \"%v\") {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "IsIn":
		g.writeLine("// IsIn validation")
		if len(rule.Args) > 0 {
			var values []string
			for _, arg := range rule.Args {
				if str, ok := arg.(string); ok {
					values = append(values, fmt.Sprintf("\"%s\"", str))
				} else {
					values = append(values, fmt.Sprintf("%v", arg))
				}
			}
			g.writeLine(fmt.Sprintf("validValues := []interface{}{%s}", strings.Join(values, ", ")))
			g.writeLine("found := false")
			g.writeLine("for _, v := range validValues {")
			g.indent()
			g.writeLine(fmt.Sprintf("if dto.%s == v {", field.Name))
			g.indent()
			g.writeLine("found = true")
			g.writeLine("break")
			g.unindent()
			g.writeLine("}")
			g.unindent()
			g.writeLine("}")
			g.writeLine("if !found {")
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "IsNotIn":
		g.writeLine("// IsNotIn validation")
		if len(rule.Args) > 0 {
			var values []string
			for _, arg := range rule.Args {
				if str, ok := arg.(string); ok {
					values = append(values, fmt.Sprintf("\"%s\"", str))
				} else {
					values = append(values, fmt.Sprintf("%v", arg))
				}
			}
			g.writeLine(fmt.Sprintf("invalidValues := []interface{}{%s}", strings.Join(values, ", ")))
			g.writeLine("for _, v := range invalidValues {")
			g.indent()
			g.writeLine(fmt.Sprintf("if dto.%s == v {", field.Name))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.writeLine("break")
			g.unindent()
			g.writeLine("}")
			g.unindent()
			g.writeLine("}")
		}
	case "Matches":
		g.writeLine("// Matches validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("matched, _ := regexp.MatchString(\"%v\", dto.%s)", rule.Args[0], field.Name))
			g.writeLine("if !matched {")
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "IsLowercase":
		g.writeLine("// IsLowercase validation")
		g.writeLine(fmt.Sprintf("if dto.%s != strings.ToLower(dto.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsUppercase":
		g.writeLine("// IsUppercase validation")
		g.writeLine(fmt.Sprintf("if dto.%s != strings.ToUpper(dto.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "Custom":
		g.writeLine("// Custom validation")
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if !%v(dto.%s) {", rule.Args[0], field.Name))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
			g.unindent()
			g.writeLine("}")
		}
	case "ValidateNested":
		g.writeLine("// ValidateNested validation")
		g.writeLine(fmt.Sprintf("if nestedErrors := Validate%s(&dto.%s); len(nestedErrors) > 0 {", strings.Title(field.Type), field.Name))
		g.indent()
		g.writeLine("for _, nestedError := range nestedErrors {")
		g.indent()
		g.writeLine(fmt.Sprintf("nestedError.Field = \"%s.\" + nestedError.Field", field.Name))
		g.writeLine("errors = append(errors, nestedError)")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
	case "IsNegative":
		g.writeLine("// IsNegative validation")
		g.writeLine(fmt.Sprintf("if dto.%s >= 0 {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsPastDate":
		g.writeLine("// IsPastDate validation")
		g.writeLine("now := time.Now()")
		g.writeLine(fmt.Sprintf("if parsedDate, err := time.Parse(time.RFC3339, dto.%s); err == nil {", field.Name))
		g.indent()
		g.writeLine("if !parsedDate.Before(now) {")
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"invalid date format\", Code: \"INVALID_DATE\", Value: dto.%s})", field.Name, field.Name))
		g.unindent()
		g.writeLine("}")
	case "IsFutureDate":
		g.writeLine("// IsFutureDate validation")
		g.writeLine("now := time.Now()")
		g.writeLine(fmt.Sprintf("if parsedDate, err := time.Parse(time.RFC3339, dto.%s); err == nil {", field.Name))
		g.indent()
		g.writeLine("if !parsedDate.After(now) {")
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\", Value: dto.%s})", field.Name, rule.Message, rule.Code, field.Name))
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"invalid date format\", Code: \"INVALID_DATE\", Value: dto.%s})", field.Name, field.Name))
		g.unindent()
		g.writeLine("}")
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
			"IsNumber", "IsInt", "IsArray", "IsObject", "IsURL", "IsUUID",
			"MinLength", "MaxLength", "Length", "Min", "Max", "IsAlpha",
			"IsAlphanumeric", "IsDecimal", "IsHexColor", "IsIP", "IsJSON",
			"Matches", "Contains", "IsDate", "IsDateString", "IsCreditCard",
			"IsISBN", "IsPhoneNumber", "IsPostalCode", "IsLatitude", "IsLongitude",
			"IsNumeric", "IsPositive", "ArrayMinSize", "ArrayMaxSize", "ArrayNotEmpty",
			"IsFloat", "IsBase64", "IsEmpty", "IsDefined", "NotEquals", "Equals",
			"NotContains", "IsIn", "IsNotIn", "IsLowercase", "IsUppercase", "Custom",
			"ValidateNested", "IsNegative", "IsPastDate", "IsFutureDate", "ValidateIf",
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
		return "must be a valid email address"
	case "IsNotEmpty":
		return "must not be empty"
	case "MinLength":
		if len(args) > 0 {
			return fmt.Sprintf("must be at least %v characters long", args[0])
		}
		return "must meet minimum length requirement"
	case "MaxLength":
		if len(args) > 0 {
			return fmt.Sprintf("must be at most %v characters long", args[0])
		}
		return "must not exceed maximum length"
	case "Min":
		if len(args) > 0 {
			return fmt.Sprintf("must be at least %v", args[0])
		}
		return "must meet minimum value requirement"
	case "Max":
		if len(args) > 0 {
			return fmt.Sprintf("must be at most %v", args[0])
		}
		return "must not exceed maximum value"
	case "Length":
		if len(args) >= 2 {
			return fmt.Sprintf("must be between %v and %v characters", args[0], args[1])
		} else if len(args) == 1 {
			return fmt.Sprintf("must be exactly %v characters", args[0])
		}
		return "must meet length requirements"
	case "IsArray":
		return "must be an array"
	case "ArrayMinSize":
		if len(args) > 0 {
			return fmt.Sprintf("must contain at least %v item(s)", args[0])
		}
		return "array must meet minimum size requirement"
	case "ArrayMaxSize":
		if len(args) > 0 {
			return fmt.Sprintf("must contain at most %v item(s)", args[0])
		}
		return "array must not exceed maximum size"
	case "ArrayNotEmpty":
		return "array must not be empty"
	case "IsURL":
		return "must be a valid URL"
	case "IsNumeric":
		return "must contain only numbers"
	case "IsAlphanumeric":
		return "must contain only letters and numbers"
	case "IsAlpha":
		return "must contain only letters"
	case "IsPositive":
		return "must be a positive number"
	case "IsFloat":
		return "must be a floating point number"
	case "IsInt":
		return "must be an integer"
	case "IsBoolean":
		return "must be a boolean"
	case "IsDate":
		return "must be a valid date"
	case "IsIP":
		return "must be a valid IP address"
	case "IsJSON":
		return "must be valid JSON"
	case "IsBase64":
		return "must be valid Base64"
	case "IsHexColor":
		return "must be a valid hex color"
	case "IsPhoneNumber":
		return "must be a valid phone number"
	case "IsCreditCard":
		return "must be a valid credit card number"
	case "IsISBN":
		return "must be a valid ISBN"
	case "IsEmpty":
		return "must be empty"
	case "IsDefined":
		return "must be defined"
	case "NotEquals":
		if len(args) > 0 {
			return fmt.Sprintf("must not equal %v", args[0])
		}
		return "must not equal specified value"
	case "Equals":
		if len(args) > 0 {
			return fmt.Sprintf("must equal %v", args[0])
		}
		return "must equal specified value"
	case "Contains":
		if len(args) > 0 {
			return fmt.Sprintf("must contain %v", args[0])
		}
		return "must contain specified value"
	case "NotContains":
		if len(args) > 0 {
			return fmt.Sprintf("must not contain %v", args[0])
		}
		return "must not contain specified value"
	case "IsIn":
		return "must be one of the allowed values"
	case "IsNotIn":
		return "must not be one of the forbidden values"
	case "Matches":
		if len(args) > 0 {
			return fmt.Sprintf("must match pattern %v", args[0])
		}
		return "must match specified pattern"
	case "IsLowercase":
		return "must be lowercase"
	case "IsUppercase":
		return "must be uppercase"
	case "Custom":
		return "custom validation failed"
	case "ValidateNested":
		return "nested validation failed"
	case "IsNegative":
		return "must be a negative number"
	case "IsPastDate":
		return "must be a date in the past"
	case "IsFutureDate":
		return "must be a date in the future"
	case "ValidateIf":
		return "conditional validation failed"
	default:
		return fmt.Sprintf("validation failed for %s", ruleType)
	}
}

// getValidationCode gets validation code for a rule
func (g *CodeGenerator) getValidationCode(ruleType string) string {
	switch ruleType {
	case "IsEmail":
		return "IS_EMAIL"
	case "IsNotEmpty":
		return "IS_NOT_EMPTY"
	case "MinLength":
		return "MIN_LENGTH"
	case "MaxLength":
		return "MAX_LENGTH"
	case "Length":
		return "LENGTH"
	case "Min":
		return "MIN_VALUE"
	case "Max":
		return "MAX_VALUE"
	case "IsArray":
		return "IS_ARRAY"
	case "ArrayMinSize":
		return "ARRAY_MIN_SIZE"
	case "ArrayMaxSize":
		return "ARRAY_MAX_SIZE"
	case "ArrayNotEmpty":
		return "ARRAY_NOT_EMPTY"
	case "IsURL":
		return "IS_URL"
	case "IsNumeric":
		return "IS_NUMERIC"
	case "IsAlphanumeric":
		return "IS_ALPHANUMERIC"
	case "IsAlpha":
		return "IS_ALPHA"
	case "IsPositive":
		return "IS_POSITIVE"
	case "IsFloat":
		return "IS_FLOAT"
	case "IsInt":
		return "IS_INT"
	case "IsBoolean":
		return "IS_BOOLEAN"
	case "IsDate":
		return "IS_DATE"
	case "IsIP":
		return "IS_IP"
	case "IsJSON":
		return "IS_JSON"
	case "IsBase64":
		return "IS_BASE64"
	case "IsHexColor":
		return "IS_HEX_COLOR"
	case "IsPhoneNumber":
		return "IS_PHONE_NUMBER"
	case "IsCreditCard":
		return "IS_CREDIT_CARD"
	case "IsISBN":
		return "IS_ISBN"
	case "IsEmpty":
		return "IS_EMPTY"
	case "IsDefined":
		return "IS_DEFINED"
	case "NotEquals":
		return "NOT_EQUALS"
	case "Equals":
		return "EQUALS"
	case "Contains":
		return "CONTAINS"
	case "NotContains":
		return "NOT_CONTAINS"
	case "IsIn":
		return "IS_IN"
	case "IsNotIn":
		return "IS_NOT_IN"
	case "Matches":
		return "MATCHES"
	case "IsLowercase":
		return "IS_LOWERCASE"
	case "IsUppercase":
		return "IS_UPPERCASE"
	case "Custom":
		return "CUSTOM"
	case "ValidateNested":
		return "VALIDATE_NESTED"
	case "IsNegative":
		return "IS_NEGATIVE"
	case "IsPastDate":
		return "IS_PAST_DATE"
	case "IsFutureDate":
		return "IS_FUTURE_DATE"
	case "ValidateIf":
		return "VALIDATE_IF"
	default:
		return "VALIDATION_ERROR"
	}
}

// generateEmailValidationHelper generates email validation helper
func (g *CodeGenerator) generateEmailValidationHelper() {
	g.writeLine("// Email validation helper")
	g.writeLine("emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$`")
	g.writeLine("")
	g.writeLine("func isValidEmail(email string) bool {")
	g.indent()
	g.writeLine("matched, _ := regexp.MatchString(emailRegex, email)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateURLValidationHelper generates URL validation helper
func (g *CodeGenerator) generateURLValidationHelper() {
	g.writeLine("// URL validation helper")
	g.writeLine("func isValidURL(url string) bool {")
	g.indent()
	g.writeLine("_, err := url.Parse(url)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateNumericValidationHelper generates numeric validation helper
func (g *CodeGenerator) generateNumericValidationHelper() {
	g.writeLine("// Numeric validation helper")
	g.writeLine("numericRegex := `^[0-9]+$`")
	g.writeLine("")
	g.writeLine("func isNumeric(str string) bool {")
	g.indent()
	g.writeLine("matched, _ := regexp.MatchString(numericRegex, str)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateAlphaValidationHelper generates alpha validation helper
func (g *CodeGenerator) generateAlphaValidationHelper() {
	g.writeLine("// Alpha validation helper")
	g.writeLine("alphaRegex := `^[a-zA-Z]+$`")
	g.writeLine("")
	g.writeLine("func isAlpha(str string) bool {")
	g.indent()
	g.writeLine("matched, _ := regexp.MatchString(alphaRegex, str)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateAlphanumericValidationHelper generates alphanumeric validation helper
func (g *CodeGenerator) generateAlphanumericValidationHelper() {
	g.writeLine("// Alphanumeric validation helper")
	g.writeLine("alphanumericRegex := `^[a-zA-Z0-9]+$`")
	g.writeLine("")
	g.writeLine("func isAlphanumeric(str string) bool {")
	g.indent()
	g.writeLine("matched, _ := regexp.MatchString(alphanumericRegex, str)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateIntValidationHelper generates integer validation helper
func (g *CodeGenerator) generateIntValidationHelper() {
	g.writeLine("// Integer validation helper")
	g.writeLine("func isInt(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case int, int8, int16, int32, int64:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case string:")
	g.indent()
	g.writeLine("_, err := strconv.Atoi(v)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateFloatValidationHelper generates float validation helper
func (g *CodeGenerator) generateFloatValidationHelper() {
	g.writeLine("// Float validation helper")
	g.writeLine("func isFloat(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case float32, float64:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case string:")
	g.indent()
	g.writeLine("_, err := strconv.ParseFloat(v, 64)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateBooleanValidationHelper generates boolean validation helper
func (g *CodeGenerator) generateBooleanValidationHelper() {
	g.writeLine("// Boolean validation helper")
	g.writeLine("func isBoolean(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case bool:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case string:")
	g.indent()
	g.writeLine("_, err := strconv.ParseBool(v)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateDateValidationHelper generates date validation helper
func (g *CodeGenerator) generateDateValidationHelper() {
	g.writeLine("// Date validation helper")
	g.writeLine("func isDate(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case time.Time:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case string:")
	g.indent()
	g.writeLine("dateFormats := []string{")
	g.indent()
	g.writeLine("time.RFC3339,")
	g.writeLine("\"2006-01-02\",")
	g.writeLine("\"2006-01-02T15:04:05\",")
	g.writeLine("\"01/02/2006\",")
	g.writeLine("\"01-02-2006\",")
	g.unindent()
	g.writeLine("}")
	g.writeLine("for _, format := range dateFormats {")
	g.indent()
	g.writeLine("if _, err := time.Parse(format, v); err == nil {")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateIPValidationHelper generates IP validation helper
func (g *CodeGenerator) generateIPValidationHelper() {
	g.writeLine("// IP validation helper")
	g.writeLine("func isIP(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("return net.ParseIP(str) != nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateJSONValidationHelper generates JSON validation helper
func (g *CodeGenerator) generateJSONValidationHelper() {
	g.writeLine("// JSON validation helper")
	g.writeLine("func isJSON(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("return json.Valid([]byte(str))")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateBase64ValidationHelper generates Base64 validation helper
func (g *CodeGenerator) generateBase64ValidationHelper() {
	g.writeLine("// Base64 validation helper")
	g.writeLine("func isBase64(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("_, err := base64.StdEncoding.DecodeString(str)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateHexColorValidationHelper generates hex color validation helper
func (g *CodeGenerator) generateHexColorValidationHelper() {
	g.writeLine("// Hex color validation helper")
	g.writeLine("func isHexColor(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if !strings.HasPrefix(str, \"#\") {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("hexPart := str[1:]")
	g.writeLine("if len(hexPart) != 3 && len(hexPart) != 6 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("hexRegex := `^[A-Fa-f0-9]+$`")
	g.writeLine("matched, _ := regexp.MatchString(hexRegex, hexPart)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generatePhoneNumberValidationHelper generates phone number validation helper
func (g *CodeGenerator) generatePhoneNumberValidationHelper() {
	g.writeLine("// Phone number validation helper")
	g.writeLine("func isPhoneNumber(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("cleanedPhone := strings.Map(func(r rune) rune {")
	g.indent()
	g.writeLine("if r >= '0' && r <= '9' || r == '+' {")
	g.indent()
	g.writeLine("return r")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return -1")
	g.unindent()
	g.writeLine("}, str)")
	g.writeLine("phoneRegex := `^\\+?[1-9]\\d{7,14}$`")
	g.writeLine("matched, _ := regexp.MatchString(phoneRegex, cleanedPhone)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateCreditCardValidationHelper generates credit card validation helper
func (g *CodeGenerator) generateCreditCardValidationHelper() {
	g.writeLine("// Credit card validation helper using Luhn algorithm")
	g.writeLine("func isCreditCard(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("// Remove spaces and validate basic format")
	g.writeLine("ccNumber := strings.ReplaceAll(str, \" \", \"\")") 
	g.writeLine("ccRegex := `^[0-9]{13,19}$`")
	g.writeLine("if matched, _ := regexp.MatchString(ccRegex, ccNumber); !matched {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("// Luhn algorithm check")
	g.writeLine("sum := 0")
	g.writeLine("alternate := false")
	g.writeLine("for i := len(ccNumber) - 1; i >= 0; i-- {")
	g.indent()
	g.writeLine("digit := int(ccNumber[i] - '0')")
	g.writeLine("if alternate {")
	g.indent()
	g.writeLine("digit *= 2")
	g.writeLine("if digit > 9 {")
	g.indent()
	g.writeLine("digit = digit%10 + digit/10")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("sum += digit")
	g.writeLine("alternate = !alternate")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return sum%10 == 0")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateISBNValidationHelper generates ISBN validation helper
func (g *CodeGenerator) generateISBNValidationHelper() {
	g.writeLine("// ISBN validation helper")
	g.writeLine("func isISBN(value interface{}) bool {")
	g.indent()
	g.writeLine("if str, ok := value.(string); ok {")
	g.indent()
	g.writeLine("// Remove any spaces and hyphens")
	g.writeLine("cleanedISBN := strings.ReplaceAll(strings.ReplaceAll(str, \"-\", \"\"), \" \", \"\")")
	g.writeLine("if len(cleanedISBN) == 10 {")
	g.indent()
	g.writeLine("return isISBN10(cleanedISBN)")
	g.unindent()
	g.writeLine("} else if len(cleanedISBN) == 13 {")
	g.indent()
	g.writeLine("return isISBN13(cleanedISBN)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Add the helper functions for ISBN10 and ISBN13
	g.writeLine("// isISBN10 validates ISBN-10")
	g.writeLine("func isISBN10(isbn string) bool {")
	g.indent()
	g.writeLine("if len(isbn) != 10 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("sum := 0")
	g.writeLine("for i := 0; i < 9; i++ {")
	g.indent()
	g.writeLine("if isbn[i] < '0' || isbn[i] > '9' {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("sum += int(isbn[i]-'0') * (10 - i)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("lastChar := isbn[9]")
	g.writeLine("if lastChar == 'X' || lastChar == 'x' {")
	g.indent()
	g.writeLine("sum += 10")
	g.unindent()
	g.writeLine("} else if lastChar >= '0' && lastChar <= '9' {")
	g.indent()
	g.writeLine("sum += int(lastChar - '0')")
	g.unindent()
	g.writeLine("} else {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return sum%11 == 0")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	g.writeLine("// isISBN13 validates ISBN-13")
	g.writeLine("func isISBN13(isbn string) bool {")
	g.indent()
	g.writeLine("if len(isbn) != 13 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("sum := 0")
	g.writeLine("for i := 0; i < 12; i++ {")
	g.indent()
	g.writeLine("if isbn[i] < '0' || isbn[i] > '9' {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("weight := 1")
	g.writeLine("if i%2 == 1 {")
	g.indent()
	g.writeLine("weight = 3")
	g.unindent()
	g.writeLine("}")
	g.writeLine("sum += int(isbn[i]-'0') * weight")
	g.unindent()
	g.writeLine("}")
	g.writeLine("if isbn[12] < '0' || isbn[12] > '9' {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("checkDigit := (10 - (sum % 10)) % 10")
	g.writeLine("return checkDigit == int(isbn[12]-'0')")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateArrayMinSizeValidationHelper generates ArrayMinSize validation helper
func (g *CodeGenerator) generateArrayMinSizeValidationHelper() {
	g.writeLine("// ArrayMinSize validation helper")
	g.writeLine("func validateArrayMinSize(arr interface{}, minSize int) bool {")
	g.indent()
	g.writeLine("if arr == nil {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("v := reflect.ValueOf(arr)")
	g.writeLine("if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return v.Len() >= minSize")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}