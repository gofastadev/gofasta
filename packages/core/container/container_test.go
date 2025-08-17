package container

import (
	"reflect"
	"testing"
)

// Mock service for testing
type TestService struct {
	ID string
}

func NewTestService() *TestService {
	return &TestService{ID: "test-service"}
}

// TestDIContainer tests the DIContainer struct and its methods
func TestDIContainer(t *testing.T) {
	// Test NewDIContainer
	t.Run("NewDIContainer", func(t *testing.T) {
		container := NewDIContainer()

		if container == nil {
			t.Fatal("NewDIContainer returned nil")
		}

		if container.providers == nil {
			t.Error("Providers map is nil")
		}

		if container.instances == nil {
			t.Error("Instances map is nil")
		}
	})

	// Test Register
	t.Run("Register", func(t *testing.T) {
		container := NewDIContainer()
		serviceType := reflect.TypeOf((*TestService)(nil)).Elem()

		container.Register(serviceType, func() interface{} {
			return NewTestService()
		}, Singleton)

		if len(container.providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(container.providers))
		}

		provider, exists := container.providers[serviceType]
		if !exists {
			t.Error("Provider was not registered")
		}

		if provider.Type != serviceType {
			t.Error("Provider type mismatch")
		}

		if provider.Lifecycle != Singleton {
			t.Error("Provider lifecycle mismatch")
		}
	})

	// Test RegisterInstance
	t.Run("RegisterInstance", func(t *testing.T) {
		container := NewDIContainer()
		serviceType := reflect.TypeOf((*TestService)(nil)).Elem()
		instance := NewTestService()

		container.RegisterInstance(serviceType, instance)

		if len(container.providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(container.providers))
		}

		if len(container.instances) != 1 {
			t.Errorf("Expected 1 instance, got %d", len(container.instances))
		}

		provider, exists := container.providers[serviceType]
		if !exists {
			t.Error("Provider was not registered")
		}

		if provider.Instance != instance {
			t.Error("Provider instance mismatch")
		}
	})

	// Test Resolve for singleton
	t.Run("ResolveSingleton", func(t *testing.T) {
		container := NewDIContainer()
		serviceType := reflect.TypeOf((*TestService)(nil)).Elem()

		container.Register(serviceType, func() interface{} {
			return NewTestService()
		}, Singleton)

		instance1, err := container.Resolve(serviceType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		instance2, err := container.Resolve(serviceType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if instance1 == nil {
			t.Error("First instance is nil")
		}

		if instance2 == nil {
			t.Error("Second instance is nil")
		}

		// For singleton, should be the same instance
		if instance1 != instance2 {
			t.Error("Singleton instances should be the same")
		}
	})

	// Test Resolve for transient
	t.Run("ResolveTransient", func(t *testing.T) {
		container := NewDIContainer()
		serviceType := reflect.TypeOf((*TestService)(nil)).Elem()

		container.Register(serviceType, func() interface{} {
			return NewTestService()
		}, Transient)

		instance1, err := container.Resolve(serviceType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		instance2, err := container.Resolve(serviceType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if instance1 == nil {
			t.Error("First instance is nil")
		}

		if instance2 == nil {
			t.Error("Second instance is nil")
		}

		// For transient, should be different instances
		if instance1 == instance2 {
			t.Error("Transient instances should be different")
		}
	})

	// Test Resolve for registered instance
	t.Run("ResolveRegisteredInstance", func(t *testing.T) {
		container := NewDIContainer()
		serviceType := reflect.TypeOf((*TestService)(nil)).Elem()
		instance := NewTestService()

		container.RegisterInstance(serviceType, instance)

		resolvedInstance, err := container.Resolve(serviceType)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if resolvedInstance == nil {
			t.Error("Resolved instance is nil")
		}

		// Should be the same instance
		if resolvedInstance != instance {
			t.Error("Resolved instance should be the same as registered instance")
		}
	})

	// Test Resolve for unregistered service
	t.Run("ResolveUnregisteredService", func(t *testing.T) {
		container := NewDIContainer()
		serviceType := reflect.TypeOf((*TestService)(nil)).Elem()

		_, err := container.Resolve(serviceType)
		if err == nil {
			t.Error("Expected error for unregistered service")
		}
	})
}
