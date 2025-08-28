package codegen

import (
	"fmt"
	"go/format"
	"strings"
	
	"github.com/healtronlabs/gofasta/tools/transpiler/core"
	"github.com/healtronlabs/gofasta/tools/transpiler/parsing"
)

// Use core types
type GofaDeclaration = core.GofaDeclaration
type GofaFile = core.GofaFile
type DecoratorNode = core.DecoratorNode
type DecoratorArg = core.DecoratorArg
type ControllerDeclaration = core.ControllerDeclaration
type ServiceDeclaration = core.ServiceDeclaration
type ModuleDeclaration = core.ModuleDeclaration
type TestSuiteDeclaration = core.TestSuiteDeclaration
type FactoryDeclaration = core.FactoryDeclaration
type MockDeclaration = core.MockDeclaration
type TestModuleDeclaration = core.TestModuleDeclaration
type WebSocketGatewayDeclaration = core.WebSocketGatewayDeclaration
type FieldNode = core.FieldNode
type MethodNode = core.MethodNode
type ParameterNode = core.ParameterNode

// ParseGofaFile parses a Gofasta file using the parsing package
func ParseGofaFile(input string) (*GofaFile, error) {
	return parsing.ParseGofaFile(input)
}

// GetDecoratorType returns the decorator type for tests
func GetDecoratorType(name string) core.DecoratorType {
	return core.GetDecoratorType(name)
}

// IsErrorHandlingDecorator checks if decorator is for error handling
func IsErrorHandlingDecorator(decoratorType core.DecoratorType) bool {
	return core.IsErrorHandlingDecorator(decoratorType)
}

// Legacy compatibility for tests - convert string to DecoratorType
func IsErrorHandlingDecoratorString(decoratorName string) bool {
	return core.IsErrorHandlingDecorator(core.GetDecoratorType(decoratorName))
}

// GetDecoratorTypeString returns decorator type as string for backward compatibility
func GetDecoratorTypeString(name string) string {
	return name // For backward compatibility, just return the string
}

// Decorator constants for backward compatibility with tests  
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

// Decorator type constants for backward compatibility
var (
	CatchDecoratorType      = core.CatchDecorator
	HeaderDecoratorType     = core.HeaderDecorator
	HttpCodeDecoratorType   = core.HttpCodeDecorator
	QueryDecoratorType      = core.QueryDecorator
	BodyDecoratorType       = core.BodyDecorator
	ParamDecoratorType      = core.ParamDecorator
	InjectDecoratorType     = core.InjectDecorator
	ControllerDecoratorType = core.ControllerDecorator
	RedirectDecoratorType   = core.RedirectDecorator
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

// NewLexer creates a new lexer - bridge to parsing package
func NewLexer(input string) *parsing.Lexer {
	return parsing.NewLexer(input)
}

// NewParser creates a new parser - bridge to parsing package  
func NewParser(lexer *parsing.Lexer) *parsing.Parser {
	return parsing.NewParser(lexer)
}