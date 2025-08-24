package codegen

import (
	"strings"
	"testing"
)

func TestUseInterceptorsDecorator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected patterns in generated code
		wantErr  bool
	}{
		{
			name: "Single interceptor on method",
			input: `package test

type UserController struct {}

@UseInterceptors(LoggingInterceptor)
@Get("/profile")
func getProfile(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/user/profile\", c.LoggingInterceptor, c.getProfile)",
				"func (c *UserController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
				"// Logging interceptor logic",
				"startTime := time.Now()",
				"method := ctx.GetMethod()",
				"path := ctx.GetPath()",
				"clientIP := ctx.GetClientIP()",
				"fmt.Printf(\"[%s] %s %s from %s\\n\", startTime.Format(time.RFC3339), method, path, clientIP)",
				"ctx.Set(\"request_start_time\", startTime)",
				"ctx.OnFinished(func() {",
				"duration := time.Since(startTime)",
				"status := ctx.GetStatusCode()",
				"fmt.Printf(\"[%s] %s %s %d - %v\\n\", time.Now().Format(time.RFC3339), method, path, status, duration)",
				"// Continue to next middleware/handler",
				"ctx.Next()",
			},
			wantErr: false,
		},
		{
			name: "Multiple interceptors on method",
			input: `package test

type DataController struct {}

@UseInterceptors(LoggingInterceptor, CacheInterceptor)
@Get("/data")
func getData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/data/data\", c.LoggingInterceptor, c.CacheInterceptor, c.getData)",
				"func (c *DataController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *DataController) CacheInterceptor(ctx *httpPackage.RequestContext) {",
				"// Cache interceptor logic",
				"cacheKey := fmt.Sprintf(\"%s:%s\", ctx.GetMethod(), ctx.GetPath())",
				"if cachedResponse := getFromCache(cacheKey); cachedResponse != nil {",
				"ctx.JSON(200, cachedResponse)",
				"return // Skip further processing",
			},
			wantErr: false,
		},
		{
			name: "Controller-level interceptor",
			input: `package test

@Controller("/api")
@UseInterceptors(LoggingInterceptor)
type ApiController struct {}

@Get("/data")
func getData(ctx *RequestContext) {
}

@Post("/data")
func createData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/api/data\", c.LoggingInterceptor, c.getData)",
				"server.Post(\"/api/data\", c.LoggingInterceptor, c.createData)",
				"func (c *ApiController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Controller and method level interceptors combined",
			input: `package test

@Controller("/admin")
@UseInterceptors(LoggingInterceptor)
type AdminController struct {}

@UseInterceptors(CacheInterceptor)
@Get("/users")
func getUsers(ctx *RequestContext) {
}

@Get("/public")
func getPublic(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/admin/users\", c.LoggingInterceptor, c.CacheInterceptor, c.getUsers)",
				"server.Get(\"/admin/public\", c.LoggingInterceptor, c.getPublic)",
				"func (c *AdminController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *AdminController) CacheInterceptor(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Transform interceptor",
			input: `package test

type TransformController struct {}

@UseInterceptors(TransformInterceptor)
@Post("/transform")
func transformData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/transform/transform\", c.TransformInterceptor, c.transformData)",
				"func (c *TransformController) TransformInterceptor(ctx *httpPackage.RequestContext) {",
				"// Transform interceptor logic",
				"if requestBody := ctx.GetRequestBody(); requestBody != nil {",
				"transformedBody := transformRequestData(requestBody)",
				"ctx.SetRequestBody(transformedBody)",
				"ctx.OnBeforeResponse(func() {",
				"transformedResponse := transformResponseData(responseData)",
				"ctx.SetResponseData(transformedResponse)",
			},
			wantErr: false,
		},
		{
			name: "Validation interceptor",
			input: `package test

type ValidationController struct {}

@UseInterceptors(ValidationInterceptor)
@Post("/validate")
func validateData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Post(\"/validation/validate\", c.ValidationInterceptor, c.validateData)",
				"func (c *ValidationController) ValidationInterceptor(ctx *httpPackage.RequestContext) {",
				"// Validation interceptor logic",
				"if contentType := ctx.GetHeader(\"Content-Type\"); contentType == \"\" {",
				"ctx.JSON(400, map[string]string{\"error\": \"Content-Type header is required\"})",
				"ctx.Abort()",
				"return",
				"if !isValidRequestStructure(requestBody) {",
				"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body structure\"})",
			},
			wantErr: false,
		},
		{
			name: "Custom interceptor",
			input: `package test

type CustomController struct {}

@UseInterceptors(MyCustomInterceptor)
@Get("/custom")
func getCustom(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/custom/custom\", c.MyCustomInterceptor, c.getCustom)",
				"func (c *CustomController) MyCustomInterceptor(ctx *httpPackage.RequestContext) {",
				"// MyCustomInterceptor interceptor logic",
				"// TODO: Implement your custom interceptor logic here",
				"// Example: Pre-processing",
				"// requestData := ctx.GetRequestData()",
				"// if !validateRequest(requestData) {",
			},
			wantErr: false,
		},
		{
			name: "Mixed interceptors and guards",
			input: `package test

@Controller("/api")
@UseInterceptors(LoggingInterceptor)
@UseGuards(AuthGuard)
type MixedController struct {}

@UseInterceptors(CacheInterceptor)
@UseGuards(RoleGuard)
@Get("/protected")
func getProtected(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/api/protected\", c.LoggingInterceptor, c.CacheInterceptor, c.AuthGuard, c.RoleGuard, c.getProtected)",
				"func (c *MixedController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) CacheInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) AuthGuard(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) RoleGuard(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Multiple interceptors with different methods",
			input: `package test

@Controller("/api")
type MultiController struct {}

@UseInterceptors(LoggingInterceptor)
@Get("/logs")
func getLogs(ctx *RequestContext) {
}

@UseInterceptors(CacheInterceptor, ValidationInterceptor)
@Post("/cached")
func createCached(ctx *RequestContext) {
}

@UseInterceptors(TransformInterceptor, LoggingInterceptor, CacheInterceptor)
@Put("/complex")
func updateComplex(ctx *RequestContext) {
}

@Get("/simple")
func getSimple(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/api/logs\", c.LoggingInterceptor, c.getLogs)",
				"server.Post(\"/api/cached\", c.CacheInterceptor, c.ValidationInterceptor, c.createCached)",
				"server.Put(\"/api/complex\", c.TransformInterceptor, c.LoggingInterceptor, c.CacheInterceptor, c.updateComplex)",
				"server.Get(\"/api/simple\", c.getSimple)", // No interceptors
				"func (c *MultiController) LoggingInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) CacheInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) ValidationInterceptor(ctx *httpPackage.RequestContext) {",
				"func (c *MultiController) TransformInterceptor(ctx *httpPackage.RequestContext) {",
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
			if strings.Contains(tt.input, "@UseInterceptors") {
				// Ensure interceptor middleware methods are generated
				if !strings.Contains(output, "// Interceptor middleware methods") {
					t.Errorf("Interceptor middleware methods section not found")
				}
			}

			// Ensure no syntax errors in generated Go code (basic check)
			if strings.Count(output, "{") != strings.Count(output, "}") {
				t.Errorf("Unbalanced braces in generated code")
			}
		})
	}
}

func TestUseInterceptorsDecoratorParsing(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expectedInterceptors []string
		wantErr              bool
	}{
		{
			name: "Single interceptor",
			input: `package test

type TestController struct {}

@UseInterceptors(LoggingInterceptor)
@Get("/test")
func (c *TestController) getTest() {}`,
			expectedInterceptors: []string{"LoggingInterceptor"},
			wantErr:              false,
		},
		{
			name: "Multiple interceptors",
			input: `package test

type TestController struct {}

@UseInterceptors(LoggingInterceptor, CacheInterceptor, ValidationInterceptor)
@Get("/test")
func (c *TestController) getTest() {}`,
			expectedInterceptors: []string{"LoggingInterceptor", "CacheInterceptor", "ValidationInterceptor"},
			wantErr:              false,
		},
		{
			name: "No interceptors",
			input: `package test

type TestController struct {}

@Get("/test")
func (c *TestController) getTest() {}`,
			expectedInterceptors: []string{},
			wantErr:              false,
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

			// Use code generator to extract interceptors
			generator := NewCodeGenerator("test")
			actualInterceptors := generator.getInterceptorDecorators(method.Decorators)

			if len(actualInterceptors) != len(tt.expectedInterceptors) {
				t.Fatalf("Expected %d interceptors, got %d", len(tt.expectedInterceptors), len(actualInterceptors))
			}

			for i, expected := range tt.expectedInterceptors {
				if actualInterceptors[i] != expected {
					t.Errorf("Expected interceptor %s at position %d, got %s", expected, i, actualInterceptors[i])
				}
			}
		})
	}
}

func TestInterceptorMiddlewareGeneration(t *testing.T) {
	input := `package test

@Controller("/api")
@UseInterceptors(LoggingInterceptor)
type ApiController struct {}

@UseInterceptors(CacheInterceptor)
@Get("/cache")
func getCache(ctx *RequestContext) {
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

	// Test specific interceptor middleware patterns
	tests := []struct {
		pattern     string
		shouldExist bool
	}{
		{"func (c *ApiController) LoggingInterceptor(ctx *httpPackage.RequestContext)", true},
		{"func (c *ApiController) CacheInterceptor(ctx *httpPackage.RequestContext)", true},
		{"// Logging interceptor logic", true},
		{"// Cache interceptor logic", true},
		{"startTime := time.Now()", true},
		{"cacheKey := fmt.Sprintf(\"%s:%s\", ctx.GetMethod(), ctx.GetPath())", true},
		{"ctx.OnFinished(func() {", true},
		{"ctx.Next()", true},
		{"server.Get(\"/api/cache\", c.LoggingInterceptor, c.CacheInterceptor, c.getCache)", true},
		{"server.Get(\"/api/public\", c.LoggingInterceptor, c.getPublic)", true},
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

func TestInterceptorAndGuardCombination(t *testing.T) {
	input := `package test

@Controller("/secure")
@UseInterceptors(LoggingInterceptor)
@UseGuards(AuthGuard)
type SecureController struct {}

@UseInterceptors(CacheInterceptor, ValidationInterceptor)
@UseGuards(RoleGuard, PermissionGuard)
@Get("/data")
func getData(ctx *RequestContext) {
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

	// Test that interceptors come before guards in the middleware chain
	expectedRoute := "server.Get(\"/secure/data\", c.LoggingInterceptor, c.CacheInterceptor, c.ValidationInterceptor, c.AuthGuard, c.RoleGuard, c.PermissionGuard, c.getData)"
	if !strings.Contains(output, expectedRoute) {
		t.Errorf("Expected middleware order not found. Expected: %s", expectedRoute)
		t.Logf("Generated code:\n%s", output)
	}

	// Test that all middleware functions are generated
	middlewareFunctions := []string{
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

func TestInterceptorErrorHandling(t *testing.T) {
	input := `package test

type TestController struct {}

@UseInterceptors(ValidationInterceptor, CacheInterceptor, TransformInterceptor, LoggingInterceptor)
@Post("/test")
func (c *TestController) postTest() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("ParseGofaFile() error = %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("GenerateGoCode() error = %v", err)
	}

	// Test that each interceptor type generates appropriate logic
	interceptorTests := []struct {
		interceptor string
		patterns    []string
	}{
		{"ValidationInterceptor", []string{
			"if contentType := ctx.GetHeader(\"Content-Type\"); contentType == \"\" {",
			"ctx.JSON(400, map[string]string{\"error\": \"Content-Type header is required\"})",
			"ctx.Abort()",
			"return",
		}},
		{"CacheInterceptor", []string{
			"cacheKey := fmt.Sprintf(\"%s:%s\", ctx.GetMethod(), ctx.GetPath())",
			"if cachedResponse := getFromCache(cacheKey); cachedResponse != nil {",
			"ctx.JSON(200, cachedResponse)",
			"return // Skip further processing",
		}},
		{"TransformInterceptor", []string{
			"transformedBody := transformRequestData(requestBody)",
			"ctx.SetRequestBody(transformedBody)",
			"transformedResponse := transformResponseData(responseData)",
			"ctx.SetResponseData(transformedResponse)",
		}},
		{"LoggingInterceptor", []string{
			"startTime := time.Now()",
			"fmt.Printf(\"[%s] %s %s from %s\\n\", startTime.Format(time.RFC3339), method, path, clientIP)",
			"duration := time.Since(startTime)",
			"status := ctx.GetStatusCode()",
		}},
	}

	for _, test := range interceptorTests {
		for _, pattern := range test.patterns {
			if !strings.Contains(output, pattern) {
				t.Errorf("Expected pattern for %s not found: %s", test.interceptor, pattern)
			}
		}
	}
}