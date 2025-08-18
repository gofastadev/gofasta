//go:build basic
// +build basic

package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
	httpPackage "github.com/healtronlabs/gofasta/packages/http"
)

// Example services
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

// @Controller("/api/v1/users")
// @UseMiddleware("cors", "logging")
type UserController struct {
	UserService *UserService `inject:""`
	Logger      *Logger      `inject:""`
}

// @Get("")
// @UseMiddleware("cache")
func (c *UserController) GetUsers(ctx *httpPackage.RequestContext) {
	users := c.UserService.GetAllUsers()
	ctx.JSON(200, map[string]interface{}{
		"users":     users,
		"count":     len(users),
		"timestamp": time.Now().Unix(),
		"version":   "v1",
	})
}

// @Get("/:id")
// @UseGuards("authenticated")
// @Validate("id", "required,uuid")
func (c *UserController) GetUser(ctx *httpPackage.RequestContext) {
	id := ctx.GetParam("id")
	c.Logger.Log(fmt.Sprintf("Getting user with ID: %s", id))
	
	ctx.JSON(200, map[string]interface{}{
		"id":      id,
		"name":    "User " + id,
		"message": "This is a demo user from Gofasta controller",
		"meta": map[string]interface{}{
			"controller": "UserController",
			"method":     "GetUser",
			"timestamp":  time.Now().Unix(),
		},
	})
}

// @Post("")
// @HttpCode(201)
// @UseMiddleware("validation")
// @UsePipes("transform", "validation")
func (c *UserController) CreateUser(ctx *httpPackage.RequestContext) {
	var user User
	if err := ctx.ParseJSON(&user); err != nil {
		ctx.JSON(400, map[string]string{"error": "Invalid request body"})
		return
	}

	c.Logger.Log(fmt.Sprintf("Creating user: %s", user.Name))
	user.ID = len(c.UserService.users) + 1
	c.UserService.users = append(c.UserService.users, user)
	
	ctx.JSON(201, map[string]interface{}{
		"user":    user,
		"message": "User created successfully",
		"id":      user.ID,
	})
}

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
		"framework": "Gofasta",
	})
}

// @Get("/info")
// @UseMiddleware("cache")
func (c *HealthController) Info(ctx *httpPackage.RequestContext) {
	ctx.JSON(200, map[string]interface{}{
		"name":        "Gofasta HTTP Example",
		"description": "Example HTTP server using Gofasta framework with comment decorators",
		"version":     "1.0.0",
		"framework":   "Gofasta",
		"features":    []string{"decorators", "dependency-injection", "middleware", "websockets"},
	})
}

// WebSocket example
type ChatWebSocketHandler struct {
	Logger *Logger
}

func (h *ChatWebSocketHandler) OnConnect(conn *httpPackage.WebSocketConnection) error {
	h.Logger.Log("WebSocket client connected")
	return conn.WriteJSON(map[string]string{
		"type":    "welcome",
		"message": "Welcome to Gofasta Chat!",
	})
}

func (h *ChatWebSocketHandler) OnMessage(conn *httpPackage.WebSocketConnection, message []byte) error {
	h.Logger.Log(fmt.Sprintf("Received message: %s", string(message)))
	
	response := map[string]interface{}{
		"type":      "echo",
		"message":   string(message),
		"timestamp": time.Now().Unix(),
	}
	return conn.WriteJSON(response)
}

func (h *ChatWebSocketHandler) OnDisconnect(conn *httpPackage.WebSocketConnection) error {
	h.Logger.Log("WebSocket client disconnected")
	return nil
}

// HTTP Module
type HTTPAppModule struct {
	core.BaseModule
}

func (m *HTTPAppModule) Configure(container *core.DIContainer) error {
	// Register services
	if err := container.RegisterProvider(&Logger{}); err != nil {
		return err
	}
	
	if err := container.RegisterProvider(&UserService{}); err != nil {
		return err
	}
	
	// Register controllers
	if err := container.RegisterController(&UserController{}); err != nil {
		return err
	}
	
	if err := container.RegisterController(&HealthController{}); err != nil {
		return err
	}
	
	return nil
}

func main() {
	fmt.Println("🚀 Starting Gofasta HTTP Example")
	
	// Create application configuration
	config := &core.ApplicationConfig{
		Port:        8080,
		Host:        "localhost",
		Environment: "development",
		LogLevel:    "info",
	}
	
	// Create application with HTTP module
	app := core.CreateApp(&HTTPAppModule{}, config)
	
	// Start the application (this initializes the DI container)
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
	
	// Create a new DI container for the HTTP server
	httpContainer := core.NewDIContainer()
	
	// Create HTTP server with the container
	httpServer := httpPackage.NewHTTPServer(httpContainer)
	
	// Get services from the main app and register them in HTTP container
	loggerType := reflect.TypeOf((*Logger)(nil)).Elem()
	loggerInstance, err := app.GetService(loggerType)
	if err != nil {
		log.Fatalf("Failed to get Logger: %v", err)
	}
	logger := loggerInstance.(*Logger)
	
	userServiceType := reflect.TypeOf((*UserService)(nil)).Elem()
	userServiceInstance, err := app.GetService(userServiceType)
	if err != nil {
		log.Fatalf("Failed to get UserService: %v", err)
	}
	
	userControllerType := reflect.TypeOf((*UserController)(nil)).Elem()
	userControllerInstance, err := app.GetService(userControllerType)
	if err != nil {
		log.Fatalf("Failed to get UserController: %v", err)
	}
	userController := userControllerInstance.(*UserController)
	
	healthControllerType := reflect.TypeOf((*HealthController)(nil)).Elem()
	healthControllerInstance, err := app.GetService(healthControllerType)
	if err != nil {
		log.Fatalf("Failed to get HealthController: %v", err)
	}
	healthController := healthControllerInstance.(*HealthController)
	
	// Register instances in the HTTP container
	httpContainer.RegisterInstance(loggerType, logger)
	httpContainer.RegisterInstance(userServiceType, userServiceInstance)
	httpContainer.RegisterInstance(userControllerType, userController)
	httpContainer.RegisterInstance(healthControllerType, healthController)
	
	// Initialize HTTP container
	if err := httpContainer.Initialize(); err != nil {
		log.Fatalf("Failed to initialize HTTP container: %v", err)
	}
	

	// Register controllers with the HTTP server
	if err := httpServer.RegisterController(userController); err != nil {
		log.Fatalf("Failed to register UserController: %v", err)
	}
	
	if err := httpServer.RegisterController(healthController); err != nil {
		log.Fatalf("Failed to register HealthController: %v", err)
	}
	
	// WebSocket route
	chatHandler := &ChatWebSocketHandler{Logger: logger}
	httpServer.WebSocketUpgrade("/ws", chatHandler)
	
	// Static files
	httpServer.Static("/static", "./static")
	
	fmt.Println("\n🌐 HTTP Server Configuration:")
	fmt.Printf("- Host: %s\n", config.Host)
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Environment: %s\n", config.Environment)
	
	fmt.Println("\n📍 Gofasta Decorator Endpoints:")
	fmt.Println("- GET    /api/v1/users        - Get all users (@Controller + @Get)")
	fmt.Println("- GET    /api/v1/users/:id    - Get user by ID (@Get with params)")
	fmt.Println("- POST   /api/v1/users        - Create user (@Post + @HttpCode(201))")
	fmt.Println("- GET    /health              - Health check (@Controller(\"/health\"))")
	fmt.Println("- GET    /health/info         - App info (@Get(\"/info\"))")
	fmt.Println("- GET    /ws                  - WebSocket chat")
	fmt.Println("- GET    /static/*            - Static files")
	
	fmt.Println("\n🎯 Gofasta Advanced Features:")
	fmt.Println("- @Controller decorators with path prefixes")
	fmt.Println("- @Get, @Post HTTP method decorators")
	fmt.Println("- @UseMiddleware for middleware application")
	fmt.Println("- @UseGuards for authentication/authorization")
	fmt.Println("- @UsePipes for validation and transformation")
	fmt.Println("- @HttpCode for custom status codes")
	fmt.Println("- @Validate for parameter validation")
	fmt.Println("- Dependency injection with inject:\"\" tags")
	
	fmt.Println("\n🔧 Example API Calls:")
	fmt.Printf("curl http://%s:%d/health\n", config.Host, config.Port)
	fmt.Printf("curl http://%s:%d/api/v1/users\n", config.Host, config.Port)
	fmt.Printf("curl -X POST http://%s:%d/api/v1/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'\n", config.Host, config.Port)
	fmt.Printf("Visit http://%s:%d/static/index.html for interactive demo\n", config.Host, config.Port)
	
	// Start HTTP server
	fmt.Printf("\n🚀 Starting HTTP server on http://%s:%d\n", config.Host, config.Port)
	if err := httpServer.Listen(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}