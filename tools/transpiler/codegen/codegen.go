package codegen

import (
	"fmt"
	"go/format"
	"strings"
)

// Import AST types from parent package - these would need to be aliased or embedded
type GofaDeclaration interface{}
type GofaFile struct {
	Declarations []GofaDeclaration
}
type DecoratorNode struct {
	Name string
	Args []DecoratorArg
}
type DecoratorArg struct {
	Value interface{}
}
type ControllerDeclaration struct {
	Name       string
	Fields     []*FieldNode
	Methods    []*MethodNode
	Decorators []*DecoratorNode
}
type ServiceDeclaration struct {
	Name       string
	Fields     []*FieldNode
	Methods    []*MethodNode
	Decorators []*DecoratorNode
}
type ModuleDeclaration struct {
	Name       string
	Decorators []*DecoratorNode
}
type TestSuiteDeclaration struct {
	Name       string
	Methods    []*MethodNode
	Decorators []*DecoratorNode
}
type FactoryDeclaration struct {
	Name       string
	TargetType string
	Methods    []*MethodNode
	Decorators []*DecoratorNode
}
type MockDeclaration struct {
	Name       string
	Methods    []*MethodNode
	Decorators []*DecoratorNode
}
type TestModuleDeclaration struct {
	Name       string
	Decorators []*DecoratorNode
}
type FieldNode struct {
	Name       string
	Type       string
	Tag        string
	Decorators []*DecoratorNode
}
type MethodNode struct {
	Name       string
	Params     []*ParameterNode
	ReturnType string
	Decorators []*DecoratorNode
}
type ParameterNode struct {
	Name       string
	Type       string
	Decorators []*DecoratorNode
}

// Placeholder functions from lexer and parser
func NewLexer(content, path string) interface{} { return nil }
func NewParser(tokens interface{}, path string) interface{} { return nil }

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
	case *TestSuiteDeclaration:
		return g.generateTestSuiteDeclaration(d)
	case *FactoryDeclaration:
		return g.generateFactoryDeclaration(d)
	case *MockDeclaration:
		return g.generateMockDeclaration(d)
	case *TestModuleDeclaration:
		return g.generateTestModuleDeclaration(d)
	default:
		return fmt.Errorf("unsupported declaration type: %T", decl)
	}
}

// TranspileFile is the main entry point for transpiling a file
func TranspileFile(inputPath string, inputContent string) (string, error) {
	// Parse the input content
	lexer := NewLexer(inputContent, inputPath)
	tokens := lexer.TokenizeAll()
	parser := NewParser(tokens, inputPath)
	ast, err := parser.ParseFile()
	if err != nil {
		return "", err
	}

	// Generate Go code
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(ast)
	if err != nil {
		return "", err
	}

	return goCode, nil
}