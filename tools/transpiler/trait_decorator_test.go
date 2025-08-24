package transpiler

import (
	"strings"
	"testing"
)

// TestTraitDecorator tests the @Trait() decorator parsing and generation
func TestTraitDecorator(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {
	@Inject("database")
	db *Database
}

@Trait("admin")
func AdminUser() *User {
	// Create admin user with special privileges
}

@Trait("premium")
func PremiumUser() *User {
	// Create premium user with enhanced features
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

	if len(factory.Methods) != 2 {
		t.Fatalf("Expected 2 trait methods, got %d", len(factory.Methods))
	}

	// Check first trait method
	adminMethod := factory.Methods[0]
	if adminMethod.Name != "AdminUser" {
		t.Errorf("Expected method name 'AdminUser', got '%s'", adminMethod.Name)
	}

	if len(adminMethod.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator on AdminUser, got %d", len(adminMethod.Decorators))
	}

	if adminMethod.Decorators[0].Name != "Trait" {
		t.Errorf("Expected decorator 'Trait', got '%s'", adminMethod.Decorators[0].Name)
	}

	// Check decorator argument
	if len(adminMethod.Decorators[0].Args) != 1 {
		t.Fatalf("Expected 1 argument on @Trait decorator, got %d", len(adminMethod.Decorators[0].Args))
	}

	traitName, ok := adminMethod.Decorators[0].Args[0].Value.(string)
	if !ok {
		t.Errorf("Expected string argument, got %T", adminMethod.Decorators[0].Args[0].Value)
	}

	if traitName != "admin" {
		t.Errorf("Expected trait name 'admin', got '%s'", traitName)
	}

	// Check second trait method
	premiumMethod := factory.Methods[1]
	if premiumMethod.Name != "PremiumUser" {
		t.Errorf("Expected method name 'PremiumUser', got '%s'", premiumMethod.Name)
	}
}

// TestTraitCodeGeneration tests the code generation for @Trait() decorator
func TestTraitCodeGeneration(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {
	@Inject("db")
	database *Database
}

@Trait("admin")
func AdminUser() *User {
	// Admin user trait
}

@Trait("guest") 
func GuestUser() *User {
	// Guest user trait
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
		"database         *Database `inject:\"db\"`",
		"func NewUserFactory() *UserFactory {",
		"func (f *UserFactory) Build(overrides interface{}) *User {",
		"func (f *UserFactory) AdminUser(instance *User) *User {",
		"// Trait: admin",
		"func (f *UserFactory) GuestUser(instance *User) *User {",
		"// Trait: guest",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(generatedCode, expected) {
			t.Errorf("Generated code missing expected element: %s", expected)
		}
	}

	// Verify trait methods have correct signature
	if !strings.Contains(generatedCode, "func (f *UserFactory) AdminUser(instance *User) *User {") {
		t.Error("AdminUser trait method has incorrect signature")
	}

	if !strings.Contains(generatedCode, "func (f *UserFactory) GuestUser(instance *User) *User {") {
		t.Error("GuestUser trait method has incorrect signature")
	}
}

// TestTraitWithoutArgument tests @Trait() decorator without arguments
func TestTraitWithoutArgument(t *testing.T) {
	input := `package main

@Factory()
type UserFactory struct {}

@Trait()
func SpecialUser() *User {
	// Special user trait without argument
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	factory := file.Declarations[0].(*FactoryDeclaration)
	method := factory.Methods[0]

	// Should have Trait decorator with no arguments
	if len(method.Decorators[0].Args) != 0 {
		t.Errorf("Expected 0 arguments on @Trait decorator, got %d", len(method.Decorators[0].Args))
	}

	// Test code generation
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should use method name as trait name when no argument provided
	if !strings.Contains(generatedCode, "// Trait: SpecialUser") {
		t.Error("Generated code should use method name as trait name when no argument provided")
	}
}

// TestMultipleTraitsOnSameFactory tests multiple trait methods on the same factory
func TestMultipleTraitsOnSameFactory(t *testing.T) {
	input := `package main

@Factory()
type ProductFactory struct {}

@Trait("electronics")
func ElectronicsProduct() *Product {}

@Trait("books")
func BookProduct() *Product {}

@Trait("clothing")
func ClothingProduct() *Product {}

@Trait("food")
func FoodProduct() *Product {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	factory := file.Declarations[0].(*FactoryDeclaration)
	
	// Should have 4 trait methods
	if len(factory.Methods) != 4 {
		t.Fatalf("Expected 4 trait methods, got %d", len(factory.Methods))
	}

	expectedTraits := map[string]string{
		"ElectronicsProduct": "electronics",
		"BookProduct":       "books", 
		"ClothingProduct":   "clothing",
		"FoodProduct":       "food",
	}

	for i, method := range factory.Methods {
		if expectedTrait, exists := expectedTraits[method.Name]; exists {
			traitArg := method.Decorators[0].Args[0].Value.(string)
			if traitArg != expectedTrait {
				t.Errorf("Method %d (%s): expected trait '%s', got '%s'", 
					i, method.Name, expectedTrait, traitArg)
			}
		} else {
			t.Errorf("Unexpected method: %s", method.Name)
		}
	}

	// Test code generation includes all traits
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	for methodName, traitName := range expectedTraits {
		expectedMethod := "func (f *ProductFactory) " + methodName + "(instance *Product) *Product {"
		if !strings.Contains(generatedCode, expectedMethod) {
			t.Errorf("Generated code missing trait method: %s", methodName)
		}

		expectedComment := "// Trait: " + traitName
		if !strings.Contains(generatedCode, expectedComment) {
			t.Errorf("Generated code missing trait comment: %s", traitName)
		}
	}
}

// TestTraitMethodSignatures tests that trait methods have correct Go method signatures
func TestTraitMethodSignatures(t *testing.T) {
	input := `package main

@Factory()
type OrderFactory struct {}

@Trait("urgent")
func UrgentOrder() *Order {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Verify trait method has receiver, takes instance parameter, and returns same type
	expectedSignature := "func (f *OrderFactory) UrgentOrder(instance *Order) *Order {"
	if !strings.Contains(generatedCode, expectedSignature) {
		t.Errorf("Generated trait method has incorrect signature. Expected: %s", expectedSignature)
		t.Logf("Generated code:\n%s", generatedCode)
	}

	// Verify method returns the instance
	if !strings.Contains(generatedCode, "return instance") {
		t.Error("Trait method should return the instance parameter")
	}
}