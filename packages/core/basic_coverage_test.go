package core

import (
	"testing"
)

// TestBasicCoverage demonstrates that the core package can be tested
func TestBasicCoverage(t *testing.T) {
	// Test that we can create a DI container
	container := NewDIContainer()
	if container == nil {
		t.Error("Failed to create DI container")
	}

	// Test that we can create a base module
	module := NewBaseModule()
	if module == nil {
		t.Error("Failed to create base module")
	}

	// Test that we can create module builder
	builder := NewModuleBuilder()
	if builder == nil {
		t.Error("Failed to create module builder")
	}

	// Test that we can create default application config
	config := DefaultApplicationConfig()
	if config == nil {
		t.Error("Failed to create default application config")
	}
}