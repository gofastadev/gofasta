package codegen

import (
	"strings"
	"testing"
)

func TestUseGuardsDecorator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected patterns in generated code
		wantErr  bool
	}{
		{
			name: "Single guard on method",
			input: `package test

type UserController struct {}

@UseGuards(AuthGuard)
@Get("/profile")
func getProfile(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/user/profile\", c.AuthGuard, c.getProfile)",
				"func (c *UserController) AuthGuard(ctx *httpPackage.RequestContext) {",
				"// Authentication guard logic",
				"token := ctx.GetHeader(\"Authorization\")",
				"if token == \"\" {",
				"ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: Missing authentication token\"})",
				"ctx.Abort()",
				"return",
				"}",
				"// Validate token (implement your token validation logic)",
				"if !isValidToken(token) {",
				"ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: Invalid token\"})",
				"ctx.Abort()",
				"return",
				"}",
				"// If guard passes, continue to next middleware/handler",
				"ctx.Next()",
			},
			wantErr: false,
		},
		{
			name: "Multiple guards on method",
			input: `package test

type AdminController struct {}

@UseGuards(AuthGuard, RoleGuard)
@Get("/admin")
func getAdmin(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/admin/admin\", c.AuthGuard, c.RoleGuard, c.getAdmin)",
				"func (c *AdminController) AuthGuard(ctx *httpPackage.RequestContext) {",
				"func (c *AdminController) RoleGuard(ctx *httpPackage.RequestContext) {",
				"// Role-based authorization guard logic",
				"user := ctx.GetUser()",
				"if user == nil {",
				"ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: User not authenticated\"})",
			},
			wantErr: false,
		},
		{
			name: "Controller-level guard",
			input: `package test

@Controller("/api")
@UseGuards(AuthGuard)
type ApiController struct {}

@Get("/data")
func getData(ctx *RequestContext) {
}

@Post("/data")
func createData(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/api/data\", c.AuthGuard, c.getData)",
				"server.Post(\"/api/data\", c.AuthGuard, c.createData)",
				"func (c *ApiController) AuthGuard(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Controller and method level guards combined",
			input: `package test

@Controller("/admin")
@UseGuards(AuthGuard)
type AdminController struct {}

@UseGuards(RoleGuard)
@Get("/users")
func getUsers(ctx *RequestContext) {
}

@Get("/public")
func getPublic(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/admin/users\", c.AuthGuard, c.RoleGuard, c.getUsers)",
				"server.Get(\"/admin/public\", c.AuthGuard, c.getPublic)",
				"func (c *AdminController) AuthGuard(ctx *httpPackage.RequestContext) {",
				"func (c *AdminController) RoleGuard(ctx *httpPackage.RequestContext) {",
			},
			wantErr: false,
		},
		{
			name: "Custom guard",
			input: `package test

type SecureController struct {}

@UseGuards(CustomSecurityGuard)
@Get("/secure")
func getSecure(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/secure/secure\", c.CustomSecurityGuard, c.getSecure)",
				"func (c *SecureController) CustomSecurityGuard(ctx *httpPackage.RequestContext) {",
				"// CustomSecurityGuard guard logic",
				"// TODO: Implement your custom guard logic here",
				"if !checkGuardCondition() {",
				"ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Guard check failed\"})",
			},
			wantErr: false,
		},
		{
			name: "Permission guard",
			input: `package test

type PermissionController struct {}

@UseGuards(PermissionGuard)
@Get("/protected")
func getProtected(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/permission/protected\", c.PermissionGuard, c.getProtected)",
				"func (c *PermissionController) PermissionGuard(ctx *httpPackage.RequestContext) {",
				"// Permission-based authorization guard logic",
				"user := ctx.GetUser()",
				"if user == nil {",
				"// Check user permissions (implement your permission checking logic)",
				"if !hasPermission(user, \"required_permission\") {",
				"ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Insufficient permissions\"})",
			},
			wantErr: false,
		},
		{
			name: "Mixed guards on different methods",
			input: `package test

@Controller("/api")
type MixedController struct {}

@UseGuards(AuthGuard)
@Get("/auth-required")
func getAuthRequired(ctx *RequestContext) {
}

@UseGuards(AuthGuard, RoleGuard, PermissionGuard)
@Post("/admin-only")
func createAdminOnly(ctx *RequestContext) {
}

@Get("/public")
func getPublic(ctx *RequestContext) {
}`,
			expected: []string{
				"server.Get(\"/api/auth-required\", c.AuthGuard, c.getAuthRequired)",
				"server.Post(\"/api/admin-only\", c.AuthGuard, c.RoleGuard, c.PermissionGuard, c.createAdminOnly)",
				"server.Get(\"/api/public\", c.getPublic)", // No guards
				"func (c *MixedController) AuthGuard(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) RoleGuard(ctx *httpPackage.RequestContext) {",
				"func (c *MixedController) PermissionGuard(ctx *httpPackage.RequestContext) {",
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
			if strings.Contains(tt.input, "@UseGuards") {
				// Ensure guard middleware methods are generated
				if !strings.Contains(output, "// Guard middleware methods") {
					t.Errorf("Guard middleware methods section not found")
				}
			}

			// Ensure no syntax errors in generated Go code (basic check)
			if strings.Count(output, "{") != strings.Count(output, "}") {
				t.Errorf("Unbalanced braces in generated code")
			}
		})
	}
}

func TestUseGuardsDecoratorParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedGuards []string
		wantErr        bool
	}{
		{
			name: "Single guard",
			input: `package test

type TestController struct {}

@UseGuards(AuthGuard)
@Get("/test")
func (c *TestController) getTest() {}`,
			expectedGuards: []string{"AuthGuard"},
			wantErr:        false,
		},
		{
			name: "Multiple guards",
			input: `package test

type TestController struct {}

@UseGuards(AuthGuard, RoleGuard, PermissionGuard)
@Get("/test")
func (c *TestController) getTest() {}`,
			expectedGuards: []string{"AuthGuard", "RoleGuard", "PermissionGuard"},
			wantErr:        false,
		},
		{
			name: "No guards",
			input: `package test

type TestController struct {}

@Get("/test")
func (c *TestController) getTest() {}`,
			expectedGuards: []string{},
			wantErr:        false,
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

			// Use code generator to extract guards
			generator := NewCodeGenerator("test")
			actualGuards := generator.getGuardDecorators(method.Decorators)

			if len(actualGuards) != len(tt.expectedGuards) {
				t.Fatalf("Expected %d guards, got %d", len(tt.expectedGuards), len(actualGuards))
			}

			for i, expected := range tt.expectedGuards {
				if actualGuards[i] != expected {
					t.Errorf("Expected guard %s at position %d, got %s", expected, i, actualGuards[i])
				}
			}
		})
	}
}

func TestGuardMiddlewareGeneration(t *testing.T) {
	input := `package test

@Controller("/api")
@UseGuards(AuthGuard)
type ApiController struct {}

@UseGuards(RoleGuard)
@Get("/admin")
func getAdmin(ctx *RequestContext) {
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

	// Test specific guard middleware patterns
	tests := []struct {
		pattern     string
		shouldExist bool
	}{
		{"func (c *ApiController) AuthGuard(ctx *httpPackage.RequestContext)", true},
		{"func (c *ApiController) RoleGuard(ctx *httpPackage.RequestContext)", true},
		{"// Authentication guard logic", true},
		{"// Role-based authorization guard logic", true},
		{"token := ctx.GetHeader(\"Authorization\")", true},
		{"user := ctx.GetUser()", true},
		{"ctx.Abort()", true},
		{"ctx.Next()", true},
		{"server.Get(\"/api/admin\", c.AuthGuard, c.RoleGuard, c.getAdmin)", true},
		{"server.Get(\"/api/public\", c.AuthGuard, c.getPublic)", true},
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

func TestGuardErrorHandling(t *testing.T) {
	input := `package test

type TestController struct {}

@UseGuards(AuthGuard, RoleGuard, PermissionGuard, CustomGuard)
@Get("/test")
func (c *TestController) getTest() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("ParseGofaFile() error = %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("GenerateGoCode() error = %v", err)
	}

	// Test that each guard type generates appropriate error responses
	errorTests := []struct {
		guard         string
		errorResponse string
	}{
		{"AuthGuard", "ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: Missing authentication token\"})"},
		{"AuthGuard", "ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: Invalid token\"})"},
		{"RoleGuard", "ctx.JSON(401, map[string]string{\"error\": \"Unauthorized: User not authenticated\"})"},
		{"RoleGuard", "ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Insufficient permissions\"})"},
		{"PermissionGuard", "ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Insufficient permissions\"})"},
		{"CustomGuard", "ctx.JSON(403, map[string]string{\"error\": \"Forbidden: Guard check failed\"})"},
	}

	for _, test := range errorTests {
		if !strings.Contains(output, test.errorResponse) {
			t.Errorf("Expected error response for %s not found: %s", test.guard, test.errorResponse)
		}
	}
}