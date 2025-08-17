package main

import (
	"context"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/http"
	"github.com/healtronlabs/gofasta/packages/orm"
	"github.com/healtronlabs/gofasta/packages/validation"
)

// User entity - works with both SQL and MongoDB
type User struct {
	ID        string    `gorm:"primaryKey" bson:"_id,omitempty" gofasta:"primary_key"`
	Email     string    `gorm:"uniqueIndex" bson:"email" gofasta:"unique,required" validate:"required,email"`
	FirstName string    `gorm:"not null" bson:"firstName" gofasta:"required" validate:"required,min=2,max=50"`
	LastName  string    `gorm:"not null" bson:"lastName" gofasta:"required" validate:"required,min=2,max=50"`
	Age       int       `gorm:"" bson:"age" validate:"gte=18,lte=120"`
	Status    string    `gorm:"type:varchar(20)" bson:"status" gofasta:"enum:active,inactive"`
	CreatedAt time.Time `gorm:"autoCreateTime" bson:"createdAt" gofasta:"auto_now_add"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" bson:"updatedAt" gofasta:"auto_now"`
}

// CreateUserDTO for validation
type CreateUserDTO struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50"`
	Age       int    `json:"age" validate:"gte=18,lte=120"`
}

// UserService demonstrates the unified repository pattern
type UserService struct {
	UserRepo orm.Repository[User] `inject:""`
}

// CreateUser creates a new user - same code works with any database
func (s *UserService) CreateUser(dto *CreateUserDTO) (*User, error) {
	user := &User{
		Email:     dto.Email,
		FirstName: dto.FirstName,
		LastName:  dto.LastName,
		Age:       dto.Age,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.UserRepo.Create(context.Background(), user)
}

// FindActiveUsers finds active users - same query syntax for all databases
func (s *UserService) FindActiveUsers() ([]*User, error) {
	return s.UserRepo.Query().
		Where("status", orm.OpEquals, "active").
		Where("age", orm.OpGreaterThan, 18).
		OrderBy("created_at", orm.DirectionDesc).
		Limit(10).
		Execute()
}

// FindUserByEmail finds user by email
func (s *UserService) FindUserByEmail(email string) (*User, error) {
	return s.UserRepo.Query().
		Where("email", orm.OpEquals, email).
		First()
}

// UserController handles HTTP requests
type UserController struct {
	UserService *UserService `inject:""`
}

// CreateUser HTTP handler
func (c *UserController) PostUser(dto *CreateUserDTO) (*User, error) {
	return c.UserService.CreateUser(dto)
}

// GetUsers HTTP handler
func (c *UserController) GetUsers() ([]*User, error) {
	return c.UserService.FindActiveUsers()
}

// GetUserByEmail HTTP handler
func (c *UserController) GetUsersByEmail(email string) (*User, error) {
	return c.UserService.FindUserByEmail(email)
}

// UserModule configures the user functionality
type UserModule struct {
	*core.BaseModule
}

func NewUserModule() *UserModule {
	module := &UserModule{
		BaseModule: core.NewBaseModule(),
	}

	module.AddProvider(&UserService{})
	module.AddController(&UserController{})

	return module
}

// AppModule - main application module
type AppModule struct {
	*core.BaseModule
}

func NewAppModule() *AppModule {
	module := &AppModule{
		BaseModule: core.NewBaseModule(),
	}

	// Import other modules
	module.AddImport(http.NewHTTPModule(3000))
	module.AddImport(validation.NewValidationModule())
	module.AddImport(NewUserModule())

	// Configure database - easily switch between databases
	// Uncomment one of the following:

	// PostgreSQL example
	module.AddImport(orm.NewGofastaOrmModuleFromURL("postgresql://user:pass@localhost:5432/gofasta_pg"))

	// MongoDB example
	// module.AddImport(orm.NewGofastaOrmModuleFromURL("mongodb://localhost:27017/gofasta_mongo"))

	// SQLite example
	// module.AddImport(orm.NewGofastaOrmModuleFromURL("sqlite://./gofasta.db"))

	return module
}

func main() {
	// Create Gofasta application
	app := core.CreateApp(NewAppModule())

	// Add global middleware
	// app.UseGlobalPipes(validation.NewValidationPipe())
	// app.UseGlobalGuards(auth.NewAuthGuard(jwtService))

	// Start the application
	if err := app.Start(); err != nil {
		panic(err)
	}

	// Listen on port 3000
	app.Listen(3000)
}

/*
This example demonstrates the revolutionary unified database API:

1. SAME MODEL DEFINITION works with PostgreSQL, MongoDB, MySQL, SQLite
2. SAME SERVICE CODE works with any database - just change the connection URL
3. SAME QUERY SYNTAX translates to database-specific queries automatically
4. ZERO CODE CHANGES needed to switch from PostgreSQL to MongoDB

To test different databases:
- Change the connection URL in NewAppModule()
- The same User model and UserService will work perfectly

Database-specific features:
- SQL databases: Uses GORM under the hood, supports JOINs, transactions
- MongoDB: Uses mongo-driver under the hood, supports document operations
- Query builder automatically translates to database-specific syntax

Benefits:
- Learn once, use everywhere
- Easy database migration without rewriting business logic
- Multi-database applications (different services can use different databases)
- Framework handles all the complexity while maintaining full database capabilities
*/
