package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

// generateCatchHandlers generates error handler methods for catch decorators
func (g *CodeGenerator) generateCatchHandlers(controller *ControllerDeclaration) {
	// Generate handlers for controller-level @Catch() decorators
	for _, decorator := range controller.Decorators {
		if decorator.Name == "Catch" {
			g.generateCatchHandler(controller, decorator, "controller")
		}
	}

	// Generate handlers for method-level @Catch() decorators
	for _, method := range controller.Methods {
		for _, decorator := range method.Decorators {
			if decorator.Name == "Catch" {
				g.generateCatchHandler(controller, decorator, "method", method.Name)
			}
		}
	}
}

// generateCatchHandler generates a specific catch handler method
func (g *CodeGenerator) generateCatchHandler(controller *ControllerDeclaration, decorator *DecoratorNode, scope string, methodName ...string) {
	config := g.getCatchFilterConfig(decorator, scope)
	
	// Generate handler method name
	var handlerName string
	if scope == "method" && len(methodName) > 0 {
		handlerName = fmt.Sprintf("handle%sError", strings.Title(methodName[0]))
	} else {
		if len(config.ErrorTypes) > 0 {
			handlerName = fmt.Sprintf("handle%sError", strings.Title(config.ErrorTypes[0]))
		} else {
			handlerName = "handleError"
		}
	}

	// Generate handler method signature
	g.writeLine(fmt.Sprintf("func (c *%s) %s(err error, ctx *httpPackage.RequestContext) {", controller.Name, handlerName))
	g.indent()

	// Generate error type checking and handling
	if len(config.ErrorTypes) > 0 {
		g.writeLine("switch e := err.(type) {")
		for _, errorType := range config.ErrorTypes {
			g.writeLine(fmt.Sprintf("case *%s:", errorType))
			g.indent()
			g.generateErrorHandlingCode(errorType)
			g.unindent()
		}
		g.writeLine("default:")
		g.indent()
		g.writeLine("// Handle unmatched error types")
		g.writeLine("ctx.JSON(500, map[string]string{\"error\": \"Internal server error\"})")
		g.unindent()
		g.writeLine("}")
	} else {
		// Global error handler (catches all errors)
		g.writeLine("// Global error handler - catches all errors")
		g.generateErrorHandlingCode("error")
	}

	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateErrorHandlingCode generates error handling code based on error type
func (g *CodeGenerator) generateErrorHandlingCode(errorType string) {
	switch strings.ToLower(errorType) {
	case "badrequesterror", "*badrequesterror":
		g.writeLine("ctx.JSON(400, map[string]string{\"error\": e.Error()})")
	case "unauthorizederror", "*unauthorizederror":
		g.writeLine("ctx.JSON(401, map[string]string{\"error\": \"Unauthorized\"})")
	case "forbiddenerror", "*forbiddenerror":
		g.writeLine("ctx.JSON(403, map[string]string{\"error\": \"Forbidden\"})")
	case "notfounderror", "*notfounderror":
		g.writeLine("ctx.JSON(404, map[string]string{\"error\": \"Not found\"})")
	case "validationerror", "*validationerror":
		g.writeLine("ctx.JSON(422, map[string]string{\"error\": \"Validation failed\", \"details\": e.Error()})")
	case "conflicterror", "*conflicterror":
		g.writeLine("ctx.JSON(409, map[string]string{\"error\": \"Conflict\"})")
	case "internalservererror", "*internalservererror":
		g.writeLine("ctx.JSON(500, map[string]string{\"error\": \"Internal server error\"})")
	default:
		// Default error handling
		g.writeLine("ctx.JSON(500, map[string]string{\"error\": err.Error()})")
	}
}

// getCatchFilterConfig extracts configuration from catch decorator
func (g *CodeGenerator) getCatchFilterConfig(decorator *DecoratorNode, scope string) CatchFilterConfig {
	config := CatchFilterConfig{
		ErrorTypes: []string{},
		Scope:      scope,
	}

	// Extract error types from decorator arguments
	for _, arg := range decorator.Args {
		if errorType, ok := arg.Value.(string); ok {
			config.ErrorTypes = append(config.ErrorTypes, errorType)
		}
	}

	return config
}

// generateHttpStatusCode generates status code setting
func (g *CodeGenerator) generateHttpStatusCode(method *MethodNode) {
	statusCode := g.getHttpStatusCode(method)
	if statusCode > 0 {
		g.writeLine(fmt.Sprintf("ctx.Status(%d)", statusCode))
		g.writeLine("")
	}
}

// getHttpStatusCode extracts status code from HttpCode decorator
func (g *CodeGenerator) getHttpStatusCode(method *MethodNode) int {
	for _, decorator := range method.Decorators {
		if decorator.Name == "HttpCode" && len(decorator.Args) > 0 {
			if statusCode, ok := decorator.Args[0].Value.(int); ok {
				return statusCode
			}
			// Try to parse as string if it's not already an int
			if statusCodeStr, ok := decorator.Args[0].Value.(string); ok {
				if parsedCode, err := strconv.Atoi(statusCodeStr); err == nil {
					return parsedCode
				}
			}
		}
	}
	return 0 // No @HttpCode decorator found
}

// hasRedirectDecorator checks if method has redirect decorator
func (g *CodeGenerator) hasRedirectDecorator(method *MethodNode) bool {
	for _, decorator := range method.Decorators {
		if decorator.Name == "Redirect" {
			return true
		}
	}
	return false
}

// generateRedirectResponse generates redirect response code
func (g *CodeGenerator) generateRedirectResponse(method *MethodNode) {
	for _, decorator := range method.Decorators {
		if decorator.Name == "Redirect" {
			redirectURL := ""
			statusCode := 302 // Default redirect status code

			// Extract URL (first argument)
			if len(decorator.Args) > 0 {
				if url, ok := decorator.Args[0].Value.(string); ok {
					redirectURL = url
				}
			}

			// Extract status code (second argument, optional)
			if len(decorator.Args) > 1 {
				if code, ok := decorator.Args[1].Value.(int); ok {
					statusCode = code
				} else if codeStr, ok := decorator.Args[1].Value.(string); ok {
					if parsedCode, err := strconv.Atoi(codeStr); err == nil {
						statusCode = parsedCode
					}
				}
			}

			// Generate parameter extraction first (in case redirect URL needs to be dynamic)
			g.generateParameterExtraction(method)

			// Generate redirect call
			if redirectURL == "" {
				g.writeLine("// TODO: Set redirect URL")
				g.writeLine(fmt.Sprintf("ctx.Redirect(%d, \"\")", statusCode))
			} else {
				g.writeLine(fmt.Sprintf("ctx.Redirect(%d, \"%s\")", statusCode, redirectURL))
			}
			return
		}
	}
}

// generateResponseHeaders generates response header setting code
func (g *CodeGenerator) generateResponseHeaders(method *MethodNode) {
	headerDecorators := g.getHeaderDecorators(method)
	if len(headerDecorators) == 0 {
		return
	}

	for _, decorator := range headerDecorators {
		headerName := ""
		headerValue := ""

		// Extract header name (first argument)
		if len(decorator.Args) > 0 {
			if name, ok := decorator.Args[0].Value.(string); ok {
				headerName = name
			}
		}

		// Extract header value (second argument)
		if len(decorator.Args) > 1 {
			if value, ok := decorator.Args[1].Value.(string); ok {
				headerValue = value
			}
		}

		// Generate header setting code
		if headerName != "" && headerValue != "" {
			g.writeLine(fmt.Sprintf("ctx.Header(\"%s\", \"%s\")", headerName, headerValue))
		}
	}

	// Add empty line after headers if any were generated
	if len(headerDecorators) > 0 {
		g.writeLine("")
	}
}

// getHeaderDecorators extracts header decorators from method
func (g *CodeGenerator) getHeaderDecorators(method *MethodNode) []*DecoratorNode {
	var headerDecorators []*DecoratorNode
	for _, decorator := range method.Decorators {
		if decorator.Name == "Header" {
			headerDecorators = append(headerDecorators, decorator)
		}
	}
	return headerDecorators
}

// getGuardDecorators extracts guard decorator names
func (g *CodeGenerator) getGuardDecorators(decorators []*DecoratorNode) []string {
	var guards []string
	
	for _, decorator := range decorators {
		if decorator.Name == "UseGuards" {
			// Extract guard names from decorator arguments
			for _, arg := range decorator.Args {
				if guardName, ok := arg.Value.(string); ok {
					guards = append(guards, guardName)
				}
			}
		}
	}
	
	return guards
}

// generateGuardMiddlewareFunctions generates guard middleware methods
func (g *CodeGenerator) generateGuardMiddlewareFunctions(controller *ControllerDeclaration) {
	allGuards := make(map[string]bool)
	
	// Collect all unique guards from controller and methods
	controllerGuards := g.getGuardDecorators(controller.Decorators)
	for _, guard := range controllerGuards {
		allGuards[guard] = true
	}
	
	for _, method := range controller.Methods {
		methodGuards := g.getGuardDecorators(method.Decorators)
		for _, guard := range methodGuards {
			allGuards[guard] = true
		}
	}
	
	if len(allGuards) == 0 {
		return
	}
	
	// Generate guard middleware methods for the controller
	g.writeLine("// Guard middleware methods")
	
	for guard := range allGuards {
		g.generateControllerGuardMethod(controller.Name, guard)
		g.writeLine("")
	}
}

// generateAuthGuardLogic generates authentication guard logic
func (g *CodeGenerator) generateAuthGuardLogic() {
	g.writeLine("// Authentication guard logic")
	g.writeLine("token := ctx.GetHeader(\"Authorization\")")
	g.writeLine("if token == \"\" {")
	g.indent()
	g.writeLine("ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: Missing authentication token\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Validate token (implement your token validation logic)")
	g.writeLine("if !isValidToken(token) {")
	g.indent()
	g.writeLine("ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: Invalid token\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateRoleGuardLogic generates role-based authorization guard logic
func (g *CodeGenerator) generateRoleGuardLogic() {
	g.writeLine("// Role-based authorization guard logic")
	g.writeLine("user := ctx.GetUser() // Implement GetUser() to extract user from context")
	g.writeLine("if user == nil {")
	g.indent()
	g.writeLine("ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: User not authenticated\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check user role (implement your role checking logic)")
	g.writeLine("if !hasRequiredRole(user, \"required_role\") {")
	g.indent()
	g.writeLine("ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Insufficient permissions\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generatePermissionGuardLogic generates permission-based authorization guard logic
func (g *CodeGenerator) generatePermissionGuardLogic() {
	g.writeLine("// Permission-based authorization guard logic")
	g.writeLine("user := ctx.GetUser() // Implement GetUser() to extract user from context")
	g.writeLine("if user == nil {")
	g.indent()
	g.writeLine("ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: User not authenticated\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check user permissions (implement your permission checking logic)")
	g.writeLine("if !hasPermission(user, \"required_permission\") {")
	g.indent()
	g.writeLine("ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Insufficient permissions\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateGenericGuardLogic generates generic guard logic
func (g *CodeGenerator) generateGenericGuardLogic(guardName string) {
	g.writeLine(fmt.Sprintf("// %s guard logic", guardName))
	g.writeLine("// TODO: Implement your custom guard logic here")
	g.writeLine("if !checkGuardCondition() {")
	g.indent()
	g.writeLine("ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Guard check failed\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// getInterceptorDecorators extracts interceptor decorator names
func (g *CodeGenerator) getInterceptorDecorators(decorators []*DecoratorNode) []string {
	var interceptors []string
	
	for _, decorator := range decorators {
		if decorator.Name == "UseInterceptors" {
			// Extract interceptor names from decorator arguments
			for _, arg := range decorator.Args {
				if interceptorName, ok := arg.Value.(string); ok {
					interceptors = append(interceptors, interceptorName)
				}
			}
		}
	}
	
	return interceptors
}

// generateInterceptorMiddlewareFunctions generates interceptor middleware methods
func (g *CodeGenerator) generateInterceptorMiddlewareFunctions(controller *ControllerDeclaration) {
	allInterceptors := make(map[string]bool)
	
	// Collect all unique interceptors from controller and methods
	controllerInterceptors := g.getInterceptorDecorators(controller.Decorators)
	for _, interceptor := range controllerInterceptors {
		allInterceptors[interceptor] = true
	}
	
	for _, method := range controller.Methods {
		methodInterceptors := g.getInterceptorDecorators(method.Decorators)
		for _, interceptor := range methodInterceptors {
			allInterceptors[interceptor] = true
		}
	}
	
	if len(allInterceptors) == 0 {
		return
	}
	
	// Generate interceptor middleware methods for the controller
	g.writeLine("// Interceptor middleware methods")
	
	for interceptor := range allInterceptors {
		g.generateControllerInterceptorMethod(controller.Name, interceptor)
		g.writeLine("")
	}
}

// generateControllerInterceptorMethod generates an interceptor method for the controller
func (g *CodeGenerator) generateControllerInterceptorMethod(controllerName, interceptorName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s interceptor middleware", interceptorName, interceptorName))
	g.writeLine(fmt.Sprintf("func (c *%s) %s(ctx *httpPackage.RequestContext) {", controllerName, interceptorName))
	g.indent()
	
	// Generate interceptor logic based on interceptor name
	switch interceptorName {
	case "LoggingInterceptor":
		g.generateLoggingInterceptorLogic()
	case "CacheInterceptor":
		g.generateCacheInterceptorLogic()
	case "TransformInterceptor":
		g.generateTransformInterceptorLogic()
	case "ValidationInterceptor":
		g.generateValidationInterceptorLogic()
	default:
		g.generateGenericInterceptorLogic(interceptorName)
	}
	
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
	g.unindent()
	g.writeLine("}")
}

// generateLoggingInterceptorLogic generates logging interceptor logic
func (g *CodeGenerator) generateLoggingInterceptorLogic() {
	g.writeLine("// Logging interceptor logic")
	g.writeLine("startTime := time.Now()")
	g.writeLine("method := ctx.GetMethod()")
	g.writeLine("path := ctx.GetPath()")
	g.writeLine("clientIP := ctx.GetClientIP()")
	g.writeLine("")
	g.writeLine("// Log incoming request")
	g.writeLine("fmt.Printf(\"[%s] %s %s from %s\\n\", startTime.Format(time.RFC3339), method, path, clientIP)")
	g.writeLine("")
}

// generateCacheInterceptorLogic generates cache interceptor logic
func (g *CodeGenerator) generateCacheInterceptorLogic() {
	g.writeLine("// Cache interceptor logic")
	g.writeLine("// TODO: Implement cache logic")
}

// generateTransformInterceptorLogic generates transform interceptor logic
func (g *CodeGenerator) generateTransformInterceptorLogic() {
	g.writeLine("// Transform interceptor logic")
	g.writeLine("// TODO: Implement transform logic")
}

// generateValidationInterceptorLogic generates validation interceptor logic
func (g *CodeGenerator) generateValidationInterceptorLogic() {
	g.writeLine("// Validation interceptor logic")
	g.writeLine("// TODO: Implement validation logic")
}

// generateGenericInterceptorLogic generates generic interceptor logic
func (g *CodeGenerator) generateGenericInterceptorLogic(interceptorName string) {
	g.writeLine(fmt.Sprintf("// %s interceptor logic", interceptorName))
	g.writeLine("// TODO: Implement your custom interceptor logic here")
	g.writeLine("")
}

// getPipeDecorators extracts pipe decorator names
func (g *CodeGenerator) getPipeDecorators(decorators []*DecoratorNode) []string {
	var pipes []string
	
	for _, decorator := range decorators {
		if decorator.Name == "UsePipes" {
			// Extract pipe names from decorator arguments
			for _, arg := range decorator.Args {
				if pipeName, ok := arg.Value.(string); ok {
					pipes = append(pipes, pipeName)
				}
			}
		}
	}
	
	return pipes
}

// generatePipeMiddlewareFunctions generates pipe middleware methods
func (g *CodeGenerator) generatePipeMiddlewareFunctions(controller *ControllerDeclaration) {
	allPipes := make(map[string]bool)
	
	// Collect all unique pipes from controller and methods
	controllerPipes := g.getPipeDecorators(controller.Decorators)
	for _, pipe := range controllerPipes {
		allPipes[pipe] = true
	}
	
	for _, method := range controller.Methods {
		methodPipes := g.getPipeDecorators(method.Decorators)
		for _, pipe := range methodPipes {
			allPipes[pipe] = true
		}
	}
	
	if len(allPipes) == 0 {
		return
	}
	
	// Generate pipe middleware methods for the controller
	g.writeLine("// Pipe middleware methods")
	
	for pipe := range allPipes {
		g.generateControllerPipeMethod(controller.Name, pipe)
		g.writeLine("")
	}
}

// generateControllerPipeMethod generates a pipe method for the controller
func (g *CodeGenerator) generateControllerPipeMethod(controllerName, pipeName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s pipe middleware", pipeName, pipeName))
	g.writeLine(fmt.Sprintf("func (c *%s) %s(ctx *httpPackage.RequestContext) {", controllerName, pipeName))
	g.indent()
	
	// Generate pipe logic based on pipe name
	switch pipeName {
	case "ValidationPipe":
		g.generateValidationPipeLogic()
	case "TransformPipe":
		g.generateTransformPipeLogic()
	default:
		g.generateGenericPipeLogic(pipeName)
	}
	
	g.unindent()
	g.writeLine("}")
}

// generateValidationPipeLogic generates validation pipe logic
func (g *CodeGenerator) generateValidationPipeLogic() {
	g.writeLine("// Validation pipe logic")
	g.writeLine("// TODO: Implement validation logic")
	g.writeLine("ctx.Next()")
}

// generateTransformPipeLogic generates transform pipe logic  
func (g *CodeGenerator) generateTransformPipeLogic() {
	g.writeLine("// Transform pipe logic")
	g.writeLine("// TODO: Implement transformation logic")
	g.writeLine("ctx.Next()")
}

// generateGenericPipeLogic generates generic pipe logic
func (g *CodeGenerator) generateGenericPipeLogic(pipeName string) {
	g.writeLine(fmt.Sprintf("// %s pipe logic", pipeName))
	g.writeLine("// TODO: Implement your custom pipe logic here")
	g.writeLine("ctx.Next()")
}