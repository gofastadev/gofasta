package core

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// Test services for DI container testing
type TestLogger struct {
	Messages []string
	Level    string
}

func (l *TestLogger) Log(message string) {
	l.Messages = append(l.Messages, message)
}

func (l *TestLogger) Initialize() error {
	l.Level = "INFO"
	l.Messages = make([]string, 0)
	return nil
}

func (l *TestLogger) Cleanup() error {
	l.Messages = nil
	return nil
}

type TestUserService struct {
	Logger *TestLogger `inject:""`
	Config *TestConfig `inject:"test-config"`
}

func (s *TestUserService) Initialize() error {
	if s.Logger != nil {
		s.Logger.Log("UserService initialized")
	}
	return nil
}

func (s *TestUserService) GetUser(id string) string {
	if s.Logger != nil {
		s.Logger.Log("Getting user: " + id)
	}
	return "User-" + id
}

func (s *TestUserService) Cleanup() error {
	if s.Logger != nil {
		s.Logger.Log("UserService cleanup")
	}
	return nil
}

type TestConfig struct {
	DatabaseURL string
	APIKey      string
}

type TestTransientService struct {
	ID string
}

func (s *TestTransientService) Initialize() error {
	s.ID = "transient-" + time.Now().Format("20060102150405")
	return nil
}

// Circular dependency test services
type TestServiceA struct {
	ServiceB *TestServiceB `inject:""`
}

type TestServiceB struct {
	ServiceC *TestServiceC `inject:""`
}

type TestServiceC struct {
	ServiceA *TestServiceA `inject:""`
}

func TestNewDIContainer(t *testing.T) {
	container := NewDIContainer()
	
	if container == nil {
		t.Fatal("NewDIContainer() returned nil")
	}
	
	if container.services == nil {
		t.Error("services map not initialized")
	}
	
	if container.namedServices == nil {
		t.Error("namedServices map not initialized")
	}
	
	if container.instances == nil {
		t.Error("instances map not initialized")
	}
	
	if container.scopedContexts == nil {
		t.Error("scopedContexts map not initialized")
	}
	
	if container.lifecycleHooks == nil {
		t.Error("lifecycleHooks map not initialized")
	}
	
	if container.dependencyGraph == nil {
		t.Error("dependencyGraph map not initialized")
	}
	
	if container.initialized {
		t.Error("container should not be initialized by default")
	}
}

func TestDIContainer_RegisterProvider(t *testing.T) {
	container := NewDIContainer()
	
	// Test registering a provider
	err := container.RegisterProvider(&TestLogger{})
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}
	
	// Test that the service is registered
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	if _, exists := container.services[loggerType]; !exists {
		t.Error("Provider not registered in services map")
	}
	
	// Test registering after initialization should fail
	container.initialized = true
	err = container.RegisterProvider(&TestUserService{})
	if err == nil {
		t.Error("Expected error when registering provider after initialization")
	}
}

func TestDIContainer_RegisterController(t *testing.T) {
	container := NewDIContainer()
	
	type TestController struct {
		UserService *TestUserService `inject:""`
	}
	
	// Test registering a controller
	err := container.RegisterController(&TestController{})
	if err != nil {
		t.Fatalf("Failed to register controller: %v", err)
	}
	
	// Test that the controller is registered
	controllerType := reflect.TypeOf((*TestController)(nil)).Elem()
	if _, exists := container.services[controllerType]; !exists {
		t.Error("Controller not registered in services map")
	}
	
	// Test registering after initialization should fail
	container.initialized = true
	err = container.RegisterController(&TestController{})
	if err == nil {
		t.Error("Expected error when registering controller after initialization")
	}
}

func TestDIContainer_RegisterService(t *testing.T) {
	container := NewDIContainer()
	
	serviceType := reflect.TypeOf((*TestConfig)(nil)).Elem()
	factory := func(ctx context.Context, c *DIContainer) (interface{}, error) {
		return &TestConfig{
			DatabaseURL: "test://localhost",
			APIKey:      "test-key",
		}, nil
	}
	
	// Test registering a service
	err := container.RegisterService(serviceType, factory, ScopeSingleton, "test-config")
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}
	
	// Test that the service is registered
	if _, exists := container.services[serviceType]; !exists {
		t.Error("Service not registered in services map")
	}
	
	// Test that named service is registered
	if _, exists := container.namedServices["test-config"]; !exists {
		t.Error("Named service not registered")
	}
}

func TestDIContainer_RegisterInstance(t *testing.T) {
	container := NewDIContainer()
	
	config := &TestConfig{
		DatabaseURL: "test://localhost",
		APIKey:      "test-key",
	}
	
	serviceType := reflect.TypeOf((*TestConfig)(nil)).Elem()
	
	// Test registering an instance
	err := container.RegisterInstance(serviceType, config)
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}
	
	// Test that the service is registered
	if _, exists := container.services[serviceType]; !exists {
		t.Error("Instance not registered in services map")
	}
	
	// Test that the instance is stored
	if instance, exists := container.instances[serviceType]; !exists || instance != config {
		t.Error("Instance not stored correctly")
	}
}

func TestDIContainer_Initialize(t *testing.T) {
	container := NewDIContainer()
	
	// Register services
	container.RegisterProvider(&TestLogger{})
	container.RegisterProvider(&TestUserService{})
	
	config := &TestConfig{
		DatabaseURL: "test://localhost",
		APIKey:      "test-key",
	}
	container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), config)
	container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
	
	// Test initialization
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	if !container.initialized {
		t.Error("Container not marked as initialized")
	}
	
	// Test double initialization should not fail
	err = container.Initialize()
	if err != nil {
		t.Error("Double initialization should not fail")
	}
}

func TestDIContainer_CircularDependencyDetection(t *testing.T) {
	container := NewDIContainer()
	
	// Register services with circular dependencies
	container.RegisterProvider(&TestServiceA{})
	container.RegisterProvider(&TestServiceB{})
	container.RegisterProvider(&TestServiceC{})
	
	// Test that initialization fails due to circular dependency
	err := container.Initialize()
	if err == nil {
		t.Error("Expected error due to circular dependency")
	}
	
	if !container.initialized {
		t.Log("Container correctly not initialized due to circular dependency")
	}
}

func TestDIContainer_Resolve(t *testing.T) {
	container := NewDIContainer()
	
	// Register and initialize services
	container.RegisterProvider(&TestLogger{})
	container.RegisterProvider(&TestUserService{})
	
	config := &TestConfig{
		DatabaseURL: "test://localhost",
		APIKey:      "test-key",
	}
	container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), config)
	container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
	
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	// Test resolving a service
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	instance, err := container.Resolve(loggerType)
	if err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}
	
	logger, ok := instance.(*TestLogger)
	if !ok {
		t.Error("Resolved instance is not of correct type")
	}
	
	if logger.Level != "INFO" {
		t.Error("Service not properly initialized")
	}
	
	// Test resolving service with dependencies
	userServiceType := reflect.TypeOf((*TestUserService)(nil)).Elem()
	instance, err = container.Resolve(userServiceType)
	if err != nil {
		t.Fatalf("Failed to resolve service with dependencies: %v", err)
	}
	
	userService, ok := instance.(*TestUserService)
	if !ok {
		t.Error("Resolved instance is not of correct type")
	}
	
	if userService.Logger == nil {
		t.Error("Dependency not injected")
	}
	
	if userService.Config == nil {
		t.Error("Named dependency not injected")
	}
}

func TestDIContainer_ResolveNamed(t *testing.T) {
	container := NewDIContainer()
	
	config := &TestConfig{
		DatabaseURL: "test://localhost",
		APIKey:      "test-key",
	}
	container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), config)
	container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
	
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	// Test resolving named service
	instance, err := container.ResolveNamed("test-config")
	if err != nil {
		t.Fatalf("Failed to resolve named service: %v", err)
	}
	
	resolvedConfig, ok := instance.(*TestConfig)
	if !ok {
		t.Error("Resolved instance is not of correct type")
	}
	
	if resolvedConfig.DatabaseURL != "test://localhost" {
		t.Error("Named service not resolved correctly")
	}
	
	// Test resolving non-existent named service
	_, err = container.ResolveNamed("non-existent")
	if err == nil {
		t.Error("Expected error when resolving non-existent named service")
	}
}

func TestDIContainer_ServiceScopes(t *testing.T) {
	container := NewDIContainer()
	
	// Register transient service
	transientType := reflect.TypeOf((*TestTransientService)(nil)).Elem()
	container.RegisterService(transientType, func(ctx context.Context, c *DIContainer) (interface{}, error) {
		service := &TestTransientService{}
		service.Initialize()
		return service, nil
	}, ScopeTransient)
	
	// Register singleton service
	container.RegisterProvider(&TestLogger{})
	
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	// Test singleton behavior
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	instance1, _ := container.Resolve(loggerType)
	instance2, _ := container.Resolve(loggerType)
	
	if instance1 != instance2 {
		t.Error("Singleton services should return the same instance")
	}
	
	// Test transient behavior
	transient1, _ := container.Resolve(transientType)
	transient2, _ := container.Resolve(transientType)
	
	if transient1 == transient2 {
		t.Error("Transient services should return different instances")
	}
	
	service1 := transient1.(*TestTransientService)
	service2 := transient2.(*TestTransientService)
	
	if service1.ID == service2.ID {
		t.Error("Transient services should have different IDs")
	}
}

func TestDIContainer_ScopedContext(t *testing.T) {
	container := NewDIContainer()
	
	// Test creating scoped context
	scope := container.CreateScope("test-scope")
	if scope == nil {
		t.Error("Failed to create scoped context")
	}
	
	// Test that scope is stored
	if storedScope, exists := container.scopedContexts["test-scope"]; !exists || storedScope != scope {
		t.Error("Scoped context not stored correctly")
	}
	
	// Test destroying scope
	err := container.DestroyScope("test-scope")
	if err != nil {
		t.Errorf("Failed to destroy scope: %v", err)
	}
	
	// Test that scope is removed
	if _, exists := container.scopedContexts["test-scope"]; exists {
		t.Error("Scoped context not removed after destruction")
	}
	
	// Test destroying non-existent scope should not error
	err = container.DestroyScope("non-existent")
	if err != nil {
		t.Error("Destroying non-existent scope should not error")
	}
}

func TestDIContainer_LifecycleHooks(t *testing.T) {
	container := NewDIContainer()
	
	var hookCalled bool
	var hookInstance interface{}
	
	// Add lifecycle hook
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	container.AddLifecycleHook(loggerType, PhaseAfterCreate, func(instance interface{}) error {
		hookCalled = true
		hookInstance = instance
		return nil
	})
	
	// Register and initialize service
	container.RegisterProvider(&TestLogger{})
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	// Resolve service to trigger hook
	instance, err := container.Resolve(loggerType)
	if err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}
	
	if !hookCalled {
		t.Error("Lifecycle hook not called")
	}
	
	if hookInstance != instance {
		t.Error("Lifecycle hook called with wrong instance")
	}
}

func TestDIContainer_Shutdown(t *testing.T) {
	container := NewDIContainer()
	
	// Register services
	container.RegisterProvider(&TestLogger{})
	container.RegisterProvider(&TestUserService{})
	
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	// Create a scope
	container.CreateScope("test-scope")
	
	// Test shutdown
	err = container.Shutdown(1 * time.Second)
	if err != nil {
		t.Errorf("Failed to shutdown container: %v", err)
	}
	
	if container.initialized {
		t.Error("Container should not be initialized after shutdown")
	}
	
	// Test that scopes are cleaned up
	if len(container.scopedContexts) > 0 {
		t.Error("Scoped contexts not cleaned up during shutdown")
	}
}

func TestDIContainer_ResolveBeforeInitialization(t *testing.T) {
	container := NewDIContainer()
	
	// Register service
	container.RegisterProvider(&TestLogger{})
	
	// Try to resolve before initialization
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	_, err := container.Resolve(loggerType)
	if err == nil {
		t.Error("Expected error when resolving before initialization")
	}
}

func TestDIContainer_ConcurrentAccess(t *testing.T) {
	container := NewDIContainer()
	
	// Register services
	container.RegisterProvider(&TestLogger{})
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}
	
	// Test concurrent resolution
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			instance, err := container.Resolve(loggerType)
			if err != nil {
				t.Errorf("Failed to resolve service concurrently: %v", err)
				return
			}
			
			if instance == nil {
				t.Error("Resolved nil instance")
			}
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNewScopedContext(t *testing.T) {
	scope := NewScopedContext()
	
	if scope == nil {
		t.Fatal("NewScopedContext() returned nil")
	}
	
	if scope.instances == nil {
		t.Error("instances map not initialized")
	}
}

func TestServiceDescriptor(t *testing.T) {
	serviceType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	factory := func(ctx context.Context, container *DIContainer) (interface{}, error) {
		return &TestLogger{}, nil
	}
	
	descriptor := &ServiceDescriptor{
		ServiceType:  serviceType,
		Factory:      factory,
		Scope:        ScopeSingleton,
		Dependencies: []reflect.Type{},
		Name:         "test-logger",
	}
	
	if descriptor.ServiceType != serviceType {
		t.Error("ServiceType not set correctly")
	}
	
	if descriptor.Scope != ScopeSingleton {
		t.Error("Scope not set correctly")
	}
	
	if descriptor.Name != "test-logger" {
		t.Error("Name not set correctly")
	}
	
	// Test factory function
	container := NewDIContainer()
	instance, err := descriptor.Factory(context.Background(), container)
	if err != nil {
		t.Errorf("Factory function failed: %v", err)
	}
	
	if _, ok := instance.(*TestLogger); !ok {
		t.Error("Factory function returned wrong type")
	}
}