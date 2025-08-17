// Additional test types for benchmark tests
package core

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

type ServiceB struct {
	D *ServiceD `inject:""`
	E *ServiceE `inject:""`
}

type ServiceC struct {
	F *ServiceF `inject:""`
}

type ServiceD struct {
	Logger *TestLogger `inject:""`
}

type ServiceE struct {
	Logger *TestLogger `inject:""`
}

type ServiceF struct {
	Logger *TestLogger `inject:""`
}

// Benchmark tests for performance critical operations

func BenchmarkDIContainer_RegisterProvider(b *testing.B) {
	container := NewDIContainer()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := &TestLogger{}
		container.RegisterProvider(service)
	}
}

func BenchmarkDIContainer_Initialize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		container := NewDIContainer()
		container.RegisterProvider(&TestLogger{})
		container.RegisterProvider(&TestUserService{})
		config := &TestConfig{DatabaseURL: "test://localhost"}
		container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), config)
		container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
		
		b.StartTimer()
		container.Initialize()
	}
}

func BenchmarkDIContainer_Resolve(b *testing.B) {
	container := NewDIContainer()
	container.RegisterProvider(&TestLogger{})
	container.Initialize()
	
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container.Resolve(loggerType)
	}
}

func BenchmarkDIContainer_ResolveWithDependencies(b *testing.B) {
	container := NewDIContainer()
	container.RegisterProvider(&TestLogger{})
	container.RegisterProvider(&TestUserService{})
	config := &TestConfig{DatabaseURL: "test://localhost"}
	container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), config)
	container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
	container.Initialize()
	
	userServiceType := reflect.TypeOf((*TestUserService)(nil)).Elem()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container.Resolve(userServiceType)
	}
}

func BenchmarkDIContainer_ConcurrentResolve(b *testing.B) {
	container := NewDIContainer()
	container.RegisterProvider(&TestLogger{})
	container.Initialize()
	
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			container.Resolve(loggerType)
		}
	})
}

func BenchmarkApplication_Start(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		module := &TestAppModule{}
		app := CreateApp(module)
		
		b.StartTimer()
		app.Start()
		b.StopTimer()
		
		app.Stop()
	}
}

func BenchmarkApplication_GetService(b *testing.B) {
	module := &TestAppModule{}
	app := CreateApp(module)
	app.Start()
	defer app.Stop()
	
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.GetService(loggerType)
	}
}

func BenchmarkExtractDecorators(b *testing.B) {
	service := &TestDecoratedService{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractDecorators(service)
	}
}

func BenchmarkExtractModuleMetadata(b *testing.B) {
	module := &TestDecoratedModule{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractModuleMetadata(module)
	}
}

func BenchmarkExtractAllRouteMetadata(b *testing.B) {
	controller := &TestController1{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractAllRouteMetadata(controller)
	}
}

func BenchmarkModuleBuilder_Build(b *testing.B) {
	service := &TestService1{}
	controller := &TestController1{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewModuleBuilder().
			WithProviders(service).
			WithControllers(controller).
			Build()
	}
}

// Stress tests for high-load scenarios

func TestDIContainer_HighVolumeRegistration(t *testing.T) {
	container := NewDIContainer()
	
	// Register 1000 services
	for i := 0; i < 1000; i++ {
		service := &TestLogger{}
		err := container.RegisterProvider(service)
		if err != nil {
			t.Fatalf("Failed to register service %d: %v", i, err)
		}
	}
	
	if len(container.services) != 1000 {
		t.Errorf("Expected 1000 services, got %d", len(container.services))
	}
	
	// Initialize should still work
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container with 1000 services: %v", err)
	}
}

func TestDIContainer_ConcurrentRegistration(t *testing.T) {
	container := NewDIContainer()
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	// Concurrent registration
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Create unique service type for each goroutine
			service := &TestTransientService{ID: string(rune(id))}
			serviceType := reflect.TypeOf(service).Elem()
			
			err := container.RegisterService(serviceType, func(ctx context.Context, c *DIContainer) (interface{}, error) {
				return service, nil
			}, ScopeTransient)
			
			if err != nil {
				errors <- err
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent registration error: %v", err)
	}
}

func TestDIContainer_ConcurrentResolution(t *testing.T) {
	container := NewDIContainer()
	container.RegisterProvider(&TestLogger{})
	container.Initialize()
	
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	instances := make(chan interface{}, 100)
	
	// Concurrent resolution
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			instance, err := container.Resolve(loggerType)
			if err != nil {
				errors <- err
			} else {
				instances <- instance
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	close(instances)
	
	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent resolution error: %v", err)
	}
	
	// All instances should be the same (singleton)
	var firstInstance interface{}
	instanceCount := 0
	for instance := range instances {
		if firstInstance == nil {
			firstInstance = instance
		} else if instance != firstInstance {
			t.Error("Singleton instances should be the same across concurrent resolutions")
		}
		instanceCount++
	}
	
	if instanceCount != 100 {
		t.Errorf("Expected 100 instances, got %d", instanceCount)
	}
}

func TestApplication_ConcurrentServiceAccess(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	app.Start()
	defer app.Stop()
	
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	
	var wg sync.WaitGroup
	errors := make(chan error, 50)
	
	// Concurrent service access
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			_, err := app.GetService(loggerType)
			if err != nil {
				errors <- err
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent service access error: %v", err)
	}
}

func TestDIContainer_MemoryUsage(t *testing.T) {
	container := NewDIContainer()
	
	// Register many services to test memory usage
	for i := 0; i < 10000; i++ {
		service := &TestLogger{}
		container.RegisterProvider(service)
	}
	
	container.Initialize()
	
	// Resolve all services
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	for i := 0; i < 1000; i++ {
		_, err := container.Resolve(loggerType)
		if err != nil {
			t.Fatalf("Failed to resolve service: %v", err)
		}
	}
	
	// Test should complete without excessive memory usage
	// This is more of a smoke test - actual memory profiling would be done separately
}

func TestApplication_LongRunningOperations(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Test multiple start/stop cycles
	for i := 0; i < 10; i++ {
		err := app.Start()
		if err != nil {
			t.Fatalf("Failed to start application on iteration %d: %v", i, err)
		}
		
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		
		err = app.Stop()
		if err != nil {
			t.Fatalf("Failed to stop application on iteration %d: %v", i, err)
		}
	}
}

func TestDIContainer_ComplexDependencyGraph(t *testing.T) {
	container := NewDIContainer()
	
	// Create a complex dependency graph
	type ServiceA struct {
		B *ServiceB `inject:""`
		C *ServiceC `inject:""`
	}
	
	type ServiceB struct {
		D *ServiceD `inject:""`
		E *ServiceE `inject:""`
	}
	
	type ServiceC struct {
		F *ServiceF `inject:""`
	}
	
	type ServiceD struct {
		Logger *TestLogger `inject:""`
	}
	
	type ServiceE struct {
		Logger *TestLogger `inject:""`
	}
	
	type ServiceF struct {
		Logger *TestLogger `inject:""`
	}
	
	// Register all services
	container.RegisterProvider(&TestLogger{})
	container.RegisterProvider(&ServiceF{})
	container.RegisterProvider(&ServiceE{})
	container.RegisterProvider(&ServiceD{})
	container.RegisterProvider(&ServiceC{})
	container.RegisterProvider(&ServiceB{})
	container.RegisterProvider(&ServiceA{})
	
	// Initialize should handle complex dependencies
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize complex dependency graph: %v", err)
	}
	
	// Resolve top-level service
	serviceAType := reflect.TypeOf((*ServiceA)(nil)).Elem()
	instance, err := container.Resolve(serviceAType)
	if err != nil {
		t.Fatalf("Failed to resolve ServiceA: %v", err)
	}
	
	serviceA := instance.(*ServiceA)
	if serviceA.B == nil || serviceA.C == nil {
		t.Error("Dependencies not injected correctly")
	}
	
	if serviceA.B.D == nil || serviceA.B.E == nil {
		t.Error("Nested dependencies not injected correctly")
	}
	
	if serviceA.C.F == nil {
		t.Error("Nested dependency not injected correctly")
	}
}

func TestDIContainer_ScopeLifecycle(t *testing.T) {
	container := NewDIContainer()
	
	// Register scoped service
	scopedType := reflect.TypeOf((*TestTransientService)(nil)).Elem()
	container.RegisterService(scopedType, func(ctx context.Context, c *DIContainer) (interface{}, error) {
		service := &TestTransientService{}
		service.Initialize()
		return service, nil
	}, ScopeScoped)
	
	container.Initialize()
	
	// Create multiple scopes
	scope1 := container.CreateScope("scope1")
	scope2 := container.CreateScope("scope2")
	
	// Resolve service in each scope
	instance1, err := container.resolveDependencies(context.Background(), scopedType, scope1)
	if err != nil {
		t.Fatalf("Failed to resolve in scope1: %v", err)
	}
	
	instance2, err := container.resolveDependencies(context.Background(), scopedType, scope2)
	if err != nil {
		t.Fatalf("Failed to resolve in scope2: %v", err)
	}
	
	// Instances should be different between scopes
	if instance1 == instance2 {
		t.Error("Scoped instances should be different between scopes")
	}
	
	// Resolve again in same scope should return same instance
	instance1Again, err := container.resolveDependencies(context.Background(), scopedType, scope1)
	if err != nil {
		t.Fatalf("Failed to resolve again in scope1: %v", err)
	}
	
	if instance1 != instance1Again {
		t.Error("Scoped instances should be same within the same scope")
	}
	
	// Cleanup scopes
	container.DestroyScope("scope1")
	container.DestroyScope("scope2")
}