package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/core/decorators"
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
	
	// Demonstrate the new decorator system
	fmt.Println("\n🎯 Demonstrating New Decorator System:")
	demonstrateDecoratorSystem()
	
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

// TestController for demonstrating decorators
type TestController struct {
	UserService *UserService `inject:""`
}

func (c *TestController) GetUsers(ctx interface{}) {
	// Mock controller method
}

func (c *TestController) CreateUser(ctx interface{}) {
	// Mock controller method
}

func demonstrateDecoratorSystem() {
	// 1. Demonstrate programmatic decorator registration
	fmt.Println("\n1. 📝 Programmatic Decorator Registration:")
	
	// Register a test controller with decorators
	testController := &TestController{}
	controllerType := reflect.TypeOf(testController)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}
	
	// Register controller with fluent API
	decorators.Controller("/api/v1/test").
		UseMiddleware("cors", "auth").
		UseGuards("authenticated").
		Version("v1").
		Register(testController).
		Get("GetUsers", "").
		UseMiddleware("cache").
		Register().
		Post("CreateUser", "").
		HttpCode(201).
		UsePipes("validation").
		Register()
	
	fmt.Println("   ✅ Registered TestController with programmatic decorators")
	
	// 2. Extract and display the registered metadata
	fmt.Println("\n2. 🔍 Extracting Registered Metadata:")
	
	controllerMeta, routesMeta, found := decorators.GetControllerMetadata(controllerType)
	if found {
		fmt.Printf("   📋 Controller: %s\n", controllerMeta.Prefix)
		fmt.Printf("   🛡️  Middleware: %v\n", controllerMeta.Middleware)
		fmt.Printf("   🔐 Guards: %v\n", controllerMeta.Guards)
		fmt.Printf("   📊 Version: %s\n", controllerMeta.Version)
		
		fmt.Printf("   📍 Routes (%d):\n", len(routesMeta))
		for methodName, routeMeta := range routesMeta {
			fullPath := decorators.BuildFullPath(controllerMeta, routeMeta)
			fmt.Printf("     - %s %s -> %s", routeMeta.Method, fullPath, methodName)
			if routeMeta.StatusCode > 0 {
				fmt.Printf(" (Status: %d)", routeMeta.StatusCode)
			}
			if len(routeMeta.Middleware) > 0 {
				fmt.Printf(" [Middleware: %v]", routeMeta.Middleware)
			}
			fmt.Println()
		}
	} else {
		fmt.Println("   ❌ No registered metadata found")
	}
	
	// 3. Demonstrate integration with existing extraction
	fmt.Println("\n3. 🔄 Integration with Core.ExtractControllerMetadata:")
	
	metadata, err := core.ExtractControllerMetadata(testController)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Successfully extracted metadata using core.ExtractControllerMetadata\n")
		fmt.Printf("   📋 Controller: %s (Routes: %d)\n", metadata.Prefix, len(metadata.Routes))
		for _, route := range metadata.Routes {
			fmt.Printf("     - %s %s -> %s", route.Method, route.Path, route.Handler)
			if len(route.Middleware) > 0 {
				fmt.Printf(" [MW: %v]", route.Middleware)
			}
			if len(route.Guards) > 0 {
				fmt.Printf(" [Guards: %v]", route.Guards)
			}
			fmt.Println()
		}
	}
	
	// 4. Demonstrate decorator validation system
	fmt.Println("\n4. ✅ Decorator Validation System:")
	
	type ValidatedUser struct {
		Name  string `json:"name" validate:"required,minlength=2"`
		Email string `json:"email" validate:"required,email"`
		Age   int    `json:"age" validate:"min=18,max=120"`
	}
	
	// Test valid user
	validUser := ValidatedUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}
	
	validationRules, err := decorators.ExtractValidationDecorators(validUser)
	if err != nil {
		fmt.Printf("   ❌ Error extracting validation rules: %v\n", err)
	} else {
		fmt.Printf("   📝 Extracted validation rules for %d fields\n", len(validationRules))
		
		result, err := decorators.ValidateStruct(validUser)
		if err != nil {
			fmt.Printf("   ❌ Validation error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Valid user: %v (Errors: %d)\n", result.Valid, len(result.Errors))
		}
	}
	
	// Test invalid user
	invalidUser := ValidatedUser{
		Name:  "A",        // Too short
		Email: "invalid", // Invalid email
		Age:   15,        // Too young
	}
	
	result, err := decorators.ValidateStruct(invalidUser)
	if err != nil {
		fmt.Printf("   ❌ Validation error: %v\n", err)
	} else {
		fmt.Printf("   ❌ Invalid user: %v (Errors: %d)\n", result.Valid, len(result.Errors))
		for _, errorMsg := range result.Errors {
			fmt.Printf("     - %s\n", errorMsg)
		}
	}
	
	fmt.Println("\n🎉 Modern Decorator System Features:")
	fmt.Println("   ✅ Programmatic decorator registration")
	fmt.Println("   ✅ Fluent API for controller and route configuration")  
	fmt.Println("   ✅ Type-safe metadata extraction")
	fmt.Println("   ✅ Advanced validation system")
	fmt.Println("   ✅ Backward compatibility with existing core functions")
	fmt.Println("   ✅ Registry-based approach (no AST parsing required)")
}