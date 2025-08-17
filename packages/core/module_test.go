package core

import (
	"reflect"
	"testing"
)

// Test module types
type TestModule1 struct {
	BaseModule
	Controllers []interface{} `controllers:"TestController1,TestController2"`
	Providers   []interface{} `providers:"TestService1,TestService2"`
	Imports     []interface{} `imports:"DatabaseModule,AuthModule"`
	Exports     []interface{} `exports:"TestService1"`
}

type TestModule2 struct {
	BaseModule
	Name        string        `module:"test-module-2"`
	Controllers []interface{} `controllers:"UserController"`
	Providers   []interface{} `providers:"UserService,EmailService"`
}

type TestController1 struct {
	Path       string        `controller:"/api/v1"`
	Middleware []interface{} `middleware:"auth,logging"`
	Service    *TestService1 `inject:""`
}

func (c *TestController1) GetUsers() string {
	return "users"
}

func (c *TestController1) PostUser() string {
	return "created"
}

type TestService1 struct {
	Name  string `service:"test-service-1" scope:"singleton"`
	Value string
}

func TestBaseModule_NewBaseModule(t *testing.T) {
	module := NewBaseModule()
	
	if module == nil {
		t.Fatal("NewBaseModule() returned nil")
	}
	
	if module.providers == nil {
		t.Error("providers slice not initialized")
	}
	
	if module.controllers == nil {
		t.Error("controllers slice not initialized")
	}
	
	if module.imports == nil {
		t.Error("imports slice not initialized")
	}
	
	if len(module.providers) != 0 {
		t.Error("providers slice should be empty initially")
	}
	
	if len(module.controllers) != 0 {
		t.Error("controllers slice should be empty initially")
	}
	
	if len(module.imports) != 0 {
		t.Error("imports slice should be empty initially")
	}
}

func TestBaseModule_AddProvider(t *testing.T) {
	module := NewBaseModule()
	service := &TestService1{}
	
	module.AddProvider(service)
	
	if len(module.providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(module.providers))
	}
	
	if module.providers[0] != service {
		t.Error("Provider not added correctly")
	}
}

func TestBaseModule_AddController(t *testing.T) {
	module := NewBaseModule()
	controller := &TestController1{}
	
	module.AddController(controller)
	
	if len(module.controllers) != 1 {
		t.Errorf("Expected 1 controller, got %d", len(module.controllers))
	}
	
	if module.controllers[0] != controller {
		t.Error("Controller not added correctly")
	}
}

func TestBaseModule_AddImport(t *testing.T) {
	module := NewBaseModule()
	importedModule := NewBaseModule()
	
	module.AddImport(importedModule)
	
	if len(module.imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(module.imports))
	}
	
	if module.imports[0] != importedModule {
		t.Error("Import not added correctly")
	}
}

func TestBaseModule_GetProviders(t *testing.T) {
	module := NewBaseModule()
	service1 := &TestService1{}
	service2 := &TestService1{}
	
	module.AddProvider(service1)
	module.AddProvider(service2)
	
	providers := module.GetProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
	
	if providers[0] != service1 || providers[1] != service2 {
		t.Error("GetProviders() returned incorrect providers")
	}
}

func TestBaseModule_GetControllers(t *testing.T) {
	module := NewBaseModule()
	controller1 := &TestController1{}
	controller2 := &TestController1{}
	
	module.AddController(controller1)
	module.AddController(controller2)
	
	controllers := module.GetControllers()
	if len(controllers) != 2 {
		t.Errorf("Expected 2 controllers, got %d", len(controllers))
	}
	
	if controllers[0] != controller1 || controllers[1] != controller2 {
		t.Error("GetControllers() returned incorrect controllers")
	}
}

func TestBaseModule_GetImports(t *testing.T) {
	module := NewBaseModule()
	import1 := NewBaseModule()
	import2 := NewBaseModule()
	
	module.AddImport(import1)
	module.AddImport(import2)
	
	imports := module.GetImports()
	if len(imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(imports))
	}
	
	if imports[0] != import1 || imports[1] != import2 {
		t.Error("GetImports() returned incorrect imports")
	}
}

func TestBaseModule_Configure(t *testing.T) {
	module := NewBaseModule()
	container := NewDIContainer()
	
	// Add some providers and controllers
	module.AddProvider(&TestService1{})
	module.AddController(&TestController1{})
	
	// Test configuration
	err := module.Configure(container)
	if err != nil {
		t.Errorf("Configure() failed: %v", err)
	}
	
	// The base implementation doesn't actually register services
	// This is just testing that the method doesn't error
}

func TestBaseModule_Initialize(t *testing.T) {
	module := NewBaseModule()
	
	err := module.Initialize()
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
}

func TestBaseModule_Cleanup(t *testing.T) {
	module := NewBaseModule()
	
	err := module.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
}

func TestModuleBuilder_NewModuleBuilder(t *testing.T) {
	builder := NewModuleBuilder()
	
	if builder == nil {
		t.Fatal("NewModuleBuilder() returned nil")
	}
	
	if builder.providers == nil {
		t.Error("providers slice not initialized")
	}
	
	if builder.controllers == nil {
		t.Error("controllers slice not initialized")
	}
	
	if builder.imports == nil {
		t.Error("imports slice not initialized")
	}
	
	if builder.exports == nil {
		t.Error("exports slice not initialized")
	}
}

func TestModuleBuilder_WithProviders(t *testing.T) {
	builder := NewModuleBuilder()
	service1 := &TestService1{}
	service2 := &TestService1{}
	
	result := builder.WithProviders(service1, service2)
	
	// Test fluent interface
	if result != builder {
		t.Error("WithProviders() should return the same builder instance")
	}
	
	if len(builder.providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(builder.providers))
	}
	
	if builder.providers[0] != service1 || builder.providers[1] != service2 {
		t.Error("Providers not added correctly")
	}
}

func TestModuleBuilder_WithControllers(t *testing.T) {
	builder := NewModuleBuilder()
	controller1 := &TestController1{}
	controller2 := &TestController1{}
	
	result := builder.WithControllers(controller1, controller2)
	
	// Test fluent interface
	if result != builder {
		t.Error("WithControllers() should return the same builder instance")
	}
	
	if len(builder.controllers) != 2 {
		t.Errorf("Expected 2 controllers, got %d", len(builder.controllers))
	}
	
	if builder.controllers[0] != controller1 || builder.controllers[1] != controller2 {
		t.Error("Controllers not added correctly")
	}
}

func TestModuleBuilder_WithImports(t *testing.T) {
	builder := NewModuleBuilder()
	import1 := NewBaseModule()
	import2 := NewBaseModule()
	
	result := builder.WithImports(import1, import2)
	
	// Test fluent interface
	if result != builder {
		t.Error("WithImports() should return the same builder instance")
	}
	
	if len(builder.imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(builder.imports))
	}
	
	if builder.imports[0] != import1 || builder.imports[1] != import2 {
		t.Error("Imports not added correctly")
	}
}

func TestModuleBuilder_WithExports(t *testing.T) {
	builder := NewModuleBuilder()
	export1 := &TestService1{}
	export2 := &TestService1{}
	
	result := builder.WithExports(export1, export2)
	
	// Test fluent interface
	if result != builder {
		t.Error("WithExports() should return the same builder instance")
	}
	
	if len(builder.exports) != 2 {
		t.Errorf("Expected 2 exports, got %d", len(builder.exports))
	}
	
	if builder.exports[0] != export1 || builder.exports[1] != export2 {
		t.Error("Exports not added correctly")
	}
}

func TestModuleBuilder_WithMetadata(t *testing.T) {
	builder := NewModuleBuilder()
	metadata := &ModuleMetadata{
		Name: "test-module",
	}
	
	result := builder.WithMetadata(metadata)
	
	// Test fluent interface
	if result != builder {
		t.Error("WithMetadata() should return the same builder instance")
	}
	
	if builder.metadata != metadata {
		t.Error("Metadata not set correctly")
	}
}

func TestModuleBuilder_Build(t *testing.T) {
	builder := NewModuleBuilder()
	service := &TestService1{}
	controller := &TestController1{}
	importModule := NewBaseModule()
	
	module := builder.
		WithProviders(service).
		WithControllers(controller).
		WithImports(importModule).
		WithExports(service).
		WithMetadata(&ModuleMetadata{Name: "test-module"}).
		Build()
	
	if module == nil {
		t.Fatal("Build() returned nil")
	}
	
	decoratedModule, ok := module.(*DecoratedModule)
	if !ok {
		t.Fatal("Build() did not return *DecoratedModule")
	}
	
	if len(decoratedModule.providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(decoratedModule.providers))
	}
	
	if len(decoratedModule.controllers) != 1 {
		t.Errorf("Expected 1 controller, got %d", len(decoratedModule.controllers))
	}
	
	if len(decoratedModule.imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(decoratedModule.imports))
	}
	
	if len(decoratedModule.exports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(decoratedModule.exports))
	}
	
	if decoratedModule.metadata == nil {
		t.Error("Metadata not set in built module")
	}
	
	if decoratedModule.metadata.Name != "test-module" {
		t.Errorf("Expected metadata name 'test-module', got %s", decoratedModule.metadata.Name)
	}
}

func TestDecoratedModule_Configure(t *testing.T) {
	builder := NewModuleBuilder()
	service := &TestService1{}
	controller := &TestController1{}
	
	module := builder.
		WithProviders(service).
		WithControllers(controller).
		Build()
	
	container := NewDIContainer()
	
	err := module.Configure(container)
	if err != nil {
		t.Errorf("Configure() failed: %v", err)
	}
	
	// Check that services were registered
	serviceType := reflect.TypeOf((*TestService1)(nil)).Elem()
	if _, exists := container.services[serviceType]; !exists {
		t.Error("Provider not registered in container")
	}
	
	controllerType := reflect.TypeOf((*TestController1)(nil)).Elem()
	if _, exists := container.services[controllerType]; !exists {
		t.Error("Controller not registered in container")
	}
}

func TestDecoratedModule_GetProviders(t *testing.T) {
	builder := NewModuleBuilder()
	service1 := &TestService1{}
	service2 := &TestService1{}
	
	module := builder.WithProviders(service1, service2).Build()
	
	providers := module.GetProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestDecoratedModule_GetControllers(t *testing.T) {
	builder := NewModuleBuilder()
	controller1 := &TestController1{}
	controller2 := &TestController1{}
	
	module := builder.WithControllers(controller1, controller2).Build()
	
	controllers := module.GetControllers()
	if len(controllers) != 2 {
		t.Errorf("Expected 2 controllers, got %d", len(controllers))
	}
}

func TestDecoratedModule_GetImports(t *testing.T) {
	builder := NewModuleBuilder()
	import1 := NewBaseModule()
	import2 := NewBaseModule()
	
	module := builder.WithImports(import1, import2).Build()
	
	imports := module.GetImports()
	if len(imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(imports))
	}
}

func TestDecoratedModule_Initialize(t *testing.T) {
	builder := NewModuleBuilder()
	module := builder.Build()
	
	err := module.Initialize()
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
}

func TestDecoratedModule_Cleanup(t *testing.T) {
	builder := NewModuleBuilder()
	module := builder.Build()
	
	err := module.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
}