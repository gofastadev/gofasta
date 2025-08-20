# Provider Factory Generation in GOFASTA

This document demonstrates the Provider Factory Generation feature implemented in the GOFASTA transpiler. This feature automatically generates factory functions and provider registration helpers for services marked with the `@Injectable()` decorator.

## Overview

When you mark a service with `@Injectable()`, the transpiler automatically generates:

1. **Factory Function**: `New{ServiceName}(container *core.DIContainer) (*{ServiceName}, error)`
2. **Provider Registration Function**: `Register{ServiceName}Provider(container *core.DIContainer) error`
3. **Dependency Injection Logic**: Automatic resolution of `@Inject()` decorated fields
4. **Error Handling**: Proper error handling for missing or mistyped dependencies

## Basic Example

### Input (.gofa file):

```gofa
package main

@Injectable()
type UserService struct {
    @Inject("database")
    DB *Database
    
    @Inject()
    Logger *Logger
}

func GetUsers() []User {
    return []User{}
}
```

### Generated Output (.go file):

```go
package main

import (
    "fmt"
    "github.com/healtronlabs/gofasta/packages/core"
    // ... other imports
)

type UserService struct {
    DB     *Database `inject:"database"`
    Logger *Logger   `inject:"logger"`
}

func (s *UserService) Initialize() error {
    // TODO: Add initialization logic
    return nil
}

// Factory function for creating instances
func NewUserService(container *core.DIContainer) (*UserService, error) {
    instance := &UserService{}

    // Inject DB
    dep, err := container.Get("database")
    if err != nil {
        return nil, fmt.Errorf("failed to inject DB: %w", err)
    }
    if typedDep, ok := dep.(*Database); ok {
        instance.DB = typedDep
    } else {
        return nil, fmt.Errorf("dependency database has wrong type, expected *Database")
    }

    // Inject Logger
    dep, err = container.Get("logger")
    if err != nil {
        return nil, fmt.Errorf("failed to inject Logger: %w", err)
    }
    if typedDep, ok := dep.(*Logger); ok {
        instance.Logger = typedDep
    } else {
        return nil, fmt.Errorf("dependency logger has wrong type, expected *Logger")
    }

    if initializer, ok := interface{}(instance).(interface{ Initialize() error }); ok {
        if err := initializer.Initialize(); err != nil {
            return nil, err
        }
    }

    return instance, nil
}

// Provider registration helper
func RegisterUserServiceProvider(container *core.DIContainer) error {
    return container.RegisterSingleton("userservice", NewUserService)
}

func (s *UserService) GetUsers() []User {
    // TODO: Implement method logic
    return nil
}
```

## Advanced Features

### 1. Optional Dependencies

Services can have optional dependencies that won't cause failure if not available:

```gofa
@Injectable()
type EmailService struct {
    @Inject("smtp")
    SMTPClient *SMTPClient
    
    @Inject({token: "templates", optional: true})
    Templates *TemplateEngine
}
```

Generated code will use `container.GetOptional()` for optional dependencies.

### 2. Scope Configuration

Services can specify their lifecycle scope:

```gofa
// Singleton scope (default)
@Injectable("singleton")
type DatabaseService struct {
    @Inject("config")
    Config *DatabaseConfig
}

// Transient scope (new instance each time)
@Injectable("transient")
type RequestService struct {
    @Inject("session")
    Session *SessionStore
}

// Request/Scoped scope
@Injectable("request")
type UserContextService struct {
    @Inject("userstore")
    UserStore *UserStore
}
```

The registration function will use the appropriate method:
- `container.RegisterSingleton()` for singleton scope
- `container.RegisterTransient()` for transient scope  
- `container.RegisterScoped()` for request/scoped scope

### 3. Complex Dependency Configuration

```gofa
@Injectable()
type NotificationService struct {
    // Required dependency with custom token
    @Inject("emailProvider")
    EmailProvider *EmailService
    
    // Optional dependency with custom configuration
    @Inject({token: "smsProvider", optional: true})
    SMSProvider *SMSService
    
    // Optional dependency with scope specification
    @Inject({token: "analytics", optional: true, scope: "singleton"})
    Analytics *AnalyticsService
}
```

### 4. Module Integration

Modules automatically use the generated provider registration functions:

```gofa
@Module({
    providers: ["UserService", "EmailService", "NotificationService"]
})
type AppModule struct {}
```

Generated module code:

```go
func (m *AppModule) Configure(container *core.DIContainer) error {
    // Register providers
    if err := RegisterUserServiceProvider(container); err != nil {
        return err
    }
    if err := RegisterEmailServiceProvider(container); err != nil {
        return err
    }
    if err := RegisterNotificationServiceProvider(container); err != nil {
        return err
    }
    
    return nil
}
```

## Benefits

1. **Automatic Dependency Resolution**: No manual wiring required
2. **Type Safety**: Compile-time checking of dependency types
3. **Error Handling**: Comprehensive error messages for missing dependencies
4. **Scope Management**: Automatic lifecycle management based on scope
5. **Optional Dependencies**: Graceful handling of optional services
6. **Module Integration**: Seamless integration with GOFASTA modules

## Usage in Controllers

Controllers can inject services just like any other dependency:

```gofa
@Controller("/api/users")
type UserController struct {
    @Inject("userservice")
    UserService *UserService
    
    @Inject("logger")
    Logger *Logger
}

@Get("/")
func GetUsers() {
    // Use injected services
}
```

## Testing

The provider factory generation feature includes comprehensive test coverage:

- Basic injectable services
- Optional dependencies
- Different lifecycle scopes
- Error handling scenarios
- Module integration
- Complex dependency patterns

Run tests with:
```bash
go test -v -run TestProviderFactory
```

## Implementation Details

The feature is implemented in the GOFASTA transpiler's code generation phase:

1. **AST Analysis**: Detects `@Injectable()` decorators on service declarations
2. **Dependency Extraction**: Analyzes `@Inject()` decorators on fields
3. **Factory Generation**: Creates factory functions with proper dependency injection
4. **Registration Generation**: Creates provider registration helpers
5. **Module Integration**: Updates module configuration methods

This enables a clean, declarative approach to dependency injection that generates efficient, type-safe Go code.