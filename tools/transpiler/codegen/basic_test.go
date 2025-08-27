package codegen

import (
	"strings"
	"testing"
)

func TestCodeGeneratorCreation(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	if generator == nil {
		t.Error("Failed to create code generator")
	}
	
	if generator.packageName != "test" {
		t.Errorf("Expected package name 'test', got '%s'", generator.packageName)
	}
}

func TestParseGofaFileBasic(t *testing.T) {
	input := `
package main

@Controller("/api")
type UserController struct {}

@Get("/users")
func (c *UserController) GetUsers() {}
`
	
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if len(file.Declarations) == 0 {
		t.Error("No declarations found")
	}
	
	controller, ok := file.Declarations[0].(*ControllerDeclaration)
	if !ok {
		t.Error("Expected ControllerDeclaration")
	}
	
	if controller.Name != "UserController" {
		t.Errorf("Expected controller name 'UserController', got '%s'", controller.Name)
	}
}

func TestBasicCodeGeneration(t *testing.T) {
	input := `
@Controller("/api")
type UserController struct {}

@Get("/users") 
func (c *UserController) GetUsers() {}
`
	
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	generator := NewCodeGenerator("main")
	code, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}
	
	if !strings.Contains(code, "package main") {
		t.Error("Generated code should contain package declaration")
	}
}

func TestDecoratorTypes(t *testing.T) {
	tests := []struct {
		name     string
		decorator string
		expected interface{}
	}{
		{"Catch", "Catch", "Catch"},
		{"Header", "Header", "Header"},
		{"HttpCode", "HttpCode", "HttpCode"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDecoratorTypeString(tt.decorator)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsErrorHandlingDecorator(t *testing.T) {
	tests := []struct {
		decorator string
		expected  bool
	}{
		{"Catch", true},
		{"Header", false},
		{"HttpCode", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.decorator, func(t *testing.T) {
			result := IsErrorHandlingDecoratorString(tt.decorator)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}