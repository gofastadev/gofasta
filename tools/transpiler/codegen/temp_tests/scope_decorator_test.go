package codegen

import (
	"fmt"
	"strings"
	"testing"
)

func TestScopeDecoratorBasic(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "Basic Scope decorator with singleton",
			input: `package main

@Scope("singleton")
type UserService struct {
    @Inject("database")
    DB *Database
}

func GetUsers() []User {
    return []User{}
}`,
			expectedOutput: []string{
				"func NewUserService(container *core.DIContainer) (*UserService, error)",
				"func RegisterUserServiceProvider(container *core.DIContainer) error",
				"return container.RegisterSingleton(\"userservice\", NewUserService)",
			},
		},
		{
			name: "Basic Scope decorator with transient",
			input: `package main

@Scope("transient")
type RequestHandler struct {
    @Inject("session")
    Session *SessionStore
}`,
			expectedOutput: []string{
				"func NewRequestHandler(container *core.DIContainer) (*RequestHandler, error)",
				"func RegisterRequestHandlerProvider(container *core.DIContainer) error",
				"return container.RegisterTransient(\"requesthandler\", NewRequestHandler)",
			},
		},
		{
			name: "Basic Scope decorator with request scope",
			input: `package main

@Scope("request")
type RequestContextService struct {
    @Inject("userstore")
    UserStore *UserStore
}`,
			expectedOutput: []string{
				"func NewRequestContextService(container *core.DIContainer) (*RequestContextService, error)",
				"func RegisterRequestContextServiceProvider(container *core.DIContainer) error",
				"return container.RegisterScoped(\"requestcontextservice\", NewRequestContextService)",
			},
		},
		{
			name: "Service without any decorator",
			input: `package main

type PlainService struct {
    @Inject("database")
    DB *Database
}`,
			notExpected: []string{
				"func NewPlainService",
				"func RegisterPlainServiceProvider",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the .gofa file
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse .gofa file: %v", err)
			}

			// Generate Go code
			generator := NewCodeGenerator("main")
			output, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain: %s\nActual output:\n%s", expected, output)
				}
			}

			// Check not expected output
			for _, notExpected := range tt.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain: %s\nActual output:\n%s", notExpected, output)
				}
			}
		})
	}
}

func TestScopeDecoratorCombinations(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string
	}{
		{
			name: "Injectable and Scope combined - Scope takes precedence",
			input: `package main

@Injectable("singleton")
@Scope("transient")
type HybridService struct {
    @Inject("data")
    Data *DataStore
}`,
			expectedOutput: []string{
				"func NewHybridService(container *core.DIContainer) (*HybridService, error)",
				"return container.RegisterTransient(\"hybridservice\", NewHybridService)",
			},
		},
		{
			name: "Injectable with scope parameter and Scope decorator",
			input: `package main

@Injectable({scope: "singleton"})
@Scope("request")
type ConflictService struct {
    @Inject("config")
    Config *Config
}`,
			expectedOutput: []string{
				"func NewConflictService(container *core.DIContainer) (*ConflictService, error)",
				"return container.RegisterScoped(\"conflictservice\", NewConflictService)",
			},
		},
		{
			name: "Only Scope decorator without Injectable",
			input: `package main

@Scope("singleton")
type ScopeOnlyService struct {
    @Inject("logger")
    Logger *Logger
}`,
			expectedOutput: []string{
				"func NewScopeOnlyService(container *core.DIContainer) (*ScopeOnlyService, error)",
				"return container.RegisterSingleton(\"scopeonlyservice\", NewScopeOnlyService)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and generate
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse .gofa file: %v", err)
			}

			generator := NewCodeGenerator("main")
			output, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain: %s\nActual output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestScopeDecoratorNormalization(t *testing.T) {
	tests := []struct {
		name          string
		scope         string
		expectedScope string
	}{
		{
			name:          "Singleton variations",
			scope:         "singleton",
			expectedScope: "RegisterSingleton",
		},
		{
			name:          "Single alias",
			scope:         "single",
			expectedScope: "RegisterSingleton",
		},
		{
			name:          "Transient variations",
			scope:         "transient",
			expectedScope: "RegisterTransient",
		},
		{
			name:          "Prototype alias",
			scope:         "prototype",
			expectedScope: "RegisterTransient",
		},
		{
			name:          "Instance alias",
			scope:         "instance",
			expectedScope: "RegisterTransient",
		},
		{
			name:          "Request variations",
			scope:         "request",
			expectedScope: "RegisterScoped",
		},
		{
			name:          "Scoped alias",
			scope:         "scoped",
			expectedScope: "RegisterScoped",
		},
		{
			name:          "Req alias",
			scope:         "req",
			expectedScope: "RegisterScoped",
		},
		{
			name:          "Custom scope",
			scope:         "custom",
			expectedScope: "RegisterSingleton", // Falls back to singleton for unknown scopes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf(`package main

@Scope("%s")
type TestService struct {
    @Inject("test")
    Test *Test
}`, tt.scope)

			// Parse and generate
			file, err := ParseGofaFile(input)
			if err != nil {
				t.Fatalf("Failed to parse .gofa file: %v", err)
			}

			generator := NewCodeGenerator("main")
			output, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check for expected scope registration method
			if !strings.Contains(output, tt.expectedScope) {
				t.Errorf("Expected output to contain: %s\nActual output:\n%s", tt.expectedScope, output)
			}
		})
	}
}

func TestScopeDecoratorObjectSyntax(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string
	}{
		{
			name: "Scope with object syntax",
			input: `package main

@Scope({scope: "transient"})
type ObjectScopeService struct {
    @Inject("data")
    Data *DataStore
}`,
			expectedOutput: []string{
				"func NewObjectScopeService(container *core.DIContainer) (*ObjectScopeService, error)",
				"return container.RegisterTransient(\"objectscopeservice\", NewObjectScopeService)",
			},
		},
		{
			name: "Scope with object and additional properties",
			input: `package main

@Scope({scope: "singleton", metadata: "test"})
type ComplexScopeService struct {
    @Inject("config")
    Config *Config
}`,
			expectedOutput: []string{
				"func NewComplexScopeService(container *core.DIContainer) (*ComplexScopeService, error)",
				"return container.RegisterSingleton(\"complexscopeservice\", NewComplexScopeService)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and generate
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse .gofa file: %v", err)
			}

			generator := NewCodeGenerator("main")
			output, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain: %s\nActual output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestScopeDecoratorWithComplexDependencies(t *testing.T) {
	input := `package main

@Scope("transient")
type ComplexService struct {
    @Inject("database")
    DB *Database
    
    @Inject({token: "cache", optional: true})
    Cache *CacheService
    
    @Inject({token: "logger"})
    Logger *Logger
}

func ProcessData() error {
    return nil
}`

	// Parse and generate
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	expectedPatterns := []string{
		"func NewComplexService(container *core.DIContainer) (*ComplexService, error)",
		"// Inject DB",
		"dep, err := container.Get(\"database\")",
		"instance.DB = typedDep",
		"// Inject Cache",
		"if dep, exists := container.GetOptional(\"cache\"); exists {",
		"instance.Cache = typedDep",
		"// Inject Logger",
		"dep, err := container.Get(\"logger\")",
		"instance.Logger = typedDep",
		"return container.RegisterTransient(\"complexservice\", NewComplexService)",
	}

	for _, expected := range expectedPatterns {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain: %s\nActual output:\n%s", expected, output)
		}
	}
}

func TestScopeDecoratorModuleIntegration(t *testing.T) {
	input := `package main

@Scope("singleton")
type DatabaseService struct {
    @Inject("config")
    Config *Config
}

@Scope("transient")
type RequestService struct {
    @Inject("session")
    Session *SessionStore
}

@Injectable("request")
type UserContextService struct {
    @Inject("userstore")
    UserStore *UserStore
}

@Module({
    providers: ["DatabaseService", "RequestService", "UserContextService"]
})
type AppModule struct {}
`

	// Parse and generate
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	expectedInModule := []string{
		"func (m *AppModule) Configure(container *core.DIContainer) error {",
		"// Register providers",
		"if err := RegisterDatabaseServiceProvider(container); err != nil {",
		"if err := RegisterRequestServiceProvider(container); err != nil {",
		"if err := RegisterUserContextServiceProvider(container); err != nil {",
	}

	expectedProviders := []string{
		"return container.RegisterSingleton(\"databaseservice\", NewDatabaseService)",
		"return container.RegisterTransient(\"requestservice\", NewRequestService)",
		"return container.RegisterScoped(\"usercontextservice\", NewUserContextService)",
	}

	// Check module integration
	for _, expected := range expectedInModule {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected module to contain: %s\nActual output:\n%s", expected, output)
		}
	}

	// Check provider generation with correct scopes
	for _, expected := range expectedProviders {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected providers to contain: %s\nActual output:\n%s", expected, output)
		}
	}
}

func TestScopeDecoratorASTParsing(t *testing.T) {
	input := `package main

@Scope("singleton")
type TestService struct {
    Field string
}`

	// Parse the .gofa file
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file: %v", err)
	}

	// Verify AST structure
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}

	service, ok := file.Declarations[0].(*ServiceDeclaration)
	if !ok {
		t.Fatalf("Expected ServiceDeclaration, got %T", file.Declarations[0])
	}

	if service.Name != "TestService" {
		t.Errorf("Expected service name 'TestService', got '%s'", service.Name)
	}

	// Check for Scope decorator
	hasScopeDecorator := false
	for _, decorator := range service.Decorators {
		if decorator.Name == "Scope" {
			hasScopeDecorator = true
			if len(decorator.Args) != 1 {
				t.Errorf("Expected 1 argument for Scope decorator, got %d", len(decorator.Args))
			} else {
				if arg, ok := decorator.Args[0].Value.(string); !ok || arg != "singleton" {
					t.Errorf("Expected Scope argument 'singleton', got %v", decorator.Args[0].Value)
				}
			}
			break
		}
	}

	if !hasScopeDecorator {
		t.Error("Expected to find Scope decorator in AST")
	}
}

func TestScopeDecoratorErrorHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "Empty scope decorator",
			input: `package main

@Scope()
type EmptyService struct {
    Field string
}`,
		},
		{
			name: "Invalid scope argument",
			input: `package main

@Scope(123)
type InvalidService struct {
    Field string
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should parse without errors even with invalid arguments
			// The transpiler should handle gracefully
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse .gofa file: %v", err)
			}

			generator := NewCodeGenerator("main")
			output, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Should default to singleton scope
			if !strings.Contains(output, "RegisterSingleton") {
				t.Error("Expected to default to singleton scope for invalid/empty scope")
			}
		})
	}
}