package transpiler

import (
	"fmt"
	"go/format"
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

// GenerateGoCode generates Go code from a Gofasta AST - basic implementation
func (g *CodeGenerator) GenerateGoCode(file *GofaFile) (string, error) {
	g.reset()

	// Generate package declaration
	g.writeLine(fmt.Sprintf("package %s", g.packageName))
	g.writeLine("")

	// Add basic imports
	g.addImport("fmt")
	g.addImport("github.com/healtronlabs/gofasta/packages/core")
	g.addImport("github.com/healtronlabs/gofasta/packages/http")
	g.addImport("net/http")
	g.addImport("strconv")
	g.addImport("strings")

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

// generateDeclaration generates code for a declaration - basic implementation
func (g *CodeGenerator) generateDeclaration(decl GofaDeclaration) error {
	switch d := decl.(type) {
	case *ControllerDeclaration:
		return g.generateControllerDeclaration(d)
	case *ServiceDeclaration:
		return g.generateServiceDeclaration(d)
	default:
		return fmt.Errorf("unsupported declaration type: %T", decl)
	}
}

// generateControllerDeclaration generates basic controller code
func (g *CodeGenerator) generateControllerDeclaration(controller *ControllerDeclaration) error {
	// Generate struct declaration
	g.writeLine(fmt.Sprintf("type %s struct {", controller.Name))
	g.indent()
	for _, field := range controller.Fields {
		g.writeLine(fmt.Sprintf("%s %s `inject:\"%s\"`", field.Name, field.Type, strings.ToLower(field.Name)))
	}
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate route registration
	g.generateRouteRegistration(controller)

	// Generate methods
	for _, method := range controller.Methods {
		g.generateControllerMethod(controller, method)
		g.writeLine("")
	}

	return nil
}

// generateServiceDeclaration generates basic service code
func (g *CodeGenerator) generateServiceDeclaration(service *ServiceDeclaration) error {
	// Generate struct declaration
	g.writeLine(fmt.Sprintf("type %s struct {", service.Name))
	g.indent()
	for _, field := range service.Fields {
		g.writeLine(fmt.Sprintf("%s %s `inject:\"%s\"`", field.Name, field.Type, strings.ToLower(field.Name)))
	}
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate constructor
	g.writeLine(fmt.Sprintf("func New%s(container *core.DIContainer) (*%s, error) {", service.Name, service.Name))
	g.indent()
	g.writeLine(fmt.Sprintf("instance := &%s{}", service.Name))
	g.writeLine("return instance, nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	return nil
}

// generateRouteRegistration generates route registration method
func (g *CodeGenerator) generateRouteRegistration(controller *ControllerDeclaration) {
	g.writeLine(fmt.Sprintf("func (c *%s) RegisterRoutes(server *httpPackage.HTTPServer) error {", controller.Name))
	g.indent()

	// Extract controller path
	controllerPath := ""
	for _, decorator := range controller.Decorators {
		if decorator.Name == "Controller" && len(decorator.Args) > 0 {
			if pathArg, ok := decorator.Args[0].Value.(string); ok {
				controllerPath = pathArg
			}
		}
	}

	// Register routes for each method
	for _, method := range controller.Methods {
		for _, decorator := range method.Decorators {
			httpMethod := ""
			routePath := ""
			
			switch decorator.Name {
			case "Get":
				httpMethod = "Get"
			case "Post":
				httpMethod = "Post"
			case "Put":
				httpMethod = "Put"
			case "Delete":
				httpMethod = "Delete"
			}

			if httpMethod != "" && len(decorator.Args) > 0 {
				if pathArg, ok := decorator.Args[0].Value.(string); ok {
					routePath = pathArg
				}
				fullPath := controllerPath + routePath
				g.writeLine(fmt.Sprintf("server.%s(\"%s\", c.%s)", httpMethod, fullPath, method.Name))
			}
		}
	}

	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateControllerMethod generates controller method
func (g *CodeGenerator) generateControllerMethod(controller *ControllerDeclaration, method *MethodNode) {
	g.writeLine(fmt.Sprintf("func (c *%s) %s(ctx *httpPackage.RequestContext) {", controller.Name, method.Name))
	g.indent()
	g.writeLine("// TODO: Implement method logic")
	g.writeLine("ctx.JSON(200, map[string]interface{}{\"message\": \"Not implemented\"})")
	g.unindent()
	g.writeLine("}")
}

// Helper methods
func (g *CodeGenerator) reset() {
	g.generatedCode.Reset()
	g.imports = []string{}
	g.indentLevel = 0
}

func (g *CodeGenerator) addImport(importPath string) {
	for _, imp := range g.imports {
		if imp == importPath {
			return
		}
	}
	g.imports = append(g.imports, importPath)
}

func (g *CodeGenerator) writeLine(line string) {
	for i := 0; i < g.indentLevel; i++ {
		g.generatedCode.WriteString("\t")
	}
	g.generatedCode.WriteString(line)
	g.generatedCode.WriteString("\n")
}

func (g *CodeGenerator) indent() {
	g.indentLevel++
}

func (g *CodeGenerator) unindent() {
	if g.indentLevel > 0 {
		g.indentLevel--
	}
}