package core

import (
	"reflect"
	"testing"
)

// Test types for decorator testing
type TestDecoratedService struct {
	Name     string `service:"decorated-service" scope:"singleton"`
	Database string `inject:"database"`
	Logger   string `inject:"logger"`
}

type TestDecoratedController struct {
	Path       string `controller:"/api/test"`
	Middleware string `middleware:"auth,cors"`
	Service    string `inject:"test-service"`
}

type TestDecoratedModule struct {
	Name        string `module:"test-decorated-module"`
	Controllers string `controllers:"TestController,UserController"`
	Providers   string `providers:"TestService,UserService"`
	Imports     string `imports:"DatabaseModule,AuthModule"`
	Exports     string `exports:"TestService"`
}

type TestValidationStruct struct {
	Email    string `json:"email" validate:"required,email"`
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Age      int    `json:"age" validate:"required,min=18,max=120"`
	Password string `json:"password" validate:"required,min=8"`
	Website  string `json:"website" validate:"url"`
}

func TestExtractDecorators(t *testing.T) {
	service := &TestDecoratedService{}
	
	decorators, err := ExtractDecorators(service)
	if err != nil {
		t.Fatalf("ExtractDecorators() failed: %v", err)
	}
	
	if decorators == nil {
		t.Fatal("ExtractDecorators() returned nil")
	}
	
	// Check for inject decorators
	injectDecorators, exists := decorators["inject"]
	if !exists {
		t.Error("No inject decorators found")
	} else {
		if len(injectDecorators) != 2 {
			t.Errorf("Expected 2 inject decorators, got %d", len(injectDecorators))
		}
		
		// Check first inject decorator
		if injectDecorators[0].Type != "inject" {
			t.Errorf("Expected decorator type 'inject', got %s", injectDecorators[0].Type)
		}
		
		if injectDecorators[0].Parameters["name"] != "database" {
			t.Errorf("Expected inject name 'database', got %v", injectDecorators[0].Parameters["name"])
		}
		
		if injectDecorators[0].Parameters["field"] != "Database" {
			t.Errorf("Expected field name 'Database', got %v", injectDecorators[0].Parameters["field"])
		}
	}
}

func TestExtractDecorators_WithValidation(t *testing.T) {
	validationStruct := &TestValidationStruct{}
	
	decorators, err := ExtractDecorators(validationStruct)
	if err != nil {
		t.Fatalf("ExtractDecorators() failed: %v", err)
	}
	
	// Check for validation decorators
	validateDecorators, exists := decorators["validate"]
	if !exists {
		t.Error("No validate decorators found")
	} else {
		// Should have multiple validation decorators (one for each validation rule)
		if len(validateDecorators) == 0 {
			t.Error("Expected validation decorators, got none")
		}
		
		// Check that validation rules are parsed correctly
		found := false
		for _, decorator := range validateDecorators {
			if decorator.Parameters["field"] == "Email" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Email field validation decorator not found")
		}
	}
	
	// Check for JSON decorators
	jsonDecorators, exists := decorators["json"]
	if !exists {
		t.Error("No json decorators found")
	} else {
		if len(jsonDecorators) != 5 {
			t.Errorf("Expected 5 json decorators, got %d", len(jsonDecorators))
		}
	}
}

func TestExtractDecorators_WithController(t *testing.T) {
	controller := &TestDecoratedController{}
	
	decorators, err := ExtractDecorators(controller)
	if err != nil {
		t.Fatalf("ExtractDecorators() failed: %v", err)
	}
	
	// Check for controller decorators
	controllerDecorators, exists := decorators["controller"]
	if !exists {
		t.Error("No controller decorators found")
	} else {
		if len(controllerDecorators) != 1 {
			t.Errorf("Expected 1 controller decorator, got %d", len(controllerDecorators))
		}
		
		if controllerDecorators[0].Parameters["path"] != "/api/test" {
			t.Errorf("Expected controller path '/api/test', got %v", controllerDecorators[0].Parameters["path"])
		}
	}
	
	// Check for inject decorators
	injectDecorators, exists := decorators["inject"]
	if !exists {
		t.Error("No inject decorators found")
	} else {
		if len(injectDecorators) != 1 {
			t.Errorf("Expected 1 inject decorator, got %d", len(injectDecorators))
		}
	}
}

func TestExtractDecorators_WithModule(t *testing.T) {
	module := &TestDecoratedModule{}
	
	decorators, err := ExtractDecorators(module)
	if err != nil {
		t.Fatalf("ExtractDecorators() failed: %v", err)
	}
	
	// Check for module decorators
	moduleDecorators, exists := decorators["module"]
	if !exists {
		t.Error("No module decorators found")
	} else {
		if len(moduleDecorators) != 1 {
			t.Errorf("Expected 1 module decorator, got %d", len(moduleDecorators))
		}
		
		if moduleDecorators[0].Parameters["name"] != "test-decorated-module" {
			t.Errorf("Expected module name 'test-decorated-module', got %v", moduleDecorators[0].Parameters["name"])
		}
	}
}

func TestParseValidationTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected []string
	}{
		{"required", []string{"required"}},
		{"required,email", []string{"required", "email"}},
		{"required,min=2,max=50", []string{"required", "min=2", "max=50"}},
		{"", nil},
		{"  required  ,  email  ", []string{"required", "email"}},
	}
	
	for _, test := range tests {
		result := parseValidationTag(test.tag)
		
		if len(result) != len(test.expected) {
			t.Errorf("For tag '%s', expected %d rules, got %d", test.tag, len(test.expected), len(result))
			continue
		}
		
		for i, expected := range test.expected {
			if result[i] != expected {
				t.Errorf("For tag '%s', expected rule '%s' at index %d, got '%s'", test.tag, expected, i, result[i])
			}
		}
	}
}

func TestExtractControllerMetadata(t *testing.T) {
	controller := &TestDecoratedController{}
	
	metadata, err := ExtractControllerMetadata(controller)
	if err != nil {
		t.Fatalf("ExtractControllerMetadata() failed: %v", err)
	}
	
	if metadata == nil {
		t.Fatal("ExtractControllerMetadata() returned nil")
	}
	
	if metadata.Path != "/api/test" {
		t.Errorf("Expected controller path '/api/test', got %s", metadata.Path)
	}
	
	if len(metadata.Middleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(metadata.Middleware))
	} else {
		if metadata.Middleware[0] != "auth" || metadata.Middleware[1] != "cors" {
			t.Errorf("Expected middleware ['auth', 'cors'], got %v", metadata.Middleware)
		}
	}
	
	if metadata.Routes == nil {
		t.Error("Routes slice should be initialized")
	}
}

func TestExtractModuleMetadata(t *testing.T) {
	module := &TestDecoratedModule{}
	
	metadata, err := ExtractModuleMetadata(module)
	if err != nil {
		t.Fatalf("ExtractModuleMetadata() failed: %v", err)
	}
	
	if metadata == nil {
		t.Fatal("ExtractModuleMetadata() returned nil")
	}
	
	if metadata.Name != "test-decorated-module" {
		t.Errorf("Expected module name 'test-decorated-module', got %s", metadata.Name)
	}
	
	if len(metadata.Controllers) != 2 {
		t.Errorf("Expected 2 controllers, got %d", len(metadata.Controllers))
	} else {
		if metadata.Controllers[0] != "TestController" || metadata.Controllers[1] != "UserController" {
			t.Errorf("Expected controllers ['TestController', 'UserController'], got %v", metadata.Controllers)
		}
	}
	
	if len(metadata.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(metadata.Providers))
	} else {
		if metadata.Providers[0] != "TestService" || metadata.Providers[1] != "UserService" {
			t.Errorf("Expected providers ['TestService', 'UserService'], got %v", metadata.Providers)
		}
	}
	
	if len(metadata.Imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(metadata.Imports))
	} else {
		if metadata.Imports[0] != "DatabaseModule" || metadata.Imports[1] != "AuthModule" {
			t.Errorf("Expected imports ['DatabaseModule', 'AuthModule'], got %v", metadata.Imports)
		}
	}
	
	if len(metadata.Exports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(metadata.Exports))
	} else {
		if metadata.Exports[0] != "TestService" {
			t.Errorf("Expected export 'TestService', got %s", metadata.Exports[0])
		}
	}
}

func TestExtractServiceProviderMetadata(t *testing.T) {
	service := &TestDecoratedService{}
	
	metadata, err := ExtractServiceProviderMetadata(service)
	if err != nil {
		t.Fatalf("ExtractServiceProviderMetadata() failed: %v", err)
	}
	
	if metadata == nil {
		t.Fatal("ExtractServiceProviderMetadata() returned nil")
	}
	
	if metadata.Name != "decorated-service" {
		t.Errorf("Expected service name 'decorated-service', got %s", metadata.Name)
	}
	
	if metadata.Scope != ScopeSingleton {
		t.Errorf("Expected scope ScopeSingleton, got %v", metadata.Scope)
	}
	
	expectedType := reflect.TypeOf((*TestDecoratedService)(nil)).Elem()
	if metadata.Type != expectedType {
		t.Errorf("Expected type %v, got %v", expectedType, metadata.Type)
	}
	
	if len(metadata.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(metadata.Dependencies))
	} else {
		if metadata.Dependencies[0] != "Database" || metadata.Dependencies[1] != "Logger" {
			t.Errorf("Expected dependencies ['Database', 'Logger'], got %v", metadata.Dependencies)
		}
	}
}

func TestExtractServiceProviderMetadata_WithDifferentScopes(t *testing.T) {
	type TransientService struct {
		Name string `service:"transient-service" scope:"transient"`
	}
	
	type ScopedService struct {
		Name string `service:"scoped-service" scope:"scoped"`
	}
	
	// Test transient scope
	transientService := &TransientService{}
	metadata, err := ExtractServiceProviderMetadata(transientService)
	if err != nil {
		t.Fatalf("ExtractServiceProviderMetadata() failed for transient: %v", err)
	}
	
	if metadata.Scope != ScopeTransient {
		t.Errorf("Expected scope ScopeTransient, got %v", metadata.Scope)
	}
	
	// Test scoped scope
	scopedService := &ScopedService{}
	metadata, err = ExtractServiceProviderMetadata(scopedService)
	if err != nil {
		t.Fatalf("ExtractServiceProviderMetadata() failed for scoped: %v", err)
	}
	
	if metadata.Scope != ScopeScoped {
		t.Errorf("Expected scope ScopeScoped, got %v", metadata.Scope)
	}
}

func TestExtractRouteMetadata(t *testing.T) {
	controller := &TestController1{}
	
	// Test extracting route metadata for GetUsers method
	metadata, err := ExtractRouteMetadata(controller, "GetUsers")
	if err != nil {
		t.Fatalf("ExtractRouteMetadata() failed: %v", err)
	}
	
	if metadata == nil {
		t.Fatal("ExtractRouteMetadata() returned nil")
	}
	
	if metadata.Method != "GET" {
		t.Errorf("Expected method 'GET', got %s", metadata.Method)
	}
	
	if metadata.Path != "/users" {
		t.Errorf("Expected path '/users', got %s", metadata.Path)
	}
	
	if metadata.Handler != "GetUsers" {
		t.Errorf("Expected handler 'GetUsers', got %s", metadata.Handler)
	}
	
	// Test extracting route metadata for PostUser method
	metadata, err = ExtractRouteMetadata(controller, "PostUser")
	if err != nil {
		t.Fatalf("ExtractRouteMetadata() failed: %v", err)
	}
	
	if metadata.Method != "POST" {
		t.Errorf("Expected method 'POST', got %s", metadata.Method)
	}
	
	if metadata.Path != "/user" {
		t.Errorf("Expected path '/user', got %s", metadata.Path)
	}
}

func TestExtractRouteMetadata_NonExistentMethod(t *testing.T) {
	controller := &TestController1{}
	
	_, err := ExtractRouteMetadata(controller, "NonExistentMethod")
	if err == nil {
		t.Error("Expected error for non-existent method")
	}
}

func TestExtractAllRouteMetadata(t *testing.T) {
	controller := &TestController1{}
	
	routes, err := ExtractAllRouteMetadata(controller)
	if err != nil {
		t.Fatalf("ExtractAllRouteMetadata() failed: %v", err)
	}
	
	if routes == nil {
		t.Fatal("ExtractAllRouteMetadata() returned nil")
	}
	
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}
	
	// Check that both methods are extracted
	methodsFound := make(map[string]bool)
	for _, route := range routes {
		methodsFound[route.Handler] = true
	}
	
	if !methodsFound["GetUsers"] {
		t.Error("GetUsers method not found in routes")
	}
	
	if !methodsFound["PostUser"] {
		t.Error("PostUser method not found in routes")
	}
}

func TestExtractAllRouteMetadata_EmptyController(t *testing.T) {
	type EmptyController struct{}
	
	controller := &EmptyController{}
	routes, err := ExtractAllRouteMetadata(controller)
	if err != nil {
		t.Fatalf("ExtractAllRouteMetadata() failed: %v", err)
	}
	
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes for empty controller, got %d", len(routes))
	}
}

func TestDecoratorMetadata(t *testing.T) {
	metadata := &DecoratorMetadata{
		Type: "inject",
		Parameters: map[string]interface{}{
			"name":  "test-service",
			"field": "TestField",
		},
	}
	
	if metadata.Type != "inject" {
		t.Errorf("Expected type 'inject', got %s", metadata.Type)
	}
	
	if metadata.Parameters["name"] != "test-service" {
		t.Errorf("Expected parameter name 'test-service', got %v", metadata.Parameters["name"])
	}
	
	if metadata.Parameters["field"] != "TestField" {
		t.Errorf("Expected parameter field 'TestField', got %v", metadata.Parameters["field"])
	}
}

func TestRouteMetadata(t *testing.T) {
	metadata := &RouteMetadata{
		Method:     "GET",
		Path:       "/api/users",
		Handler:    "GetUsers",
		Guards:     []string{"auth", "admin"},
		Pipes:      []string{"validation", "transform"},
		Validation: "required",
	}
	
	if metadata.Method != "GET" {
		t.Errorf("Expected method 'GET', got %s", metadata.Method)
	}
	
	if metadata.Path != "/api/users" {
		t.Errorf("Expected path '/api/users', got %s", metadata.Path)
	}
	
	if metadata.Handler != "GetUsers" {
		t.Errorf("Expected handler 'GetUsers', got %s", metadata.Handler)
	}
	
	if len(metadata.Guards) != 2 {
		t.Errorf("Expected 2 guards, got %d", len(metadata.Guards))
	}
	
	if len(metadata.Pipes) != 2 {
		t.Errorf("Expected 2 pipes, got %d", len(metadata.Pipes))
	}
	
	if metadata.Validation != "required" {
		t.Errorf("Expected validation 'required', got %s", metadata.Validation)
	}
}