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

// Example controllers with route metadata
type UserController struct {
	UserService *UserService `inject:""`
	Logger      *Logger      `inject:""`
}

// @Route GET /users
func (c *UserController) GetUsers(ctx *httpPackage.RequestContext) {
	users := c.UserService.GetAllUsers()
	ctx.JSON(200, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// @Route GET /users/:id
func (c *UserController) GetUser(ctx *httpPackage.RequestContext) {
	id := ctx.GetParam("id")
	c.Logger.Log(fmt.Sprintf("Getting user with ID: %s", id))
	
	ctx.JSON(200, map[string]interface{}{
		"id":      id,
		"name":    "User " + id,
		"message": "This is a demo user",
	})
}

// @Route POST /users
func (c *UserController) CreateUser(ctx *httpPackage.RequestContext) {
	var user User
	if err := ctx.ParseJSON(&user); err != nil {
		ctx.JSON(400, map[string]string{"error": "Invalid request body"})
		return
	}

	c.Logger.Log(fmt.Sprintf("Creating user: %s", user.Name))
	user.ID = len(c.UserService.users) + 1
	c.UserService.users = append(c.UserService.users, user)
	
	ctx.JSON(201, user)
}

type HealthController struct {
	Logger *Logger `inject:""`
}

// @Route GET /health
func (c *HealthController) Health(ctx *httpPackage.RequestContext) {
	c.Logger.Log("Health check requested")
	ctx.JSON(200, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// @Route GET /info
func (c *HealthController) Info(ctx *httpPackage.RequestContext) {
	ctx.JSON(200, map[string]interface{}{
		"name":        "Gofasta HTTP Example",
		"description": "Example HTTP server using Gofasta framework",
		"version":     "1.0.0",
		"framework":   "Gofasta",
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
	
	fmt.Println("\n📍 Available Endpoints:")
	fmt.Println("- Health and Info endpoints (registered via controller metadata)")
	fmt.Println("- User management endpoints (registered via controller metadata)")
	fmt.Println("- GET    /ws                  - WebSocket chat")
	fmt.Println("- GET    /static/*            - Static files")
	
	fmt.Println("\n🔧 Example Usage:")
	fmt.Printf("Visit http://%s:%d/static/index.html for interactive demo\n", config.Host, config.Port)
	fmt.Println("\nNote: This example demonstrates HTTP server features.")
	fmt.Println("Actual routes are determined by controller metadata annotations.")
	
	// Start HTTP server
	fmt.Printf("\n🚀 Starting HTTP server on http://%s:%d\n", config.Host, config.Port)
	if err := httpServer.Listen(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}