package codegen

import (
	"fmt"
	"strings"
	"testing"
)

func TestProviderFactoryGeneration(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "Basic Injectable Service",
			input: `package main

@Injectable()
type UserService struct {
    @Inject("database")
    DB *Database
    
    @Inject()
    Logger *Logger
}

func GetUsers() []User {
    return []User{}
}`,
			expectedOutput: []string{
				"func NewUserService(container *core.DIContainer) (*UserService, error)",
				"instance := &UserService{}",
				"// Inject DB",
				"dep, err := container.Get(\"database\")",
				"instance.DB = typedDep",
				"// Inject Logger", 
				"dep, err := container.Get(\"logger\")",
				"instance.Logger = typedDep",
				"return instance, nil",
				"func RegisterUserServiceProvider(container *core.DIContainer) error",
				"return container.RegisterSingleton(\"userservice\", NewUserService)",
			},
		},
		{
			name: "Injectable Service with Optional Dependencies",
			input: `package main

@Injectable()
type NotificationService struct {
    @Inject("emailProvider")
    EmailProvider *EmailProvider
    
    @Inject({token: "smsProvider", optional: true})
    SMSProvider *SMSProvider
    
    @Inject({token: "analytics", optional: true})
    Analytics *Analytics
}`,
			expectedOutput: []string{
				"func NewNotificationService(container *core.DIContainer) (*NotificationService, error)",
				"// Inject EmailProvider",
				"dep, err := container.Get(\"emailProvider\")",
				"// Inject SMSProvider",
				"if dep, exists := container.GetOptional(\"smsProvider\"); exists {",
				"if typedDep, ok := dep.(*SMSProvider); ok {",
				"instance.SMSProvider = typedDep",
				"// Inject Analytics",
				"if dep, exists := container.GetOptional(\"analytics\"); exists {",
				"if typedDep, ok := dep.(*Analytics); ok {",
				"instance.Analytics = typedDep",
			},
		},
		{
			name: "Injectable Service with Singleton Scope",
			input: `package main

@Injectable("singleton")
type DatabaseService struct {
    @Inject("config")
    Config *Config
}`,
			expectedOutput: []string{
				"func NewDatabaseService(container *core.DIContainer) (*DatabaseService, error)",
				"func RegisterDatabaseServiceProvider(container *core.DIContainer) error",
				"return container.RegisterSingleton(\"databaseservice\", NewDatabaseService)",
			},
		},
		{
			name: "Injectable Service with Transient Scope",
			input: `package main

@Injectable("transient")
type RequestService struct {
    @Inject("logger")
    Logger *Logger
}`,
			expectedOutput: []string{
				"func NewRequestService(container *core.DIContainer) (*RequestService, error)",
				"func RegisterRequestServiceProvider(container *core.DIContainer) error",
				"return container.RegisterTransient(\"requestservice\", NewRequestService)",
			},
		},
		{
			name: "Injectable Service with Request Scope",
			input: `package main

@Injectable("request")
type RequestContextService struct {
    @Inject("sessionStore")
    SessionStore *SessionStore
}`,
			expectedOutput: []string{
				"func NewRequestContextService(container *core.DIContainer) (*RequestContextService, error)",
				"func RegisterRequestContextServiceProvider(container *core.DIContainer) error",
				"return container.RegisterScoped(\"requestcontextservice\", NewRequestContextService)",
			},
		},
		{
			name: "Injectable Service with Complex Dependencies",
			input: `package main

@Injectable()
type UserService struct {
    @Inject("userRepository")
    UserRepository *UserRepository
    
    @Inject({token: "emailService"})
    EmailService *EmailService
    
    @Inject({token: "logger", optional: true})
    Logger *Logger
    
    @Inject({token: "cache", optional: false})
    Cache *Cache
    
    @Inject({token: "metrics", optional: true, scope: "singleton"})
    Metrics *Metrics
}`,
			expectedOutput: []string{
				"func NewUserService(container *core.DIContainer) (*UserService, error)",
				"// Inject UserRepository",
				"dep, err := container.Get(\"userRepository\")",
				"// Inject EmailService", 
				"dep, err := container.Get(\"emailService\")",
				"// Inject Logger",
				"if dep, exists := container.GetOptional(\"logger\"); exists {",
				"// Inject Cache",
				"dep, err := container.Get(\"cache\")",
				"// Inject Metrics",
				"if dep, exists := container.GetOptional(\"metrics\"); exists {",
				"if initializer, ok := interface{}(instance).(interface{ Initialize() error }); ok {",
				"if err := initializer.Initialize(); err != nil {",
				"return nil, err",
			},
		},
		{
			name: "Service without Injectable decorator",
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
		{
			name: "Injectable Service with Object Scope Configuration",
			input: `package main

@Injectable({scope: "transient"})
type TempService struct {
    @Inject("tempData")
    TempData *TempData
}`,
			expectedOutput: []string{
				"func NewTempService(container *core.DIContainer) (*TempService, error)",
				"func RegisterTempServiceProvider(container *core.DIContainer) error",
				"return container.RegisterTransient(\"tempservice\", NewTempService)",
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

func TestProviderFactoryErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string
	}{
		{
			name: "Injectable Service with Error Handling",
			input: `package main

@Injectable()
type ServiceWithErrors struct {
    @Inject("criticalDep")
    CriticalDep *CriticalDep
    
    @Inject({token: "optionalDep", optional: true})
    OptionalDep *OptionalDep
}`,
			expectedOutput: []string{
				"dep, err := container.Get(\"criticalDep\")",
				"if err != nil {",
				"return nil, fmt.Errorf(\"failed to inject CriticalDep: %w\", err)",
				"if typedDep, ok := dep.(*CriticalDep); ok {",
				"instance.CriticalDep = typedDep",
				"} else {",
				"return nil, fmt.Errorf(\"dependency criticalDep has wrong type, expected *CriticalDep\")",
				"if dep, exists := container.GetOptional(\"optionalDep\"); exists {",
				"if typedDep, ok := dep.(*OptionalDep); ok {",
				"instance.OptionalDep = typedDep",
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
		})
	}
}

func TestProviderFactoryModuleIntegration(t *testing.T) {
	input := `package main

@Injectable()
type UserService struct {
    @Inject("database")
    DB *Database
}

@Injectable()
type EmailService struct {
    @Inject("smtp")
    SMTP *SMTP
}

@Module({
    providers: ["UserService", "EmailService"]
})
type AppModule struct {}
`

	// Parse the .gofa file
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file: %v", err)
	}

	// Generate Go code
	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	expectedInModule := []string{
		"func (m *AppModule) Configure(container *core.DIContainer) error {",
		"// Register providers",
		"if err := RegisterUserServiceProvider(container); err != nil {",
		"return err",
		"if err := RegisterEmailServiceProvider(container); err != nil {",
		"return err",
	}

	expectedProviders := []string{
		"func NewUserService(container *core.DIContainer) (*UserService, error)",
		"func RegisterUserServiceProvider(container *core.DIContainer) error",
		"func NewEmailService(container *core.DIContainer) (*EmailService, error)",
		"func RegisterEmailServiceProvider(container *core.DIContainer) error",
	}

	// Check module integration
	for _, expected := range expectedInModule {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected module to contain: %s\nActual output:\n%s", expected, output)
		}
	}

	// Check provider generation
	for _, expected := range expectedProviders {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected providers to contain: %s\nActual output:\n%s", expected, output)
		}
	}
}

func TestProviderFactoryScopeVariations(t *testing.T) {
	tests := []struct {
		name        string
		scope       string
		expectedReg string
	}{
		{
			name:        "Singleton scope",
			scope:       "singleton",
			expectedReg: "return container.RegisterSingleton",
		},
		{
			name:        "Transient scope", 
			scope:       "transient",
			expectedReg: "return container.RegisterTransient",
		},
		{
			name:        "Request scope",
			scope:       "request",
			expectedReg: "return container.RegisterScoped",
		},
		{
			name:        "Scoped scope",
			scope:       "scoped", 
			expectedReg: "return container.RegisterScoped",
		},
		{
			name:        "Unknown scope defaults to singleton",
			scope:       "unknown",
			expectedReg: "return container.RegisterSingleton",
		},
		{
			name:        "Empty scope defaults to singleton",
			scope:       "",
			expectedReg: "return container.RegisterSingleton",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input string
			if tt.scope == "" {
				input = `package main

@Injectable()
type TestService struct {
    @Inject("test")
    Test *Test
}`
			} else {
				input = fmt.Sprintf(`package main

@Injectable("%s")
type TestService struct {
    @Inject("test")
    Test *Test
}`, tt.scope)
			}

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

			// Check registration method
			if !strings.Contains(output, tt.expectedReg) {
				t.Errorf("Expected output to contain: %s\nActual output:\n%s", tt.expectedReg, output)
			}
		})
	}
}

func TestProviderFactoryInjectionTokens(t *testing.T) {
	input := `package main

@Injectable()
type ComplexService struct {
    // No @Inject decorator - should use field name as token
    DefaultField *DefaultType
    
    // String token
    @Inject("customToken")
    CustomField *CustomType
    
    // Object with token
    @Inject({token: "objectToken"})
    ObjectField *ObjectType
    
    // Object with token and optional
    @Inject({token: "optionalToken", optional: true})
    OptionalField *OptionalType
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

	expectedTokens := []string{
		"container.Get(\"defaultfield\")",          // Field name as token
		"container.Get(\"customToken\")",           // String token
		"container.Get(\"objectToken\")",           // Object token
		"container.GetOptional(\"optionalToken\")", // Optional token
	}

	for _, expected := range expectedTokens {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain: %s\nActual output:\n%s", expected, output)
		}
	}
}