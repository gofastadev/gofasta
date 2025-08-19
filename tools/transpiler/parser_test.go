package transpiler

import (
	"strings"
	"testing"
)

// TestParserFixForInfiniteLoop tests that parser no longer hangs on complex input
func TestParserFixForInfiniteLoop(t *testing.T) {
	input := `package main

import "fmt"

@Controller("/test")
type TestController struct {
	Service *TestService ` + "`inject:\"\"`" + `
}

@Get("/")
func GetTest() {
}`

	_, err := ParseGofaFile(input)
	if err != nil {
		t.Logf("Parse error (expected): %v", err)
		// This is okay - we just want to make sure it doesn't hang
	}
}

// TestParserWithSimpleInput tests parser with minimal valid input
func TestParserWithSimpleInput(t *testing.T) {
	input := `package main

type TestController struct {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Simple parse failed: %v", err)
	}
	
	if file == nil || file.Package.Name != "main" {
		t.Error("Failed to parse simple input correctly")
	}
	
	if len(file.Declarations) != 1 {
		t.Errorf("Expected 1 declaration, got %d", len(file.Declarations))
	}
}

// TestParserWithDecorator tests parsing with a simple decorator
func TestParserWithDecorator(t *testing.T) {
	input := `package main

@Controller("/test")
type TestController struct {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Decorator parse failed: %v", err)
	}
	
	if file == nil || len(file.Declarations) != 1 {
		t.Error("Failed to parse decorated struct")
	}
	
	controller, ok := file.Declarations[0].(*ControllerDeclaration)
	if !ok {
		t.Error("Expected ControllerDeclaration")
	} else if len(controller.Decorators) != 1 {
		t.Errorf("Expected 1 decorator, got %d", len(controller.Decorators))
	}
}

// TestParserErrorRecoveryFixed tests that parser recovers from errors without hanging
func TestParserErrorRecoveryFixed(t *testing.T) {
	tests := []string{
		"@Controller",          // incomplete decorator
		"type struct {}",       // missing type name
		"@Get(/invalid)",       // invalid decorator args
		"func InvalidFunction(", // incomplete function
	}
	
	for _, input := range tests {
		t.Run("input: "+input, func(t *testing.T) {
			_, err := ParseGofaFile(input)
			// We expect errors, but the parser shouldn't hang
			if err == nil {
				t.Log("Unexpectedly succeeded, but that's okay")
			} else {
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}

// TestParserWithInvalidTokens tests parser with various invalid tokens
func TestParserWithInvalidTokens(t *testing.T) {
	// Test with tokens that might cause infinite loops
	inputs := []string{
		"###invalid###",
		"@@@@@",
		"(((((",
		"}}}}",
		strings.Repeat("@Controller", 100), // Repetitive tokens
	}
	
	for i, input := range inputs {
		t.Run("invalid_input_"+string(rune('A'+i)), func(t *testing.T) {
			_, err := ParseGofaFile(input)
			// Should get an error without hanging
			if err == nil {
				t.Error("Expected error for invalid input")
			} else {
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}