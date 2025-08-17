package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Example service
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

func (l *Logger) Cleanup() error {
	l.Log("Logger cleanup")
	return nil
}

// Example service with dependency
type UserService struct {
	Logger *Logger `inject:""`
}

func (s *UserService) Initialize() error {
	s.Logger.Log("UserService initialized")
	return nil
}

func (s *UserService) GetUser(id string) string {
	s.Logger.Log(fmt.Sprintf("Getting user with ID: %s", id))
	return fmt.Sprintf("User-%s", id)
}

func (s *UserService) Cleanup() error {
	s.Logger.Log("UserService cleanup")
	return nil
}

// Example controller
type UserController struct {
	UserService *UserService `inject:""`
}

func (c *UserController) GetUser(id string) string {
	return c.UserService.GetUser(id)
}

func (c *UserController) GetUsers() []string {
	return []string{"User-1", "User-2", "User-3"}
}

// Example module
type AppModule struct {
	core.BaseModule
}

func (m *AppModule) Configure(container *core.DIContainer) error {
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
	
	return nil
}

func main() {
	fmt.Println("🚀 Starting Gofasta Core Example")
	
	// Create application configuration
	config := &core.ApplicationConfig{
		Port:        8080,
		Host:        "localhost",
		Environment: "development",
		LogLevel:    "info",
	}
	
	// Create application with root module
	app := core.CreateApp(&AppModule{}, config)
	
	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
	
	// Demonstrate service resolution
	fmt.Println("\n📦 Demonstrating Service Resolution:")
	
	// Get UserService
	userServiceType := reflect.TypeOf((*UserService)(nil)).Elem()
	userServiceInstance, err := app.GetService(userServiceType)
	if err != nil {
		log.Fatalf("Failed to get UserService: %v", err)
	}
	
	userService := userServiceInstance.(*UserService)
	user := userService.GetUser("123")
	fmt.Printf("Retrieved: %s\n", user)
	
	// Get UserController
	controllerType := reflect.TypeOf((*UserController)(nil)).Elem()
	controllerInstance, err := app.GetService(controllerType)
	if err != nil {
		log.Fatalf("Failed to get UserController: %v", err)
	}
	
	controller := controllerInstance.(*UserController)
	users := controller.GetUsers()
	fmt.Printf("All users: %v\n", users)
	
	// Demonstrate scoped services
	fmt.Println("\n🔄 Demonstrating Scoped Services:")
	scope := app.CreateScope("example-scope")
	fmt.Printf("Created scope: %v\n", scope != nil)
	
	// Clean up scope
	if err := app.DestroyScope("example-scope"); err != nil {
		log.Printf("Error destroying scope: %v", err)
	}
	
	// Demonstrate route metadata extraction
	fmt.Println("\n🛣️  Demonstrating Route Metadata Extraction:")
	routes, err := core.ExtractAllRouteMetadata(&UserController{})
	if err != nil {
		log.Printf("Error extracting routes: %v", err)
	} else {
		for _, route := range routes {
			fmt.Printf("Route: %s %s -> %s\n", route.Method, route.Path, route.Handler)
		}
	}
	
	// Demonstrate module metadata extraction
	fmt.Println("\n📋 Demonstrating Module Metadata Extraction:")
	moduleMetadata, err := core.ExtractModuleMetadata(&AppModule{})
	if err != nil {
		log.Printf("Error extracting module metadata: %v", err)
	} else {
		fmt.Printf("Module: %s\n", moduleMetadata.Name)
		fmt.Printf("Controllers: %v\n", moduleMetadata.Controllers)
		fmt.Printf("Providers: %v\n", moduleMetadata.Providers)
	}
	
	// Graceful shutdown
	fmt.Println("\n🛑 Shutting down application...")
	if err := app.Stop(); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}
	
	fmt.Println("✅ Gofasta Core Example completed successfully!")
}