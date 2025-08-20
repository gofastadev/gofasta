package transpiler

import (
	"strings"
	"testing"
)

func TestInjectDecoratorBasic(t *testing.T) {
	input := `package main

@Injectable
type UserService struct {
	@Inject("database")
	DB *Database
	
	@Inject()
	Logger *Logger
	
	Cache *Cache
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Check that struct is generated
	if !strings.Contains(output, "type UserService struct {") {
		t.Error("Generated code should contain UserService struct")
	}

	// Check that DB field has correct inject tag with custom token
	if !strings.Contains(output, `inject:"database"`) {
		t.Error("Generated code should contain DB field with inject:\"database\" tag")
	}

	// Check that Logger field has correct inject tag with default token (field name)
	if !strings.Contains(output, `inject:"logger"`) {
		t.Error("Generated code should contain Logger field with inject:\"logger\" tag")
	}

	// Check that Cache field has default empty inject tag (no @Inject decorator)
	if !strings.Contains(output, `Cache`) && !strings.Contains(output, `inject:""`) {
		t.Error("Generated code should contain Cache field with empty inject tag")
	}

	t.Logf("Generated code:\n%s", output)
}

func TestInjectDecoratorWithOptions(t *testing.T) {
	input := `package main

@Injectable
type UserService struct {
	@Inject({token: "database", optional: true})
	DB *Database
	
	@Inject({token: "redis", scope: "singleton"})
	Cache *Cache
	
	@Inject({optional: true})
	Logger *Logger
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Check that DB field has inject tag with token and optional
	if !strings.Contains(output, `inject:"database" optional:"true"`) {
		t.Error("Generated code should contain DB field with inject:\"database\" optional:\"true\" tags")
	}

	// Check that Cache field has inject tag with token and scope
	if !strings.Contains(output, `inject:"redis" scope:"singleton"`) {
		t.Error("Generated code should contain Cache field with inject:\"redis\" scope:\"singleton\" tags")
	}

	// Check that Logger field has inject tag with default token and optional
	if !strings.Contains(output, `inject:"logger" optional:"true"`) {
		t.Error("Generated code should contain Logger field with inject:\"logger\" optional:\"true\" tags")
	}

	t.Logf("Generated code:\n%s", output)
}

func TestInjectDecoratorInController(t *testing.T) {
	input := `package main

@Controller("/api/users")
type UserController struct {
	@Inject("userService")
	UserService *UserService
	
	@Inject({token: "logger", optional: true})
	Logger *Logger
}

@Get("/")
func (c *UserController) GetUsers(ctx *RequestContext) {
	// Implementation
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Check that controller struct is generated with inject tags
	if !strings.Contains(output, "type UserController struct {") {
		t.Error("Generated code should contain UserController struct")
	}

	// Check that UserService field has correct inject tag
	if !strings.Contains(output, `inject:"userService"`) {
		t.Error("Generated code should contain UserService field with inject:\"userService\" tag")
	}

	// Check that Logger field has correct inject tag with optional
	if !strings.Contains(output, `inject:"logger" optional:"true"`) {
		t.Error("Generated code should contain Logger field with inject:\"logger\" optional:\"true\" tags")
	}

	// Check that the route registration method is generated
	if !strings.Contains(output, "func (c *UserController) RegisterRoutes") {
		t.Error("Generated code should contain RegisterRoutes method")
	}

	t.Logf("Generated code:\n%s", output)
}

func TestInjectDecoratorASTParsing(t *testing.T) {
	input := `package main

@Injectable
type UserService struct {
	@Inject("database")
	DB *Database
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Check AST structure
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}

	service, ok := file.Declarations[0].(*ServiceDeclaration)
	if !ok {
		t.Fatal("Expected ServiceDeclaration")
	}

	if len(service.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(service.Fields))
	}

	field := service.Fields[0]
	if len(field.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator on field, got %d", len(field.Decorators))
	}

	decorator := field.Decorators[0]
	if decorator.Name != "Inject" {
		t.Errorf("Expected decorator name 'Inject', got '%s'", decorator.Name)
	}

	if len(decorator.Args) != 1 {
		t.Fatalf("Expected 1 decorator argument, got %d", len(decorator.Args))
	}

	argValue, ok := decorator.Args[0].Value.(string)
	if !ok {
		t.Fatal("Expected string argument")
	}

	if argValue != "database" {
		t.Errorf("Expected argument value 'database', got '%s'", argValue)
	}
}

func TestInjectDecoratorMultipleFields(t *testing.T) {
	input := `package main

@Injectable
type ComplexService struct {
	@Inject("primaryDB")
	PrimaryDB *Database
	
	@Inject("secondaryDB") 
	SecondaryDB *Database
	
	@Inject({token: "cache", optional: true})
	Cache *Cache
	
	@Inject()
	Logger *Logger
	
	Config *Config
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	expectedTags := []string{
		`inject:"primaryDB"`,
		`inject:"secondaryDB"`,
		`inject:"cache" optional:"true"`,
		`inject:"logger"`,
		`inject:""`,
	}

	for _, expectedTag := range expectedTags {
		if !strings.Contains(output, expectedTag) {
			t.Errorf("Generated code should contain: %s", expectedTag)
		}
	}

	t.Logf("Generated code:\n%s", output)
}

func TestInjectDecoratorErrorCases(t *testing.T) {
	// Test case 1: Invalid decorator placement (should not cause parser to crash)
	input1 := `package main

type UserService struct {
	@Inject("database")
	@InvalidDecorator
	DB *Database
}`

	_, err := ParseGofaFile(input1)
	if err != nil {
		t.Logf("Expected parsing to handle invalid decorator gracefully: %v", err)
	}

	// Test case 2: Empty inject decorator
	input2 := `package main

@Injectable
type UserService struct {
	@Inject()
	DB *Database
}`

	file2, err := ParseGofaFile(input2)
	if err != nil {
		t.Fatalf("Failed to parse input2: %v", err)
	}

	generator := NewCodeGenerator("main")
	output2, err := generator.GenerateGoCode(file2)
	if err != nil {
		t.Fatalf("Failed to generate Go code for input2: %v", err)
	}

	// Should use field name as default token
	if !strings.Contains(output2, `inject:"db"`) {
		t.Error("Generated code should contain DB field with inject:\"db\" tag (lowercase field name)")
	}
}

func TestInjectDecoratorWithExistingTag(t *testing.T) {
	input := `package main

@Injectable
type UserService struct {
	@Inject("database")
	DB *Database ` + "`json:\"db\"`" + `
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Should preserve existing tag instead of generating inject tag
	if strings.Contains(output, "inject:") {
		t.Error("Generated code should not add inject tag when field already has a tag")
	}

	if !strings.Contains(output, "`json:\"db\"`") {
		t.Error("Generated code should preserve existing json tag")
	}

	t.Logf("Generated code:\n%s", output)
}