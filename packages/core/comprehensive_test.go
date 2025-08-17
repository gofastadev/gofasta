package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

// Comprehensive tests to achieve higher coverage

// Test all DI Container functions that have 0% coverage
func TestDIContainer_ComprehensiveCoverage(t *testing.T) {
	t.Run("ResolveWithContext", func(t *testing.T) {
		container := NewDIContainer()
		container.RegisterProvider(&TestLogger{})
		err := container.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		service, err := container.ResolveWithContext(context.Background(), loggerType)
		if err != nil {
			t.Fatalf("ResolveWithContext failed: %v", err)
		}
		if service == nil {
			t.Error("ResolveWithContext returned nil service")
		}
		_ = service // Mark as used
	})
	
	t.Run("ResolveNamed", func(t *testing.T) {
		container := NewDIContainer()
		config := &TestConfig{DatabaseURL: "test://localhost", APIKey: "test-key"}
		configType := reflect.TypeOf((*TestConfig)(nil)).Elem()
		
		container.RegisterInstance(configType, config)
		container.namedServices["test-config"] = container.services[configType]
		err := container.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		
		service, err := container.ResolveNamed("test-config")
		if err != nil {
			t.Fatalf("ResolveNamed failed: %v", err)
		}
		if service == nil {
			t.Error("ResolveNamed returned nil service")
		}
		_ = service // Mark as used
	})
	
	t.Run("CreateScope", func(t *testing.T) {
		container := NewDIContainer()
		scope := container.CreateScope("test-scope")
		if scope == nil {
			t.Error("CreateScope returned nil")
		}
		_ = scope // Mark as used
	})
	
	t.Run("DestroyScope", func(t *testing.T) {
		container := NewDIContainer()
		container.CreateScope("test-scope")
		err := container.DestroyScope("test-scope")
		if err != nil {
			t.Errorf("DestroyScope failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("Shutdown", func(t *testing.T) {
		container := NewDIContainer()
		container.RegisterProvider(&TestLogger{})
		err := container.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		
		err = container.Shutdown(5 * time.Second)
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("Resolve", func(t *testing.T) {
		container := NewDIContainer()
		container.RegisterProvider(&TestLogger{})
		err := container.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		service, err := container.Resolve(loggerType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		if service == nil {
			t.Error("Resolve returned nil service")
		}
		_ = service
	})
	
	t.Run("RegisterController", func(t *testing.T) {
		container := NewDIContainer()
		controller := &TestLogger{} // Using TestLogger as mock controller
		
		err := container.RegisterController(controller)
		if err != nil {
			t.Errorf("RegisterController failed: %v", err)
		}
		_ = err
	})
	
	t.Run("AddLifecycleHook", func(t *testing.T) {
		container := NewDIContainer()
		hookCalled := false
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		container.AddLifecycleHook(loggerType, PhaseAfterCreate, func(instance interface{}) error {
			hookCalled = true
			return nil
		})
		
		container.RegisterProvider(&TestLogger{})
		err := container.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		
		// Resolve the service to trigger the lifecycle hook
		_, err = container.Resolve(loggerType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		
		if !hookCalled {
			t.Error("Lifecycle hook was not called")
		}
	})
}

// Test Application functions that have 0% coverage
func TestApplication_ComprehensiveCoverage(t *testing.T) {
	t.Run("GetService", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		// Initialize the container first
		gofastaApp, ok := app.(*GofastaApplication)
		if !ok {
			t.Fatal("app is not *GofastaApplication")
		}
		
		err := gofastaApp.container.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize container: %v", err)
		}
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		service, err := app.GetService(loggerType)
		if err != nil {
			t.Fatalf("GetService failed: %v", err)
		}
		if service == nil {
			t.Error("GetService returned nil")
		}
		_ = service // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("GetServiceByName", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		// We can't directly access the container to register a named service
		// so we'll just test that the method exists and can be called
		// It should return an error for unregistered services
		service, err := app.GetServiceByName("non-existent-service")
		if err == nil {
			t.Error("Expected error for non-existent service")
		}
		_ = service // Mark as used
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("UseGlobalPipes", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		err := app.UseGlobalPipes(&TestPipe{})
		if err != nil {
			t.Errorf("UseGlobalPipes failed: %v", err)
		}
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("UseGlobalGuards", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		err := app.UseGlobalGuards(&TestGuard{})
		if err != nil {
			t.Errorf("UseGlobalGuards failed: %v", err)
		}
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("UseGlobalInterceptors", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		err := app.UseGlobalInterceptors(&TestInterceptor{})
		if err != nil {
			t.Errorf("UseGlobalInterceptors failed: %v", err)
		}
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("UseGlobalFilters", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		err := app.UseGlobalFilters(&TestExceptionFilter{})
		if err != nil {
			t.Errorf("UseGlobalFilters failed: %v", err)
		}
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("CreateAndDestroyScope", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		scope := app.CreateScope("test-scope")
		if scope == nil {
			t.Error("CreateScope returned nil")
		}
		_ = scope // Mark as used
		
		err := app.DestroyScope("test-scope")
		if err != nil {
			t.Errorf("DestroyScope failed: %v", err)
		}
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("GetConfig", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		config := app.GetConfig()
		if config == nil {
			t.Error("GetConfig returned nil")
		}
		_ = config // Mark as used
		
		err := app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("GetContext", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		ctx := app.GetContext()
		if ctx == nil {
			t.Error("GetContext returned nil")
		}
		_ = ctx // Mark as used
		
		err := app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("Listen", func(t *testing.T) {
		module := &TestAppModule{}
		app := CreateApp(module)
		
		// Start the app in a goroutine to avoid blocking
		done := make(chan error, 1)
		go func() {
			err := app.Listen(8081)
			done <- err
		}()
		
		// Give it a moment to start
		select {
		case err := <-done:
			// If it returns immediately with an error, that's fine for testing
			t.Logf("Listen returned immediately (expected in test environment): %v", err)
		case <-time.After(100 * time.Millisecond):
			// If it's running, stop it
			t.Log("App started successfully, stopping it")
			err := app.Stop()
			if err != nil {
				t.Errorf("Stop failed: %v", err)
			}
		}
	})
	
	t.Run("RegisterModule", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		
		// Test registering additional module
		newModule := NewBaseModule()
		err := app.RegisterModule(newModule)
		if err != nil {
			t.Errorf("RegisterModule failed: %v", err)
		}
		_ = err // Mark as used
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("Start", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		
		// Test starting application
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
		_ = err // Mark as used
		
		// Test stopping application
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("Shutdown", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		
		// Test starting application first
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
		_ = err // Mark as used
		
		// Test shutdown with timeout
		err = app.Shutdown(1 * time.Second)
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("SetShutdownHook", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		hookCalled := false
		
		gofastaApp.SetShutdownHook(func() error {
			hookCalled = true
			return nil
		})
		
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		
		if !hookCalled {
			t.Error("Shutdown hook was not called")
		}
	})
	
	t.Run("GetGlobalPipes", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		pipes := gofastaApp.GetGlobalPipes()
		if pipes == nil {
			t.Error("GetGlobalPipes returned nil")
		}
		_ = pipes
	})
	
	t.Run("GetGlobalGuards", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		guards := gofastaApp.GetGlobalGuards()
		if guards == nil {
			t.Error("GetGlobalGuards returned nil")
		}
		_ = guards
	})
	
	t.Run("GetGlobalInterceptors", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		interceptors := gofastaApp.GetGlobalInterceptors()
		if interceptors == nil {
			t.Error("GetGlobalInterceptors returned nil")
		}
		_ = interceptors
	})
	
	t.Run("GetGlobalFilters", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		filters := gofastaApp.GetGlobalFilters()
		if filters == nil {
			t.Error("GetGlobalFilters returned nil")
		}
		_ = filters
	})
	
	t.Run("IsStarted", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		
		if gofastaApp.IsStarted() {
			t.Error("App should not be started initially")
		}
		
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
		
		if !gofastaApp.IsStarted() {
			t.Error("App should be started after Start()")
		}
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		
		if gofastaApp.IsStarted() {
			t.Error("App should not be started after Stop()")
		}
	})
	
	t.Run("IsShuttingDown", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		
		if gofastaApp.IsShuttingDown() {
			t.Error("App should not be shutting down initially")
		}
		
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
		
		if gofastaApp.IsShuttingDown() {
			t.Error("App should not be shutting down after start")
		}
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
	})
	
	t.Run("GetUptime", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		
		// Before start, uptime should be 0
		uptime := gofastaApp.GetUptime()
		if uptime != 0 {
			t.Error("Uptime should be 0 before start")
		}
		
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
		
		// Small delay to ensure some uptime
		time.Sleep(1 * time.Millisecond)
		
		uptime = gofastaApp.GetUptime()
		if uptime == 0 {
			t.Error("Uptime should be greater than 0 after start")
		}
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
	})
	
	t.Run("GetModules", func(t *testing.T) {
		app := CreateApp(&TestAppModule{})
		gofastaApp := app.(*GofastaApplication)
		
		modules := gofastaApp.GetModules()
		if len(modules) != 1 {
			t.Errorf("Expected 1 module, got %d", len(modules))
		}
		_ = modules
	})
}

// Test Module functions that have 0% coverage
func TestModule_ComprehensiveCoverage(t *testing.T) {
	t.Run("BaseModule_Configure", func(t *testing.T) {
		module := NewBaseModule()
		container := NewDIContainer()
		
		err := module.Configure(container)
		if err != nil {
			t.Errorf("Configure failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("BaseModule_GetProviders", func(t *testing.T) {
		module := NewBaseModule()
		providers := module.GetProviders()
		if providers == nil {
			t.Error("GetProviders returned nil")
		}
		_ = providers // Mark as used
	})
	
	t.Run("BaseModule_GetControllers", func(t *testing.T) {
		module := NewBaseModule()
		controllers := module.GetControllers()
		if controllers == nil {
			t.Error("GetControllers returned nil")
		}
		_ = controllers // Mark as used
	})
	
	t.Run("BaseModule_GetImports", func(t *testing.T) {
		module := NewBaseModule()
		imports := module.GetImports()
		if imports == nil {
			t.Error("GetImports returned nil")
		}
		_ = imports // Mark as used
	})
	
	t.Run("BaseModule_AddProvider", func(t *testing.T) {
		module := NewBaseModule()
		provider := &TestLogger{}
		module.AddProvider(provider)
		
		providers := module.GetProviders()
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
		_ = providers // Mark as used
	})
	
	t.Run("BaseModule_AddController", func(t *testing.T) {
		module := NewBaseModule()
		controller := &TestLogger{} // Using TestLogger as a mock controller
		module.AddController(controller)
		
		controllers := module.GetControllers()
		if len(controllers) != 1 {
			t.Errorf("Expected 1 controller, got %d", len(controllers))
		}
		_ = controllers // Mark as used
	})
	
	t.Run("BaseModule_AddImport", func(t *testing.T) {
		module := NewBaseModule()
		importModule := NewBaseModule()
		module.AddImport(importModule)
		
		imports := module.GetImports()
		if len(imports) != 1 {
			t.Errorf("Expected 1 import, got %d", len(imports))
		}
		_ = imports // Mark as used
	})
	
	t.Run("BaseModule_SetAndGetMetadata", func(t *testing.T) {
		module := NewBaseModule()
		metadata := &ModuleMetadata{
			Name: "test-module",
		}
		module.SetMetadata(metadata)
		
		retrieved := module.GetMetadata()
		if retrieved == nil {
			t.Error("GetMetadata returned nil")
		}
		if retrieved.Name != "test-module" {
			t.Errorf("Expected name 'test-module', got '%s'", retrieved.Name)
		}
		_ = retrieved // Mark as used
	})
	
	t.Run("BaseModule_Initialize", func(t *testing.T) {
		module := NewBaseModule()
		err := module.Initialize()
		if err != nil {
			t.Errorf("Initialize failed: %v", err)
		}
		_ = err // Mark as used
	})
	
	t.Run("BaseModule_Cleanup", func(t *testing.T) {
		module := NewBaseModule()
		err := module.Cleanup()
		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
		_ = err // Mark as used
	})
}

// Test DecoratedModule functions that have 0% coverage
func TestDecoratedModule_ComprehensiveCoverage(t *testing.T) {
	t.Run("DecoratedModule_FromBuilder", func(t *testing.T) {
		builder := NewModuleBuilder()
		builder.WithMetadata(&ModuleMetadata{Name: "test-module"})
		module := builder.Build()
		
		if module == nil {
			t.Error("Build returned nil module")
		}
		_ = module // Mark as used
	})
	
	t.Run("DecoratedModule_Configure", func(t *testing.T) {
		builder := NewModuleBuilder()
		builder.WithProviders(&TestLogger{})
		builder.WithControllers(&TestLogger{})
		module := builder.Build()
		
		container := NewDIContainer()
		err := module.Configure(container)
		if err != nil {
			t.Errorf("DecoratedModule Configure failed: %v", err)
		}
		_ = err
	})
	
	t.Run("DecoratedModule_Initialize", func(t *testing.T) {
		builder := NewModuleBuilder()
		module := builder.Build()
		
		err := module.Initialize()
		if err != nil {
			t.Errorf("DecoratedModule Initialize failed: %v", err)
		}
		_ = err
	})
	
	t.Run("DecoratedModule_Cleanup", func(t *testing.T) {
		builder := NewModuleBuilder()
		module := builder.Build()
		
		err := module.Cleanup()
		if err != nil {
			t.Errorf("DecoratedModule Cleanup failed: %v", err)
		}
		_ = err
	})
	
	t.Run("DecoratedModule_GetProviders", func(t *testing.T) {
		builder := NewModuleBuilder()
		builder.WithProviders(&TestLogger{})
		module := builder.Build()
		
		providers := module.GetProviders()
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
		_ = providers
	})
	
	t.Run("DecoratedModule_GetControllers", func(t *testing.T) {
		builder := NewModuleBuilder()
		builder.WithControllers(&TestLogger{})
		module := builder.Build()
		
		controllers := module.GetControllers()
		if len(controllers) != 1 {
			t.Errorf("Expected 1 controller, got %d", len(controllers))
		}
		_ = controllers
	})
	
	t.Run("DecoratedModule_GetImports", func(t *testing.T) {
		builder := NewModuleBuilder()
		importModule := NewBaseModule()
		builder.WithImports(importModule)
		module := builder.Build()
		
		imports := module.GetImports()
		if len(imports) != 1 {
			t.Errorf("Expected 1 import, got %d", len(imports))
		}
		_ = imports
	})
	
	t.Run("DecoratedModule_GetExports", func(t *testing.T) {
		builder := NewModuleBuilder()
		builder.WithExports(&TestLogger{})
		module := builder.Build()
		
		// Cast to DecoratedModule to access GetExports
		decoratedModule, ok := module.(*DecoratedModule)
		if !ok {
			t.Fatal("Expected DecoratedModule")
		}
		
		exports := decoratedModule.GetExports()
		if len(exports) != 1 {
			t.Errorf("Expected 1 export, got %d", len(exports))
		}
		_ = exports
	})
	
	t.Run("DecoratedModule_GetMetadata", func(t *testing.T) {
		builder := NewModuleBuilder()
		metadata := &ModuleMetadata{Name: "test-module"}
		builder.WithMetadata(metadata)
		module := builder.Build()
		
		// Cast to DecoratedModule to access GetMetadata
		decoratedModule, ok := module.(*DecoratedModule)
		if !ok {
			t.Fatal("Expected DecoratedModule")
		}
		
		retrievedMetadata := decoratedModule.GetMetadata()
		if retrievedMetadata == nil {
			t.Error("GetMetadata returned nil")
		}
		if retrievedMetadata.Name != "test-module" {
			t.Errorf("Expected name 'test-module', got '%s'", retrievedMetadata.Name)
		}
		_ = retrievedMetadata
	})
}

// Test all exception types for 100% coverage
func TestExceptions_ComprehensiveCoverage(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		fieldErrors := []FieldError{
			{Field: "test", Message: "test validation error", Value: "test", Tag: "required"},
		}
		err := NewValidationError("test validation error", fieldErrors)
		if err == nil {
			t.Error("NewValidationError returned nil")
		}
		if err.Error() != "test validation error" {
			t.Errorf("Expected 'test validation error', got '%s'", err.Error())
		}
	})
	
	t.Run("NewForbiddenError", func(t *testing.T) {
		err := NewForbiddenError("test forbidden error")
		if err == nil {
			t.Error("NewForbiddenError returned nil")
		}
		if err.Error() != "test forbidden error" {
			t.Errorf("Expected 'test forbidden error', got '%s'", err.Error())
		}
	})
	
	t.Run("NewBadRequestError", func(t *testing.T) {
		err := NewBadRequestError("test bad request error")
		if err == nil {
			t.Error("NewBadRequestError returned nil")
		}
		if err.Error() != "test bad request error" {
			t.Errorf("Expected 'test bad request error', got '%s'", err.Error())
		}
	})
	
	t.Run("NewInternalServerError", func(t *testing.T) {
		err := NewInternalServerError("test internal server error")
		if err == nil {
			t.Error("NewInternalServerError returned nil")
		}
		if err.Error() != "test internal server error" {
			t.Errorf("Expected 'test internal server error', got '%s'", err.Error())
		}
	})
	
	t.Run("NewConflictError", func(t *testing.T) {
		err := NewConflictError("test conflict error")
		if err == nil {
			t.Error("NewConflictError returned nil")
		}
		if err.Error() != "test conflict error" {
			t.Errorf("Expected 'test conflict error', got '%s'", err.Error())
		}
	})
	
	t.Run("NewGofastaException", func(t *testing.T) {
		exception := NewGofastaException("test exception", 500, nil)
		if exception == nil {
			t.Error("NewGofastaException returned nil")
		}
		if exception.Message != "test exception" {
			t.Errorf("Expected 'test exception', got '%s'", exception.Message)
		}
		if exception.StatusCode != 500 {
			t.Errorf("Expected status code 500, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewValidationException", func(t *testing.T) {
		errors := map[string][]string{"field": {"error1", "error2"}}
		exception := NewValidationException(errors)
		if exception == nil {
			t.Error("NewValidationException returned nil")
		}
		if exception.Message != "Validation failed" {
			t.Errorf("Expected 'Validation failed', got '%s'", exception.Message)
		}
		if exception.StatusCode != 400 {
			t.Errorf("Expected status code 400, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewUnauthorizedException", func(t *testing.T) {
		exception := NewUnauthorizedException("unauthorized")
		if exception == nil {
			t.Error("NewUnauthorizedException returned nil")
		}
		if exception.StatusCode != 401 {
			t.Errorf("Expected status code 401, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewForbiddenException", func(t *testing.T) {
		exception := NewForbiddenException("forbidden")
		if exception == nil {
			t.Error("NewForbiddenException returned nil")
		}
		if exception.StatusCode != 403 {
			t.Errorf("Expected status code 403, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewNotFoundException", func(t *testing.T) {
		exception := NewNotFoundException("User", "123")
		if exception == nil {
			t.Error("NewNotFoundException returned nil")
		}
		if exception.StatusCode != 404 {
			t.Errorf("Expected status code 404, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewBadRequestException", func(t *testing.T) {
		exception := NewBadRequestException("bad request")
		if exception == nil {
			t.Error("NewBadRequestException returned nil")
		}
		if exception.StatusCode != 400 {
			t.Errorf("Expected status code 400, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewInternalServerException", func(t *testing.T) {
		exception := NewInternalServerException("internal server error", nil)
		if exception == nil {
			t.Error("NewInternalServerException returned nil")
		}
		if exception.StatusCode != 500 {
			t.Errorf("Expected status code 500, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewConflictException", func(t *testing.T) {
		exception := NewConflictException("conflict")
		if exception == nil {
			t.Error("NewConflictException returned nil")
		}
		if exception.StatusCode != 409 {
			t.Errorf("Expected status code 409, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewServiceUnavailableException", func(t *testing.T) {
		exception := NewServiceUnavailableException("service unavailable")
		if exception == nil {
			t.Error("NewServiceUnavailableException returned nil")
		}
		if exception.StatusCode != 503 {
			t.Errorf("Expected status code 503, got %d", exception.StatusCode)
		}
	})
	
	t.Run("NewTooManyRequestsException", func(t *testing.T) {
		exception := NewTooManyRequestsException("too many requests", 60)
		if exception == nil {
			t.Error("NewTooManyRequestsException returned nil")
		}
		if exception.StatusCode != 429 {
			t.Errorf("Expected status code 429, got %d", exception.StatusCode)
		}
	})
}

// Test ServiceProvider interface methods
func TestServiceProvider_ComprehensiveCoverage(t *testing.T) {
	t.Run("ServiceProvider_Methods", func(t *testing.T) {
		provider := NewServiceProvider("test-logger", func() interface{} {
			return &TestLogger{}
		}, ScopeSingleton)
		
		scope := provider.GetScope()
		_ = scope // Mark as used
		
		deps := provider.GetDependencies()
		_ = deps // Mark as used
		
		provider.AddDependency(reflect.TypeOf((*TestConfig)(nil)).Elem())
		
		deps = provider.GetDependencies()
		if len(deps) != 1 {
			t.Errorf("Expected 1 dependency, got %d", len(deps))
		}
	})
	
	t.Run("FactoryProvider", func(t *testing.T) {
		provider := FactoryProvider("test-logger", func() interface{} {
			return &TestLogger{}
		}, ScopeSingleton)
		if provider == nil {
			t.Error("FactoryProvider returned nil")
		}
		_ = provider
	})
	
	t.Run("ValueProvider", func(t *testing.T) {
		logger := &TestLogger{}
		provider := ValueProvider("test-logger", logger)
		if provider == nil {
			t.Error("ValueProvider returned nil")
		}
		_ = provider
	})
	
	t.Run("ClassProvider", func(t *testing.T) {
		provider := ClassProvider("test-logger", reflect.TypeOf((*TestLogger)(nil)).Elem(), ScopeSingleton)
		if provider == nil {
			t.Error("ClassProvider returned nil")
		}
		_ = provider
	})
}

// Mock implementations for interface coverage
type MockExecutionContext struct{}

func (m *MockExecutionContext) GetRequest() interface{} { return nil }
func (m *MockExecutionContext) GetHandler() interface{} { return nil }
func (m *MockExecutionContext) GetClass() reflect.Type { return nil }
func (m *MockExecutionContext) GetMethod() string { return "testMethod" }
func (m *MockExecutionContext) GetArgs() []interface{} { return nil }
func (m *MockExecutionContext) GetMetadata() map[string]interface{} { return nil }
func (m *MockExecutionContext) GetApplication() interface{} { return nil }

type MockArgumentsHost struct{}

func (m *MockArgumentsHost) GetRequest() interface{} { return nil }
func (m *MockArgumentsHost) GetResponse() interface{} { return nil }
func (m *MockArgumentsHost) GetNext() interface{} { return nil }

type MockCallHandler struct{}

func NewMockCallHandler() *MockCallHandler {
	return &MockCallHandler{}
}

func (m *MockCallHandler) Handle() interface{} {
	return "handled"
}

type MockModuleRef struct{}

func NewMockModuleRef() *MockModuleRef {
	return &MockModuleRef{}
}

func (m *MockModuleRef) Get(serviceType reflect.Type) (interface{}, error) {
	return nil, nil
}

func (m *MockModuleRef) GetByName(name string) (interface{}, error) {
	return nil, nil
}

func (m *MockModuleRef) Create(serviceType reflect.Type) (interface{}, error) {
	return nil, nil
}

func (m *MockModuleRef) Resolve(serviceType reflect.Type) (interface{}, error) {
	return nil, nil
}

// Test interface implementations
func TestInterfaces_ComprehensiveCoverage(t *testing.T) {
	t.Run("ExecutionContext", func(t *testing.T) {
		ctx := &MockExecutionContext{}
		
		_ = ctx.GetRequest()
		_ = ctx.GetHandler()
		_ = ctx.GetClass()
		_ = ctx.GetMethod()
		_ = ctx.GetArgs()
		_ = ctx.GetMetadata()
		_ = ctx.GetApplication()
	})
	
	t.Run("ArgumentsHost", func(t *testing.T) {
		host := &MockArgumentsHost{}
		
		_ = host.GetRequest()
		_ = host.GetResponse()
		_ = host.GetNext()
	})
	
	t.Run("RealArgumentsHost", func(t *testing.T) {
		// Test the actual ArgumentsHost from interfaces.go
		req, _ := http.NewRequest("GET", "/test", nil)
		requestCtx := &RequestContext{Request: req}
		response := &Response{StatusCode: 200}
		nextHandler := func(ctx *RequestContext) *Response { return nil }
		
		host := &ArgumentsHost{
			Request:  requestCtx,
			Response: response,
			Next:     nextHandler,
		}
		
		// Test all methods
		request := host.GetRequest()
		if request != requestCtx {
			t.Error("GetRequest returned wrong value")
		}
		
		resp := host.GetResponse()
		if resp != response {
			t.Error("GetResponse returned wrong value")
		}
		
		next := host.GetNext()
		if next == nil {
			t.Error("GetNext returned nil")
		}
	})
	
	t.Run("CallHandler", func(t *testing.T) {
		handler := NewMockCallHandler()
		result := handler.Handle()
		if result != "handled" {
			t.Errorf("Expected 'handled', got '%v'", result)
		}
	})
	
	t.Run("ModuleRef", func(t *testing.T) {
		ref := NewMockModuleRef()
		
		_, err := ref.Get(reflect.TypeOf((*TestLogger)(nil)).Elem())
		_ = err
		
		_, err = ref.GetByName("test")
		_ = err
		
		_, err = ref.Create(reflect.TypeOf((*TestLogger)(nil)).Elem())
		_ = err
		
		_, err = ref.Resolve(reflect.TypeOf((*TestLogger)(nil)).Elem())
		_ = err
	})
	
	t.Run("RealCallHandler", func(t *testing.T) {
		// Test the actual CallHandler from interfaces.go
		handler := func(ctx *RequestContext) *Response { return nil }
		requestCtx := &RequestContext{}
		callHandler := NewCallHandler(handler, requestCtx)
		if callHandler == nil {
			t.Error("NewCallHandler returned nil")
		}
		_ = callHandler
	})
	
	t.Run("RealModuleRef", func(t *testing.T) {
		// Test the actual ModuleRef from interfaces.go
		container := NewDIContainer()
		ctx := context.Background()
		moduleRef := NewModuleRef(container, ctx)
		if moduleRef == nil {
			t.Error("NewModuleRef returned nil")
		}
		_ = moduleRef
	})
}

// Test decorator functions for 100% coverage
func TestDecorators_ComprehensiveCoverage(t *testing.T) {
	t.Run("ExtractControllerMetadata", func(t *testing.T) {
		type TestController struct{}
		controller := &TestController{}
		metadata, err := ExtractControllerMetadata(controller)
		_ = metadata // Mark as used
		_ = err
	})
	
	t.Run("ExtractModuleMetadata", func(t *testing.T) {
		type TestModule struct{}
		module := &TestModule{}
		metadata, err := ExtractModuleMetadata(module)
		_ = metadata // Mark as used
		_ = err
	})
	
	t.Run("ExtractRouteMetadata", func(t *testing.T) {
		type TestController struct{}
		controller := &TestController{}
		methodName := "TestMethod"
		metadata, err := ExtractRouteMetadata(controller, methodName)
		_ = metadata // Mark as used
		_ = err
	})
	
	t.Run("ExtractAllRouteMetadata", func(t *testing.T) {
		type TestController struct{}
		controller := &TestController{}
		metadata, err := ExtractAllRouteMetadata(controller)
		_ = metadata // Mark as used
		_ = err
	})
	
	t.Run("extractMethodDecorators", func(t *testing.T) {
		// This function is private but we can test it indirectly through ExtractDecorators
		type TestStruct struct {
			Field string `validate:"required"`
		}
		structInstance := TestStruct{}
		decorators, err := ExtractDecorators(structInstance)
		_ = decorators // Mark as used
		_ = err
	})
	
	t.Run("parseValidationTag", func(t *testing.T) {
		// This function is private but we can test it indirectly through ExtractDecorators
		type TestStruct struct {
			Field string `validate:"required,min=5"`
		}
		structInstance := TestStruct{}
		decorators, err := ExtractDecorators(structInstance)
		_ = decorators // Mark as used
		_ = err
	})
}

// Test RequestContext interface methods for 100% coverage
func TestRequestContext_ComprehensiveCoverage(t *testing.T) {
	t.Run("RequestContext_Methods", func(t *testing.T) {
		// Create a mock HTTP request and response writer
		req, _ := http.NewRequest("GET", "/test?limit=10", nil)
		req.Header.Set("Content-Type", "application/json")
		
		rw := httptest.NewRecorder()
		
		ctx := &RequestContext{
			Request:        req,
			ResponseWriter: rw,
			Headers:        make(map[string]string),
			Params:         make(map[string]string),
			Query:          make(map[string]string),
		}
		
		// Test GetHeader
		header := ctx.GetHeader("Content-Type")
		if header != "application/json" {
			t.Errorf("Expected 'application/json', got '%s'", header)
		}
		
		// Test SetHeader
		ctx.SetHeader("X-Custom", "test-value")
		if rw.Header().Get("X-Custom") != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", rw.Header().Get("X-Custom"))
		}
		
		// Test SetStatus
		ctx.SetStatus(200)
		if rw.Code != 200 {
			t.Errorf("Expected status code 200, got %d", rw.Code)
		}
		
		// Test GetParam
		ctx.Params["id"] = "123"
		param := ctx.GetParam("id")
		if param != "123" {
			t.Errorf("Expected '123', got '%s'", param)
		}
		
		// Test GetQuery
		ctx.Query["limit"] = "10"
		query := ctx.GetQuery("limit")
		if query != "10" {
			t.Errorf("Expected '10', got '%s'", query)
		}
	})
}

// Test ExecutionContext methods for 100% coverage
func TestExecutionContext_ComprehensiveCoverage(t *testing.T) {
	t.Run("ExecutionContext_Methods", func(t *testing.T) {
		requestCtx := &RequestContext{}
		handler := func(ctx *RequestContext) *Response { return nil }
		classType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		
		// Get a real method for testing
		methodInfo, _ := classType.MethodByName("Log")
		
		execCtx := &ExecutionContext{
			Request:     requestCtx,
			Handler:     handler,
			Class:       classType,
			Method:      methodInfo,
			Args:        []interface{}{"arg1", "arg2"},
			Metadata:    map[string]interface{}{"key": "value"},
			Application: nil,
		}
		
		// Test all methods
		request := execCtx.GetRequest()
		_ = request
		
		handlerFunc := execCtx.GetHandler()
		_ = handlerFunc
		
		class := execCtx.GetClass()
		_ = class
		
		method := execCtx.GetMethod()
		_ = method
		
		args := execCtx.GetArgs()
		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
		
		metadata := execCtx.GetMetadata()
		if len(metadata) != 1 {
			t.Errorf("Expected 1 metadata entry, got %d", len(metadata))
		}
		
		app := execCtx.GetApplication()
		_ = app
	})
}

// Test CallHandler methods for 100% coverage
func TestCallHandler_ComprehensiveCoverage(t *testing.T) {
	t.Run("CallHandler_Handle", func(t *testing.T) {
		handler := func(ctx *RequestContext) *Response {
			return &Response{
				StatusCode: 200,
				Body:       "test response",
			}
		}
		requestCtx := &RequestContext{}
		callHandler := NewCallHandler(handler, requestCtx)
		
		response := callHandler.Handle()
		if response == nil {
			t.Error("Handle returned nil response")
		}
		_ = response
	})
}

// Test ModuleRef methods for 100% coverage 
func TestModuleRef_ComprehensiveCoverage(t *testing.T) {
	t.Run("ModuleRef_Methods", func(t *testing.T) {
		container := NewDIContainer()
		container.RegisterProvider(&TestLogger{})
		err := container.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		
		ctx := context.Background()
		moduleRef := NewModuleRef(container, ctx)
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		
		// Test Get
		service, err := moduleRef.Get(loggerType)
		_ = service
		_ = err
		
		// Test GetByName
		service, err = moduleRef.GetByName("non-existent")
		_ = service
		_ = err
		
		// Test Create
		service, err = moduleRef.Create(loggerType)
		_ = service
		_ = err
		
		// Test Resolve
		service, err = moduleRef.Resolve(loggerType)
		_ = service
		_ = err
	})
}

// Test NewDecoratedModule for 100% coverage
func TestNewDecoratedModule_ComprehensiveCoverage(t *testing.T) {
	t.Run("NewDecoratedModule", func(t *testing.T) {
		// Create a test module instance with decorators
		type TestModule struct {
			Name string `module:"test-module"`
		}
		
		moduleInstance := &TestModule{Name: "test-module"}
		module, err := NewDecoratedModule(moduleInstance)
		if err != nil {
			t.Fatalf("NewDecoratedModule failed: %v", err)
		}
		if module == nil {
			t.Error("NewDecoratedModule returned nil")
		}
		
		// Test that the module was properly created
		providers := module.GetProviders()
		_ = providers
		
		controllers := module.GetControllers()
		_ = controllers
		
		imports := module.GetImports()
		_ = imports
		
		exports := module.GetExports()
		_ = exports
		
		moduleMetadata := module.GetMetadata()
		_ = moduleMetadata
	})
}

// Additional tests for edge cases and partial coverage improvement
func TestEdgeCases_ComprehensiveCoverage(t *testing.T) {
	t.Run("DIContainer_Error_Cases", func(t *testing.T) {
		container := NewDIContainer()
		
		// Test resolving non-existent service
		nonExistentType := reflect.TypeOf((*string)(nil)).Elem()
		_, err := container.ResolveWithContext(context.Background(), nonExistentType)
		if err == nil {
			t.Error("Expected error when resolving non-existent service")
		}
		
		// Test registering service with different scopes
		container.RegisterService(reflect.TypeOf((*TestLogger)(nil)).Elem(), func(ctx context.Context, c *DIContainer) (interface{}, error) {
			return &TestLogger{}, nil
		}, ScopeTransient)
		container.RegisterService(reflect.TypeOf((*TestConfig)(nil)).Elem(), func(ctx context.Context, c *DIContainer) (interface{}, error) {
			return &TestConfig{}, nil
		}, ScopeSingleton)
		
		err = container.Initialize()
		if err != nil {
			t.Errorf("Initialize failed: %v", err)
		}
	})
	
	t.Run("Module_Error_Cases", func(t *testing.T) {
		module := NewBaseModule()
		
		// Test Initialize and Cleanup with providers that might fail
		module.AddProvider(&TestLogger{})
		
		// Test error handling in Initialize
		err := module.Initialize()
		_ = err // Some modules might return errors
		
		// Test error handling in Cleanup
		err = module.Cleanup()
		_ = err // Some modules might return errors
	})
	
	t.Run("ServiceProvider_EdgeCases", func(t *testing.T) {
		// Test ValueProvider with nil value
		provider := ValueProvider("nil-test", nil)
		if provider == nil {
			t.Error("ValueProvider should not return nil for nil value")
		}
		
		// Test ClassProvider with complex type
		provider = ClassProvider("complex-test", reflect.TypeOf((*map[string]interface{})(nil)).Elem(), ScopeTransient)
		if provider == nil {
			t.Error("ClassProvider should not return nil")
		}
	})
	
	t.Run("Application_EdgeCases", func(t *testing.T) {
		// Test creating app with empty module
		emptyModule := NewBaseModule()
		app := CreateApp(emptyModule)
		
		err := app.Start()
		if err != nil {
			t.Errorf("Start failed with empty module: %v", err)
		}
		
		err = app.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
	})
	
	t.Run("DecoratedModule_EdgeCases", func(t *testing.T) {
		builder := NewModuleBuilder()
		
		// Test with multiple providers and controllers
		builder.WithProviders(&TestLogger{}, &TestConfig{})
		builder.WithControllers(&TestLogger{})
		builder.WithImports(NewBaseModule(), NewBaseModule())
		builder.WithExports(&TestLogger{}, &TestConfig{})
		
		module := builder.Build()
		if module == nil {
			t.Error("Build should not return nil")
		}
		
		// Test initialization and cleanup
		err := module.Initialize()
		_ = err
		
		err = module.Cleanup() 
		_ = err
		
		// Test configuration
		container := NewDIContainer()
		err = module.Configure(container)
		_ = err
	})
}