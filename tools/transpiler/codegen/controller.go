package codegen

import (
	"fmt"
	"strings"
)

// generateWebSocketGatewayDeclaration generates a WebSocket gateway declaration
func (g *CodeGenerator) generateWebSocketGatewayDeclaration(gateway *WebSocketGatewayDeclaration) error {
	// WebSocket gateways are similar to controllers but handle WebSocket connections
	// For now, treat them like controllers
	return g.generateControllerDeclaration(&ControllerDeclaration{
		Name:       gateway.Name,
		Fields:     gateway.Fields,
		Methods:    gateway.Methods,
		Decorators: gateway.Decorators,
	})
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