# Gofasta Core Package

The core package is the foundation of the Gofasta framework, providing enterprise-grade architectural patterns for Go applications. It implements dependency injection, modular architecture, and declarative programming patterns while maintaining Go's performance characteristics and type safety.

## Features

### 🏗️ Dependency Injection Container
- **Comprehensive DI System**: Full-featured dependency injection with lifecycle management
- **Multiple Scopes**: Singleton, Transient, and Scoped service lifetimes
- **Named Services**: Register and resolve services by name
- **Circular Dependency Detection**: Automatic detection and prevention of circular dependencies
- **Lifecycle Hooks**: Before/after create and destroy hooks for services
- **Graceful Shutdown**: Proper cleanup of resources during application shutdown

### 📦 Modular Architecture
- **Module System**: Organize your application into cohesive, reusable modules
- **Import/Export**: Control service visibility between modules
- **Decorator-Based Configuration**: Use struct tags for declarative module configuration
- **Dynamic Module Loading**: Runtime module registration and configuration
- **Module Builder**: Fluent API for building complex modules

### 🚀 Application Context
- **Application Lifecycle**: Complete application startup and shutdown management
- **Configuration Management**: Environment-based configuration with defaults
- **Graceful Shutdown**: Signal handling and graceful resource cleanup
- **Health Checks**: Built-in health check endpoints
- **Metrics Support**: Optional metrics collection and exposure

### 🔧 Service Provider Pattern
- **Multiple Provider Types**: Factory, Value, and Class providers
- **Metadata Extraction**: Automatic dependency analysis from struct tags
- **Scoped Resolution**: Context-aware service resolution
- **Custom Factories**: Support for complex service creation logic

### 🛡️ Enterprise Features
- **Guards**: Authentication and authorization middleware
- **Pipes**: Request/response transformation and validation
- **Interceptors**: Cross-cutting concerns and aspect-oriented programming
- **Exception Filters**: Centralized error handling and response formatting
- **Execution Context**: Rich context information for request processing

## Quick Start

### Basic Application Setup

```go
package main

import (
    "github.com/healtronlabs/gofasta/packages/core"
)

// Define a service
type UserService struct {
    Logger *Logger `inject:"logger"`
}

// Define a controller
type UserController struct {
    UserService *UserService `inject:""`
}

// Define a module
type AppModule struct {
    core.BaseModule
}

func (m *AppModule) Configure(container *core.DIContainer) error {
    // Register services
    container.RegisterProvider(&UserService{})
    container.RegisterController(&UserController{})
    return nil
}

func main() {
    // Create application
    app := core.CreateApp(&AppModule{})
    
    // Start and listen
    app.Listen(8080)
}
```

### Using Decorators and Struct Tags

```go
// Service with metadata
type EmailService struct {
    Config *EmailConfig `inject:"email-config"`
} `service:"email" scope:"singleton"`

// Module with declarative configuration
type UserModule struct {
    Controllers []interface{} `controllers:"UserController,AdminController"`
    Providers   []interface{} `providers:"UserService,EmailService"`
    Imports     []interface{} `imports:"DatabaseModule,AuthModule"`
    Exports     []interface{} `exports:"UserService"`
} `module:"user"`
```

### Advanced DI Container Usage

```go
// Create container
container := core.NewDIContainer()

// Register services with different scopes
container.RegisterService(
    reflect.TypeOf((*UserService)(nil)).Elem(),
    func(ctx context.Context, c *core.DIContainer) (interface{}, error) {
        return &UserService{}, nil
    },
    core.ScopeSingleton,
    "user-service",
)

// Register lifecycle hooks
container.AddLifecycleHook(
    reflect.TypeOf((*UserService)(nil)).Elem(),
    core.PhaseAfterCreate,
    func(instance interface{}) error {
        service := instance.(*UserService)
        return service.Initialize()
    },
)

// Initialize container
container.Initialize()

// Resolve services
userService, err := container.Resolve(reflect.TypeOf((*UserService)(nil)).Elem())
```

### Module Builder Pattern

```go
module := core.NewModuleBuilder().
    WithProviders(&UserService{}, &EmailService{}).
    WithControllers(&UserController{}).
    WithImports(databaseModule, authModule).
    WithExports(&UserService{}).
    WithMetadata(&core.ModuleMetadata{
        Name: "user-module",
    }).
    Build()
```

### Application Configuration

```go
config := &core.ApplicationConfig{
    Port:            8080,
    Host:            "0.0.0.0",
    Environment:     "production",
    LogLevel:        "info",
    EnableCORS:      true,
    EnableMetrics:   true,
    ShutdownTimeout: 30 * time.Second,
}

app := core.CreateApp(rootModule, config)
```

### Scoped Services

```go
// Create a scoped context
scope := app.CreateScope("request-123")

// Resolve scoped services
service, err := container.ResolveWithContext(ctx, serviceType)

// Cleanup scope when done
app.DestroyScope("request-123")
```

## Architecture

### Dependency Injection Flow

1. **Registration Phase**: Services are registered with the container
2. **Analysis Phase**: Dependencies are analyzed and circular dependencies detected
3. **Initialization Phase**: Singleton services are created in dependency order
4. **Resolution Phase**: Services are resolved on-demand with proper scoping

### Module Lifecycle

1. **Configuration**: Module configures services with the DI container
2. **Registration**: Services and controllers are registered
3. **Initialization**: Module and its dependencies are initialized
4. **Runtime**: Services are resolved and used
5. **Cleanup**: Resources are cleaned up during shutdown

### Service Scopes

- **Singleton**: One instance per application lifetime
- **Transient**: New instance every time it's resolved
- **Scoped**: One instance per scope (e.g., per HTTP request)

## Best Practices

### Service Design

```go
// Good: Interface-based dependencies
type UserService struct {
    Repository UserRepository `inject:""`
    Logger     Logger         `inject:"logger"`
}

// Good: Implement lifecycle interfaces
func (s *UserService) Initialize() error {
    return s.Repository.Connect()
}

func (s *UserService) Cleanup() error {
    return s.Repository.Disconnect()
}
```

### Module Organization

```go
// Good: Focused, cohesive modules
type UserModule struct {
    core.BaseModule
}

func (m *UserModule) Configure(container *core.DIContainer) error {
    // Register only user-related services
    container.RegisterProvider(&UserService{})
    container.RegisterProvider(&UserRepository{})
    container.RegisterController(&UserController{})
    return nil
}
```

### Error Handling

```go
// Good: Proper error handling in factories
func createUserService(ctx context.Context, container *core.DIContainer) (interface{}, error) {
    config, err := container.ResolveNamed("user-config")
    if err != nil {
        return nil, fmt.Errorf("failed to resolve user config: %w", err)
    }
    
    return &UserService{Config: config.(*UserConfig)}, nil
}
```

## Testing

### Unit Testing with Mocks

```go
func TestUserService(t *testing.T) {
    // Create test container
    container := core.NewDIContainer()
    
    // Register mocks
    mockRepo := &MockUserRepository{}
    container.RegisterInstance(
        reflect.TypeOf((*UserRepository)(nil)).Elem(),
        mockRepo,
    )
    
    // Register service under test
    container.RegisterProvider(&UserService{})
    container.Initialize()
    
    // Resolve and test
    service, err := container.Resolve(reflect.TypeOf((*UserService)(nil)).Elem())
    assert.NoError(t, err)
    
    userService := service.(*UserService)
    // Test service methods...
}
```

### Integration Testing

```go
func TestUserModule(t *testing.T) {
    // Create test application
    app := core.CreateApp(&UserModule{})
    
    // Start application
    err := app.Start()
    assert.NoError(t, err)
    defer app.Stop()
    
    // Test module functionality
    userService, err := app.GetServiceByName("user-service")
    assert.NoError(t, err)
    // Test integration...
}
```

## Performance Considerations

- **Singleton Services**: Use for expensive-to-create services
- **Transient Services**: Use for lightweight, stateless services
- **Scoped Services**: Use for request-specific state
- **Lazy Loading**: Services are created only when needed
- **Circular Dependency Detection**: Performed once during initialization
- **Thread Safety**: Container is thread-safe for concurrent access

## Migration Guide

### From Other DI Frameworks

The Gofasta DI container is designed to be familiar to developers coming from other enterprise frameworks:

- **Spring (Java)**: Similar annotation-based configuration using struct tags
- **NestJS (TypeScript)**: Similar decorator patterns and module system
- **ASP.NET Core (C#)**: Similar service lifetimes and dependency injection patterns

### Gradual Adoption

You can gradually adopt Gofasta in existing Go applications:

1. Start with the DI container for new services
2. Migrate existing services to use dependency injection
3. Organize code into modules
4. Add enterprise features (guards, interceptors, etc.)

## Contributing

The core package is the foundation of Gofasta. When contributing:

1. Maintain backward compatibility
2. Add comprehensive tests
3. Update documentation
4. Follow Go best practices
5. Consider performance implications

## License

This package is part of the Gofasta framework and is licensed under the same terms as the main project.