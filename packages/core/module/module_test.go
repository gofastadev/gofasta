package module

import (
	"testing"

	"github.com/healtronlabs/gofasta/packages/core/container"
)

// Mock provider for testing
type TestProvider struct {
	Name string
}

// TestModule tests the Module implementations
func TestModule(t *testing.T) {
	// Test NewBaseModule
	t.Run("NewBaseModule", func(t *testing.T) {
		module := NewBaseModule()
		
		if module == nil {
			t.Fatal("NewBaseModule returned nil")
		}
		
		if module.providers == nil {
			t.Error("Providers slice is nil")
		}
		
		if module.controllers == nil {
			t.Error("Controllers slice is nil")
		}
		
		if module.imports == nil {
			t.Error("Imports slice is nil")
		}
	})
	
	// Test Configure
	t.Run("Configure", func(t *testing.T) {
		module := NewBaseModule()
		container := container.NewDIContainer()
		
		err := module.Configure(container)
		if err != nil {
			t.Errorf("Configure failed: %v", err)
		}
	})
	
	// Test Initialize
	t.Run("Initialize", func(t *testing.T) {
		module := NewBaseModule()
		
		err := module.Initialize()
		if err != nil {
			t.Errorf("Initialize failed: %v", err)
		}
	})
	
	// Test Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		module := NewBaseModule()
		
		err := module.Cleanup()
		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
	})
	
	// Test GetProviders
	t.Run("GetProviders", func(t *testing.T) {
		module := NewBaseModule()
		
		providers := module.GetProviders()
		if providers == nil {
			t.Error("GetProviders returned nil")
		}
		
		if len(providers) != 0 {
			t.Errorf("Expected 0 providers, got %d", len(providers))
		}
	})
	
	// Test GetControllers
	t.Run("GetControllers", func(t *testing.T) {
		module := NewBaseModule()
		
		controllers := module.GetControllers()
		if controllers == nil {
			t.Error("GetControllers returned nil")
		}
		
		if len(controllers) != 0 {
			t.Errorf("Expected 0 controllers, got %d", len(controllers))
		}
	})
	
	// Test GetImports
	t.Run("GetImports", func(t *testing.T) {
		module := NewBaseModule()
		
		imports := module.GetImports()
		if imports == nil {
			t.Error("GetImports returned nil")
		}
		
		if len(imports) != 0 {
			t.Errorf("Expected 0 imports, got %d", len(imports))
		}
	})
	
	// Test AddProvider
	t.Run("AddProvider", func(t *testing.T) {
		module := NewBaseModule()
		provider := &TestProvider{Name: "test-provider"}
		
		module.AddProvider(provider)
		
		providers := module.GetProviders()
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
		
		if providers[0] != provider {
			t.Error("Added provider does not match")
		}
	})
	
	// Test AddController
	t.Run("AddController", func(t *testing.T) {
		module := NewBaseModule()
		controller := &TestProvider{Name: "test-controller"}
		
		module.AddController(controller)
		
		controllers := module.GetControllers()
		if len(controllers) != 1 {
			t.Errorf("Expected 1 controller, got %d", len(controllers))
		}
		
		if controllers[0] != controller {
			t.Error("Added controller does not match")
		}
	})
	
	// Test AddImport
	t.Run("AddImport", func(t *testing.T) {
		module := NewBaseModule()
		importedModule := NewBaseModule()
		
		module.AddImport(importedModule)
		
		imports := module.GetImports()
		if len(imports) != 1 {
			t.Errorf("Expected 1 import, got %d", len(imports))
		}
		
		if imports[0] != importedModule {
			t.Error("Added import does not match")
		}
	})
}