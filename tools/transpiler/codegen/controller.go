package codegen

import (
	"fmt"
	"strings"
)

// generateWebSocketGatewayDeclaration generates a WebSocket gateway declaration
func (g *CodeGenerator) generateWebSocketGatewayDeclaration(gateway *WebSocketGatewayDeclaration) error {
	// Generate the WebSocket gateway struct
	g.writeLine(fmt.Sprintf("type %s struct {", gateway.Name))
	g.indent()

	// Generate fields with injection tags
	for _, field := range gateway.Fields {
		tag := g.generateInjectionTag(field)
		if tag != "" {
			g.writeLine(fmt.Sprintf("%s %s `%s`", field.Name, field.Type, tag))
		} else {
			g.writeLine(fmt.Sprintf("%s %s", field.Name, field.Type))
		}
	}

	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate WebSocket server setup function
	g.generateWebSocketServerSetup(gateway)
	g.writeLine("")

	// Generate message handler registration function
	g.generateWebSocketHandlerRegistration(gateway)
	g.writeLine("")

	// Generate methods (message handlers and lifecycle handlers)
	for _, method := range gateway.Methods {
		if err := g.generateWebSocketMethod(gateway, method); err != nil {
			return err
		}
		g.writeLine("")
	}

	return nil
}

// generateWebSocketServerSetup generates the WebSocket server setup function
func (g *CodeGenerator) generateWebSocketServerSetup(gateway *WebSocketGatewayDeclaration) {
	// Extract WebSocket gateway configuration
	wsConfig := g.getWebSocketGatewayConfig(gateway)
	
	g.writeLine(fmt.Sprintf("// Setup%s initializes the WebSocket server for %s", gateway.Name, gateway.Name))
	g.writeLine(fmt.Sprintf("func (ws *%s) Setup%s() *websocket.WebSocketServer {", gateway.Name, gateway.Name))
	g.indent()
	
	// Create WebSocket server with configuration
	g.writeLine(fmt.Sprintf("server := websocket.NewWebSocketServer(&websocket.Config{"))
	g.indent()
	g.writeLine(fmt.Sprintf("Port: %d,", wsConfig.Port))
	if wsConfig.Namespace != "" {
		g.writeLine(fmt.Sprintf("Namespace: \"%s\",", wsConfig.Namespace))
	}
	if wsConfig.CORS {
		g.writeLine("CORS: true,")
	}
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
	
	// Register lifecycle handlers
	g.writeLine("// Register lifecycle handlers")
	if g.hasWebSocketLifecycleHandler(gateway, "OnGatewayConnection") {
		g.writeLine("server.OnConnection(ws.HandleConnection)")
	}
	if g.hasWebSocketLifecycleHandler(gateway, "OnGatewayDisconnect") {
		g.writeLine("server.OnDisconnection(ws.HandleDisconnect)")
	}
	if g.hasWebSocketLifecycleHandler(gateway, "OnGatewayInit") {
		g.writeLine("server.OnInit(ws.AfterInit)")
	}
	g.writeLine("")
	
	// Register message handlers
	g.writeLine("// Register message handlers")
	g.generateMessageHandlerRegistrations(gateway)
	
	g.writeLine("return server")
	g.unindent()
	g.writeLine("}")
}

// generateWebSocketHandlerRegistration generates message handler registration function
func (g *CodeGenerator) generateWebSocketHandlerRegistration(gateway *WebSocketGatewayDeclaration) {
	g.writeLine(fmt.Sprintf("// Register%sHandlers registers all WebSocket message handlers", gateway.Name))
	g.writeLine(fmt.Sprintf("func (ws *%s) Register%sHandlers(server *websocket.WebSocketServer) {", gateway.Name, gateway.Name))
	g.indent()
	
	g.generateMessageHandlerRegistrations(gateway)
	
	g.unindent()
	g.writeLine("}")
}

// generateMessageHandlerRegistrations generates the message handler registrations
func (g *CodeGenerator) generateMessageHandlerRegistrations(gateway *WebSocketGatewayDeclaration) {
	// Register methods from the gateway itself
	for _, method := range gateway.Methods {
		// Look for @SubscribeMessage decorators
		for _, decorator := range method.Decorators {
			if decorator.Name == "SubscribeMessage" && len(decorator.Args) > 0 {
				if eventName, ok := decorator.Args[0].Value.(string); ok {
					g.writeLine(fmt.Sprintf("server.OnMessage(\"%s\", ws.%s)", eventName, method.Name))
				} else if eventArray, ok := decorator.Args[0].Value.([]interface{}); ok {
					// Handle multiple events
					for _, event := range eventArray {
						if eventStr, ok := event.(string); ok {
							g.writeLine(fmt.Sprintf("server.OnMessage(\"%s\", ws.%s)", eventStr, method.Name))
						}
					}
				}
			}
		}
	}
	
	// Register standalone WebSocket functions (need to be collected globally)
	// This is a temporary approach - ideally we should collect these during file parsing
	g.registerStandaloneWebSocketFunctions()
}

// registerStandaloneWebSocketFunctions registers standalone WebSocket handler functions
func (g *CodeGenerator) registerStandaloneWebSocketFunctions() {
	// Keep track of registered handlers to avoid duplicates
	registeredMessages := make(map[string]bool)
	registeredLifecycle := make(map[string]bool)
	
	// Register collected standalone WebSocket functions
	for _, wsFunc := range g.webSocketFunctions {
		// Look for @SubscribeMessage decorators in standalone functions
		for _, decorator := range wsFunc.Decorators {
			if decorator.Name == "SubscribeMessage" && len(decorator.Args) > 0 {
				if eventName, ok := decorator.Args[0].Value.(string); ok {
					registrationKey := fmt.Sprintf("%s:%s", eventName, wsFunc.Name)
					if !registeredMessages[registrationKey] {
						g.writeLine(fmt.Sprintf("server.OnMessage(\"%s\", %s)", eventName, wsFunc.Name))
						registeredMessages[registrationKey] = true
					}
				} else if eventArray, ok := decorator.Args[0].Value.([]interface{}); ok {
					// Handle multiple events
					for _, event := range eventArray {
						if eventStr, ok := event.(string); ok {
							registrationKey := fmt.Sprintf("%s:%s", eventStr, wsFunc.Name)
							if !registeredMessages[registrationKey] {
								g.writeLine(fmt.Sprintf("server.OnMessage(\"%s\", %s)", eventStr, wsFunc.Name))
								registeredMessages[registrationKey] = true
							}
						}
					}
				}
			}
		}
		
		// Register lifecycle handlers
		for _, decorator := range wsFunc.Decorators {
			switch decorator.Name {
			case "OnGatewayConnection":
				if !registeredLifecycle[wsFunc.Name] {
					g.writeLine(fmt.Sprintf("server.OnConnection(%s)", wsFunc.Name))
					registeredLifecycle[wsFunc.Name] = true
				}
			case "OnGatewayDisconnect":
				if !registeredLifecycle[wsFunc.Name] {
					g.writeLine(fmt.Sprintf("server.OnDisconnection(%s)", wsFunc.Name))
					registeredLifecycle[wsFunc.Name] = true
				}
			case "OnGatewayInit":
				if !registeredLifecycle[wsFunc.Name] {
					g.writeLine(fmt.Sprintf("server.OnInit(%s)", wsFunc.Name))
					registeredLifecycle[wsFunc.Name] = true
				}
			}
		}
	}
}

// generateWebSocketMethod generates a WebSocket message handler or lifecycle method
func (g *CodeGenerator) generateWebSocketMethod(gateway *WebSocketGatewayDeclaration, method *MethodNode) error {
	// Check if this is a message handler or lifecycle handler
	isMessageHandler := g.hasDecoratorByName(method, "SubscribeMessage")
	isLifecycleHandler := g.hasWebSocketLifecycleDecorator(method)
	
	if isMessageHandler {
		return g.generateWebSocketMessageHandler(gateway, method)
	} else if isLifecycleHandler {
		return g.generateWebSocketLifecycleHandler(gateway, method)
	}
	
	// Regular method - generate as normal controller method
	return g.generateControllerMethod(&ControllerDeclaration{
		Name:    gateway.Name,
		Fields:  gateway.Fields,
		Methods: []*MethodNode{method},
	}, method)
}

// generateWebSocketMessageHandler generates a WebSocket message handler method
func (g *CodeGenerator) generateWebSocketMessageHandler(gateway *WebSocketGatewayDeclaration, method *MethodNode) error {
	// Generate method signature with WebSocket context
	signature := g.generateWebSocketMessageHandlerSignature(gateway.Name, method)
	g.writeLine(signature + " {")
	g.indent()

	// Generate parameter extraction for WebSocket context
	g.generateWebSocketParameterExtraction(method)

	// Get the message event name for context
	eventName := g.getSubscribeMessageEventName(method)
	
	g.writeLine(fmt.Sprintf("// Handle WebSocket message: %s", eventName))
	g.writeLine("// Extract message data and process")
	g.writeLine("")
	
	// Generate message validation if needed
	g.generateWebSocketMessageValidation(method)
	
	// Generate business logic placeholder
	g.writeLine("// TODO: Implement WebSocket message handling logic")
	g.writeLine("// Process the incoming message and send responses if needed")
	g.writeLine("")
	
	// Generate response/acknowledgment logic
	g.generateWebSocketResponseLogic(method)

	// Generate return statement if method has return type
	if method.ReturnType != "" {
		if method.ReturnType == "error" {
			g.writeLine("return nil")
		}
	}

	g.unindent()
	g.writeLine("}")
	return nil
}

// generateWebSocketLifecycleHandler generates a WebSocket lifecycle handler method
func (g *CodeGenerator) generateWebSocketLifecycleHandler(gateway *WebSocketGatewayDeclaration, method *MethodNode) error {
	// Generate method signature
	signature := g.generateWebSocketLifecycleHandlerSignature(gateway.Name, method)
	g.writeLine(signature + " {")
	g.indent()

	// Generate parameter extraction for WebSocket lifecycle context
	g.generateWebSocketLifecycleParameterExtraction(method)
	
	// Generate lifecycle-specific logic based on decorator type
	for _, decorator := range method.Decorators {
		switch decorator.Name {
		case "OnGatewayConnection":
			g.generateOnConnectionHandler(method)
		case "OnGatewayDisconnect":
			g.generateOnDisconnectHandler(method)
		case "OnGatewayInit":
			g.generateOnInitHandler(method)
		}
	}

	g.writeLine("")
	g.writeLine("// TODO: Implement lifecycle handler logic")

	// Generate return statement if method has return type
	if method.ReturnType != "" {
		if method.ReturnType == "error" {
			g.writeLine("return nil")
		}
	}

	g.unindent()
	g.writeLine("}")
	return nil
}

// generateWebSocketFunctionDeclaration generates a WebSocket lifecycle function
func (g *CodeGenerator) generateWebSocketFunctionDeclaration(wsFunc *WebSocketFunctionDeclaration) error {
	// Generate function signature
	funcSignature := fmt.Sprintf("func %s(", wsFunc.Name)
	
	// Add parameters
	var params []string
	for _, param := range wsFunc.Params {
		paramStr := ""
		if param.Name != "" {
			paramStr = param.Name + " "
		}
		paramStr += param.Type
		params = append(params, paramStr)
	}
	
	funcSignature += strings.Join(params, ", ") + ")"
	
	// Add return type if present
	if wsFunc.ReturnType != "" {
		funcSignature += " " + wsFunc.ReturnType
	}
	
	// Generate function
	g.writeLine(funcSignature + " {")
	g.indent()
	
	// Add function body based on decorator type
	addedComment := false
	for _, decorator := range wsFunc.Decorators {
		if !addedComment {
			switch decorator.Name {
			case "OnGatewayConnection":
				g.writeLine("// Handle new WebSocket connection")
				g.writeLine("// Add connection logic here")
			case "OnGatewayDisconnect":
				g.writeLine("// Handle WebSocket disconnection")
				g.writeLine("// Add cleanup logic here")
			case "OnGatewayInit":
				g.writeLine("// Initialize WebSocket gateway")
				g.writeLine("// Add initialization logic here")
			case "SubscribeMessage":
				if len(decorator.Args) > 0 {
					if argValue, ok := decorator.Args[0].Value.(string); ok {
						g.writeLine(fmt.Sprintf("// Handle WebSocket message: %s", argValue))
					} else if argArray, ok := decorator.Args[0].Value.([]interface{}); ok {
						var events []string
						for _, event := range argArray {
							if eventStr, ok := event.(string); ok {
								events = append(events, eventStr)
							}
						}
						g.writeLine(fmt.Sprintf("// Handle WebSocket messages: %s", strings.Join(events, ", ")))
					}
				}
				g.writeLine("// Add message handling logic here")
			}
			addedComment = true
		}
	}
	
	// Add return statement if function has return type
	if wsFunc.ReturnType != "" {
		if wsFunc.ReturnType == "error" {
			g.writeLine("return nil")
		}
	}
	
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	return nil
}

// generateControllerDeclaration generates Go code for a controller
func (g *CodeGenerator) generateControllerDeclaration(controller *ControllerDeclaration) error {
	// Generate struct declaration
	g.writeLine(fmt.Sprintf("type %s struct {", controller.Name))
	g.indent()

	// Generate fields with injection tags
	for _, field := range controller.Fields {
		tag := g.generateInjectionTag(field)
		if tag != "" {
			g.writeLine(fmt.Sprintf("%s %s `%s`", field.Name, field.Type, tag))
		} else {
			g.writeLine(fmt.Sprintf("%s %s", field.Name, field.Type))
		}
	}

	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate route registration function
	g.generateControllerRouteRegistration(controller)
	g.writeLine("")

	// Generate methods
	for _, method := range controller.Methods {
		if err := g.generateControllerMethod(controller, method); err != nil {
			return err
		}
		g.writeLine("")
	}

	// Generate error filter handlers for controller-level @Catch() decorators
	g.generateCatchHandlers(controller)
	
	// Generate guard middleware functions
	g.generateGuardMiddlewareFunctions(controller)
	
	// Generate interceptor middleware functions
	g.generateInterceptorMiddlewareFunctions(controller)
	
	// Generate pipe middleware functions
	g.generatePipeMiddlewareFunctions(controller)

	return nil
}

// generateControllerRouteRegistration generates route registration code
func (g *CodeGenerator) generateControllerRouteRegistration(controller *ControllerDeclaration) {
	g.writeLine(fmt.Sprintf("func (c *%s) RegisterRoutes(server *httpPackage.HTTPServer) error {", controller.Name))
	g.indent()

	controllerPath := g.getControllerPath(controller)

	// Register error filters first
	g.generateErrorFilterRegistration(controller)

	for _, method := range controller.Methods {
		routeInfo := g.getRouteInfo(method)
		if routeInfo.Method != "" {
			fullPath := g.combineRoutePaths(controllerPath, routeInfo.Path)

			// Generate route registration with middleware chain (guards + interceptors)
			middlewareChain := g.generateMiddlewareChain(controller, method)
			if middlewareChain != "" {
				// Register route with middleware chain
				g.writeLine(fmt.Sprintf("server.%s(\"%s\", %s, c.%s)",
					strings.Title(strings.ToLower(routeInfo.Method)),
					fullPath,
					middlewareChain,
					method.Name))
			} else {
				// Generate route registration without middleware
				g.writeLine(fmt.Sprintf("server.%s(\"%s\", c.%s)",
					strings.Title(strings.ToLower(routeInfo.Method)),
					fullPath,
					method.Name))
			}
		}
	}

	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
}

// generateErrorFilterRegistration generates error filter registration code
func (g *CodeGenerator) generateErrorFilterRegistration(controller *ControllerDeclaration) {
	// Check for controller-level @Catch() decorators
	hasControllerFilters := false
	for _, decorator := range controller.Decorators {
		if decorator.Name == "Catch" {
			hasControllerFilters = true
			config := g.getCatchFilterConfig(decorator, "controller")
			
			// Generate filter registration
			if len(config.ErrorTypes) > 0 {
				for _, errorType := range config.ErrorTypes {
					handlerName := fmt.Sprintf("handle%sError", strings.Title(errorType))
					g.writeLine(fmt.Sprintf("server.RegisterErrorFilter(\"%s\", c.%s)", errorType, handlerName))
				}
			} else {
				// Global error filter
				g.writeLine("server.RegisterGlobalErrorFilter(c.handleError)")
			}
		}
	}

	// Check for method-level @Catch() decorators
	hasMethodFilters := false
	for _, method := range controller.Methods {
		for _, decorator := range method.Decorators {
			if decorator.Name == "Catch" {
				hasMethodFilters = true
				config := g.getCatchFilterConfig(decorator, "method")
				handlerName := fmt.Sprintf("handle%sError", strings.Title(method.Name))
				
				if len(config.ErrorTypes) > 0 {
					for _, errorType := range config.ErrorTypes {
						g.writeLine(fmt.Sprintf("server.RegisterMethodErrorFilter(\"%s\", \"%s\", c.%s)", 
							method.Name, errorType, handlerName))
					}
				} else {
					// Method-level global filter
					g.writeLine(fmt.Sprintf("server.RegisterMethodErrorFilter(\"%s\", \"*\", c.%s)", 
						method.Name, handlerName))
				}
			}
		}
	}

	if hasControllerFilters || hasMethodFilters {
		g.writeLine("")
	}
}

// generateControllerMethod generates a controller method with HTTP context
func (g *CodeGenerator) generateControllerMethod(controller *ControllerDeclaration, method *MethodNode) error {
	// Generate method signature
	signature := g.generateMethodSignature(controller.Name, method, true)
	g.writeLine(signature + " {")
	g.indent()

	// Check for @Redirect() decorator first - it takes precedence over other response handling
	if g.hasRedirectDecorator(method) {
		g.generateRedirectResponse(method)
		g.unindent()
		g.writeLine("}")
		return nil
	}

	// Generate HTTP status code setting if @HttpCode() decorator is present (before any returns)
	g.generateHttpStatusCode(method)

	// Generate response header setting if @Header() decorators are present
	g.generateResponseHeaders(method)

	// Generate parameter extraction from HTTP context
	g.generateParameterExtraction(method)

	// Generate method body placeholder
	g.writeLine("// TODO: Implement method logic")
	
	// Generate response with appropriate status code
	statusCode := g.getHttpStatusCode(method)
	if statusCode == 0 {
		statusCode = 200 // Default status code
	}
	g.writeLine(fmt.Sprintf("ctx.JSON(%d, map[string]interface{}{\"message\": \"Not implemented\"})", statusCode))

	g.unindent()
	g.writeLine("}")

	return nil
}

// generateControllerGuardMethod generates a guard middleware method for the controller
func (g *CodeGenerator) generateControllerGuardMethod(controllerName, guardName string) {
	g.writeLine(fmt.Sprintf("// %s implements the %s guard middleware", guardName, guardName))
	g.writeLine(fmt.Sprintf("func (c *%s) %s(ctx *httpPackage.RequestContext) {", controllerName, guardName))
	g.indent()
	
	// Generate guard logic based on guard name
	switch guardName {
	case "AuthGuard":
		g.generateAuthGuardLogic()
	case "RoleGuard":
		g.generateRoleGuardLogic()
	case "PermissionGuard":
		g.generatePermissionGuardLogic()
	default:
		g.generateGenericGuardLogic(guardName)
	}
	
	g.writeLine("// If guard passes, continue to next middleware/handler")
	g.writeLine("ctx.Next()")
	g.unindent()
	g.writeLine("}")
}

// getControllerPath extracts the base path from controller decorators
func (g *CodeGenerator) getControllerPath(controller *ControllerDeclaration) string {
	var basePath string
	
	// Get the base path from @Controller decorator
	controllerDecorator := g.getDecorator(controller.Decorators, "Controller")
	if controllerDecorator != nil && len(controllerDecorator.Args) > 0 {
		basePath = g.getDecoratorArgValue(controllerDecorator, 0)
	} else {
		// Default path based on controller name
		name := strings.TrimSuffix(controller.Name, "Controller")
		basePath = "/" + strings.ToLower(name)
	}

	// Check for @Version decorator and prepend version prefix
	versionDecorator := g.getDecorator(controller.Decorators, "Version")
	if versionDecorator != nil && len(versionDecorator.Args) > 0 {
		version := g.getDecoratorArgValue(versionDecorator, 0)
		
		// Skip empty version strings
		if strings.TrimSpace(version) == "" {
			return basePath
		}
		
		// Normalize version to lowercase for consistency
		version = strings.ToLower(version)
		
		// Ensure version starts with "v" if it's not already there
		if !strings.HasPrefix(version, "v") && !strings.HasPrefix(version, "/v") {
			version = "v" + version
		}
		// Ensure version starts with "/" if it doesn't already
		if !strings.HasPrefix(version, "/") {
			version = "/" + version
		}
		
		// Combine version with base path
		basePath = version + basePath
	}
	return basePath
}

// getRouteInfo extracts HTTP method and path information from method decorators
func (g *CodeGenerator) getRouteInfo(method *MethodNode) RouteInfo {
	routeInfo := RouteInfo{}

	// Check for HTTP method decorators
	httpMethods := []string{"Get", "Post", "Put", "Delete", "Patch", "Options", "Head"}

	for _, decorator := range method.Decorators {
		for _, httpMethod := range httpMethods {
			if decorator.Name == httpMethod {
				routeInfo.Method = strings.ToUpper(httpMethod)
				if len(decorator.Args) > 0 {
					routeInfo.Path = g.getDecoratorArgValue(decorator, 0)
				}
				return routeInfo
			}
		}
	}

	// Fallback to method name convention
	methodName := strings.ToLower(method.Name)
	for _, httpMethod := range []string{"get", "post", "put", "delete", "patch"} {
		if strings.HasPrefix(methodName, httpMethod) {
			routeInfo.Method = strings.ToUpper(httpMethod)
			routeInfo.Path = "/" + strings.TrimPrefix(methodName, httpMethod)
			break
		}
	}

	return routeInfo
}

// combineRoutePaths combines controller and method paths
func (g *CodeGenerator) combineRoutePaths(controllerPath, methodPath string) string {
	if methodPath == "" {
		methodPath = ""
	}

	path := strings.TrimSuffix(controllerPath, "/") + "/" + strings.TrimPrefix(methodPath, "/")
	
	// Clean up multiple consecutive slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	if path == "/" {
		return "/"
	}

	return strings.TrimSuffix(path, "/")
}

// generateMiddlewareChain generates middleware chain for a route
func (g *CodeGenerator) generateMiddlewareChain(controller *ControllerDeclaration, method *MethodNode) string {
	var middleware []string
	
	// Collect pipes first (they run before interceptors and guards in the pipeline)
	controllerPipes := g.getPipeDecorators(controller.Decorators)
	middleware = append(middleware, controllerPipes...)
	
	methodPipes := g.getPipeDecorators(method.Decorators)
	middleware = append(middleware, methodPipes...)
	
	// Collect interceptors second (they run after pipes but before guards)
	controllerInterceptors := g.getInterceptorDecorators(controller.Decorators)
	middleware = append(middleware, controllerInterceptors...)
	
	methodInterceptors := g.getInterceptorDecorators(method.Decorators)
	middleware = append(middleware, methodInterceptors...)
	
	// Collect guards last (they run after pipes and interceptors)
	controllerGuards := g.getGuardDecorators(controller.Decorators)
	middleware = append(middleware, controllerGuards...)
	
	methodGuards := g.getGuardDecorators(method.Decorators)
	middleware = append(middleware, methodGuards...)
	
	
	if len(middleware) == 0 {
		return ""
	}
	
	// Create middleware chain - generate the middleware instance calls
	var middlewareChain []string
	for _, mw := range middleware {
		middlewareChain = append(middlewareChain, fmt.Sprintf("c.%s", mw))
	}
	
	return strings.Join(middlewareChain, ", ")
}

// generateParameterExtraction generates parameter extraction code from HTTP context
func (g *CodeGenerator) generateParameterExtraction(method *MethodNode) {
	for _, param := range method.Params {
		paramDecorators := g.getParameterDecorators(param)

		for _, decorator := range paramDecorators {
			switch decorator.Name {
			case "Body":
				g.writeLine(fmt.Sprintf("var %s %s", param.Name, param.Type))
				g.writeLine(fmt.Sprintf("if err := ctx.ParseJSON(&%s); err != nil {", param.Name))
				g.indent()
				g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})")
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")

			case "Param":
				paramName := g.getDecoratorArgValue(decorator, 0)
				if paramName == "" {
					paramName = param.Name
				}
				
				// Get parameter constraint options
				constraintOptions := g.getParameterConstraintOptions(decorator)
				
				// Extract parameter value
				g.writeLine(fmt.Sprintf("%sValue := ctx.GetParam(\"%s\")", param.Name, paramName))
				
				// Handle required parameter validation
				if constraintOptions.Required {
					g.writeLine(fmt.Sprintf("if %sValue == \"\" {", param.Name))
					g.indent()
					g.writeLine(fmt.Sprintf("ctx.JSON(400, map[string]string{\"error\": \"Parameter '%s' is required\"})", param.Name))
					g.writeLine("return")
					g.unindent()
					g.writeLine("}")
				}
				
				// Generate constraint validation if parameter is not empty
				if len(constraintOptions.Constraints) > 0 || constraintOptions.Transform != "" {
					g.writeLine(fmt.Sprintf("if %sValue != \"\" {", param.Name))
					g.indent()
					g.generateParameterConstraintValidation(param, fmt.Sprintf("%sValue", param.Name), constraintOptions)
					g.unindent()
					g.writeLine("}")
				}
				
				// Assign final value
				g.writeLine(fmt.Sprintf("%s := %sValue", param.Name, param.Name))

			case "Query":
				g.generateQueryParameterExtraction(param, decorator)

			case "Headers":
				g.generateHeaderParameterExtraction(param, decorator)
			
			case "Req":
				g.generateRequestParameterExtraction(param, decorator)
			
			case "Res":
				g.generateResponseParameterExtraction(param, decorator)
			
			case "Session":
				g.generateSessionParameterExtraction(param, decorator)
			
			case "Ip":
				g.generateIpParameterExtraction(param, decorator)
			
			case "HostParam":
				g.generateHostParamParameterExtraction(param, decorator)
			}
		}
	}
}

// WebSocket helper functions

// getWebSocketGatewayConfig extracts WebSocket configuration from gateway decorators
func (g *CodeGenerator) getWebSocketGatewayConfig(gateway *WebSocketGatewayDeclaration) WebSocketConfig {
	config := WebSocketConfig{
		Port: 8080, // Default port
		CORS: false,
	}

	// Look for @WebSocketGateway decorator
	wsGatewayDecorator := g.getDecorator(gateway.Decorators, "WebSocketGateway")
	if wsGatewayDecorator != nil && len(wsGatewayDecorator.Args) > 0 {
		// Handle different argument types
		if portNum, ok := wsGatewayDecorator.Args[0].Value.(int); ok {
			config.Port = portNum
		} else if configMap, ok := wsGatewayDecorator.Args[0].Value.(map[string]interface{}); ok {
			// Handle configuration object
			if port, ok := configMap["port"].(int); ok {
				config.Port = port
			}
			if namespace, ok := configMap["namespace"].(string); ok {
				config.Namespace = namespace
			}
			if cors, ok := configMap["cors"].(bool); ok {
				config.CORS = cors
			}
		}
	}

	return config
}

// hasWebSocketLifecycleHandler checks if gateway has a specific lifecycle handler
func (g *CodeGenerator) hasWebSocketLifecycleHandler(gateway *WebSocketGatewayDeclaration, handlerType string) bool {
	for _, method := range gateway.Methods {
		for _, decorator := range method.Decorators {
			if decorator.Name == handlerType {
				return true
			}
		}
	}
	return false
}

// hasDecoratorByName checks if method has a decorator with the given name
func (g *CodeGenerator) hasDecoratorByName(method *MethodNode, decoratorName string) bool {
	for _, decorator := range method.Decorators {
		if decorator.Name == decoratorName {
			return true
		}
	}
	return false
}

// hasWebSocketLifecycleDecorator checks if method has any WebSocket lifecycle decorator
func (g *CodeGenerator) hasWebSocketLifecycleDecorator(method *MethodNode) bool {
	lifecycleDecorators := []string{"OnGatewayConnection", "OnGatewayDisconnect", "OnGatewayInit"}
	for _, decorator := range method.Decorators {
		for _, lifecycle := range lifecycleDecorators {
			if decorator.Name == lifecycle {
				return true
			}
		}
	}
	return false
}

// generateWebSocketMessageHandlerSignature generates method signature for WebSocket message handlers
func (g *CodeGenerator) generateWebSocketMessageHandlerSignature(gatewayName string, method *MethodNode) string {
	signature := fmt.Sprintf("func (ws *%s) %s(", gatewayName, method.Name)
	
	// Add WebSocket context as first parameter
	signature += "wsCtx *websocket.MessageContext"
	
	// Add original parameters
	for _, param := range method.Params {
		// Skip WebSocket-specific parameter decorators as they're handled by context
		hasWSDecorator := false
		for _, decorator := range g.getParameterDecorators(param) {
			if g.isWebSocketParameterDecorator(decorator.Name) {
				hasWSDecorator = true
				break
			}
		}
		
		if !hasWSDecorator {
			signature += fmt.Sprintf(", %s %s", param.Name, param.Type)
		}
	}
	
	signature += ")"
	
	// Add return type if present
	if method.ReturnType != "" {
		signature += " " + method.ReturnType
	}
	
	return signature
}

// generateWebSocketLifecycleHandlerSignature generates method signature for WebSocket lifecycle handlers
func (g *CodeGenerator) generateWebSocketLifecycleHandlerSignature(gatewayName string, method *MethodNode) string {
	signature := fmt.Sprintf("func (ws *%s) %s(", gatewayName, method.Name)
	
	// Add WebSocket context parameter
	signature += "wsCtx *websocket.LifecycleContext"
	
	// Add original parameters (filtered for WebSocket context)
	for _, param := range method.Params {
		hasWSDecorator := false
		for _, decorator := range g.getParameterDecorators(param) {
			if g.isWebSocketParameterDecorator(decorator.Name) {
				hasWSDecorator = true
				break
			}
		}
		
		if !hasWSDecorator {
			signature += fmt.Sprintf(", %s %s", param.Name, param.Type)
		}
	}
	
	signature += ")"
	
	// Add return type if present
	if method.ReturnType != "" {
		signature += " " + method.ReturnType
	}
	
	return signature
}

// generateWebSocketLifecycleParameterExtraction generates parameter extraction for WebSocket lifecycle handlers
func (g *CodeGenerator) generateWebSocketLifecycleParameterExtraction(method *MethodNode) {
	g.writeLine("// Extract parameters from WebSocket lifecycle context")
	
	for _, param := range method.Params {
		for _, decorator := range g.getParameterDecorators(param) {
			switch decorator.Name {
			case "ConnectedSocket":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client()", param.Name))
			case "Headers":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Headers()", param.Name))
			case "Query":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Query()", param.Name))
			case "ClientIP":
				g.writeLine(fmt.Sprintf("%s := wsCtx.ClientIP()", param.Name))
			case "CurrentUser":
				g.writeLine(fmt.Sprintf("%s := wsCtx.User()", param.Name))
			case "DisconnectReason":
				g.writeLine(fmt.Sprintf("%s := wsCtx.DisconnectReason()", param.Name))
			case "Session":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Session()", param.Name))
			}
		}
	}
	
	if g.hasWebSocketParameterDecorators(method) {
		g.writeLine("")
	}
}

// generateOnConnectionHandler generates enhanced connection handler logic
func (g *CodeGenerator) generateOnConnectionHandler(method *MethodNode) {
	g.writeLine("// WebSocket connection established")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("headers := wsCtx.Headers()")
	g.writeLine("")
	
	g.writeLine("// Log the connection")
	g.writeLine("fmt.Printf(\"Client connected: %s from %s\\n\", client.ID(), wsCtx.ClientIP())")
	g.writeLine("")
	
	g.writeLine("// Authentication and authorization")
	g.writeLine("if authHeader := headers[\"Authorization\"]; authHeader != \"\" {")
	g.indent()
	g.writeLine("// Validate authentication token")
	g.writeLine("// user, err := authService.ValidateToken(authHeader)")
	g.writeLine("// if err != nil {")
	g.writeLine("//     client.Disconnect(\"Authentication failed\")")
	g.writeLine("//     return")
	g.writeLine("// }")
	g.writeLine("// wsCtx.SetUser(user)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	g.writeLine("// Join default rooms or user-specific rooms")
	g.writeLine("client.Join(\"global\")")
	g.writeLine("// client.Join(\"user_\" + user.ID)")
	g.writeLine("")
	
	g.writeLine("// Notify other clients of new connection")
	g.writeLine("client.Broadcast().Emit(\"user_connected\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"client_id\": client.ID(),")
	g.writeLine("\"timestamp\": time.Now(),")
	g.unindent()
	g.writeLine("})")
}

// generateOnDisconnectHandler generates enhanced disconnect handler logic
func (g *CodeGenerator) generateOnDisconnectHandler(method *MethodNode) {
	g.writeLine("// WebSocket connection terminated")
	g.writeLine("client := wsCtx.Client()")
	g.writeLine("reason := wsCtx.DisconnectReason()")
	g.writeLine("")
	
	g.writeLine("// Log the disconnection")
	g.writeLine("fmt.Printf(\"Client disconnected: %s, Reason: %s\\n\", client.ID(), reason)")
	g.writeLine("")
	
	g.writeLine("// Cleanup user-specific data")
	g.writeLine("// Remove user from active sessions")
	g.writeLine("// sessionManager.RemoveUser(client.ID())")
	g.writeLine("")
	
	g.writeLine("// Leave all rooms")
	g.writeLine("client.LeaveAllRooms()")
	g.writeLine("")
	
	g.writeLine("// Notify other clients of disconnection")
	g.writeLine("client.Broadcast().Emit(\"user_disconnected\", map[string]interface{}{")
	g.indent()
	g.writeLine("\"client_id\": client.ID(),")
	g.writeLine("\"reason\": reason,")
	g.writeLine("\"timestamp\": time.Now(),")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
	
	g.writeLine("// Cleanup resources")
	g.writeLine("// Close any open streams, database connections, etc.")
	g.writeLine("// if user := wsCtx.User(); user != nil {")
	g.writeLine("//     userCleanupService.CleanupUserSession(user.ID)")
	g.writeLine("// }")
}

// generateOnInitHandler generates enhanced init handler logic
func (g *CodeGenerator) generateOnInitHandler(method *MethodNode) {
	g.writeLine("// WebSocket gateway initialization")
	g.writeLine("fmt.Println(\"Initializing WebSocket gateway...\")")
	g.writeLine("")
	
	g.writeLine("// Initialize gateway state")
	g.writeLine("// Set up internal data structures")
	g.writeLine("// connectionPool := make(map[string]*websocket.Client)")
	g.writeLine("// activeRooms := make(map[string][]string)")
	g.writeLine("")
	
	g.writeLine("// Connect to external services")
	g.writeLine("// database, err := connectToDatabase()")
	g.writeLine("// if err != nil {")
	g.writeLine("//     log.Fatal(\"Failed to connect to database:\", err)")
	g.writeLine("// }")
	g.writeLine("")
	
	g.writeLine("// Start background services")
	g.writeLine("// go startHeartbeatService()")
	g.writeLine("// go startCleanupService()")
	g.writeLine("")
	
	g.writeLine("// Load configuration")
	g.writeLine("// config := loadGatewayConfig()")
	g.writeLine("// applyConfiguration(config)")
	g.writeLine("")
	
	g.writeLine("fmt.Println(\"WebSocket gateway initialized successfully\")")
}

// hasWebSocketParameterDecorators checks if method has any WebSocket parameter decorators
func (g *CodeGenerator) hasWebSocketParameterDecorators(method *MethodNode) bool {
	for _, param := range method.Params {
		for _, decorator := range g.getParameterDecorators(param) {
			if g.isWebSocketParameterDecorator(decorator.Name) {
				return true
			}
		}
	}
	return false
}

// generateWebSocketParameterExtraction generates parameter extraction for WebSocket context
func (g *CodeGenerator) generateWebSocketParameterExtraction(method *MethodNode) {
	for _, param := range method.Params {
		paramDecorators := g.getParameterDecorators(param)
		
		for _, decorator := range paramDecorators {
			switch decorator.Name {
			case "MessageBody":
				g.writeLine(fmt.Sprintf("// Extract message body"))
				g.writeLine(fmt.Sprintf("var %s %s", param.Name, param.Type))
				g.writeLine(fmt.Sprintf("if err := wsCtx.ParseMessageBody(&%s); err != nil {", param.Name))
				g.indent()
				g.writeLine("wsCtx.SendError(\"Invalid message body\")")
				g.writeLine("return")
				g.unindent()
				g.writeLine("}")
			case "ConnectedSocket":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client", param.Name))
			case "MessageAck":
				g.writeLine(fmt.Sprintf("%s := wsCtx.AckCallback", param.Name))
			case "Headers":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Headers", param.Name))
			case "Session":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Session", param.Name))
			case "Rooms":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client.GetRooms()", param.Name))
			case "Namespace":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Namespace", param.Name))
			case "CurrentUser":
				g.writeLine(fmt.Sprintf("%s := wsCtx.User", param.Name))
			case "ClientIP":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client.RemoteAddr()", param.Name))
			case "EventName":
				g.writeLine(fmt.Sprintf("%s := wsCtx.EventName", param.Name))
			case "MessagePattern":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Pattern", param.Name))
			}
		}
	}
}

// getSubscribeMessageEventName extracts event name from @SubscribeMessage decorator
func (g *CodeGenerator) getSubscribeMessageEventName(method *MethodNode) string {
	for _, decorator := range method.Decorators {
		if decorator.Name == "SubscribeMessage" && len(decorator.Args) > 0 {
			if eventName, ok := decorator.Args[0].Value.(string); ok {
				return eventName
			}
		}
	}
	return "unknown"
}

// generateWebSocketMessageValidation generates message validation logic
func (g *CodeGenerator) generateWebSocketMessageValidation(method *MethodNode) {
	// Check if method has validation decorators
	hasValidation := false
	for _, param := range method.Params {
		paramDecorators := g.getParameterDecorators(param)
		for _, decorator := range paramDecorators {
			if g.isValidationDecorator(decorator.Name) {
				hasValidation = true
				break
			}
		}
		if hasValidation {
			break
		}
	}
	
	if hasValidation {
		g.writeLine("// Validate message parameters")
		g.writeLine("if err := wsCtx.ValidateMessage(); err != nil {")
		g.indent()
		g.writeLine("wsCtx.SendValidationError(err)")
		g.writeLine("return")
		g.unindent()
		g.writeLine("}")
		g.writeLine("")
	}
}

// generateWebSocketResponseLogic generates response/acknowledgment logic
func (g *CodeGenerator) generateWebSocketResponseLogic(method *MethodNode) {
	// Check if method has @MessageAck parameter
	hasAck := false
	for _, param := range method.Params {
		paramDecorators := g.getParameterDecorators(param)
		for _, decorator := range paramDecorators {
			if decorator.Name == "MessageAck" {
				hasAck = true
				break
			}
		}
		if hasAck {
			break
		}
	}
	
	if hasAck {
		g.writeLine("// Send acknowledgment if callback is provided")
		g.writeLine("if ack != nil {")
		g.indent()
		g.writeLine("ack.Send(map[string]interface{}{\"status\": \"success\"})")
		g.unindent()
		g.writeLine("}")
	}
}

// isWebSocketParameterDecorator checks if decorator is WebSocket-specific
func (g *CodeGenerator) isWebSocketParameterDecorator(decoratorName string) bool {
	wsParameterDecorators := []string{
		"MessageBody", "ConnectedSocket", "MessageAck", "MessagePattern",
		"Rooms", "Namespace", "CurrentUser", "ClientIP", "EventName", "Server",
		"DisconnectReason", "RawMessage",
	}
	
	for _, wsDecorator := range wsParameterDecorators {
		if decoratorName == wsDecorator {
			return true
		}
	}
	return false
}

// isValidationDecorator checks if decorator is for validation
func (g *CodeGenerator) isValidationDecorator(decoratorName string) bool {
	validationDecorators := []string{
		"IsString", "IsNumber", "IsEmail", "IsNotEmpty", "MinLength", "MaxLength",
		"Min", "Max", "IsPositive", "IsNegative", "Custom", // Add more as needed
	}
	
	for _, validation := range validationDecorators {
		if decoratorName == validation {
			return true
		}
	}
	return false
}


// WebSocketConfig represents WebSocket gateway configuration
type WebSocketConfig struct {
	Port      int
	Namespace string
	CORS      bool
}