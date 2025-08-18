package transpiler

import (
	"fmt"
	"go/format"
	"path/filepath"
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

	for _, method := range controller.Methods {
		routeInfo := g.getRouteInfo(method)
		if routeInfo.Method != "" {
			fullPath := g.combineRoutePaths(controllerPath, routeInfo.Path)

			// Generate route registration
			g.writeLine(fmt.Sprintf("server.%s(\"%s\", c.%s)",
				strings.Title(strings.ToLower(routeInfo.Method)),
				fullPath,
				method.Name))
		}
	}

	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
}

// generateControllerMethod generates a controller method with HTTP context
func (g *CodeGenerator) generateControllerMethod(controller *ControllerDeclaration, method *MethodNode) error {
	// Generate method signature
	signature := g.generateMethodSignature(controller.Name, method, true)
	g.writeLine(signature + " {")
	g.indent()

	// Generate parameter extraction from HTTP context
	g.generateParameterExtraction(method)

	// Generate method body placeholder
	g.writeLine("// TODO: Implement method logic")
	g.writeLine("ctx.JSON(200, map[string]interface{}{\"message\": \"Not implemented\"})")

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
			g.writeLine(fmt.Sprintf("if err := container.RegisterProvider(&%s{}); err != nil {", provider))
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
		if len(method.Params) > 0 {
			sig.WriteString(", ")
		}
	}

	for i, param := range method.Params {
		if i > 0 {
			sig.WriteString(", ")
		}
		sig.WriteString(fmt.Sprintf("%s %s", param.Name, param.Type))
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
				queryName := g.getDecoratorArgValue(decorator, 0)
				if queryName == "" {
					queryName = param.Name
				}
				g.writeLine(fmt.Sprintf("%s := ctx.GetQuery(\"%s\")", param.Name, queryName))

			case "Headers":
				headerName := g.getDecoratorArgValue(decorator, 0)
				if headerName == "" {
					headerName = param.Name
				}
				g.writeLine(fmt.Sprintf("%s := ctx.GetHeader(\"%s\")", param.Name, headerName))
			}
		}
	}
}

// generateInjectionTag generates injection struct tags
func (g *CodeGenerator) generateInjectionTag(field *FieldNode) string {
	if field.Tag != "" {
		return field.Tag
	}

	// Generate inject tag based on field name/type
	return `inject:""`
}

// collectImports collects all necessary imports
func (g *CodeGenerator) collectImports(file *GofaFile) {
	// Standard imports for Gofasta
	g.addImport("github.com/healtronlabs/gofasta/packages/core")
	g.addImport("github.com/healtronlabs/gofasta/packages/http")

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

// getControllerPath gets the controller base path
func (g *CodeGenerator) getControllerPath(controller *ControllerDeclaration) string {
	controllerDecorator := g.getDecorator(controller.Decorators, "Controller")
	if controllerDecorator != nil && len(controllerDecorator.Args) > 0 {
		return g.getDecoratorArgValue(controllerDecorator, 0)
	}

	// Default path based on controller name
	name := strings.TrimSuffix(controller.Name, "Controller")
	return "/" + strings.ToLower(name)
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

// getModuleConfig extracts module configuration from decorators
func (g *CodeGenerator) getModuleConfig(module *ModuleDeclaration) ModuleConfig {
	config := ModuleConfig{}

	moduleDecorator := g.getDecorator(module.Decorators, "Module")
	if moduleDecorator == nil {
		return config
	}

	// Extract configuration from decorator arguments
	// This is simplified - in reality, you'd parse the object syntax
	for _, arg := range moduleDecorator.Args {
		if arg.Key == "controllers" {
			if controllers, ok := arg.Value.([]string); ok {
				config.Controllers = controllers
			}
		} else if arg.Key == "providers" {
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
	path = strings.ReplaceAll(path, "//", "/")

	if path == "/" {
		return "/"
	}

	return strings.TrimSuffix(path, "/")
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
