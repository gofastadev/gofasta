package application

import (
	"testing"

	"github.com/healtronlabs/gofasta/packages/core/module"
)

// TestApplication tests the Application struct and its methods
func TestApplication(t *testing.T) {
	// Test creating application with default config
	t.Run("CreateAppWithDefaultConfig", func(t *testing.T) {
		testModule := module.NewBaseModule()
		app := CreateApp(testModule)
		
		if app == nil {
			t.Fatal("CreateApp returned nil")
		}
		
		if app.config == nil {
			t.Error("Application config is nil")
		}
		
		if app.config.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", app.config.Port)
		}
		
		if app.config.Environment != "development" {
			t.Errorf("Expected default environment 'development', got '%s'", app.config.Environment)
		}
		
		if app.config.LogLevel != "info" {
			t.Errorf("Expected default log level 'info', got '%s'", app.config.LogLevel)
		}
	})
	
	// Test creating application with custom config
	t.Run("CreateAppWithCustomConfig", func(t *testing.T) {
		testModule := module.NewBaseModule()
		customConfig := &Config{
			Port:        3000,
			Environment: "production",
			LogLevel:    "error",
		}
		app := CreateApp(testModule, customConfig)
		
		if app == nil {
			t.Fatal("CreateApp returned nil")
		}
		
		if app.config.Port != 3000 {
			t.Errorf("Expected custom port 3000, got %d", app.config.Port)
		}
		
		if app.config.Environment != "production" {
			t.Errorf("Expected custom environment 'production', got '%s'", app.config.Environment)
		}
		
		if app.config.LogLevel != "error" {
			t.Errorf("Expected custom log level 'error', got '%s'", app.config.LogLevel)
		}
	})
	
	// Test RegisterModule
	t.Run("RegisterModule", func(t *testing.T) {
		app := &Application{
			modules: make([]module.Module, 0),
		}
		
		testModule := module.NewBaseModule()
		err := app.RegisterModule(testModule)
		
		if err != nil {
			t.Errorf("RegisterModule failed: %v", err)
		}
		
		if len(app.modules) != 1 {
			t.Errorf("Expected 1 module, got %d", len(app.modules))
		}
	})
}