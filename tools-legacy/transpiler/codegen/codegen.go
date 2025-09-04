package codegen

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/healtronlabs/gofasta/transpiler/core"
	"github.com/healtronlabs/gofasta/transpiler/parsing"
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
type WebSocketFunctionDeclaration = core.WebSocketFunctionDeclaration
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

// GenerationContext represents the current code generation context
type GenerationContext string

const (
	HTTPContext      GenerationContext = "http"
	WebSocketContext GenerationContext = "websocket"
)

// CodeGenerator generates Go code from Gofasta AST
type CodeGenerator struct {
	packageName        string
	imports            []string
	decoratorRegistry  map[string]*DecoratorNode
	generatedCode      strings.Builder
	indentLevel        int
	webSocketFunctions []*WebSocketFunctionDeclaration // Track standalone WebSocket functions
	currentContext     GenerationContext               // Track current generation context
	currentMethod      *MethodNode                     // Track current method being processed
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(packageName string) *CodeGenerator {
	return &CodeGenerator{
		packageName:        packageName,
		imports:            []string{},
		decoratorRegistry:  make(map[string]*DecoratorNode),
		webSocketFunctions: []*WebSocketFunctionDeclaration{},
		currentContext:     HTTPContext, // Default to HTTP context
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

	// Add WebSocket imports if needed
	g.addWebSocketImportsIfNeeded(file)

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

	// First, collect all WebSocket function declarations for later registration
	g.collectWebSocketFunctions(file)

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
	case *WebSocketFunctionDeclaration:
		// WebSocket functions are already collected, just generate them
		return g.generateWebSocketFunctionDeclaration(d)
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

// addWebSocketImportsIfNeeded adds WebSocket imports if needed
func (g *CodeGenerator) addWebSocketImportsIfNeeded(file *GofaFile) {
	hasWebSocket := false

	// Check if file has WebSocket declarations
	for _, decl := range file.Declarations {
		switch decl.(type) {
		case *WebSocketGatewayDeclaration:
			hasWebSocket = true
		case *WebSocketFunctionDeclaration:
			hasWebSocket = true
		}
		if hasWebSocket {
			break
		}
	}

	if hasWebSocket {
		// Add WebSocket package import
		g.imports = append(g.imports, "github.com/healtronlabs/gofasta/packages/websocket")

		// Ensure HTTP package is also imported for WebSocket integration
		httpImportExists := false
		for _, imp := range g.imports {
			if strings.Contains(imp, "github.com/healtronlabs/gofasta/packages/http") {
				httpImportExists = true
				break
			}
		}

		if !httpImportExists {
			g.imports = append(g.imports, "github.com/healtronlabs/gofasta/packages/http")
		}

		// Add conditional imports based on WebSocket features used
		g.addConditionalWebSocketImports(file)
	}
}

// addConditionalWebSocketImports adds additional imports based on WebSocket features used
func (g *CodeGenerator) addConditionalWebSocketImports(file *GofaFile) {
	needsErrors := false
	needsJSON := false
	needsLog := false
	needsContext := false

	for _, decl := range file.Declarations {
		switch d := decl.(type) {
		case *WebSocketGatewayDeclaration:
			// Check for middleware usage (needs context)
			if g.hasWebSocketMiddleware(d) {
				needsContext = true
			}

			// Check for error handling decorators
			if g.hasWebSocketErrorHandling(d) {
				needsErrors = true
			}

			// Check for JSON parameter decorators
			if g.hasWebSocketJSONFeatures(d) {
				needsJSON = true
			}

			// Check for logging features
			if g.hasWebSocketLogging(d) {
				needsLog = true
			}
		case *WebSocketFunctionDeclaration:
			// Check standalone functions for similar features
			if g.hasWebSocketParameterDecoratorsForParams(d.Params) {
				needsJSON = true
			}
		}
	}

	// Add imports only if needed to avoid unnecessary imports
	if needsErrors && !g.hasImport("errors") {
		g.imports = append(g.imports, "errors")
	}
	if needsJSON && !g.hasImport("encoding/json") {
		g.imports = append(g.imports, "encoding/json")
	}
	if needsLog && !g.hasImport("log") {
		g.imports = append(g.imports, "log")
	}
	if needsContext && !g.hasImport("context") {
		g.imports = append(g.imports, "context")
	}
}

// hasImport checks if an import already exists
func (g *CodeGenerator) hasImport(importPath string) bool {
	for _, imp := range g.imports {
		if strings.Contains(imp, importPath) {
			return true
		}
	}
	return false
}

// hasWebSocketMiddleware checks if WebSocket gateway uses middleware
func (g *CodeGenerator) hasWebSocketMiddleware(gateway *WebSocketGatewayDeclaration) bool {
	middlewareDecorators := []string{"UseGuards", "UseInterceptors", "UsePipes", "UseFilters"}

	// Check gateway-level middleware
	for _, decorator := range gateway.Decorators {
		for _, middleware := range middlewareDecorators {
			if decorator.Name == middleware {
				return true
			}
		}
	}

	// Check method-level middleware
	for _, method := range gateway.Methods {
		for _, decorator := range method.Decorators {
			for _, middleware := range middlewareDecorators {
				if decorator.Name == middleware {
					return true
				}
			}
		}
	}

	return false
}

// hasWebSocketErrorHandling checks if WebSocket gateway uses error handling
func (g *CodeGenerator) hasWebSocketErrorHandling(gateway *WebSocketGatewayDeclaration) bool {
	// Check for error handling decorators or return types
	for _, method := range gateway.Methods {
		// Check for error return type (various possible formats)
		if method.ReturnType != "" && strings.Contains(strings.ToLower(method.ReturnType), "error") {
			return true
		}

		// Also check method signature if ReturnType is not set
		if method.ReturnType == "" {
			// If we have parameters but no return type specified, assume simple method
			// We'll need to analyze the generated code for error patterns
			continue
		}

		// Check for Catch decorators
		for _, decorator := range method.Decorators {
			if decorator.Name == "Catch" {
				return true
			}
		}
	}

	// For now, if we don't have explicit error handling, don't add errors import
	// This can be enhanced later to analyze the generated method bodies
	return false
}

// hasWebSocketJSONFeatures checks if WebSocket gateway uses JSON features
func (g *CodeGenerator) hasWebSocketJSONFeatures(gateway *WebSocketGatewayDeclaration) bool {
	// Check for complex message body types that would require JSON serialization
	for _, method := range gateway.Methods {
		for _, param := range method.Params {
			// Check for MessageBody decorators with complex types
			for _, decorator := range param.Decorators {
				if decorator.Name == "MessageBody" {
					// If parameter type is a struct pointer or complex type, needs JSON
					if strings.HasPrefix(param.Type, "*") && !isSimpleType(param.Type) {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasWebSocketLogging checks if WebSocket gateway uses logging features
func (g *CodeGenerator) hasWebSocketLogging(gateway *WebSocketGatewayDeclaration) bool {
	// Check for logging-related middleware or decorators
	for _, decorator := range gateway.Decorators {
		if strings.Contains(strings.ToLower(decorator.Name), "log") {
			return true
		}
	}

	for _, method := range gateway.Methods {
		for _, decorator := range method.Decorators {
			if strings.Contains(strings.ToLower(decorator.Name), "log") {
				return true
			}
		}
	}

	return false
}

// hasWebSocketParameterDecoratorsForParams checks if parameters have WebSocket decorators
func (g *CodeGenerator) hasWebSocketParameterDecoratorsForParams(params []*ParameterNode) bool {
	wsDecorators := []string{"MessageBody", "ConnectedSocket", "MessageAck", "MessagePattern",
		"Rooms", "Namespace", "CurrentUser", "ClientIP", "EventName", "Server"}

	for _, param := range params {
		for _, decorator := range param.Decorators {
			for _, wsDecorator := range wsDecorators {
				if decorator.Name == wsDecorator {
					return true
				}
			}
		}
	}

	return false
}

// isSimpleType checks if a type is a simple built-in type
func isSimpleType(typeStr string) bool {
	simpleTypes := []string{"string", "int", "bool", "float64", "byte"}
	for _, simple := range simpleTypes {
		if strings.Contains(typeStr, simple) {
			return true
		}
	}
	return false
}

// collectWebSocketFunctions collects all WebSocket function declarations for registration
func (g *CodeGenerator) collectWebSocketFunctions(file *GofaFile) {
	for _, decl := range file.Declarations {
		if wsFunc, ok := decl.(*WebSocketFunctionDeclaration); ok {
			g.webSocketFunctions = append(g.webSocketFunctions, wsFunc)
		}
	}
}
