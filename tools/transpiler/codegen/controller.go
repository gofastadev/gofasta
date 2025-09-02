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

	// Generate WebSocket catch handlers (error handlers)
	g.generateWebSocketCatchHandlers(gateway)

	// Generate WebSocket middleware functions
	g.generateWebSocketGuardMiddlewareFunctions(gateway)
	g.generateWebSocketInterceptorMiddlewareFunctions(gateway)
	g.generateWebSocketPipeMiddlewareFunctions(gateway)
	g.generateWebSocketFilterMiddlewareFunctions(gateway)

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
	
	// Handle CORS configuration
	g.generateWebSocketCORSConfig(wsConfig)
	
	// Handle transports configuration
	if len(wsConfig.Transports) > 0 {
		g.writeLine("Transports: []string{")
		g.indent()
		for i, transport := range wsConfig.Transports {
			if i == len(wsConfig.Transports)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", transport))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", transport))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	// Handle ping configuration
	if wsConfig.PingTimeout != 20000 { // Only output if different from default
		g.writeLine(fmt.Sprintf("PingTimeout: %d,", wsConfig.PingTimeout))
	}
	if wsConfig.PingInterval != 25000 { // Only output if different from default
		g.writeLine(fmt.Sprintf("PingInterval: %d,", wsConfig.PingInterval))
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
	
	// Register middleware
	g.generateWebSocketMiddlewareRegistration(gateway)
	
	// Register message handlers
	g.writeLine("// Register message handlers")
	g.generateMessageHandlerRegistrations(gateway)
	
	// Register WebSocket routes using comprehensive route registration
	g.writeLine("")
	g.writeLine("// Register WebSocket routes")
	g.generateWebSocketRouteRegistration(gateway)
	
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

// generateWebSocketRouteRegistration generates comprehensive WebSocket route registration
func (g *CodeGenerator) generateWebSocketRouteRegistration(gateway *WebSocketGatewayDeclaration) {
	g.writeLine(fmt.Sprintf("// Register comprehensive WebSocket routes for %s", gateway.Name))
	
	// Create route registry with metadata
	g.writeLine("routeRegistry := websocket.NewRouteRegistry()")
	g.writeLine("")
	
	// Register gateway-level routes with middleware information
	for _, method := range gateway.Methods {
		g.registerWebSocketRoute(gateway, method)
	}
	
	// Register standalone WebSocket function routes
	g.registerStandaloneWebSocketRoutes()
	
	// Apply route registry to server
	g.writeLine("// Apply route registry to server")
	g.writeLine("server.ApplyRouteRegistry(routeRegistry)")
	g.writeLine("")
	
	// Generate route documentation/metadata
	g.generateWebSocketRouteMetadata(gateway)
}

// registerWebSocketRoute registers a single WebSocket route with full metadata
func (g *CodeGenerator) registerWebSocketRoute(gateway *WebSocketGatewayDeclaration, method *MethodNode) {
	// Look for @SubscribeMessage decorators
	for _, decorator := range method.Decorators {
		if decorator.Name == "SubscribeMessage" && len(decorator.Args) > 0 {
			// Extract event name(s)
			if eventName, ok := decorator.Args[0].Value.(string); ok {
				g.generateSingleRouteRegistration(gateway, method, eventName)
			} else if eventArray, ok := decorator.Args[0].Value.([]interface{}); ok {
				// Handle multiple events
				for _, event := range eventArray {
					if eventStr, ok := event.(string); ok {
						g.generateSingleRouteRegistration(gateway, method, eventStr)
					}
				}
			}
		}
	}
}

// generateSingleRouteRegistration generates registration for a single WebSocket route
func (g *CodeGenerator) generateSingleRouteRegistration(gateway *WebSocketGatewayDeclaration, method *MethodNode, eventName string) {
	g.writeLine(fmt.Sprintf("// Register route: %s -> %s.%s", eventName, gateway.Name, method.Name))
	
	// Create route configuration
	g.writeLine(fmt.Sprintf("routeConfig := &websocket.RouteConfig{"))
	g.indent()
	g.writeLine(fmt.Sprintf("EventName: \"%s\",", eventName))
	g.writeLine(fmt.Sprintf("Handler: ws.%s,", method.Name))
	g.writeLine(fmt.Sprintf("Gateway: \"%s\",", gateway.Name))
	g.writeLine(fmt.Sprintf("Method: \"%s\",", method.Name))
	
	// Add middleware information
	g.generateRouteMiddlewareConfig(method)
	
	// Add parameter metadata
	g.generateRouteParameterMetadata(method)
	
	// Add validation metadata
	g.generateRouteValidationMetadata(method)
	
	g.unindent()
	g.writeLine("}")
	
	// Register the route
	g.writeLine(fmt.Sprintf("routeRegistry.Register(\"%s\", routeConfig)", eventName))
	g.writeLine("")
}

// generateRouteMiddlewareConfig generates middleware configuration for a route
func (g *CodeGenerator) generateRouteMiddlewareConfig(method *MethodNode) {
	guards := g.getWebSocketGuardDecorators(method.Decorators)
	interceptors := g.getWebSocketInterceptorDecorators(method.Decorators)
	pipes := g.getWebSocketPipeDecorators(method.Decorators)
	filters := g.getWebSocketFilterDecorators(method.Decorators)
	
	if len(guards) > 0 {
		g.writeLine("Guards: []string{")
		g.indent()
		for i, guard := range guards {
			if i == len(guards)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", guard))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", guard))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	if len(interceptors) > 0 {
		g.writeLine("Interceptors: []string{")
		g.indent()
		for i, interceptor := range interceptors {
			if i == len(interceptors)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", interceptor))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", interceptor))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	if len(pipes) > 0 {
		g.writeLine("Pipes: []string{")
		g.indent()
		for i, pipe := range pipes {
			if i == len(pipes)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", pipe))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", pipe))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	if len(filters) > 0 {
		g.writeLine("Filters: []string{")
		g.indent()
		for i, filter := range filters {
			if i == len(filters)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", filter))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", filter))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
}

// generateRouteParameterMetadata generates parameter metadata for a route
func (g *CodeGenerator) generateRouteParameterMetadata(method *MethodNode) {
	if len(method.Params) > 0 {
		g.writeLine("Parameters: []websocket.ParameterMetadata{")
		g.indent()
		
		for i, param := range method.Params {
			paramDecorators := g.getParameterDecorators(param)
			if len(paramDecorators) > 0 {
				g.writeLine("{")
				g.indent()
				g.writeLine(fmt.Sprintf("Name: \"%s\",", param.Name))
				g.writeLine(fmt.Sprintf("Type: \"%s\",", param.Type))
				
				// Add parameter decorator information
				g.writeLine("Decorators: []string{")
				g.indent()
				for j, decorator := range paramDecorators {
					if j == len(paramDecorators)-1 {
						g.writeLine(fmt.Sprintf("\"%s\",", decorator.Name))
					} else {
						g.writeLine(fmt.Sprintf("\"%s\",", decorator.Name))
					}
				}
				g.unindent()
				g.writeLine("},")
				
				g.unindent()
				if i == len(method.Params)-1 {
					g.writeLine("},")
				} else {
					g.writeLine("},")
				}
			}
		}
		
		g.unindent()
		g.writeLine("},")
	}
}

// generateRouteValidationMetadata generates validation metadata for a route
func (g *CodeGenerator) generateRouteValidationMetadata(method *MethodNode) {
	// Check if method has any validation decorators
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
		g.writeLine("HasValidation: true,")
		g.writeLine("ValidationRules: []string{")
		g.indent()
		
		for _, param := range method.Params {
			paramDecorators := g.getParameterDecorators(param)
			for _, decorator := range paramDecorators {
				if g.isValidationDecorator(decorator.Name) {
					g.writeLine(fmt.Sprintf("\"%s:%s\",", param.Name, decorator.Name))
				}
			}
		}
		
		g.unindent()
		g.writeLine("},")
	}
}

// registerStandaloneWebSocketRoutes registers routes for standalone WebSocket functions
func (g *CodeGenerator) registerStandaloneWebSocketRoutes() {
	g.writeLine("// Register standalone WebSocket function routes")
	
	for _, wsFunc := range g.webSocketFunctions {
		// Look for @SubscribeMessage decorators in standalone functions
		for _, decorator := range wsFunc.Decorators {
			if decorator.Name == "SubscribeMessage" && len(decorator.Args) > 0 {
				if eventName, ok := decorator.Args[0].Value.(string); ok {
					g.generateStandaloneRouteRegistration(wsFunc, eventName)
				} else if eventArray, ok := decorator.Args[0].Value.([]interface{}); ok {
					// Handle multiple events
					for _, event := range eventArray {
						if eventStr, ok := event.(string); ok {
							g.generateStandaloneRouteRegistration(wsFunc, eventStr)
						}
					}
				}
			}
		}
	}
}

// generateStandaloneRouteRegistration generates route registration for standalone functions
func (g *CodeGenerator) generateStandaloneRouteRegistration(wsFunc *WebSocketFunctionDeclaration, eventName string) {
	g.writeLine(fmt.Sprintf("// Register standalone route: %s -> %s", eventName, wsFunc.Name))
	
	g.writeLine(fmt.Sprintf("standaloneConfig := &websocket.RouteConfig{"))
	g.indent()
	g.writeLine(fmt.Sprintf("EventName: \"%s\",", eventName))
	g.writeLine(fmt.Sprintf("Handler: %s,", wsFunc.Name))
	g.writeLine("Gateway: \"Standalone\",")
	g.writeLine(fmt.Sprintf("Method: \"%s\",", wsFunc.Name))
	g.writeLine("IsStandalone: true,")
	
	// Add middleware configuration for standalone functions
	g.generateStandaloneRouteMiddlewareConfig(wsFunc)
	
	// Add parameter metadata for standalone functions
	g.generateStandaloneRouteParameterMetadata(wsFunc)
	
	// Add validation metadata for standalone functions
	g.generateStandaloneRouteValidationMetadata(wsFunc)
	
	g.unindent()
	g.writeLine("}")
	
	g.writeLine(fmt.Sprintf("routeRegistry.Register(\"%s\", standaloneConfig)", eventName))
	g.writeLine("")
}

// generateStandaloneRouteMiddlewareConfig generates middleware configuration for standalone functions
func (g *CodeGenerator) generateStandaloneRouteMiddlewareConfig(wsFunc *WebSocketFunctionDeclaration) {
	guards := g.getWebSocketGuardDecorators(wsFunc.Decorators)
	interceptors := g.getWebSocketInterceptorDecorators(wsFunc.Decorators)
	pipes := g.getWebSocketPipeDecorators(wsFunc.Decorators)
	filters := g.getWebSocketFilterDecorators(wsFunc.Decorators)
	
	if len(guards) > 0 {
		g.writeLine("Guards: []string{")
		g.indent()
		for i, guard := range guards {
			if i == len(guards)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", guard))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", guard))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	if len(interceptors) > 0 {
		g.writeLine("Interceptors: []string{")
		g.indent()
		for i, interceptor := range interceptors {
			if i == len(interceptors)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", interceptor))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", interceptor))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	if len(pipes) > 0 {
		g.writeLine("Pipes: []string{")
		g.indent()
		for i, pipe := range pipes {
			if i == len(pipes)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", pipe))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", pipe))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	
	if len(filters) > 0 {
		g.writeLine("Filters: []string{")
		g.indent()
		for i, filter := range filters {
			if i == len(filters)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", filter))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", filter))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
}

// generateStandaloneRouteParameterMetadata generates parameter metadata for standalone functions
func (g *CodeGenerator) generateStandaloneRouteParameterMetadata(wsFunc *WebSocketFunctionDeclaration) {
	if len(wsFunc.Params) > 0 {
		g.writeLine("Parameters: []websocket.ParameterMetadata{")
		g.indent()
		
		for i, param := range wsFunc.Params {
			paramDecorators := g.getParameterDecorators(param)
			if len(paramDecorators) > 0 {
				g.writeLine("{")
				g.indent()
				g.writeLine(fmt.Sprintf("Name: \"%s\",", param.Name))
				g.writeLine(fmt.Sprintf("Type: \"%s\",", param.Type))
				
				// Add parameter decorator information
				g.writeLine("Decorators: []string{")
				g.indent()
				for j, decorator := range paramDecorators {
					if j == len(paramDecorators)-1 {
						g.writeLine(fmt.Sprintf("\"%s\",", decorator.Name))
					} else {
						g.writeLine(fmt.Sprintf("\"%s\",", decorator.Name))
					}
				}
				g.unindent()
				g.writeLine("},")
				
				g.unindent()
				if i == len(wsFunc.Params)-1 {
					g.writeLine("},")
				} else {
					g.writeLine("},")
				}
			}
		}
		
		g.unindent()
		g.writeLine("},")
	}
}

// generateStandaloneRouteValidationMetadata generates validation metadata for standalone functions
func (g *CodeGenerator) generateStandaloneRouteValidationMetadata(wsFunc *WebSocketFunctionDeclaration) {
	// Check if function has any validation decorators
	hasValidation := false
	for _, param := range wsFunc.Params {
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
		g.writeLine("HasValidation: true,")
		g.writeLine("ValidationRules: []string{")
		g.indent()
		
		for _, param := range wsFunc.Params {
			paramDecorators := g.getParameterDecorators(param)
			for _, decorator := range paramDecorators {
				if g.isValidationDecorator(decorator.Name) {
					g.writeLine(fmt.Sprintf("\"%s:%s\",", param.Name, decorator.Name))
				}
			}
		}
		
		g.unindent()
		g.writeLine("},")
	}
}

// generateWebSocketRouteMetadata generates comprehensive route metadata
func (g *CodeGenerator) generateWebSocketRouteMetadata(gateway *WebSocketGatewayDeclaration) {
	g.writeLine("// Generate WebSocket route metadata")
	g.writeLine("metadata := websocket.RouteMetadata{")
	g.indent()
	g.writeLine(fmt.Sprintf("GatewayName: \"%s\",", gateway.Name))
	g.writeLine("Routes: routeRegistry.GetAllRoutes(),")
	g.writeLine("GeneratedAt: time.Now(),")
	
	// Add gateway configuration metadata
	wsConfig := g.getWebSocketGatewayConfig(gateway)
	g.writeLine("GatewayConfig: websocket.GatewayConfigMetadata{")
	g.indent()
	g.writeLine(fmt.Sprintf("Port: %d,", wsConfig.Port))
	if wsConfig.Namespace != "" {
		g.writeLine(fmt.Sprintf("Namespace: \"%s\",", wsConfig.Namespace))
	}
	g.writeLine(fmt.Sprintf("CORS: %t,", wsConfig.CORS != false))
	if len(wsConfig.Transports) > 0 {
		g.writeLine("Transports: []string{")
		g.indent()
		for i, transport := range wsConfig.Transports {
			if i == len(wsConfig.Transports)-1 {
				g.writeLine(fmt.Sprintf("\"%s\",", transport))
			} else {
				g.writeLine(fmt.Sprintf("\"%s\",", transport))
			}
		}
		g.unindent()
		g.writeLine("},")
	}
	g.unindent()
	g.writeLine("},")
	
	// Add route statistics
	g.generateRouteStatistics(gateway)
	
	g.unindent()
	g.writeLine("}")
	
	g.writeLine("server.SetRouteMetadata(metadata)")
}

// generateRouteStatistics generates route statistics for metadata
func (g *CodeGenerator) generateRouteStatistics(gateway *WebSocketGatewayDeclaration) {
	// Count different types of routes
	messageHandlers := 0
	lifecycleHandlers := 0
	middlewareCount := 0
	
	for _, method := range gateway.Methods {
		if g.hasDecoratorByName(method, "SubscribeMessage") {
			messageHandlers++
		}
		if g.hasWebSocketLifecycleDecorator(method) {
			lifecycleHandlers++
		}
		
		// Count middleware
		guards := g.getWebSocketGuardDecorators(method.Decorators)
		interceptors := g.getWebSocketInterceptorDecorators(method.Decorators)
		pipes := g.getWebSocketPipeDecorators(method.Decorators)
		filters := g.getWebSocketFilterDecorators(method.Decorators)
		middlewareCount += len(guards) + len(interceptors) + len(pipes) + len(filters)
	}
	
	// Count standalone functions
	standaloneHandlers := len(g.webSocketFunctions)
	
	g.writeLine("Statistics: websocket.RouteStatistics{")
	g.indent()
	g.writeLine(fmt.Sprintf("MessageHandlers: %d,", messageHandlers))
	g.writeLine(fmt.Sprintf("LifecycleHandlers: %d,", lifecycleHandlers))
	g.writeLine(fmt.Sprintf("StandaloneHandlers: %d,", standaloneHandlers))
	g.writeLine(fmt.Sprintf("TotalMiddleware: %d,", middlewareCount))
	g.writeLine(fmt.Sprintf("TotalRoutes: %d,", messageHandlers+standaloneHandlers))
	g.unindent()
	g.writeLine("},")
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
	
	// Generate parameter extraction for WebSocket context
	// Convert WebSocketFunctionDeclaration to MethodNode for parameter extraction
	method := &MethodNode{
		Name:       wsFunc.Name,
		Params:     wsFunc.Params,
		ReturnType: wsFunc.ReturnType,
		Decorators: wsFunc.Decorators,
	}
	g.generateWebSocketParameterExtraction(method)
	
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
		Port:         8080, // Default port
		CORS:         false,
		Transports:   []string{"websocket", "polling"}, // Default transports
		PingTimeout:  20000, // Default ping timeout (20s)
		PingInterval: 25000, // Default ping interval (25s)
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
			
			// Handle CORS configuration (boolean or object)
			if cors, ok := configMap["cors"].(bool); ok {
				config.CORS = cors
			} else if corsMap, ok := configMap["cors"].(map[string]interface{}); ok {
				corsConfig := WebSocketCORSConfig{}
				if origin, ok := corsMap["origin"].(string); ok {
					corsConfig.Origin = origin
				}
				if credentials, ok := corsMap["credentials"].(bool); ok {
					corsConfig.Credentials = credentials
				}
				config.CORS = corsConfig
			}
			
			// Handle transports array
			if transports, ok := configMap["transports"].([]interface{}); ok {
				config.Transports = []string{}
				for _, transport := range transports {
					if transportStr, ok := transport.(string); ok {
						config.Transports = append(config.Transports, transportStr)
					}
				}
			}
			
			// Handle ping timeout
			if pingTimeout, ok := configMap["pingTimeout"].(int); ok {
				config.PingTimeout = pingTimeout
			}
			
			// Handle ping interval
			if pingInterval, ok := configMap["pingInterval"].(int); ok {
				config.PingInterval = pingInterval
			}
		}
	}

	return config
}

// generateWebSocketCORSConfig generates CORS configuration code
func (g *CodeGenerator) generateWebSocketCORSConfig(wsConfig WebSocketConfig) {
	if corsConfig, ok := wsConfig.CORS.(WebSocketCORSConfig); ok {
		// Handle CORS as object
		g.writeLine("CORS: websocket.CORSConfig{")
		g.indent()
		if corsConfig.Origin != "" {
			g.writeLine(fmt.Sprintf("Origin: \"%s\",", corsConfig.Origin))
		}
		g.writeLine(fmt.Sprintf("Credentials: %t,", corsConfig.Credentials))
		g.unindent()
		g.writeLine("},")
	} else if corsBool, ok := wsConfig.CORS.(bool); ok && corsBool {
		// Handle CORS as boolean
		g.writeLine("CORS: true,")
	}
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
	// Set WebSocket context and current method for parameter extraction
	originalContext := g.currentContext
	originalMethod := g.currentMethod
	g.currentContext = WebSocketContext
	g.currentMethod = method
	defer func() {
		g.currentContext = originalContext
		g.currentMethod = originalMethod
	}()
	
	for _, param := range method.Params {
		paramDecorators := g.getParameterDecorators(param)
		
		for _, decorator := range paramDecorators {
			switch decorator.Name {
			case "MessageBody":
				g.generateMessageBodyParameterExtraction(param, decorator)
			case "ConnectedSocket":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client()", param.Name))
			case "MessageAck":
				g.writeLine(fmt.Sprintf("%s := wsCtx.AckCallback()", param.Name))
			case "Headers":
				// Use the full HTTP-style @Headers() logic with WebSocket context
				g.generateHeaderParameterExtraction(param, decorator)
			case "Query":
				// Use the full HTTP-style @Query() logic with WebSocket context
				g.generateQueryParameterExtraction(param, decorator)
			case "Session":
				// Use the full HTTP-style @Session() logic with WebSocket context
				g.generateSessionParameterExtraction(param, decorator)
			case "Exception":
				// WebSocket error parameter extraction
				g.writeLine(fmt.Sprintf("%s := wsCtx.Error", param.Name))
			case "EventName":
				g.writeLine(fmt.Sprintf("%s := wsCtx.EventName", param.Name))
			case "Rooms":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client.GetRooms()", param.Name))
			case "Namespace":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Namespace", param.Name))
			case "CurrentUser":
				g.writeLine(fmt.Sprintf("%s := wsCtx.User", param.Name))
			case "ClientIP":
				g.writeLine(fmt.Sprintf("%s := wsCtx.Client.RemoteAddr()", param.Name))
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
// WebSocketCORSConfig represents CORS configuration for WebSocket
type WebSocketCORSConfig struct {
	Origin      string
	Credentials bool
}

// WebSocketConfig represents configuration for WebSocket Gateway
type WebSocketConfig struct {
	Port         int
	Namespace    string
	CORS         interface{} // Can be bool or WebSocketCORSConfig
	Transports   []string
	PingTimeout  int
	PingInterval int
}

// ===================== WebSocket Middleware Generation =====================

// generateWebSocketGuardMiddlewareFunctions generates guard middleware methods for WebSocket gateways
func (g *CodeGenerator) generateWebSocketGuardMiddlewareFunctions(gateway *WebSocketGatewayDeclaration) {
	allGuards := make(map[string]bool)
	
	// Collect all unique guards from gateway and methods
	gatewayGuards := g.getWebSocketGuardDecorators(gateway.Decorators)
	for _, guard := range gatewayGuards {
		allGuards[guard] = true
	}
	
	for _, method := range gateway.Methods {
		methodGuards := g.getWebSocketGuardDecorators(method.Decorators)
		for _, guard := range methodGuards {
			allGuards[guard] = true
		}
	}
	
	// Collect from standalone WebSocket functions
	for _, wsFunc := range g.webSocketFunctions {
		funcGuards := g.getWebSocketGuardDecorators(wsFunc.Decorators)
		for _, guard := range funcGuards {
			allGuards[guard] = true
		}
	}
	
	if len(allGuards) == 0 {
		return
	}
	
	// Generate WebSocket guard middleware methods for the gateway
	g.writeLine("// WebSocket Guard middleware methods")
	
	for guard := range allGuards {
		g.generateWebSocketGuardMethod(gateway.Name, guard)
		g.writeLine("")
	}
}

// generateWebSocketInterceptorMiddlewareFunctions generates interceptor middleware methods for WebSocket gateways
func (g *CodeGenerator) generateWebSocketInterceptorMiddlewareFunctions(gateway *WebSocketGatewayDeclaration) {
	allInterceptors := make(map[string]bool)
	
	// Collect all unique interceptors from gateway and methods
	gatewayInterceptors := g.getWebSocketInterceptorDecorators(gateway.Decorators)
	for _, interceptor := range gatewayInterceptors {
		allInterceptors[interceptor] = true
	}
	
	for _, method := range gateway.Methods {
		methodInterceptors := g.getWebSocketInterceptorDecorators(method.Decorators)
		for _, interceptor := range methodInterceptors {
			allInterceptors[interceptor] = true
		}
	}
	
	// Collect from standalone WebSocket functions
	for _, wsFunc := range g.webSocketFunctions {
		funcInterceptors := g.getWebSocketInterceptorDecorators(wsFunc.Decorators)
		for _, interceptor := range funcInterceptors {
			allInterceptors[interceptor] = true
		}
	}
	
	if len(allInterceptors) == 0 {
		return
	}
	
	// Generate WebSocket interceptor middleware methods for the gateway
	g.writeLine("// WebSocket Interceptor middleware methods")
	
	for interceptor := range allInterceptors {
		g.generateWebSocketInterceptorMethod(gateway.Name, interceptor)
		g.writeLine("")
	}
}

// generateWebSocketPipeMiddlewareFunctions generates pipe middleware methods for WebSocket gateways
func (g *CodeGenerator) generateWebSocketPipeMiddlewareFunctions(gateway *WebSocketGatewayDeclaration) {
	allPipes := make(map[string]bool)
	
	// Collect all unique pipes from gateway and methods
	gatewayPipes := g.getWebSocketPipeDecorators(gateway.Decorators)
	for _, pipe := range gatewayPipes {
		allPipes[pipe] = true
	}
	
	for _, method := range gateway.Methods {
		methodPipes := g.getWebSocketPipeDecorators(method.Decorators)
		for _, pipe := range methodPipes {
			allPipes[pipe] = true
		}
	}
	
	// Collect from standalone WebSocket functions
	for _, wsFunc := range g.webSocketFunctions {
		funcPipes := g.getWebSocketPipeDecorators(wsFunc.Decorators)
		for _, pipe := range funcPipes {
			allPipes[pipe] = true
		}
	}
	
	if len(allPipes) == 0 {
		return
	}
	
	// Generate WebSocket pipe middleware methods for the gateway
	g.writeLine("// WebSocket Pipe middleware methods")
	
	for pipe := range allPipes {
		g.generateWebSocketPipeMethod(gateway.Name, pipe)
		g.writeLine("")
	}
}

// generateWebSocketFilterMiddlewareFunctions generates filter middleware methods for WebSocket gateways
func (g *CodeGenerator) generateWebSocketFilterMiddlewareFunctions(gateway *WebSocketGatewayDeclaration) {
	allFilters := make(map[string]bool)
	
	// Collect all unique filters from gateway and methods
	gatewayFilters := g.getWebSocketFilterDecorators(gateway.Decorators)
	for _, filter := range gatewayFilters {
		allFilters[filter] = true
	}
	
	for _, method := range gateway.Methods {
		methodFilters := g.getWebSocketFilterDecorators(method.Decorators)
		for _, filter := range methodFilters {
			allFilters[filter] = true
		}
	}
	
	// Collect from standalone WebSocket functions
	for _, wsFunc := range g.webSocketFunctions {
		funcFilters := g.getWebSocketFilterDecorators(wsFunc.Decorators)
		for _, filter := range funcFilters {
			allFilters[filter] = true
		}
	}
	
	if len(allFilters) == 0 {
		return
	}
	
	// Generate WebSocket filter middleware methods for the gateway
	g.writeLine("// WebSocket Filter middleware methods")
	
	for filter := range allFilters {
		g.generateWebSocketFilterMethod(gateway.Name, filter)
		g.writeLine("")
	}
}

// ===================== Helper Functions for WebSocket Middleware Decorators =====================

// getWebSocketGuardDecorators extracts WebSocket guard decorator names
func (g *CodeGenerator) getWebSocketGuardDecorators(decorators []*DecoratorNode) []string {
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

// getWebSocketInterceptorDecorators extracts WebSocket interceptor decorator names
func (g *CodeGenerator) getWebSocketInterceptorDecorators(decorators []*DecoratorNode) []string {
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

// getWebSocketPipeDecorators extracts WebSocket pipe decorator names
func (g *CodeGenerator) getWebSocketPipeDecorators(decorators []*DecoratorNode) []string {
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

// getWebSocketFilterDecorators extracts WebSocket filter decorator names (UseFilters)
func (g *CodeGenerator) getWebSocketFilterDecorators(decorators []*DecoratorNode) []string {
	var filters []string
	
	for _, decorator := range decorators {
		if decorator.Name == "UseFilters" {
			// Extract filter names from decorator arguments
			for _, arg := range decorator.Args {
				if filterName, ok := arg.Value.(string); ok {
					filters = append(filters, filterName)
				}
			}
		}
	}
	
	return filters
}

// generateWebSocketMiddlewareRegistration generates middleware registration for WebSocket gateways
func (g *CodeGenerator) generateWebSocketMiddlewareRegistration(gateway *WebSocketGatewayDeclaration) {
	// Collect all middleware from gateway and methods
	allGuards := make(map[string]bool)
	allInterceptors := make(map[string]bool)
	allPipes := make(map[string]bool)
	allFilters := make(map[string]bool)
	
	// Collect from gateway decorators
	gatewayGuards := g.getWebSocketGuardDecorators(gateway.Decorators)
	for _, guard := range gatewayGuards {
		allGuards[guard] = true
	}
	
	gatewayInterceptors := g.getWebSocketInterceptorDecorators(gateway.Decorators)
	for _, interceptor := range gatewayInterceptors {
		allInterceptors[interceptor] = true
	}
	
	gatewayPipes := g.getWebSocketPipeDecorators(gateway.Decorators)
	for _, pipe := range gatewayPipes {
		allPipes[pipe] = true
	}
	
	gatewayFilters := g.getWebSocketFilterDecorators(gateway.Decorators)
	for _, filter := range gatewayFilters {
		allFilters[filter] = true
	}
	
	// Collect from method decorators
	for _, method := range gateway.Methods {
		methodGuards := g.getWebSocketGuardDecorators(method.Decorators)
		for _, guard := range methodGuards {
			allGuards[guard] = true
		}
		
		methodInterceptors := g.getWebSocketInterceptorDecorators(method.Decorators)
		for _, interceptor := range methodInterceptors {
			allInterceptors[interceptor] = true
		}
		
		methodPipes := g.getWebSocketPipeDecorators(method.Decorators)
		for _, pipe := range methodPipes {
			allPipes[pipe] = true
		}
		
		methodFilters := g.getWebSocketFilterDecorators(method.Decorators)
		for _, filter := range methodFilters {
			allFilters[filter] = true
		}
	}
	
	// Collect from standalone WebSocket functions
	for _, wsFunc := range g.webSocketFunctions {
		funcGuards := g.getWebSocketGuardDecorators(wsFunc.Decorators)
		for _, guard := range funcGuards {
			allGuards[guard] = true
		}
		
		funcInterceptors := g.getWebSocketInterceptorDecorators(wsFunc.Decorators)
		for _, interceptor := range funcInterceptors {
			allInterceptors[interceptor] = true
		}
		
		funcPipes := g.getWebSocketPipeDecorators(wsFunc.Decorators)
		for _, pipe := range funcPipes {
			allPipes[pipe] = true
		}
		
		funcFilters := g.getWebSocketFilterDecorators(wsFunc.Decorators)
		for _, filter := range funcFilters {
			allFilters[filter] = true
		}
	}
	
	// Register middleware if any exist
	if len(allGuards) > 0 || len(allInterceptors) > 0 || len(allPipes) > 0 || len(allFilters) > 0 {
		g.writeLine("// Register middleware")
	}
	
	// Register guards
	if len(allGuards) > 0 {
		g.writeLine("// Register guards")
		for guard := range allGuards {
			g.writeLine(fmt.Sprintf("server.UseGuard(\"%s\", ws.%s)", guard, guard))
		}
		g.writeLine("")
	}
	
	// Register interceptors
	if len(allInterceptors) > 0 {
		g.writeLine("// Register interceptors")
		for interceptor := range allInterceptors {
			g.writeLine(fmt.Sprintf("server.UseInterceptor(\"%s\", ws.%s)", interceptor, interceptor))
		}
		g.writeLine("")
	}
	
	// Register pipes
	if len(allPipes) > 0 {
		g.writeLine("// Register pipes")
		for pipe := range allPipes {
			g.writeLine(fmt.Sprintf("server.UsePipe(\"%s\", ws.%s)", pipe, pipe))
		}
		g.writeLine("")
	}
	
	// Register filters
	if len(allFilters) > 0 {
		g.writeLine("// Register filters")
		for filter := range allFilters {
			g.writeLine(fmt.Sprintf("server.UseFilter(\"%s\", ws.%s)", filter, filter))
		}
		g.writeLine("")
	}
}