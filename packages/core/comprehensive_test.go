package core

import (
	"context"
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
		container.Initialize()
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		service, err := container.ResolveWithContext(context.Background(), loggerType)
		if err != nil {
			t.Fatalf("ResolveWithContext failed: %v", err)
		}
		if service == nil {
			t.Error("ResolveWithContext returned nil service")
		}
	})
	
	t.Run("ResolveNamed", func(t *testing.T) {
		container := NewDIContainer()
		config := &TestConfig{DatabaseURL: "test://localhost", APIKey: "test-key"}
		configType := reflect.TypeOf((*TestConfig)(nil)).Elem()
		
		container.RegisterInstance(configType, config)
		container.namedServices["test-config"] = container.services[configType]
		container.Initialize()
		
		service, err := container.ResolveNamed("test-config")
		if err != nil {
			t.Fatalf("ResolveNamed failed: %v", err)
		}
		if service == nil {
			t.Error("ResolveNamed returned nil service")
		}
	})
	
	t.Run("CreateScope", func(t *testing.T) {
		container := NewDIContainer()
		scope := container.CreateScope("test-scope")
		if scope == nil {
			t.Error("CreateScope returned nil")
		}
		if container.scopedContexts["test-scope"] != scope {
			t.Error("Scope not stored correctly")
		}
	})
	
	t.Run("DestroyScope", func(t *testing.T) {
		container := NewDIContainer()
		container.CreateScope("test-scope")
		
		err := container.DestroyScope("test-scope")
		if err != nil {
			t.Fatalf("DestroyScope failed: %v", err)
		}
		if _, exists := container.scopedContexts["test-scope"]; exists {
			t.Error("Scope not destroyed")
		}
	})
	
	t.Run("Shutdown", func(t *testing.T) {
		container := NewDIContainer()
		container.RegisterProvider(&TestLogger{})
		container.Initialize()
		
		err := container.Shutdown(5 * time.Second)
		if err != nil {
			t.Fatalf("Shutdown failed: %v", err)
		}
		if container.initialized {
			t.Error("Container should not be initialized after shutdown")
		}
	})
}

// Test all Application functions that have 0% coverage
func TestApplication_ComprehensiveCoverage(t *testing.T) {
	t.Run("GetService", func(t *testing.T) {
		module := NewBaseModule()
		module.AddProvider(&TestLogger{})
		app := CreateApp(module)
		app.Start()
		defer app.Stop()
		
		loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
		service, err := app.GetService(loggerType)
		if err != nil {
			t.Fatalf("GetService failed: %v", err)
		}
		if service == nil {
			t.Error("GetService returned nil")
		}
	})
	
	t.Run("UseGlobalPipes", func(t *testing.T) {
		module := NewBaseModule()
		app := CreateApp(module)
		
		err := app.UseGlobalPipes()
		if err != nil {
			t.Fatalf("UseGlobalPipes failed: %v", err)
		}
	})
	
	t.Run("UseGlobalGuards", func(t *testing.T) {
		module := NewBaseModule()
		app := CreateApp(module)
		
		err := app.UseGlobalGuards()
		if err != nil {
			t.Fatalf("UseGlobalGuards failed: %v", err)
		}
	})
	
	t.Run("UseGlobalInterceptors", func(t *testing.T) {
		module := NewBaseModule()
		app := CreateApp(module)
		
		err := app.UseGlobalInterceptors()
		if err != nil {
			t.Fatalf("UseGlobalInterceptors failed: %v", err)
		}
	})
	
	t.Run("UseGlobalFilters", func(t *testing.T) {
		module := NewBaseModule()
		app := CreateApp(module)
		
		err := app.UseGlobalFilters()
		if err != nil {
			t.Fatalf("UseGlobalFilters failed: %v", err)
		}
	})
	
	t.Run("CreateAndDestroyScope", func(t *testing.T) {
		module := NewBaseModule()
		app := CreateApp(module)
		app.Start()
		defer app.Stop()
		
		scope := app.CreateScope("test-scope")
		if scope == nil {
			t.Error("CreateScope returned nil")
		}
		
		err := app.DestroyScope("test-scope")
		if err != nil {
			t.Fatalf("DestroyScope failed: %v", err)
		}
	})
	
	t.Run("GetConfig", func(t *testing.T) {
		config := &ApplicationConfig{
			Port:        8080,
			Environment: "test",
			LogLevel:    "debug",
		}
		module := NewBaseModule()
		app := CreateApp(module, config)
		
		retrievedConfig := app.GetConfig()
		if retrievedConfig != config {
			t.Error("GetConfig returned wrong config")
		}
	})
	
	t.Run("GetContext", func(t *testing.T) {
		module := NewBaseModule()
		app := CreateApp(module)
		app.Start()
		defer app.Stop()
		
		ctx := app.GetContext()
		if ctx == nil {
			t.Error("GetContext returned nil")
		}
	})
}

// Test all Module functions that have 0% coverage
func TestModule_ComprehensiveCoverage(t *testing.T) {
	t.Run("BaseModule_Configure", func(t *testing.T) {
		module := NewBaseModule()
		container := NewDIContainer()
		
		err := module.Configure(container)
		if err != nil {
			t.Fatalf("Configure failed: %v", err)
		}
	})
	
	t.Run("BaseModule_GetProviders", func(t *testing.T) {
		module := NewBaseModule()
		service := &TestLogger{}
		module.AddProvider(service)
		
		providers := module.GetProviders()
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
		if providers[0] != service {
			t.Error("GetProviders returned wrong provider")
		}
	})
	
	t.Run("BaseModule_GetControllers", func(t *testing.T) {
		module := NewBaseModule()
		controller := &TestController1{}
		module.AddController(controller)
		
		controllers := module.GetControllers()
		if len(controllers) != 1 {
			t.Errorf("Expected 1 controller, got %d", len(controllers))
		}
		if controllers[0] != controller {
			t.Error("GetControllers returned wrong controller")
		}
	})
	
	t.Run("BaseModule_GetImports", func(t *testing.T) {
		module := NewBaseModule()
		importedModule := NewBaseModule()
		module.AddImport(importedModule)
		
		imports := module.GetImports()
		if len(imports) != 1 {
			t.Errorf("Expected 1 import, got %d", len(imports))
		}
		if imports[0] != importedModule {
			t.Error("GetImports returned wrong import")
		}
	})
	
	t.Run("BaseModule_AddProvider", func(t *testing.T) {
		module := NewBaseModule()
		service := &TestLogger{}
		
		module.AddProvider(service)
		
		if len(module.providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(module.providers))
		}
		if module.providers[0] != service {
			t.Error("AddProvider didn't add provider correctly")
		}
	})
	
	t.Run("BaseModule_AddController", func(t *testing.T) {
		module := NewBaseModule()
		controller := &TestController1{}
		
		module.AddController(controller)
		
		if len(module.controllers) != 1 {
			t.Errorf("Expected 1 controller, got %d", len(module.controllers))
		}
		if module.controllers[0] != controller {
			t.Error("AddController didn't add controller correctly")
		}
	})
	
	t.Run("BaseModule_AddImport", func(t *testing.T) {
		module := NewBaseModule()
		importedModule := NewBaseModule()
		
		module.AddImport(importedModule)
		
		if len(module.imports) != 1 {
			t.Errorf("Expected 1 import, got %d", len(module.imports))
		}
		if module.imports[0] != importedModule {
			t.Error("AddImport didn't add import correctly")
		}
	})
	
	t.Run("BaseModule_SetAndGetMetadata", func(t *testing.T) {
		module := NewBaseModule()
		metadata := &ModuleMetadata{
			Name: "test-module",
		}
		
		module.SetMetadata(metadata)
		
		if module.metadata == nil {
			t.Error("SetMetadata didn't set metadata")
		}
		if module.metadata.Name != "test-module" {
			t.Error("SetMetadata didn't set metadata correctly")
		}
		
		retrievedMetadata := module.GetMetadata()
		if retrievedMetadata == nil {
			t.Error("GetMetadata returned nil")
		}
		if retrievedMetadata.Name != "test-module" {
			t.Error("GetMetadata returned wrong metadata")
		}
	})
}

// Test DecoratedModule functions using ModuleBuilder
func TestDecoratedModule_ComprehensiveCoverage(t *testing.T) {
	t.Run("DecoratedModule_FromBuilder", func(t *testing.T) {
		testModule := NewModuleBuilder().
			WithProviders(&TestLogger{}).
			WithControllers(&TestController1{}).
			Build()
		
		if testModule == nil {
			t.Error("ModuleBuilder.Build() returned nil")
		}
		
		// Test Configure
		container := NewDIContainer()
		err := testModule.Configure(container)
		if err != nil {
			t.Fatalf("DecoratedModule Configure failed: %v", err)
		}
		
		// Test Initialize
		err = testModule.Initialize()
		if err != nil {
			t.Fatalf("DecoratedModule Initialize failed: %v", err)
		}
		
		// Test Cleanup
		err = testModule.Cleanup()
		if err != nil {
			t.Fatalf("DecoratedModule Cleanup failed: %v", err)
		}
		
		// Test GetProviders
		providers := testModule.GetProviders()
		if providers == nil {
			t.Error("DecoratedModule GetProviders returned nil")
		}
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
		
		// Test GetControllers
		controllers := testModule.GetControllers()
		if controllers == nil {
			t.Error("DecoratedModule GetControllers returned nil")
		}
		if len(controllers) != 1 {
			t.Errorf("Expected 1 controller, got %d", len(controllers))
		}
		
		// Test GetImports
		imports := testModule.GetImports()
		if imports == nil {
			t.Error("DecoratedModule GetImports returned nil")
		}
	})
}