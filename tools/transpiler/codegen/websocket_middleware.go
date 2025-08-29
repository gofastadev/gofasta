package codegen

import (
	"fmt"
)

// ===================== WebSocket Guard Middleware Generation =====================

// generateWebSocketGuardMethod generates a WebSocket guard method
func (g *CodeGenerator) generateWebSocketGuardMethod(gatewayName, guardName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s guard middleware for WebSocket", guardName, guardName))
	g.writeLine(fmt.Sprintf("func (ws *%s) %s(wsCtx *websocket.GuardContext) bool {", gatewayName, guardName))
	g.indent()
	
	// Generate guard logic based on guard name
	switch guardName {
	case "WSAuthGuard", "AuthGuard":
		g.generateWebSocketAuthGuardLogic()
	case "WSRoleGuard", "RoleGuard":
		g.generateWebSocketRoleGuardLogic()
	case "WSPermissionGuard", "PermissionGuard":
		g.generateWebSocketPermissionGuardLogic()
	case "WSRateLimitGuard", "RateLimitGuard":
		g.generateWebSocketRateLimitGuardLogic()
	case "WSRoomGuard", "RoomGuard":
		g.generateWebSocketRoomGuardLogic()
	default:
		g.generateWebSocketGenericGuardLogic(guardName)
	}
	
	g.writeLine("// Return true to allow access, false to deny")
	g.writeLine("return true")
	g.unindent()
	g.writeLine("}")
}

// generateWebSocketAuthGuardLogic generates WebSocket authentication guard logic
func (g *CodeGenerator) generateWebSocketAuthGuardLogic() {
	g.writeLine("// WebSocket authentication guard logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("headers := wsCtx.Headers()")
	g.writeLine("")
	g.writeLine("// Extract authentication token from headers or query params")
	g.writeLine("token := headers[\"Authorization\"]")
	g.writeLine("if token == \"\" {")
	g.indent()
	g.writeLine("token = wsCtx.Query()[\"token\"] // Fallback to query parameter")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("if token == \"\" {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Authentication required\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Validate token (implement your token validation logic)")
	g.writeLine("if !isValidToken(token) {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Invalid authentication token\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Set authenticated user in context")
	g.writeLine("user := getUserFromToken(token) // Implement this function")
	g.writeLine("wsCtx.SetUser(user)")
	g.writeLine("")
}

// generateWebSocketRoleGuardLogic generates WebSocket role-based authorization guard logic
func (g *CodeGenerator) generateWebSocketRoleGuardLogic() {
	g.writeLine("// WebSocket role-based authorization guard logic")
	g.writeLine("user := wsCtx.User()")
	g.writeLine("if user == nil {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"User not authenticated\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check user role (implement your role checking logic)")
	g.writeLine("requiredRoles := []string{\"admin\", \"moderator\"} // Define required roles")
	g.writeLine("if !hasAnyRole(user, requiredRoles) {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Insufficient role permissions\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateWebSocketPermissionGuardLogic generates WebSocket permission-based authorization guard logic
func (g *CodeGenerator) generateWebSocketPermissionGuardLogic() {
	g.writeLine("// WebSocket permission-based authorization guard logic")
	g.writeLine("user := wsCtx.User()")
	g.writeLine("if user == nil {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"User not authenticated\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check user permissions")
	g.writeLine("requiredPermissions := []string{\"websocket:connect\", \"room:join\"}")
	g.writeLine("if !hasAllPermissions(user, requiredPermissions) {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Insufficient permissions\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateWebSocketRateLimitGuardLogic generates WebSocket rate limiting guard logic
func (g *CodeGenerator) generateWebSocketRateLimitGuardLogic() {
	g.writeLine("// WebSocket rate limiting guard logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("clientIP := wsCtx.ClientIP()")
	g.writeLine("")
	g.writeLine("// Check rate limit based on client IP or user ID")
	g.writeLine("if isRateLimited(clientIP) {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Rate limit exceeded\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Update rate limit counter")
	g.writeLine("updateRateLimit(clientIP)")
	g.writeLine("")
}

// generateWebSocketRoomGuardLogic generates WebSocket room authorization guard logic
func (g *CodeGenerator) generateWebSocketRoomGuardLogic() {
	g.writeLine("// WebSocket room authorization guard logic")
	g.writeLine("user := wsCtx.User()")
	g.writeLine("roomID := wsCtx.Query()[\"room\"]")
	g.writeLine("")
	g.writeLine("if roomID == \"\" {")
	g.indent()
	g.writeLine("// Allow connection without room restriction")
	g.writeLine("return true")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check if user can access the requested room")
	g.writeLine("if !canAccessRoom(user, roomID) {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Access denied to room: \" + roomID)")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateWebSocketGenericGuardLogic generates generic WebSocket guard logic
func (g *CodeGenerator) generateWebSocketGenericGuardLogic(guardName string) {
	g.writeLine(fmt.Sprintf("// %s WebSocket guard logic", guardName))
	g.writeLine("// TODO: Implement your custom WebSocket guard logic here")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("user := wsCtx.User()")
	g.writeLine("headers := wsCtx.Headers()")
	g.writeLine("query := wsCtx.Query()")
	g.writeLine("")
	g.writeLine("// Example guard condition")
	g.writeLine("if !checkWebSocketGuardCondition(client, user, headers, query) {")
	g.indent()
	g.writeLine("wsCtx.Disconnect(\"Guard check failed\")")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// ===================== WebSocket Interceptor Middleware Generation =====================

// generateWebSocketInterceptorMethod generates a WebSocket interceptor method
func (g *CodeGenerator) generateWebSocketInterceptorMethod(gatewayName, interceptorName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s interceptor middleware for WebSocket", interceptorName, interceptorName))
	g.writeLine(fmt.Sprintf("func (ws *%s) %s(wsCtx *websocket.InterceptorContext) {", gatewayName, interceptorName))
	g.indent()
	
	// Generate interceptor logic based on interceptor name
	switch interceptorName {
	case "WSLoggingInterceptor", "LoggingInterceptor":
		g.generateWebSocketLoggingInterceptorLogic()
	case "WSTransformInterceptor", "TransformInterceptor":
		g.generateWebSocketTransformInterceptorLogic()
	case "WSValidationInterceptor", "ValidationInterceptor":
		g.generateWebSocketValidationInterceptorLogic()
	case "WSCacheInterceptor", "CacheInterceptor":
		g.generateWebSocketCacheInterceptorLogic()
	case "WSMetricsInterceptor", "MetricsInterceptor":
		g.generateWebSocketMetricsInterceptorLogic()
	default:
		g.generateWebSocketGenericInterceptorLogic(interceptorName)
	}
	
	g.writeLine("// Continue to next interceptor/handler")
	g.writeLine("wsCtx.Next()")
	g.unindent()
	g.writeLine("}")
}

// generateWebSocketLoggingInterceptorLogic generates WebSocket logging interceptor logic
func (g *CodeGenerator) generateWebSocketLoggingInterceptorLogic() {
	g.writeLine("// WebSocket logging interceptor logic")
	g.writeLine("startTime := time.Now()")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("event := wsCtx.EventName()")
	g.writeLine("user := wsCtx.User()")
	g.writeLine("")
	g.writeLine("// Log incoming WebSocket event")
	g.writeLine("logger.Info(\"WebSocket event received\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":     event,")
	g.writeLine("\"client_id\": client.ID(),")
	g.writeLine("\"user_id\":   getUserID(user),")
	g.writeLine("\"timestamp\": startTime,")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
	g.writeLine("// Execute handler and measure time")
	g.writeLine("defer func() {")
	g.indent()
	g.writeLine("duration := time.Since(startTime)")
	g.writeLine("logger.Info(\"WebSocket event processed\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":    event,")
	g.writeLine("\"duration\": duration,")
	g.unindent()
	g.writeLine("})")
	g.unindent()
	g.writeLine("}()")
	g.writeLine("")
}

// generateWebSocketTransformInterceptorLogic generates WebSocket transform interceptor logic
func (g *CodeGenerator) generateWebSocketTransformInterceptorLogic() {
	g.writeLine("// WebSocket transform interceptor logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Transform incoming message data")
	g.writeLine("transformedData := transformWebSocketMessage(messageBody)")
	g.writeLine("wsCtx.SetMessageBody(transformedData)")
	g.writeLine("")
	g.writeLine("// Add custom headers or metadata")
	g.writeLine("wsCtx.SetMetadata(\"transformed\", true)")
	g.writeLine("wsCtx.SetMetadata(\"transform_time\", time.Now())")
	g.writeLine("")
}

// generateWebSocketValidationInterceptorLogic generates WebSocket validation interceptor logic
func (g *CodeGenerator) generateWebSocketValidationInterceptorLogic() {
	g.writeLine("// WebSocket validation interceptor logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Validate message structure and content")
	g.writeLine("if err := validateWebSocketMessage(eventName, messageBody); err != nil {")
	g.indent()
	g.writeLine("// Send validation error to client")
	g.writeLine("wsCtx.Client().Emit(\"validation_error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":   eventName,")
	g.writeLine("\"error\":   err.Error(),")
	g.writeLine("\"message\": \"Message validation failed\",")
	g.unindent()
	g.writeLine("})")
	g.writeLine("wsCtx.Abort() // Stop further processing")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateWebSocketCacheInterceptorLogic generates WebSocket cache interceptor logic
func (g *CodeGenerator) generateWebSocketCacheInterceptorLogic() {
	g.writeLine("// WebSocket cache interceptor logic")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Generate cache key based on event and content")
	g.writeLine("cacheKey := generateCacheKey(eventName, messageBody)")
	g.writeLine("")
	g.writeLine("// Check if response is cached")
	g.writeLine("if cachedResponse, found := getFromCache(cacheKey); found {")
	g.indent()
	g.writeLine("// Send cached response")
	g.writeLine("wsCtx.Client().Emit(eventName+\"_response\", cachedResponse)")
	g.writeLine("wsCtx.Abort() // Skip handler execution")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Set flag to cache response after handler execution")
	g.writeLine("wsCtx.SetMetadata(\"cache_key\", cacheKey)")
	g.writeLine("")
}

// generateWebSocketMetricsInterceptorLogic generates WebSocket metrics interceptor logic
func (g *CodeGenerator) generateWebSocketMetricsInterceptorLogic() {
	g.writeLine("// WebSocket metrics interceptor logic")
	g.writeLine("startTime := time.Now()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("")
	g.writeLine("// Increment event counter")
	g.writeLine("incrementEventCounter(eventName)")
	g.writeLine("incrementActiveConnections()")
	g.writeLine("")
	g.writeLine("// Track handler execution time")
	g.writeLine("defer func() {")
	g.indent()
	g.writeLine("duration := time.Since(startTime)")
	g.writeLine("recordEventDuration(eventName, duration)")
	g.writeLine("decrementActiveConnections()")
	g.unindent()
	g.writeLine("}()")
	g.writeLine("")
}

// generateWebSocketGenericInterceptorLogic generates generic WebSocket interceptor logic
func (g *CodeGenerator) generateWebSocketGenericInterceptorLogic(interceptorName string) {
	g.writeLine(fmt.Sprintf("// %s WebSocket interceptor logic", interceptorName))
	g.writeLine("// TODO: Implement your custom WebSocket interceptor logic here")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("event := wsCtx.EventName()")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Example interceptor processing")
	g.writeLine("processWebSocketMessage(client, event, messageBody)")
	g.writeLine("")
	g.writeLine("// Add custom metadata")
	g.writeLine("wsCtx.SetMetadata(\"interceptor\", \"" + interceptorName + "\")")
	g.writeLine("wsCtx.SetMetadata(\"processed_at\", time.Now())")
	g.writeLine("")
}

// ===================== WebSocket Pipe Middleware Generation =====================

// generateWebSocketPipeMethod generates a WebSocket pipe method
func (g *CodeGenerator) generateWebSocketPipeMethod(gatewayName, pipeName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s pipe middleware for WebSocket", pipeName, pipeName))
	g.writeLine(fmt.Sprintf("func (ws *%s) %s(wsCtx *websocket.PipeContext) interface{} {", gatewayName, pipeName))
	g.indent()
	
	// Generate pipe logic based on pipe name
	switch pipeName {
	case "WSValidationPipe", "ValidationPipe":
		g.generateWebSocketValidationPipeLogic()
	case "WSTransformPipe", "TransformPipe":
		g.generateWebSocketTransformPipeLogic()
	case "WSParseIntPipe", "ParseIntPipe":
		g.generateWebSocketParseIntPipeLogic()
	case "WSParseFloatPipe", "ParseFloatPipe":
		g.generateWebSocketParseFloatPipeLogic()
	case "WSSanitizePipe", "SanitizePipe":
		g.generateWebSocketSanitizePipeLogic()
	default:
		g.generateWebSocketGenericPipeLogic(pipeName)
	}
	
	g.writeLine("// Return transformed data or original if no transformation needed")
	g.writeLine("return wsCtx.MessageBody()")
	g.unindent()
	g.writeLine("}")
}

// generateWebSocketValidationPipeLogic generates WebSocket validation pipe logic
func (g *CodeGenerator) generateWebSocketValidationPipeLogic() {
	g.writeLine("// WebSocket validation pipe logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Validate message body structure")
	g.writeLine("if err := validateMessageStructure(eventName, messageBody); err != nil {")
	g.indent()
	g.writeLine("// Throw validation error")
	g.writeLine("wsCtx.ThrowError(fmt.Errorf(\"validation failed: %w\", err))")
	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Apply validation rules")
	g.writeLine("validatedData := applyValidationRules(messageBody)")
	g.writeLine("")
}

// generateWebSocketTransformPipeLogic generates WebSocket transform pipe logic
func (g *CodeGenerator) generateWebSocketTransformPipeLogic() {
	g.writeLine("// WebSocket transform pipe logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Transform message data")
	g.writeLine("transformedData := transformMessageData(messageBody)")
	g.writeLine("")
	g.writeLine("// Apply data normalization")
	g.writeLine("normalizedData := normalizeData(transformedData)")
	g.writeLine("")
	g.writeLine("return normalizedData")
}

// generateWebSocketParseIntPipeLogic generates WebSocket parse int pipe logic
func (g *CodeGenerator) generateWebSocketParseIntPipeLogic() {
	g.writeLine("// WebSocket parse int pipe logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Parse string values to integers")
	g.writeLine("if bodyMap, ok := messageBody.(map[string]interface{}); ok {")
	g.indent()
	g.writeLine("for key, value := range bodyMap {")
	g.indent()
	g.writeLine("if strVal, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if intVal, err := strconv.Atoi(strVal); err == nil {")
	g.indent()
	g.writeLine("bodyMap[key] = intVal")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return bodyMap")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateWebSocketParseFloatPipeLogic generates WebSocket parse float pipe logic
func (g *CodeGenerator) generateWebSocketParseFloatPipeLogic() {
	g.writeLine("// WebSocket parse float pipe logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Parse string values to floats")
	g.writeLine("if bodyMap, ok := messageBody.(map[string]interface{}); ok {")
	g.indent()
	g.writeLine("for key, value := range bodyMap {")
	g.indent()
	g.writeLine("if strVal, ok := value.(string); ok {")
	g.indent()
	g.writeLine("if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {")
	g.indent()
	g.writeLine("bodyMap[key] = floatVal")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return bodyMap")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateWebSocketSanitizePipeLogic generates WebSocket sanitize pipe logic
func (g *CodeGenerator) generateWebSocketSanitizePipeLogic() {
	g.writeLine("// WebSocket sanitize pipe logic")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("")
	g.writeLine("// Sanitize string values")
	g.writeLine("sanitizedData := sanitizeWebSocketData(messageBody)")
	g.writeLine("")
	g.writeLine("// Remove dangerous content")
	g.writeLine("cleanedData := removeScriptTags(sanitizedData)")
	g.writeLine("filteredData := filterProfanity(cleanedData)")
	g.writeLine("")
	g.writeLine("return filteredData")
}

// generateWebSocketGenericPipeLogic generates generic WebSocket pipe logic
func (g *CodeGenerator) generateWebSocketGenericPipeLogic(pipeName string) {
	g.writeLine(fmt.Sprintf("// %s WebSocket pipe logic", pipeName))
	g.writeLine("// TODO: Implement your custom WebSocket pipe logic here")
	g.writeLine("messageBody := wsCtx.MessageBody()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Example pipe processing")
	g.writeLine("processedData := processWebSocketData(eventName, messageBody)")
	g.writeLine("")
	g.writeLine("return processedData")
}

// ===================== WebSocket Filter Middleware Generation =====================

// generateWebSocketFilterMethod generates a WebSocket filter method
func (g *CodeGenerator) generateWebSocketFilterMethod(gatewayName, filterName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s filter middleware for WebSocket", filterName, filterName))
	g.writeLine(fmt.Sprintf("func (ws *%s) %s(wsCtx *websocket.FilterContext, err error) {", gatewayName, filterName))
	g.indent()
	
	// Generate filter logic based on filter name
	switch filterName {
	case "WSGlobalExceptionFilter", "GlobalExceptionFilter":
		g.generateWebSocketGlobalExceptionFilterLogic()
	case "WSValidationExceptionFilter", "ValidationExceptionFilter":
		g.generateWebSocketValidationExceptionFilterLogic()
	case "WSHttpExceptionFilter", "HttpExceptionFilter":
		g.generateWebSocketHttpExceptionFilterLogic()
	case "WSAuthExceptionFilter", "AuthExceptionFilter":
		g.generateWebSocketAuthExceptionFilterLogic()
	case "WSBusinessExceptionFilter", "BusinessExceptionFilter":
		g.generateWebSocketBusinessExceptionFilterLogic()
	default:
		g.generateWebSocketGenericFilterLogic(filterName)
	}
	
	g.unindent()
	g.writeLine("}")
}

// generateWebSocketGlobalExceptionFilterLogic generates WebSocket global exception filter logic
func (g *CodeGenerator) generateWebSocketGlobalExceptionFilterLogic() {
	g.writeLine("// WebSocket global exception filter logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Log the error")
	g.writeLine("logger.Error(\"WebSocket error occurred\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":    eventName,")
	g.writeLine("\"client_id\": client.ID(),")
	g.writeLine("\"error\":     err.Error(),")
	g.writeLine("\"timestamp\": time.Now(),")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
	g.writeLine("// Send generic error response to client")
	g.writeLine("client.Emit(\"error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":     eventName,")
	g.writeLine("\"message\":   \"An error occurred processing your request\",")
	g.writeLine("\"timestamp\": time.Now(),")
	g.writeLine("\"type\":      \"internal_error\",")
	g.unindent()
	g.writeLine("})")
}

// generateWebSocketValidationExceptionFilterLogic generates WebSocket validation exception filter logic
func (g *CodeGenerator) generateWebSocketValidationExceptionFilterLogic() {
	g.writeLine("// WebSocket validation exception filter logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Send validation error response")
	g.writeLine("client.Emit(\"validation_error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":     eventName,")
	g.writeLine("\"message\":   \"Validation failed\",")
	g.writeLine("\"details\":   err.Error(),")
	g.writeLine("\"timestamp\": time.Now(),")
	g.writeLine("\"type\":      \"validation_error\",")
	g.unindent()
	g.writeLine("})")
}

// generateWebSocketHttpExceptionFilterLogic generates WebSocket HTTP exception filter logic
func (g *CodeGenerator) generateWebSocketHttpExceptionFilterLogic() {
	g.writeLine("// WebSocket HTTP exception filter logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Handle HTTP-like errors in WebSocket context")
	g.writeLine("var statusCode int")
	g.writeLine("var message string")
	g.writeLine("")
	g.writeLine("switch err.(type) {")
	g.writeLine("case *UnauthorizedError:")
	g.indent()
	g.writeLine("statusCode = 401")
	g.writeLine("message = \"Unauthorized\"")
	g.unindent()
	g.writeLine("case *ForbiddenError:")
	g.indent()
	g.writeLine("statusCode = 403")
	g.writeLine("message = \"Forbidden\"")
	g.unindent()
	g.writeLine("case *NotFoundError:")
	g.indent()
	g.writeLine("statusCode = 404")
	g.writeLine("message = \"Not Found\"")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("statusCode = 500")
	g.writeLine("message = \"Internal Server Error\"")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("client.Emit(\"http_error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":      eventName,")
	g.writeLine("\"statusCode\": statusCode,")
	g.writeLine("\"message\":    message,")
	g.writeLine("\"details\":    err.Error(),")
	g.writeLine("\"timestamp\":  time.Now(),")
	g.unindent()
	g.writeLine("})")
}

// generateWebSocketAuthExceptionFilterLogic generates WebSocket auth exception filter logic
func (g *CodeGenerator) generateWebSocketAuthExceptionFilterLogic() {
	g.writeLine("// WebSocket authentication exception filter logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Send authentication error and disconnect client")
	g.writeLine("client.Emit(\"auth_error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":     eventName,")
	g.writeLine("\"message\":   \"Authentication failed\",")
	g.writeLine("\"details\":   err.Error(),")
	g.writeLine("\"timestamp\": time.Now(),")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
	g.writeLine("// Disconnect the client after sending error")
	g.writeLine("client.Disconnect(\"Authentication failed\")")
}

// generateWebSocketBusinessExceptionFilterLogic generates WebSocket business exception filter logic
func (g *CodeGenerator) generateWebSocketBusinessExceptionFilterLogic() {
	g.writeLine("// WebSocket business exception filter logic")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Handle business logic errors")
	g.writeLine("client.Emit(\"business_error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":     eventName,")
	g.writeLine("\"message\":   \"Business logic error\",")
	g.writeLine("\"details\":   err.Error(),")
	g.writeLine("\"timestamp\": time.Now(),")
	g.writeLine("\"type\":      \"business_error\",")
	g.unindent()
	g.writeLine("})")
}

// generateWebSocketGenericFilterLogic generates generic WebSocket filter logic
func (g *CodeGenerator) generateWebSocketGenericFilterLogic(filterName string) {
	g.writeLine(fmt.Sprintf("// %s WebSocket filter logic", filterName))
	g.writeLine("// TODO: Implement your custom WebSocket filter logic here")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("eventName := wsCtx.EventName()")
	g.writeLine("")
	g.writeLine("// Log the error")
	g.writeLine("logger.Error(fmt.Sprintf(\"WebSocket error in %s\", eventName), err)")
	g.writeLine("")
	g.writeLine("// Send custom error response")
	g.writeLine("client.Emit(\"custom_error\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"event\":     eventName,")
	g.writeLine("\"filter\":    \"" + filterName + "\",")
	g.writeLine("\"message\":   err.Error(),")
	g.writeLine("\"timestamp\": time.Now(),")
	g.unindent()
	g.writeLine("})")
}