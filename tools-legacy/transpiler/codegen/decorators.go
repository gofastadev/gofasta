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

// generateWebSocketCatchHandlers generates WebSocket error handler methods for catch decorators
func (g *CodeGenerator) generateWebSocketCatchHandlers(gateway *WebSocketGatewayDeclaration) {
	// Generate handlers for gateway-level @Catch() decorators
	for _, decorator := range gateway.Decorators {
		if decorator.Name == "Catch" {
			g.generateWebSocketCatchHandler(gateway, decorator, "gateway")
		}
	}

	// Generate handlers for method-level @Catch() decorators
	for _, method := range gateway.Methods {
		for _, decorator := range method.Decorators {
			if decorator.Name == "Catch" {
				g.generateWebSocketCatchHandler(gateway, decorator, "method", method.Name)
			}
		}
	}

	// Generate handlers for standalone WebSocket function @Catch() decorators
	for _, wsFunc := range g.webSocketFunctions {
		for _, decorator := range wsFunc.Decorators {
			if decorator.Name == "Catch" {
				g.generateWebSocketFunctionCatchHandler(gateway, wsFunc, decorator)
			}
		}
	}
}

// generateWebSocketFunctionCatchHandler generates WebSocket error handler for standalone functions
func (g *CodeGenerator) generateWebSocketFunctionCatchHandler(gateway *WebSocketGatewayDeclaration, wsFunc *WebSocketFunctionDeclaration, decorator *DecoratorNode) {
	config := g.getCatchFilterConfig(decorator, "method")
	
	// Generate handler method name based on the function name
	handlerName := fmt.Sprintf("handle%sError", wsFunc.Name)

	// Generate handler method signature with WebSocket-specific parameters
	g.writeLine(fmt.Sprintf("func (g *%s) %s(", gateway.Name, handlerName))
	g.indent()
	g.writeLine("err error,")
	g.writeLine("client *WebSocketClient,")
	g.writeLine("eventName string")
	g.unindent()
	g.writeLine(") {")
	g.indent()

	// Generate error type checking and WebSocket error handling
	if len(config.ErrorTypes) > 0 {
		g.writeLine("switch e := err.(type) {")
		for _, errorType := range config.ErrorTypes {
			g.writeLine(fmt.Sprintf("case *%s:", errorType))
			g.indent()
			g.generateWebSocketErrorHandlingCode(errorType)
			g.unindent()
		}
		g.writeLine("default:")
		g.indent()
		g.writeLine("// Handle unmatched error types")
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"InternalServerError\",")
		g.writeLine("\"message\": \"Internal server error\",")
		g.writeLine("\"details\": err.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
		g.unindent()
		g.writeLine("}")
	} else {
		// Global error handler (catches all errors)
		g.writeLine("// Global WebSocket error handler - catches all errors")
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"InternalServerError\",")
		g.writeLine("\"message\": err.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	}

	g.unindent()
	g.writeLine("}")
	g.writeLine("")
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

// generateWebSocketCatchHandler generates a specific WebSocket catch handler method
func (g *CodeGenerator) generateWebSocketCatchHandler(gateway *WebSocketGatewayDeclaration, decorator *DecoratorNode, scope string, methodName ...string) {
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

	// Generate WebSocket handler method signature with @Exception() and @ConnectedSocket() parameters
	g.writeLine(fmt.Sprintf("func (g *%s) %s(", gateway.Name, handlerName))
	g.indent()
	
	// Add @Exception() parameter
	g.writeLine("err error,")
	
	// Add @ConnectedSocket() parameter  
	g.writeLine("client *WebSocketClient,")
	
	// Add @EventName() parameter if available
	g.writeLine("eventName string")
	
	g.unindent()
	g.writeLine(") {")
	g.indent()

	// Generate error type checking and handling for WebSocket
	if len(config.ErrorTypes) > 0 {
		g.writeLine("switch e := err.(type) {")
		for _, errorType := range config.ErrorTypes {
			g.writeLine(fmt.Sprintf("case *%s:", errorType))
			g.indent()
			g.generateWebSocketErrorHandlingCode(errorType)
			g.unindent()
		}
		g.writeLine("default:")
		g.indent()
		g.writeLine("// Handle unmatched error types")
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"InternalServerError\",")
		g.writeLine("\"message\": \"Internal server error\",")
		g.writeLine("\"details\": err.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
		g.unindent()
		g.writeLine("}")
	} else {
		// Global error handler (catches all errors)
		g.writeLine("// Global WebSocket error handler - catches all errors")
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"InternalServerError\",")
		g.writeLine("\"message\": err.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
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

// generateWebSocketErrorHandlingCode generates WebSocket error handling code based on error type
func (g *CodeGenerator) generateWebSocketErrorHandlingCode(errorType string) {
	switch strings.ToLower(errorType) {
	case "badrequesterror", "*badrequesterror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"BadRequestError\",")
		g.writeLine("\"message\": e.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "unauthorizederror", "*unauthorizederror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"UnauthorizedError\",")
		g.writeLine("\"message\": \"Unauthorized\",")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "forbiddenerror", "*forbiddenerror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"ForbiddenError\",")
		g.writeLine("\"message\": \"Forbidden\",")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "notfounderror", "*notfounderror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"NotFoundError\",")
		g.writeLine("\"message\": \"Not found\",")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "validationerror", "*validationerror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"ValidationError\",")
		g.writeLine("\"message\": \"Validation failed\",")
		g.writeLine("\"details\": e.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "conflicterror", "*conflicterror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"ConflictError\",")
		g.writeLine("\"message\": \"Conflict\",")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "internalservererror", "*internalservererror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"InternalServerError\",")
		g.writeLine("\"message\": \"Internal server error\",")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	case "authenticationerror", "*authenticationerror":
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine("\"type\": \"AuthenticationError\",")
		g.writeLine("\"message\": \"Authentication failed\",")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
	default:
		// Default WebSocket error handling - use the actual error type name
		errorTypeName := strings.TrimPrefix(errorType, "*")
		g.writeLine("client.Emit(\"error\", map[string]interface{}{")
		g.indent()
		g.writeLine(fmt.Sprintf("\"type\": \"%s\",", errorTypeName))
		g.writeLine("\"message\": e.Error(),")
		g.writeLine("\"event\": eventName,")
		g.unindent()
		g.writeLine("})")
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
	g.writeLine("// Store start time for later use")
	g.writeLine("ctx.Set(\"request_start_time\", startTime)")
	g.writeLine("")
	g.writeLine("// Setup completion logging")
	g.writeLine("ctx.OnFinished(func() {")
	g.indent()
	g.writeLine("duration := time.Since(startTime)")
	g.writeLine("status := ctx.GetStatusCode()")
	g.writeLine("fmt.Printf(\"[%s] %s %s %d - %v\\n\", time.Now().Format(time.RFC3339), method, path, status, duration)")
	g.unindent()
	g.writeLine("})")
}

// generateCacheInterceptorLogic generates cache interceptor logic
func (g *CodeGenerator) generateCacheInterceptorLogic() {
	g.writeLine("// Cache interceptor logic")
	g.writeLine("cacheKey := fmt.Sprintf(\"%s:%s\", ctx.GetMethod(), ctx.GetPath())")
	g.writeLine("if cachedResponse := getFromCache(cacheKey); cachedResponse != nil {")
	g.indent()
	g.writeLine("ctx.JSON(200, cachedResponse)")
	g.writeLine("return // Skip further processing")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Setup caching for response")
	g.writeLine("ctx.OnFinished(func() {")
	g.indent()
	g.writeLine("if ctx.GetStatusCode() == 200 {")
	g.indent()
	g.writeLine("responseData := ctx.GetResponseData()")
	g.writeLine("setCache(cacheKey, responseData, 300) // 5 minute cache")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("})")
}

// generateTransformInterceptorLogic generates transform interceptor logic
func (g *CodeGenerator) generateTransformInterceptorLogic() {
	g.writeLine("// Transform interceptor logic")
	g.writeLine("if requestBody := ctx.GetRequestBody(); requestBody != nil {")
	g.indent()
	g.writeLine("transformedBody := transformRequestData(requestBody)")
	g.writeLine("ctx.SetRequestBody(transformedBody)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Setup response transformation")
	g.writeLine("ctx.OnBeforeResponse(func() {")
	g.indent()
	g.writeLine("if responseData := ctx.GetResponseData(); responseData != nil {")
	g.indent()
	g.writeLine("transformedResponse := transformResponseData(responseData)")
	g.writeLine("ctx.SetResponseData(transformedResponse)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("})")
}

// generateValidationInterceptorLogic generates validation interceptor logic
func (g *CodeGenerator) generateValidationInterceptorLogic() {
	g.writeLine("// Validation interceptor logic")
	g.writeLine("if contentType := ctx.GetHeader(\"Content-Type\"); contentType == \"\" {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Content-Type header is required\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Validate request body if present")
	g.writeLine("if requestBody := ctx.GetRequestBody(); requestBody != nil {")
	g.indent()
	g.writeLine("if !isValidRequestStructure(requestBody) {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Invalid request body structure\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("if err := validateRequestData(requestBody); err != nil {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Request validation failed\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateGenericInterceptorLogic generates generic interceptor logic
func (g *CodeGenerator) generateGenericInterceptorLogic(interceptorName string) {
	g.writeLine(fmt.Sprintf("// %s interceptor logic", interceptorName))
	g.writeLine("// TODO: Implement your custom interceptor logic here")
	g.writeLine("")
	g.writeLine("// Example: Pre-processing")
	g.writeLine("// requestData := ctx.GetRequestData()")
	g.writeLine("// if !validateRequest(requestData) {")
	g.writeLine("//     ctx.JSON(400, map[string]string{\"error\": \"Request validation failed\"})")
	g.writeLine("//     ctx.Abort()")
	g.writeLine("//     return")
	g.writeLine("// }")
	g.writeLine("")
	g.writeLine("// Example: Post-processing")
	g.writeLine("// ctx.OnFinished(func() {")
	g.writeLine("//     // Custom cleanup or logging")
	g.writeLine("// })")
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
	case "ParseIntPipe":
		g.generateParseIntPipeLogic()
	case "ParseBoolPipe":
		g.generateParseBoolPipeLogic()
	case "ParseArrayPipe":
		g.generateParseArrayPipeLogic()
	case "DefaultValuePipe":
		g.generateDefaultValuePipeLogic()
	default:
		g.generateGenericPipeLogic(pipeName)
	}
	
	g.unindent()
	g.writeLine("}")
}

// generateValidationPipeLogic generates validation pipe logic
func (g *CodeGenerator) generateValidationPipeLogic() {
	g.writeLine("// Validation pipe logic")
	g.writeLine("requestBody := ctx.GetRequestBody()")
	g.writeLine("if requestBody == nil {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Request body is required\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("contentType := ctx.GetHeader(\"Content-Type\")")
	g.writeLine("if !strings.Contains(contentType, \"application/json\") {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Content-Type must be application/json\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("if err := validateRequestBody(requestBody); err != nil {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Validation failed: %s\", err.Error())})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("ctx.Next()")
}

// generateTransformPipeLogic generates transform pipe logic  
func (g *CodeGenerator) generateTransformPipeLogic() {
	g.writeLine("// Transform pipe logic")
	g.writeLine("requestData := ctx.GetRequestData()")
	g.writeLine("if requestData != nil {")
	g.indent()
	g.writeLine("transformedData := transformRequestData(requestData)")
	g.writeLine("ctx.SetRequestData(transformedData)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Transform query parameters")
	g.writeLine("for key, value := range ctx.GetAllQueryParams() {")
	g.indent()
	g.writeLine("if strValue, ok := value.(string); ok {")
	g.indent()
	g.writeLine("transformedValue := transformQueryValue(key, value)")
	g.writeLine("ctx.SetQueryParam(key, transformedValue)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("ctx.Next()")
}

// generateParseIntPipeLogic generates ParseInt pipe logic
func (g *CodeGenerator) generateParseIntPipeLogic() {
	g.writeLine("// ParseInt pipe logic")
	g.writeLine("pathParams := ctx.GetPathParams()")
	g.writeLine("for key, value := range pathParams {")
	g.indent()
	g.writeLine("if strValue, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if intValue, err := strconv.Atoi(strValue); err == nil {")
	g.indent()
	g.writeLine("ctx.SetPathParam(key, intValue)")
	g.unindent()
	g.writeLine("} else {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Invalid integer value for parameter %s: %s\", key, strValue)})")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Parse query parameters")
	g.writeLine("queryParams := ctx.GetQueryParams()")
	g.writeLine("for key, value := range queryParams {")
	g.indent()
	g.writeLine("if strValue, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if intValue, err := strconv.Atoi(strValue); err == nil {")
	g.indent()
	g.writeLine("ctx.SetQueryParam(key, intValue)")
	g.unindent()
	g.writeLine("} else if strValue != \"\" {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Invalid integer value for query parameter %s: %s\", key, strValue)})")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("ctx.Next()")
}

// generateParseBoolPipeLogic generates ParseBool pipe logic
func (g *CodeGenerator) generateParseBoolPipeLogic() {
	g.writeLine("// ParseBool pipe logic")
	g.writeLine("for key, value := range ctx.GetAllQueryParams() {")
	g.indent()
	g.writeLine("if strValue, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if boolValue, err := strconv.ParseBool(strValue); err == nil {")
	g.indent()
	g.writeLine("ctx.SetQueryParam(key, boolValue)")
	g.unindent()
	g.writeLine("} else if strings.ToLower(strValue) == \"true\" || strings.ToLower(strValue) == \"false\" {")
	g.indent()
	g.writeLine("ctx.SetQueryParam(key, strings.ToLower(strValue) == \"true\")")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("ctx.Next()")
}

// generateParseArrayPipeLogic generates ParseArray pipe logic
func (g *CodeGenerator) generateParseArrayPipeLogic() {
	g.writeLine("// ParseArray pipe logic")
	g.writeLine("for key, value := range ctx.GetAllQueryParams() {")
	g.indent()
	g.writeLine("if strValue, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if strings.Contains(strValue, \",\") {")
	g.indent()
	g.writeLine("arrayValue := strings.Split(strValue, \",\")")
	g.writeLine("for i, item := range arrayValue {")
	g.indent()
	g.writeLine("arrayValue[i] = strings.TrimSpace(item)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("ctx.SetQueryParam(key, arrayValue)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("ctx.Next()")
}

// generateDefaultValuePipeLogic generates DefaultValue pipe logic
func (g *CodeGenerator) generateDefaultValuePipeLogic() {
	g.writeLine("// DefaultValue pipe logic")
	g.writeLine("defaultValues := map[string]interface{}{")
	g.indent()
	g.writeLine("\"limit\":  10,")
	g.writeLine("\"offset\": 0,")
	g.writeLine("\"sort\":   \"created_at\",")
	g.writeLine("\"order\":  \"desc\",")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("for key, defaultValue := range defaultValues {")
	g.indent()
	g.writeLine("if ctx.GetQueryParam(key) == nil {")
	g.indent()
	g.writeLine("ctx.SetQueryParam(key, defaultValue)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("ctx.Next()")
}

// generateGenericPipeLogic generates generic pipe logic
func (g *CodeGenerator) generateGenericPipeLogic(pipeName string) {
	g.writeLine(fmt.Sprintf("// %s pipe logic", pipeName))
	g.writeLine("// TODO: Implement your custom pipe logic here")
	g.writeLine("")
	g.writeLine("// Example: Data validation")
	g.writeLine("// requestData := ctx.GetRequestData()")
	g.writeLine("// if err := validateData(requestData); err != nil {")
	g.writeLine("//     ctx.JSON(400, map[string]string{\"error\": err.Error()})")
	g.writeLine("//     return")
	g.writeLine("// }")
	g.writeLine("")
	g.writeLine("// Example: Data transformation")
	g.writeLine("// transformedData := transformData(requestData)")
	g.writeLine("// ctx.SetRequestData(transformedData)")
	g.writeLine("")
	g.writeLine("ctx.Next()")
}