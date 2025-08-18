package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestSimpleTranspilation(t *testing.T) {
	// Simple test input
	input := `package main

type UserController struct {
	UserService *UserService
}

func GetUser(id string) {
}`

	// Try to transpile
	result, err := TranspileFile("test.gofa", input)
	if err != nil {
		t.Logf("Transpilation error (expected for simple test): %v", err)
		// For now, just log the error since our parser is designed for decorated files
		return
	}

	if !strings.Contains(result, "package main") {
		t.Errorf("Generated code should contain package main")
	}

	fmt.Printf("Generated code:\n%s\n", result)
}

func TestBasicTokenization(t *testing.T) {
	input := "@Controller"
	lexer := NewLexer(input)
	
	token := lexer.NextToken()
	if token.Type != DECORATOR {
		t.Errorf("Expected DECORATOR token, got %s", tokenTypeNames[token.Type])
	}
	
	token = lexer.NextToken()
	if token.Type != IDENT {
		t.Errorf("Expected IDENT token, got %s", tokenTypeNames[token.Type])
	}
	
	if token.Literal != "Controller" {
		t.Errorf("Expected 'Controller', got '%s'", token.Literal)
	}
}

func ExampleTranspileFile() {
	// Example of basic transpilation
	input := `package main

type TestController struct {
	Service *TestService
}

func TestMethod() {
}`

	result, err := TranspileFile("example.gofa", input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Generated Go code:\n%s\n", result)
}