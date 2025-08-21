# Scope Decorators in GOFASTA

This document demonstrates the `@Scope()` decorator feature implemented in the GOFASTA transpiler. This feature provides fine-grained control over service lifecycles and instance management in dependency injection.

## Overview

The `@Scope()` decorator allows you to specify exactly how services are instantiated and managed:

- **Singleton**: One shared instance across the entire application
- **Transient**: New instance created for every injection
- **Request/Scoped**: One instance per request or operation scope

## Syntax

### Basic Scope Declaration

```gofa
@Scope("singleton")
type DatabaseService struct {
    @Inject("config")
    Config *DatabaseConfig
}
```

### Object Syntax

```gofa
@Scope({scope: "transient"})
type EmailService struct {
    @Inject("emailconfig")
    Config *EmailConfig
}
```

## Scope Types

### 1. Singleton Scope

**Description**: Single instance shared across the entire application lifecycle.

**Use Cases**: 
- Database connections
- Configuration services
- Shared caches
- Application-wide utilities

```gofa
@Scope("singleton")
type ConfigService struct {
    @Inject("appconfig")
    Config *AppConfig
    
    // Shared configuration data
    settings map[string]string
}

func (c *ConfigService) GetSetting(key string) string {
    return c.settings[key]
}
```

**Generated Code**:
```go
func RegisterConfigServiceProvider(container *core.DIContainer) error {
    return container.RegisterSingleton("configservice", NewConfigService)
}
```

### 2. Transient Scope

**Description**: New instance created for every injection request.

**Use Cases**:
- Stateful operations
- Task processors
- Short-lived services
- Operations requiring fresh state

```gofa
@Scope("transient")
type EmailSender struct {
    @Inject("emailconfig")
    Config *EmailConfig
    
    // Fresh state for each email
    messageID string
    status    string
}

func (e *EmailSender) SendEmail(to, subject, body string) error {
    // Each email gets a fresh sender instance
    return nil
}
```

**Generated Code**:
```go
func RegisterEmailSenderProvider(container *core.DIContainer) error {
    return container.RegisterTransient("emailsender", NewEmailSender)
}
```

### 3. Request/Scoped Scope

**Description**: One instance per request, operation, or defined scope.

**Use Cases**:
- Request-specific data
- User context
- Transaction scoping
- Audit logging

```gofa
@Scope("request")
type UserContext struct {
    @Inject("userstore")
    Store *UserStore
    
    // Request-specific data
    userID    string
    sessionID string
    requestID string
}

func (u *UserContext) GetCurrentUser() (*User, error) {
    // Uses request-scoped user context
    return nil, nil
}
```

**Generated Code**:
```go
func RegisterUserContextProvider(container *core.DIContainer) error {
    return container.RegisterScoped("usercontext", NewUserContext)
}
```

## Scope Aliases

The `@Scope()` decorator supports multiple aliases for convenience:

| Standard | Aliases | Generated Registration |
|----------|---------|----------------------|
| `singleton` | `single` | `RegisterSingleton()` |
| `transient` | `prototype`, `instance` | `RegisterTransient()` |
| `request` | `scoped`, `req` | `RegisterScoped()` |

### Examples

```gofa
@Scope("prototype")  // Same as transient
type TaskProcessor struct {
    taskID string
}

@Scope("scoped")     // Same as request
type AuditLogger struct {
    requestID string
}
```

## Scope Decorator vs Injectable

### Scope Takes Precedence

When both `@Injectable` and `@Scope` decorators are present, `@Scope` takes precedence:

```gofa
@Injectable("singleton")  // This will be ignored
@Scope("transient")       // This takes precedence
type HybridService struct {
    @Inject("data")
    Data *DataStore
}
```

**Result**: Service will be registered as transient, not singleton.

### Standalone Scope Decorator

You can use `@Scope()` without `@Injectable`:

```gofa
@Scope("singleton")
type CacheService struct {
    @Inject("cacheconfig")
    Config *CacheConfig
}
```

This will automatically generate:
- Factory function: `NewCacheService()`
- Registration function: `RegisterCacheServiceProvider()`
- Proper scope registration: `RegisterSingleton()`

## Real-World Usage Patterns

### 1. Web Application Architecture

```gofa
// Singleton services - shared across all requests
@Scope("singleton")
type DatabaseService struct {
    @Inject("dbconfig")
    Config *DatabaseConfig
    
    connectionPool *ConnectionPool
}

@Scope("singleton")
type CacheService struct {
    @Inject("redis")
    Redis *RedisClient
    
    cache map[string]interface{}
}

// Request-scoped services - fresh per request
@Scope("request")
type RequestContext struct {
    @Inject("sessionstore")
    SessionStore *SessionStore
    
    userID    string
    requestID string
    startTime time.Time
}

@Scope("request")
type AuditLogger struct {
    @Inject("auditstore")
    Store *AuditStore
    
    requestID string
    userID    string
    actions   []string
}

// Transient services - fresh per operation
@Scope("transient")
type EmailSender struct {
    @Inject("emailservice")
    EmailService *EmailService
    
    messageID string
    status    string
}
```

### 2. Microservices Pattern

```gofa
// Singleton: Shared infrastructure
@Scope("singleton")
type MetricsCollector struct {
    @Inject("metricsconfig")
    Config *MetricsConfig
    
    metrics map[string]float64
}

// Transient: Per-event processing
@Scope("transient")
type EventProcessor struct {
    @Inject("metricsCollector")  // Injects singleton
    Metrics *MetricsCollector
    
    eventID string
    data    interface{}
}

// Request: Per-request correlation
@Scope("request")
type CorrelationContext struct {
    @Inject("tracing")
    Tracing *TracingService
    
    correlationID string
    spanID        string
}
```

### 3. Controller Integration

```gofa
@Controller("/api/users")
type UserController struct {
    // Singleton services - shared
    @Inject("databaseservice")
    DB *DatabaseService
    
    @Inject("cacheservice")
    Cache *CacheService
    
    // Request-scoped services - per request
    @Inject("requestcontext")
    Context *RequestContext
    
    @Inject("auditlogger")
    Audit *AuditLogger
}

@Get("/:id")
func (u *UserController) GetUser(@Param("id") userID string) {
    // Shared DB and cache, but fresh context and audit per request
}

@Post("/")
func (u *UserController) CreateUser(@Body() userData CreateUserRequest) {
    // Fresh transient EmailSender can be injected via factory
}
```

## Module Configuration

Services with different scopes work seamlessly in modules:

```gofa
@Module({
    providers: [
        "DatabaseService",    // Singleton
        "CacheService",       // Singleton
        "RequestContext",     // Request-scoped
        "AuditLogger",        // Request-scoped
        "EmailSender"         // Transient
    ]
})
type AppModule struct {}
```

**Generated Module Code**:
```go
func (m *AppModule) Configure(container *core.DIContainer) error {
    // Register providers with their respective scopes
    if err := RegisterDatabaseServiceProvider(container); err != nil {
        return err
    }
    if err := RegisterCacheServiceProvider(container); err != nil {
        return err
    }
    if err := RegisterRequestContextProvider(container); err != nil {
        return err
    }
    if err := RegisterAuditLoggerProvider(container); err != nil {
        return err
    }
    if err := RegisterEmailSenderProvider(container); err != nil {
        return err
    }
    
    return nil
}
```

## Generated Code Structure

For each service with a `@Scope()` decorator, the transpiler generates:

### 1. Factory Function

```go
func NewServiceName(container *core.DIContainer) (*ServiceName, error) {
    instance := &ServiceName{}
    
    // Dependency injection code
    // Error handling
    // Initialization
    
    return instance, nil
}
```

### 2. Provider Registration Function

```go
func RegisterServiceNameProvider(container *core.DIContainer) error {
    return container.RegisterSingleton("servicename", NewServiceName)
    // or RegisterTransient() or RegisterScoped() based on scope
}
```

### 3. Module Integration

```go
func (m *ModuleName) Configure(container *core.DIContainer) error {
    if err := RegisterServiceNameProvider(container); err != nil {
        return err
    }
    return nil
}
```

## Benefits

1. **Explicit Lifecycle Management**: Clear control over instance creation and sharing
2. **Performance Optimization**: Singletons for expensive-to-create services
3. **Memory Management**: Transient services for short-lived operations
4. **Request Isolation**: Request-scoped services for web applications
5. **Testing Support**: Easy mocking and testing with controlled lifecycles
6. **Framework Integration**: Seamless integration with GOFASTA's DI container

## Best Practices

### ✅ **Recommended Patterns**

```gofa
// Expensive resources: Singleton
@Scope("singleton")
type DatabaseConnection struct {}

// Stateful operations: Transient
@Scope("transient")
type OrderProcessor struct {}

// Request correlation: Request-scoped
@Scope("request")
type UserSession struct {}
```

### ❌ **Anti-Patterns to Avoid**

```gofa
// Don't: Heavy resources as transient
@Scope("transient")
type DatabaseConnection struct {}  // Expensive to create repeatedly

// Don't: Shared state as transient
@Scope("transient")
type GlobalCache struct {}  // Should be singleton

// Don't: Stateless services as request-scoped
@Scope("request")
type MathUtility struct {}  // Should be singleton or transient
```

## Scope Selection Guide

| Service Type | Recommended Scope | Reason |
|--------------|------------------|---------|
| Database connections | Singleton | Expensive to create, connection pooling |
| Configuration | Singleton | Shared across application |
| Caches | Singleton | Shared state |
| User context | Request | Request-specific data |
| Session data | Request | Request lifecycle |
| Task processors | Transient | Fresh state per operation |
| Event handlers | Transient | Stateful processing |
| Utility services | Singleton | Stateless, reusable |

## Testing

The scope decorator feature includes comprehensive test coverage:

```bash
# Run scope decorator tests
go test -v -run TestScopeDecorator

# Run specific test categories
go test -v -run TestScopeDecoratorBasic
go test -v -run TestScopeDecoratorCombinations
go test -v -run TestScopeDecoratorNormalization
```

## Migration from Injectable

If you're currently using `@Injectable` with scope parameters:

### Before:
```gofa
@Injectable("transient")
type EmailService struct {}

@Injectable({scope: "singleton"})
type CacheService struct {}
```

### After:
```gofa
@Scope("transient")
type EmailService struct {}

@Scope("singleton")
type CacheService struct {}
```

Both approaches work, but `@Scope()` provides clearer intent and better separation of concerns.

---

The `@Scope()` decorator brings enterprise-level dependency injection capabilities to GOFASTA, enabling you to build scalable, maintainable applications with precise control over service lifecycles.