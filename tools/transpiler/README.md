# Gofasta Transpiler

A powerful transpiler that transforms `.gofa` files with advanced decorators into optimized Go code. This transpiler brings elegant decorator syntax and dependency injection patterns to the Go ecosystem while maintaining Go's performance characteristics.

## 🚀 Features

### Core Transpilation
- **Advanced Decorators**: `@Controller`, `@Get`, `@Post`, `@Injectable`, etc.
- **Dependency Injection**: Automatic DI with `@Injectable` and struct tags
- **Parameter Decorators**: `@Body()`, `@Param()`, `@Query()`, `@Headers()`
- **Module System**: `@Module()` with imports, exports, providers, and controllers
- **Route Generation**: Automatic HTTP route registration from decorators

### Performance & Scalability
- **Parallel Processing**: Multi-core transpilation for faster builds
- **Optimized Output**: Generated Go code optimized for compilation speed
- **AST-based**: Full Abstract Syntax Tree parsing for accurate transformations
- **Memory Efficient**: Streaming lexer and parser for large files

### Developer Experience
- **CLI Tool**: Command-line interface with watch mode
- **File Watching**: Automatic re-transpilation on file changes
- **Detailed Diagnostics**: Comprehensive error reporting with line numbers
- **Statistics**: Build performance metrics and success rates

## 📦 Installation

### As a Library
```bash
go get github.com/healtronlabs/gofasta/tools/transpiler
```

### As a CLI Tool
```bash
go install github.com/healtronlabs/gofasta/tools/transpiler/cmd/gofasta-transpiler@latest
```

## 🎯 Quick Start

### 1. Write a `.gofa` Controller

**user.controller.gofa**
```typescript
package main

@Controller("/api/v1/users")
@UseGuards("auth")
type UserController struct {
    UserService *UserService `inject:""`
    Logger      *Logger      `inject:"logger"`
}

@Get("")
func GetUsers(ctx *httpPackage.RequestContext) {
    users := c.UserService.GetAllUsers()
    ctx.JSON(200, users)
}

@Get("/:id")
@UseGuards("resource-owner")
func GetUser(@Param("id") id string, ctx *httpPackage.RequestContext) {
    user, err := c.UserService.GetUserById(id)
    if err != nil {
        ctx.JSON(404, map[string]string{"error": "User not found"})
        return
    }
    ctx.JSON(200, user)
}

@Post("")
@HttpCode(201)
func CreateUser(@Body() createUserDto CreateUserDto, ctx *httpPackage.RequestContext) {
    user, err := c.UserService.CreateUser(createUserDto)
    if err != nil {
        ctx.JSON(400, map[string]string{"error": err.Error()})
        return
    }
    ctx.JSON(201, user)
}
```

### 2. Write a `.gofa` Service

**user.service.gofa**
```typescript
package main

@Injectable()
type UserService struct {
    UserRepository *UserRepository `inject:""`
    EmailService   *EmailService   `inject:""`
    Logger         *Logger         `inject:"logger"`
}

func GetAllUsers() []User {
    return s.UserRepository.FindAll()
}

func GetUserById(id string) (*User, error) {
    return s.UserRepository.FindById(id)
}

func CreateUser(createUserDto CreateUserDto) (*User, error) {
    user := &User{
        Name:  createUserDto.Name,
        Email: createUserDto.Email,
    }
    
    savedUser, err := s.UserRepository.Save(user)
    if err != nil {
        return nil, err
    }
    
    // Send welcome email asynchronously
    go s.EmailService.SendWelcomeEmail(savedUser.Email)
    
    return savedUser, nil
}
```

### 3. Create a Module

**app.module.gofa**
```typescript
package main

@Module({
    controllers: ["UserController", "AuthController"],
    providers: ["UserService", "AuthService", "EmailService"],
    imports: ["DatabaseModule", "ConfigModule"],
    exports: ["UserService"]
})
type AppModule struct {
}
```

### 4. Transpile to Go

```bash
# Transpile all .gofa files in current directory
gofasta-transpiler transpile

# Transpile with specific input/output directories
gofasta-transpiler transpile -input ./src -output ./dist -verbose

# Watch for changes and auto-transpile
gofasta-transpiler watch -input ./src -debounce 1s
```

### 5. Generated Go Code

The transpiler generates optimized Go code:

```go
package main

import (
    "github.com/healtronlabs/gofasta/packages/core"
    "github.com/healtronlabs/gofasta/packages/http"
)

type UserController struct {
    UserService *UserService `inject:""`
    Logger      *Logger      `inject:"logger"`
}

func (c *UserController) RegisterRoutes(server *httpPackage.HTTPServer) error {
    server.Get("/api/v1/users", c.GetUsers)
    server.Get("/api/v1/users/:id", c.GetUser)
    server.Post("/api/v1/users", c.CreateUser)
    return nil
}

func (c *UserController) GetUsers(ctx *httpPackage.RequestContext) {
    users := c.UserService.GetAllUsers()
    ctx.JSON(200, users)
}

func (c *UserController) GetUser(ctx *httpPackage.RequestContext) {
    id := ctx.GetParam("id")
    user, err := c.UserService.GetUserById(id)
    if err != nil {
        ctx.JSON(404, map[string]string{"error": "User not found"})
        return
    }
    ctx.JSON(200, user)
}

func (c *UserController) CreateUser(ctx *httpPackage.RequestContext) {
    var createUserDto CreateUserDto
    if err := ctx.ParseJSON(&createUserDto); err != nil {
        ctx.JSON(400, map[string]string{"error": "Invalid request body"})
        return
    }
    
    user, err := c.UserService.CreateUser(createUserDto)
    if err != nil {
        ctx.JSON(400, map[string]string{"error": err.Error()})
        return
    }
    ctx.JSON(201, user)
}
```

## 📋 CLI Commands

### Transpile Command
```bash
gofasta-transpiler transpile [options]

Options:
  -input string        Input directory containing .gofa files (default ".")
  -output string       Output directory for .go files (default: same as input)
  -workers int         Maximum number of worker goroutines (default: CPU cores)
  -preserve            Preserve directory structure in output (default true)
  -verbose             Enable verbose output
  -file string         Transpile a single .gofa file
  -dry-run             Show what would be transpiled without actually doing it
  -force               Overwrite existing .go files
```

### Watch Command
```bash
gofasta-transpiler watch [options]

Options:
  -input string        Input directory to watch for .gofa files (default ".")
  -output string       Output directory for .go files (default: same as input)
  -workers int         Maximum number of worker goroutines
  -preserve            Preserve directory structure in output (default true)
  -debounce duration   Debounce delay for file changes (default 500ms)
  -verbose             Enable verbose output
```

## 🎨 Supported Decorators

### Class Decorators
- `@Controller(path)` - Define HTTP controllers with base path
- `@Injectable()` - Mark classes as injectable services
- `@Module(config)` - Define application modules
- `@UseGuards(guards...)` - Apply guards to entire controller
- `@UseMiddleware(middleware...)` - Apply middleware to controller

### Method Decorators
- `@Get(path)`, `@Post(path)`, `@Put(path)`, `@Delete(path)` - HTTP methods
- `@Patch(path)`, `@Options(path)`, `@Head(path)` - Additional HTTP methods
- `@HttpCode(code)` - Set response status code
- `@UseGuards(guards...)` - Apply guards to specific routes
- `@UsePipes(pipes...)` - Apply pipes for validation/transformation
- `@UseFilters(filters...)` - Apply exception filters
- `@UseInterceptors(interceptors...)` - Apply response interceptors

### Parameter Decorators
- `@Body()` - Extract request body
- `@Param(key)` - Extract route parameters
- `@Query(key)` - Extract query parameters  
- `@Headers(key)` - Extract request headers
- `@Req()` - Inject request object
- `@Res()` - Inject response object

## ⚡ Performance

The transpiler is optimized for speed and can handle large codebases:

- **Parallel Processing**: Utilizes all CPU cores for maximum throughput
- **Streaming**: Memory-efficient processing of large files
- **Incremental**: Only re-transpiles changed files in watch mode
- **Fast Compilation**: Generated Go code optimized for fast compilation

### Benchmarks
```bash
go test -bench=. -benchmem

BenchmarkLexer-8       	  500000	      2847 ns/op	     896 B/op	      12 allocs/op
BenchmarkParser-8      	   50000	     28473 ns/op	    8192 B/op	      89 allocs/op
BenchmarkTranspile-8   	   10000	    156847 ns/op	   32768 B/op	     234 allocs/op
```

## 🛠 Architecture

The transpiler follows a three-phase architecture:

1. **Lexical Analysis**: Tokenizes `.gofa` source into tokens
2. **Parsing**: Builds an Abstract Syntax Tree (AST) with decorator metadata
3. **Code Generation**: Transforms AST into optimized Go code

### Key Components

- **Lexer** (`lexer.go`): Tokenizes Gofasta syntax including decorators
- **Parser** (`parser.go`): Builds typed AST from token stream
- **AST** (`ast.go`): Defines AST node types and visitor patterns
- **CodeGen** (`codegen.go`): Generates Go code from AST
- **Parallel** (`parallel.go`): Multi-core transpilation engine
- **CLI** (`cli.go`): Command-line interface

## 🧪 Testing

Run the test suite:
```bash
go test -v
go test -bench=. -benchmem
```

Test specific functionality:
```bash
go test -run TestLexer -v
go test -run TestParser -v
go test -run TestCodeGeneration -v
```

## 🔧 Integration

### With Build Systems

**Makefile**
```makefile
.PHONY: transpile build

transpile:
	gofasta-transpiler transpile -input ./src -output ./generated

build: transpile
	go build -o app ./generated/...

watch:
	gofasta-transpiler watch -input ./src -output ./generated -verbose
```

**Docker**
```dockerfile
FROM golang:1.21-alpine AS transpiler
RUN go install github.com/healtronlabs/gofasta/tools/transpiler/cmd/gofasta-transpiler@latest
COPY src/ /app/src/
WORKDIR /app
RUN gofasta-transpiler transpile -input src -output generated

FROM golang:1.21-alpine AS builder
COPY --from=transpiler /app/generated /app/
WORKDIR /app
RUN go build -o main .

FROM alpine:latest
COPY --from=builder /app/main /usr/local/bin/
ENTRYPOINT ["main"]
```

### With IDEs

The transpiler works with any editor/IDE:

- **VS Code**: Use tasks.json for build automation
- **GoLand**: Configure external tools for transpilation
- **Vim/Neovim**: Set up autocmd for .gofa files

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built on Go's excellent parsing and AST libraries
- Thanks to the Gofasta community for feedback and contributions
