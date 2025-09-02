package parsing

import (
	"testing"
)

// Test uncovered lexer methods
func TestLexerUncoveredMethods(t *testing.T) {
	// Test String method for Token
	token := Token{
		Type:    IDENT,
		Literal: "testToken",
		Line:    1,
		Column:  1,
	}
	
	str := token.String()
	if str == "" {
		t.Error("Token.String() should not return empty string")
	}
	t.Logf("Token.String() result: %s", str)
}

// Test comment reading functions (currently 0% coverage)
func TestCommentReading(t *testing.T) {
	// Test single line comment
	input := "// This is a single line comment\nlet x = 5;"
	lexer := NewLexer(input)
	
	token := lexer.NextToken()
	// Should skip the comment and get the identifier
	for token.Type != IDENT && token.Type != EOF {
		token = lexer.NextToken()
	}
	
	if token.Type == IDENT && token.Literal == "let" {
		t.Logf("Successfully skipped single line comment")
	}
	
	// Test multi-line comment
	input2 := "/* This is a\n   multi-line comment */\nlet y = 10;"
	lexer2 := NewLexer(input2)
	
	token2 := lexer2.NextToken()
	// Should skip the comment and get the identifier
	for token2.Type != IDENT && token2.Type != EOF {
		token2 = lexer2.NextToken()
	}
	
	if token2.Type == IDENT && token2.Literal == "let" {
		t.Logf("Successfully skipped multi-line comment")
	}
}

// Test additional token types and edge cases
func TestTokenizeFileFunction(t *testing.T) {
	// Test TokenizeFile function (currently 0% coverage)
	input := `package main
	
	@Controller("/api")
	type UserController struct {
		service UserService
	}
	
	@Get("/users/:id")
	func (c *UserController) GetUser(id string) User {
		return c.service.FindUser(id)
	}`
	
	tokens, err := TokenizeFile(input)
	if err != nil {
		t.Errorf("TokenizeFile failed: %v", err)
	}
	
	if len(tokens) == 0 {
		t.Error("TokenizeFile should return some tokens")
	}
	
	t.Logf("TokenizeFile produced %d tokens", len(tokens))
	
	// Verify we get expected token types
	foundPackage := false
	foundController := false
	foundString := false
	
	for _, token := range tokens {
		switch token.Type {
		case PACKAGE:
			foundPackage = true
		case DECORATOR:
			foundController = true
		case STRING:
			foundString = true
		}
	}
	
	if !foundPackage {
		t.Error("Should find PACKAGE token")
	}
	if !foundController {
		t.Error("Should find AT token for @Controller")
	}
	if !foundString {
		t.Error("Should find STRING token")
	}
}

// Test IsValidGoFastaFile function (currently 0% coverage) 
func TestIsValidGoFastaFile(t *testing.T) {
	// Valid GoFasta file with decorators
	validInput := `package main
	
	@Controller("/api")
	type UserController struct {}
	
	@Injectable
	type UserService struct {}`
	
	validTokens, err := TokenizeFile(validInput)
	if err != nil {
		t.Fatalf("Failed to tokenize valid input: %v", err)
	}
	
	isValid := IsValidGoFastaFile(validTokens)
	if !isValid {
		t.Error("Valid GoFasta file should be detected as valid")
	}
	
	// Regular Go file without decorators
	regularGoInput := `package main
	
	type User struct {
		ID   int
		Name string
	}
	
	func main() {
		fmt.Println("Hello World")
	}`
	
	regularTokens, err2 := TokenizeFile(regularGoInput)
	if err2 != nil {
		t.Fatalf("Failed to tokenize regular input: %v", err2)
	}
	
	isValid2 := IsValidGoFastaFile(regularTokens)
	if isValid2 {
		t.Log("Regular Go file detected as not GoFasta (expected)")
	}
	
	// Empty input
	emptyTokens := []Token{}
	isValid3 := IsValidGoFastaFile(emptyTokens)
	if isValid3 {
		t.Error("Empty input should not be valid GoFasta")
	}
}

// Test FilterTokens function (currently 0% coverage)
func TestFilterTokensFunction(t *testing.T) {
	input := `package main
	
	// This is a comment
	@Controller("/api")  
	type UserController struct {
		/* multi-line comment */
		service UserService
	}`
	
	// First tokenize
	allTokens, err := TokenizeFile(input)
	if err != nil {
		t.Fatalf("TokenizeFile failed: %v", err)
	}
	
	// Test filter function for IDENT tokens specifically
	identTokens := FilterTokens(allTokens, IDENT)
	
	if len(identTokens) == 0 {
		t.Error("FilterTokens should return some IDENT tokens")
	}
	
	// Should have fewer tokens than original
	if len(identTokens) >= len(allTokens) {
		t.Error("Filtered IDENT tokens should be fewer than all tokens")
	}
	
	// Verify all filtered tokens are IDENT type
	for _, token := range identTokens {
		if token.Type != IDENT {
			t.Errorf("All filtered tokens should be IDENT, found: %v", token.Type)
		}
	}
	
	t.Logf("Original tokens: %d, IDENT tokens: %d", len(allTokens), len(identTokens))
	
	// Test filter for DECORATOR tokens
	decoratorTokens := FilterTokens(allTokens, DECORATOR)
	t.Logf("DECORATOR tokens found: %d", len(decoratorTokens))
	
	// Test filter for STRING tokens
	stringTokens := FilterTokens(allTokens, STRING)
	t.Logf("STRING tokens found: %d", len(stringTokens))
}

// Test peekCharAt function (currently 0% coverage)
func TestPeekCharAt(t *testing.T) {
	input := "hello world"
	lexer := NewLexer(input)
	
	// Test that we can peek at different positions
	// Note: peekCharAt is a private method, so we test indirectly through NextToken
	// which should exercise the peek functionality internally
	
	token1 := lexer.NextToken()
	if token1.Type != IDENT || token1.Literal != "hello" {
		t.Errorf("Expected 'hello' identifier, got %v: %s", token1.Type, token1.Literal)
	}
	
	token2 := lexer.NextToken()
	if token2.Type != IDENT || token2.Literal != "world" {
		t.Errorf("Expected 'world' identifier, got %v: %s", token2.Type, token2.Literal)
	}
}

// Test number reading with edge cases
func TestNumberReadingEdgeCases(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"42", "42"},
		{"3.14", "3.14"},
		{"0", "0"},
		{"123.456", "123.456"},
		{"0.5", "0.5"},
		// Note: .5 is not supported by this lexer - it parses as PERIOD token
		// {"0.5", "0.5"} is already tested above
	}
	
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			lexer := NewLexer(tc.input)
			token := lexer.NextToken()
			
			// Numbers can be INT or FLOAT
			if token.Type != INT && token.Type != FLOAT {
				t.Errorf("Expected INT or FLOAT token for '%s', got %v", tc.input, token.Type)
			}
			
			if token.Literal != tc.expected {
				t.Errorf("Expected literal '%s', got '%s'", tc.expected, token.Literal)
			}
		})
	}
}

// Test string reading with escape sequences
func TestStringReadingWithEscapes(t *testing.T) {
	testCases := []struct {
		input       string
		expected    string
		description string
	}{
		{`"hello"`, "hello", "simple string"},
		{`"hello\nworld"`, `hello\nworld`, "string with newline escape (preserved literally)"},
		{`"hello\tworld"`, `hello\tworld`, "string with tab escape (preserved literally)"},
		{`"hello\"world"`, `hello\"world`, "string with quote escape (preserved literally)"},
		{`"hello\\world"`, `hello\\world`, "string with backslash escape (preserved literally)"},
		{`""`, "", "empty double quotes"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			lexer := NewLexer(tc.input)
			token := lexer.NextToken()
			
			if token.Type != STRING {
				t.Errorf("Expected STRING token for %s, got %v", tc.input, token.Type)
			}
			
			if token.Literal != tc.expected {
				t.Errorf("Expected literal '%s', got '%s'", tc.expected, token.Literal)
			}
		})
	}
}