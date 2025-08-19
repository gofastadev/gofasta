# Gofasta Transpiler Example

This example demonstrates how to use the Gofasta transpiler to transform decorator-based `.gofa` files into standard Go code.

## 🎯 What This Example Shows

- **Controller Declaration**: Using `@Controller` with route prefixes
- **Route Methods**: `@Get`, `@Post`, `@Put`, `@Delete` decorators
- **Dependency Injection**: Using `inject:""` tags
- **Parameter Decorators**: `@Param`, `@Body`, `@Query`, `@Headers` (with advanced features) for request handling
- **Service Pattern**: `@Injectable` services with DI
- **Module System**: `@Module` for organizing dependencies
- **Guards & Middleware**: `@UseGuards`, `@HttpCode` decorators
- **Error Handling**: `@Catch()` decorators for automatic error filtering

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
│   ├── user.service.gofa    # User business logic service  
│   ├── types.gofa           # Data models and DTOs
│   ├── simple-test.gofa     # Simple test controller
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