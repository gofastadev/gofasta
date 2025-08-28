package core

import (
	"go/ast"
	"go/token"
)

// GofaASTNode represents the base interface for all Gofasta AST nodes
type GofaASTNode interface {
	String() string
	Pos() token.Pos
}

// GofaFile represents a complete .gofa file
type GofaFile struct {
	Package      *ast.Ident          // package name
	Imports      []*ast.ImportSpec   // import declarations
	Decorators   []*DecoratorNode    // file-level decorators
	Declarations []GofaDeclaration   // type and function declarations
	Comments     []*ast.CommentGroup // comments
	Position     token.Pos
}

func (f *GofaFile) String() string {
	return "GofaFile"
}

func (f *GofaFile) Pos() token.Pos {
	return f.Position
}

// GofaDeclaration represents any top-level declaration in a .gofa file
type GofaDeclaration interface {
	GofaASTNode
	isDeclaration()
}

// DecoratorNode represents a decorator like @Controller("/api")
type DecoratorNode struct {
	Name     string          // decorator name (Controller, Get, etc.)
	Args     []DecoratorArg  // decorator arguments
	Target   GofaDeclaration // what this decorator applies to
	Position token.Pos
}

func (d *DecoratorNode) String() string {
	return "@" + d.Name
}

func (d *DecoratorNode) Pos() token.Pos {
	return d.Position
}

// DecoratorArg represents an argument to a decorator
type DecoratorArg struct {
	Key      string      // for named arguments like {path: "/api"}
	Value    interface{} // string, number, boolean, array, or object
	Position token.Pos
}

// ControllerDeclaration represents a controller class
type ControllerDeclaration struct {
	Name       string           // controller name
	Decorators []*DecoratorNode // @Controller, @UseGuards, etc.
	Fields     []*FieldNode     // injected dependencies
	Methods    []*MethodNode    // controller methods/routes
	Position   token.Pos
}

func (c *ControllerDeclaration) String() string {
	return "Controller: " + c.Name
}

func (c *ControllerDeclaration) Pos() token.Pos {
	return c.Position
}

func (c *ControllerDeclaration) isDeclaration() {}

// ServiceDeclaration represents a service/provider class
type ServiceDeclaration struct {
	Name       string           // service name
	Decorators []*DecoratorNode // @Injectable, etc.
	Fields     []*FieldNode     // injected dependencies
	Methods    []*MethodNode    // service methods
	Position   token.Pos
}

func (s *ServiceDeclaration) String() string {
	return "Service: " + s.Name
}

func (s *ServiceDeclaration) Pos() token.Pos {
	return s.Position
}

func (s *ServiceDeclaration) isDeclaration() {}

// ModuleDeclaration represents a module
type ModuleDeclaration struct {
	Name       string           // module name
	Decorators []*DecoratorNode // @Module decorator
	Position   token.Pos
}

func (m *ModuleDeclaration) String() string {
	return "Module: " + m.Name
}

func (m *ModuleDeclaration) Pos() token.Pos {
	return m.Position
}

func (m *ModuleDeclaration) isDeclaration() {}

// WebSocketGatewayDeclaration represents a WebSocket gateway class
type WebSocketGatewayDeclaration struct {
	Name       string           // gateway name
	Decorators []*DecoratorNode // @WebSocketGateway, @UseGuards, etc.
	Fields     []*FieldNode     // injected dependencies
	Methods    []*MethodNode    // gateway methods/event handlers
	Port       *int             // WebSocket port from @WebSocketGateway
	Namespace  *string          // WebSocket namespace from @WebSocketGateway
	Config     map[string]interface{} // additional configuration
	Position   token.Pos
}

func (w *WebSocketGatewayDeclaration) String() string {
	return "WebSocketGateway: " + w.Name
}

func (w *WebSocketGatewayDeclaration) Pos() token.Pos {
	return w.Position
}

func (w *WebSocketGatewayDeclaration) isDeclaration() {}

// WebSocketFunctionDeclaration represents a standalone WebSocket lifecycle function
type WebSocketFunctionDeclaration struct {
	Name       string           // function name
	Decorators []*DecoratorNode // @OnGatewayConnection, @OnGatewayDisconnect, @OnGatewayInit, @SubscribeMessage
	Params     []*ParameterNode // method parameters with WebSocket decorators
	ReturnType string           // return type
	Body       []ast.Stmt       // function body statements
	Position   token.Pos
}

func (w *WebSocketFunctionDeclaration) String() string {
	return "WebSocketFunction: " + w.Name
}

func (w *WebSocketFunctionDeclaration) Pos() token.Pos {
	return w.Position
}

func (w *WebSocketFunctionDeclaration) isDeclaration() {}

// TestSuiteDeclaration represents a test suite class
type TestSuiteDeclaration struct {
	Name       string           // test suite name
	Decorators []*DecoratorNode // @TestSuite, @HTTPTest, @DatabaseTest, etc.
	Fields     []*FieldNode     // injected dependencies and mocks
	Methods    []*MethodNode    // test methods with @Test, @BeforeEach, etc.
	Position   token.Pos
}

func (ts *TestSuiteDeclaration) String() string {
	return "TestSuite: " + ts.Name
}

func (ts *TestSuiteDeclaration) Pos() token.Pos {
	return ts.Position
}

func (ts *TestSuiteDeclaration) isDeclaration() {}

// FactoryDeclaration represents a test data factory class
type FactoryDeclaration struct {
	Name       string           // factory name (e.g., "UserFactory")
	TargetType string           // target struct type (e.g., "User")  
	Decorators []*DecoratorNode // @Factory decorator
	Fields     []*FieldNode     // factory configuration fields
	Methods    []*MethodNode    // Build method and traits
	Position   token.Pos
}

func (f *FactoryDeclaration) String() string {
	return "Factory: " + f.Name
}

func (f *FactoryDeclaration) Pos() token.Pos {
	return f.Position
}

func (f *FactoryDeclaration) isDeclaration() {}

// MockDeclaration represents a mock class for testing
type MockDeclaration struct {
	Name         string           // mock name (e.g., "MockUserRepository")
	TargetType   string           // target interface type (e.g., "UserRepository") 
	Decorators   []*DecoratorNode // @Mock decorator
	Fields       []*FieldNode     // additional mock configuration fields
	Methods      []*MethodNode    // mock method implementations
	Position     token.Pos
}

func (m *MockDeclaration) String() string {
	return "Mock: " + m.Name
}

func (m *MockDeclaration) Pos() token.Pos {
	return m.Position
}

func (m *MockDeclaration) isDeclaration() {}

// TestModuleDeclaration represents a test module for dependency injection setup
type TestModuleDeclaration struct {
	Name       string              // module name (e.g., "TestAppModule")
	Decorators []*DecoratorNode    // @TestModule decorator with configuration
	Providers  []string            // providers array from decorator (e.g., ["UserService", "MockDatabase"])
	Imports    []string            // imported modules array from decorator
	Fields     []*FieldNode        // additional fields
	Methods    []*MethodNode       // setup methods if any
	Position   token.Pos
}

func (t *TestModuleDeclaration) String() string {
	return "TestModule: " + t.Name
}

func (t *TestModuleDeclaration) Pos() token.Pos {
	return t.Position
}

func (t *TestModuleDeclaration) isDeclaration() {}

// FieldNode represents a field with dependency injection
type FieldNode struct {
	Name       string           // field name
	Type       string           // field type
	Tag        string           // struct tag for injection
	Decorators []*DecoratorNode // field decorators if any
	Position   token.Pos
}

func (f *FieldNode) String() string {
	return f.Name + " " + f.Type
}

func (f *FieldNode) Pos() token.Pos {
	return f.Position
}

// MethodNode represents a method with possible route decorators
type MethodNode struct {
	Name       string           // method name
	Params     []*ParameterNode // method parameters
	ReturnType string           // return type
	Body       []ast.Stmt       // method body statements
	Decorators []*DecoratorNode // @Get, @Post, etc.
	Position   token.Pos
}

func (m *MethodNode) String() string {
	return "Method: " + m.Name
}

func (m *MethodNode) Pos() token.Pos {
	return m.Position
}

// ParameterNode represents a method parameter with possible decorators
type ParameterNode struct {
	Name       string           // parameter name
	Type       string           // parameter type
	Decorators []*DecoratorNode // @Body, @Param, @Query, etc.
	Position   token.Pos
}

func (p *ParameterNode) String() string {
	return p.Name + " " + p.Type
}

func (p *ParameterNode) Pos() token.Pos {
	return p.Position
}

// Visitor interface for traversing the AST
type Visitor interface {
	Visit(node GofaASTNode) Visitor
}

// Walk traverses an AST in depth-first order
func Walk(v Visitor, node GofaASTNode) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *GofaFile:
		for _, decl := range n.Declarations {
			Walk(v, decl)
		}
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}

	case *ControllerDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
		}

	case *ServiceDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
		}

	case *ModuleDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}

	case *TestSuiteDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
		}

	case *WebSocketGatewayDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
		}

	case *WebSocketFunctionDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, param := range n.Params {
			Walk(v, param)
		}

	case *MethodNode:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, param := range n.Params {
			Walk(v, param)
		}

	case *ParameterNode:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}

	case *FieldNode:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
	}
}