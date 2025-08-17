# Quick Start Guide

Get up and running with Gofasta in under 5 minutes.

## Prerequisites

- **Go 1.22+** - [Install Go](https://golang.org/dl/)
- **Git** - For package management
- **Database** (optional) - PostgreSQL, MongoDB, MySQL, or SQLite

## Step 1: Install Gofasta CLI

```bash
go install github.com/healtronlabs/gofasta/packages/cli@latest
```

Verify installation:
```bash
gofasta version
```

## Step 2: Create Your First Project

```bash
# Create a new API project
gofasta new my-first-api

# Navigate to project directory
cd my-first-api

# Install dependencies
go mod tidy
```

## Step 3: Explore the Generated Project

```
my-first-api/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── controllers/         # HTTP controllers
│   ├── services/           # Business logic
│   ├── models/             # Data models
│   ├── repositories/       # Data access layer
│   └── config/             # Configuration
├── migrations/             # Database migrations
├── .env.example           # Environment variables template
├── go.mod                 # Go module file
└── README.md              # Project documentation
```

## Step 4: Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit configuration (optional)
# The default SQLite database works out of the box
```

## Step 5: Start Development Server

```bash
# Start with hot reload
gofasta dev
```

Your API is now running at `http://localhost:8080`!

## Step 6: Test Your API

```bash
# Health check
curl http://localhost:8080/health

# Get users (requires authentication)
curl -H "Authorization: Bearer your-jwt-token" http://localhost:8080/users
```

## Understanding the Architecture

### Application Structure

```go
// cmd/main.go - Application entry point
func main() {
    // Load configuration
    cfg := config.Load()

    // Create application with dependency injection
    app := core.CreateApp(&AppModule{})

    // Register framework modules
    app.RegisterModule(orm.NewGofastaOrmModuleFromURL(cfg.DatabaseURL))
    app.RegisterModule(http.NewHTTPModule(cfg.Port))
    app.RegisterModule(auth.NewAuthModule(cfg.JWTSecret))

    // Start server
    app.Listen(cfg.Port)
}

// AppModule configures your application
type AppModule struct {
    *core.BaseModule
}

func (m *AppModule) Configure() {
    // Dependency injection automatically handles these
    m.AddController(&controllers.UserController{})
    m.AddProvider(&services.UserService{})
    m.AddProvider(&repositories.UserRepository{})
}
```

### Unified Database API

The same code works with any database:

```go
type UserService struct {
    UserRepo orm.Repository[models.User] `inject:""`
}

func (s *UserService) FindActiveUsers() ([]*models.User, error) {
    return s.UserRepo.Query().
        Where("status", orm.OpEquals, "active").
        OrderBy("created_at", orm.DirectionDesc).
        Limit(10).
        Execute()
}
```

### Universal Model Definition

```go
type User struct {
    // Works with SQL and NoSQL databases
    ID        string    `gorm:"primaryKey" bson:"_id,omitempty" json:"id"`
    Email     string    `gorm:"uniqueIndex" bson:"email" json:"email" validate:"required,email"`
    FirstName string    `gorm:"not null" bson:"firstName" json:"firstName" validate:"required"`
    Status    string    `gorm:"type:varchar(20)" bson:"status" json:"status"`
    CreatedAt time.Time `gorm:"autoCreateTime" bson:"createdAt" json:"createdAt"`
}
```

## Next Steps

### 1. Generate Additional Components

```bash
# Generate a new model
gofasta generate model Product --validation

# Generate service and controller
gofasta generate service Product
gofasta generate controller Product --crud

# Generate repository
gofasta generate repository Product --model Product
```

### 2. Database Migrations

```bash
# Create a migration
gofasta migration create create_products_table

# Edit the migration files in migrations/
# Then run migrations
gofasta migration run
```

### 3. Add Authentication

The generated project includes JWT authentication. Create a user and get a token:

```bash
# Create a user (implement this endpoint)
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'

# Login to get token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

### 4. Switch Databases

Change database by updating your `.env` file:

```bash
# PostgreSQL
DATABASE_URL=postgresql://localhost:5432/myapp

# MongoDB
DATABASE_URL=mongodb://localhost:27017/myapp

# MySQL
DATABASE_URL=mysql://user:password@localhost:3306/myapp

# SQLite (default)
DATABASE_URL=sqlite://./myapp.db
```

No code changes required - the same application works with any database!

### 5. Build for Production

```bash
# Build optimized binary
gofasta build --optimize

# Cross-compile for Linux
gofasta build --platform linux --arch amd64
```

## Common Patterns

### Adding a New Feature

1. **Create Model**:
   ```bash
   gofasta generate model Article --validation
   ```

2. **Create Service**:
   ```bash
   gofasta generate service Article
   ```

3. **Create Controller**:
   ```bash
   gofasta generate controller Article --crud
   ```

4. **Create Migration**:
   ```bash
   gofasta migration create create_articles_table
   gofasta migration run
   ```

### Dependency Injection

Gofasta automatically resolves dependencies using struct tags:

```go
type ArticleController struct {
    ArticleService *services.ArticleService `inject:""`
    AuthService    *services.AuthService    `inject:""`
}

type ArticleService struct {
    ArticleRepo orm.Repository[models.Article] `inject:""`
    UserRepo    orm.Repository[models.User]    `inject:""`
}
```

### Middleware and Guards

```go
// Add authentication guard to routes
func (c *ArticleController) Routes() []core.Route {
    return []core.Route{
        {
            Method:  "GET",
            Path:    "/articles",
            Handler: c.GetArticles(),
            Guards:  []core.Guard{&auth.JWTGuard{}},
        },
        {
            Method:  "POST",
            Path:    "/articles",
            Handler: c.CreateArticle(),
            Guards:  []core.Guard{&auth.JWTGuard{}, &auth.RoleGuard{Role: "admin"}},
        },
    }
}
```

### Validation

```go
type CreateArticleDTO struct {
    Title   string `json:"title" validate:"required,min=3,max=100"`
    Content string `json:"content" validate:"required,min=10"`
    Tags    []string `json:"tags" validate:"max=5,dive,min=2,max=20"`
}

func (c *ArticleController) CreateArticle() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var dto CreateArticleDTO
        if err := core.ParseAndValidateJSON(r, &dto); err != nil {
            core.WriteErrorResponse(w, err)
            return
        }
        
        // Process validated data...
    }
}
```

## Learning Resources

### Documentation
- [Installation Guide](installation.md) - Detailed installation instructions
- [API Reference](api/) - Complete API documentation
- [Best Practices](best-practices.md) - Recommended patterns

### Examples
- [Basic API](../examples/basic-api/) - Simple REST API
- [Multi-Database](../examples/multi-database/) - Database switching demo
- [E-commerce API](../examples/e-commerce/) - Complex business logic
- [Microservices](../examples/microservices/) - Service architecture

### Community
- [GitHub Discussions](https://github.com/healtronlabs/gofasta/discussions)
- [Discord Community](https://discord.gg/gofasta)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/gofasta)

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
# Change port in .env file or command line
gofasta dev --port 3000
```

**Database connection failed:**
```bash
# Check database is running and accessible
# Verify DATABASE_URL in .env file
gofasta migration status
```

**Module not found:**
```bash
# Ensure dependencies are installed
go mod tidy
go mod download
```

### Getting Help

1. Check the [troubleshooting guide](troubleshooting.md)
2. Search [existing issues](https://github.com/healtronlabs/gofasta/issues)
3. Ask on [Discord](https://discord.gg/gofasta)
4. Create a [new issue](https://github.com/healtronlabs/gofasta/issues/new)

## What's Next?

Now that you have Gofasta running:

1. **Explore the Framework**: Read the [architecture overview](architecture.md)
2. **Build Something Real**: Try the [tutorial series](tutorials/)
3. **Deploy to Production**: Follow the [deployment guide](deployment.md)
4. **Join the Community**: Contribute to [open source](contributing.md)

Welcome to the Gofasta community! 🚀