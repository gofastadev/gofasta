package transpiler

import (
	"fmt"
	"go/format"
	"path/filepath"
	"strconv"
	"strings"
)

// CodeGenerator generates Go code from Gofasta AST
type CodeGenerator struct {
	packageName       string
	imports           []string
	decoratorRegistry map[string]*DecoratorNode
	generatedCode     strings.Builder
	indentLevel       int
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(packageName string) *CodeGenerator {
	return &CodeGenerator{
		packageName:       packageName,
		imports:           []string{},
		decoratorRegistry: make(map[string]*DecoratorNode),
	}
}

// GenerateGoCode generates Go code from a Gofasta AST
func (g *CodeGenerator) GenerateGoCode(file *GofaFile) (string, error) {
	g.reset()

	// Generate package declaration
	g.writeLine(fmt.Sprintf("package %s", g.packageName))
	g.writeLine("")

	// Add validation imports if needed
	g.addValidationImportsIfNeeded(file)
	
	// Collect imports
	g.collectImports(file)

	// Generate import statements
	if len(g.imports) > 0 {
		g.writeLine("import (")
		g.indent()
		for _, imp := range g.imports {
			g.writeLine(fmt.Sprintf(`"%s"`, imp))
		}
		g.unindent()
		g.writeLine(")")
		g.writeLine("")
	}

	// Generate validation code if needed
	g.generateValidationCodeIfNeeded(file)

	// Generate declarations
	for _, decl := range file.Declarations {
		if err := g.generateDeclaration(decl); err != nil {
			return "", err
		}
		g.writeLine("")
	}

	// Format the generated code
	formatted, err := format.Source([]byte(g.generatedCode.String()))
	if err != nil {
		// Return unformatted code if formatting fails
		return g.generatedCode.String(), nil
	}

	return string(formatted), nil
}

// generateDeclaration generates code for a declaration
func (g *CodeGenerator) generateDeclaration(decl GofaDeclaration) error {
	switch d := decl.(type) {
	case *ControllerDeclaration:
		return g.generateControllerDeclaration(d)
	case *ServiceDeclaration:
		return g.generateServiceDeclaration(d)
	case *ModuleDeclaration:
		return g.generateModuleDeclaration(d)
	default:
		return fmt.Errorf("unsupported declaration type: %T", decl)
	}
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

// generateServiceDeclaration generates Go code for a service
func (g *CodeGenerator) generateServiceDeclaration(service *ServiceDeclaration) error {
	// Generate struct declaration
	g.writeLine(fmt.Sprintf("type %s struct {", service.Name))
	g.indent()

	// Generate fields with injection tags
	for _, field := range service.Fields {
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

	// Generate Initialize method if Injectable
	if g.hasDecorator(service.Decorators, "Injectable") {
		g.generateServiceInitializeMethod(service)
		g.writeLine("")
	}

	// Generate Provider factory if Injectable or Scope decorator is present
	if g.hasDecorator(service.Decorators, "Injectable") || g.hasDecorator(service.Decorators, "Scope") {
		g.generateProviderFactory(service)
		g.writeLine("")
	}

	// Generate methods
	for _, method := range service.Methods {
		if err := g.generateServiceMethod(service, method); err != nil {
			return err
		}
		g.writeLine("")
	}

	return nil
}

// generateModuleDeclaration generates Go code for a module
func (g *CodeGenerator) generateModuleDeclaration(module *ModuleDeclaration) error {
	// Generate module struct
	g.writeLine(fmt.Sprintf("type %s struct {", module.Name))
	g.indent()
	g.writeLine("core.BaseModule")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate Configure method
	g.generateModuleConfigureMethod(module)

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

// generateServiceMethod generates a regular service method
func (g *CodeGenerator) generateServiceMethod(service *ServiceDeclaration, method *MethodNode) error {
	// Generate method signature
	signature := g.generateMethodSignature(service.Name, method, false)
	g.writeLine(signature + " {")
	g.indent()

	// Generate method body placeholder
	g.writeLine("// TODO: Implement method logic")
	if method.ReturnType != "" && method.ReturnType != "void" {
		g.writeLine("return nil")
	}

	g.unindent()
	g.writeLine("}")

	return nil
}

// generateServiceInitializeMethod generates Initialize method for services
func (g *CodeGenerator) generateServiceInitializeMethod(service *ServiceDeclaration) {
	g.writeLine(fmt.Sprintf("func (s *%s) Initialize() error {", service.Name))
	g.indent()
	g.writeLine("// TODO: Add initialization logic")
	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
}

// generateProviderFactory generates a provider factory function for injectable services
func (g *CodeGenerator) generateProviderFactory(service *ServiceDeclaration) {
	factoryName := fmt.Sprintf("New%s", service.Name)
	
	// Extract scope from @Injectable decorator
	scope := g.getInjectableScope(service)
	
	// Generate factory function signature
	g.writeLine(fmt.Sprintf("func %s(container *core.DIContainer) (*%s, error) {", factoryName, service.Name))
	g.indent()
	
	// Create instance
	g.writeLine(fmt.Sprintf("instance := &%s{}", service.Name))
	g.writeLine("")
	
	// Generate dependency injection for each field
	for _, field := range service.Fields {
		g.generateFieldDependencyInjection(field)
	}
	
	// Initialize the service if it has an Initialize method
	g.writeLine("")
	g.writeLine("if initializer, ok := interface{}(instance).(interface{ Initialize() error }); ok {")
	g.indent()
	g.writeLine("if err := initializer.Initialize(); err != nil {")
	g.indent()
	g.writeLine("return nil, err")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	
	g.writeLine("")
	g.writeLine("return instance, nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Generate provider registration helper
	g.generateProviderRegistration(service, factoryName, scope)
}

// generateModuleConfigureMethod generates Configure method for modules
func (g *CodeGenerator) generateModuleConfigureMethod(module *ModuleDeclaration) {
	g.writeLine(fmt.Sprintf("func (m *%s) Configure(container *core.DIContainer) error {", module.Name))
	g.indent()

	// Extract module configuration from decorators
	moduleConfig := g.getModuleConfig(module)

	// Register providers
	if len(moduleConfig.Providers) > 0 {
		g.writeLine("// Register providers")
		for _, provider := range moduleConfig.Providers {
			// Use the generated provider registration function if it exists
			registrationFunctionName := fmt.Sprintf("Register%sProvider", provider)
			g.writeLine(fmt.Sprintf("if err := %s(container); err != nil {", registrationFunctionName))
			g.indent()
			g.writeLine("return err")
			g.unindent()
			g.writeLine("}")
		}
		g.writeLine("")
	}

	// Register controllers
	if len(moduleConfig.Controllers) > 0 {
		g.writeLine("// Register controllers")
		for _, controllerName := range moduleConfig.Controllers {
			g.writeLine(fmt.Sprintf("if err := container.RegisterController(&%s{}); err != nil {", controllerName))
			g.indent()
			g.writeLine("return err")
			g.unindent()
			g.writeLine("}")
		}
		g.writeLine("")
	}

	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
}

// Helper methods

// generateMethodSignature generates method signature
func (g *CodeGenerator) generateMethodSignature(receiverType string, method *MethodNode, isController bool) string {
	var sig strings.Builder

	// Receiver
	var receiverChar string
	if isController {
		receiverChar = "c" // Standard 'c' for controllers
	} else {
		receiverChar = "s" // Standard 's' for services
	}
	sig.WriteString(fmt.Sprintf("func (%s *%s) %s", receiverChar, receiverType, method.Name))

	// Parameters
	sig.WriteString("(")
	if isController {
		// Controllers always get HTTP context as first parameter
		sig.WriteString("ctx *httpPackage.RequestContext")
		
		// For controllers, only include non-decorated parameters in the signature
		// Decorated parameters are extracted in the method body
		nonDecoratedParams := []string{}
		for _, param := range method.Params {
			if len(param.Decorators) == 0 {
				nonDecoratedParams = append(nonDecoratedParams, fmt.Sprintf("%s %s", param.Name, param.Type))
			}
		}
		if len(nonDecoratedParams) > 0 {
			sig.WriteString(", ")
			sig.WriteString(strings.Join(nonDecoratedParams, ", "))
		}
	} else {
		// For services, include all parameters
		for i, param := range method.Params {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("%s %s", param.Name, param.Type))
		}
	}
	sig.WriteString(")")

	// Return type
	if method.ReturnType != "" && method.ReturnType != "void" {
		if isController {
			// Controllers don't return values, they write to context
		} else {
			sig.WriteString(" " + method.ReturnType)
		}
	}

	return sig.String()
}

// generateParameterExtraction generates parameter extraction from HTTP context
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
				g.writeLine(fmt.Sprintf("%s := ctx.GetParam(\"%s\")", param.Name, paramName))

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

// generateQueryParameterExtraction generates enhanced query parameter extraction with advanced features
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

// generateHeaderParameterExtraction generates enhanced header parameter extraction with advanced features
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
	
	// Handle type conversion based on parameter type and options
	g.generateHeaderTypeConversion(param, "headerValue", options)
	
	g.writeLine("")
}

// generateRequestParameterExtraction generates request object parameter extraction
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

// generateResponseParameterExtraction generates response object parameter extraction
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

// generateSessionParameterExtraction generates session data parameter extraction
func (g *CodeGenerator) generateSessionParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// For @Session(), we provide access to session data through the context
	// The parameter can be a specific session key or the entire session object
	
	// Check if a session key is specified in the decorator
	sessionKey := g.getDecoratorArgValue(decorator, 0)
	
	if sessionKey != "" {
		// Extract specific session value by key
		paramType := strings.ToLower(param.Type)
		
		// Generate variable declaration
		g.writeLine(fmt.Sprintf("var %s %s", param.Name, param.Type))
		
		// Get session value
		g.writeLine(fmt.Sprintf("if sessionValue := ctx.GetSession(\"%s\"); sessionValue != nil {", sessionKey))
		g.indent()
		
		// Handle type conversion based on parameter type
		switch {
		case strings.Contains(paramType, "string"):
			g.writeLine(fmt.Sprintf("if strValue, ok := sessionValue.(string); ok {"))
			g.indent()
			g.writeLine(fmt.Sprintf("%s = strValue", param.Name))
			g.unindent()
			g.writeLine("}")
			
		case strings.Contains(paramType, "int"):
			g.writeLine(fmt.Sprintf("if intValue, ok := sessionValue.(int); ok {"))
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
			
		case strings.Contains(paramType, "bool"):
			g.writeLine(fmt.Sprintf("if boolValue, ok := sessionValue.(bool); ok {"))
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
			
		default:
			// Handle interface{} or custom types
			g.writeLine(fmt.Sprintf("if typedValue, ok := sessionValue.(%s); ok {", param.Type))
			g.indent()
			g.writeLine(fmt.Sprintf("%s = typedValue", param.Name))
			g.unindent()
			g.writeLine("}")
		}
		
		g.unindent()
		g.writeLine("}")
		
	} else {
		// Extract entire session object
		paramType := strings.ToLower(param.Type)
		
		if strings.Contains(paramType, "session") || strings.Contains(paramType, "*session") {
			// Assign the session object directly
			g.writeLine(fmt.Sprintf("%s := ctx.GetSessionStore()", param.Name))
		} else if strings.Contains(paramType, "map") || strings.Contains(paramType, "interface") {
			// Assign the session data as map or interface
			g.writeLine(fmt.Sprintf("%s := ctx.GetAllSessionData()", param.Name))
		} else {
			// Default to session store
			g.writeLine(fmt.Sprintf("%s := ctx.GetSessionStore()", param.Name))
		}
	}
	
	g.writeLine("")
}

// generateIpParameterExtraction generates client IP address parameter extraction
func (g *CodeGenerator) generateIpParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// For @Ip(), we extract the client's IP address from the request
	// This handles various scenarios like proxies, load balancers, and direct connections
	
	// The implementation should try multiple sources in order:
	// 1. X-Forwarded-For header (most common with proxies/load balancers)
	// 2. X-Real-IP header (nginx and others)
	// 3. X-Client-IP header (some proxies)
	// 4. CF-Connecting-IP header (Cloudflare)
	// 5. Remote address from the connection
	
	g.writeLine(fmt.Sprintf("%s := ctx.GetClientIP()", param.Name))
	
	g.writeLine("")
}

// generateHostParamParameterExtraction generates host parameter extraction
func (g *CodeGenerator) generateHostParamParameterExtraction(param *ParameterNode, decorator *DecoratorNode) {
	// @HostParam() extracts host information from the request
	// This can extract various parts of the host header like:
	// - Full host (host:port)
	// - Hostname only
	// - Port only
	// - Subdomain
	
	g.writeLine(fmt.Sprintf("%s := ctx.GetHost()", param.Name))
	
	g.writeLine("")
}

// QueryParameterOptions represents options for query parameter handling
type QueryParameterOptions struct {
	DefaultValue string
	Required     bool
	Type         string // "string", "int", "bool", "array", "float"
	Separator    string // for array types, default ","
	Transform    string // "lowercase", "uppercase", "trim"
}

// HeaderParameterOptions represents options for header parameter handling
type HeaderParameterOptions struct {
	DefaultValue string
	Required     bool
	Type         string // "string", "int", "bool", "array", "float"
	Separator    string // for array types, default ","
	Transform    string // "lowercase", "uppercase", "trim"
	CaseInsensitive bool // whether header matching should be case insensitive (default: true)
}

// getQueryParameterOptions extracts query parameter options from decorator
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

// getHeaderParameterOptions extracts header parameter options from decorator
func (g *CodeGenerator) getHeaderParameterOptions(decorator *DecoratorNode) HeaderParameterOptions {
	options := HeaderParameterOptions{
		Type:            "string",
		Separator:       ",",
		CaseInsensitive: true, // Headers are case-insensitive by default per HTTP spec
	}
	
	// If there's only one string argument, it's the header name
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
			if caseInsensitive, exists := objValue["caseInsensitive"]; exists {
				if caseBool, ok := caseInsensitive.(bool); ok {
					options.CaseInsensitive = caseBool
				}
			}
		}
	}
	
	return options
}

// generateQueryTypeConversion generates type conversion code for query parameters
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

// generateHeaderTypeConversion generates type conversion code for header parameters
func (g *CodeGenerator) generateHeaderTypeConversion(param *ParameterNode, valueVar string, options HeaderParameterOptions) {
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
		// Array type - headers can have comma-separated values
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
		
	case paramType == "float64" || paramType == "float32":
		// Float conversion
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
		
	case paramType == "bool":
		// Boolean conversion
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
		// String type (default)
		g.writeLine(fmt.Sprintf("%s = %s", param.Name, valueVar))
	}
}

// generateInjectionTag generates injection struct tags
func (g *CodeGenerator) generateInjectionTag(field *FieldNode) string {
	if field.Tag != "" {
		return field.Tag
	}

	// Check for @Inject() decorator
	injectDecorator := g.getInjectDecorator(field)
	if injectDecorator != nil {
		return g.generateInjectTag(injectDecorator, field)
	}

	// Generate inject tag based on field name/type if no decorator is present
	return `inject:""`
}

// getInjectDecorator finds @Inject() decorator on a field
func (g *CodeGenerator) getInjectDecorator(field *FieldNode) *DecoratorNode {
	for _, decorator := range field.Decorators {
		if decorator.Name == "Inject" {
			return decorator
		}
	}
	return nil
}

// generateInjectTag generates inject struct tag from @Inject() decorator
func (g *CodeGenerator) generateInjectTag(decorator *DecoratorNode, field *FieldNode) string {
	// Default inject tag
	tag := `inject:""`
	
	// If no arguments, use field name as token
	if len(decorator.Args) == 0 {
		// Convert field name to lowercase for token
		token := strings.ToLower(field.Name)
		tag = fmt.Sprintf(`inject:"%s"`, token)
		return tag
	}
	
	// Process decorator arguments
	for _, arg := range decorator.Args {
		// Handle string argument (injection token)
		if tokenValue, ok := arg.Value.(string); ok {
			tag = fmt.Sprintf(`inject:"%s"`, tokenValue)
			break
		}
		
		// Handle object argument with injection configuration
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			var parts []string
			
			// Extract token
			if token, exists := objValue["token"]; exists {
				if tokenStr, ok := token.(string); ok {
					parts = append(parts, fmt.Sprintf(`inject:"%s"`, tokenStr))
				}
			} else {
				// Use field name as default token
				token := strings.ToLower(field.Name)
				parts = append(parts, fmt.Sprintf(`inject:"%s"`, token))
			}
			
			// Extract optional (for optional injection)
			if optional, exists := objValue["optional"]; exists {
				if optBool, ok := optional.(bool); ok && optBool {
					parts = append(parts, `optional:"true"`)
				}
			}
			
			// Extract scope
			if scope, exists := objValue["scope"]; exists {
				if scopeStr, ok := scope.(string); ok {
					parts = append(parts, fmt.Sprintf(`scope:"%s"`, scopeStr))
				}
			}
			
			if len(parts) > 0 {
				tag = strings.Join(parts, " ")
			}
			break
		}
	}
	
	return tag
}

// getInjectableScope extracts the scope from @Scope or @Injectable decorators
func (g *CodeGenerator) getInjectableScope(service *ServiceDeclaration) string {
	// First, check for dedicated @Scope() decorator - it takes precedence
	scopeDecorator := g.getDecorator(service.Decorators, "Scope")
	if scopeDecorator != nil {
		scope := g.getScopeFromDecorator(scopeDecorator)
		if scope != "" {
			return scope
		}
	}
	
	// Fall back to @Injectable decorator scope
	injectableDecorator := g.getDecorator(service.Decorators, "Injectable")
	if injectableDecorator != nil {
		scope := g.getScopeFromDecorator(injectableDecorator)
		if scope != "" {
			return scope
		}
	}
	
	return "singleton" // Default scope
}

// getScopeFromDecorator extracts scope value from any decorator (Injectable or Scope)
func (g *CodeGenerator) getScopeFromDecorator(decorator *DecoratorNode) string {
	// Check for scope in decorator arguments
	for _, arg := range decorator.Args {
		// Handle string argument (scope)
		if scopeValue, ok := arg.Value.(string); ok {
			return g.normalizeScopeName(scopeValue)
		}
		
		// Handle object argument with scope property
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			if scope, exists := objValue["scope"]; exists {
				if scopeStr, ok := scope.(string); ok {
					return g.normalizeScopeName(scopeStr)
				}
			}
		}
	}
	
	return ""
}

// normalizeScopeName normalizes scope names to standard values
func (g *CodeGenerator) normalizeScopeName(scope string) string {
	switch strings.ToLower(scope) {
	case "singleton", "single":
		return "singleton"
	case "transient", "prototype", "instance":
		return "transient"
	case "request", "scoped", "req":
		return "request"
	default:
		return scope // Return as-is for custom scopes
	}
}

// generateFieldDependencyInjection generates dependency injection code for a field
func (g *CodeGenerator) generateFieldDependencyInjection(field *FieldNode) {
	// Get injection configuration from @Inject decorator or default
	injectionConfig := g.getFieldInjectionConfig(field)
	
	g.writeLine(fmt.Sprintf("// Inject %s", field.Name))
	
	if injectionConfig.Optional {
		// Optional dependency - don't fail if not found
		g.writeLine(fmt.Sprintf("if dep, exists := container.GetOptional(\"%s\"); exists {", injectionConfig.Token))
		g.indent()
		g.writeLine(fmt.Sprintf("if typedDep, ok := dep.(%s); ok {", field.Type))
		g.indent()
		g.writeLine(fmt.Sprintf("instance.%s = typedDep", field.Name))
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
	} else {
		// Required dependency
		g.writeLine(fmt.Sprintf("dep, err := container.Get(\"%s\")", injectionConfig.Token))
		g.writeLine("if err != nil {")
		g.indent()
		g.writeLine(fmt.Sprintf("return nil, fmt.Errorf(\"failed to inject %s: %%w\", err)", field.Name))
		g.unindent()
		g.writeLine("}")
		g.writeLine(fmt.Sprintf("if typedDep, ok := dep.(%s); ok {", field.Type))
		g.indent()
		g.writeLine(fmt.Sprintf("instance.%s = typedDep", field.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("return nil, fmt.Errorf(\"dependency %s has wrong type, expected %s\")", injectionConfig.Token, field.Type))
		g.unindent()
		g.writeLine("}")
	}
	
	g.writeLine("")
}

// FieldInjectionConfig represents field injection configuration
type FieldInjectionConfig struct {
	Token    string
	Optional bool
	Scope    string
}

// getFieldInjectionConfig extracts injection configuration from field
func (g *CodeGenerator) getFieldInjectionConfig(field *FieldNode) FieldInjectionConfig {
	config := FieldInjectionConfig{
		Token:    strings.ToLower(field.Name), // Default to field name
		Optional: false,
		Scope:    "singleton",
	}
	
	// Check for @Inject decorator
	injectDecorator := g.getInjectDecorator(field)
	if injectDecorator == nil {
		return config
	}
	
	// Process decorator arguments
	for _, arg := range injectDecorator.Args {
		// Handle string argument (injection token)
		if tokenValue, ok := arg.Value.(string); ok {
			config.Token = tokenValue
			break
		}
		
		// Handle object argument with injection configuration
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			// Extract token
			if token, exists := objValue["token"]; exists {
				if tokenStr, ok := token.(string); ok {
					config.Token = tokenStr
				}
			}
			
			// Extract optional
			if optional, exists := objValue["optional"]; exists {
				if optBool, ok := optional.(bool); ok {
					config.Optional = optBool
				}
			}
			
			// Extract scope
			if scope, exists := objValue["scope"]; exists {
				if scopeStr, ok := scope.(string); ok {
					config.Scope = scopeStr
				}
			}
			break
		}
	}
	
	return config
}

// generateProviderRegistration generates provider registration helper function
func (g *CodeGenerator) generateProviderRegistration(service *ServiceDeclaration, factoryName, scope string) {
	registrationName := fmt.Sprintf("Register%sProvider", service.Name)
	
	g.writeLine(fmt.Sprintf("// %s registers the %s provider with the DI container", registrationName, service.Name))
	g.writeLine(fmt.Sprintf("func %s(container *core.DIContainer) error {", registrationName))
	g.indent()
	
	// Register the provider with the specified scope
	serviceToken := strings.ToLower(service.Name)
	
	switch scope {
	case "singleton":
		g.writeLine(fmt.Sprintf("return container.RegisterSingleton(\"%s\", %s)", serviceToken, factoryName))
	case "transient":
		g.writeLine(fmt.Sprintf("return container.RegisterTransient(\"%s\", %s)", serviceToken, factoryName))
	case "request", "scoped":
		g.writeLine(fmt.Sprintf("return container.RegisterScoped(\"%s\", %s)", serviceToken, factoryName))
	default:
		// Default to singleton
		g.writeLine(fmt.Sprintf("return container.RegisterSingleton(\"%s\", %s)", serviceToken, factoryName))
	}
	
	g.unindent()
	g.writeLine("}")
}

// collectImports collects all necessary imports
func (g *CodeGenerator) collectImports(file *GofaFile) {
	// Standard imports for Gofasta
	g.addImport("github.com/healtronlabs/gofasta/packages/core")
	g.addImport("github.com/healtronlabs/gofasta/packages/http")
	
	// Standard Go library imports for query parameter handling
	g.addImport("strconv")
	g.addImport("strings")
	g.addImport("net/http")
	g.addImport("fmt")

	// Check if we need additional imports based on decorators
	for _, decl := range file.Declarations {
		g.collectImportsFromDeclaration(decl)
	}
}

// collectImportsFromDeclaration collects imports from a declaration
func (g *CodeGenerator) collectImportsFromDeclaration(decl GofaDeclaration) {
	switch d := decl.(type) {
	case *ControllerDeclaration:
		// Controllers need HTTP package
		g.addImport("github.com/healtronlabs/gofasta/packages/http")

	case *ServiceDeclaration:
		// Check for database-related imports
		for _, field := range d.Fields {
			if strings.Contains(field.Type, "Repository") {
				g.addImport("github.com/healtronlabs/gofasta/packages/orm")
			}
		}

	case *ModuleDeclaration:
		// Modules need core package
		g.addImport("github.com/healtronlabs/gofasta/packages/core")
	}
}

// addImport adds an import if not already present
func (g *CodeGenerator) addImport(imp string) {
	for _, existing := range g.imports {
		if existing == imp {
			return
		}
	}
	g.imports = append(g.imports, imp)
}

// Decorator helper methods

// hasDecorator checks if a decorator exists in a list
func (g *CodeGenerator) hasDecorator(decorators []*DecoratorNode, name string) bool {
	for _, decorator := range decorators {
		if decorator.Name == name {
			return true
		}
	}
	return false
}

// getDecorator finds a decorator by name
func (g *CodeGenerator) getDecorator(decorators []*DecoratorNode, name string) *DecoratorNode {
	for _, decorator := range decorators {
		if decorator.Name == name {
			return decorator
		}
	}
	return nil
}

// getDecoratorArgValue gets decorator argument value
func (g *CodeGenerator) getDecoratorArgValue(decorator *DecoratorNode, index int) string {
	if index < len(decorator.Args) {
		if val, ok := decorator.Args[index].Value.(string); ok {
			return val
		}
	}
	return ""
}

// getControllerPath gets the controller base path with version prefix
func (g *CodeGenerator) getControllerPath(controller *ControllerDeclaration) string {
	var basePath string
	
	// Debug: uncomment to see decorator parsing
	// fmt.Printf("DEBUG: Controller %s has %d decorators:\n", controller.Name, len(controller.Decorators))
	// for i, decorator := range controller.Decorators {
	//     fmt.Printf("  Decorator %d: Name='%s' Args=%d\n", i, decorator.Name, len(decorator.Args))
	// }
	
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
		
		// fmt.Printf("DEBUG: Found @Version decorator with value: '%s'\n", version)
		
		// Normalize version to lowercase for consistency (industry best practice)
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
		// fmt.Printf("DEBUG: Final versioned path: '%s'\n", basePath)
	}
	return basePath
}

// RouteInfo represents route information
type RouteInfo struct {
	Method string
	Path   string
}

// getRouteInfo extracts route information from method decorators
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

// getParameterDecorators gets decorators for a parameter
func (g *CodeGenerator) getParameterDecorators(param *ParameterNode) []*DecoratorNode {
	return param.Decorators
}

// ModuleConfig represents module configuration
type ModuleConfig struct {
	Controllers []string
	Providers   []string
	Imports     []string
	Exports     []string
}

// CatchFilterConfig represents configuration for @Catch() decorators
type CatchFilterConfig struct {
	ErrorTypes []string // The error types this filter catches
	Scope      string   // "method", "controller", or "global"
	Handler    string   // The handler method name
}

// getModuleConfig extracts module configuration from decorators
func (g *CodeGenerator) getModuleConfig(module *ModuleDeclaration) ModuleConfig {
	config := ModuleConfig{}

	moduleDecorator := g.getDecorator(module.Decorators, "Module")
	if moduleDecorator == nil {
		return config
	}

	// Extract configuration from decorator arguments
	for _, arg := range moduleDecorator.Args {
		// Handle object literal argument like { controllers: [...], providers: [...] }
		if objectValue, ok := arg.Value.(map[string]interface{}); ok {
			if controllersValue, exists := objectValue["controllers"]; exists {
				if controllers, ok := controllersValue.([]interface{}); ok {
					for _, controller := range controllers {
						if controllerStr, ok := controller.(string); ok {
							config.Controllers = append(config.Controllers, controllerStr)
						}
					}
				}
			}
			if providersValue, exists := objectValue["providers"]; exists {
				if providers, ok := providersValue.([]interface{}); ok {
					for _, provider := range providers {
						if providerStr, ok := provider.(string); ok {
							config.Providers = append(config.Providers, providerStr)
						}
					}
				}
			}
			if importsValue, exists := objectValue["imports"]; exists {
				if imports, ok := importsValue.([]interface{}); ok {
					for _, imp := range imports {
						if impStr, ok := imp.(string); ok {
							config.Imports = append(config.Imports, impStr)
						}
					}
				}
			}
			if exportsValue, exists := objectValue["exports"]; exists {
				if exports, ok := exportsValue.([]interface{}); ok {
					for _, exp := range exports {
						if expStr, ok := exp.(string); ok {
							config.Exports = append(config.Exports, expStr)
						}
					}
				}
			}
		} else if arg.Key == "controllers" {
			// Handle named arguments (fallback)
			if controllers, ok := arg.Value.([]string); ok {
				config.Controllers = controllers
			}
		} else if arg.Key == "providers" {
			// Handle named arguments (fallback)
			if providers, ok := arg.Value.([]string); ok {
				config.Providers = providers
			}
		}
	}

	return config
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

// generateCatchHandlers generates error filter handlers for controllers
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

// generateCatchHandler generates a single catch handler method
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

// getCatchFilterConfig extracts configuration from @Catch() decorator
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

// generateHttpStatusCode generates HTTP status code setting if @HttpCode() decorator is present
func (g *CodeGenerator) generateHttpStatusCode(method *MethodNode) {
	statusCode := g.getHttpStatusCode(method)
	if statusCode > 0 {
		g.writeLine(fmt.Sprintf("ctx.Status(%d)", statusCode))
		g.writeLine("")
	}
}

// getHttpStatusCode extracts the status code from @HttpCode() decorator
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

// hasRedirectDecorator checks if method has @Redirect() decorator
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

// generateResponseHeaders generates response header setting code for @Header() decorators
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

// getHeaderDecorators extracts all @Header() decorators from a method
func (g *CodeGenerator) getHeaderDecorators(method *MethodNode) []*DecoratorNode {
	var headerDecorators []*DecoratorNode
	for _, decorator := range method.Decorators {
		if decorator.Name == "Header" {
			headerDecorators = append(headerDecorators, decorator)
		}
	}
	return headerDecorators
}

// Writing helper methods

// reset resets the generator state
func (g *CodeGenerator) reset() {
	g.generatedCode.Reset()
	g.indentLevel = 0
	g.imports = []string{}
}

// writeLine writes a line with proper indentation
func (g *CodeGenerator) writeLine(line string) {
	for i := 0; i < g.indentLevel; i++ {
		g.generatedCode.WriteString("\t")
	}
	g.generatedCode.WriteString(line)
	g.generatedCode.WriteString("\n")
}

// indent increases indentation level
func (g *CodeGenerator) indent() {
	g.indentLevel++
}

// unindent decreases indentation level
func (g *CodeGenerator) unindent() {
	if g.indentLevel > 0 {
		g.indentLevel--
	}
}

// generateGuardsMiddleware generates guard middleware chain for method and controller
func (g *CodeGenerator) generateGuardsMiddleware(controller *ControllerDeclaration, method *MethodNode) string {
	var guards []string
	
	// Collect controller-level guards first
	controllerGuards := g.getGuardDecorators(controller.Decorators)
	guards = append(guards, controllerGuards...)
	
	// Collect method-level guards (these take precedence/are applied after controller guards)
	methodGuards := g.getGuardDecorators(method.Decorators)
	guards = append(guards, methodGuards...)
	
	if len(guards) == 0 {
		return ""
	}
	
	// Create middleware chain - generate the middleware instance calls
	var middlewareChain []string
	for _, guard := range guards {
		// Convert guard name to middleware function call
		middlewareChain = append(middlewareChain, fmt.Sprintf("c.%s", guard))
	}
	
	return strings.Join(middlewareChain, ", ")
}

// getGuardDecorators extracts guard names from @UseGuards() decorators
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

// generateGuardMiddlewareFunctions generates guard middleware function implementations
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

// generateRoleGuardLogic generates role-based guard logic
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

// generatePermissionGuardLogic generates permission-based guard logic
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

// generateGenericGuardLogic generates generic guard logic for custom guards
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

// generateMiddlewareChain generates combined middleware chain for pipes, interceptors, and guards
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

// getInterceptorDecorators extracts interceptor names from @UseInterceptors() decorators
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

// generateInterceptorMiddlewareFunctions generates interceptor middleware function implementations
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

// generateControllerInterceptorMethod generates an interceptor middleware method for the controller
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
	g.writeLine("// Store start time for response logging")
	g.writeLine("ctx.Set(\"request_start_time\", startTime)")
	g.writeLine("")
	g.writeLine("// Add response logging after request completes")
	g.writeLine("ctx.OnFinished(func() {")
	g.indent()
	g.writeLine("duration := time.Since(startTime)")
	g.writeLine("status := ctx.GetStatusCode()")
	g.writeLine("fmt.Printf(\"[%s] %s %s %d - %v\\n\", time.Now().Format(time.RFC3339), method, path, status, duration)")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
}

// generateCacheInterceptorLogic generates caching interceptor logic
func (g *CodeGenerator) generateCacheInterceptorLogic() {
	g.writeLine("// Cache interceptor logic")
	g.writeLine("cacheKey := fmt.Sprintf(\"%s:%s\", ctx.GetMethod(), ctx.GetPath())")
	g.writeLine("queryParams := ctx.GetQuery()")
	g.writeLine("if len(queryParams) > 0 {")
	g.indent()
	g.writeLine("cacheKey += \"?\" + queryParams")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check cache for existing response")
	g.writeLine("if cachedResponse := getFromCache(cacheKey); cachedResponse != nil {")
	g.indent()
	g.writeLine("ctx.JSON(200, cachedResponse)")
	g.writeLine("return // Skip further processing")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Store cache key for response caching")
	g.writeLine("ctx.Set(\"cache_key\", cacheKey)")
	g.writeLine("")
	g.writeLine("// Cache response after request completes (for successful responses)")
	g.writeLine("ctx.OnFinished(func() {")
	g.indent()
	g.writeLine("status := ctx.GetStatusCode()")
	g.writeLine("if status >= 200 && status < 300 {")
	g.indent()
	g.writeLine("responseData := ctx.GetResponseData()")
	g.writeLine("cacheResponse(cacheKey, responseData, 300) // Cache for 5 minutes")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
}

// generateTransformInterceptorLogic generates data transformation interceptor logic
func (g *CodeGenerator) generateTransformInterceptorLogic() {
	g.writeLine("// Transform interceptor logic")
	g.writeLine("// Transform request data before processing")
	g.writeLine("if requestBody := ctx.GetRequestBody(); requestBody != nil {")
	g.indent()
	g.writeLine("// Apply request transformations (e.g., data normalization, validation)")
	g.writeLine("transformedBody := transformRequestData(requestBody)")
	g.writeLine("ctx.SetRequestBody(transformedBody)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Set up response transformation")
	g.writeLine("ctx.OnBeforeResponse(func() {")
	g.indent()
	g.writeLine("// Transform response data before sending")
	g.writeLine("responseData := ctx.GetResponseData()")
	g.writeLine("if responseData != nil {")
	g.indent()
	g.writeLine("transformedResponse := transformResponseData(responseData)")
	g.writeLine("ctx.SetResponseData(transformedResponse)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
}

// generateValidationInterceptorLogic generates validation interceptor logic
func (g *CodeGenerator) generateValidationInterceptorLogic() {
	g.writeLine("// Validation interceptor logic")
	g.writeLine("// Validate request headers")
	g.writeLine("if contentType := ctx.GetHeader(\"Content-Type\"); contentType == \"\" {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Content-Type header is required\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Validate request body structure")
	g.writeLine("if requestBody := ctx.GetRequestBody(); requestBody != nil {")
	g.indent()
	g.writeLine("if !isValidRequestStructure(requestBody) {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Invalid request body structure\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Add response validation")
	g.writeLine("ctx.OnBeforeResponse(func() {")
	g.indent()
	g.writeLine("responseData := ctx.GetResponseData()")
	g.writeLine("if responseData != nil && !isValidResponseStructure(responseData) {")
	g.indent()
	g.writeLine("// Log validation error and send generic error response")
	g.writeLine("fmt.Printf(\"Warning: Response validation failed for %s %s\\n\", ctx.GetMethod(), ctx.GetPath())")
	g.writeLine("ctx.JSON(500, map[string]string{\"error\": \"Internal server error\"})")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("})")
	g.writeLine("")
}

// generateGenericInterceptorLogic generates generic interceptor logic for custom interceptors
func (g *CodeGenerator) generateGenericInterceptorLogic(interceptorName string) {
	g.writeLine(fmt.Sprintf("// %s interceptor logic", interceptorName))
	g.writeLine("// TODO: Implement your custom interceptor logic here")
	g.writeLine("")
	g.writeLine("// Example: Pre-processing")
	g.writeLine("// requestData := ctx.GetRequestData()")
	g.writeLine("// if !validateRequest(requestData) {")
	g.writeLine("//     ctx.JSON(400, map[string]string{\"error\": \"Invalid request\"})")
	g.writeLine("//     ctx.Abort()")
	g.writeLine("//     return")
	g.writeLine("// }")
	g.writeLine("")
	g.writeLine("// Example: Post-processing")
	g.writeLine("// ctx.OnFinished(func() {")
	g.writeLine("//     // Perform cleanup or logging")
	g.writeLine("//     fmt.Printf(\"Request completed: %s %s\\n\", ctx.GetMethod(), ctx.GetPath())")
	g.writeLine("// })")
	g.writeLine("")
}

// getPipeDecorators extracts pipe names from @UsePipes() decorators
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

// generatePipeMiddlewareFunctions generates pipe middleware function implementations
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

// generateControllerPipeMethod generates a pipe middleware method for the controller
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
	g.writeLine("// Validate request body structure and data types")
	g.writeLine("")
	g.writeLine("// Get request body")
	g.writeLine("requestBody := ctx.GetRequestBody()")
	g.writeLine("if requestBody == nil {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Request body is required\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Validate content type")
	g.writeLine("contentType := ctx.GetHeader(\"Content-Type\")")
	g.writeLine("if !strings.Contains(contentType, \"application/json\") {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": \"Content-Type must be application/json\"})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Perform validation (implement your validation logic)")
	g.writeLine("if err := validateRequestBody(requestBody); err != nil {")
	g.indent()
	g.writeLine("ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Validation failed: %s\", err.Error())})")
	g.writeLine("ctx.Abort()")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// generateTransformPipeLogic generates transform pipe logic
func (g *CodeGenerator) generateTransformPipeLogic() {
	g.writeLine("// Transform pipe logic")
	g.writeLine("// Transform incoming data before processing")
	g.writeLine("")
	g.writeLine("// Get request data")
	g.writeLine("requestData := ctx.GetRequestData()")
	g.writeLine("if requestData != nil {")
	g.indent()
	g.writeLine("// Apply transformations (implement your transformation logic)")
	g.writeLine("transformedData := transformRequestData(requestData)")
	g.writeLine("ctx.SetRequestData(transformedData)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Transform query parameters")
	g.writeLine("queryParams := ctx.GetQueryParams()")
	g.writeLine("for key, value := range queryParams {")
	g.indent()
	g.writeLine("// Example: Convert string values to appropriate types")
	g.writeLine("transformedValue := transformQueryValue(key, value)")
	g.writeLine("ctx.SetQueryParam(key, transformedValue)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// generateParseIntPipeLogic generates parse int pipe logic
func (g *CodeGenerator) generateParseIntPipeLogic() {
	g.writeLine("// ParseInt pipe logic")
	g.writeLine("// Parse string parameters to integers")
	g.writeLine("")
	g.writeLine("// Parse path parameters")
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
	g.writeLine("ctx.Abort()")
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
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// generateParseBoolPipeLogic generates parse bool pipe logic  
func (g *CodeGenerator) generateParseBoolPipeLogic() {
	g.writeLine("// ParseBool pipe logic")
	g.writeLine("// Parse string parameters to booleans")
	g.writeLine("")
	g.writeLine("// Parse query parameters")
	g.writeLine("queryParams := ctx.GetQueryParams()")
	g.writeLine("for key, value := range queryParams {")
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
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// generateParseArrayPipeLogic generates parse array pipe logic
func (g *CodeGenerator) generateParseArrayPipeLogic() {
	g.writeLine("// ParseArray pipe logic")
	g.writeLine("// Parse comma-separated strings to arrays")
	g.writeLine("")
	g.writeLine("// Parse query parameters")
	g.writeLine("queryParams := ctx.GetQueryParams()")
	g.writeLine("for key, value := range queryParams {")
	g.indent()
	g.writeLine("if strValue, ok := value.(string); ok && strings.Contains(strValue, \",\") {")
	g.indent()
	g.writeLine("arrayValue := strings.Split(strValue, \",\")")
	g.writeLine("// Trim whitespace from each element")
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
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// generateDefaultValuePipeLogic generates default value pipe logic
func (g *CodeGenerator) generateDefaultValuePipeLogic() {
	g.writeLine("// DefaultValue pipe logic")
	g.writeLine("// Set default values for missing parameters")
	g.writeLine("")
	g.writeLine("// Set default query parameters")
	g.writeLine("queryParams := ctx.GetQueryParams()")
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
	g.writeLine("if _, exists := queryParams[key]; !exists {")
	g.indent()
	g.writeLine("ctx.SetQueryParam(key, defaultValue)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// generateGenericPipeLogic generates generic pipe logic for custom pipes
func (g *CodeGenerator) generateGenericPipeLogic(pipeName string) {
	g.writeLine(fmt.Sprintf("// %s pipe logic", pipeName))
	g.writeLine("// TODO: Implement your custom pipe logic here")
	g.writeLine("")
	g.writeLine("// Example: Data validation")
	g.writeLine("// requestData := ctx.GetRequestData()")
	g.writeLine("// if err := validateData(requestData); err != nil {")
	g.writeLine("//     ctx.JSON(400, map[string]string{\"error\": err.Error()})")
	g.writeLine("//     ctx.Abort()")
	g.writeLine("//     return")
	g.writeLine("// }")
	g.writeLine("")
	g.writeLine("// Example: Data transformation")
	g.writeLine("// transformedData := transformData(requestData)")
	g.writeLine("// ctx.SetRequestData(transformedData)")
	g.writeLine("")
	g.writeLine("// Continue to next middleware/handler")
	g.writeLine("ctx.Next()")
}

// TranspileFile is the main entry point for transpiling a .gofa file to .go
func TranspileFile(inputPath string, inputContent string) (string, error) {
	// Parse the .gofa file
	file, err := ParseGofaFile(inputContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse .gofa file: %w", err)
	}

	// Use package name from parsed file, fallback to directory name
	var packageName string
	if file.Package != nil && file.Package.Name != "" {
		packageName = file.Package.Name
	} else {
		// Fallback to directory-based package name
		packageName = filepath.Base(filepath.Dir(inputPath))
		if packageName == "." {
			packageName = "main"
		}
	}

	// Generate Go code
	generator := NewCodeGenerator(packageName)
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		return "", fmt.Errorf("failed to generate Go code: %w", err)
	}

	return goCode, nil
}

// ======= VALIDATION DECORATOR CODE GENERATION =======

// ValidationStructInfo represents information about a struct that needs validation
type ValidationStructInfo struct {
	Name   string
	Fields []*ValidationFieldInfo
}

// ValidationFieldInfo represents information about a field that needs validation
type ValidationFieldInfo struct {
	Name       string
	Type       string
	Tag        string
	Validators []ValidationRule
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Type    string
	Args    []interface{}
	Message string
	Code    string
}

// generateValidationCode generates validation functions for DTOs with validation decorators
func (g *CodeGenerator) generateValidationCode(file *GofaFile) {
	dtoStructs := g.findDTOStructsWithValidation(file)
	
	if len(dtoStructs) == 0 {
		return
	}
	
	// Generate ValidationError struct first
	g.generateValidationErrorStruct()
	g.writeLine("")
	
	// Generate helper validation functions
	g.generateValidationHelperFunctions()
	g.writeLine("")
	
	// Generate validation functions for each DTO
	for _, dto := range dtoStructs {
		g.generateDTOValidationFunction(dto)
		g.writeLine("")
	}
}

// findDTOStructsWithValidation finds all structs that have validation decorators
func (g *CodeGenerator) findDTOStructsWithValidation(file *GofaFile) map[string]*ValidationStructInfo {
	dtos := make(map[string]*ValidationStructInfo)
	
	// Look for standalone type declarations (DTOs) that have validation decorators
	for _, decl := range file.Declarations {
		switch d := decl.(type) {
		case *ControllerDeclaration:
			structInfo := g.extractValidationFromStruct(d.Name, d.Fields)
			if structInfo != nil && len(structInfo.Fields) > 0 {
				dtos[d.Name] = structInfo
			}
		case *ServiceDeclaration:
			structInfo := g.extractValidationFromStruct(d.Name, d.Fields)
			if structInfo != nil && len(structInfo.Fields) > 0 {
				dtos[d.Name] = structInfo
			}
		}
	}
	
	return dtos
}

// extractValidationFromStruct extracts validation information from struct fields
func (g *CodeGenerator) extractValidationFromStruct(structName string, fields []*FieldNode) *ValidationStructInfo {
	var validationFields []*ValidationFieldInfo
	
	for _, field := range fields {
		if g.hasValidationDecorators(field) {
			validators := g.parseValidationDecoratorsFromField(field)
			if len(validators) > 0 {
				validationField := &ValidationFieldInfo{
					Name:       field.Name,
					Type:       field.Type,
					Tag:        field.Tag,
					Validators: validators,
				}
				validationFields = append(validationFields, validationField)
			}
		}
	}
	
	if len(validationFields) == 0 {
		return nil
	}
	
	return &ValidationStructInfo{
		Name:   structName,
		Fields: validationFields,
	}
}

// hasValidationDecorators checks if a field has validation decorators
func (g *CodeGenerator) hasValidationDecorators(field *FieldNode) bool {
	for _, decorator := range field.Decorators {
		if IsValidationDecorator(GetDecoratorType(decorator.Name)) {
			return true
		}
	}
	return false
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

// convertDecoratorArgsToInterface converts decorator arguments to interface{} slice
func (g *CodeGenerator) convertDecoratorArgsToInterface(args []DecoratorArg) []interface{} {
	var result []interface{}
	for _, arg := range args {
		result = append(result, arg.Value)
	}
	return result
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

// extractTagContent extracts content from a struct tag
func (g *CodeGenerator) extractTagContent(tag, key string) string {
	// Extract content between quotes after key:
	// `validate:"@IsEmail() @Min(18)"` -> "@IsEmail() @Min(18)"
	keyPattern := key + `:"` 
	startIndex := strings.Index(tag, keyPattern)
	if startIndex == -1 {
		return ""
	}
	
	startIndex += len(keyPattern)
	endIndex := strings.Index(tag[startIndex:], `"`)
	if endIndex == -1 {
		return ""
	}
	
	return tag[startIndex : startIndex+endIndex]
}

// extractDecorators extracts individual decorators from validation content
func (g *CodeGenerator) extractDecorators(content string) []string {
	var decorators []string
	content = strings.TrimSpace(content)
	
	// Split by @ but keep the @
	parts := strings.Split(content, "@")
	for i, part := range parts {
		if i == 0 && part == "" {
			continue // Skip empty first part
		}
		if part != "" {
			decorators = append(decorators, "@"+part)
		}
	}
	
	return decorators
}

// parseValidationRule parses a single validation rule like @IsEmail() or @Min(18)
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
	
	// Extract name and arguments
	name := decorator[:parenIndex]
	argsStr := decorator[parenIndex+1:]
	
	// Remove closing parenthesis
	if strings.HasSuffix(argsStr, ")") {
		argsStr = argsStr[:len(argsStr)-1]
	}
	
	// Parse arguments
	args := g.parseValidationArgs(argsStr)
	
	return &ValidationRule{
		Type:    name,
		Args:    args,
		Message: g.getValidationMessage(name, args),
		Code:    g.getValidationCode(name),
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
		
		if floatVal, err := strconv.ParseFloat(part, 64); err == nil {
			args = append(args, floatVal)
			continue
		}
		
		// Remove quotes if present
		if (strings.HasPrefix(part, `"`) && strings.HasSuffix(part, `"`)) ||
		   (strings.HasPrefix(part, `'`) && strings.HasSuffix(part, `'`)) {
			part = part[1 : len(part)-1]
		}
		
		args = append(args, part)
	}
	
	return args
}

// getValidationMessage gets the validation error message for a rule
func (g *CodeGenerator) getValidationMessage(ruleType string, args []interface{}) string {
	switch ruleType {
	case "IsEmail":
		return "must be a valid email address"
	case "IsURL":
		return "must be a valid URL"
	case "IsNotEmpty":
		return "must not be empty"
	case "Min":
		if len(args) > 0 {
			return fmt.Sprintf("must be at least %v", args[0])
		}
		return "value too small"
	case "Max":
		if len(args) > 0 {
			return fmt.Sprintf("must be at most %v", args[0])
		}
		return "value too large"
	case "Length":
		if len(args) >= 2 {
			return fmt.Sprintf("must be between %v and %v characters", args[0], args[1])
		}
		return "invalid length"
	case "IsArray":
		return "must be an array"
	case "ArrayMinSize":
		if len(args) > 0 {
			return fmt.Sprintf("must contain at least %v item(s)", args[0])
		}
		return "array too small"
	case "ArrayMaxSize":
		if len(args) > 0 {
			return fmt.Sprintf("must contain at most %v item(s)", args[0])
		}
		return "array too large"
	case "ArrayNotEmpty":
		return "must not be empty"
	case "IsEmpty":
		return "must be empty"
	case "IsString":
		return "must be a string"
	case "IsNumber":
		return "must be a number"
	case "IsInt":
		return "must be an integer"
	case "IsFloat":
		return "must be a floating point number"
	case "IsBoolean":
		return "must be a boolean"
	case "IsDate":
		return "must be a valid date"
	case "IsIP":
		return "must be a valid IP address"
	case "IsJSON":
		return "must be valid JSON"
	case "IsHexColor":
		return "must be a valid hex color"
	case "IsPhoneNumber":
		return "must be a valid phone number"
	case "IsCreditCard":
		return "must be a valid credit card number"
	case "IsISBN":
		return "must be a valid ISBN"
	case "IsBase64":
		return "must be valid Base64"
	case "MinLength":
		if len(args) > 0 {
			return fmt.Sprintf("must be at least %v characters long", args[0])
		}
		return "too short"
	case "MaxLength":
		if len(args) > 0 {
			return fmt.Sprintf("must be at most %v characters long", args[0])
		}
		return "too long"
	case "IsPositive":
		return "must be a positive number"
	case "IsNegative":
		return "must be a negative number"
	case "Matches":
		if len(args) > 0 {
			return fmt.Sprintf("must match pattern %v", args[0])
		}
		return "must match specified pattern"
	case "IsAlpha":
		return "must contain only letters"
	case "IsAlphanumeric":
		return "must contain only letters and numbers"
	case "IsNumeric":
		return "must contain only numbers"
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
		return "must contain specified substring"
	case "NotContains":
		if len(args) > 0 {
			return fmt.Sprintf("must not contain %v", args[0])
		}
		return "must not contain specified substring"
	case "IsIn":
		return "must be one of the allowed values"
	case "IsNotIn":
		return "must not be one of the forbidden values"
	case "IsLowercase":
		return "must be lowercase"
	case "IsUppercase":
		return "must be uppercase"
	case "ValidateNested":
		return "nested validation failed"
	case "ValidateIf":
		return "conditional validation failed"
	case "Custom":
		return "custom validation failed"
	case "IsPastDate":
		return "must be a date in the past"
	case "IsFutureDate":
		return "must be a date in the future"
	default:
		return fmt.Sprintf("%s validation failed", ruleType)
	}
}

// getValidationCode gets the validation error code for a rule
func (g *CodeGenerator) getValidationCode(ruleType string) string {
	// Handle specific cases for expected test values
	switch ruleType {
	case "Min":
		return "MIN_VALUE"
	case "Max":
		return "MAX_VALUE"
	case "IsPositive":
		return "IS_POSITIVE"
	case "IsNegative":
		return "IS_NEGATIVE"
	case "IsNotEmpty":
		return "IS_NOT_EMPTY"
	case "ArrayMinSize":
		return "ARRAY_MIN_SIZE"
	case "ArrayMaxSize":
		return "ARRAY_MAX_SIZE"
	case "ArrayNotEmpty":
		return "ARRAY_NOT_EMPTY"
	case "IsEmpty":
		return "IS_EMPTY"
	case "IsURL":
		return "IS_URL"
	case "IsNumeric":
		return "IS_NUMERIC"
	case "IsAlphanumeric":
		return "IS_ALPHANUMERIC"
	case "IsAlpha":
		return "IS_ALPHA"
	case "IsIP":
		return "IS_IP"
	case "IsJSON":
		return "IS_JSON"
	case "IsISBN":
		return "IS_ISBN"
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
	case "ValidateNested":
		return "VALIDATE_NESTED"
	case "ValidateIf":
		return "VALIDATE_IF"
	case "Custom":
		return "CUSTOM"
	case "IsPastDate":
		return "IS_PAST_DATE"
	case "IsFutureDate":
		return "IS_FUTURE_DATE"
	default:
		// Convert CamelCase to SNAKE_CASE for other cases
		var result strings.Builder
		for i, r := range ruleType {
			if i > 0 && (r >= 'A' && r <= 'Z') {
				result.WriteRune('_')
			}
			result.WriteRune(r)
		}
		return strings.ToUpper(result.String())
	}
}

// generateValidationErrorStruct generates the ValidationError struct
func (g *CodeGenerator) generateValidationErrorStruct() {
	g.writeLine("// ValidationError represents a validation error")
	g.writeLine("type ValidationError struct {")
	g.indent()
	g.writeLine("Field   string      `json:\"field\"`")
	g.writeLine("Value   interface{} `json:\"value\"`") 
	g.writeLine("Message string      `json:\"message\"`")
	g.writeLine("Code    string      `json:\"code\"`")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	g.writeLine("// ValidationResult represents the result of validation")
	g.writeLine("type ValidationResult struct {")
	g.indent()
	g.writeLine("IsValid bool              `json:\"isValid\"`")
	g.writeLine("Errors  []ValidationError `json:\"errors,omitempty\"`")
	g.unindent()
	g.writeLine("}")
}

// generateValidationHelperFunctions generates helper validation functions
func (g *CodeGenerator) generateValidationHelperFunctions() {
	g.writeLine("// Validation helper functions")
	g.writeLine("")
	
	// Email validation
	g.writeLine("func isValidEmail(email string) bool {")
	g.indent()
	g.writeLine("emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$`")
	g.writeLine("matched, _ := regexp.MatchString(emailRegex, email)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// URL validation
	g.writeLine("func isValidURL(url string) bool {")
	g.indent()
	g.writeLine("_, err := url.Parse(url)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// UUID validation
	g.writeLine("func isValidUUID(uuid string) bool {")
	g.indent()
	g.writeLine("uuidRegex := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`")
	g.writeLine("matched, _ := regexp.MatchString(uuidRegex, uuid)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Numeric validation
	g.writeLine("func isNumeric(str string) bool {")
	g.indent()
	g.writeLine("numericRegex := `^[0-9]+$`")
	g.writeLine("matched, _ := regexp.MatchString(numericRegex, str)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Alpha validation
	g.writeLine("func isAlpha(str string) bool {")
	g.indent()
	g.writeLine("alphaRegex := `^[a-zA-Z]+$`")
	g.writeLine("matched, _ := regexp.MatchString(alphaRegex, str)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Alphanumeric validation
	g.writeLine("func isAlphanumeric(str string) bool {")
	g.indent()
	g.writeLine("alphanumericRegex := `^[a-zA-Z0-9]+$`")
	g.writeLine("matched, _ := regexp.MatchString(alphanumericRegex, str)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Integer validation
	g.writeLine("func isInt(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case int, int8, int16, int32, int64:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case uint, uint8, uint16, uint32, uint64:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case string:")
	g.indent()
	g.writeLine("_, err := strconv.Atoi(v)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("case float32, float64:")
	g.indent()
	g.writeLine("// Check if float is actually an integer (no decimal part)")
	g.writeLine("if f, ok := value.(float64); ok {")
	g.indent()
	g.writeLine("return f == float64(int64(f))")
	g.unindent()
	g.writeLine("}")
	g.writeLine("if f, ok := value.(float32); ok {")
	g.indent()
	g.writeLine("return f == float32(int32(f))")
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
	
	// Float validation
	g.writeLine("func isFloat(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case float32, float64:")
	g.indent()
	g.writeLine("return true")
	g.unindent()
	g.writeLine("case int, int8, int16, int32, int64:")
	g.indent()
	g.writeLine("return true // Integers are valid floats")
	g.unindent()
	g.writeLine("case uint, uint8, uint16, uint32, uint64:")
	g.indent()
	g.writeLine("return true // Unsigned integers are valid floats")
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
	
	// Boolean validation
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
	g.writeLine("case int, int8, int16, int32, int64:")
	g.indent()
	g.writeLine("return v == 0 || v == 1 // Only 0 and 1 are valid boolean integers")
	g.unindent()
	g.writeLine("case uint, uint8, uint16, uint32, uint64:")
	g.indent()
	g.writeLine("return v == 0 || v == 1 // Only 0 and 1 are valid boolean integers")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Date validation
	g.writeLine("func isDate(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case time.Time:")
	g.indent()
	g.writeLine("return !v.IsZero()")
	g.unindent()
	g.writeLine("case string:")
	g.indent()
	g.writeLine("// Try common date formats")
	g.writeLine("dateFormats := []string{")
	g.indent()
	g.writeLine("time.RFC3339,     // 2006-01-02T15:04:05Z07:00")
	g.writeLine("time.RFC3339Nano, // 2006-01-02T15:04:05.999999999Z07:00")
	g.writeLine("\"2006-01-02\",     // YYYY-MM-DD")
	g.writeLine("\"2006/01/02\",     // YYYY/MM/DD")
	g.writeLine("\"01/02/2006\",     // MM/DD/YYYY")
	g.writeLine("\"02-01-2006\",     // DD-MM-YYYY")
	g.writeLine("\"2006-01-02 15:04:05\", // YYYY-MM-DD HH:MM:SS")
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
	g.writeLine("case int64:")
	g.indent()
	g.writeLine("// Unix timestamp validation")
	g.writeLine("if v > 0 && v < 4102444800 { // Between 1970 and 2100")
	g.indent()
	g.writeLine("return true")
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
	
	// IP validation
	g.writeLine("func isIP(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case string:")
	g.indent()
	g.writeLine("// Parse IP address using net.ParseIP")
	g.writeLine("ip := net.ParseIP(v)")
	g.writeLine("return ip != nil")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// JSON validation
	g.writeLine("func isJSON(value interface{}) bool {")
	g.indent()
	g.writeLine("switch v := value.(type) {")
	g.writeLine("case string:")
	g.indent()
	g.writeLine("// Parse JSON string using json.Valid")
	g.writeLine("return json.Valid([]byte(v))")
	g.unindent()
	g.writeLine("case []byte:")
	g.indent()
	g.writeLine("// Parse JSON bytes using json.Valid")
	g.writeLine("return json.Valid(v)")
	g.unindent()
	g.writeLine("default:")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Hex color validation
	g.writeLine("func isHexColor(value interface{}) bool {")
	g.indent()
	g.writeLine("str, ok := value.(string)")
	g.writeLine("if !ok {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Remove leading # if present")
	g.writeLine("if strings.HasPrefix(str, \"#\") {")
	g.indent()
	g.writeLine("str = str[1:]")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check for valid hex color lengths (3 or 6 characters)")
	g.writeLine("if len(str) != 3 && len(str) != 6 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check if all characters are valid hexadecimal")
	g.writeLine("for _, char := range str {")
	g.indent()
	g.writeLine("if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return true")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Phone number validation
	g.writeLine("func isPhoneNumber(value interface{}) bool {")
	g.indent()
	g.writeLine("str, ok := value.(string)")
	g.writeLine("if !ok {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Remove all non-digit characters for validation")
	g.writeLine("digits := strings.Map(func(r rune) rune {")
	g.indent()
	g.writeLine("if r >= '0' && r <= '9' {")
	g.indent()
	g.writeLine("return r")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return -1")
	g.unindent()
	g.writeLine("}, str)")
	g.writeLine("")
	g.writeLine("// Check for valid phone number length (7-15 digits)")
	g.writeLine("if len(digits) < 7 || len(digits) > 15 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Basic phone number pattern validation")
	g.writeLine("// Accepts formats like: +1234567890, (123) 456-7890, 123-456-7890, etc.")
	g.writeLine("phoneRegex := `^[\\+]?[1-9]?[0-9]{7,14}$`")
	g.writeLine("matched, _ := regexp.MatchString(phoneRegex, digits)")
	g.writeLine("return matched")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Credit card validation using Luhn algorithm
	g.writeLine("func isCreditCard(value interface{}) bool {")
	g.indent()
	g.writeLine("str, ok := value.(string)")
	g.writeLine("if !ok {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Remove all non-digit characters")
	g.writeLine("digits := strings.Map(func(r rune) rune {")
	g.indent()
	g.writeLine("if r >= '0' && r <= '9' {")
	g.indent()
	g.writeLine("return r")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return -1")
	g.unindent()
	g.writeLine("}, str)")
	g.writeLine("")
	g.writeLine("// Check for valid credit card length (13-19 digits)")
	g.writeLine("if len(digits) < 13 || len(digits) > 19 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Luhn algorithm validation")
	g.writeLine("sum := 0")
	g.writeLine("alternate := false")
	g.writeLine("for i := len(digits) - 1; i >= 0; i-- {")
	g.indent()
	g.writeLine("digit := int(digits[i] - '0')")
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
	g.writeLine("")
	g.writeLine("return sum%10 == 0")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// ISBN validation (both ISBN-10 and ISBN-13)
	g.writeLine("func isISBN(value interface{}) bool {")
	g.indent()
	g.writeLine("str, ok := value.(string)")
	g.writeLine("if !ok {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Remove all non-digit and non-X characters")
	g.writeLine("isbn := strings.Map(func(r rune) rune {")
	g.indent()
	g.writeLine("if (r >= '0' && r <= '9') || r == 'X' || r == 'x' {")
	g.indent()
	g.writeLine("return r")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return -1")
	g.unindent()
	g.writeLine("}, str)")
	g.writeLine("")
	g.writeLine("// Convert to uppercase for X handling")
	g.writeLine("isbn = strings.ToUpper(isbn)")
	g.writeLine("")
	g.writeLine("// Check length for ISBN-10 or ISBN-13")
	g.writeLine("if len(isbn) == 10 {")
	g.indent()
	g.writeLine("return isISBN10(isbn)")
	g.unindent()
	g.writeLine("} else if len(isbn) == 13 {")
	g.indent()
	g.writeLine("return isISBN13(isbn)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// ISBN-10 validation helper
	g.writeLine("func isISBN10(isbn string) bool {")
	g.indent()
	g.writeLine("sum := 0")
	g.writeLine("for i := 0; i < 9; i++ {")
	g.indent()
	g.writeLine("digit := int(isbn[i] - '0')")
	g.writeLine("sum += digit * (10 - i)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Handle check digit (can be X for 10)")
	g.writeLine("var checkDigit int")
	g.writeLine("if isbn[9] == 'X' {")
	g.indent()
	g.writeLine("checkDigit = 10")
	g.unindent()
	g.writeLine("} else {")
	g.indent()
	g.writeLine("checkDigit = int(isbn[9] - '0')")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("sum += checkDigit")
	g.writeLine("return sum%11 == 0")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// ISBN-13 validation helper
	g.writeLine("func isISBN13(isbn string) bool {")
	g.indent()
	g.writeLine("sum := 0")
	g.writeLine("for i := 0; i < 13; i++ {")
	g.indent()
	g.writeLine("digit := int(isbn[i] - '0')")
	g.writeLine("if i%2 == 0 {")
	g.indent()
	g.writeLine("sum += digit")
	g.unindent()
	g.writeLine("} else {")
	g.indent()
	g.writeLine("sum += digit * 3")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return sum%10 == 0")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Base64 validation
	g.writeLine("func isBase64(value interface{}) bool {")
	g.indent()
	g.writeLine("str, ok := value.(string)")
	g.writeLine("if !ok {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Empty string is not valid base64")
	g.writeLine("if len(str) == 0 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check if length is multiple of 4")
	g.writeLine("if len(str)%4 != 0 {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Check for valid base64 characters")
	g.writeLine("for i, char := range str {")
	g.indent()
	g.writeLine("if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '+' || char == '/') {")
	g.indent()
	g.writeLine("// Allow padding characters only at the end")
	g.writeLine("if char == '=' {")
	g.indent()
	g.writeLine("// Check if padding is at the end")
	g.writeLine("for j := i; j < len(str); j++ {")
	g.indent()
	g.writeLine("if str[j] != '=' {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("break")
	g.unindent()
	g.writeLine("} else {")
	g.indent()
	g.writeLine("return false")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Additional validation using Go's base64 decoder")
	g.writeLine("_, err := base64.StdEncoding.DecodeString(str)")
	g.writeLine("return err == nil")
	g.unindent()
	g.writeLine("}")
}

// generateDTOValidationFunction generates validation function for a DTO
func (g *CodeGenerator) generateDTOValidationFunction(dto *ValidationStructInfo) {
	funcName := fmt.Sprintf("Validate%s", dto.Name)
	
	g.writeLine(fmt.Sprintf("// %s validates %s struct", funcName, dto.Name))
	g.writeLine(fmt.Sprintf("func %s(dto *%s) []ValidationError {", funcName, dto.Name))
	g.indent()
	g.writeLine("var errors []ValidationError")
	g.writeLine("")
	
	// Generate validation for each field
	for _, field := range dto.Fields {
		g.generateFieldValidation(field)
	}
	
	g.writeLine("return errors")
	g.unindent()
	g.writeLine("}")
}

// generateFieldValidation generates validation code for a field, handling @ValidateIf conditionals
func (g *CodeGenerator) generateFieldValidation(field *ValidationFieldInfo) {
	if len(field.Validators) == 0 {
		return
	}
	
	i := 0
	for i < len(field.Validators) {
		rule := field.Validators[i]
		
		if rule.Type == "ValidateIf" {
			// Handle conditional validation
			i = g.generateConditionalValidation(field, i)
		} else {
			// Handle regular validation
			g.generateValidationRule(field, rule)
			g.writeLine("")
			i++
		}
	}
}

// generateConditionalValidation handles @ValidateIf and groups subsequent validators
func (g *CodeGenerator) generateConditionalValidation(field *ValidationFieldInfo, startIndex int) int {
	validateIfRule := field.Validators[startIndex]
	
	if len(validateIfRule.Args) == 0 {
		g.writeLine("// ValidateIf validation - missing condition argument")
		return startIndex + 1
	}
	
	condition := fmt.Sprintf("%v", validateIfRule.Args[0])
	// Remove surrounding quotes but preserve inner quotes
	if strings.HasPrefix(condition, "\"") && strings.HasSuffix(condition, "\"") {
		condition = condition[1 : len(condition)-1]
	}
	
	g.writeLine(fmt.Sprintf("// %s validation", validateIfRule.Type))
	g.writeLine(fmt.Sprintf("if %s {", condition))
	g.indent()
	
	// Find all consecutive non-ValidateIf validators to group under this condition
	i := startIndex + 1
	for i < len(field.Validators) && field.Validators[i].Type != "ValidateIf" {
		rule := field.Validators[i]
		g.generateValidationRule(field, rule)
		g.writeLine("")
		i++
	}
	
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	return i
}

// generateValidationRule generates code for a single validation rule
func (g *CodeGenerator) generateValidationRule(field *ValidationFieldInfo, rule ValidationRule) {
	fieldName := "dto." + strings.Title(field.Name) // Use actual field name
	
	// Skip @IsOptional() as it's a modifier, not a validator
	if rule.Type == "IsOptional" {
		return
	}
	
	// Check if field is optional
	isOptional := g.isFieldOptional(field)
	
	// Helper function to wrap validation with optional guard
	wrapValidation := func(validationLogic func()) {
		if isOptional {
			var optionalGuard string
			if strings.HasPrefix(field.Type, "[]") {
				// For slices, check if not nil and not empty
				optionalGuard = fmt.Sprintf("if %s != nil && len(%s) > 0", fieldName, fieldName)
			} else if field.Type == "string" {
				// For strings, check if not empty after trimming
				optionalGuard = fmt.Sprintf("if strings.TrimSpace(%s) != \"\"", fieldName)
			} else {
				// For other types, check if not nil
				optionalGuard = fmt.Sprintf("if %s != nil", fieldName)
			}
			
			g.writeLine(optionalGuard + " {")
			g.indent()
			validationLogic()
			g.unindent()
			g.writeLine("}")
		} else {
			validationLogic()
		}
	}
	
	switch rule.Type {
	case "IsEmail":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine(fmt.Sprintf("if !isValidEmail(%s) {", fieldName))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "IsURL":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine(fmt.Sprintf("if !isValidURL(%s) {", fieldName))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "IsNotEmpty":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if strings.TrimSpace(%s) == \"\" {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "Min":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			if g.isStringType(field.Type) {
				g.writeLine(fmt.Sprintf("if len(%s) < %v {", fieldName, rule.Args[0]))
			} else {
				g.writeLine(fmt.Sprintf("if %s < %v {", fieldName, rule.Args[0]))
			}
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		}
		
	case "Max":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			if g.isStringType(field.Type) {
				g.writeLine(fmt.Sprintf("if len(%s) > %v {", fieldName, rule.Args[0]))
			} else {
				g.writeLine(fmt.Sprintf("if %s > %v {", fieldName, rule.Args[0]))
			}
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		}
		
	case "Length":
		if len(rule.Args) >= 2 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			g.writeLine(fmt.Sprintf("if len(%s) < %v || len(%s) > %v {", fieldName, rule.Args[0], fieldName, rule.Args[1]))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		}
		
	case "IsArray":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if %s == nil {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "ArrayMinSize":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("if %s != nil && len(%s) < %v {", fieldName, fieldName, rule.Args[0]))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "ArrayMaxSize":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			g.writeLine(fmt.Sprintf("if %s != nil && len(%s) > %v {", fieldName, fieldName, rule.Args[0]))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		}
		
	case "ArrayNotEmpty":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if %s == nil || len(%s) == 0 {", fieldName, fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsEmpty":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		// Check if the field type starts with [] (slice) or contains [] 
		if strings.HasPrefix(field.Type, "[]") {
			g.writeLine(fmt.Sprintf("if %s != nil && len(%s) > 0 {", fieldName, fieldName))
		} else if field.Type == "string" {
			g.writeLine(fmt.Sprintf("if strings.TrimSpace(%s) != \"\" {", fieldName))
		} else {
			// For other types, check if they're not nil/zero value
			g.writeLine(fmt.Sprintf("if %s != nil {", fieldName))
		}
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsString":
		// Type validation is typically handled at compile time in Go, but we can add runtime checks
		g.writeLine(fmt.Sprintf("// %s validation (compile-time type check)", rule.Type))
		
	case "IsNumber":
		// For interface{} types, add runtime type checking
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if _, ok := %s.(int); !ok {", fieldName))
		g.indent()
		g.writeLine(fmt.Sprintf("if _, ok := %s.(float64); !ok {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
		
	case "IsInt":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		// Check if field is interface{} type, otherwise assume compile-time type safety
		if field.Type == "interface{}" {
			g.writeLine(fmt.Sprintf("if !isInt(%s) {", fieldName))
		} else {
			// For typed fields, we can use reflection to check for interface{} conversion
			g.writeLine(fmt.Sprintf("if !isInt(%s) {", fieldName))
		}
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsFloat":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isFloat(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsBoolean":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isBoolean(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsDate":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isDate(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsIP":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isIP(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsJSON":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isJSON(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsHexColor":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isHexColor(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsPhoneNumber":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isPhoneNumber(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsCreditCard":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isCreditCard(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsISBN":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isISBN(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsBase64":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isBase64(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "MinLength":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("if len(%s) < %v {", fieldName, rule.Args[0]))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "MaxLength":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			g.writeLine(fmt.Sprintf("if len(%s) > %v {", fieldName, rule.Args[0]))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		}
		
	case "IsPositive":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if %s <= 0 {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsAlpha":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isAlpha(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsAlphanumeric":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		g.writeLine(fmt.Sprintf("if !isAlphanumeric(%s) {", fieldName))
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "IsNumeric":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine(fmt.Sprintf("if !isNumeric(%s) {", fieldName))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "IsDefined":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		if strings.HasPrefix(field.Type, "[]") {
			g.writeLine(fmt.Sprintf("if %s == nil {", fieldName))
		} else if field.Type == "string" {
			g.writeLine(fmt.Sprintf("if %s == \"\" {", fieldName))
		} else {
			g.writeLine(fmt.Sprintf("if %s == nil {", fieldName))
		}
		g.generateValidationError(field.Name, fieldName, rule)
		g.writeLine("}")
		
	case "NotEquals":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("if %s == %v {", fieldName, g.formatValue(rule.Args[0])))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "Equals":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("if %s != %v {", fieldName, g.formatValue(rule.Args[0])))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "Contains":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("if !strings.Contains(%s, %v) {", fieldName, g.formatValue(rule.Args[0])))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "NotContains":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("if strings.Contains(%s, %v) {", fieldName, g.formatValue(rule.Args[0])))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "IsIn":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("validValues := []interface{}{%s}", g.formatValueList(rule.Args)))
				g.writeLine("found := false")
				g.writeLine("for _, v := range validValues {")
				g.indent()
				g.writeLine(fmt.Sprintf("if %s == v {", fieldName))
				g.indent()
				g.writeLine("found = true")
				g.writeLine("break")
				g.unindent()
				g.writeLine("}")
				g.unindent()
				g.writeLine("}")
				g.writeLine("if !found {")
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "IsNotIn":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("invalidValues := []interface{}{%s}", g.formatValueList(rule.Args)))
				g.writeLine("for _, v := range invalidValues {")
				g.indent()
				g.writeLine(fmt.Sprintf("if %s == v {", fieldName))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("break")
				g.writeLine("}")
				g.unindent()
				g.writeLine("}")
			})
		}
		
	case "Matches":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				g.writeLine(fmt.Sprintf("matched, _ := regexp.MatchString(%v, %s)", g.formatValue(rule.Args[0]), fieldName))
				g.writeLine("if !matched {")
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "IsLowercase":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine(fmt.Sprintf("if %s != strings.ToLower(%s) {", fieldName, fieldName))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "IsUppercase":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine(fmt.Sprintf("if %s != strings.ToUpper(%s) {", fieldName, fieldName))
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "ValidateNested":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			// Handle different nested types: struct, pointer to struct, slice of structs
			switch {
			case strings.HasPrefix(field.Type, "[]"):
				// Handle slice of structs: []UserDto, []*UserDto
				g.generateNestedSliceValidation(field, fieldName)
			case strings.HasPrefix(field.Type, "*"):
				// Handle pointer to struct: *UserDto
				g.generateNestedPointerValidation(field, fieldName)
			default:
				// Handle direct struct: UserDto
				g.generateNestedStructValidation(field, fieldName)
			}
		})
		
	case "ValidateIf":
		// ValidateIf is handled at a higher level in generateFieldValidation
		// This case should not be reached in normal flow
		g.writeLine("// ValidateIf validation - handled by generateConditionalValidation")
		
	case "Custom":
		if len(rule.Args) > 0 {
			g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
			wrapValidation(func() {
				validatorFunc := fmt.Sprintf("%v", rule.Args[0])
				g.writeLine(fmt.Sprintf("if !%s(%s) {", validatorFunc, fieldName))
				g.generateValidationError(field.Name, fieldName, rule)
				g.writeLine("}")
			})
		}
		
	case "IsNegative":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			// Generate numeric type checking first
			switch field.Type {
			case "int", "int8", "int16", "int32", "int64",
				 "uint", "uint8", "uint16", "uint32", "uint64",
				 "float32", "float64":
				g.writeLine(fmt.Sprintf("if %s >= 0 {", fieldName))
			default:
				// For interface{} or other types, use runtime checking
				g.writeLine(fmt.Sprintf("if val, ok := %s.(int); ok && val >= 0 {", fieldName))
				g.indent()
				g.generateValidationError(field.Name, fieldName, rule)
				g.unindent()
				g.writeLine(fmt.Sprintf("} else if val, ok := %s.(float64); ok && val >= 0 {", fieldName))
				g.indent()
				g.generateValidationError(field.Name, fieldName, rule)
				g.unindent()
				g.writeLine(fmt.Sprintf("} else if val, ok := %s.(float32); ok && val >= 0 {", fieldName))
				g.indent()
				g.generateValidationError(field.Name, fieldName, rule)
				g.unindent()
				g.writeLine("}")
				return
			}
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "IsPastDate":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine("now := time.Now()")
			switch field.Type {
			case "time.Time":
				g.writeLine(fmt.Sprintf("if !%s.Before(now) {", fieldName))
			case "string":
				g.writeLine(fmt.Sprintf("if dateVal, err := time.Parse(time.RFC3339, %s); err != nil || !dateVal.Before(now) {", fieldName))
			case "*time.Time":
				g.writeLine(fmt.Sprintf("if %s == nil || !%s.Before(now) {", fieldName, fieldName))
			default:
				// For interface{} or other types
				g.writeLine(fmt.Sprintf("if dateVal, ok := %s.(time.Time); !ok || !dateVal.Before(now) {", fieldName))
			}
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	case "IsFutureDate":
		g.writeLine(fmt.Sprintf("// %s validation", rule.Type))
		wrapValidation(func() {
			g.writeLine("now := time.Now()")
			switch field.Type {
			case "time.Time":
				g.writeLine(fmt.Sprintf("if !%s.After(now) {", fieldName))
			case "string":
				g.writeLine(fmt.Sprintf("if dateVal, err := time.Parse(time.RFC3339, %s); err != nil || !dateVal.After(now) {", fieldName))
			case "*time.Time":
				g.writeLine(fmt.Sprintf("if %s == nil || !%s.After(now) {", fieldName, fieldName))
			default:
				// For interface{} or other types
				g.writeLine(fmt.Sprintf("if dateVal, ok := %s.(time.Time); !ok || !dateVal.After(now) {", fieldName))
			}
			g.generateValidationError(field.Name, fieldName, rule)
			g.writeLine("}")
		})
		
	}
}

// generateValidationError generates code to append a validation error
func (g *CodeGenerator) generateValidationError(fieldName, fieldValue string, rule ValidationRule) {
	g.indent()
	g.writeLine("errors = append(errors, ValidationError{")
	g.indent()
	g.writeLine(fmt.Sprintf("Field:   \"%s\",", fieldName))
	g.writeLine(fmt.Sprintf("Value:   %s,", fieldValue))
	g.writeLine(fmt.Sprintf("Message: \"%s\",", rule.Message))
	g.writeLine(fmt.Sprintf("Code:    \"%s\",", rule.Code))
	g.unindent()
	g.writeLine("})")
	g.unindent()
}

// isStringType checks if a type is a string type
func (g *CodeGenerator) isStringType(typeName string) bool {
	return typeName == "string" || strings.Contains(typeName, "string")
}

// isSliceType checks if a type is a slice/array type
func (g *CodeGenerator) isSliceType(typeName string) bool {
	return strings.HasPrefix(typeName, "[]") || strings.Contains(typeName, "[]")
}

// isFieldOptional checks if a field has the @IsOptional() decorator
func (g *CodeGenerator) isFieldOptional(field *ValidationFieldInfo) bool {
	for _, rule := range field.Validators {
		if rule.Type == "IsOptional" {
			return true
		}
	}
	return false
}

// formatValue formats a value for Go code generation
func (g *CodeGenerator) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatValueList formats a list of values for Go code generation
func (g *CodeGenerator) formatValueList(values []interface{}) string {
	var formattedValues []string
	for _, value := range values {
		formattedValues = append(formattedValues, g.formatValue(value))
	}
	return strings.Join(formattedValues, ", ")
}

// generateNestedStructValidation generates validation for direct struct type
func (g *CodeGenerator) generateNestedStructValidation(field *ValidationFieldInfo, fieldName string) {
	nestedValidationFunc := g.getNestedValidationFunctionName(field.Type)
	
	// For direct structs, we always validate regardless of @IsOptional()
	// Direct structs can't be nil, so @IsOptional() doesn't apply in the same way
	g.writeLine(fmt.Sprintf("if nestedErrors := %s(&%s); len(nestedErrors) > 0 {", nestedValidationFunc, fieldName))
	g.indent()
	g.writeLine("for _, nestedError := range nestedErrors {")
	g.indent()
	g.writeLine(fmt.Sprintf("nestedError.Field = \"%s.\" + nestedError.Field", field.Name))
	g.writeLine("errors = append(errors, nestedError)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateNestedPointerValidation generates validation for pointer to struct type
func (g *CodeGenerator) generateNestedPointerValidation(field *ValidationFieldInfo, fieldName string) {
	// Remove * prefix to get the actual struct type
	structType := strings.TrimPrefix(field.Type, "*")
	nestedValidationFunc := g.getNestedValidationFunctionName(structType)
	g.writeLine(fmt.Sprintf("if %s != nil {", fieldName))
	g.indent()
	g.writeLine(fmt.Sprintf("if nestedErrors := %s(%s); len(nestedErrors) > 0 {", nestedValidationFunc, fieldName))
	g.indent()
	g.writeLine("for _, nestedError := range nestedErrors {")
	g.indent()
	g.writeLine(fmt.Sprintf("nestedError.Field = \"%s.\" + nestedError.Field", field.Name))
	g.writeLine("errors = append(errors, nestedError)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateNestedSliceValidation generates validation for slice of structs
func (g *CodeGenerator) generateNestedSliceValidation(field *ValidationFieldInfo, fieldName string) {
	// Remove [] prefix and handle []*Type or []Type
	sliceType := strings.TrimPrefix(field.Type, "[]")
	isPointerSlice := strings.HasPrefix(sliceType, "*")
	if isPointerSlice {
		sliceType = strings.TrimPrefix(sliceType, "*")
	}
	
	nestedValidationFunc := g.getNestedValidationFunctionName(sliceType)
	g.writeLine(fmt.Sprintf("if %s != nil && len(%s) > 0 {", fieldName, fieldName))
	g.indent()
	g.writeLine(fmt.Sprintf("for i, item := range %s {", fieldName))
	g.indent()
	
	if isPointerSlice {
		g.writeLine("if item != nil {")
		g.indent()
		g.writeLine(fmt.Sprintf("if nestedErrors := %s(item); len(nestedErrors) > 0 {", nestedValidationFunc))
	} else {
		g.writeLine(fmt.Sprintf("if nestedErrors := %s(&item); len(nestedErrors) > 0 {", nestedValidationFunc))
	}
	
	g.indent()
	g.writeLine("for _, nestedError := range nestedErrors {")
	g.indent()
	g.writeLine(fmt.Sprintf("nestedError.Field = fmt.Sprintf(\"%s[%%d].%%s\", i, nestedError.Field)", field.Name))
	g.writeLine("errors = append(errors, nestedError)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	
	if isPointerSlice {
		g.unindent()
		g.writeLine("}")
	}
	
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// getNestedValidationFunctionName generates the validation function name for a nested type
func (g *CodeGenerator) getNestedValidationFunctionName(typeName string) string {
	// Handle different naming conventions
	if strings.HasSuffix(typeName, "Dto") {
		return fmt.Sprintf("Validate%s", typeName)
	}
	if strings.HasSuffix(typeName, "DTO") {
		return fmt.Sprintf("Validate%s", typeName)
	}
	// Default: add Dto suffix if not present
	return fmt.Sprintf("Validate%sDto", typeName)
}

// addValidationImportsIfNeeded adds validation imports if needed
func (g *CodeGenerator) addValidationImportsIfNeeded(file *GofaFile) {
	// Check if there are any structs with validation decorators
	dtoStructs := g.findDTOStructsWithValidation(file)
	
	if len(dtoStructs) > 0 {
		// Add required imports based on what validations are used
		g.addImport("strings")
		g.addImport("regexp")
		g.addImport("fmt") // For nested validation error formatting
		
		// Check if URL validation is used
		needsURLImport := g.usesURLValidation(dtoStructs)
		if needsURLImport {
			g.addImport("net/url")
		}
		
		// Check if Date validation is used
		needsTimeImport := g.usesDateValidation(dtoStructs)
		if needsTimeImport {
			g.addImport("time")
		}
		
		// Check if IP validation is used
		needsNetImport := g.usesIPValidation(dtoStructs)
		if needsNetImport {
			g.addImport("net")
		}
		
		// Check if JSON validation is used
		needsJSONImport := g.usesJSONValidation(dtoStructs)
		if needsJSONImport {
			g.addImport("encoding/json")
		}
		
		// Check if Base64 validation is used
		needsBase64Import := g.usesBase64Validation(dtoStructs)
		if needsBase64Import {
			g.addImport("encoding/base64")
		}
	}
}

// generateValidationCodeIfNeeded generates validation code if needed
func (g *CodeGenerator) generateValidationCodeIfNeeded(file *GofaFile) {
	// Check if there are any structs with validation decorators
	dtoStructs := g.findDTOStructsWithValidation(file)
	
	if len(dtoStructs) > 0 {
		// Generate validation code
		g.generateValidationCode(file)
	}
}

// usesURLValidation checks if any validation rules use URL validation
func (g *CodeGenerator) usesURLValidation(dtoStructs map[string]*ValidationStructInfo) bool {
	for _, dto := range dtoStructs {
		for _, field := range dto.Fields {
			for _, rule := range field.Validators {
				if rule.Type == "IsURL" {
					return true
				}
			}
		}
	}
	return false
}

// usesDateValidation checks if any validation rules use Date validation
func (g *CodeGenerator) usesDateValidation(dtoStructs map[string]*ValidationStructInfo) bool {
	for _, dto := range dtoStructs {
		for _, field := range dto.Fields {
			for _, rule := range field.Validators {
				if rule.Type == "IsDate" {
					return true
				}
			}
		}
	}
	return false
}

// usesIPValidation checks if any validation rules use IP validation
func (g *CodeGenerator) usesIPValidation(dtoStructs map[string]*ValidationStructInfo) bool {
	for _, dto := range dtoStructs {
		for _, field := range dto.Fields {
			for _, rule := range field.Validators {
				if rule.Type == "IsIP" {
					return true
				}
			}
		}
	}
	return false
}

// usesJSONValidation checks if any validation rules use JSON validation
func (g *CodeGenerator) usesJSONValidation(dtoStructs map[string]*ValidationStructInfo) bool {
	for _, dto := range dtoStructs {
		for _, field := range dto.Fields {
			for _, rule := range field.Validators {
				if rule.Type == "IsJSON" {
					return true
				}
			}
		}
	}
	return false
}

// usesBase64Validation checks if any validation rules use Base64 validation
func (g *CodeGenerator) usesBase64Validation(dtoStructs map[string]*ValidationStructInfo) bool {
	for _, dto := range dtoStructs {
		for _, field := range dto.Fields {
			for _, rule := range field.Validators {
				if rule.Type == "IsBase64" {
					return true
				}
			}
		}
	}
	return false
}
