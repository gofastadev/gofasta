package core

import (
	"testing"
)

// Simple working tests to demonstrate functionality

func TestBasicFunctionality(t *testing.T) {
	t.Run("NewDIContainer", func(t *testing.T) {
		container := NewDIContainer()
		if container == nil {
			t.Fatal("NewDIContainer() returned nil")
		}
	})
	
	t.Run("DefaultApplicationConfig", func(t *testing.T) {
		config := DefaultApplicationConfig()
		if config == nil {
			t.Fatal("DefaultApplicationConfig() returned nil")
		}
		if config.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", config.Port)
		}
	})
	
	t.Run("NewBaseModule", func(t *testing.T) {
		module := NewBaseModule()
		if module == nil {
			t.Fatal("NewBaseModule() returned nil")
		}
	})
	
	t.Run("NewModuleBuilder", func(t *testing.T) {
		builder := NewModuleBuilder()
		if builder == nil {
			t.Fatal("NewModuleBuilder() returned nil")
		}
		
		// Test fluent interface
		module := builder.
			WithProviders(&TestLogger{}).
			WithControllers(&TestController1{}).
			Build()
		
		if module == nil {
			t.Fatal("ModuleBuilder.Build() returned nil")
		}
	})
	
	t.Run("ExtractDecorators", func(t *testing.T) {
		service := &TestDecoratedService{}
		decorators, err := ExtractDecorators(service)
		if err != nil {
			t.Fatalf("ExtractDecorators() failed: %v", err)
		}
		if decorators == nil {
			t.Fatal("ExtractDecorators() returned nil")
		}
	})
	
	t.Run("GofastaError", func(t *testing.T) {
		err := NewGofastaError("TEST_ERROR", "Test message", 400)
		if err == nil {
			t.Fatal("NewGofastaError() returned nil")
		}
		if err.Code != "TEST_ERROR" {
			t.Errorf("Expected error code 'TEST_ERROR', got %s", err.Code)
		}
		if err.Error() != "Test message" {
			t.Errorf("Expected error message 'Test message', got %s", err.Error())
		}
	})
	
	t.Run("ServiceProviders", func(t *testing.T) {
		factory := func() interface{} {
			return &TestLogger{}
		}
		
		provider := NewServiceProvider("test-logger", factory, ScopeSingleton)
		if provider == nil {
			t.Fatal("NewServiceProvider() returned nil")
		}
		
		if provider.GetName() != "test-logger" {
			t.Errorf("Expected name 'test-logger', got %s", provider.GetName())
		}
		
		instance := provider.Provide()
		if instance == nil {
			t.Fatal("Provider.Provide() returned nil")
		}
	})
}