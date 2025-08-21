# @UseGuards() Decorator Implementation

This document describes the implementation and usage of the `@UseGuards()` decorator in the GOFASTA transpiler.

## Overview

The `@UseGuards()` decorator provides middleware-based authentication and authorization for controllers and methods. It generates guard middleware functions that execute before the route handler.

## Features

✅ **Implemented:**
- Single guard on methods: `@UseGuards(AuthGuard)`
- Multiple guards on methods: `@UseGuards(AuthGuard, RoleGuard)`
- Controller-level guards: Applied to all methods in controller
- Method-level guards: Combined with controller-level guards
- Built-in guard types: `AuthGuard`, `RoleGuard`, `PermissionGuard`
- Custom guard support: Any guard name generates appropriate middleware
- Proper error handling: 401/403 responses for auth failures
- Middleware chain generation: Guards execute in specified order

## Usage Examples

### Basic Method-Level Guard

```gofa
type UserController struct {}

@UseGuards(AuthGuard)
@Get("/profile")
func getProfile() {
}
```

**Generated code:**
```go
func (c *UserController) RegisterRoutes(server *httpPackage.HTTPServer) error {
    server.Get("/user/profile", c.AuthGuard, c.getProfile)
    return nil
}

func (c *UserController) AuthGuard(ctx *httpPackage.RequestContext) {
    // Authentication guard logic
    token := ctx.GetHeader("Authorization")
    if token == "" {
        ctx.JSON(401, map[string]string{"error": "Unauthorized: Missing authentication token"})
        ctx.Abort()
        return
    }
    // ... validation logic
    ctx.Next()
}
```

### Multiple Guards

```gofa
@UseGuards(AuthGuard, RoleGuard, PermissionGuard)
@Post("/admin/delete")
func deleteResource() {
}
```

**Generated middleware chain:**
```go
server.Post("/controller/admin/delete", c.AuthGuard, c.RoleGuard, c.PermissionGuard, c.deleteResource)
```

### Controller-Level Guards

```gofa
@Controller("/admin")
@UseGuards(AuthGuard)
type AdminController struct {}

@Get("/users")           // Protected by AuthGuard
func getUsers() {}

@UseGuards(RoleGuard)    // Protected by AuthGuard + RoleGuard
@Delete("/user/:id")
func deleteUser() {}
```

## Built-in Guard Types

### AuthGuard
- Validates `Authorization` header
- Checks for valid authentication token
- Returns 401 for missing/invalid tokens

### RoleGuard
- Validates user roles from context
- Requires authenticated user
- Returns 403 for insufficient permissions

### PermissionGuard
- Validates specific permissions
- Fine-grained access control
- Returns 403 for insufficient permissions

### Custom Guards
- Any guard name generates middleware template
- Includes TODO comments for implementation
- Returns 403 for failed guard conditions

## Implementation Details

### Code Generation

1. **Middleware Methods**: Generated as controller methods (`func (c *Controller) GuardName(ctx *RequestContext)`)
2. **Route Registration**: Guards inserted as middleware in route registration
3. **Guard Chain**: Multiple guards execute in declaration order
4. **Error Handling**: Guards use `ctx.Abort()` to stop execution
5. **Success Path**: Guards call `ctx.Next()` to continue

### Guard Execution Order

1. Controller-level guards (in declaration order)
2. Method-level guards (in declaration order)
3. Route handler

### Error Responses

- **401 Unauthorized**: Missing or invalid authentication
- **403 Forbidden**: Insufficient permissions or failed guard conditions

## Testing

The implementation includes comprehensive test coverage:

- `TestUseGuardsDecorator`: End-to-end transpilation tests
- `TestUseGuardsDecoratorParsing`: Decorator parsing verification
- `TestGuardMiddlewareGeneration`: Middleware generation validation
- `TestGuardErrorHandling`: Error response verification

## Example Test Cases

See `useguards_decorator_test.go` for complete test examples including:
- Single and multiple guards
- Controller and method-level combinations
- Built-in and custom guard types
- Error handling scenarios

## Integration

The `@UseGuards()` decorator integrates with:
- HTTP method decorators (`@Get`, `@Post`, etc.)
- Parameter decorators (`@Body`, `@Query`, etc.)
- Other middleware decorators (when implemented)
- Dependency injection system
- Error handling framework

## Future Enhancements

Potential improvements for future versions:
- Guard parameter passing
- Conditional guard application
- Dynamic guard resolution
- Guard composition patterns
- Integration with validation decorators