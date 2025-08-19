# Gofasta HTTP Examples

This directory demonstrates three different approaches to building HTTP servers with the Gofasta framework. Each approach is independent and showcases different decorator patterns.

## 📁 File Structure

```
http-example/
├── main.go                      # Example selector (shows available options)
├── approach-basic.go           # Basic HTTP server with simple decorators
├── approach-comments.go        # Comment-based decorator system
├── approach-programmatic.go    # Modern programmatic decorators (recommended)
├── static/                     # Static files for web demo
│   ├── index.html
│   └── style.css
└── README.md                   # This file
```

## 🚀 Running Examples

### Option 1: Basic Approach (Learning)
```bash
go run -tags=basic examples/http-example/.
```
**Features:**
- Simple controller setup
- Basic comment decorators
- Good for learning Gofasta basics
- WebSocket and static file serving

### Option 2: Comment Decorators (Advanced)
```bash
go run -tags=comments examples/http-example/.
```
**Features:**
- Advanced comment-based annotations (`@Controller("/path")`)
- Complex routing with nested resources
- Comprehensive middleware, guards, and pipes
- API versioning and role-based access
- Multiple controller types

### Option 3: Programmatic Approach (Recommended)
```bash
go run -tags=programmatic examples/http-example/.
```
**Features:**
- **Modern fluent API** for decorator registration
- **Type-safe** controller and route configuration
- **Production-ready** approach
- Real-time route metadata extraction
- Advanced validation system

## 📊 Comparison

| Approach | Build Tag | Complexity | Best For | Port |
|----------|-----------|------------|----------|------|
| **Basic** | `basic` | Low | Learning basics | 8080 |
| **Comments** | `comments` | Medium | Complex routing | 8080 |
| **Programmatic** | `programmatic` | Medium | Production apps | 8081 |

## 🔧 Testing Endpoints

### Basic & Comments Examples (Port 8080)
```bash
# Health check
curl http://localhost:8080/health

# Get users
curl http://localhost:8080/api/v1/users

# Create user
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com"}'

# Get specific user
curl http://localhost:8080/api/v1/users/123

# Static demo page
open http://localhost:8080/static/index.html
```

### Programmatic Example (Port 8081)
```bash
# Health endpoints
curl http://localhost:8081/health
curl http://localhost:8081/health/detailed

# User management
curl http://localhost:8081/api/v1/users
curl -X POST http://localhost:8081/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Bob","email":"bob@example.com"}'

# WebSocket chat
# Connect to ws://localhost:8081/ws

# Static files
open http://localhost:8081/static/index.html
```

## 🎯 Key Differences

### Basic Approach
- **Simple setup** with minimal configuration
- **Comment decorators** in method signatures
- **Convention-based routing**
- Perfect for **getting started**

```go
// @Controller("/api/v1/users")
type UserController struct {
    UserService *UserService `inject:""`
}

// @Get("")
func (c *UserController) GetUsers(ctx *RequestContext) {
    // Implementation
}
```

### Comment Decorators Approach  
- **Advanced comment annotations**
- **Complex middleware composition**
- **Nested resource routing**
- **API versioning support**

```go
// @Controller("/api/v1/users")
// @UseMiddleware("auth", "logging")
// @UseGuards("authenticated")
type UsersController struct {
    UserService *UserService `inject:""`
}

// @Get("/:id/posts")
// @UseMiddleware("cache")
func (c *UsersController) GetUserPosts(ctx *RequestContext) {
    // Generates: GET /api/v1/users/:id/posts
}
```

### Programmatic Approach (Recommended)
- **Type-safe registration**
- **Fluent API design**
- **Runtime route metadata**
- **Production-ready architecture**

```go
// Register decorators programmatically
decorators.Controller("/api/v1/users").
    UseMiddleware("cors", "logging").
    UseGuards("authenticated").
    Register(&UserController{}).
    Get("GetUsers", "").
    UseMiddleware("cache").
    Register().
    Post("CreateUser", "").
    HttpCode(201).
    Register()
```

## 🛡️ Security Features

All approaches demonstrate:
- **CORS middleware** configuration
- **Authentication guards**
- **Input validation**
- **Error handling**
- **Request logging**

## 🔌 WebSocket Support

Each approach includes WebSocket functionality:
- **Real-time chat** implementation
- **Connection management**  
- **Message broadcasting**
- **Graceful disconnection**

## 📱 Interactive Demo

All approaches serve a static demo page at `/static/index.html` featuring:
- **API testing interface**
- **WebSocket chat functionality**
- **Real-time user management**
- **Responsive design**

## 🚀 Recommended Learning Path

1. **Start with Basic** - Understand core concepts
2. **Explore Comments** - See advanced routing patterns  
3. **Use Programmatic** - Adopt for production projects

## 💡 Production Tips

- **Use Programmatic approach** for production applications
- **Enable CORS** for frontend integration
- **Implement proper authentication** guards
- **Add input validation** middleware
- **Configure proper logging**
- **Use environment-specific configuration**

## 🐛 Troubleshooting

### Build Tag Issues
```bash
# If you get "no Go files" error, make sure to use build tags:
go run -tags=basic examples/http-example/.

# Or build first:
go build -tags=basic examples/http-example/.
```

### Port Conflicts
- Basic & Comments use port **8080**
- Programmatic uses port **8081**
- Make sure ports are available before running

### Import Issues
All examples use the same Gofasta packages:
- `github.com/healtronlabs/gofasta/packages/core`
- `github.com/healtronlabs/gofasta/packages/http`
- `github.com/healtronlabs/gofasta/packages/core/decorators`