package codegen

import (
	"fmt"
	"strings"
)


// generateQueryParameterExtraction generates query parameter extraction code
func (g *CodeGenerator) generateQueryParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	queryName := g.getDecoratorArgValue(decorator, 0)
	if queryName == "" {
		queryName = param.Name
	}

	// Get query parameter options from decorator
	options := g.getQueryParameterOptions(decorator)

	// Generate variable declaration
	g.writeLine(fmt.Sprintf("var %s %s", param.Name, param.Type))

	// Get raw query value
	g.writeLine(fmt.Sprintf("queryValue := ctx.GetQuery(\"%s\")", queryName))

	// Handle default value
	if options.DefaultValue != "" {
		g.writeLine(fmt.Sprintf("if queryValue == \"\" {"))
		g.indent()
		g.writeLine(fmt.Sprintf("queryValue = \"%s\"", options.DefaultValue))
		g.unindent()
		g.writeLine("}")
	}

	// Handle required validation
	if options.Required && options.DefaultValue == "" {
		g.writeLine("if queryValue == \"\" {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Query parameter '%s' is required\"})", queryName))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
	}

	// Handle type conversion based on parameter type and options
	g.generateQueryTypeConversion(param, "queryValue", options)

	g.writeLine("")
}

// generateHeaderParameterExtraction generates header parameter extraction code
func (g *CodeGenerator) generateHeaderParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	headerName := g.getDecoratorArgValue(decorator, 0)
	if headerName == "" {
		headerName = param.Name
	}

	// Get header parameter options from decorator
	options := g.getHeaderParameterOptions(decorator)
	
	// Generate variable declaration
	g.writeLine(fmt.Sprintf("var %s %s", param.Name, param.Type))
	
	// Get raw header value
	g.writeLine(fmt.Sprintf("headerValue := ctx.GetHeader(\"%s\")", headerName))
	
	// Handle default value
	if options.DefaultValue != "" {
		g.writeLine(fmt.Sprintf("if headerValue == \"\" {"))
		g.indent()
		g.writeLine(fmt.Sprintf("headerValue = \"%s\"", options.DefaultValue))
		g.unindent()
		g.writeLine("}")
	}
	
	// Handle required validation
	if options.Required && options.DefaultValue == "" {
		g.writeLine("if headerValue == \"\" {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Header '%s' is required\"})", headerName))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
	}
	
	// Apply transformations before type conversion
	if options.Transform != "" {
		switch options.Transform {
		case "lowercase":
			g.writeLine("headerValue = strings.ToLower(headerValue)")
		case "uppercase":
			g.writeLine("headerValue = strings.ToUpper(headerValue)")
		case "trim":
			g.writeLine("headerValue = strings.TrimSpace(headerValue)")
		}
	}

	// Handle type conversion based on parameter type and options
	g.generateHeaderTypeConversion(param, "headerValue", options)
	
	g.writeLine("")
}

// generateRequestParameterExtraction generates request parameter extraction code
func (g *CodeGenerator) generateRequestParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// For @Req(), we simply assign the HTTP context's underlying request
	// The parameter should be of type *http.Request or *httpPackage.RequestContext
	paramType := strings.ToLower(param.Type)
	
	if strings.Contains(paramType, "requestcontext") || strings.Contains(paramType, "*requestcontext") {
		// Assign the context directly
		g.writeLine(fmt.Sprintf("%s := ctx", param.Name))
	} else if strings.Contains(paramType, "request") || strings.Contains(paramType, "*request") {
		// Extract the underlying HTTP request from the context
		g.writeLine(fmt.Sprintf("%s := ctx.GetRequest()", param.Name))
	} else {
		// Default to request context if type is unclear
		g.writeLine(fmt.Sprintf("%s := ctx", param.Name))
	}
	
	g.writeLine("")
}

// generateResponseParameterExtraction generates response parameter extraction code
func (g *CodeGenerator) generateResponseParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// For @Res(), we provide access to the HTTP response writer and context
	// The parameter should be of type *http.ResponseWriter, *httpPackage.ResponseContext, or *httpPackage.RequestContext
	paramType := strings.ToLower(param.Type)
	
	if strings.Contains(paramType, "responsewriter") || strings.Contains(paramType, "*responsewriter") {
		// Assign the response writer from context
		g.writeLine(fmt.Sprintf("%s := ctx.GetResponseWriter()", param.Name))
	} else if strings.Contains(paramType, "responsecontext") || strings.Contains(paramType, "*responsecontext") {
		// For response context, provide the context directly since it handles response operations
		g.writeLine(fmt.Sprintf("%s := ctx", param.Name))
	} else if strings.Contains(paramType, "requestcontext") || strings.Contains(paramType, "*requestcontext") {
		// Also allow RequestContext since it provides response methods
		g.writeLine(fmt.Sprintf("%s := ctx", param.Name))
	} else {
		// Default to request context which has response capabilities
		g.writeLine(fmt.Sprintf("%s := ctx", param.Name))
	}
	
	g.writeLine("")
}

// generateSessionParameterExtraction generates session parameter extraction code
func (g *CodeGenerator) generateSessionParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// For @Session(), we provide access to session data through the context
	sessionKey := g.getDecoratorArgValue(decorator, 0)
	
	if sessionKey != "" {
		// Extract specific session value by key
		g.writeLine(fmt.Sprintf("var %s %s", param.Name, param.Type))
		g.writeLine(fmt.Sprintf("if sessionValue := ctx.GetSession(\"%s\"); sessionValue != nil {", sessionKey))
		g.indent()
		
		// Generate type-specific extraction logic
		switch param.Type {
		case "int":
			// First try direct int cast, then fallback to string conversion
			g.writeLine("if intValue, ok := sessionValue.(int); ok {")
			g.indent()
			g.writeLine(fmt.Sprintf("%s = intValue", param.Name))
			g.unindent()
			g.writeLine("} else if strValue, ok := sessionValue.(string); ok {")
			g.indent()
			g.writeLine("if parsedInt, err := strconv.Atoi(strValue); err == nil {")
			g.indent()
			g.writeLine(fmt.Sprintf("%s = parsedInt", param.Name))
			g.unindent()
			g.writeLine("}")
			g.unindent()
			g.writeLine("}")
			g.addImport("strconv")
			
		case "bool":
			// First try direct bool cast, then fallback to string conversion
			g.writeLine("if boolValue, ok := sessionValue.(bool); ok {")
			g.indent()
			g.writeLine(fmt.Sprintf("%s = boolValue", param.Name))
			g.unindent()
			g.writeLine("} else if strValue, ok := sessionValue.(string); ok {")
			g.indent()
			g.writeLine("if parsedBool, err := strconv.ParseBool(strValue); err == nil {")
			g.indent()
			g.writeLine(fmt.Sprintf("%s = parsedBool", param.Name))
			g.unindent()
			g.writeLine("}")
			g.unindent()
			g.writeLine("}")
			g.addImport("strconv")
			
		case "string":
			// Simple string cast
			g.writeLine("if strValue, ok := sessionValue.(string); ok {")
			g.indent()
			g.writeLine(fmt.Sprintf("%s = strValue", param.Name))
			g.unindent()
			g.writeLine("}")
			
		default:
			// Custom types - use typedValue
			g.writeLine(fmt.Sprintf("if typedValue, ok := sessionValue.(%s); ok {", param.Type))
			g.indent()
			g.writeLine(fmt.Sprintf("%s = typedValue", param.Name))
			g.unindent()
			g.writeLine("}")
		}
		
		g.unindent()
		g.writeLine("}")
	} else {
		// Extract entire session object - check for special types
		if param.Type == "map[string]interface{}" || param.Type == "interface{}" {
			// For map and interface{} types, use GetAllSessionData()
			g.writeLine(fmt.Sprintf("%s := ctx.GetAllSessionData()", param.Name))
		} else {
			// For other types like *Session, use GetSessionStore()
			g.writeLine(fmt.Sprintf("%s := ctx.GetSessionStore()", param.Name))
		}
	}
	
	g.writeLine("")
}

// generateIpParameterExtraction generates IP parameter extraction code
func (g *CodeGenerator) generateIpParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// For @Ip(), we extract the client's IP address from the request
	g.writeLine(fmt.Sprintf("%s := ctx.GetClientIP()", param.Name))
	g.writeLine("")
}

// generateHostParamParameterExtraction generates host parameter extraction code
func (g *CodeGenerator) generateHostParamParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// @HostParam() extracts host information from the request
	g.writeLine(fmt.Sprintf("%s := ctx.GetHost()", param.Name))
	g.writeLine("")
}

// getQueryParameterOptions extracts options from query decorator
func (g *CodeGenerator) getQueryParameterOptions(decorator *DecoratorNode) QueryParameterOptions {
	options := QueryParameterOptions{
		Type:      "string",
		Separator: ",",
	}

	// If there's only one string argument, it's the query name
	if len(decorator.Args) == 1 {
		if _, ok := decorator.Args[0].Value.(string); ok {
			return options
		}
	}

	// Look for object argument with options
	for _, arg := range decorator.Args {
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			if defaultVal, exists := objValue["defaultValue"]; exists {
				if defaultStr, ok := defaultVal.(string); ok {
					options.DefaultValue = defaultStr
				}
			}
			if required, exists := objValue["required"]; exists {
				if reqBool, ok := required.(bool); ok {
					options.Required = reqBool
				}
			}
			if typeVal, exists := objValue["type"]; exists {
				if typeStr, ok := typeVal.(string); ok {
					options.Type = typeStr
				}
			}
			if separator, exists := objValue["separator"]; exists {
				if sepStr, ok := separator.(string); ok {
					options.Separator = sepStr
				}
			}
			if transform, exists := objValue["transform"]; exists {
				if transformStr, ok := transform.(string); ok {
					options.Transform = transformStr
				}
			}
		}
	}

	return options
}

// getHeaderParameterOptions extracts options from header decorator
func (g *CodeGenerator) getHeaderParameterOptions(decorator *DecoratorNode) HeaderParameterOptions {
	options := HeaderParameterOptions{
		Type: "string",
		Separator: ",", // default separator for arrays
		CaseInsensitive: true,
	}
	
	// Extract options from decorator arguments (simplified)
	for i, arg := range decorator.Args {
		if i == 1 { // Second argument might contain options
			if objValue, ok := arg.Value.(map[string]interface{}); ok {
				if defaultVal, exists := objValue["default"]; exists {
					if strVal, ok := defaultVal.(string); ok {
						options.DefaultValue = strVal
					}
				}
				if defaultVal, exists := objValue["defaultValue"]; exists {
					if strVal, ok := defaultVal.(string); ok {
						options.DefaultValue = strVal
					}
				}
				if required, exists := objValue["required"]; exists {
					if boolVal, ok := required.(bool); ok {
						options.Required = boolVal
					}
				}
				if separator, exists := objValue["separator"]; exists {
					if strVal, ok := separator.(string); ok {
						options.Separator = strVal
					}
				}
				if transform, exists := objValue["transform"]; exists {
					if strVal, ok := transform.(string); ok {
						options.Transform = strVal
					}
				}
			}
		}
	}
	
	return options
}

// getParameterConstraintOptions extracts constraint options from decorator
func (g *CodeGenerator) getParameterConstraintOptions(decorator *DecoratorNode) ParameterConstraintOptions {
	options := ParameterConstraintOptions{
		Required: false,
	}

	// Look for object argument with options
	for _, arg := range decorator.Args {
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			if required, exists := objValue["required"]; exists {
				if reqBool, ok := required.(bool); ok {
					options.Required = reqBool
				}
			}
			if transform, exists := objValue["transform"]; exists {
				if transformStr, ok := transform.(string); ok {
					options.Transform = transformStr
				}
			}
			if constraints, exists := objValue["constraints"]; exists {
				if constraintsArr, ok := constraints.([]interface{}); ok {
					for _, constraint := range constraintsArr {
						if constraintStr, ok := constraint.(string); ok {
							parsedConstraint := g.parseConstraint(constraintStr)
							options.Constraints = append(options.Constraints, parsedConstraint)
						}
					}
				}
			}
		}
	}

	return options
}

// generateQueryTypeConversion generates type conversion for query parameters
func (g *CodeGenerator) generateQueryTypeConversion(param *ParameterNode, valueVar string, options QueryParameterOptions) {
	paramType := strings.ToLower(param.Type)

	// Apply string transformations first
	if options.Transform != "" {
		switch options.Transform {
		case "lowercase":
			g.writeLine(fmt.Sprintf("%s = strings.ToLower(%s)", valueVar, valueVar))
		case "uppercase":
			g.writeLine(fmt.Sprintf("%s = strings.ToUpper(%s)", valueVar, valueVar))
		case "trim":
			g.writeLine(fmt.Sprintf("%s = strings.TrimSpace(%s)", valueVar, valueVar))
		}
	}

	// Handle different parameter types
	switch {
	case strings.Contains(paramType, "[]") || options.Type == "array":
		// Array type
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = strings.Split(%s, \"%s\")", param.Name, valueVar, options.Separator))
		// Trim whitespace from array elements
		g.writeLine(fmt.Sprintf("for i, v := range %s {", param.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("%s[i] = strings.TrimSpace(v)", param.Name))
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")

	case paramType == "int" || paramType == "int64" || paramType == "int32":
		// Integer conversion
		g.addImport("strconv")
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("if parsedInt, err := strconv.Atoi(%s); err == nil {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = parsedInt", param.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Invalid integer value for parameter '%s'\"})", param.Name))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")

	case paramType == "float64" || paramType == "float32":
		// Float conversion
		g.addImport("strconv")
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("if parsedFloat, err := strconv.ParseFloat(%s, 64); err == nil {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = parsedFloat", param.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Invalid float value for parameter '%s'\"})", param.Name))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")

	case paramType == "bool":
		// Boolean conversion
		g.addImport("strconv")
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("if parsedBool, err := strconv.ParseBool(%s); err == nil {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = parsedBool", param.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for parameter '%s' (use true/false)\"})", param.Name))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")

	default:
		// String type (default)
		g.writeLine(fmt.Sprintf("%s = %s", param.Name, valueVar))
	}
}

// generateHeaderTypeConversion generates type conversion for header parameters
func (g *CodeGenerator) generateHeaderTypeConversion(param *ParameterNode, valueVar string, options HeaderParameterOptions) {
	paramType := strings.ToLower(param.Type)
	
	switch {
	case strings.Contains(paramType, "[]") || strings.Contains(paramType, "array"):
		// Array type - split by separator and trim spaces
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = strings.Split(%s, \"%s\")", param.Name, valueVar, options.Separator))
		// Trim whitespace from array elements
		g.writeLine(fmt.Sprintf("for i, v := range %s {", param.Name))
		g.indent()
		g.writeLine(fmt.Sprintf("%s[i] = strings.TrimSpace(v)", param.Name))
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
		
	case strings.Contains(paramType, "int"):
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("if parsedInt, err := strconv.Atoi(%s); err == nil {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = parsedInt", param.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Invalid integer value for header '%s'\"})", param.Name))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
		
	case strings.Contains(paramType, "float"):
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("if parsedFloat, err := strconv.ParseFloat(%s, 64); err == nil {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = parsedFloat", param.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Invalid float value for header '%s'\"})", param.Name))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
		
	case strings.Contains(paramType, "bool"):
		g.writeLine(fmt.Sprintf("if %s != \"\" {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("if parsedBool, err := strconv.ParseBool(%s); err == nil {", valueVar))
		g.indent()
		g.writeLine(fmt.Sprintf("%s = parsedBool", param.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for header '%s' (use true/false)\"})", param.Name))
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
		
	default:
		// String type or others
		g.writeLine(fmt.Sprintf("%s = %s", param.Name, valueVar))
	}
}

// generateParameterConstraintValidation generates validation code for parameter constraints
func (g *CodeGenerator) generateParameterConstraintValidation(param *ParameterNode, valueVar string, options ParameterConstraintOptions) {
	// Apply string transformations first
	if options.Transform != "" {
		switch options.Transform {
		case "lowercase":
			g.writeLine(fmt.Sprintf("%s = strings.ToLower(%s)", valueVar, valueVar))
		case "uppercase":
			g.writeLine(fmt.Sprintf("%s = strings.ToUpper(%s)", valueVar, valueVar))
		case "trim":
			g.writeLine(fmt.Sprintf("%s = strings.TrimSpace(%s)", valueVar, valueVar))
		}
	}

	// Generate validation code for each constraint
	for _, constraint := range options.Constraints {
		switch constraint.Type {
		case "int":
			g.addImport("strconv")
			g.writeLine(fmt.Sprintf("if _, err := strconv.Atoi(%s); err != nil {", valueVar))
			g.indent()
			g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be an integer\"})", param.Name))
			g.writeLine("return")
			g.unindent()
			g.writeLine("}")

		case "bool":
			g.addImport("strconv")
			g.writeLine(fmt.Sprintf("if _, err := strconv.ParseBool(%s); err != nil {", valueVar))
			g.indent()
			g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be a boolean (true/false)\"})", param.Name))
			g.writeLine("return")
			g.unindent()
			g.writeLine("}")

		case "guid":
			g.addImport("github.com/google/uuid")
			g.writeLine(fmt.Sprintf("if _, err := uuid.Parse(%s); err != nil {", valueVar))
			g.indent()
			g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be a valid GUID\"})", param.Name))
			g.writeLine("return")
			g.unindent()
			g.writeLine("}")

		case "alpha":
			g.addImport("regexp")
			g.writeLine(fmt.Sprintf("if matched, _ := regexp.MatchString(\"^[a-zA-Z]+$\", %s); !matched {", valueVar))
			g.indent()
			g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must contain only alphabetic characters\"})", param.Name))
			g.writeLine("return")
			g.unindent()
			g.writeLine("}")

		case "regex":
			if constraint.Value != "" {
				g.addImport("regexp")
				g.writeLine(fmt.Sprintf("if matched, _ := regexp.MatchString(\"%s\", %s); !matched {", constraint.Value, valueVar))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' does not match required pattern\"})", param.Name))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
			}

		case "min":
			if constraint.Value != "" {
				g.addImport("strconv")
				g.writeLine(fmt.Sprintf("if intVal, err := strconv.Atoi(%s); err == nil {", valueVar))
				g.indent()
				g.writeLine(fmt.Sprintf("if intVal < %s {", constraint.Value))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be at least %s\"})", param.Name, constraint.Value))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
				g.unindent()
				g.writeLine("}")
			}

		case "max":
			if constraint.Value != "" {
				g.addImport("strconv")
				g.writeLine(fmt.Sprintf("if intVal, err := strconv.Atoi(%s); err == nil {", valueVar))
				g.indent()
				g.writeLine(fmt.Sprintf("if intVal > %s {", constraint.Value))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be at most %s\"})", param.Name, constraint.Value))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
				g.unindent()
				g.writeLine("}")
			}

		case "range":
			if constraint.Value != "" && constraint.Value2 != "" {
				g.addImport("strconv")
				g.writeLine(fmt.Sprintf("if intVal, err := strconv.Atoi(%s); err == nil {", valueVar))
				g.indent()
				g.writeLine(fmt.Sprintf("if intVal < %s || intVal > %s {", constraint.Value, constraint.Value2))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be between %s and %s\"})", param.Name, constraint.Value, constraint.Value2))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
				g.unindent()
				g.writeLine("}")
			}

		case "length":
			if constraint.Value != "" {
				g.writeLine(fmt.Sprintf("if len(%s) != %s {", valueVar, constraint.Value))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be exactly %s characters long\"})", param.Name, constraint.Value))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
			}

		case "minlength":
			if constraint.Value != "" {
				g.writeLine(fmt.Sprintf("if len(%s) < %s {", valueVar, constraint.Value))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be at least %s characters long\"})", param.Name, constraint.Value))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
			}

		case "maxlength":
			if constraint.Value != "" {
				g.writeLine(fmt.Sprintf("if len(%s) > %s {", valueVar, constraint.Value))
				g.indent()
				g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' must be at most %s characters long\"})", param.Name, constraint.Value))
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
			}
		}
	}
}