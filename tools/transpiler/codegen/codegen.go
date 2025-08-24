package codegen

import (
	"fmt"
	"go/format"
	"strings"
	
	transpiler "github.com/healtronlabs/gofasta/tools/transpiler"
)

// Use AST types from parent package
type GofaDeclaration = transpiler.GofaDeclaration
type GofaFile = transpiler.GofaFile
type DecoratorNode = transpiler.DecoratorNode
type DecoratorArg = transpiler.DecoratorArg
type ControllerDeclaration = transpiler.ControllerDeclaration
type ServiceDeclaration = transpiler.ServiceDeclaration
type ModuleDeclaration = transpiler.ModuleDeclaration
type TestSuiteDeclaration = transpiler.TestSuiteDeclaration
type FactoryDeclaration = transpiler.FactoryDeclaration
type MockDeclaration = transpiler.MockDeclaration
type TestModuleDeclaration = transpiler.TestModuleDeclaration
type WebSocketGatewayDeclaration = transpiler.WebSocketGatewayDeclaration
type FieldNode = transpiler.FieldNode
type MethodNode = transpiler.MethodNode
type ParameterNode = transpiler.ParameterNode

// Parser types
type Lexer = transpiler.Lexer
type Parser = transpiler.Parser

// ParseGofaFile parses a Gofasta file using the parent package parser
func ParseGofaFile(input string) (*GofaFile, error) {
	// Use the real parser from parent package
	lexer := transpiler.NewLexer(input)
	parser := transpiler.NewParser(lexer)
	return parser.ParseFile()
}

// GetDecoratorType returns the decorator type for tests
func GetDecoratorType(name string) interface{} {
	return name
}

// IsErrorHandlingDecorator checks if decorator is for error handling
func IsErrorHandlingDecorator(decoratorType interface{}) bool {
	if str, ok := decoratorType.(string); ok {
		return str == "Catch"
	}
	return false
}

// Decorator constants for tests
const (
	CatchDecorator      = "Catch"
	HeaderDecorator     = "Header"
	HttpCodeDecorator   = "HttpCode"
	QueryDecorator      = "Query"
	BodyDecorator       = "Body"
	ParamDecorator      = "Param"
	InjectDecorator     = "Inject"
	ControllerDecorator = "Controller"
	RedirectDecorator   = "Redirect"
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
	case *TestSuiteDeclaration:
		return g.generateTestSuiteDeclaration(d)
	case *FactoryDeclaration:
		return g.generateFactoryDeclaration(d)
	case *MockDeclaration:
		return g.generateMockDeclaration(d)
	case *TestModuleDeclaration:
		return g.generateTestModuleDeclaration(d)
	case *WebSocketGatewayDeclaration:
		return g.generateWebSocketGatewayDeclaration(d)
	default:
		return fmt.Errorf("unsupported declaration type: %T", decl)
	}
}

// TranspileFile is the main entry point for transpiling a file
func TranspileFile(inputPath string, inputContent string) (string, error) {
	// Parse the input content
	file, err := ParseGofaFile(inputContent)
	if err != nil {
		return "", err
	}

	// Generate Go code
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		return "", err
	}

	return goCode, nil
}

// TokenType for tests
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	IDENT
	DECORATOR
	STRING
	CHAR
	INT
	FLOAT
	ASSIGN
	PLUS
	MINUS
	MULTIPLY
	DIVIDE
	MODULO
	LPAREN
	RPAREN
	LBRACE
	RBRACE
	TYPE
	STRUCT
)

// GofaASTNode interface for tests
type GofaASTNode interface{}

// Visitor interface for tests
type Visitor interface{}

// NewLexer creates a new lexer - bridge to parent package
func NewLexer(input string) *Lexer {
	return transpiler.NewLexer(input)
}

// NewParser creates a new parser - bridge to parent package  
func NewParser(lexer *Lexer) *Parser {
	return transpiler.NewParser(lexer)
}