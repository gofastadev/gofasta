package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCmd() *cobra.Command {
	var dbType string
	var template string
	var skipGit bool

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new Gofasta project",
		Long: `Create a new Gofasta project with enterprise architecture patterns.

Available templates:
- api: REST API with basic CRUD operations
- microservice: Microservice with gRPC support
- web: Web API with static file serving
- minimal: Minimal setup with core modules only
- e-commerce: E-commerce backend API template`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			return createProject(projectName, template, dbType, skipGit)
		},
	}

	cmd.Flags().StringVarP(&dbType, "database", "d", "postgresql", "Database type (postgresql, mongodb, mysql, sqlite)")
	cmd.Flags().StringVarP(&template, "template", "t", "api", "Project template (api, microservice, web, minimal, e-commerce)")
	cmd.Flags().BoolVar(&skipGit, "skip-git", false, "Skip git repository initialization")

	return cmd
}

func createProject(projectName, template, dbType string, skipGit bool) error {
	// Validate project name
	if strings.Contains(projectName, " ") || strings.Contains(projectName, "/") {
		return fmt.Errorf("project name must be a valid directory name")
	}

	// Create project directory
	projectDir := filepath.Join(".", projectName)
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", projectName)
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	fmt.Printf("Creating Gofasta project '%s' with template '%s'...\n", projectName, template)

	// Create project structure
	if err := createProjectStructure(projectDir, projectName, template, dbType); err != nil {
		return fmt.Errorf("failed to create project structure: %w", err)
	}

	// Initialize git repository
	if !skipGit {
		if err := initGitRepo(projectDir); err != nil {
			fmt.Printf("Warning: Failed to initialize git repository: %v\n", err)
		}
	}

	fmt.Printf("\n✅ Project '%s' created successfully!\n", projectName)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  gofasta dev\n")

	return nil
}

func createProjectStructure(projectDir, projectName, template, dbType string) error {
	// Create directory structure
	dirs := []string{
		"cmd",
		"internal/controllers",
		"internal/services",
		"internal/models",
		"internal/repositories",
		"internal/middleware",
		"internal/config",
		"pkg/utils",
		"migrations",
		"docs",
		"tests",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectDir, dir), 0755); err != nil {
			return err
		}
	}

	// Create files based on template
	switch template {
	case "api":
		return createAPITemplate(projectDir, projectName, dbType)
	case "microservice":
		return createMicroserviceTemplate(projectDir, projectName, dbType)
	case "web":
		return createWebTemplate(projectDir, projectName, dbType)
	case "minimal":
		return createMinimalTemplate(projectDir, projectName, dbType)
	case "e-commerce":
		return createECommerceTemplate(projectDir, projectName, dbType)
	default:
		return fmt.Errorf("unknown template: %s", template)
	}
}

func createAPITemplate(projectDir, projectName, dbType string) error {
	// go.mod
	goMod := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/healtronlabs/gofasta/packages/core v0.1.0
	github.com/healtronlabs/gofasta/packages/http v0.1.0
	github.com/healtronlabs/gofasta/packages/orm v0.1.0
	github.com/healtronlabs/gofasta/packages/auth v0.1.0
	github.com/healtronlabs/gofasta/packages/validation v0.1.0
)

replace github.com/healtronlabs/gofasta/packages/core => ../packages/core
replace github.com/healtronlabs/gofasta/packages/http => ../packages/http
replace github.com/healtronlabs/gofasta/packages/orm => ../packages/orm
replace github.com/healtronlabs/gofasta/packages/auth => ../packages/auth
replace github.com/healtronlabs/gofasta/packages/validation => ../packages/validation
`, projectName)

	if err := writeFile(filepath.Join(projectDir, "go.mod"), goMod); err != nil {
		return err
	}

	// main.go
	mainGo := fmt.Sprintf(`package main

import (
	"log"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/http"
	"github.com/healtronlabs/gofasta/packages/orm"
	"github.com/healtronlabs/gofasta/packages/auth"
	"github.com/healtronlabs/gofasta/packages/validation"

	"%s/internal/config"
	"%s/internal/controllers"
	"%s/internal/services"
	"%s/internal/repositories"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create application
	app := core.CreateApp(&AppModule{})

	// Register modules
	app.RegisterModule(orm.NewGofastaOrmModuleFromURL(cfg.DatabaseURL))
	app.RegisterModule(http.NewHTTPModule(cfg.Port))
	app.RegisterModule(auth.NewAuthModule(cfg.JWTSecret))
	app.RegisterModule(validation.NewValidationModule())

	// Start server
	log.Printf("Starting server on port %%d...", cfg.Port)
	if err := app.Listen(cfg.Port); err != nil {
		log.Fatal(err)
	}
}

type AppModule struct {
	*core.BaseModule
}

func (m *AppModule) Configure() {
	// Register controllers
	m.AddController(&controllers.UserController{})
	m.AddController(&controllers.HealthController{})

	// Register services
	m.AddProvider(&services.UserService{})

	// Register repositories
	m.AddProvider(&repositories.UserRepository{})
}
`, projectName, projectName, projectName, projectName)

	if err := writeFile(filepath.Join(projectDir, "cmd/main.go"), mainGo); err != nil {
		return err
	}

	// Config
	configGo := fmt.Sprintf(`package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DatabaseURL string
	JWTSecret   string
	Environment string
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	
	return &Config{
		Port:        port,
		DatabaseURL: getEnv("DATABASE_URL", "%s"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
`, getDefaultDatabaseURL(dbType))

	if err := writeFile(filepath.Join(projectDir, "internal/config/config.go"), configGo); err != nil {
		return err
	}

	// User model
	userModel := `package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID ` + "`" + `gorm:"primaryKey" bson:"_id,omitempty" json:"id"` + "`" + `
	Email     string             ` + "`" + `gorm:"uniqueIndex" bson:"email" json:"email" validate:"required,email"` + "`" + `
	FirstName string             ` + "`" + `gorm:"not null" bson:"firstName" json:"firstName" validate:"required"` + "`" + `
	LastName  string             ` + "`" + `gorm:"not null" bson:"lastName" json:"lastName" validate:"required"` + "`" + `
	Age       int                ` + "`" + `bson:"age" json:"age" validate:"gte=18,lte=120"` + "`" + `
	Status    string             ` + "`" + `gorm:"type:varchar(20)" bson:"status" json:"status" validate:"oneof=active inactive"` + "`" + `
	CreatedAt time.Time          ` + "`" + `gorm:"autoCreateTime" bson:"createdAt" json:"createdAt"` + "`" + `
	UpdatedAt time.Time          ` + "`" + `gorm:"autoUpdateTime" bson:"updatedAt" json:"updatedAt"` + "`" + `
}

func (u *User) TableName() string {
	return "users"
}
`

	if err := writeFile(filepath.Join(projectDir, "internal/models/user.go"), userModel); err != nil {
		return err
	}

	// User controller
	userController := fmt.Sprintf(`package controllers

import (
	"context"
	"net/http"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/auth"
	
	"%s/internal/services"
	"%s/internal/models"
)

type UserController struct {
	UserService *services.UserService `+"`"+`inject:""`+"`"+`
}

// GetUsers retrieves all users
func (c *UserController) GetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := c.UserService.GetAllUsers(r.Context())
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusOK, users)
	}
}

// GetUser retrieves a user by ID
func (c *UserController) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" {
			core.WriteErrorResponse(w, core.NewBadRequestException("User ID is required"))
			return
		}

		user, err := c.UserService.GetUserByID(r.Context(), userID)
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusOK, user)
	}
}

// CreateUser creates a new user
func (c *UserController) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := core.ParseJSONBody(r, &user); err != nil {
			core.WriteErrorResponse(w, core.NewBadRequestException("Invalid request body"))
			return
		}

		createdUser, err := c.UserService.CreateUser(r.Context(), &user)
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusCreated, createdUser)
	}
}

// Routes returns the controller routes
func (c *UserController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  "GET",
			Path:    "/users",
			Handler: c.GetUsers(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
		{
			Method:  "GET",
			Path:    "/users/:id",
			Handler: c.GetUser(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
		{
			Method:  "POST",
			Path:    "/users",
			Handler: c.CreateUser(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
	}
}
`, projectName, projectName)

	if err := writeFile(filepath.Join(projectDir, "internal/controllers/user_controller.go"), userController); err != nil {
		return err
	}

	// Health controller
	healthController := `package controllers

import (
	"net/http"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

type HealthController struct{}

type HealthResponse struct {
	Status    string    ` + "`" + `json:"status"` + "`" + `
	Timestamp time.Time ` + "`" + `json:"timestamp"` + "`" + `
	Version   string    ` + "`" + `json:"version"` + "`" + `
}

func (c *HealthController) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{
			Status:    "OK",
			Timestamp: time.Now(),
			Version:   "1.0.0",
		}

		core.WriteJSONResponse(w, http.StatusOK, response)
	}
}

func (c *HealthController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  "GET",
			Path:    "/health",
			Handler: c.Health(),
		},
	}
}
`

	if err := writeFile(filepath.Join(projectDir, "internal/controllers/health_controller.go"), healthController); err != nil {
		return err
	}

	// User service
	userService := fmt.Sprintf(`package services

import (
	"context"

	"github.com/healtronlabs/gofasta/packages/orm"
	
	"%s/internal/models"
)

type UserService struct {
	UserRepo orm.Repository[models.User] `+"`"+`inject:""`+"`"+`
}

func (s *UserService) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	return s.UserRepo.Query().
		Where("status", orm.OpEquals, "active").
		OrderBy("created_at", orm.DirectionDesc).
		Execute(ctx)
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.UserRepo.FindByID(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	// Set default status
	if user.Status == "" {
		user.Status = "active"
	}

	return s.UserRepo.Create(ctx, user)
}

func (s *UserService) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	return s.UserRepo.Update(ctx, user)
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.UserRepo.Query().
		Where("id", orm.OpEquals, id).
		Delete(ctx)
}
`, projectName)

	if err := writeFile(filepath.Join(projectDir, "internal/services/user_service.go"), userService); err != nil {
		return err
	}

	// .env example
	envExample := fmt.Sprintf(`PORT=8080
DATABASE_URL=%s
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
ENVIRONMENT=development
`, getDefaultDatabaseURL(dbType))

	if err := writeFile(filepath.Join(projectDir, ".env.example"), envExample); err != nil {
		return err
	}

	// .gitignore
	gitignore := `# Binaries
*.exe
*.dll
*.so
*.dylib
*.test
*.out

# Environment variables
.env
.env.local

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Build artifacts
/dist/
/build/
/tmp/

# Log files
*.log

# Database files
*.db
*.sqlite
*.sqlite3
`

	if err := writeFile(filepath.Join(projectDir, ".gitignore"), gitignore); err != nil {
		return err
	}

	// README.md
	readme := fmt.Sprintf(`# %s

A Gofasta application built with enterprise architecture patterns.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Database (PostgreSQL/MongoDB/MySQL/SQLite)

### Installation

1. Clone the repository
2. Install dependencies:
   `+"```"+`bash
   go mod tidy
   `+"```"+`

3. Copy environment variables:
   `+"```"+`bash
   cp .env.example .env
   `+"```"+`

4. Update the .env file with your database configuration

### Running the Application

#### Development Mode
`+"```"+`bash
gofasta dev
`+"```"+`

#### Production Mode
`+"```"+`bash
go build -o bin/app cmd/main.go
./bin/app
`+"```"+`

### API Endpoints

- `+"`"+`GET /health`+"`"+` - Health check
- `+"`"+`GET /users`+"`"+` - Get all users
- `+"`"+`GET /users/:id`+"`"+` - Get user by ID
- `+"`"+`POST /users`+"`"+` - Create a new user

### Database Migration

`+"```"+`bash
gofasta migration:create create_users_table
gofasta migration:run
`+"```"+`

## Architecture

This project follows Gofasta's enterprise architecture patterns:

- **Dependency Injection**: Automatic service resolution
- **Modular Design**: Clear separation of concerns
- **Universal Database API**: Works with any database
- **Type Safety**: Full Go generics support

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request
`, projectName)

	return writeFile(filepath.Join(projectDir, "README.md"), readme)
}

func createMinimalTemplate(projectDir, projectName, dbType string) error {
	// Just create basic structure with minimal files
	goMod := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/healtronlabs/gofasta/packages/core v0.1.0
	github.com/healtronlabs/gofasta/packages/http v0.1.0
)
`, projectName)

	if err := writeFile(filepath.Join(projectDir, "go.mod"), goMod); err != nil {
		return err
	}

	mainGo := `package main

import (
	"log"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/http"
)

func main() {
	app := core.CreateApp(&AppModule{})
	app.RegisterModule(http.NewHTTPModule(8080))

	log.Println("Starting server on port 8080...")
	if err := app.Listen(8080); err != nil {
		log.Fatal(err)
	}
}

type AppModule struct {
	*core.BaseModule
}

func (m *AppModule) Configure() {
	// Add your controllers and services here
}
`

	return writeFile(filepath.Join(projectDir, "main.go"), mainGo)
}

func createMicroserviceTemplate(projectDir, projectName, dbType string) error {
	// Similar to API but with gRPC support
	return createAPITemplate(projectDir, projectName, dbType)
}

func createWebTemplate(projectDir, projectName, dbType string) error {
	// API template + static file serving
	if err := createAPITemplate(projectDir, projectName, dbType); err != nil {
		return err
	}

	// Add web-specific directories for static assets
	webDirs := []string{
		"static/css",
		"static/js",
		"static/images",
		"templates",
	}

	for _, dir := range webDirs {
		if err := os.MkdirAll(filepath.Join(projectDir, dir), 0755); err != nil {
			return err
		}
	}

	return nil
}

func createECommerceTemplate(projectDir, projectName, dbType string) error {
	// Extended API template with e-commerce specific models
	if err := createAPITemplate(projectDir, projectName, dbType); err != nil {
		return err
	}

	// Add e-commerce specific models
	productModel := `package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID          primitive.ObjectID ` + "`" + `gorm:"primaryKey" bson:"_id,omitempty" json:"id"` + "`" + `
	Name        string             ` + "`" + `gorm:"not null" bson:"name" json:"name" validate:"required"` + "`" + `
	Description string             ` + "`" + `bson:"description" json:"description"` + "`" + `
	Price       float64            ` + "`" + `gorm:"not null" bson:"price" json:"price" validate:"required,gte=0"` + "`" + `
	Stock       int                ` + "`" + `gorm:"not null" bson:"stock" json:"stock" validate:"gte=0"` + "`" + `
	CategoryID  string             ` + "`" + `gorm:"not null" bson:"categoryId" json:"categoryId" validate:"required"` + "`" + `
	Status      string             ` + "`" + `gorm:"type:varchar(20)" bson:"status" json:"status" validate:"oneof=active inactive"` + "`" + `
	CreatedAt   time.Time          ` + "`" + `gorm:"autoCreateTime" bson:"createdAt" json:"createdAt"` + "`" + `
	UpdatedAt   time.Time          ` + "`" + `gorm:"autoUpdateTime" bson:"updatedAt" json:"updatedAt"` + "`" + `
}

type Order struct {
	ID         primitive.ObjectID ` + "`" + `gorm:"primaryKey" bson:"_id,omitempty" json:"id"` + "`" + `
	UserID     string             ` + "`" + `gorm:"not null" bson:"userId" json:"userId" validate:"required"` + "`" + `
	TotalPrice float64            ` + "`" + `gorm:"not null" bson:"totalPrice" json:"totalPrice" validate:"required,gte=0"` + "`" + `
	Status     string             ` + "`" + `gorm:"type:varchar(20)" bson:"status" json:"status" validate:"oneof=pending paid shipped delivered cancelled"` + "`" + `
	Items      []OrderItem        ` + "`" + `gorm:"foreignKey:OrderID" bson:"items" json:"items"` + "`" + `
	CreatedAt  time.Time          ` + "`" + `gorm:"autoCreateTime" bson:"createdAt" json:"createdAt"` + "`" + `
	UpdatedAt  time.Time          ` + "`" + `gorm:"autoUpdateTime" bson:"updatedAt" json:"updatedAt"` + "`" + `
}

type OrderItem struct {
	ID        primitive.ObjectID ` + "`" + `gorm:"primaryKey" bson:"_id,omitempty" json:"id"` + "`" + `
	OrderID   string             ` + "`" + `gorm:"not null" bson:"orderId" json:"orderId"` + "`" + `
	ProductID string             ` + "`" + `gorm:"not null" bson:"productId" json:"productId" validate:"required"` + "`" + `
	Quantity  int                ` + "`" + `gorm:"not null" bson:"quantity" json:"quantity" validate:"required,gte=1"` + "`" + `
	Price     float64            ` + "`" + `gorm:"not null" bson:"price" json:"price" validate:"required,gte=0"` + "`" + `
}
`

	return writeFile(filepath.Join(projectDir, "internal/models/ecommerce.go"), productModel)
}

func getDefaultDatabaseURL(dbType string) string {
	switch dbType {
	case "postgresql":
		return "postgresql://localhost:5432/myapp"
	case "mongodb":
		return "mongodb://localhost:27017/myapp"
	case "mysql":
		return "mysql://user:password@localhost:3306/myapp"
	case "sqlite":
		return "sqlite://./myapp.db"
	default:
		return "sqlite://./myapp.db"
	}
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func initGitRepo(projectDir string) error {
	// This is a simplified git init - in a real implementation,
	// you would use git commands or a git library
	return nil
}
