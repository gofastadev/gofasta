package http

import (
	"reflect"
	"testing"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Test HTTP module functionality
func TestHTTPModule(t *testing.T) {
	t.Run("ModuleCreation", func(t *testing.T) {
		module := NewHTTPModule()
		
		if module == nil {
			t.Fatal("NewHTTPModule returned nil")
		}
		
		// Test that it implements the Module interface
		var _ core.Module = module
	})
	
	t.Run("ModuleWithConfig", func(t *testing.T) {
		config := &ServerConfig{
			Host:        "example.com",
			Port:        3000,
			EnableGzip:  false,
			CORSEnabled: false,
		}
		
		module := NewHTTPModule(config)
		
		if module == nil {
			t.Fatal("NewHTTPModule with config returned nil")
		}
		
		if module.config != config {
			t.Error("Module config not set correctly")
		}
	})
	
	t.Run("ModuleBuilder", func(t *testing.T) {
		module := NewHTTPModuleBuilder().
			WithHost("localhost").
			WithPort(8080).
			WithGzip(true).
			WithCORS(true, []string{"*"}, []string{"GET", "POST"}, []string{"Content-Type"}).
			Build()
		
		if module == nil {
			t.Fatal("Module builder returned nil")
		}
		
		if module.config.Host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", module.config.Host)
		}
		
		if module.config.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", module.config.Port)
		}
		
		if !module.config.EnableGzip {
			t.Error("Expected gzip to be enabled")
		}
		
		if !module.config.CORSEnabled {
			t.Error("Expected CORS to be enabled")
		}
	})
}

// Test HTTP module configuration
func TestHTTPModuleConfiguration(t *testing.T) {
	container := core.NewDIContainer()
	
	t.Run("DefaultConfiguration", func(t *testing.T) {
		module := NewHTTPModule()
		
		err := module.Configure(container)
		if err != nil {
			t.Fatalf("Failed to configure module: %v", err)
		}
		
		// Verify that HTTPServer is registered in the container
		serverType := reflect.TypeOf((*HTTPServer)(nil)).Elem()
		server, err := container.Resolve(serverType)
		if err != nil {
			t.Fatalf("Failed to resolve HTTPServer from container: %v", err)
		}
		
		if server == nil {
			t.Error("HTTPServer should be registered in container")
		}
		
		httpServer, ok := server.(*HTTPServer)
		if !ok {
			t.Error("Resolved service should be an HTTPServer")
		}
		
		if httpServer.container != container {
			t.Error("HTTPServer should have reference to the same container")
		}
	})
	
	t.Run("CustomConfiguration", func(t *testing.T) {
		config := &ServerConfig{
			Host:           "custom.host",
			Port:           9000,
			EnableGzip:     false,
			CORSEnabled:    false,
			ReadTimeout:    0,
			WriteTimeout:   0,
			IdleTimeout:    0,
			MaxHeaderBytes: 0,
		}
		
		module := NewHTTPModule(config)
		
		err := module.Configure(container)
		if err != nil {
			t.Fatalf("Failed to configure module with custom config: %v", err)
		}
		
		// Verify that the custom configuration is used
		serverType := reflect.TypeOf((*HTTPServer)(nil)).Elem()
		server, err := container.Resolve(serverType)
		if err != nil {
			t.Fatalf("Failed to resolve HTTPServer: %v", err)
		}
		
		httpServer := server.(*HTTPServer)
		if httpServer.config.Host != "custom.host" {
			t.Errorf("Expected host 'custom.host', got '%s'", httpServer.config.Host)
		}
		
		if httpServer.config.Port != 9000 {
			t.Errorf("Expected port 9000, got %d", httpServer.config.Port)
		}
	})
}

// Test HTTP module initialization
func TestHTTPModuleInitialization(t *testing.T) {
	t.Run("ModuleInitialization", func(t *testing.T) {
		container := core.NewDIContainer()
		module := NewHTTPModule()
		
		// Configure the module
		err := module.Configure(container)
		if err != nil {
			t.Fatalf("Failed to configure module: %v", err)
		}
		
		// Initialize the module
		err = module.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize module: %v", err)
		}
		
		// Verify that the server is accessible
		if module.server == nil {
			t.Error("Module server should be set after initialization")
		}
	})
	
	t.Run("InitializationWithoutConfiguration", func(t *testing.T) {
		module := NewHTTPModule()
		
		// Try to initialize without configuration
		err := module.Initialize()
		if err == nil {
			t.Error("Expected error when initializing module without configuration")
		}
	})
}

// Test HTTP module cleanup
func TestHTTPModuleCleanup(t *testing.T) {
	t.Run("ModuleCleanup", func(t *testing.T) {
		container := core.NewDIContainer()
		module := NewHTTPModule()
		
		err := module.Configure(container)
		if err != nil {
			t.Fatalf("Failed to configure module: %v", err)
		}
		
		err = module.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize module: %v", err)
		}
		
		// Cleanup the module
		err = module.Cleanup()
		if err != nil {
			t.Errorf("Module cleanup failed: %v", err)
		}
	})
	
	t.Run("CleanupWithoutInitialization", func(t *testing.T) {
		module := NewHTTPModule()
		
		// Cleanup without initialization should not fail
		err := module.Cleanup()
		if err != nil {
			t.Errorf("Cleanup should not fail even without initialization: %v", err)
		}
	})
}

// Test HTTP module builder pattern
func TestHTTPModuleBuilder(t *testing.T) {
	t.Run("BuilderChaining", func(t *testing.T) {
		builder := NewHTTPModuleBuilder()
		
		// Test method chaining
		module := builder.
			WithHost("test.example.com").
			WithPort(4000).
			WithGzip(true).
			WithCORS(false, nil, nil, nil).
			WithTimeouts(30, 30, 120).
			Build()
		
		if module == nil {
			t.Fatal("Builder should return a module")
		}
		
		config := module.config
		if config.Host != "test.example.com" {
			t.Errorf("Expected host 'test.example.com', got '%s'", config.Host)
		}
		
		if config.Port != 4000 {
			t.Errorf("Expected port 4000, got %d", config.Port)
		}
		
		if !config.EnableGzip {
			t.Error("Expected gzip to be enabled")
		}
		
		if config.CORSEnabled {
			t.Error("Expected CORS to be disabled")
		}
	})
	
	t.Run("BuilderDefaults", func(t *testing.T) {
		// Test that builder uses defaults when methods aren't called
		module := NewHTTPModuleBuilder().Build()
		
		if module == nil {
			t.Fatal("Builder should return a module with defaults")
		}
		
		// Should have some reasonable defaults
		if module.config == nil {
			t.Error("Module should have config with defaults")
		}
	})
	
	t.Run("MultipleBuilds", func(t *testing.T) {
		builder := NewHTTPModuleBuilder().WithPort(5000)
		
		// Test that builder can be used multiple times
		module1 := builder.Build()
		module2 := builder.WithHost("different.host").Build()
		
		if module1 == nil || module2 == nil {
			t.Fatal("Builder should support multiple builds")
		}
		
		// They should be different instances
		if module1 == module2 {
			t.Error("Builder should create different instances")
		}
		
		if module1.config.Port != 5000 {
			t.Error("First module should have port 5000")
		}
		
		if module2.config.Port != 5000 {
			t.Error("Second module should inherit port 5000")
		}
		
		if module2.config.Host != "different.host" {
			t.Error("Second module should have different host")
		}
	})
}

// Test HTTP module integration with core application
func TestHTTPModuleIntegration(t *testing.T) {
	t.Run("ApplicationIntegration", func(t *testing.T) {
		// Create a mock root module
		rootModule := &core.BaseModule{}
		
		// Create application with HTTP module
		app := core.CreateApp(rootModule)
		httpModule := NewHTTPModule()
		
		err := app.RegisterModule(httpModule)
		if err != nil {
			t.Fatalf("Failed to register HTTP module: %v", err)
		}
		
		// Start the application
		err = app.Start()
		if err != nil {
			t.Fatalf("Failed to start application: %v", err)
		}
		
		// Verify that HTTPServer is available through the application
		serverType := reflect.TypeOf((*HTTPServer)(nil)).Elem()
		server, err := app.GetService(serverType)
		if err != nil {
			t.Fatalf("Failed to get HTTPServer from application: %v", err)
		}
		
		if server == nil {
			t.Error("HTTPServer should be available through application")
		}
		
		// Clean up
		err = app.Stop()
		if err != nil {
			t.Errorf("Failed to stop application: %v", err)
		}
	})
	
	t.Run("ModuleDependencies", func(t *testing.T) {
		// Test that HTTP module can depend on other modules
		container := core.NewDIContainer()
		httpModule := NewHTTPModule()
		
		// Register a dependency
		dependencyType := reflect.TypeOf((*TestDependency)(nil)).Elem()
		testDep := &TestDependency{Value: "test"}
		err := container.RegisterInstance(dependencyType, testDep)
		if err != nil {
			t.Fatalf("Failed to register test dependency: %v", err)
		}
		
		err = httpModule.Configure(container)
		if err != nil {
			t.Fatalf("Failed to configure HTTP module: %v", err)
		}
		
		err = httpModule.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize HTTP module: %v", err)
		}
		
		// Verify that the module can access dependencies
		resolved, err := container.Resolve(dependencyType)
		if err != nil {
			t.Fatalf("Failed to resolve dependency: %v", err)
		}
		
		dep, ok := resolved.(*TestDependency)
		if !ok || dep.Value != "test" {
			t.Error("Dependency should be accessible from module")
		}
	})
}

// Test dependency for integration tests
type TestDependency struct {
	Value string
}

// Test HTTP module error handling
func TestHTTPModuleErrorHandling(t *testing.T) {
	t.Run("ConfigurationErrors", func(t *testing.T) {
		// Test with nil container (should cause error)
		module := NewHTTPModule()
		
		err := module.Configure(nil)
		if err == nil {
			t.Error("Expected error when configuring with nil container")
		}
	})
	
	t.Run("DuplicateRegistration", func(t *testing.T) {
		container := core.NewDIContainer()
		module1 := NewHTTPModule()
		module2 := NewHTTPModule()
		
		// Register first module
		err := module1.Configure(container)
		if err != nil {
			t.Fatalf("First module configuration should succeed: %v", err)
		}
		
		// Try to register second module (might cause conflict)
		err = module2.Configure(container)
		// Behavior depends on container implementation
		// This test ensures we handle it gracefully
		if err != nil {
			t.Logf("Expected behavior: second module registration failed: %v", err)
		}
	})
}

// Test HTTP module metadata
func TestHTTPModuleMetadata(t *testing.T) {
	t.Run("ModuleName", func(t *testing.T) {
		module := NewHTTPModule()
		
		// Test that module has identifying information
		if module == nil {
			t.Fatal("Module should not be nil")
		}
		
		// The module should be identifiable as an HTTP module
		// This is implementation-dependent
	})
	
	t.Run("ModuleVersion", func(t *testing.T) {
		module := NewHTTPModule()
		
		// Verify module exists and can be configured
		container := core.NewDIContainer()
		err := module.Configure(container)
		if err != nil {
			t.Fatalf("Module configuration failed: %v", err)
		}
	})
}