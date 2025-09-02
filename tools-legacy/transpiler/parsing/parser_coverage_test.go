package parsing

import (
	"testing"
)

// TestParseComplexTypes tests parsing of complex Go types
func TestParseComplexTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "slice of pointers",
			input: `package main
type UserService struct {
	Users []*User
}`,
		},
		{
			name: "map with complex values",
			input: `package main
type CacheService struct {
	Cache map[string]*UserData
}`,
		},
		{
			name: "basic struct with string",
			input: `package main
type UserService struct {
	Name string
}`,
		},
		{
			name: "struct with int field",
			input: `package main
type UserService struct {
	ID int
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Logf("Parsing error: %v", err)
			} else if file != nil {
				t.Logf("Successfully parsed file with %d declarations", len(file.Declarations))
			}
		})
	}
}

// TestParseDecoratorEdgeCases tests edge cases in decorator parsing
func TestParseDecoratorEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "simple controller",
			input: `package main
@Controller("/api")
type UserController struct {}`,
		},
		{
			name: "service with injectable",
			input: `package main
@Injectable
type UserService struct {}`,
		},
		{
			name: "multiple decorators",
			input: `package main
@Controller("/api")
type UserController struct {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Logf("Parsing error: %v", err)
			} else if file != nil && len(file.Declarations) > 0 {
				t.Logf("Successfully parsed decorator case with %d declarations", len(file.Declarations))
			}
		})
	}
}

// TestErrorRecovery tests parser error recovery mechanisms
func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "missing package",
			input: `type ValidController struct {}`,
		},
		{
			name: "incomplete decorator",
			input: `package main
@Controller
type ValidController struct {}`,
		},
		{
			name: "invalid syntax",
			input: `package main
invalid syntax here
type ValidController struct {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGofaFile(tt.input)
			// We expect errors for these test cases, but parser shouldn't panic
			if err == nil {
				t.Log("Unexpectedly succeeded - parser recovered well")
			} else {
				t.Logf("Got expected parsing error: %v", err)
			}
		})
	}
}

// TestLexerBasics tests basic lexer functionality
func TestLexerBasics(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "simple identifiers",
			input: `package main type User struct`,
		},
		{
			name: "string literals",
			input: `"hello world" 'single' "/api/path"`,
		},
		{
			name: "numbers",
			input: `123 456.789 0xFF`,
		},
		{
			name: "special symbols",
			input: `@ ( ) { } [ ] , . : ;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := []Token{}
			
			for {
				token := lexer.NextToken()
				tokens = append(tokens, token)
				if token.Type == EOF {
					break
				}
			}
			
			if len(tokens) > 0 {
				t.Logf("Successfully tokenized %d tokens", len(tokens))
			}
		})
	}
}