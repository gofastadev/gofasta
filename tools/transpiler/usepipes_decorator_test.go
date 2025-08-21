package transpiler

import (
	"strings"
	"testing"
)

func TestUsePipesDecorator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected patterns in generated code
		wantErr  bool
	}{
		{
			name: "Single pipe on method",
			input: `package test

type UserController struct {}

@UsePipes(ValidationPipe)
@Post("/users")
func createUser(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/user/users\", c.ValidationPipe, c.createUser)",
				"func (c *UserController) ValidationPipe(ctx *httpPackage.RequestContext) {",
				"// Validation pipe logic",
				"requestBody := ctx.GetRequestBody()",
				"if requestBody == nil {",
				"ctx.JSON(400, map[string]string{\"error\": \"Request body is required\"})",
				"ctx.Abort()",
				"return",
				"}",
				"contentType := ctx.GetHeader(\"Content-Type\")",
				"if !strings.Contains(contentType, \"application/json\") {",
				"ctx.JSON(400, map[string]string{\"error\": \"Content-Type must be application/json\"})",
				"if err := validateRequestBody(requestBody); err != nil {",
				"ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Validation failed: %s\", err.Error())})",
				"ctx.Next()",
			},
			wantErr: false,
		},
		{
			name: "Multiple pipes on method",
			input: `package test

type DataController struct {}

@UsePipes(ValidationPipe, TransformPipe)
@Post("/data")
func createData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/data/data\", c.ValidationPipe, c.TransformPipe, c.createData)",
				"func (c *DataController) ValidationPipe(ctx *httpPackage.RequestContext) {",
				"func (c *DataController) TransformPipe(ctx *httpPackage.RequestContext) {",
				"// Validation pipe logic",
				"// Transform pipe logic",
				"transformedData := transformRequestData(requestData)",
				"ctx.SetRequestData(transformedData)",
				"transformedValue := transformQueryValue(key, value)",
				"ctx.SetQueryParam(key, transformedValue)",
			},
			wantErr: false,
		},
		{
			name: "Controller-level pipe",
			input: `package test

@Controller("/api")
@UsePipes(ValidationPipe)
type ApiController struct {}

@Post("/data")
func createData(ctx *RequestContext) {
}

@Put("/data")
func updateData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/api/data\", c.ValidationPipe, c.createData)",
				"server.Put(\"/api/data\", c.ValidationPipe, c.updateData)",
				"func (c *ApiController) ValidationPipe(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Controller and method level pipes combined",
			input: `package test

@Controller("/admin")
@UsePipes(ValidationPipe)
type AdminController struct {}

@UsePipes(TransformPipe)
@Post("/users")
func createUser(ctx *RequestContext) {
}

@Get("/users")
func getUsers(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/admin/users\", c.ValidationPipe, c.TransformPipe, c.createUser)",
				"server.Get(\"/admin/users\", c.ValidationPipe, c.getUsers)",
				"func (c *AdminController) ValidationPipe(ctx *httpPackage.RequestContext) {",
				"func (c *AdminController) TransformPipe(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "ParseInt pipe",
			input: `package test

type NumberController struct {}

@UsePipes(ParseIntPipe)
@Get("/numbers/:id")
func getNumber(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/number/numbers/:id\", c.ParseIntPipe, c.getNumber)",
				"func (c *NumberController) ParseIntPipe(ctx *httpPackage.RequestContext) {",
				"// ParseInt pipe logic",
				"pathParams := ctx.GetPathParams()",
				"if intValue, err := strconv.Atoi(strValue); err == nil {",
				"ctx.SetPathParam(key, intValue)",
				"ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Invalid integer value for parameter %s: %s\", key, strValue)})",
				"queryParams := ctx.GetQueryParams()",
				"ctx.SetQueryParam(key, intValue)",
			},
			wantErr: false,
		},
		{
			name: "ParseBool pipe",
			input: `package test

type BoolController struct {}

@UsePipes(ParseBoolPipe)
@Get("/flags")
func getFlags(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/bool/flags\", c.ParseBoolPipe, c.getFlags)",
				"func (c *BoolController) ParseBoolPipe(ctx *httpPackage.RequestContext) {",
				"// ParseBool pipe logic",
				"if boolValue, err := strconv.ParseBool(strValue); err == nil {",
				"ctx.SetQueryParam(key, boolValue)",
				"strings.ToLower(strValue) == \"true\"",
			},
			wantErr: false,
		},
		{
			name: "ParseArray pipe",
			input: `package test

type ArrayController struct {}

@UsePipes(ParseArrayPipe)
@Get("/items")
func getItems(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/array/items\", c.ParseArrayPipe, c.getItems)",
				"func (c *ArrayController) ParseArrayPipe(ctx *httpPackage.RequestContext) {",
				"// ParseArray pipe logic",
				"strings.Contains(strValue, \",\")",
				"arrayValue := strings.Split(strValue, \",\")",
				"arrayValue[i] = strings.TrimSpace(item)",
				"ctx.SetQueryParam(key, arrayValue)",
			},
			wantErr: false,
		},
		{
			name: "DefaultValue pipe",
			input: `package test

type DefaultController struct {}

@UsePipes(DefaultValuePipe)
@Get("/search")
func search(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/default/search\", c.DefaultValuePipe, c.search)",
				"func (c *DefaultController) DefaultValuePipe(ctx *httpPackage.RequestContext) {",
				"// DefaultValue pipe logic",
				"defaultValues := map[string]interface{}{",
				"\"limit\":  10,",
				"\"offset\": 0,",
				"\"sort\":   \"created_at\",",
				"\"order\":  \"desc\",",
				"ctx.SetQueryParam(key, defaultValue)",
			},
			wantErr: false,
		},
		{
			name: "Custom pipe",
			input: `package test

type CustomController struct {}

@UsePipes(MyCustomPipe)
@Post("/custom")
func createCustom(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/custom/custom\", c.MyCustomPipe, c.createCustom)",
				"func (c *CustomController) MyCustomPipe(ctx *httpPackage.RequestContext) {",
				"// MyCustomPipe pipe logic",
				"// TODO: Implement your custom pipe logic here",
				"// Example: Data validation",
				"// requestData := ctx.GetRequestData()",
				"// if err := validateData(requestData); err != nil {",
				"// Example: Data transformation",
				"// transformedData := transformData(requestData)",
				"// ctx.SetRequestData(transformedData)",
			},
			wantErr: false,
		},
		{
			name: "Mixed pipes, interceptors, and guards",
			input: `package test

@Controller("/api")
@UsePipes(ValidationPipe)
@UseInterceptors(LoggingInterceptor)
@UseGuards(AuthGuard)
type MixedController struct {}

@UsePipes(TransformPipe)
@UseInterceptors(CacheInterceptor)
@UseGuards(RoleGuard)
@Post("/protected")
func createProtected(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/api/protected\", c.ValidationPipe, c.TransformPipe, c.LoggingInterceptor, c.CacheInterceptor, c.AuthGuard, c.RoleGuard, c.createProtected)",
				"func (c *MixedController) ValidationPipe(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) TransformPipe(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) CacheInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) AuthGuard(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) RoleGuard(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Multiple pipes with different methods",
			input: `package test

@Controller("/api")
type MultiController struct {}

@UsePipes(ValidationPipe)
@Post("/validate")
func createValidated(ctx *RequestContext) {
}

@UsePipes(ParseIntPipe, DefaultValuePipe)
@Get("/numbers")
func getNumbers(ctx *RequestContext) {
}

@UsePipes(TransformPipe, ValidationPipe, ParseArrayPipe)
@Put("/complex")
func updateComplex(ctx *RequestContext) {
}

@Get("/simple")
func getSimple(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/api/validate\", c.ValidationPipe, c.createValidated)",
				"server.Get(\"/api/numbers\", c.ParseIntPipe, c.DefaultValuePipe, c.getNumbers)",
				"server.Put(\"/api/complex\", c.TransformPipe, c.ValidationPipe, c.ParseArrayPipe, c.updateComplex)",
				"server.Get(\"/api/simple\", c.getSimple)", // No pipes
				"func (c *MultiController) ValidationPipe(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) ParseIntPipe(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) DefaultValuePipe(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) TransformPipe(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) ParseArrayPipe(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			file, err := ParseGofaFile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGofaFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Generate Go code
			generator := NewCodeGenerator("test")
			output, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("GenerateGoCode() error = %v", err)
			}

			// Check expected patterns
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected pattern not found in output: %s", expected)
					t.Logf("Generated code:\n%s", output)
				}
			}

			// Additional validations
			if strings.Contains(tt.input, "@UsePipes") {
				// Ensure pipe middleware methods are generated
				if !strings.Contains(output, "// Pipe middleware methods") {
					t.Errorf("Pipe middleware methods section not found")
				}
			}

			// Ensure no syntax errors in generated Go code (basic check)
			if strings.Count(output, "{") != strings.Count(output, "}") {
				t.Errorf("Unbalanced braces in generated code")
			}
		})
	}
}

func TestUsePipesDecoratorParsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedPipes []string
		wantErr       bool
	}{
		{
			name: "Single pipe",
			input: `package test

type TestController struct {}

@UsePipes(ValidationPipe)
@Post("/test")
func (c *TestController) createTest() {}`,
			expectedPipes: []string{"ValidationPipe"},
			wantErr:       false,
		},
		{
			name: "Multiple pipes",
			input: `package test

type TestController struct {}

@UsePipes(ValidationPipe, TransformPipe, ParseIntPipe)
@Post("/test")
func (c *TestController) createTest() {}`,
			expectedPipes: []string{"ValidationPipe", "TransformPipe", "ParseIntPipe"},
			wantErr:       false,
		},
		{
			name: "No pipes",
			input: `package test

type TestController struct {}

@Post("/test")
func (c *TestController) createTest() {}`,
			expectedPipes: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGofaFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Find the controller and method
			if len(file.Declarations) == 0 {
				t.Fatal("No declarations found")
			}

			controller, ok := file.Declarations[0].(*ControllerDeclaration)
			if !ok {
				t.Fatal("First declaration is not a controller")
			}

			if len(controller.Methods) == 0 {
				t.Fatal("No methods found in controller")
			}

			method := controller.Methods[0]

			// Use code generator to extract pipes
			generator := NewCodeGenerator("test")
			actualPipes := generator.getPipeDecorators(method.Decorators)

			if len(actualPipes) != len(tt.expectedPipes) {
				t.Fatalf("Expected %d pipes, got %d", len(tt.expectedPipes), len(actualPipes))
			}

			for i, expected := range tt.expectedPipes {
				if actualPipes[i] != expected {
					t.Errorf("Expected pipe %s at position %d, got %s", expected, i, actualPipes[i])
				}
			}
		})
	}
}

func TestPipeMiddlewareGeneration(t *testing.T) {
	input := `package test

@Controller("/api")
@UsePipes(ValidationPipe)
type ApiController struct {}

@UsePipes(TransformPipe)
@Post("/transform")
func createTransformed(ctx *RequestContext) {
}

@Get("/public")  
func getPublic(ctx *RequestContext) {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("ParseGofaFile() error = %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("GenerateGoCode() error = %v", err)
	}

	// Test specific pipe middleware patterns
	tests := []struct {
		pattern     string
		shouldExist bool
	}{
		{"func (c *ApiController) ValidationPipe(ctx *httpPackage.RequestContext)", true},
		{"func (c *ApiController) TransformPipe(ctx *httpPackage.RequestContext)", true},
		{"// Validation pipe logic", true},
		{"// Transform pipe logic", true},
		{"requestBody := ctx.GetRequestBody()", true},
		{"transformedData := transformRequestData(requestData)", true},
		{"ctx.Abort()", true},
		{"ctx.Next()", true},
		{"server.Post(\"/api/transform\", c.ValidationPipe, c.TransformPipe, c.createTransformed)", true},
		{"server.Get(\"/api/public\", c.ValidationPipe, c.getPublic)", true},
	}

	for _, test := range tests {
		exists := strings.Contains(output, test.pattern)
		if exists != test.shouldExist {
			t.Errorf("Pattern '%s' existence: expected %v, got %v", test.pattern, test.shouldExist, exists)
			if !exists && test.shouldExist {
				t.Logf("Generated code:\n%s", output)
			}
		}
	}
}

func TestPipeMiddlewareOrder(t *testing.T) {
	input := `package test

@Controller("/secure")
@UsePipes(ValidationPipe)
@UseInterceptors(LoggingInterceptor)
@UseGuards(AuthGuard)
type SecureController struct {}

@UsePipes(TransformPipe, ParseIntPipe)
@UseInterceptors(CacheInterceptor, ValidationInterceptor)
@UseGuards(RoleGuard, PermissionGuard)
@Post("/data")
func createData(ctx *RequestContext) {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("ParseGofaFile() error = %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("GenerateGoCode() error = %v", err)
	}

	// Test that pipes come before interceptors which come before guards in the middleware chain
	expectedRoute := "server.Post(\"/secure/data\", c.ValidationPipe, c.TransformPipe, c.ParseIntPipe, c.LoggingInterceptor, c.CacheInterceptor, c.ValidationInterceptor, c.AuthGuard, c.RoleGuard, c.PermissionGuard, c.createData)"
	if !strings.Contains(output, expectedRoute) {
		t.Errorf("Expected middleware order not found. Expected: %s", expectedRoute)
		t.Logf("Generated code:\n%s", output)
	}

	// Test that all middleware functions are generated
	middlewareFunctions := []string{
		"func (c *SecureController) ValidationPipe(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) TransformPipe(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) ParseIntPipe(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) LoggingInterceptor(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) CacheInterceptor(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) ValidationInterceptor(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) AuthGuard(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) RoleGuard(ctx *httpPackage.RequestContext)",
		"func (c *SecureController) PermissionGuard(ctx *httpPackage.RequestContext)",
	}

	for _, function := range middlewareFunctions {
		if !strings.Contains(output, function) {
			t.Errorf("Expected middleware function not found: %s", function)
		}
	}
}

func TestPipeLogicGeneration(t *testing.T) {
	input := `package test

type TestController struct {}

@UsePipes(ValidationPipe, TransformPipe, ParseIntPipe, ParseBoolPipe, ParseArrayPipe, DefaultValuePipe)
@Post("/test")
func (c *TestController) createTest() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("ParseGofaFile() error = %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("GenerateGoCode() error = %v", err)
	}

	// Test that each pipe type generates appropriate logic
	pipeTests := []struct {
		pipe     string
		patterns []string
	}{
		{"ValidationPipe", []string{
			"requestBody := ctx.GetRequestBody()",
			"if requestBody == nil {",
			"ctx.JSON(400, map[string]string{\"error\": \"Request body is required\"})",
			"contentType := ctx.GetHeader(\"Content-Type\")",
			"if !strings.Contains(contentType, \"application/json\") {",
			"if err := validateRequestBody(requestBody); err != nil {",
		}},
		{"TransformPipe", []string{
			"requestData := ctx.GetRequestData()",
			"transformedData := transformRequestData(requestData)",
			"ctx.SetRequestData(transformedData)",
			"transformedValue := transformQueryValue(key, value)",
			"ctx.SetQueryParam(key, transformedValue)",
		}},
		{"ParseIntPipe", []string{
			"pathParams := ctx.GetPathParams()",
			"if intValue, err := strconv.Atoi(strValue); err == nil {",
			"ctx.SetPathParam(key, intValue)",
			"ctx.JSON(400, map[string]string{\"error\": fmt.Sprintf(\"Invalid integer value for parameter %s: %s\", key, strValue)})",
		}},
		{"ParseBoolPipe", []string{
			"if boolValue, err := strconv.ParseBool(strValue); err == nil {",
			"ctx.SetQueryParam(key, boolValue)",
			"strings.ToLower(strValue) == \"true\"",
		}},
		{"ParseArrayPipe", []string{
			"strings.Contains(strValue, \",\")",
			"arrayValue := strings.Split(strValue, \",\")",
			"arrayValue[i] = strings.TrimSpace(item)",
		}},
		{"DefaultValuePipe", []string{
			"defaultValues := map[string]interface{}{",
			"\"limit\":  10,",
			"\"offset\": 0,",
			"ctx.SetQueryParam(key, defaultValue)",
		}},
	}

	for _, test := range pipeTests {
		for _, pattern := range test.patterns {
			if !strings.Contains(output, pattern) {
				t.Errorf("Expected pattern for %s not found: %s", test.pipe, pattern)
			}
		}
	}
}