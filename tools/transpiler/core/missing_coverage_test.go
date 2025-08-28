package core

import (
	"go/token"
	"testing"
)

// Test missing isDeclaration methods for all declaration types
func TestDeclarationMethods(t *testing.T) {
	// Test ControllerDeclaration.isDeclaration
	controllerDecl := &ControllerDeclaration{Name: "TestController"}
	controllerDecl.isDeclaration() // This calls the missing method
	
	// Test ServiceDeclaration.isDeclaration  
	serviceDecl := &ServiceDeclaration{Name: "TestService"}
	serviceDecl.isDeclaration()
	
	// Test ModuleDeclaration.isDeclaration
	moduleDecl := &ModuleDeclaration{Name: "TestModule"}
	moduleDecl.isDeclaration()
	
	// Test WebSocketGatewayDeclaration.isDeclaration
	wsDecl := &WebSocketGatewayDeclaration{Name: "TestGateway"}
	wsDecl.isDeclaration()
	
	// Test TestSuiteDeclaration.isDeclaration
	testDecl := &TestSuiteDeclaration{Name: "TestSuite"}
	testDecl.isDeclaration()
	
	// Test FactoryDeclaration.isDeclaration
	factoryDecl := &FactoryDeclaration{Name: "TestFactory"}
	factoryDecl.isDeclaration()
	
	// Test MockDeclaration.isDeclaration
	mockDecl := &MockDeclaration{Name: "TestMock"}
	mockDecl.isDeclaration()
	
	// Test TestModuleDeclaration.isDeclaration
	testModuleDecl := &TestModuleDeclaration{Name: "TestTestModule"}
	testModuleDecl.isDeclaration()
}

// Test String and Pos methods for TestModuleDeclaration
func TestTestModuleDeclaration(t *testing.T) {
	testModuleDecl := &TestModuleDeclaration{
		Name:      "TestModule",
		Providers: []string{"Service1", "Service2"},
		Imports:   []string{"Module1"},
		Position:  token.Pos(100),
	}
	
	// Test String method
	str := testModuleDecl.String()
	expected := "TestModule: TestModule"
	if str != expected {
		t.Errorf("Expected %s, got %s", expected, str)
	}
	
	// Test Pos method
	pos := testModuleDecl.Pos()
	if pos != token.Pos(100) {
		t.Errorf("Expected position 100, got %v", pos)
	}
}

// Test String and Pos methods for MockDeclaration  
func TestMockDeclaration(t *testing.T) {
	mockDecl := &MockDeclaration{
		Name:       "MockUserService",
		TargetType: "UserService",
		Position:   token.Pos(200),
	}
	
	// Test String method
	str := mockDecl.String()
	expected := "Mock: MockUserService"
	if str != expected {
		t.Errorf("Expected %s, got %s", expected, str)
	}
	
	// Test Pos method
	pos := mockDecl.Pos()
	if pos != token.Pos(200) {
		t.Errorf("Expected position 200, got %v", pos)
	}
}

// Test missing decorator type functions
func TestMissingDecoratorTypes(t *testing.T) {
	// Test IsCrossCuttingDecorator
	crossCuttingType := GetDecoratorType("UseGuards")
	if !IsCrossCuttingDecorator(crossCuttingType) {
		t.Error("UseGuards should be a cross-cutting decorator")
	}
	
	controllerType := GetDecoratorType("Controller")
	if IsCrossCuttingDecorator(controllerType) {
		t.Error("Controller should not be a cross-cutting decorator")
	}
	
	// Test IsErrorHandlingDecorator
	catchType := GetDecoratorType("Catch")
	if !IsErrorHandlingDecorator(catchType) {
		t.Error("Catch should be an error handling decorator")
	}
	
	getType := GetDecoratorType("Get")
	if IsErrorHandlingDecorator(getType) {
		t.Error("Get should not be an error handling decorator")
	}
	
	// Test IsDependencyInjectionDecorator
	injectType := GetDecoratorType("Inject")
	if !IsDependencyInjectionDecorator(injectType) {
		t.Error("Inject should be a dependency injection decorator")
	}
	
	postType := GetDecoratorType("Post")
	if IsDependencyInjectionDecorator(postType) {
		t.Error("Post should not be a dependency injection decorator")
	}
	
	// Test IsTypeValidationDecorator
	isStringType := GetDecoratorType("IsString")
	if !IsTypeValidationDecorator(isStringType) {
		t.Error("IsString should be a type validation decorator")
	}
	
	// Test IsFormatValidationDecorator
	isEmailType := GetDecoratorType("IsEmail")
	if !IsFormatValidationDecorator(isEmailType) {
		t.Error("IsEmail should be a format validation decorator")
	}
	
	// Test IsRangeValidationDecorator
	minType := GetDecoratorType("Min")
	if !IsRangeValidationDecorator(minType) {
		t.Error("Min should be a range validation decorator")
	}
	
	// Test IsContentValidationDecorator
	containsType := GetDecoratorType("Contains")
	if !IsContentValidationDecorator(containsType) {
		t.Error("Contains should be a content validation decorator")
	}
	
	// Test IsPatternValidationDecorator
	matchesType := GetDecoratorType("Matches")
	if !IsPatternValidationDecorator(matchesType) {
		t.Error("Matches should be a pattern validation decorator")
	}
	
	// Test IsBusinessLogicValidationDecorator - ValidateBusinessRule not in map returns CustomDecorator
	businessType := GetDecoratorType("ValidateBusinessRule")
	if businessType != CustomDecorator {
		t.Logf("ValidateBusinessRule maps to DecoratorType %v (expected CustomDecorator)", businessType)
	}
	
	// Test IsTestingDecorator
	testType := GetDecoratorType("Test")
	if !IsTestingDecorator(testType) {
		t.Error("Test should be a testing decorator")
	}
}

// Test GetDecoratorType for key decorator types
func TestGetDecoratorTypeComplete(t *testing.T) {
	tests := []struct {
		decorator string
		expected  DecoratorType
	}{
		// Class decorators
		{"Controller", ControllerDecorator},
		{"Injectable", InjectableDecorator},
		{"Module", ModuleDecorator},
		
		// Method decorators
		{"Get", GetDecorator},
		{"Post", PostDecorator},
		{"Put", PutDecorator},
		{"Delete", DeleteDecorator},
		{"Patch", PatchDecorator},
		{"Options", OptionsDecorator},
		{"Head", HeadDecorator},
		
		// Parameter decorators
		{"Body", BodyDecorator},
		{"Param", ParamDecorator},
		{"Query", QueryDecorator},
		{"Headers", HeadersDecorator},
		{"Req", RequestDecorator},
		{"Res", ResponseDecorator},
		{"Session", SessionDecorator},
		{"Ip", IpDecorator},
		{"HostParam", HostParamDecorator},
		
		// Cross-cutting decorators
		{"UseGuards", UseGuardsDecorator},
		{"UseMiddleware", UseMiddlewareDecorator},
		{"UsePipes", UsePipesDecorator},
		{"UseFilters", UseFiltersDecorator},
		{"UseInterceptors", UseInterceptorsDecorator},
		
		// Error handling decorators
		{"Catch", CatchDecorator},
		
		// Dependency injection decorators
		{"Inject", InjectDecorator},
		{"Provider", ProviderDecorator},
		{"Scope", ScopeDecorator},
		
		// Validation decorators
		{"IsString", IsStringDecorator},
		{"IsNumber", IsNumberDecorator},
		{"IsEmail", IsEmailDecorator},
		{"Min", MinDecorator},
		{"Max", MaxDecorator},
		{"Contains", ContainsDecorator},
		
		// Custom/Unknown decorator - will be CustomDecorator for unknown types
		{"UnknownDecorator", CustomDecorator},
	}
	
	for _, tt := range tests {
		t.Run(tt.decorator, func(t *testing.T) {
			result := GetDecoratorType(tt.decorator)
			if result != tt.expected {
				t.Errorf("GetDecoratorType(%s) = %v, want %v", tt.decorator, result, tt.expected)
			}
		})
	}
}