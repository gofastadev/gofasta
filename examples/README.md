# Gofasta Framework Examples

This directory contains comprehensive examples demonstrating different features and approaches of the Gofasta framework.

## 📁 Available Examples

### 🏗️ **Core Example** (`core-example/`)
**Purpose**: Demonstrates core framework features and the new decorator system

**Features**:
- Dependency injection container
- Service lifecycle management
- Programmatic decorator registration
- Advanced validation system
- Module configuration

**Run**:
```bash
go run examples/core-example/main.go
```

### 🌐 **HTTP Examples** (`http-example/`)
**Purpose**: Multiple approaches to HTTP server development in a single directory

The `http-example` directory contains **three different approaches** using build tags:

#### 1. **Basic Approach** (Learning)
**Run**:
```bash
go run -tags=basic examples/http-example/.
```
**Features**:
- Simple controller setup
- Basic comment decorators
- Good for learning Gofasta basics
- WebSocket and static file serving

#### 2. **Comment Decorators** (Advanced)
**Run**:
```bash
go run -tags=comments examples/http-example/.
```
**Features**:
- Advanced comment-based annotations (`@Controller("/path")`)
- Complex routing with nested resources
- Comprehensive middleware, guards, and pipes
- API versioning and role-based access

#### 3. **Programmatic Decorators** (Recommended)
**Run**:
```bash
go run -tags=programmatic examples/http-example/.
```
**Features**:
- **Fluent API** for decorator registration
- **Type-safe** controller and route configuration
- **Production-ready** approach
- Real-time route metadata extraction

## 🚀 Quick Start

### Test All Examples
```bash
# Core framework features
go run examples/core-example/main.go

# HTTP Examples (choose one approach):
go run -tags=basic examples/http-example/.        # Basic approach
go run -tags=comments examples/http-example/.     # Comment decorators  
go run -tags=programmatic examples/http-example/. # Programmatic (recommended)

# Or see all available options:
go run examples/http-example/main.go
```

### Test API Endpoints

**Basic & Comments Examples** (Port 8080):
```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/users
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

**Programmatic Example** (Port 8081):
```bash
curl http://localhost:8081/health
curl http://localhost:8081/api/v1/users
curl -X POST http://localhost:8081/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Bob","email":"bob@example.com"}'
```

## 📊 Comparison

| Approach | Build Tag | Decorator Type | Complexity | Use Case | Port |
|----------|-----------|---------------|------------|----------|------|
| **Basic** | `basic` | Comment-based | Low | Learning basics | 8080 |
| **Comments** | `comments` | Comment-based | Medium | Complex routing | 8080 |
| **Programmatic** | `programmatic` | Programmatic | Medium | Production apps | 8081 |
| **Core** | N/A | Programmatic | Low | Framework internals | N/A |

## 🎯 Recommended Learning Path

1. **Start with Core Example** - Understand dependency injection and framework basics
2. **Basic HTTP Example** - Learn HTTP server setup and basic decorators
3. **Modern HTTP Example** - Experience the powerful programmatic decorator API
4. **Comment Decorators Example** - See alternative decorator approaches

## 🔧 Features Demonstrated

### ✅ Core Framework
- Dependency injection container
- Service lifecycle management  
- Module system
- Application configuration
- Scoped services

### ✅ HTTP Server
- RESTful API endpoints
- WebSocket support
- Static file serving
- Request/response handling
- Error handling

### ✅ Decorator Systems
- **Comment-based**: `@Controller`, `@Get`, `@Post`
- **Programmatic**: Fluent API with type safety
- Middleware composition
- Guard implementation
- Pipe validation
- Status code configuration

### ✅ Advanced Features
- Real-time route metadata extraction
- Advanced validation with error messages
- Controller and route registration
- Backward compatibility
- Type-safe configuration

## 📝 Notes

- All examples use the same core Gofasta framework
- Examples are self-contained and can run independently
- Modern HTTP example demonstrates the recommended approach for production
- Comment decorators require future AST parsing improvements for full functionality
- All examples maintain backward compatibility with existing code