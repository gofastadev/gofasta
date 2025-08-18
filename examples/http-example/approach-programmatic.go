//go:build programmatic
// +build programmatic

package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/core/decorators"
	httpPackage "github.com/healtronlabs/gofasta/packages/http"
)

// Services (same as before)
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

// Modern Controllers using programmatic decorator registration
type ModernUserController struct {
	UserService *UserService `inject:""`
	Logger      *Logger      `inject:""`
}

func (c *ModernUserController) GetUsers(ctx *httpPackage.RequestContext) {
	users := c.UserService.GetAllUsers()
	ctx.JSON(200, map[string]interface{}{
		"users":     users,
		"count":     len(users),
		"timestamp": time.Now().Unix(),
		"version":   "v1",
	})
}

func (c *ModernUserController) GetUser(ctx *httpPackage.RequestContext) {
	id := ctx.GetParam("id")
	c.Logger.Log(fmt.Sprintf("Getting user with ID: %s", id))
	
	ctx.JSON(200, map[string]interface{}{
		"id":      id,
		"name":    "User " + id,
		"message": "Retrieved using modern Gofasta decorators",
		"meta": map[string]interface{}{
			"controller": "ModernUserController",
			"method":     "GetUser",
			"timestamp":  time.Now().Unix(),
		},
	})
}

func (c *ModernUserController) CreateUser(ctx *httpPackage.RequestContext) {
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
		"message": "User created successfully with modern decorators",
		"id":      user.ID,
	})
}

type ModernHealthController struct {
	Logger *Logger `inject:""`
}

func (c *ModernHealthController) Health(ctx *httpPackage.RequestContext) {
	c.Logger.Log("Health check requested")
	ctx.JSON(200, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"framework": "Gofasta",
		"decorators": "modern",
	})
}

func (c *ModernHealthController) DetailedHealth(ctx *httpPackage.RequestContext) {
	ctx.JSON(200, map[string]interface{}{
		"status": "healthy",
		"services": map[string]string{
			"database": "connected",
			"redis":    "connected", 
			"storage":  "available",
		},
		"uptime":    "24h 30m 15s",
		"timestamp": time.Now().Unix(),
		"decorators": "modern programmatic",
	})
}

// Register decorators programmatically
func registerDecorators() {
	// Register UserController with programmatic decorators
	decorators.Controller("/api/v1/users").
		UseMiddleware("cors", "logging").
		UseGuards("authenticated").
		Register(&ModernUserController{}).
		Get("GetUsers", "").
		UseMiddleware("cache").
		Register().
		Get("GetUser", "/:id").
		UseGuards("resource-owner").
		Register().
		Post("CreateUser", "").
		HttpCode(201).
		UseMiddleware("validation").
		UsePipes("transform").
		Register()

	// Register HealthController with programmatic decorators  
	decorators.Controller("/health").
		Register(&ModernHealthController{}).
		Get("Health", "").
		Register().
		Get("DetailedHealth", "/detailed").
		UseMiddleware("cache").
		Register()
}

// WebSocket example
type ChatWebSocketHandler struct {
	Logger *Logger
}

func (h *ChatWebSocketHandler) OnConnect(conn *httpPackage.WebSocketConnection) error {
	h.Logger.Log("WebSocket client connected")
	return conn.WriteJSON(map[string]string{
		"type":    "welcome",
		"message": "Welcome to Modern Gofasta Chat!",
	})
}

func (h *ChatWebSocketHandler) OnMessage(conn *httpPackage.WebSocketConnection, message []byte) error {
	h.Logger.Log(fmt.Sprintf("Received message: %s", string(message)))
	
	response := map[string]interface{}{
		"type":      "echo",
		"message":   string(message),
		"timestamp": time.Now().Unix(),
		"server":    "modern-gofasta",
	}
	return conn.WriteJSON(response)
}

func (h *ChatWebSocketHandler) OnDisconnect(conn *httpPackage.WebSocketConnection) error {
	h.Logger.Log("WebSocket client disconnected")
	return nil
}

// HTTP Module
type ModernHTTPAppModule struct {
	core.BaseModule
}

func (m *ModernHTTPAppModule) Configure(container *core.DIContainer) error {
	// Register services
	if err := container.RegisterProvider(&Logger{}); err != nil {
		return err
	}
	
	if err := container.RegisterProvider(&UserService{}); err != nil {
		return err
	}
	
	// Register controllers
	if err := container.RegisterController(&ModernUserController{}); err != nil {
		return err
	}
	
	if err := container.RegisterController(&ModernHealthController{}); err != nil {
		return err
	}
	
	return nil
}

func main() {
	fmt.Println("🚀 Starting Modern Gofasta HTTP Example")
	
	// Register decorators before starting the application
	registerDecorators()
	
	// Create application configuration
	config := &core.ApplicationConfig{
		Port:        8081,
		Host:        "localhost",
		Environment: "development",
		LogLevel:    "info",
	}
	
	// Create application with HTTP module
	app := core.CreateApp(&ModernHTTPAppModule{}, config)
	
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
	
	userControllerType := reflect.TypeOf((*ModernUserController)(nil)).Elem()
	userControllerInstance, err := app.GetService(userControllerType)
	if err != nil {
		log.Fatalf("Failed to get ModernUserController: %v", err)
	}
	userController := userControllerInstance.(*ModernUserController)
	
	healthControllerType := reflect.TypeOf((*ModernHealthController)(nil)).Elem()
	healthControllerInstance, err := app.GetService(healthControllerType)
	if err != nil {
		log.Fatalf("Failed to get ModernHealthController: %v", err)
	}
	healthController := healthControllerInstance.(*ModernHealthController)
	
	// Register instances in the HTTP container
	httpContainer.RegisterInstance(loggerType, logger)
	httpContainer.RegisterInstance(userServiceType, userServiceInstance)
	httpContainer.RegisterInstance(userControllerType, userController)
	httpContainer.RegisterInstance(healthControllerType, healthController)
	
	// Initialize HTTP container
	if err := httpContainer.Initialize(); err != nil {
		log.Fatalf("Failed to initialize HTTP container: %v", err)
	}
	
	// Debug: Check what metadata is being extracted
	userMeta, err := core.ExtractControllerMetadata(userController)
	if err != nil {
		log.Fatalf("Failed to extract user controller metadata: %v", err)
	}
	fmt.Printf("\n🔍 ModernUserController metadata: %d routes\n", len(userMeta.Routes))
	for _, route := range userMeta.Routes {
		fmt.Printf("- %s %s -> %s (Status: %d, Middleware: %v)\n", 
			route.Method, route.Path, route.Handler, route.StatusCode, route.Middleware)
	}

	healthMeta, err := core.ExtractControllerMetadata(healthController)
	if err != nil {
		log.Fatalf("Failed to extract health controller metadata: %v", err)
	}
	fmt.Printf("\n🔍 ModernHealthController metadata: %d routes\n", len(healthMeta.Routes))
	for _, route := range healthMeta.Routes {
		fmt.Printf("- %s %s -> %s (Middleware: %v)\n", 
			route.Method, route.Path, route.Handler, route.Middleware)
	}

	// Register controllers with the HTTP server
	if err := httpServer.RegisterController(userController); err != nil {
		log.Fatalf("Failed to register ModernUserController: %v", err)
	}
	
	if err := httpServer.RegisterController(healthController); err != nil {
		log.Fatalf("Failed to register ModernHealthController: %v", err)
	}
	
	// WebSocket route
	chatHandler := &ChatWebSocketHandler{Logger: logger}
	httpServer.WebSocketUpgrade("/ws", chatHandler)
	
	// Static files
	httpServer.Static("/static", "./examples/http-example/static")
	
	fmt.Println("\n🌐 Modern Gofasta HTTP Server Configuration:")
	fmt.Printf("- Host: %s\n", config.Host)
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Environment: %s\n", config.Environment)
	
	fmt.Println("\n📍 Modern Decorator Endpoints:")
	fmt.Println("- GET    /api/v1/users        - Get all users (programmatic @Controller + @Get)")
	fmt.Println("- GET    /api/v1/users/:id    - Get user by ID (@Get with params)")
	fmt.Println("- POST   /api/v1/users        - Create user (@Post + @HttpCode(201))")
	fmt.Println("- GET    /health              - Health check (@Controller(\"/health\"))")
	fmt.Println("- GET    /health/detailed     - Detailed health (@Get(\"/detailed\"))")
	fmt.Println("- GET    /ws                  - WebSocket chat")
	fmt.Println("- GET    /static/*            - Static files")
	
	fmt.Println("\n🎯 Modern Gofasta Features:")
	fmt.Println("- ✅ Programmatic decorator registration")
	fmt.Println("- ✅ Fluent API for controller and route configuration")
	fmt.Println("- ✅ @Controller, @Get, @Post HTTP method decorators")
	fmt.Println("- ✅ @UseMiddleware, @UseGuards, @UsePipes decorators")
	fmt.Println("- ✅ @HttpCode for custom status codes")
	fmt.Println("- ✅ Dependency injection with inject:\"\" tags")
	fmt.Println("- ✅ Type-safe decorator registration")
	
	fmt.Println("\n🔧 Example API Calls:")
	fmt.Printf("curl http://%s:%d/health\n", config.Host, config.Port)
	fmt.Printf("curl http://%s:%d/api/v1/users\n", config.Host, config.Port)
	fmt.Printf("curl -X POST http://%s:%d/api/v1/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'\n", config.Host, config.Port)
	fmt.Printf("Visit http://%s:%d/static/index.html for interactive demo\n", config.Host, config.Port)
	
	// Start HTTP server
	fmt.Printf("\n🚀 Starting Modern Gofasta HTTP server on http://%s:%d\n", config.Host, config.Port)
	if err := httpServer.Listen(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}