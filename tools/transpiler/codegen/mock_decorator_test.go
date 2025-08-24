package codegen

import (
	"strings"
	"testing"
)

// TestMockDecorator tests the @Mock() decorator parsing and generation
func TestMockDecorator(t *testing.T) {
	input := `package main

@Mock()
type MockUserRepository struct {
	@Inject("database")
	db *Database
}

func FindByID(id int) (*User, error) {
	// Mock implementation
}

func Save(user *User) error {
	// Mock implementation  
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify parsing
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}

	mock, ok := file.Declarations[0].(*MockDeclaration)
	if !ok {
		t.Fatalf("Expected MockDeclaration, got %T", file.Declarations[0])
	}

	if mock.Name != "MockUserRepository" {
		t.Errorf("Expected mock name 'MockUserRepository', got '%s'", mock.Name)
	}

	if mock.TargetType != "UserRepository" {
		t.Errorf("Expected target type 'UserRepository', got '%s'", mock.TargetType)
	}

	if len(mock.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator, got %d", len(mock.Decorators))
	}

	if mock.Decorators[0].Name != "Mock" {
		t.Errorf("Expected decorator 'Mock', got '%s'", mock.Decorators[0].Name)
	}

	// Check fields
	if len(mock.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(mock.Fields))
	}

	dbField := mock.Fields[0]
	if dbField.Name != "db" || dbField.Type != "*Database" {
		t.Errorf("Expected field 'db *Database', got '%s %s'", dbField.Name, dbField.Type)
	}

	// Check methods 
	if len(mock.Methods) != 2 {
		t.Logf("Mock: %+v", mock)
		for i, method := range mock.Methods {
			t.Logf("Method %d: %s with %d decorators", i, method.Name, len(method.Decorators))
		}
		t.Fatalf("Expected 2 methods, got %d", len(mock.Methods))
	}

	// Verify methods
	findMethod := mock.Methods[0]
	if findMethod.Name != "FindByID" {
		t.Errorf("Expected first method to be 'FindByID', got '%s'", findMethod.Name)
	}

	saveMethod := mock.Methods[1]
	if saveMethod.Name != "Save" {
		t.Errorf("Expected second method to be 'Save', got '%s'", saveMethod.Name)
	}
}

// TestMockCodeGeneration tests the code generation for @Mock() decorator
func TestMockCodeGeneration(t *testing.T) {
	input := `package main

@Mock()
type MockUserRepository struct {
	@Inject("database")
	db *Database
}

func FindByID(id int) (*User, error) {
	// Mock implementation
}

func Save(user *User) error {
	// Mock implementation  
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Debug: print generated code
	t.Logf("Generated code:\n%s", generatedCode)

	// Verify generated code contains expected elements
	expectedElements := []string{
		"type MockUserRepository struct {",
		"CallLog      []MockCall",
		"expectations []MockExpectation",
		"t            *testing.T",
		"db           *Database `inject:\"database\"`",
		"func NewMockUserRepository(t *testing.T) *MockUserRepository {",
		"type MockCall struct {",
		"type MockExpectation struct {",
		"func (m *MockUserRepository) On(method string, args ...interface{}) *MockExpectation {",
		"func (e *MockExpectation) Return(values ...interface{}) *MockExpectation {",
		"func (m *MockUserRepository) AssertExpectations(t *testing.T) {",
		"func (m *MockUserRepository) FindByID() {",
		"func (m *MockUserRepository) Save() {",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(generatedCode, expected) {
			t.Errorf("Generated code missing expected element: %s", expected)
		}
	}

	// Verify imports are added
	expectedImports := []string{
		`"testing"`,
		`"errors"`,
		`"fmt"`,
	}

	for _, expectedImport := range expectedImports {
		if !strings.Contains(generatedCode, expectedImport) {
			t.Errorf("Generated code missing expected import: %s", expectedImport)
		}
	}
}

// TestSimpleMock tests a minimal mock without fields
func TestSimpleMock(t *testing.T) {
	input := `package main

@Mock()
type MockEmailService struct {}

func SendEmail(to, subject, body string) error {
	// Mock implementation
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mock := file.Declarations[0].(*MockDeclaration)
	
	if mock.Name != "MockEmailService" {
		t.Errorf("Expected mock name 'MockEmailService', got '%s'", mock.Name)
	}

	if mock.TargetType != "EmailService" {
		t.Errorf("Expected target type 'EmailService', got '%s'", mock.TargetType)
	}

	// Should have no fields
	if len(mock.Fields) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(mock.Fields))
	}

	// Should have 1 method
	if len(mock.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(mock.Methods))
	}

	if mock.Methods[0].Name != "SendEmail" {
		t.Errorf("Expected method 'SendEmail', got '%s'", mock.Methods[0].Name)
	}

	// Test code generation for simple mock
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate all required methods and structures
	requiredElements := []string{
		"func NewMockEmailService(t *testing.T) *MockEmailService {",
		"func (m *MockEmailService) SendEmail() {",
		"type MockCall struct {",
		"type MockExpectation struct {",
		"func (m *MockEmailService) On(method string, args ...interface{}) *MockExpectation {",
		"func (m *MockEmailService) AssertExpectations(t *testing.T) {",
	}

	for _, requiredElement := range requiredElements {
		if !strings.Contains(generatedCode, requiredElement) {
			t.Errorf("Generated code missing required element: %s", requiredElement)
		}
	}
}

// TestMockWithMultipleMethods tests a mock with multiple method implementations
func TestMockWithMultipleMethods(t *testing.T) {
	input := `package main

@Mock()
type MockUserService struct {}

func Create(user *User) (*User, error) {
	// Mock implementation
}

func FindByEmail(email string) (*User, error) {
	// Mock implementation
}

func Update(user *User) error {
	// Mock implementation
}

func Delete(id int) error {
	// Mock implementation
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mock := file.Declarations[0].(*MockDeclaration)

	// Should have 4 methods
	if len(mock.Methods) != 4 {
		t.Fatalf("Expected 4 methods, got %d", len(mock.Methods))
	}

	expectedMethods := []string{"Create", "FindByEmail", "Update", "Delete"}
	for i, expectedMethod := range expectedMethods {
		if mock.Methods[i].Name != expectedMethod {
			t.Errorf("Expected method %d to be '%s', got '%s'", i, expectedMethod, mock.Methods[i].Name)
		}
	}

	// Test code generation
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate all method implementations
	expectedMethodImpls := []string{
		"func (m *MockUserService) Create() {",
		"func (m *MockUserService) FindByEmail() {",
		"func (m *MockUserService) Update() {",
		"func (m *MockUserService) Delete() {",
	}

	for _, expectedMethodImpl := range expectedMethodImpls {
		if !strings.Contains(generatedCode, expectedMethodImpl) {
			t.Errorf("Generated code missing expected method implementation: %s", expectedMethodImpl)
		}
	}
}

// TestMockNamingConvention tests that target type is correctly extracted from mock name
func TestMockNamingConvention(t *testing.T) {
	testCases := []struct {
		mockName     string
		expectedType string
	}{
		{"MockUserRepository", "UserRepository"},
		{"MockEmailService", "EmailService"}, 
		{"MockPaymentGateway", "PaymentGateway"},
		{"MockOrderProcessor", "OrderProcessor"},
		{"Mock", ""}, // Edge case
	}

	for _, testCase := range testCases {
		input := `package main

@Mock()
type ` + testCase.mockName + ` struct {}`

		file, err := ParseGofaFile(input)
		if err != nil {
			t.Fatalf("Failed to parse mock %s: %v", testCase.mockName, err)
		}

		mock := file.Declarations[0].(*MockDeclaration)
		if mock.TargetType != testCase.expectedType {
			t.Errorf("Mock %s: expected target type '%s', got '%s'",
				testCase.mockName, testCase.expectedType, mock.TargetType)
		}
	}
}

// TestMockWithComplexFields tests mock with various field types and decorators
func TestMockWithComplexFields(t *testing.T) {
	input := `package main

@Mock()
type MockOrderService struct {
	@Inject("database")
	database *Database
	
	@Inject()
	logger *Logger
	
	@Inject("paymentGateway")
	paymentGW *PaymentGateway
	
	// Custom configuration
	maxRetries int
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mock := file.Declarations[0].(*MockDeclaration)

	// Should have 4 fields
	if len(mock.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(mock.Fields))
	}

	// Test code generation includes all fields
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}
	
	// Debug: print generated code
	t.Logf("Generated code:\n%s", generatedCode)

	expectedFields := []string{
		"database     *Database       `inject:\"database\"`",
		"logger       *Logger         `inject:\"logger\"`",
		"paymentGW    *PaymentGateway `inject:\"paymentGateway\"`",
		"maxRetries   int             `inject:\"\"`",
	}

	for _, expectedField := range expectedFields {
		if !strings.Contains(generatedCode, expectedField) {
			t.Errorf("Generated code missing expected field: %s", expectedField)
		}
	}
}