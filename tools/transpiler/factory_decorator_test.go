package transpiler

import (
	"strings"
	"testing"
)

// TestFactoryDecorator tests the @Factory() decorator parsing and generation
func TestFactoryDecorator(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {
	@Inject("database")
	db *Database
}

func Build(overrides interface{}) *User {
	// Build logic
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify parsing
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}

	factory, ok := file.Declarations[0].(*FactoryDeclaration)
	if !ok {
		t.Fatalf("Expected FactoryDeclaration, got %T", file.Declarations[0])
	}

	if factory.Name != "UserFactory" {
		t.Errorf("Expected factory name 'UserFactory', got '%s'", factory.Name)
	}

	if factory.TargetType != "User" {
		t.Errorf("Expected target type 'User', got '%s'", factory.TargetType)
	}

	if len(factory.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator, got %d", len(factory.Decorators))
	}

	if factory.Decorators[0].Name != "Factory" {
		t.Errorf("Expected decorator 'Factory', got '%s'", factory.Decorators[0].Name)
	}

	// Check fields
	if len(factory.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(factory.Fields))
	}

	dbField := factory.Fields[0]
	if dbField.Name != "db" || dbField.Type != "*Database" {
		t.Errorf("Expected field 'db *Database', got '%s %s'", dbField.Name, dbField.Type)
	}

	// Check methods - for now we only parse the Build method  
	// TODO: Add support for trait method parsing
	if len(factory.Methods) != 1 {
		t.Logf("Factory: %+v", factory)
		for i, method := range factory.Methods {
			t.Logf("Method %d: %s with %d decorators", i, method.Name, len(method.Decorators))
		}
		t.Fatalf("Expected 1 method (Build), got %d", len(factory.Methods))
	}

	// Verify Build method
	buildMethod := factory.Methods[0]
	if buildMethod.Name != "Build" {
		t.Errorf("Expected first method to be 'Build', got '%s'", buildMethod.Name)
	}

	// TODO: Add trait method verification once trait parsing is implemented
}

// TestFactoryCodeGeneration tests the code generation for @Factory() decorator
func TestFactoryCodeGeneration(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {
	@Inject("database")
	db *Database
}

func Build(overrides interface{}) *User {
	return &User{
		ID:   1,
		Name: "John Doe",
	}
}

@Trait("admin")
func AsAdmin(user *User) *User {
	user.Role = "admin"
	return user
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
		"type UserFactory struct {",
		"sequenceCounters map[string]int",
		"rand             *rand.Rand",
		"db               *Database `inject:\"database\"`",
		"func NewUserFactory() *UserFactory {",
		"func (f *UserFactory) Build(overrides interface{}) *User {",
		"instance := &User{",
		"func (f *UserFactory) getSequence(name string) int {",
		"func (f *UserFactory) generateRandomString() string {",
		"func (f *UserFactory) generateRandomInt(min, max int) int {",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(generatedCode, expected) {
			t.Errorf("Generated code missing expected element: %s", expected)
		}
	}

	// Verify imports are added
	expectedImports := []string{
		`"math/rand"`,
		`"time"`,
		`"fmt"`,
	}

	for _, expectedImport := range expectedImports {
		if !strings.Contains(generatedCode, expectedImport) {
			t.Errorf("Generated code missing expected import: %s", expectedImport)
		}
	}
}

// TestSimpleFactory tests a minimal factory without traits
func TestSimpleFactory(t *testing.T) {
	input := `package main

@Factory()
type ProductFactory struct {}

func Build(overrides interface{}) *Product {
	return &Product{
		Name: "Test Product",
	}
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	factory := file.Declarations[0].(*FactoryDeclaration)
	
	if factory.Name != "ProductFactory" {
		t.Errorf("Expected factory name 'ProductFactory', got '%s'", factory.Name)
	}

	if factory.TargetType != "Product" {
		t.Errorf("Expected target type 'Product', got '%s'", factory.TargetType)
	}

	// Should have no fields
	if len(factory.Fields) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(factory.Fields))
	}

	// Should have 1 method (Build)
	if len(factory.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(factory.Methods))
	}

	if factory.Methods[0].Name != "Build" {
		t.Errorf("Expected method 'Build', got '%s'", factory.Methods[0].Name)
	}

	// Test code generation for simple factory
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate all required methods even for simple factory
	requiredMethods := []string{
		"func NewProductFactory() *ProductFactory {",
		"func (f *ProductFactory) Build(overrides interface{}) *Product {",
		"func (f *ProductFactory) getSequence(name string) int {",
		"func (f *ProductFactory) generateRandomString() string {",
		"func (f *ProductFactory) generateRandomInt(min, max int) int {",
	}

	for _, requiredMethod := range requiredMethods {
		if !strings.Contains(generatedCode, requiredMethod) {
			t.Errorf("Generated code missing required method: %s", requiredMethod)
		}
	}
}

// TestFactoryWithMultipleTraits tests a factory with multiple trait methods
func TestFactoryWithMultipleTraits(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {}

func Build(overrides interface{}) *User {
	return &User{Name: "John Doe"}
}

@Trait("admin")
func AsAdmin(user *User) *User {
	user.Role = "admin"
	return user
}

@Trait("verified")
func AsVerified(user *User) *User {
	user.Verified = true
	return user
}

@Trait("premium")
func AsPremium(user *User) *User {
	user.Plan = "premium"
	return user
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	factory := file.Declarations[0].(*FactoryDeclaration)

	// Should have 4 methods (1 Build + 3 Traits)
	if len(factory.Methods) != 4 {
		t.Fatalf("Expected 4 methods, got %d", len(factory.Methods))
	}

	// Count trait methods
	traitCount := 0
	for _, method := range factory.Methods {
		for _, decorator := range method.Decorators {
			if decorator.Name == "Trait" {
				traitCount++
				break
			}
		}
	}

	if traitCount != 3 {
		t.Errorf("Expected 3 trait methods, got %d", traitCount)
	}

	// Test code generation
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate all trait methods
	expectedTraitMethods := []string{
		"func (f *UserFactory) AsAdmin(instance *User) *User {",
		"func (f *UserFactory) AsVerified(instance *User) *User {",
		"func (f *UserFactory) AsPremium(instance *User) *User {",
		"// Trait: admin",
		"// Trait: verified",
		"// Trait: premium",
	}

	for _, expectedTraitMethod := range expectedTraitMethods {
		if !strings.Contains(generatedCode, expectedTraitMethod) {
			t.Errorf("Generated code missing expected trait method: %s", expectedTraitMethod)
		}
	}
}

// TestFactoryNamingConvention tests that target type is correctly extracted from factory name
func TestFactoryNamingConvention(t *testing.T) {
	testCases := []struct {
		factoryName string
		expectedType string
	}{
		{"UserFactory", "User"},
		{"ProductFactory", "Product"},
		{"OrderItemFactory", "OrderItem"},
		{"BlogPostFactory", "BlogPost"},
		{"Factory", ""}, // Edge case
	}

	for _, testCase := range testCases {
		input := `package main

@Factory()
type ` + testCase.factoryName + ` struct {}`

		file, err := ParseGofaFile(input)
		if err != nil {
			t.Fatalf("Failed to parse factory %s: %v", testCase.factoryName, err)
		}

		factory := file.Declarations[0].(*FactoryDeclaration)
		if factory.TargetType != testCase.expectedType {
			t.Errorf("Factory %s: expected target type '%s', got '%s'",
				testCase.factoryName, testCase.expectedType, factory.TargetType)
		}
	}
}

// TestFactoryWithComplexFields tests factory with various field types and decorators
func TestFactoryWithComplexFields(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {
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

	factory := file.Declarations[0].(*FactoryDeclaration)

	// Should have 4 fields
	if len(factory.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(factory.Fields))
	}

	// Test code generation includes all fields
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	expectedFields := []string{
		"database *Database `inject:\"primaryDB\"`",
		"logger *Logger `inject:\"logger\"`",
		"config *Config `inject:\"config\"`",
		"customData string",
	}

	for _, expectedField := range expectedFields {
		if !strings.Contains(generatedCode, expectedField) {
			t.Errorf("Generated code missing expected field: %s", expectedField)
		}
	}
}