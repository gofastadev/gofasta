package transpiler

import (
	"strings"
	"testing"
)

func TestValidationDecoratorTranspilation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // What we expect to see in the generated code
	}{
		{
			name: "IsEmail validation decorator",
			input: `package main

type CreateUserDto struct {
	Email string ` + "`" + `validate:"@IsEmail()"` + "`" + `
}

@Controller("/users")
type UserController struct {
	service UserService ` + "`" + `inject:"userService"` + "`" + `
}

@Post("/")
func (c *UserController) createUser(@Body() data CreateUserDto) User {
	return User{}
}`,
			expected: []string{
				"ValidationError",
				"isValidEmail",
				"emailRegex",
				"must be a valid email address",
				"IS_EMAIL",
			},
		},
		{
			name: "Multiple validation decorators",
			input: `package main

type CreateUserDto struct {
	Email    string ` + "`" + `validate:"@IsEmail()"` + "`" + `
	Age      int    ` + "`" + `validate:"@Min(18) @Max(120)"` + "`" + `
	Name     string ` + "`" + `validate:"@IsNotEmpty() @Length(2,50)"` + "`" + `
	Tags     []string ` + "`" + `validate:"@IsArray() @ArrayMinSize(1)"` + "`" + `
}

@Controller("/users")
type UserController struct {
	service UserService ` + "`" + `inject:"userService"` + "`" + `
}

@Post("/")
func (c *UserController) createUser(@Body() data CreateUserDto) User {
	return User{}
}`,
			expected: []string{
				"ValidationError",
				"ValidateCreateUserDto",
				"isValidEmail",
				"must be a valid email address",
				"must be at least 18",
				"must be at most 120",
				"must not be empty", 
				"must be between 2 and 50 characters",
				"must be an array",
				"must contain at least 1 item(s)",
				"MIN_VALUE",
				"MAX_VALUE",
				"IS_NOT_EMPTY",
				"LENGTH",
				"IS_ARRAY",
				"ARRAY_MIN_SIZE",
			},
		},
		{
			name: "Format validation decorators", 
			input: `package main

type UserProfileDto struct {
	Website     string ` + "`" + `validate:"@IsURL()"` + "`" + `
	PhoneNumber string ` + "`" + `validate:"@IsNumeric()"` + "`" + `
	Username    string ` + "`" + `validate:"@IsAlphanumeric()"` + "`" + `
	Bio         string ` + "`" + `validate:"@IsAlpha()"` + "`" + `
}

@Controller("/profiles")
type ProfileController struct {
}

@Post("/")
func (c *ProfileController) updateProfile(@Body() data UserProfileDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateUserProfileDto",
				"isValidURL",
				"isNumeric",
				"isAlphanumeric", 
				"isAlpha",
				"must be a valid URL",
				"must contain only numbers",
				"must contain only letters and numbers",
				"must contain only letters",
				"IS_URL",
				"IS_NUMERIC",
				"IS_ALPHANUMERIC", 
				"IS_ALPHA",
			},
		},
		{
			name: "Business logic validation decorators",
			input: `package main

type ProductDto struct {
	Price  float64 ` + "`" + `validate:"@IsPositive()"` + "`" + `
	Stock  int     ` + "`" + `validate:"@Min(0)"` + "`" + `
	Rating float64 ` + "`" + `validate:"@Min(0) @Max(5)"` + "`" + `
}

@Controller("/products")
type ProductController struct {
}

@Post("/")
func (c *ProductController) createProduct(@Body() data ProductDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateProductDto",
				"must be a positive number",
				"must be at least 0",
				"must be at most 5",
				"IS_POSITIVE",
				"MIN_VALUE",
				"MAX_VALUE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			file, err := parser.ParseFile()
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Generate Go code
			generator := NewCodeGenerator("main")
			goCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check if expected strings are present in generated code
			for _, expected := range tt.expected {
				if !strings.Contains(goCode, expected) {
					t.Errorf("Expected to find '%s' in generated code but didn't.\nGenerated code:\n%s", expected, goCode)
				}
			}

			// Print generated code for debugging (optional)
			t.Logf("Generated code:\n%s", goCode)
		})
	}
}

func TestValidationHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "Email validation helper",
			input: `package main

type UserDto struct {
	Email string ` + "`" + `validate:"@IsEmail()"` + "`" + `
}

@Controller("/users")
type UserController struct{}`,
			expected: []string{
				"func isValidEmail(email string) bool",
				"emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$`",
				"regexp.MatchString(emailRegex, email)",
			},
		},
		{
			name: "URL validation helper",
			input: `package main

type LinkDto struct {
	Website string ` + "`" + `validate:"@IsURL()"` + "`" + `
}

@Controller("/links")
type LinkController struct{}`,
			expected: []string{
				"func isValidURL(url string) bool",
				"url.Parse(url)",
			},
		},
		{
			name: "Multiple validation helpers",
			input: `package main

type ComplexDto struct {
	Email    string ` + "`" + `validate:"@IsEmail()"` + "`" + `
	Website  string ` + "`" + `validate:"@IsURL()"` + "`" + `
	Username string ` + "`" + `validate:"@IsAlphanumeric()"` + "`" + `
	Code     string ` + "`" + `validate:"@IsNumeric()"` + "`" + `
}

@Controller("/complex")
type ComplexController struct{}`,
			expected: []string{
				"func isValidEmail(email string) bool",
				"func isValidURL(url string) bool", 
				"func isAlphanumeric(str string) bool",
				"func isNumeric(str string) bool",
				"alphanumericRegex := `^[a-zA-Z0-9]+$`",
				"numericRegex := `^[0-9]+$`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			file, err := parser.ParseFile()
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Generate Go code
			generator := NewCodeGenerator("main")
			goCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check if expected helper functions are present
			for _, expected := range tt.expected {
				if !strings.Contains(goCode, expected) {
					t.Errorf("Expected to find '%s' in generated code but didn't.\nGenerated code:\n%s", expected, goCode)
				}
			}
		})
	}
}

func TestValidationStructGeneration(t *testing.T) {
	input := `package main

type TestDto struct {
	Field1 string ` + "`" + `validate:"@IsEmail()"` + "`" + `
}

@Controller("/test")
type TestController struct{}`

	// Parse the input
	lexer := NewLexer(input)
	parser := NewParser(lexer)
	file, err := parser.ParseFile()
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Generate Go code
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Check ValidationError struct generation
	expectedStructs := []string{
		"type ValidationError struct {",
		"Field   string      `json:\"field\"`",
		"Value   interface{} `json:\"value\"`",
		"Message string      `json:\"message\"`", 
		"Code    string      `json:\"code\"`",
		"type ValidationResult struct {",
		"IsValid bool              `json:\"isValid\"`",
		"Errors  []ValidationError `json:\"errors,omitempty\"`",
	}

	for _, expected := range expectedStructs {
		if !strings.Contains(goCode, expected) {
			t.Errorf("Expected to find ValidationError struct definition '%s' in generated code", expected)
		}
	}
}

func TestValidationDecoratorParsing(t *testing.T) {
	generator := NewCodeGenerator("main")
	
	tests := []struct {
		name     string
		tagContent string
		expected []ValidationRule
	}{
		{
			name: "Single decorator",
			tagContent: "@IsEmail()",
			expected: []ValidationRule{
				{
					Type:    "IsEmail",
					Args:    []interface{}{},
					Message: "must be a valid email address", 
					Code:    "ISEMAIL",
				},
			},
		},
		{
			name: "Multiple decorators",
			tagContent: "@Min(18) @Max(120)",
			expected: []ValidationRule{
				{
					Type:    "Min",
					Args:    []interface{}{18},
					Message: "must be at least 18",
					Code:    "MIN",
				},
				{
					Type:    "Max", 
					Args:    []interface{}{120},
					Message: "must be at most 120",
					Code:    "MAX",
				},
			},
		},
		{
			name: "Decorator with multiple args",
			tagContent: "@Length(2,50)",
			expected: []ValidationRule{
				{
					Type:    "Length",
					Args:    []interface{}{2, 50},
					Message: "must be between 2 and 50 characters",
					Code:    "LENGTH",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := generator.parseValidationDecorators(`validate:"` + tt.tagContent + `"`)
			
			if len(rules) != len(tt.expected) {
				t.Fatalf("Expected %d rules, got %d", len(tt.expected), len(rules))
			}
			
			for i, rule := range rules {
				expected := tt.expected[i]
				
				if rule.Type != expected.Type {
					t.Errorf("Expected rule type '%s', got '%s'", expected.Type, rule.Type)
				}
				
				if len(rule.Args) != len(expected.Args) {
					t.Errorf("Expected %d args, got %d", len(expected.Args), len(rule.Args))
					continue
				}
				
				for j, arg := range rule.Args {
					if arg != expected.Args[j] {
						t.Errorf("Expected arg[%d] to be %v, got %v", j, expected.Args[j], arg)
					}
				}
				
				if rule.Message != expected.Message {
					t.Errorf("Expected message '%s', got '%s'", expected.Message, rule.Message)
				}
			}
		})
	}
}

func TestValidationImportGeneration(t *testing.T) {
	input := `package main

type TestDto struct {
	Email   string ` + "`" + `validate:"@IsEmail()"` + "`" + `
	Website string ` + "`" + `validate:"@IsURL()"` + "`" + `
}

@Controller("/test")
type TestController struct{}`

	// Parse the input
	lexer := NewLexer(input)
	parser := NewParser(lexer)
	file, err := parser.ParseFile()
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Generate Go code
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Check that required imports are added for validation
	expectedImports := []string{
		"strings",
		"regexp", 
		"net/url",
	}

	for _, expectedImport := range expectedImports {
		if !strings.Contains(goCode, expectedImport) {
			t.Errorf("Expected to find import %s in generated code", expectedImport)
		}
	}
}