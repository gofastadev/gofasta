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
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsEmail":
		g.writeLine(fmt.Sprintf("if obj.%s != \"\" && !isValidEmail(obj.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsNotEmpty":
		g.writeLine(fmt.Sprintf("if obj.%s == \"\" {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "Min":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if obj.%s < %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "Max":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if obj.%s > %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "Length":
		if len(rule.Args) >= 2 {
			g.writeLine(fmt.Sprintf("if len(obj.%s) < %v || len(obj.%s) > %v {", field.Name, rule.Args[0], field.Name, rule.Args[1]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "IsArray":
		g.writeLine(fmt.Sprintf("if reflect.TypeOf(obj.%s).Kind() != reflect.Slice {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "ArrayMinSize":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if len(obj.%s) < %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "ArrayMaxSize":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if len(obj.%s) > %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "ArrayNotEmpty":
		g.writeLine(fmt.Sprintf("if len(obj.%s) == 0 {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsURL":
		g.writeLine(fmt.Sprintf("if obj.%s != \"\" && !isValidURL(obj.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsNumeric":
		g.writeLine(fmt.Sprintf("if obj.%s != \"\" && !isNumeric(obj.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsAlphanumeric":
		g.writeLine(fmt.Sprintf("if obj.%s != \"\" && !isAlphanumeric(obj.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsAlpha":
		g.writeLine(fmt.Sprintf("if obj.%s != \"\" && !isAlpha(obj.%s) {", field.Name, field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsPositive":
		g.writeLine(fmt.Sprintf("if obj.%s <= 0 {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsFloat":
		g.writeLine(fmt.Sprintf("if _, err := strconv.ParseFloat(fmt.Sprintf(\"%%v\", obj.%s), 64); err != nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsInt":
		g.writeLine(fmt.Sprintf("if _, err := strconv.Atoi(fmt.Sprintf(\"%%v\", obj.%s)); err != nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsBoolean":
		g.writeLine(fmt.Sprintf("if _, err := strconv.ParseBool(fmt.Sprintf(\"%%v\", obj.%s)); err != nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsDate":
		g.writeLine(fmt.Sprintf("if _, err := time.Parse(time.RFC3339, fmt.Sprintf(\"%%v\", obj.%s)); err != nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsIP":
		g.writeLine(fmt.Sprintf("if net.ParseIP(fmt.Sprintf(\"%%v\", obj.%s)) == nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsJSON":
		g.writeLine(fmt.Sprintf("var jsonData interface{}"))
		g.writeLine(fmt.Sprintf("if err := json.Unmarshal([]byte(fmt.Sprintf(\"%%v\", obj.%s)), &jsonData); err != nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsBase64":
		g.writeLine(fmt.Sprintf("if _, err := base64.StdEncoding.DecodeString(fmt.Sprintf(\"%%v\", obj.%s)); err != nil {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "IsEmpty":
		g.writeLine(fmt.Sprintf("if fmt.Sprintf(\"%%v\", obj.%s) != \"\" {", field.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
		g.unindent()
		g.writeLine("}")
	case "MinLength":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if len(fmt.Sprintf(\"%%v\", obj.%s)) < %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
			g.unindent()
			g.writeLine("}")
		}
	case "MaxLength":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("if len(fmt.Sprintf(\"%%v\", obj.%s)) > %v {", field.Name, rule.Args[0]))
			g.indent()
			g.writeLine(fmt.Sprintf("errors = append(errors, ValidationError{Field: \"%s\", Message: \"%s\", Code: \"%s\"})", field.Name, rule.Message, rule.Code))
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
			"IsNumber", "IsInt", "IsArray", "IsObject", "IsURL", "IsUUID",
			"MinLength", "MaxLength", "Length", "Min", "Max", "IsAlpha",
			"IsAlphanumeric", "IsDecimal", "IsHexColor", "IsIP", "IsJSON",
			"Matches", "Contains", "IsDate", "IsDateString", "IsCreditCard",
			"IsISBN", "IsPhoneNumber", "IsPostalCode", "IsLatitude", "IsLongitude",
			"IsNumeric", "IsPositive", "ArrayMinSize", "ArrayMaxSize", "ArrayNotEmpty",
			"IsFloat", "IsBase64", "IsEmpty",
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
		return "must be a valid float number"
	case "IsInt":
		return "must be a valid integer"
	case "IsBoolean":
		return "must be a valid boolean"
	case "IsDate":
		return "must be a valid date"
	case "IsIP":
		return "must be a valid IP address"
	case "IsJSON":
		return "must be valid JSON"
	case "IsBase64":
		return "must be valid base64"
	case "IsEmpty":
		return "must be empty"
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
	case "IsEmpty":
		return "IS_EMPTY"
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