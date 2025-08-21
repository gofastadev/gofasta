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

@Injectable()
type CreateUserDto struct {
	@IsEmail()
	Email string
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

@Injectable()
type CreateUserDto struct {
	@IsEmail()
	Email string
	
	@Min(18)
	@Max(120)
	Age int
	
	@IsNotEmpty()
	@Length(2,50)
	Name string
	
	@IsArray()
	@ArrayMinSize(1)
	Tags []string
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

@Injectable()
type UserProfileDto struct {
	@IsURL()
	Website string
	
	@IsNumeric()
	PhoneNumber string
	
	@IsAlphanumeric()
	Username string
	
	@IsAlpha()
	Bio string
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

@Injectable()
type ProductDto struct {
	@IsPositive()
	Price float64
	
	@Min(0)
	Stock int
	
	@Min(0)
	@Max(5)
	Rating float64
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
		{
			name: "IsInt validation decorator",
			input: `package main

@Injectable()
type TestDto struct {
	@IsInt()
	Age int
	
	@IsInt()
	ID interface{}
}

@Controller("/test")
type TestController struct {
}

@Post("/")
func (c *TestController) createTest(@Body() data TestDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateTestDto",
				"func isInt(value interface{}) bool",
				"IsInt validation",
				"if !isInt(dto.Age) {",
				"if !isInt(dto.ID) {",
				"must be an integer",
				"IS_INT",
			},
		},
		{
			name: "IsFloat validation decorator",
			input: `package main

@Injectable()
type FloatTestDto struct {
	@IsFloat()
	Price float64
	
	@IsFloat()
	@Min(0.0)
	@Max(100.0)
	Score float32
	
	@IsFloat()
	Value interface{}
}

@Controller("/float-test")
type FloatTestController struct {
}

@Post("/")
func (c *FloatTestController) createTest(@Body() data FloatTestDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateFloatTestDto",
				"func isFloat(value interface{}) bool",
				"IsFloat validation",
				"if !isFloat(dto.Price) {",
				"if !isFloat(dto.Score) {",
				"if !isFloat(dto.Value) {",
				"must be a floating point number",
				"IS_FLOAT",
			},
		},
		{
			name: "IsBoolean validation decorator",
			input: `package main

@Injectable()
type BooleanTestDto struct {
	@IsBoolean()
	IsActive bool
	
	@IsBoolean()
	IsEnabled interface{}
	
	@IsBoolean()
	IsPublished bool
}

@Controller("/boolean-test")
type BooleanTestController struct {
}

@Post("/")
func (c *BooleanTestController) createTest(@Body() data BooleanTestDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateBooleanTestDto",
				"func isBoolean(value interface{}) bool",
				"IsBoolean validation",
				"if !isBoolean(dto.IsActive) {",
				"if !isBoolean(dto.IsEnabled) {",
				"if !isBoolean(dto.IsPublished) {",
				"must be a boolean",
				"IS_BOOLEAN",
			},
		},
		{
			name: "IsDate validation decorator",
			input: `package main

@Injectable()
type DateTestDto struct {
	@IsDate()
	CreatedAt string
	
	@IsDate()
	UpdatedAt interface{}
	
	@IsDate()
	ExpiresAt string
}

@Controller("/date-test")
type DateTestController struct {
}

@Post("/")
func (c *DateTestController) createTest(@Body() data DateTestDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDateTestDto",
				"func isDate(value interface{}) bool",
				"IsDate validation",
				"if !isDate(dto.CreatedAt) {",
				"if !isDate(dto.UpdatedAt) {",
				"if !isDate(dto.ExpiresAt) {",
				"must be a valid date",
				"IS_DATE",
				"time.RFC3339",
				"time.Parse",
			},
		},
		{
			name: "IsIP validation decorator",
			input: `package main

@Injectable()
type NetworkDto struct {
	@IsIP()
	ServerIP string
	
	@IsIP()
	ClientIP interface{}
	
	@IsIP()
	GatewayIP string
}

@Controller("/network")
type NetworkController struct {
}

@Post("/")
func (c *NetworkController) createTest(@Body() data NetworkDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateNetworkDto",
				"func isIP(value interface{}) bool",
				"IsIP validation",
				"if !isIP(dto.ServerIP) {",
				"if !isIP(dto.ClientIP) {",
				"if !isIP(dto.GatewayIP) {",
				"must be a valid IP address",
				"IS_IP",
				"net.ParseIP",
			},
		},
		{
			name: "IsJSON validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@IsJSON()
	Config string
	
	@IsJSON()
	Metadata interface{}
	
	@IsJSON()
	Settings string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"func isJSON(value interface{}) bool",
				"IsJSON validation",
				"if !isJSON(dto.Config) {",
				"if !isJSON(dto.Metadata) {",
				"if !isJSON(dto.Settings) {",
				"must be valid JSON",
				"IS_JSON",
				"json.Valid",
			},
		},
		{
			name: "IsHexColor validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@IsHexColor()
	PrimaryColor string
	
	@IsHexColor()
	BackgroundColor string
	
	@IsHexColor()
	BorderColor string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"func isHexColor(value interface{}) bool",
				"IsHexColor validation",
				"if !isHexColor(dto.PrimaryColor) {",
				"if !isHexColor(dto.BackgroundColor) {",
				"if !isHexColor(dto.BorderColor) {",
				"must be a valid hex color",
				"IS_HEX_COLOR",
				"strings.HasPrefix",
			},
		},
		{
			name: "IsPhoneNumber validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@IsPhoneNumber()
	HomePhone string
	
	@IsPhoneNumber()
	MobilePhone string
	
	@IsPhoneNumber()
	WorkPhone string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"func isPhoneNumber(value interface{}) bool",
				"IsPhoneNumber validation",
				"if !isPhoneNumber(dto.HomePhone) {",
				"if !isPhoneNumber(dto.MobilePhone) {",
				"if !isPhoneNumber(dto.WorkPhone) {",
				"must be a valid phone number",
				"IS_PHONE_NUMBER",
				"strings.Map",
			},
		},
		{
			name: "IsCreditCard validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@IsCreditCard()
	CreditCardNumber string
	
	@IsCreditCard()
	BackupCard string
	
	@IsCreditCard()
	BusinessCard string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"func isCreditCard(value interface{}) bool",
				"IsCreditCard validation",
				"if !isCreditCard(dto.CreditCardNumber) {",
				"if !isCreditCard(dto.BackupCard) {",
				"if !isCreditCard(dto.BusinessCard) {",
				"must be a valid credit card number",
				"IS_CREDIT_CARD",
				"Luhn algorithm",
			},
		},
		{
			name: "IsISBN validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@IsISBN()
	BookISBN string
	
	@IsISBN()
	Magazine string
	
	@IsISBN()
	Reference string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"func isISBN(value interface{}) bool",
				"IsISBN validation",
				"if !isISBN(dto.BookISBN) {",
				"if !isISBN(dto.Magazine) {",
				"if !isISBN(dto.Reference) {",
				"must be a valid ISBN",
				"IS_ISBN",
				"isISBN10",
				"isISBN13",
			},
		},
		{
			name: "IsBase64 validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@IsBase64()
	EncodedData string
	
	@IsBase64()
	Image string
	
	@IsBase64()
	Document string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"func isBase64(value interface{}) bool",
				"IsBase64 validation",
				"if !isBase64(dto.EncodedData) {",
				"if !isBase64(dto.Image) {",
				"if !isBase64(dto.Document) {",
				"must be valid Base64",
				"IS_BASE64",
				"base64.StdEncoding.DecodeString",
			},
		},
		{
			name: "MinLength validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@MinLength(5)
	Username string
	
	@MinLength(8)
	Password string
	
	@MinLength(3)
	Name string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"MinLength validation",
				"if len(dto.Username) < 5 {",
				"if len(dto.Password) < 8 {",
				"if len(dto.Name) < 3 {",
				"must be at least 5 characters long",
				"must be at least 8 characters long",
				"must be at least 3 characters long",
				"MIN_LENGTH",
			},
		},
		{
			name: "MaxLength validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@MaxLength(20)
	Username string
	
	@MaxLength(100)
	Description string
	
	@MaxLength(50)
	Title string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"MaxLength validation",
				"if len(dto.Username) > 20 {",
				"if len(dto.Description) > 100 {",
				"if len(dto.Title) > 50 {",
				"must be at most 20 characters long",
				"must be at most 100 characters long",
				"must be at most 50 characters long",
				"MAX_LENGTH",
			},
		},
		{
			name: "ArrayMaxSize validation decorator",
			input: `package main

@Injectable()
type DataDto struct {
	@ArrayMaxSize(10)
	Tags []string
	
	@ArrayMaxSize(5)
	Categories []string
	
	@ArrayMaxSize(3)
	Labels []string
}

@Controller("/data")
type DataController struct {
}

@Post("/")
func (c *DataController) createTest(@Body() data DataDto) {
}`,
			expected: []string{
				"ValidationError",
				"ValidateDataDto",
				"ArrayMaxSize validation",
				"if dto.Tags != nil && len(dto.Tags) > 10 {",
				"if dto.Categories != nil && len(dto.Categories) > 5 {",
				"if dto.Labels != nil && len(dto.Labels) > 3 {",
				"must contain at most 10 item(s)",
				"must contain at most 5 item(s)",
				"must contain at most 3 item(s)",
				"ARRAY_MAX_SIZE",
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

@Injectable()
type UserDto struct {
	@IsEmail()
	Email string
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

@Injectable()
type LinkDto struct {
	@IsURL()
	Website string
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

@Injectable()
type ComplexDto struct {
	@IsEmail()
	Email string
	
	@IsURL()
	Website string
	
	@IsAlphanumeric()
	Username string
	
	@IsNumeric()
	Code string
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

@Injectable()
type TestDto struct {
	@IsEmail()
	Field1 string
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
		decorators []*DecoratorNode
		expected []ValidationRule
	}{
		{
			name: "Single decorator",
			decorators: []*DecoratorNode{
				{
					Name: "IsEmail",
					Args: []DecoratorArg{},
				},
			},
			expected: []ValidationRule{
				{
					Type:    "IsEmail",
					Args:    []interface{}{},
					Message: "must be a valid email address", 
					Code:    "IS_EMAIL",
				},
			},
		},
		{
			name: "Decorator with args",
			decorators: []*DecoratorNode{
				{
					Name: "Min",
					Args: []DecoratorArg{
						{Value: 18},
					},
				},
			},
			expected: []ValidationRule{
				{
					Type:    "Min",
					Args:    []interface{}{18},
					Message: "must be at least 18",
					Code:    "MIN_VALUE",
				},
			},
		},
		{
			name: "Decorator with multiple args",
			decorators: []*DecoratorNode{
				{
					Name: "Length",
					Args: []DecoratorArg{
						{Value: 2},
						{Value: 50},
					},
				},
			},
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
			field := &FieldNode{
				Name: "TestField",
				Type: "string",
				Decorators: tt.decorators,
			}
			rules := generator.parseValidationDecoratorsFromField(field)
			
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

@Injectable()
type TestDto struct {
	@IsEmail()
	Email string
	
	@IsURL()
	Website string
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