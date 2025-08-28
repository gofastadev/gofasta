package core

import (
	"go/ast"
	"go/token"
	"testing"
)

// TestDecoratorNode tests the DecoratorNode AST node
func TestDecoratorNode(t *testing.T) {
	decorator := &DecoratorNode{
		Name:     "Controller",
		Args:     []DecoratorArg{{Value: "/api"}},
		Position: token.Pos(1),
	}

	// Test String method
	expected := "@Controller"
	if got := decorator.String(); got != expected {
		t.Errorf("DecoratorNode.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := decorator.Pos(); got != token.Pos(1) {
		t.Errorf("DecoratorNode.Pos() = %v, want %v", got, token.Pos(1))
	}
}

// TestFieldNode tests the FieldNode AST node
func TestFieldNode(t *testing.T) {
	field := &FieldNode{
		Name:     "username",
		Type:     "string",
		Position: token.Pos(10),
		Decorators: []*DecoratorNode{
			{Name: "IsEmail", Args: []DecoratorArg{}},
		},
	}

	// Test String method
	expected := "username string"
	if got := field.String(); got != expected {
		t.Errorf("FieldNode.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := field.Pos(); got != token.Pos(10) {
		t.Errorf("FieldNode.Pos() = %v, want %v", got, token.Pos(10))
	}
}

// TestMethodNode tests the MethodNode AST node  
func TestMethodNode(t *testing.T) {
	method := &MethodNode{
		Name:     "getUserById",
		Position: token.Pos(20),
		Decorators: []*DecoratorNode{
			{Name: "Get", Args: []DecoratorArg{{Value: "/user/:id"}}},
		},
	}

	// Test String method
	expected := "Method: getUserById"
	if got := method.String(); got != expected {
		t.Errorf("MethodNode.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := method.Pos(); got != token.Pos(20) {
		t.Errorf("MethodNode.Pos() = %v, want %v", got, token.Pos(20))
	}
}

// TestControllerDeclaration tests the ControllerDeclaration AST node
func TestControllerDeclaration(t *testing.T) {
	controller := &ControllerDeclaration{
		Name:     "UserController",
		Position: token.Pos(5),
		Decorators: []*DecoratorNode{
			{Name: "Controller", Args: []DecoratorArg{{Value: "/users"}}},
		},
		Fields: []*FieldNode{
			{Name: "service", Type: "*UserService"},
		},
		Methods: []*MethodNode{
			{Name: "getUsers", Position: token.Pos(15)},
		},
	}

	// Test String method
	expected := "Controller: UserController"
	if got := controller.String(); got != expected {
		t.Errorf("ControllerDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := controller.Pos(); got != token.Pos(5) {
		t.Errorf("ControllerDeclaration.Pos() = %v, want %v", got, token.Pos(5))
	}

	// Test isDeclaration method (should not panic)
	controller.isDeclaration()
}

// TestServiceDeclaration tests the ServiceDeclaration AST node
func TestServiceDeclaration(t *testing.T) {
	service := &ServiceDeclaration{
		Name:     "UserService",
		Position: token.Pos(30),
		Decorators: []*DecoratorNode{
			{Name: "Injectable", Args: []DecoratorArg{}},
		},
		Fields: []*FieldNode{
			{Name: "db", Type: "*Database"},
		},
		Methods: []*MethodNode{
			{Name: "findUser", Position: token.Pos(40)},
		},
	}

	// Test String method
	expected := "Service: UserService"
	if got := service.String(); got != expected {
		t.Errorf("ServiceDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := service.Pos(); got != token.Pos(30) {
		t.Errorf("ServiceDeclaration.Pos() = %v, want %v", got, token.Pos(30))
	}

	// Test isDeclaration method (should not panic)
	service.isDeclaration()
}

// TestModuleDeclarationAST tests the ModuleDeclaration AST node
func TestModuleDeclarationAST(t *testing.T) {
	module := &ModuleDeclaration{
		Name:     "AppModule",
		Position: token.Pos(50),
		Decorators: []*DecoratorNode{
			{Name: "Module", Args: []DecoratorArg{}},
		},
	}

	// Test String method
	expected := "Module: AppModule"
	if got := module.String(); got != expected {
		t.Errorf("ModuleDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := module.Pos(); got != token.Pos(50) {
		t.Errorf("ModuleDeclaration.Pos() = %v, want %v", got, token.Pos(50))
	}

	// Test isDeclaration method (should not panic)
	module.isDeclaration()
}

// TestWebSocketGatewayDeclaration tests the WebSocketGatewayDeclaration AST node
func TestWebSocketGatewayDeclarationAST(t *testing.T) {
	port := 8080
	namespace := "/chat"
	gateway := &WebSocketGatewayDeclaration{
		Name:     "ChatGateway",
		Position: token.Pos(60),
		Decorators: []*DecoratorNode{
			{Name: "WebSocketGateway", Args: []DecoratorArg{}},
		},
		Fields: []*FieldNode{
			{Name: "chatService", Type: "*ChatService"},
		},
		Methods: []*MethodNode{
			{Name: "handleMessage", Position: token.Pos(70)},
		},
		Port:      &port,
		Namespace: &namespace,
		Config: map[string]interface{}{
			"cors": true,
		},
	}

	// Test String method
	expected := "WebSocketGateway: ChatGateway"
	if got := gateway.String(); got != expected {
		t.Errorf("WebSocketGatewayDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := gateway.Pos(); got != token.Pos(60) {
		t.Errorf("WebSocketGatewayDeclaration.Pos() = %v, want %v", got, token.Pos(60))
	}

	// Test isDeclaration method (should not panic)
	gateway.isDeclaration()
}

// TestTestSuiteDeclaration tests the TestSuiteDeclaration AST node
func TestTestSuiteDeclarationAST(t *testing.T) {
	testSuite := &TestSuiteDeclaration{
		Name:     "UserControllerTest",
		Position: token.Pos(80),
		Decorators: []*DecoratorNode{
			{Name: "TestSuite", Args: []DecoratorArg{}},
		},
		Fields: []*FieldNode{
			{Name: "controller", Type: "*UserController"},
		},
		Methods: []*MethodNode{
			{Name: "testGetUser", Position: token.Pos(90)},
		},
	}

	// Test String method
	expected := "TestSuite: UserControllerTest"
	if got := testSuite.String(); got != expected {
		t.Errorf("TestSuiteDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := testSuite.Pos(); got != token.Pos(80) {
		t.Errorf("TestSuiteDeclaration.Pos() = %v, want %v", got, token.Pos(80))
	}

	// Test isDeclaration method (should not panic)
	testSuite.isDeclaration()
}

// TestFactoryDeclaration tests the FactoryDeclaration AST node
func TestFactoryDeclarationAST(t *testing.T) {
	factory := &FactoryDeclaration{
		Name:       "UserFactory",
		TargetType: "User",
		Position:   token.Pos(100),
		Decorators: []*DecoratorNode{
			{Name: "Factory", Args: []DecoratorArg{{Value: "User"}}},
		},
		Fields: []*FieldNode{
			{Name: "sequenceCounters", Type: "map[string]int"},
		},
		Methods: []*MethodNode{
			{Name: "Build", Position: token.Pos(110)},
		},
	}

	// Test String method
	expected := "Factory: UserFactory"
	if got := factory.String(); got != expected {
		t.Errorf("FactoryDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := factory.Pos(); got != token.Pos(100) {
		t.Errorf("FactoryDeclaration.Pos() = %v, want %v", got, token.Pos(100))
	}

	// Test isDeclaration method (should not panic)
	factory.isDeclaration()
}

// TestMockDeclaration tests the MockDeclaration AST node
func TestMockDeclarationAST(t *testing.T) {
	mock := &MockDeclaration{
		Name:       "MockUserRepository",
		TargetType: "UserRepository",
		Position:   token.Pos(120),
		Decorators: []*DecoratorNode{
			{Name: "Mock", Args: []DecoratorArg{{Value: "UserRepository"}}},
		},
		Fields: []*FieldNode{
			{Name: "calls", Type: "[]MockCall"},
		},
		Methods: []*MethodNode{
			{Name: "FindById", Position: token.Pos(130)},
		},
	}

	// Test String method
	expected := "Mock: MockUserRepository"
	if got := mock.String(); got != expected {
		t.Errorf("MockDeclaration.String() = %q, want %q", got, expected)
	}

	// Test Pos method
	if got := mock.Pos(); got != token.Pos(120) {
		t.Errorf("MockDeclaration.Pos() = %v, want %v", got, token.Pos(120))
	}

	// Test isDeclaration method (should not panic)
	mock.isDeclaration()
}

// TestGofaFile tests the GofaFile AST node
func TestGofaFile(t *testing.T) {
	pkg := &ast.Ident{Name: "main"}
	imports := []*ast.ImportSpec{
		{Path: &ast.BasicLit{Value: `"fmt"`}},
		{Path: &ast.BasicLit{Value: `"net/http"`}},
	}
	
	file := &GofaFile{
		Package: pkg,
		Imports: imports,
		Decorators: []*DecoratorNode{
			{Name: "Module", Args: []DecoratorArg{}},
		},
		Declarations: []GofaDeclaration{
			&ControllerDeclaration{Name: "TestController", Position: token.Pos(1)},
			&ServiceDeclaration{Name: "TestService", Position: token.Pos(2)},
		},
		Position: token.Pos(0),
	}

	// Test String method
	if file.String() != "GofaFile" {
		t.Errorf("Expected String() to return 'GofaFile', got %q", file.String())
	}

	// Test Pos method
	if file.Pos() != token.Pos(0) {
		t.Errorf("Expected Pos() to return 0, got %v", file.Pos())
	}

	// Test basic structure
	if file.Package.Name != "main" {
		t.Errorf("Expected package name 'main', got %q", file.Package.Name)
	}

	if len(file.Imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(file.Imports))
	}

	if len(file.Decorators) != 1 {
		t.Errorf("Expected 1 decorator, got %d", len(file.Decorators))
	}

	if len(file.Declarations) != 2 {
		t.Errorf("Expected 2 declarations, got %d", len(file.Declarations))
	}
}

// TestDecoratorArg tests the DecoratorArg structure
func TestDecoratorArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      DecoratorArg
		expected interface{}
	}{
		{
			name:     "string value",
			arg:      DecoratorArg{Value: "test"},
			expected: "test",
		},
		{
			name:     "int value",
			arg:      DecoratorArg{Value: 42},
			expected: 42,
		},
		{
			name:     "bool value",
			arg:      DecoratorArg{Value: true},
			expected: true,
		},
		{
			name:     "float value",
			arg:      DecoratorArg{Value: 3.14},
			expected: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.arg.Value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.arg.Value)
			}
		})
	}
}