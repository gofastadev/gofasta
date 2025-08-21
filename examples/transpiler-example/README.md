# Gofasta Transpiler Example

This example demonstrates how to use the Gofasta transpiler to transform decorator-based `.gofa` files into standard Go code.

## 🎯 What This Example Shows

- **Controller Declaration**: Using `@Controller` with route prefixes
- **Route Methods**: `@Get`, `@Post`, `@Put`, `@Delete` decorators
- **Dependency Injection**: Using `inject:""` tags
- **Parameter Decorators**: `@Param`, `@Body`, `@Query`, `@Headers` (with advanced features) for request handling
- **Service Pattern**: `@Injectable` services with DI
- **Provider Factory Generation**: Automatic factory functions for `@Injectable` services
- **Scope Decorators**: `@Scope()` for fine-grained service lifecycle management
- **Module System**: `@Module` for organizing dependencies
- **Guards & Middleware**: `@UseGuards`, `@UsePipes`, `@UseInterceptors`, `@HttpCode` decorators
- **Error Handling**: `@Catch()` decorators for automatic error filtering
- **HTTP Status Codes**: `@HttpCode()` decorators for custom response status codes
- **Redirects**: `@Redirect()` decorators for URL redirections with custom status codes

## 📁 File Structure

```
transpiler-example/
├── README.md                 # This file
├── go.mod                   # Go module definition
├── main.go                  # Demo server to run the example
├── *.gofa                   # Source files (decorator-based)
│   ├── user.controller.gofa # User CRUD controller with @Query and @Headers examples
│   ├── product.controller.gofa # Product controller showcasing advanced @Query features
│   ├── headers-example.gofa # Comprehensive @Headers decorator examples
│   ├── catch-decorator-example.gofa # Error handling with @Catch() decorator
│   ├── httpcode-example.gofa # HTTP status codes with @HttpCode() decorator
│   ├── redirect-example.gofa # URL redirections with @Redirect() decorator
│   ├── usepipes-example.gofa # Pipe validation and transformation examples
│   ├── user.service.gofa    # User business logic service  
│   ├── types.gofa           # Data models and DTOs
│   ├── simple-test.gofa     # Simple test controller
│   ├── provider-demo.gofa   # Provider factory generation demo
│   ├── provider-factory-example.gofa # Comprehensive provider factory example
│   ├── PROVIDER_FACTORY_README.md    # Provider factory documentation
│   ├── scope-demo.gofa      # Scope decorator demonstration  
│   ├── SCOPE_DECORATOR_README.md     # Scope decorator documentation
│   └── app.module.gofa      # Application module configuration
└── *.go                     # Generated files (standard Go)
    ├── user.controller.go   # Generated from user.controller.gofa
    ├── product.controller.go # Generated from product.controller.gofa
    ├── user.service.go      # Generated from user.service.gofa
    ├── types.go             # Generated from types.gofa
    ├── simple-test.go       # Generated from simple-test.gofa
    └── app.module.go        # Generated from app.module.gofa
```

## 🚀 Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **Gofasta transpiler** built (see build instructions below)

### Step 1: Build the Transpiler

```bash
# From the project root
cd packages/transpiler
go build -o gofasta ./cmd/gofasta-transpiler
```

### Step 2: Run the Transpiler

```bash
# Navigate to this example directory
cd examples/transpiler-example

# Transpile all .gofa files to .go files
../../tools/transpiler/gofasta transpile -verbose

# Or use force flag to overwrite existing files
../../tools/transpiler/gofasta transpile -verbose -force
```

### Step 3: Run the Example

```bash
# Run the generated Go code
go run .
```

You should see output like:
```
🚀 Starting Gofasta Transpiler Example
=====================================
✅ All .gofa files have been transpiled to .go files
🌐 Server starting on http://localhost:8080

📍 Available endpoints:
  GET  /                         - API information
  
  👥 User Management:
  GET  /api/v1/users             - Get users (with pagination, search, filters)
  GET  /api/v1/users/search      - Search users (required query param)
  GET  /api/v1/users/:id         - Get user by ID (with field selection)
  POST /api/v1/users             - Create new user
  PUT  /api/v1/users/:id         - Update user
  DELETE /api/v1/users/:id       - Delete user
  
  🛍️ Product Management:
  GET  /api/v1/products          - Get products (comprehensive filtering)
  GET  /api/v1/products/search   - Search products (required query)
  GET  /api/v1/products/recommendations - Get product recommendations
  GET  /api/v1/products/analytics - Get product analytics
  GET  /api/v1/products/:id      - Get product by ID (with localization)
  POST /api/v1/products          - Create new product
  PUT  /api/v1/products/:id      - Update product
  DELETE /api/v1/products/:id    - Delete product
  
  🧪 Simple Test:
  GET  /api/test/hello           - Simple hello endpoint

🔗 Try: curl http://localhost:8080/
```

## 🧪 Testing the Generated API

### Basic API Information
```bash
curl http://localhost:8080/
```

### Test Simple Endpoint
```bash
curl http://localhost:8080/api/test/hello
```

### Test Enhanced @Query Parameters

#### 👥 User Endpoints with Advanced Queries

```bash
# Get all users (basic)
curl http://localhost:8080/api/v1/users

# Get users with pagination and sorting
curl "http://localhost:8080/api/v1/users?page=2&limit=5&sort=email&active=true"

# Get users with array filters
curl "http://localhost:8080/api/v1/users?roles=admin,editor&search=john"

# Search users (required query parameter)
curl "http://localhost:8080/api/v1/users/search?q=developer&minAge=25&maxAge=40&exactMatch=true"

# Search with departments (pipe-separated array)
curl "http://localhost:8080/api/v1/users/search?q=senior&departments=engineering|design|product"

# Get user by ID with field selection
curl "http://localhost:8080/api/v1/users/123?include=profile,permissions,stats"
```

#### 🛍️ Product Endpoints with Comprehensive Filtering

```bash
# Get products with comprehensive filters (category is required)
curl "http://localhost:8080/api/v1/products?category=electronics&page=1&limit=10"

# Advanced product filtering
curl "http://localhost:8080/api/v1/products?category=gadgets&sort=price&order=desc&minPrice=50&maxPrice=500&inStock=true&featured=true"

# Array filters with different separators
curl "http://localhost:8080/api/v1/products?category=tech&tags=wireless,portable&brands=apple|samsung&features=wifi;bluetooth;waterproof"

# Product search (required query)
curl "http://localhost:8080/api/v1/products/search?q=smartphone&type=exact&fields=name,description,tags"

# Product recommendations
curl "http://localhost:8080/api/v1/products/recommendations?userId=123&algorithm=collaborative&limit=5&excludeOwned=true"

# Product analytics (required metric)
curl "http://localhost:8080/api/v1/products/analytics?metric=sales&period=7d&productIds=1,2,3&groupBy=category|brand"

# Get product with localization
curl "http://localhost:8080/api/v1/products/456?currency=EUR&locale=de-DE&include=reviews,specs"
```

#### ⚠️ Error Examples (showcasing validation)

```bash
# Missing required parameter (will return 400)
curl "http://localhost:8080/api/v1/users/search"  # Missing required 'q' parameter

# Missing required category for products (will return 400)
curl "http://localhost:8080/api/v1/products"     # Missing required 'category' parameter

# Invalid type conversion (will return 400)
curl "http://localhost:8080/api/v1/users?page=invalid"  # Invalid integer value
```

#### 📊 Default Values and Transformations Demo

```bash
# Default values applied when parameters are missing
curl "http://localhost:8080/api/v1/users"
# Uses: page=1, limit=10, sort="name"

# String transformations (lowercase, uppercase, trim)
curl "http://localhost:8080/api/v1/products?category=ELECTRONICS&sort=PRICE&search=  smartphone  "
# Results in: category="electronics", sort="price", search="smartphone"

# Boolean parsing
curl "http://localhost:8080/api/v1/users?active=true&roles=admin"
curl "http://localhost:8080/api/v1/users?active=1&roles=admin"      # Also works
curl "http://localhost:8080/api/v1/users?active=false&roles=admin"
```

### Test Basic User CRUD

```bash
# Create user (POST)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","age":30}'

# Update user (PUT)
curl -X PUT http://localhost:8080/api/v1/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane Doe","email":"jane@example.com"}'

# Delete user
curl -X DELETE http://localhost:8080/api/v1/users/123
```

## 📊 How to Verify It's Working

### ✅ Successful Transpilation Signs

1. **No Errors**: Transpiler reports `✅ All files transpiled successfully!`
2. **Generated Files**: All `.go` files are created from `.gofa` files
3. **Valid Go Code**: Generated files compile without errors
4. **Server Starts**: The demo server starts on port 8080
5. **Routes Work**: All endpoints respond correctly

### ❌ Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| `parsing errors: unexpected token` | Check `.gofa` syntax - remove unsupported features like interfaces or package-qualified types |
| `output files already exist` | Use `-force` flag to overwrite existing files |
| `go: cannot find module` | Run `go mod tidy` to resolve dependencies |
| `server failed to start` | Check if port 8080 is available or change the port |

## 🔍 Enhanced Parameter Decorators

### 🚀 @Query() Decorator Features

The enhanced `@Query()` decorator supports **NestJS-level parity** with advanced features:

#### 🎯 Basic Usage
```go
@Query("paramName") paramValue string
@Query("page") page int        // Auto-converted from string to int
@Query("active") active bool   // Supports: true/false, 1/0, yes/no
```

#### 🎛️ Default Values
```go
@Query("page", { defaultValue: "1" }) page int
@Query("sort", { defaultValue: "name" }) sortBy string
@Query("limit", { defaultValue: "10" }) limit int
```

#### ✅ Required Parameters
```go
@Query("category", { required: true }) category string
// Returns 400 error if missing: "Query parameter 'category' is required"
```

#### 🔄 String Transformations
```go
@Query("sort", { transform: "lowercase" }) sortBy string     // "NAME" → "name"
@Query("currency", { transform: "uppercase" }) currency string // "usd" → "USD" 
@Query("search", { transform: "trim" }) query string         // "  hello  " → "hello"
```

#### 🎨 Type Conversion (Auto-detected)
```go
@Query("page") page int                    // String → Int with validation
@Query("price") price float64              // String → Float with validation  
@Query("active") active bool               // String → Bool (true/false/1/0)
@Query("tags") tags []string               // String → Array (comma-separated)
```

#### 📋 Array Parameters
```go
// Different separators for arrays
@Query("tags") tags []string                                    // Default: comma
@Query("brands", { type: "array", separator: "|" }) brands []string    // Pipe-separated
@Query("features", { separator: ";" }) features []string       // Semicolon-separated

// Example URLs:
// ?tags=electronics,gadgets,new
// ?brands=apple|samsung|google  
// ?features=wifi;bluetooth;waterproof
```

#### 🎭 Complex Combined Features
```go
@Query("sort", { 
  defaultValue: "name", 
  transform: "lowercase" 
}) sortBy string

@Query("categories", { 
  type: "array", 
  separator: "|", 
  defaultValue: "general" 
}) categories []string

@Query("query", { 
  required: true, 
  transform: "trim" 
}) searchQuery string
```

### 🔄 Generated Code Quality

**Before (Basic):**
```go
func GetUsers(ctx *httpPackage.RequestContext) {
    page := ctx.GetQuery("page")  // Always string, no validation
    // Manual conversion and validation needed
}
```

**After (Enhanced):**
```go
func GetUsers(ctx *httpPackage.RequestContext) {
    var page int
    queryValue := ctx.GetQuery("page")
    if queryValue == "" {
        queryValue = "1"  // Default value applied
    }
    if queryValue != "" {
        if parsedInt, err := strconv.Atoi(queryValue); err == nil {
            page = parsedInt
        } else {
            ctx.JSON(400, map[string]string{"error": "Invalid integer value for parameter 'page'"})
            return
        }
    }
    // page is now properly typed and validated!
}
```

### 📊 Error Handling & Validation

The enhanced `@Query` decorator provides comprehensive error handling:

1. **Type Validation**: Automatic conversion with error responses
2. **Required Validation**: Missing required parameters return 400 errors
3. **Descriptive Errors**: Clear error messages for debugging
4. **Graceful Defaults**: Fallback values when parameters are missing

### 🚀 @Headers() Decorator Features

The new `@Headers()` decorator provides powerful HTTP header extraction with **NestJS-level parity**:

#### 🎯 Basic Usage
```go
@Headers("Authorization") authToken string
@Headers("User-Agent") userAgent string
@Headers("Content-Type") contentType string
```

#### ✅ Required Headers
```go
@Headers("X-API-Key", { required: true }) apiKey string
@Headers("Authorization", { required: true }) authToken string
// Returns 400 error if missing: "Header 'Authorization' is required"
```

#### 🎛️ Default Values
```go
@Headers("X-API-Version", { defaultValue: "v1" }) apiVersion string
@Headers("Content-Type", { defaultValue: "application/json" }) contentType string
@Headers("X-Timeout", { defaultValue: "30" }) timeout int
```

#### 🔄 String Transformations
```go
@Headers("User-Agent", { transform: "lowercase" }) userAgent string     // Case-insensitive processing
@Headers("X-Service", { transform: "uppercase" }) service string        // Force uppercase
@Headers("X-Token", { transform: "trim" }) token string                 // Remove whitespace
```

#### 🎨 Type Conversion (Auto-detected)
```go
@Headers("Content-Length") contentLength int              // String → Int with validation
@Headers("X-Rate-Limit") rateLimit float64               // String → Float with validation  
@Headers("X-Debug") debug bool                           // String → Bool (true/false/1/0)
@Headers("Accept") acceptTypes []string                  // String → Array (comma-separated)
```

#### 📋 Array Headers (Comma-Separated Values)
```go
// Different separators for array headers
@Headers("Accept") acceptTypes []string                                    // Default: comma
@Headers("X-Tags", { type: "array", separator: "|" }) tags []string      // Pipe-separated
@Headers("X-Features", { separator: ";" }) features []string             // Semicolon-separated

// Example headers:
// Accept: application/json, application/xml, text/html
// X-Tags: electronics|gadgets|new  
// X-Features: wifi;bluetooth;waterproof
```

#### 🎭 Complex Combined Features
```go
@Headers("Authorization", { 
  required: true, 
  transform: "trim" 
}) authToken string

@Headers("Content-Type", { 
  required: true,
  defaultValue: "application/json", 
  transform: "lowercase" 
}) contentType string

@Headers("X-Custom-Headers", { 
  type: "array", 
  separator: ";",
  defaultValue: "default"
}) customHeaders []string
```

#### 🌐 Common HTTP Headers Examples
```go
// Standard HTTP headers
@Headers("Host") host string
@Headers("Referer") referer string
@Headers("Origin") origin string                        // CORS
@Headers("User-Agent") userAgent string
@Headers("Accept-Language") acceptLanguage string
@Headers("Cache-Control") cacheControl string
@Headers("Connection") connection string

// Custom application headers
@Headers("X-Forwarded-For") forwardedFor string        // Proxy headers
@Headers("X-Real-IP") realIP string
@Headers("X-Request-ID") requestId string              // Request tracing
@Headers("X-Client-Version") clientVersion string      // Client information
```

### 🔄 Generated Header Processing Code

**Before (Basic):**
```go
func CreateUser(ctx *httpPackage.RequestContext) {
    auth := ctx.GetHeader("Authorization")  // Always string, no validation
    // Manual validation and conversion needed
}
```

**After (Enhanced):**
```go
func CreateUser(ctx *httpPackage.RequestContext) {
    var auth string
    headerValue := ctx.GetHeader("Authorization")
    if headerValue == "" {
        ctx.JSON(400, map[string]string{"error": "Header 'Authorization' is required"})
        return
    }
    headerValue = strings.TrimSpace(headerValue)
    auth = headerValue

    var contentLength int
    headerValue := ctx.GetHeader("Content-Length")
    if headerValue != "" {
        if parsedInt, err := strconv.Atoi(headerValue); err == nil {
            contentLength = parsedInt
        } else {
            ctx.JSON(400, map[string]string{"error": "Invalid integer value for header 'contentLength'"})
            return
        }
    }
    // Headers are now properly typed and validated!
}
```

### 📊 Header Processing Features

The enhanced `@Headers` decorator provides:

1. **Type Validation**: Automatic conversion with error responses for int, float, bool types
2. **Required Validation**: Missing required headers return 400 errors  
3. **Default Values**: Fallback values when headers are missing
4. **String Transformations**: lowercase, uppercase, trim operations
5. **Array Processing**: Split comma-separated (or custom separator) values
6. **HTTP Spec Compliance**: Case-insensitive header names by default
7. **Descriptive Errors**: Clear error messages for debugging

### 🚀 @Catch() Decorator Features

The new `@Catch()` decorator provides automatic error handling and filtering with **NestJS-level parity**:

#### 🎯 Basic Usage - Method Level
```go
// Handle specific error types in a method
@Catch(NotFoundError)
@Get("/:id")
func GetUser(@Param("id") id string) {
    // If NotFoundError is thrown, it will be caught and handled automatically
    if id == "999" {
        panic(&NotFoundError{Resource: "User", ID: id})
    }
}
```

#### 🏢 Controller Level Error Handling
```go
// Handle errors for all methods in the controller
@Controller("/api/v1/users")
@Catch(InternalServerError)  // Catches system errors for all methods
type UserController struct {
    UserService *UserService `inject:""`
    Logger      *Logger      `inject:"logger"`
}
```

#### 🎭 Multiple Error Types
```go
// Handle multiple specific error types
@Catch(BadRequestError, ValidationError)
@Post("/")
func CreateUser(@Body() userData CreateUserDto) {
    // Both BadRequestError and ValidationError will be caught
    if userData.Email == "" {
        panic(&ValidationError{Message: "Email is required"})
    }
    if userData.Age < 0 {
        panic(&BadRequestError{Message: "Invalid age"})
    }
}
```

#### 🌐 Global Error Handlers
```go
// Catch ALL error types (global handler)
@Catch()  // Empty parentheses = catch everything
@Put("/:id")
func UpdateUser(@Param("id") id string, @Body() updateData UpdateUserDto) {
    // Any error thrown will be caught by this global handler
    switch id {
    case "auth_error":
        panic(&UnauthorizedError{Message: "Not authorized"})
    case "system_error":
        panic(&InternalServerError{Message: "System failure"})
    }
}
```

#### 🏗️ Complex Nested Error Handling
```go
@Controller("/api/v1/orders")
@Catch(InternalServerError, UnauthorizedError)  // Controller-level handlers
type OrderController struct {
    OrderService *OrderService `inject:""`
}

@Catch(NotFoundError, ValidationError)  // Method-level handlers
@Get("/:orderId/items/:itemId")
func GetOrderItem(
    @Param("orderId") orderId string,
    @Param("itemId") itemId string,
) {
    // Error handling priority:
    // 1. Method-level @Catch() for NotFoundError, ValidationError
    // 2. Controller-level @Catch() for InternalServerError, UnauthorizedError
    
    if orderId == "" {
        panic(&ValidationError{Message: "Order ID required"})  // Method handler
    }
    if orderId == "unauthorized" {
        panic(&UnauthorizedError{Message: "Access denied"})     // Controller handler
    }
}
```

### 🔄 Generated Error Handling Code

**Before (Manual Error Handling):**
```go
func GetUser(ctx *httpPackage.RequestContext) {
    // Manual error handling required
    defer func() {
        if r := recover(); r != nil {
            // Manual error type checking and response generation
            switch e := r.(type) {
            case *NotFoundError:
                ctx.JSON(404, map[string]string{"error": e.Error()})
            default:
                ctx.JSON(500, map[string]string{"error": "Internal server error"})
            }
        }
    }()
    
    // Method logic here...
}
```

**After (@Catch() Generated):**
```go
// Generated error handler method
func (c *UserController) handleGetUserError(err error, ctx *httpPackage.RequestContext) {
    switch e := err.(type) {
    case *NotFoundError:
        ctx.JSON(404, map[string]string{"error": e.Error()})
    default:
        ctx.JSON(500, map[string]string{"error": "Internal server error"})
    }
}

// Generated error filter registration
func (c *UserController) RegisterRoutes(server *httpPackage.HTTPServer) error {
    // Register error filters first
    server.RegisterMethodErrorFilter("GetUser", "NotFoundError", c.handleGetUserError)
    
    // Then register routes
    server.Get("/api/v1/users/:id", c.GetUser)
    return nil
}

// Clean method implementation
func (c *UserController) GetUser(ctx *httpPackage.RequestContext) {
    id := ctx.GetParam("id")
    // Method logic - errors are automatically caught and handled
    if id == "999" {
        panic(&NotFoundError{Resource: "User", ID: id})  // Automatically handled!
    }
}
```

### 📊 Error Types and HTTP Status Codes

The `@Catch()` decorator automatically maps error types to appropriate HTTP status codes:

| Error Type | HTTP Status | Description |
|------------|-------------|-------------|
| `BadRequestError` | 400 | Invalid client request |
| `UnauthorizedError` | 401 | Authentication required |
| `ForbiddenError` | 403 | Access forbidden |
| `NotFoundError` | 404 | Resource not found |
| `ValidationError` | 422 | Validation failed |
| `ConflictError` | 409 | Resource conflict |
| `InternalServerError` | 500 | System error |
| Custom errors | 500 | Default to server error |

### 🎯 Error Handling Scopes

#### 1. Method Scope
```go
@Catch(ValidationError)
@Post("/users")
func CreateUser() {
    // Only handles ValidationError for this specific method
}
```

#### 2. Controller Scope  
```go
@Controller("/api/users")
@Catch(UnauthorizedError)  // Applies to ALL methods in controller
type UserController struct {}
```

#### 3. Mixed Scope (Priority: Method > Controller)
```go
@Controller("/api/orders")
@Catch(InternalServerError)  // Controller-level: system errors
type OrderController struct {}

@Catch(ValidationError)      // Method-level: validation errors
@Post("/")
func CreateOrder() {
    // ValidationError → Method handler (priority)
    // InternalServerError → Controller handler
    // Other errors → Default 500 response
}
```

### 🛡️ Error Handling Best Practices

#### ✅ **Recommended Patterns**
```go
// 1. Specific errors at method level
@Catch(ValidationError, BadRequestError)
@Post("/users")
func CreateUser() { /* ... */ }

// 2. System errors at controller level  
@Controller("/api/users")
@Catch(InternalServerError, DatabaseError)
type UserController struct {}

// 3. Global fallback for critical methods
@Catch()  // Catches everything
@Delete("/users/:id")
func DeleteUser() { /* ... */ }
```

#### ❌ **Anti-Patterns to Avoid**
```go
// Don't: Too many error types in one handler
@Catch(Error1, Error2, Error3, Error4, Error5, Error6)  // Hard to maintain

// Don't: Overly broad controller-level handlers
@Controller("/api")
@Catch()  // Catches everything for entire controller - too broad

// Don't: Conflicting error handlers
@Controller("/api/users")
@Catch(ValidationError)     // Controller handles ValidationError
type UserController struct {}

@Catch(ValidationError)     // Method also handles ValidationError - conflict!
@Post("/")
func CreateUser() {}
```

### 🎭 Real-World Error Handling Examples

#### E-commerce Order Processing
```go
@Controller("/api/v1/orders")
@Catch(InternalServerError)  // System-level failures
type OrderController struct {
    OrderService    *OrderService    `inject:""`
    PaymentService  *PaymentService  `inject:""`
    InventoryService *InventoryService `inject:""`
}

@Catch(ValidationError, BadRequestError)  // Input validation
@Post("/")
func CreateOrder(@Body() orderData CreateOrderDto) {
    // Validation errors caught automatically
    if len(orderData.Items) == 0 {
        panic(&ValidationError{Message: "Order must contain items"})
    }
}

@Catch(NotFoundError)  // Resource lookup
@Get("/:orderId")
func GetOrder(@Param("orderId") orderId string) {
    // 404 errors handled automatically
    if orderId == "nonexistent" {
        panic(&NotFoundError{Resource: "Order", ID: orderId})
    }
}

@Catch(ConflictError)  // Business logic conflicts
@Put("/:orderId/cancel") 
func CancelOrder(@Param("orderId") orderId string) {
    // Conflict errors handled automatically
    if orderStatus == "shipped" {
        panic(&ConflictError{Message: "Cannot cancel shipped order"})
    }
}
```

#### User Authentication & Authorization
```go
@Controller("/api/v1/admin")
@Catch(UnauthorizedError, ForbiddenError)  // Auth errors for admin area
type AdminController struct {}

@Catch(ValidationError)  // Login validation
@Post("/login")
func AdminLogin(@Body() credentials LoginDto) {
    if credentials.Username == "" {
        panic(&ValidationError{Message: "Username required"})
    }
    if !isValidCredentials(credentials) {
        panic(&UnauthorizedError{Message: "Invalid credentials"})
    }
}

@Catch()  // Global handler for sensitive operations
@Delete("/users/:id")
func DeleteUser(@Param("id") id string) {
    // Any error in user deletion is caught and handled securely
    if !hasAdminPermission() {
        panic(&ForbiddenError{Message: "Admin access required"})
    }
}
```

### 📈 Performance & Benefits

The `@Catch()` decorator provides:

1. **Automatic Error Handling**: No manual try-catch blocks needed
2. **Consistent Error Responses**: Standardized HTTP status codes and messages
3. **Code Cleanliness**: Business logic separated from error handling
4. **Type Safety**: Compile-time error type checking
5. **Maintainability**: Centralized error handling logic
6. **Debugging**: Clear error paths and stack traces
7. **Performance**: Optimized error filter registration

### 🚀 @HttpCode() Decorator Features

The new `@HttpCode()` decorator provides custom HTTP status code setting with **NestJS-level parity**:

#### 🎯 Basic Usage
```go
// Set custom status code for responses
@HttpCode(201)
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {
    // Returns 201 Created instead of default 200 OK
}
```

#### 🏗️ REST API Status Codes
```go
@Controller("/api/v1/users")
type UserController struct {}

// GET endpoints - use default 200 OK
@Get("/")
func GetUsers() {}

// POST endpoints - return 201 Created for new resources
@HttpCode(201)
@Post("/")
func CreateUser(@Body() userData CreateUserDto) {}

// PUT endpoints - use default 200 OK for updates
@Put("/:id")  
func UpdateUser(@Param("id") id string, @Body() updateData UpdateUserDto) {}

// DELETE endpoints - return 204 No Content
@HttpCode(204)
@Delete("/:id")
func DeleteUser(@Param("id") id string) {}
```

#### ⚡ Async Operations
```go
@Controller("/api/jobs")
type JobController struct {}

// Async operations - return 202 Accepted
@HttpCode(202)
@Post("/")
func StartAsyncJob(@Body() jobData JobDto) {}

@HttpCode(202)
@Put("/:id/restart")
func RestartJob(@Param("id") id string) {}

// Status check - use default 200 OK
@Get("/:id/status")
func GetJobStatus(@Param("id") id string) {}
```

#### 🗂️ Caching and Special Cases
```go
@Controller("/api/cache")
type CacheController struct {}

// Not Modified responses
@HttpCode(304)
@Get("/data")
func GetCachedData(@Headers("If-Modified-Since") ifModifiedSince string) {}

// Cache operations - No Content
@HttpCode(204)
@Delete("/cache")
func ClearCache() {}
```

### 🔄 Generated HTTP Status Code

**Before (@HttpCode() decorator):**
```go
func (c *UserController) CreateUser(ctx *httpPackage.RequestContext) {
    // Default behavior - always returns 200
    ctx.JSON(200, map[string]interface{}{"message": "User created"})
}
```

**After (@HttpCode(201) decorator):**
```go
func (c *UserController) CreateUser(ctx *httpPackage.RequestContext) {
    ctx.Status(201)  // Set status code first

    var userData CreateUserDto
    if err := ctx.ParseJSON(&userData); err != nil {
        ctx.JSON(400, map[string]string{"error": "Invalid request body"})
        return
    }
    // TODO: Implement method logic
    ctx.JSON(201, map[string]interface{}{"message": "Not implemented"})  // Use custom status
}
```

### 📊 Common HTTP Status Codes

| Status Code | Description | Use Case | Example |
|-------------|-------------|----------|---------|
| **200** | OK | Default success | `@Get()` endpoints |
| **201** | Created | Resource creation | `@HttpCode(201) @Post()` |
| **202** | Accepted | Async operations | `@HttpCode(202)` for jobs |
| **204** | No Content | Successful deletion | `@HttpCode(204) @Delete()` |
| **304** | Not Modified | Cached responses | `@HttpCode(304)` for cache |

### 🎭 Status Code Setting Order

The `@HttpCode()` decorator ensures proper status code setting:

1. **Status Set First**: `ctx.Status(code)` is called immediately after method entry
2. **Before Parameter Validation**: Status is set before any potential early returns
3. **Consistent Response**: Final `ctx.JSON()` uses the same status code

```go
@HttpCode(201)
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {
    // Generated code order:
    // 1. ctx.Status(201)           ← Set status first
    // 2. Parameter validation       ← May return early with 400
    // 3. Business logic            ← Implementation
    // 4. ctx.JSON(201, result)     ← Final response with correct status
}
```

### 🛡️ Error Handling with HttpCode

The `@HttpCode()` decorator works seamlessly with error handling:

```go
@HttpCode(201)
@Catch(ValidationError)
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {
    // Success case: Returns 201 Created
    // Error case: ValidationError caught by @Catch() decorator
    // Validation failure: Returns 400 Bad Request (parameter validation)
}
```

### 🎯 Best Practices

#### ✅ **Recommended Usage**
```go
// RESTful resource creation
@HttpCode(201)
@Post("/users")
func CreateUser() {}

// Successful deletion
@HttpCode(204)  
@Delete("/users/:id")
func DeleteUser() {}

// Async operations
@HttpCode(202)
@Post("/jobs")
func StartJob() {}

// Conditional responses
@HttpCode(304)
@Get("/data")
func GetCachedData() {}
```

#### ❌ **Avoid These Patterns**
```go
// Don't: Unnecessary HttpCode for default 200
@HttpCode(200)  // Redundant - 200 is default
@Get("/users")
func GetUsers() {}

// Don't: Wrong status for operation type
@HttpCode(201)  // Wrong - updates should be 200
@Put("/users/:id")
func UpdateUser() {}

// Don't: Generic status for specific operations
@HttpCode(200)  // Should be 204 for deletions
@Delete("/users/:id")
func DeleteUser() {}
```

### 🏗️ Real-World Examples

#### E-commerce API
```go
@Controller("/api/v1/orders")
type OrderController struct {}

@HttpCode(201)  // Created
@Post("/")
func CreateOrder(@Body() orderData CreateOrderDto) {}

@HttpCode(202)  // Accepted for processing
@Post("/:id/ship")
func ShipOrder(@Param("id") id string) {}

@HttpCode(204)  // No content after cancellation
@Delete("/:id")
func CancelOrder(@Param("id") id string) {}

@Get("/:id")    // Default 200 OK
func GetOrder(@Param("id") id string) {}
```

#### File Management API
```go
@Controller("/api/files")
type FileController struct {}

@HttpCode(201)  // File created
@Post("/upload")
func UploadFile(@Body() fileData FileUploadDto) {}

@HttpCode(202)  // Processing accepted
@Post("/:id/process")
func ProcessFile(@Param("id") id string) {}

@HttpCode(204)  // File deleted
@Delete("/:id")
func DeleteFile(@Param("id") id string) {}

@HttpCode(304)  // Not modified
@Get("/:id")
func GetFile(@Headers("If-Modified-Since") ifModifiedSince string) {}
```

### 📈 Benefits

The `@HttpCode()` decorator provides:

1. **RESTful Compliance**: Proper HTTP status codes for different operations
2. **Clear Intent**: Explicit status codes make API behavior obvious
3. **Client Integration**: Better client-side handling with correct status codes
4. **Automatic Setting**: No manual `ctx.Status()` calls needed
5. **Early Setting**: Status set before any potential early returns
6. **Consistency**: Generated code always uses the specified status
7. **NestJS Compatibility**: Familiar decorator syntax for NestJS developers

### 🚀 @Redirect() Decorator Features

The new `@Redirect()` decorator provides powerful URL redirection capabilities with **NestJS-level parity**:

#### 🎯 Basic Usage
```go
// Basic temporary redirect (302) with default status code
@Redirect("https://newdomain.com/users")
@Get("/old-users")
func RedirectUsers() {}

// Permanent redirect (301) for moved resources
@Redirect("https://newdomain.com/products", 301)
@Get("/old-products")
func RedirectProducts() {}
```

#### 🏗️ HTTP Redirect Status Codes
```go
@Controller("/migration")
type MigrationController struct {}

// Permanent redirect (301) - Moved Permanently
@Redirect("https://newsite.com/", 301)
@Get("/")
func RedirectHomepage() {}

// Temporary redirect (302) - Found (default)
@Redirect("/maintenance", 302)
@Get("/admin")
func RedirectAdmin() {}

// Temporary redirect (307) - Temporary Redirect (preserves method)
@Redirect("/api/v2/orders", 307)
@Post("/v1/orders")
func RedirectCreateOrder() {}

// Permanent redirect (308) - Permanent Redirect (preserves method)
@Redirect("/api/v2/payments", 308)
@Put("/v1/payments/:id")
func RedirectUpdatePayment() {}
```

#### 🌐 Common Redirect Scenarios

##### API Versioning
```go
@Controller("/api/v1")
type APIVersionController struct {}

// Redirect old API endpoints to new versions
@Redirect("/api/v2/users", 301)
@Get("/users")
func RedirectV1Users() {}

@Redirect("/api/v2/auth/login", 301)
@Post("/auth/login")
func RedirectV1Login() {}
```

##### Domain Migration
```go
@Controller("/")
type DomainMigrationController struct {}

// Permanent domain redirects
@Redirect("https://app.newcompany.com/", 301)
@Get("/")
func RedirectHomepage() {}

@Redirect("https://app.newcompany.com/dashboard", 301)
@Get("/dashboard")
func RedirectDashboard() {}
```

##### SEO-Friendly URL Changes
```go
@Controller("/")
type SEOController struct {}

// SEO redirects for better URL structure
@Redirect("/blog", 301)
@Get("/news")
func RedirectNewsToBlog() {}

@Redirect("/products", 301)
@Get("/items")
func RedirectItemsToProducts() {}

@Redirect("/support", 301)
@Get("/help")
func RedirectHelpToSupport() {}
```

### 🔄 Generated Redirect Code

**Before (@Redirect() decorator):**
```go
func (c *UserController) RedirectUsers(ctx *httpPackage.RequestContext) {
    // Manual redirect handling needed
    ctx.Status(302)
    ctx.Header("Location", "https://newdomain.com/users")
    // Manual implementation required
}
```

**After (@Redirect("https://newdomain.com/users", 302) decorator):**
```go
func (c *UserController) RedirectUsers(ctx *httpPackage.RequestContext) {
    ctx.Redirect(302, "https://newdomain.com/users")
}
```

### 📊 HTTP Redirect Status Codes

| Status Code | Description | Use Case | Example |
|-------------|-------------|----------|---------|
| **301** | Moved Permanently | SEO-friendly permanent moves | Domain migration, URL restructuring |
| **302** | Found (Temporary) | Default temporary redirects | Maintenance mode, temporary moves |
| **307** | Temporary Redirect | Preserves HTTP method | API versioning with method preservation |
| **308** | Permanent Redirect | Preserves HTTP method | Permanent API moves with method preservation |

### 🎭 Redirect Decorator Behavior

#### Status Code Priority
```go
@Redirect("https://example.com", 301)
@HttpCode(201)  // Ignored - @Redirect takes precedence
@Get("/old-route")
func RedirectRoute() {
    // Always generates: ctx.Redirect(301, "https://example.com")
    // HttpCode decorator is ignored when Redirect is present
}
```

#### Parameter Support
```go
@Redirect("https://example.com", 302)
@Get("/redirect/:id")
func RedirectWithParams(@Param("id") id string) {
    // Parameters are still extracted before redirect
    // Generated code:
    // id := ctx.GetParam("id")
    // ctx.Redirect(302, "https://example.com")
}
```

### 🛡️ Best Practices

#### ✅ **Recommended Usage**
```go
// Permanent moves for SEO
@Redirect("/new-path", 301)
@Get("/old-path")
func PermanentMove() {}

// Temporary redirects
@Redirect("/maintenance", 302)
@Get("/admin")
func TemporaryMove() {}

// Method-preserving redirects
@Redirect("/api/v2/endpoint", 307)
@Post("/api/v1/endpoint")
func MethodPreservingRedirect() {}

// Default temporary redirect
@Redirect("/new-location")  // Defaults to 302
@Get("/old-location")
func DefaultRedirect() {}
```

### ⚡ Performance & SEO Benefits

The `@Redirect()` decorator provides:

1. **SEO Preservation**: Proper 301 redirects maintain search rankings
2. **User Experience**: Seamless redirection without broken links  
3. **Clean URLs**: Enable URL restructuring without losing traffic
4. **Method Preservation**: 307/308 status codes maintain HTTP methods
5. **Cache Control**: Browsers cache permanent redirects appropriately
6. **Automatic Generation**: No manual redirect logic needed

### 🚀 @UsePipes() Decorator Features

The new `@UsePipes()` decorator provides powerful data validation, transformation, and parsing capabilities with **NestJS-level parity**:

#### 🎯 Basic Usage
```go
// Basic validation pipe
@UsePipes(ValidationPipe)
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {
    // Request body is automatically validated before method execution
}

// Multiple pipes with transformation and validation
@UsePipes(ValidationPipe, TransformPipe)
@Put("/users/:id")
func UpdateUser(@Param("id") id string, @Body() userData UpdateUserDto) {
    // Data is validated then transformed before method execution
}
```

#### 🏗️ Controller-Level Pipes
```go
@Controller("/api/v1/data")
@UsePipes(ValidationPipe)  // Applied to ALL methods in controller
type DataController struct {
    DataService *DataService `inject:""`
}

@Post("/")
func CreateData(@Body() data CreateDataRequest) {
    // Inherits ValidationPipe from controller
}

@UsePipes(TransformPipe)  // Additional method-level pipe
@Put("/:id")
func UpdateData(@Param("id") id string, @Body() data UpdateDataRequest) {
    // Uses both ValidationPipe (controller) and TransformPipe (method)
}
```

#### 📝 Built-in Pipe Types

##### ValidationPipe - Request Validation
```go
@UsePipes(ValidationPipe)
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {
    // Generated validation logic:
    // - Validates Content-Type is application/json
    // - Validates request body structure
    // - Validates required fields and data types
    // - Returns 400 errors for validation failures
}
```

##### TransformPipe - Data Transformation
```go
@UsePipes(TransformPipe)
@Post("/data")
func ProcessData(@Body() request ProcessRequest) {
    // Generated transformation logic:
    // - Transforms request data before processing
    // - Converts string values to appropriate types
    // - Applies data normalization and formatting
}
```

##### ParseIntPipe - Integer Parsing
```go
@UsePipes(ParseIntPipe)
@Get("/items/:id")
func GetItem(@Param("id") id int, @Query("limit") limit int) {
    // Generated parsing logic:
    // - Automatically converts string path/query params to integers
    // - Returns 400 errors for invalid integer values
    // - Validates numeric constraints
}
```

##### ParseBoolPipe - Boolean Parsing
```go
@UsePipes(ParseBoolPipe)
@Get("/settings")
func GetSettings(@Query("active") active bool, @Query("verified") verified bool) {
    // Generated parsing logic:
    // - Converts string values like "true"/"false", "1"/"0" to booleans
    // - Handles case-insensitive boolean parsing
    // - Validates boolean format
}
```

##### ParseArrayPipe - Array Parsing
```go
@UsePipes(ParseArrayPipe)
@Get("/search")
func SearchItems(@Query("categories") categories []string, @Query("tags") tags []string) {
    // Generated parsing logic:
    // - Splits comma-separated strings into arrays
    // - Trims whitespace from array elements
    // - Handles empty arrays gracefully
}
```

##### DefaultValuePipe - Default Values
```go
@UsePipes(DefaultValuePipe)
@Get("/products")
func GetProducts(@Query("limit") limit int, @Query("offset") offset int, @Query("sort") sort string) {
    // Generated default value logic:
    // - Sets limit=10, offset=0, sort="created_at" when missing
    // - Provides consistent API behavior
    // - Reduces client-side complexity
}
```

#### 🎭 Complex Combined Pipes
```go
// Comprehensive pipe pipeline
@UsePipes(ValidationPipe, ParseIntPipe, DefaultValuePipe, TransformPipe)
@Post("/advanced")
func AdvancedProcessing(
    @Body() request AdvancedRequest, 
    @Query("retries") retries int,
    @Query("async") async bool,
) {
    // Execution order:
    // 1. ValidationPipe: Validates request body structure
    // 2. ParseIntPipe: Converts string "retries" to integer
    // 3. DefaultValuePipe: Sets default values for missing params
    // 4. TransformPipe: Transforms and normalizes data
    // 5. Finally executes the method with validated, parsed data
}
```

#### 🔄 Pipe Execution Order

Pipes execute in a specific order within the middleware chain:

```go
@Controller("/secure")
@UsePipes(ValidationPipe)           // Controller pipe
@UseInterceptors(LoggingInterceptor) // Controller interceptor  
@UseGuards(AuthGuard)               // Controller guard
type SecureController struct {}

@UsePipes(TransformPipe, ParseIntPipe)        // Method pipes
@UseInterceptors(CacheInterceptor)            // Method interceptor
@UseGuards(RoleGuard)                         // Method guard
@Post("/data")
func ProcessSecureData(@Body() request SecureRequest, @Query("version") version int) {
    // Middleware execution order:
    // 1. ValidationPipe (controller)
    // 2. TransformPipe, ParseIntPipe (method)
    // 3. LoggingInterceptor (controller)
    // 4. CacheInterceptor (method)
    // 5. AuthGuard (controller)
    // 6. RoleGuard (method)
    // 7. Method execution
}
```

### 🔄 Generated Pipe Code

**Before (@UsePipes() decorator):**
```go
func CreateUser(ctx *httpPackage.RequestContext) {
    // Manual validation and parsing needed
    var userData CreateUserDto
    if err := ctx.ParseJSON(&userData); err != nil {
        ctx.JSON(400, map[string]string{"error": "Invalid JSON"})
        return
    }
    
    // Manual validation
    if userData.Name == "" {
        ctx.JSON(400, map[string]string{"error": "Name required"})
        return
    }
    
    // Manual type conversion
    limitStr := ctx.GetQuery("limit")
    limit := 10 // default
    if limitStr != "" {
        if parsed, err := strconv.Atoi(limitStr); err == nil {
            limit = parsed
        }
    }
}
```

**After (@UsePipes(ValidationPipe, ParseIntPipe, DefaultValuePipe) decorator):**
```go
// Generated pipe middleware methods
func (c *UserController) ValidationPipe(ctx *httpPackage.RequestContext) {
    requestBody := ctx.GetRequestBody()
    if requestBody == nil {
        ctx.JSON(400, map[string]string{"error": "Request body is required"})
        ctx.Abort()
        return
    }
    
    contentType := ctx.GetHeader("Content-Type")
    if !strings.Contains(contentType, "application/json") {
        ctx.JSON(400, map[string]string{"error": "Content-Type must be application/json"})
        ctx.Abort()
        return
    }
    
    if err := validateRequestBody(requestBody); err != nil {
        ctx.JSON(400, map[string]string{"error": fmt.Sprintf("Validation failed: %s", err.Error())})
        ctx.Abort()
        return
    }
    
    ctx.Next()
}

func (c *UserController) ParseIntPipe(ctx *httpPackage.RequestContext) {
    queryParams := ctx.GetQueryParams()
    for key, value := range queryParams {
        if strValue, ok := value.(string); ok {
            if intValue, err := strconv.Atoi(strValue); err == nil {
                ctx.SetQueryParam(key, intValue)
            } else {
                ctx.JSON(400, map[string]string{"error": fmt.Sprintf("Invalid integer value for parameter %s: %s", key, strValue)})
                ctx.Abort()
                return
            }
        }
    }
    ctx.Next()
}

func (c *UserController) DefaultValuePipe(ctx *httpPackage.RequestContext) {
    defaultValues := map[string]interface{}{
        "limit":  10,
        "offset": 0,
        "sort":   "created_at",
    }
    
    for key, defaultValue := range defaultValues {
        if _, exists := queryParams[key]; !exists {
            ctx.SetQueryParam(key, defaultValue)
        }
    }
    ctx.Next()
}

// Route registration with pipe middleware
func (c *UserController) RegisterRoutes(server *httpPackage.HTTPServer) error {
    server.Post("/users", c.ValidationPipe, c.ParseIntPipe, c.DefaultValuePipe, c.CreateUser)
    return nil
}

// Clean method implementation
func (c *UserController) CreateUser(ctx *httpPackage.RequestContext) {
    // Data is already validated, parsed, and has defaults applied!
    var userData CreateUserDto
    ctx.ParseJSON(&userData) // Safe - already validated
    
    // Query params are already parsed to correct types
    limit := ctx.GetQuery("limit").(int)    // Already an int
    offset := ctx.GetQuery("offset").(int)  // Already an int
    
    // Method logic here...
}
```

### 📊 Pipe Validation Features

The pipe system provides comprehensive validation:

1. **Request Body Validation**: Automatic JSON structure and content validation
2. **Parameter Type Conversion**: String to int/bool/array conversion with error handling
3. **Default Value Setting**: Consistent defaults for missing parameters
4. **Content Type Validation**: Ensures proper Content-Type headers
5. **Array Processing**: Comma-separated string to array conversion
6. **Error Standardization**: Consistent 400 Bad Request responses

### 🎯 Custom Pipe Example

```go
// Using custom pipes
@UsePipes(MyCustomPipe)
@Post("/custom")
func ProcessCustomData(@Body() request CustomRequest) {
    // Custom pipe logic is generated with TODO implementation
}
```

Generated custom pipe:
```go
func (c *Controller) MyCustomPipe(ctx *httpPackage.RequestContext) {
    // MyCustomPipe pipe logic
    // TODO: Implement your custom pipe logic here
    
    // Example: Data validation
    // requestData := ctx.GetRequestData()
    // if err := validateData(requestData); err != nil {
    //     ctx.JSON(400, map[string]string{"error": err.Error()})
    //     ctx.Abort()
    //     return
    // }
    
    // Continue to next middleware/handler
    ctx.Next()
}
```

### 🏗️ Real-World Pipe Examples

#### E-commerce Product API
```go
@Controller("/api/products")
@UsePipes(ValidationPipe)  // All endpoints validate input
type ProductController struct {}

@UsePipes(ParseIntPipe, DefaultValuePipe)
@Get("/")
func GetProducts(@Query("page") page int, @Query("limit") limit int, @Query("categoryId") categoryId int) {
    // page, limit, categoryId are automatically parsed and have defaults
}

@UsePipes(TransformPipe)
@Post("/")
func CreateProduct(@Body() productData CreateProductRequest) {
    // Request validated and data transformed before method execution
}

@UsePipes(ParseArrayPipe)
@Get("/search")
func SearchProducts(@Query("tags") tags []string, @Query("categories") categories []string) {
    // Comma-separated strings automatically parsed to arrays
}
```

#### User Management with Complex Validation
```go
@Controller("/api/users")
@UsePipes(ValidationPipe)  // Global validation for all user operations
type UserController struct {}

@UsePipes(TransformPipe, ParseIntPipe, DefaultValuePipe)
@Get("/")
func GetUsers(
    @Query("page") page int,        // Parsed from string, default: 1
    @Query("limit") limit int,      // Parsed from string, default: 10
    @Query("active") active bool,   // Parsed from string
    @Query("roles") roles []string, // Parsed from comma-separated string
) {
    // All parameters properly typed and validated
}

@UsePipes(ParseBoolPipe)
@Put("/:id/status")
func UpdateUserStatus(@Param("id") id string, @Query("active") active bool, @Query("verified") verified bool) {
    // Boolean query parameters properly parsed
}
```

### ⚡ Performance & Benefits

The `@UsePipes()` decorator provides:

1. **Type Safety**: Automatic type conversion with validation
2. **Error Consistency**: Standardized 400 Bad Request responses
3. **Code Reduction**: Eliminates manual parsing and validation boilerplate
4. **Request Pipeline**: Clean separation of validation, parsing, and business logic
5. **Default Handling**: Consistent default values across endpoints
6. **Array Processing**: Automatic comma-separated string parsing
7. **Custom Logic**: Easy extension with custom pipe implementations

### 🛡️ Best Practices

#### ✅ **Recommended Usage**
```go
// Controller-level validation for consistent input handling
@Controller("/api/data")
@UsePipes(ValidationPipe)
type DataController struct {}

// Method-level parsing for specific parameter needs
@UsePipes(ParseIntPipe, DefaultValuePipe)
@Get("/items")
func GetItems(@Query("limit") limit int) {}

// Combined pipes for complex endpoints
@UsePipes(ValidationPipe, TransformPipe, ParseArrayPipe)
@Post("/complex")
func ProcessComplex(@Body() data ComplexRequest, @Query("tags") tags []string) {}
```

#### ❌ **Anti-Patterns to Avoid**
```go
// Don't: Redundant pipes
@UsePipes(ValidationPipe)
@Controller("/api")
@UsePipes(ValidationPipe)  // Duplicate - ValidationPipe already at controller
type Controller struct {}

// Don't: Wrong pipe for data type
@UsePipes(ParseIntPipe)
@Get("/text")
func GetText(@Query("message") message string) {}  // No need for ParseIntPipe

// Don't: Over-piping simple endpoints
@UsePipes(ValidationPipe, TransformPipe, ParseIntPipe, ParseBoolPipe, DefaultValuePipe)
@Get("/health")
func HealthCheck() {}  // Simple endpoint doesn't need all pipes
```

## 🔍 Detailed Example Analysis

### Controller Example (`user.controller.gofa` → `user.controller.go`)

**Before (GoFasta):**
```go
@Controller("/api/v1/users")
@UseGuards("auth")
type UserController struct {
    UserService *UserService `inject:""`
    Logger      *Logger      `inject:"logger"`
}

@Get("/:id")
@UseGuards("resource-owner")
func GetUser(@Param("id") id string) {
    // Get user by ID
}
```

**After (Generated Go):**
```go
type UserController struct {
    UserService *UserService `inject:""`
    Logger      *Logger      `inject:"logger"`
}

func (c *UserController) RegisterRoutes(server *httpPackage.HTTPServer) error {
    server.Get("/api/v1/users/:id", c.GetUser)
    return nil
}

func (c *UserController) GetUser(ctx *httpPackage.RequestContext, id string) {
    id := ctx.GetParam("id")
    // TODO: Implement method logic
    ctx.JSON(200, map[string]interface{}{"message": "Not implemented"})
}
```

### Service Example (`user.service.gofa` → `user.service.go`)

**Before (GoFasta):**
```go
@Injectable()
type UserService struct {
    UserRepository *UserRepository `inject:""`
    EmailService   *EmailService   `inject:""`
    Logger         *Logger         `inject:"logger"`
}

func GetUserById(id string) (*User, error) {
    // Implementation
}
```

**After (Generated Go):**
```go
type UserService struct {
    UserRepository *UserRepository `inject:""`
    EmailService   *EmailService   `inject:""`
    Logger         *Logger         `inject:"logger"`
}

func (s *UserService) Initialize() error {
    // TODO: Add initialization logic
    return nil
}

func (s *UserService) GetUserById(id string) *User {
    // TODO: Implement method logic
    return nil
}
```

## 🎓 Key Learning Points

1. **Declarative Syntax**: GoFasta uses decorators to declare routes and dependencies
2. **Code Generation**: The transpiler generates boilerplate Go code automatically
3. **Dependency Injection**: `inject:""` tags enable automatic DI setup
4. **Route Generation**: Method decorators become HTTP route registrations
5. **Type Safety**: Generated Go code maintains full type safety

## 🛠️ Advanced Usage

### Custom Transpilation Options

```bash
# Transpile with custom output directory
../../tools/transpiler/gofasta transpile -output ./generated

# Preserve directory structure
../../tools/transpiler/gofasta transpile -preserve=true

# Use custom number of workers
../../tools/transpiler/gofasta transpile -workers 4

# Dry run (see what would be transpiled)
../../tools/transpiler/gofasta transpile -dry-run
```

### Watch Mode for Development

```bash
# Automatically transpile when .gofa files change
../../tools/transpiler/gofasta watch -verbose
```

## 🔧 Customizing the Example

### Adding New Routes

1. Add methods to `user.controller.gofa` with appropriate decorators
2. Run transpiler: `../../tools/transpiler/gofasta transpile -force`
3. Restart the server: `go run .`

### Adding New Services

1. Create new `.gofa` file with `@Injectable()` decorator
2. Add to `app.module.gofa` providers list
3. Transpile and test

### Adding New Models

1. Add struct definitions to `types.gofa`
2. Use `@Injectable()` if they need DI
3. Transpile and use in controllers/services

## 📈 Performance

The transpiler processes files efficiently:
- **Average**: ~1ms per file
- **Parallel**: Uses multiple workers for large projects
- **Coverage**: 86.6% test coverage ensures reliability

## 🏆 Success Criteria

Your example is working correctly if you see:

✅ **Transpilation**: `🎯 Success rate: 100.0%`  
✅ **Compilation**: `go run .` starts without errors  
✅ **Server**: Responds on `http://localhost:8080`  
✅ **Routes**: All endpoints return JSON responses  
✅ **Generated Code**: Readable, idiomatic Go code  

---

**Happy coding with Gofasta! 🎉**