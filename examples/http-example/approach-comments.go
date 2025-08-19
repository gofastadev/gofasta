//go:build comments
// +build comments

package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
	httpPackage "github.com/healtronlabs/gofasta/packages/http"
)

// Shared types (same as main example)
type Logger struct {
	level string
}

func (l *Logger) Log(message string) {
	fmt.Printf("[%s] %s\n", l.level, message)
}

func (l *Logger) Initialize() error {
	l.level = "INFO"
	l.Log("Logger initialized")
	return nil
}

type UserService struct {
	Logger *Logger `inject:""`
	users  []User
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *UserService) Initialize() error {
	s.Logger.Log("UserService initialized")
	s.users = []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
		{ID: 3, Name: "Bob Johnson", Email: "bob@example.com"},
	}
	return nil
}

func (s *UserService) GetAllUsers() []User {
	s.Logger.Log("Getting all users")
	return s.users
}

func (s *UserService) GetUserByID(id int) (*User, error) {
	s.Logger.Log(fmt.Sprintf("Getting user with ID: %d", id))
	for _, user := range s.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (s *UserService) CreateUser(user User) User {
	s.Logger.Log(fmt.Sprintf("Creating user: %s", user.Name))
	user.ID = len(s.users) + 1
	s.users = append(s.users, user)
	return user
}

// Advanced Gofasta Controllers with Modern Decorators

// @Controller("/api/v1/users")
// @UseMiddleware("auth", "logging")
// @UseGuards("authenticated")
type UsersController struct {
	UserService *UserService `inject:""`
	Logger      *Logger      `inject:""`
}

// @Get("")
// @UseMiddleware("ratelimit")
func (c *UsersController) GetUsers(ctx *httpPackage.RequestContext) {
	users := c.UserService.GetAllUsers()
	ctx.JSON(200, map[string]interface{}{
		"users": users,
		"count": len(users),
		"meta": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"version":   "v1",
		},
	})
}

// @Get("/:id")
// @UseGuards("authenticated", "resource-owner")
// @UsePipes("validation")
func (c *UsersController) GetUser(ctx *httpPackage.RequestContext) {
	id := ctx.GetParam("id")
	c.Logger.Log(fmt.Sprintf("Getting user with ID: %s", id))
	
	user, err := c.UserService.GetUserByID(1) // Mock implementation
	if err != nil {
		ctx.JSON(404, map[string]string{"error": "User not found"})
		return
	}
	
	ctx.JSON(200, user)
}

// @Post("")
// @HttpCode(201)
// @UseMiddleware("validation")
// @UsePipes("transform", "validation")
func (c *UsersController) CreateUser(ctx *httpPackage.RequestContext) {
	var user User
	if err := ctx.ParseJSON(&user); err != nil {
		ctx.JSON(400, map[string]string{"error": "Invalid request body"})
		return
	}

	createdUser := c.UserService.CreateUser(user)
	ctx.JSON(201, createdUser)
}

// @Put("/:id")
// @UseGuards("authenticated", "resource-owner")
// @UsePipes("validation")
func (c *UsersController) UpdateUser(ctx *httpPackage.RequestContext) {
	id := ctx.GetParam("id")
	
	var user User
	if err := ctx.ParseJSON(&user); err != nil {
		ctx.JSON(400, map[string]string{"error": "Invalid request body"})
		return
	}

	c.Logger.Log(fmt.Sprintf("Updating user with ID: %s", id))
	ctx.JSON(200, map[string]interface{}{
		"id":      id,
		"message": "User updated successfully",
		"user":    user,
	})
}

// @Delete("/:id")
// @HttpCode(204)
// @UseGuards("authenticated", "admin")
func (c *UsersController) DeleteUser(ctx *httpPackage.RequestContext) {
	id := ctx.GetParam("id")
	c.Logger.Log(fmt.Sprintf("Deleting user with ID: %s", id))
	ctx.JSON(204, nil)
}

// Nested Resource: User Posts
// @Get("/:id/posts")
// @UseMiddleware("cache")
func (c *UsersController) GetUserPosts(ctx *httpPackage.RequestContext) {
	userId := ctx.GetParam("id")
	c.Logger.Log(fmt.Sprintf("Getting posts for user: %s", userId))
	
	posts := []map[string]interface{}{
		{"id": 1, "title": "First Post", "userId": userId},
		{"id": 2, "title": "Second Post", "userId": userId},
	}
	
	ctx.JSON(200, map[string]interface{}{
		"posts":  posts,
		"userId": userId,
		"count":  len(posts),
	})
}

// @Post("/:id/posts")
// @HttpCode(201)
// @UseGuards("authenticated", "resource-owner")
func (c *UsersController) CreateUserPost(ctx *httpPackage.RequestContext) {
	userId := ctx.GetParam("id")
	
	var post map[string]interface{}
	if err := ctx.ParseJSON(&post); err != nil {
		ctx.JSON(400, map[string]string{"error": "Invalid request body"})
		return
	}
	
	post["userId"] = userId
	post["id"] = time.Now().Unix() // Mock ID generation
	
	ctx.JSON(201, post)
}

// Admin Controller with API versioning
// @Controller("/api/v1/admin")
// @Version("v1")
// @UseGuards("authenticated", "admin")
// @UseMiddleware("audit-log")
type AdminController struct {
	Logger *Logger `inject:""`
}

// @Get("/stats")
// @UseMiddleware("cache")
func (c *AdminController) GetStats(ctx *httpPackage.RequestContext) {
	c.Logger.Log("Admin requesting system stats")
	
	ctx.JSON(200, map[string]interface{}{
		"users":         100,
		"posts":         500,
		"activeUsers":   75,
		"systemHealth": "good",
		"timestamp":    time.Now().Unix(),
	})
}

// @Post("/maintenance")
// @HttpCode(202)
// @UseFilters("admin-exception")
func (c *AdminController) TriggerMaintenance(ctx *httpPackage.RequestContext) {
	c.Logger.Log("Admin triggered maintenance mode")
	
	ctx.JSON(202, map[string]interface{}{
		"message": "Maintenance mode activated",
		"eta":     "5 minutes",
	})
}

// @Get("/users/:id/audit")
// @UseGuards("authenticated", "admin")
func (c *AdminController) GetUserAudit(ctx *httpPackage.RequestContext) {
	userId := ctx.GetParam("id")
	
	auditLog := []map[string]interface{}{
		{
			"action":    "login",
			"timestamp": time.Now().Add(-time.Hour).Unix(),
			"ip":        "192.168.1.1",
		},
		{
			"action":    "profile_update",
			"timestamp": time.Now().Add(-30 * time.Minute).Unix(),
			"ip":        "192.168.1.1",
		},
	}
	
	ctx.JSON(200, map[string]interface{}{
		"userId":   userId,
		"auditLog": auditLog,
		"count":    len(auditLog),
	})
}

// Health Controller - Simple endpoints
// @Controller("/health")
type HealthController struct {
	Logger *Logger `inject:""`
}

// @Get("")
func (c *HealthController) Health(ctx *httpPackage.RequestContext) {
	c.Logger.Log("Health check requested")
	ctx.JSON(200, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// @Get("/detailed")
// @UseMiddleware("cache")
func (c *HealthController) DetailedHealth(ctx *httpPackage.RequestContext) {
	ctx.JSON(200, map[string]interface{}{
		"status": "healthy",
		"services": map[string]string{
			"database": "connected",
			"redis":    "connected", 
			"storage":  "available",
		},
		"uptime":    "24h 30m 15s",
		"timestamp": time.Now().Unix(),
	})
}

// WebSocket Controller
// @Controller("/ws")
type WebSocketController struct {
	Logger *Logger `inject:""`
}

// WebSocket handler for real-time features
type ChatWebSocketHandler struct {
	Logger *Logger
}

func (h *ChatWebSocketHandler) OnConnect(conn *httpPackage.WebSocketConnection) error {
	h.Logger.Log("WebSocket client connected")
	return conn.WriteJSON(map[string]interface{}{
		"type":      "connection",
		"message":   "Connected to Gofasta Chat",
		"timestamp": time.Now().Unix(),
		"features":  []string{"chat", "notifications", "real-time updates"},
	})
}

func (h *ChatWebSocketHandler) OnMessage(conn *httpPackage.WebSocketConnection, message []byte) error {
	h.Logger.Log(fmt.Sprintf("Received message: %s", string(message)))
	
	// Echo with enhanced response
	response := map[string]interface{}{
		"type":      "message_echo",
		"original":  string(message),
		"processed": true,
		"timestamp": time.Now().Unix(),
		"server":    "gofasta-v1",
	}
	
	return conn.WriteJSON(response)
}

func (h *ChatWebSocketHandler) OnDisconnect(conn *httpPackage.WebSocketConnection) error {
	h.Logger.Log("WebSocket client disconnected")
	return nil
}

// Application Module with Gofasta's advanced configuration
type AdvancedAppModule struct {
	core.BaseModule
}

func (m *AdvancedAppModule) Configure(container *core.DIContainer) error {
	// Register core services
	if err := container.RegisterProvider(&Logger{}); err != nil {
		return err
	}
	
	if err := container.RegisterProvider(&UserService{}); err != nil {
		return err
	}
	
	// Register controllers
	controllers := []core.Controller{
		&UsersController{},
		&AdminController{},
		&HealthController{},
		&WebSocketController{},
	}
	
	for _, controller := range controllers {
		if err := container.RegisterController(controller); err != nil {
			return fmt.Errorf("failed to register controller %T: %w", controller, err)
		}
	}
	
	return nil
}

func runAdvancedExample() {
	fmt.Println("🚀 Starting Gofasta Advanced Decorator Example")
	
	// Create application configuration
	config := &core.ApplicationConfig{
		Port:        8080,
		Host:        "localhost",
		Environment: "development",
		LogLevel:    "info",
	}
	
	// Create application with advanced module
	app := core.CreateApp(&AdvancedAppModule{}, config)
	
	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
	
	// Create HTTP server
	httpContainer := core.NewDIContainer()
	httpServer := httpPackage.NewHTTPServer(httpContainer)
	
	// Get and register services
	loggerType := reflect.TypeOf((*Logger)(nil)).Elem()
	loggerInstance, err := app.GetService(loggerType)
	if err != nil {
		log.Fatalf("Failed to get Logger: %v", err)
	}
	logger := loggerInstance.(*Logger)
	
	// Register all controllers
	controllerTypes := []reflect.Type{
		reflect.TypeOf((*UsersController)(nil)).Elem(),
		reflect.TypeOf((*AdminController)(nil)).Elem(),
		reflect.TypeOf((*HealthController)(nil)).Elem(),
	}
	
	for _, controllerType := range controllerTypes {
		controllerInstance, err := app.GetService(controllerType)
		if err != nil {
			log.Fatalf("Failed to get controller %s: %v", controllerType.Name(), err)
		}
		
		if err := httpServer.RegisterController(controllerInstance.(core.Controller)); err != nil {
			log.Fatalf("Failed to register controller %s: %v", controllerType.Name(), err)
		}
	}
	
	// WebSocket setup
	chatHandler := &ChatWebSocketHandler{Logger: logger}
	httpServer.WebSocketUpgrade("/ws/chat", chatHandler)
	
	// Static files
	httpServer.Static("/static", "./static")
	
	fmt.Println("\n🌐 Gofasta Advanced HTTP Server Configuration:")
	fmt.Printf("- Host: %s\n", config.Host)
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Environment: %s\n", config.Environment)
	
	fmt.Println("\n📍 Advanced Decorator Endpoints:")
	fmt.Println("- GET    /api/v1/users           - Get all users")
	fmt.Println("- GET    /api/v1/users/:id       - Get user by ID")
	fmt.Println("- POST   /api/v1/users           - Create user")
	fmt.Println("- PUT    /api/v1/users/:id       - Update user")
	fmt.Println("- DELETE /api/v1/users/:id       - Delete user")
	fmt.Println("- GET    /api/v1/users/:id/posts - Get user posts")
	fmt.Println("- POST   /api/v1/users/:id/posts - Create user post")
	fmt.Println("- GET    /api/v1/admin/stats     - Admin statistics")
	fmt.Println("- POST   /api/v1/admin/maintenance - Trigger maintenance")
	fmt.Println("- GET    /api/v1/admin/users/:id/audit - User audit log")
	fmt.Println("- GET    /health                 - Basic health check")
	fmt.Println("- GET    /health/detailed        - Detailed health")
	fmt.Println("- GET    /ws/chat                - WebSocket chat")
	fmt.Println("- GET    /static/*               - Static files")
	
	fmt.Println("\n🎯 Gofasta Advanced Features:")
	fmt.Println("- @Controller decorators with path prefixes")
	fmt.Println("- @Get, @Post, @Put, @Delete route decorators")
	fmt.Println("- @UseMiddleware, @UseGuards, @UsePipes decorators")
	fmt.Println("- @HttpCode for custom status codes")
	fmt.Println("- @Version for API versioning")
	fmt.Println("- Nested resources (users/:id/posts)")
	fmt.Println("- Role-based access (@UseGuards)")
	fmt.Println("- Dependency injection with @inject tags")
	
	fmt.Println("\n🔧 Example API Calls:")
	fmt.Printf("curl http://%s:%d/health\n", config.Host, config.Port)
	fmt.Printf("curl http://%s:%d/api/v1/users\n", config.Host, config.Port)
	fmt.Printf("curl -X POST http://%s:%d/api/v1/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'\n", config.Host, config.Port)
	fmt.Printf("curl http://%s:%d/api/v1/users/1/posts\n", config.Host, config.Port)
	
	// Start HTTP server
	fmt.Printf("\n🚀 Starting Gofasta advanced HTTP server on http://%s:%d\n", config.Host, config.Port)
	if err := httpServer.Listen(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

// Run the comment-based decorators example
func main() {
	runAdvancedExample()
}