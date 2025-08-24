package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// TestTestModuleDecorator tests the @TestModule() decorator parsing and generation
func TestTestModuleDecorator(t *testing.T) {
	input := `package main

@TestModule({
    providers: [UserService, MockDatabase]
})
type TestAppModule struct {
	@Inject("config")
	config *Config
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify parsing
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}

	testModule, ok := file.Declarations[0].(*TestModuleDeclaration)
	if !ok {
		t.Fatalf("Expected TestModuleDeclaration, got %T", file.Declarations[0])
	}

	if testModule.Name != "TestAppModule" {
		t.Errorf("Expected module name 'TestAppModule', got '%s'", testModule.Name)
	}

	if len(testModule.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator, got %d", len(testModule.Decorators))
	}

	if testModule.Decorators[0].Name != "TestModule" {
		t.Errorf("Expected decorator 'TestModule', got '%s'", testModule.Decorators[0].Name)
	}

	// Check providers
	expectedProviders := []string{"UserService", "MockDatabase"}
	if len(testModule.Providers) != len(expectedProviders) {
		t.Fatalf("Expected %d providers, got %d", len(expectedProviders), len(testModule.Providers))
	}

	for i, expected := range expectedProviders {
		if testModule.Providers[i] != expected {
			t.Errorf("Expected provider '%s', got '%s'", expected, testModule.Providers[i])
		}
	}

	// Check fields
	if len(testModule.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(testModule.Fields))
	}

	configField := testModule.Fields[0]
	if configField.Name != "config" || configField.Type != "*Config" {
		t.Errorf("Expected field 'config *Config', got '%s %s'", configField.Name, configField.Type)
	}
}

// TestTestModuleCodeGeneration tests the code generation for @TestModule() decorator
func TestTestModuleCodeGeneration(t *testing.T) {
	input := `package main

@TestModule({
    providers: [UserService, MockDatabase],
    imports: [UserModule]
})
type TestAppModule struct {
	@Inject("config")
	config *Config
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
		"type TestAppModule struct {",
		"container *core.DIContainer",
		"config    *Config `inject:\"config\"`",
		"func NewTestAppModule() *TestAppModule {",
		"func (tm *TestAppModule) SetupTest(t *testing.T) {",
		"func (tm *TestAppModule) TeardownTest() {",
		"tm.container.RegisterProvider(\"userservice\", func() interface{} {",
		"tm.container.RegisterProvider(\"mockdatabase\", func() interface{} {",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(generatedCode, expected) {
			t.Errorf("Generated code missing expected element: %s", expected)
		}
	}

	// Verify imports are added
	expectedImports := []string{
		`"testing"`,
		`"github.com/healtronlabs/gofasta/packages/core"`,
	}

	for _, expectedImport := range expectedImports {
		if !strings.Contains(generatedCode, expectedImport) {
			t.Errorf("Generated code missing expected import: %s", expectedImport)
		}
	}
}

// TestSimpleTestModule tests a minimal test module without imports
func TestSimpleTestModule(t *testing.T) {
	input := `package main

@TestModule({
    providers: [UserService]
})
type SimpleTestModule struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	testModule := file.Declarations[0].(*TestModuleDeclaration)
	
	if testModule.Name != "SimpleTestModule" {
		t.Errorf("Expected module name 'SimpleTestModule', got '%s'", testModule.Name)
	}

	// Should have 1 provider
	if len(testModule.Providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(testModule.Providers))
	}

	if testModule.Providers[0] != "UserService" {
		t.Errorf("Expected provider 'UserService', got '%s'", testModule.Providers[0])
	}

	// Should have no imports
	if len(testModule.Imports) != 0 {
		t.Errorf("Expected 0 imports, got %d", len(testModule.Imports))
	}

	// Should have no fields
	if len(testModule.Fields) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(testModule.Fields))
	}

	// Test code generation for simple test module
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate all required methods even for simple test module
	requiredMethods := []string{
		"func NewSimpleTestModule() *SimpleTestModule {",
		"func (tm *SimpleTestModule) SetupTest(t *testing.T) {",
		"func (tm *SimpleTestModule) TeardownTest() {",
	}

	for _, requiredMethod := range requiredMethods {
		if !strings.Contains(generatedCode, requiredMethod) {
			t.Errorf("Generated code missing required method: %s", requiredMethod)
		}
	}
}

// TestTestModuleWithMultipleProviders tests a test module with multiple providers
func TestTestModuleWithMultipleProviders(t *testing.T) {
	input := `package main

@TestModule({
    providers: [UserService, OrderService, MockDatabase, MockEmailService],
    imports: [UserModule, OrderModule]
})
type ComprehensiveTestModule struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	testModule := file.Declarations[0].(*TestModuleDeclaration)

	// Should have 4 providers
	if len(testModule.Providers) != 4 {
		t.Fatalf("Expected 4 providers, got %d", len(testModule.Providers))
	}

	expectedProviders := []string{"UserService", "OrderService", "MockDatabase", "MockEmailService"}
	for i, expected := range expectedProviders {
		if testModule.Providers[i] != expected {
			t.Errorf("Expected provider '%s', got '%s'", expected, testModule.Providers[i])
		}
	}

	// Should have 2 imports
	if len(testModule.Imports) != 2 {
		t.Fatalf("Expected 2 imports, got %d", len(testModule.Imports))
	}

	expectedImports := []string{"UserModule", "OrderModule"}
	for i, expected := range expectedImports {
		if testModule.Imports[i] != expected {
			t.Errorf("Expected import '%s', got '%s'", expected, testModule.Imports[i])
		}
	}

	// Test code generation
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate provider registrations for all providers
	expectedProviderRegistrations := []string{
		"tm.container.RegisterProvider(\"userservice\"",
		"tm.container.RegisterProvider(\"orderservice\"",
		"tm.container.RegisterProvider(\"mockdatabase\"",
		"tm.container.RegisterProvider(\"mockemailservice\"",
	}

	for _, expectedReg := range expectedProviderRegistrations {
		if !strings.Contains(generatedCode, expectedReg) {
			t.Errorf("Generated code missing expected provider registration: %s", expectedReg)
		}
	}

	// Should handle Mock vs non-Mock providers differently
	mockProviders := []string{"MockDatabase", "MockEmailService"}
	regularProviders := []string{"UserService", "OrderService"}

	for _, mockProvider := range mockProviders {
		expectedMockCode := fmt.Sprintf("return New%s(t)", mockProvider)
		if !strings.Contains(generatedCode, expectedMockCode) {
			t.Errorf("Generated code missing mock constructor call: %s", expectedMockCode)
		}
	}

	for _, regularProvider := range regularProviders {
		expectedRegularCode := fmt.Sprintf("return &%s{}", regularProvider)
		if !strings.Contains(generatedCode, expectedRegularCode) {
			t.Errorf("Generated code missing regular constructor call: %s", expectedRegularCode)
		}
	}
}

// TestTestModuleWithComplexFields tests test module with various field types and decorators
func TestTestModuleWithComplexFields(t *testing.T) {
	input := `package main

@TestModule({
    providers: [UserService]
})
type TestModuleWithFields struct {
	@Inject("primaryDB")
	database *Database
	
	@Inject()
	logger *Logger
	
	@Inject("config")
	config *Config
	
	customData string
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	testModule := file.Declarations[0].(*TestModuleDeclaration)

	// Should have 4 fields
	if len(testModule.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(testModule.Fields))
	}

	// Test code generation includes all fields
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	expectedFields := []string{
		"database   *Database `inject:\"primaryDB\"`",
		"logger     *Logger   `inject:\"logger\"`",
		"config     *Config   `inject:\"config\"`",
		"customData string    `inject:\"\"`",
	}

	for _, expectedField := range expectedFields {
		if !strings.Contains(generatedCode, expectedField) {
			t.Errorf("Generated code missing expected field: %s", expectedField)
		}
	}
}

// TestTestModuleNamingConvention tests that test modules work with various naming patterns
func TestTestModuleNamingConvention(t *testing.T) {
	testCases := []struct {
		moduleName string
	}{
		{"TestAppModule"},
		{"TestUserModule"}, 
		{"TestOrderModule"},
		{"IntegrationTestModule"},
		{"UnitTestModule"},
	}

	for _, testCase := range testCases {
		input := `package main

@TestModule({
    providers: [UserService]
})
type ` + testCase.moduleName + ` struct {}`

		file, err := ParseGofaFile(input)
		if err != nil {
			t.Fatalf("Failed to parse module %s: %v", testCase.moduleName, err)
		}

		testModule := file.Declarations[0].(*TestModuleDeclaration)
		if testModule.Name != testCase.moduleName {
			t.Errorf("Module %s: expected name '%s', got '%s'",
				testCase.moduleName, testCase.moduleName, testModule.Name)
		}
	}
}