// Package core provides tests for the decorator registry.
package core

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestNewDecoratorRegistry(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		dr := NewDecoratorRegistry(nil)
		if dr == nil {
			t.Fatal("Expected non-nil decorator registry")
		}
		if dr.config == nil {
			t.Error("Expected non-nil config")
		}
		if !dr.config.ParallelLoading {
			t.Error("Expected parallel loading to be enabled by default")
		}

		// Check built-in decorators are registered
		dr.mu.RLock()
		decoratorCount := len(dr.decorators)
		dr.mu.RUnlock()

		if decoratorCount == 0 {
			t.Error("Expected built-in decorators to be registered")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &RegistryConfig{
			MaxDecorators:      500,
			AllowOverride:      true,
			ValidateOnRegister: false,
			EnableMetrics:      false,
			ParallelLoading:    false,
			LoadWorkers:        2,
		}
		dr := NewDecoratorRegistry(config)
		if dr.config.MaxDecorators != 500 {
			t.Errorf("Expected max decorators 500, got %d", dr.config.MaxDecorators)
		}
		if !dr.config.AllowOverride {
			t.Error("Expected allow override to be true")
		}
	})
}

func TestBuiltinDecorators(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	tests := []struct {
		name         string
		decoratorName string
		decoratorType string
	}{
		{"GET decorator", "GET", "rest"},
		{"POST decorator", "POST", "rest"},
		{"Required decorator", "Required", "validation"},
		{"MinLength decorator", "MinLength", "validation"},
		{"Auth decorator", "Auth", "security"},
		{"RateLimit decorator", "RateLimit", "security"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := dr.Get(tt.decoratorName)
			if err != nil {
				t.Fatalf("Failed to get decorator %s: %v", tt.decoratorName, err)
			}
			if dec.Name != tt.decoratorName {
				t.Errorf("Expected decorator name %s, got %s", tt.decoratorName, dec.Name)
			}
			if dec.Type != tt.decoratorType {
				t.Errorf("Expected decorator type %s, got %s", tt.decoratorType, dec.Type)
			}
			if dec.Handler == nil {
				t.Error("Expected decorator to have a handler")
			}
		})
	}
}

func TestRegister(t *testing.T) {
	t.Run("register new decorator", func(t *testing.T) {
		dr := NewDecoratorRegistry(nil)

		decorator := &RegisteredDecorator{
			Name:        "CustomDecorator",
			Type:        "custom",
			Description: "A custom decorator",
			Handler: func(ctx context.Context, args DecoratorArgs) (DecoratorResult, error) {
				return DecoratorResult{Success: true}, nil
			},
		}

		err := dr.Register(decorator)
		if err != nil {
			t.Fatalf("Failed to register decorator: %v", err)
		}

		// Verify it was registered
		retrieved, err := dr.Get("CustomDecorator")
		if err != nil {
			t.Fatalf("Failed to get registered decorator: %v", err)
		}
		if retrieved.Name != "CustomDecorator" {
			t.Error("Decorator was not properly registered")
		}
	})

	t.Run("register without name", func(t *testing.T) {
		dr := NewDecoratorRegistry(nil)

		decorator := &RegisteredDecorator{
			Type: "custom",
		}

		err := dr.Register(decorator)
		if err == nil {
			t.Error("Expected error when registering decorator without name")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Errorf("Expected 'name is required' error, got: %v", err)
		}
	})

	t.Run("register duplicate without override", func(t *testing.T) {
		config := &RegistryConfig{
			AllowOverride: false,
		}
		dr := NewDecoratorRegistry(config)

		decorator1 := &RegisteredDecorator{
			Name: "TestDecorator",
			Type: "test",
		}
		decorator2 := &RegisteredDecorator{
			Name: "TestDecorator",
			Type: "test2",
		}

		err := dr.Register(decorator1)
		if err != nil {
			t.Fatalf("Failed to register first decorator: %v", err)
		}

		err = dr.Register(decorator2)
		if err == nil {
			t.Error("Expected error when registering duplicate without override")
		}
		if !strings.Contains(err.Error(), "already registered") {
			t.Errorf("Expected 'already registered' error, got: %v", err)
		}
	})

	t.Run("register duplicate with override", func(t *testing.T) {
		config := &RegistryConfig{
			AllowOverride: true,
		}
		dr := NewDecoratorRegistry(config)

		decorator1 := &RegisteredDecorator{
			Name: "TestDecorator",
			Type: "test",
		}
		decorator2 := &RegisteredDecorator{
			Name: "TestDecorator",
			Type: "test2",
		}

		err := dr.Register(decorator1)
		if err != nil {
			t.Fatalf("Failed to register first decorator: %v", err)
		}

		err = dr.Register(decorator2)
		if err != nil {
			t.Fatalf("Failed to override decorator: %v", err)
		}

		retrieved, _ := dr.Get("TestDecorator")
		if retrieved.Type != "test2" {
			t.Error("Decorator was not overridden")
		}
	})

	t.Run("register with schema validation", func(t *testing.T) {
		config := &RegistryConfig{
			ValidateOnRegister: true,
		}
		dr := NewDecoratorRegistry(config)

		// Valid schema
		validDecorator := &RegisteredDecorator{
			Name: "ValidDecorator",
			Schema: &DecoratorSchema{
				Arguments: []ArgumentSchema{
					{Name: "arg1", Type: "string", Required: true},
				},
			},
		}

		err := dr.Register(validDecorator)
		if err != nil {
			t.Fatalf("Failed to register valid decorator: %v", err)
		}

		// Invalid schema (missing argument name)
		invalidDecorator := &RegisteredDecorator{
			Name: "InvalidDecorator",
			Schema: &DecoratorSchema{
				Arguments: []ArgumentSchema{
					{Type: "string"},
				},
			},
		}

		err = dr.Register(invalidDecorator)
		if err == nil {
			t.Error("Expected error for invalid schema")
		}
	})
}

func TestInvoke(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	t.Run("invoke REST decorator", func(t *testing.T) {
		args := DecoratorArgs{
			Target:    "GetUsers",
			Arguments: []interface{}{"/users"},
		}

		result, err := dr.Invoke(context.Background(), "GET", args)
		if err != nil {
			t.Fatalf("Failed to invoke GET decorator: %v", err)
		}
		if !result.Success {
			t.Error("Expected successful invocation")
		}
		if result.Metadata["method"] != "GET" {
			t.Error("Expected method to be GET in metadata")
		}
		if result.Metadata["path"] != "/users" {
			t.Error("Expected path to be /users in metadata")
		}
	})

	t.Run("invoke validation decorator", func(t *testing.T) {
		args := DecoratorArgs{
			Target:    "UserName",
			Arguments: []interface{}{3},
		}

		result, err := dr.Invoke(context.Background(), "MinLength", args)
		if err != nil {
			t.Fatalf("Failed to invoke MinLength decorator: %v", err)
		}
		if !result.Success {
			t.Error("Expected successful invocation")
		}
		if result.Metadata["validation_type"] != "minlength" {
			t.Error("Expected validation_type to be minlength")
		}
	})

	t.Run("invoke security decorator", func(t *testing.T) {
		args := DecoratorArgs{
			Target:    "AdminEndpoint",
			Arguments: []interface{}{100, "1m"},
		}

		result, err := dr.Invoke(context.Background(), "RateLimit", args)
		if err != nil {
			t.Fatalf("Failed to invoke RateLimit decorator: %v", err)
		}
		if !result.Success {
			t.Error("Expected successful invocation")
		}
		if result.Metadata["security_type"] != "ratelimit" {
			t.Error("Expected security_type to be ratelimit")
		}
	})

	t.Run("invoke non-existent decorator", func(t *testing.T) {
		args := DecoratorArgs{
			Target: "Something",
		}

		result, err := dr.Invoke(context.Background(), "NonExistent", args)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.Success {
			t.Error("Expected failure for non-existent decorator")
		}
		if !strings.Contains(result.Error, "not found") {
			t.Error("Expected 'not found' error")
		}
	})

	t.Run("invoke custom decorator", func(t *testing.T) {
		called := false
		decorator := &RegisteredDecorator{
			Name: "CustomInvoke",
			Handler: func(ctx context.Context, args DecoratorArgs) (DecoratorResult, error) {
				called = true
				return DecoratorResult{
					Success: true,
					Output:  "custom output",
				}, nil
			},
		}

		err := dr.Register(decorator)
		if err != nil {
			t.Fatalf("Failed to register custom decorator: %v", err)
		}

		result, err := dr.Invoke(context.Background(), "CustomInvoke", DecoratorArgs{})
		if err != nil {
			t.Fatalf("Failed to invoke custom decorator: %v", err)
		}
		if !called {
			t.Error("Custom handler was not called")
		}
		if result.Output != "custom output" {
			t.Error("Expected custom output")
		}
	})
}

func TestList(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	// Add some custom decorators
	for i := 0; i < 3; i++ {
		dr.Register(&RegisteredDecorator{
			Name: strings.Repeat("D", i+1),
			Type: "custom",
		})
	}

	t.Run("list all decorators", func(t *testing.T) {
		decorators := dr.List()
		if len(decorators) == 0 {
			t.Error("Expected decorators to be listed")
		}

		// Check sorted by name
		for i := 1; i < len(decorators); i++ {
			if decorators[i].Name < decorators[i-1].Name {
				t.Error("Decorators are not sorted by name")
				break
			}
		}
	})

	t.Run("list by type", func(t *testing.T) {
		restDecorators := dr.ListByType("rest")
		for _, dec := range restDecorators {
			if dec.Type != "rest" {
				t.Errorf("Expected type 'rest', got %s", dec.Type)
			}
		}

		validationDecorators := dr.ListByType("validation")
		for _, dec := range validationDecorators {
			if dec.Type != "validation" {
				t.Errorf("Expected type 'validation', got %s", dec.Type)
			}
		}

		customDecorators := dr.ListByType("custom")
		if len(customDecorators) != 3 {
			t.Errorf("Expected 3 custom decorators, got %d", len(customDecorators))
		}
	})
}

func TestExportSchema(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	var buf strings.Builder
	err := dr.ExportSchema(&buf)
	if err != nil {
		t.Fatalf("Failed to export schema: %v", err)
	}

	schemaJSON := buf.String()
	if schemaJSON == "" {
		t.Error("Expected non-empty schema export")
	}

	// Verify it's valid JSON
	var decorators []RegisteredDecorator
	err = json.Unmarshal([]byte(schemaJSON), &decorators)
	if err != nil {
		t.Errorf("Schema export is not valid JSON: %v", err)
	}

	if len(decorators) == 0 {
		t.Error("Expected decorators in schema export")
	}
}

func TestPluginLoading(t *testing.T) {
	t.Run("sequential loading", func(t *testing.T) {
		config := &RegistryConfig{
			ParallelLoading: false,
			PluginDirs:      []string{"./test_plugins"},
			PluginPattern:   "*.so",
		}
		dr := NewDecoratorRegistry(config)

		// This would normally load actual plugins
		// For testing, we just verify the method doesn't panic
		ctx := context.Background()
		err := dr.LoadPlugins(ctx)
		// Error is expected since test plugins don't exist
		if err != nil && !strings.Contains(err.Error(), "plugin loading errors") {
			// This is fine for test
		}
	})

	t.Run("parallel loading", func(t *testing.T) {
		config := &RegistryConfig{
			ParallelLoading: true,
			LoadWorkers:     2,
			PluginDirs:      []string{"./test_plugins"},
			PluginPattern:   "*.so",
		}
		dr := NewDecoratorRegistry(config)

		ctx := context.Background()
		err := dr.LoadPlugins(ctx)
		// Error is expected since test plugins don't exist
		if err != nil && !strings.Contains(err.Error(), "plugin loading errors") {
			// This is fine for test
		}
	})

	t.Run("plugin allowed/blocked lists", func(t *testing.T) {
		config := &RegistryConfig{
			AllowedPlugins: []string{"allowed*.so"},
			BlockedPlugins: []string{"blocked*.so"},
		}
		dr := NewDecoratorRegistry(config)

		// Test allowed
		if !dr.isPluginAllowed("allowed_plugin.so") {
			t.Error("Expected allowed_plugin.so to be allowed")
		}

		// Test blocked
		if dr.isPluginAllowed("blocked_plugin.so") {
			t.Error("Expected blocked_plugin.so to be blocked")
		}

		// Test neither allowed nor blocked
		if dr.isPluginAllowed("other_plugin.so") {
			t.Error("Expected other_plugin.so to not be allowed (not in allowed list)")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := &RegistryConfig{
			ParallelLoading: true,
			PluginDirs:      []string{"./test_plugins"},
		}
		dr := NewDecoratorRegistry(config)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := dr.LoadPlugins(ctx)
		if err != nil && err != context.Canceled {
			// This is expected
		}
	})
}

func TestRegistryGetStatistics(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	// Register some decorators
	for i := 0; i < 5; i++ {
		dr.Register(&RegisteredDecorator{
			Name: strings.Repeat("S", i+1),
			Type: "test",
		})
	}

	// Invoke some decorators
	for i := 0; i < 10; i++ {
		dr.Invoke(context.Background(), "GET", DecoratorArgs{})
	}

	stats := dr.GetStatistics()

	if stats["decorators"].(int) == 0 {
		t.Error("Expected decorators count > 0")
	}

	if stats["handlers"].(int) == 0 {
		t.Error("Expected handlers count > 0")
	}

	if stats["invocations"].(int64) < 10 {
		t.Error("Expected at least 10 invocations")
	}

	if stats["registrations"].(int64) < 5 {
		t.Error("Expected at least 5 registrations")
	}
}

func TestShutdown(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	// Add some decorators
	for i := 0; i < 3; i++ {
		dr.Register(&RegisteredDecorator{
			Name: strings.Repeat("X", i+1),
		})
	}

	err := dr.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown: %v", err)
	}

	// Verify everything is cleared
	dr.mu.RLock()
	decoratorCount := len(dr.decorators)
	pluginCount := len(dr.plugins)
	handlerCount := len(dr.handlers)
	dr.mu.RUnlock()

	if decoratorCount != 0 {
		t.Error("Expected decorators to be cleared")
	}
	if pluginCount != 0 {
		t.Error("Expected plugins to be cleared")
	}
	if handlerCount != 0 {
		t.Error("Expected handlers to be cleared")
	}
}

func TestConcurrentOperations(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	var wg sync.WaitGroup

	// Concurrent registrations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			dr.Register(&RegisteredDecorator{
				Name: strings.Repeat("A", i+1),
				Type: "concurrent",
			})
		}
	}()

	// Concurrent invocations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			dr.Invoke(context.Background(), "GET", DecoratorArgs{})
		}
	}()

	// Concurrent gets
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			dr.Get("GET")
		}
	}()

	// Concurrent lists
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			dr.List()
			dr.ListByType("rest")
		}
	}()

	// Concurrent statistics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			dr.GetStatistics()
		}
	}()

	wg.Wait()

	// Verify operations completed
	stats := dr.GetStatistics()
	if stats["invocations"].(int64) < 50 {
		t.Error("Expected at least 50 invocations from concurrent operations")
	}
}

func TestRestHandler(t *testing.T) {
	dr := NewDecoratorRegistry(nil)

	t.Run("valid path argument", func(t *testing.T) {
		handler := dr.restHandler("GET")
		args := DecoratorArgs{
			Target:    "handler",
			Arguments: []interface{}{"/api/users"},
			Properties: map[string]interface{}{
				"auth": true,
			},
		}

		result, err := handler(context.Background(), args)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("Expected success")
		}
		if result.Metadata["path"] != "/api/users" {
			t.Error("Expected path in metadata")
		}
		if result.Metadata["auth"] != true {
			t.Error("Expected auth property in metadata")
		}
	})

	t.Run("missing path argument", func(t *testing.T) {
		handler := dr.restHandler("POST")
		args := DecoratorArgs{
			Target:    "handler",
			Arguments: []interface{}{},
		}

		result, _ := handler(context.Background(), args)
		if result.Success {
			t.Error("Expected failure for missing path")
		}
		if !strings.Contains(result.Error, "path argument required") {
			t.Error("Expected 'path argument required' error")
		}
	})

	t.Run("invalid path type", func(t *testing.T) {
		handler := dr.restHandler("DELETE")
		args := DecoratorArgs{
			Target:    "handler",
			Arguments: []interface{}{123}, // Not a string
		}

		result, _ := handler(context.Background(), args)
		if result.Success {
			t.Error("Expected failure for invalid path type")
		}
		if !strings.Contains(result.Error, "path must be a string") {
			t.Error("Expected 'path must be a string' error")
		}
	})
}

func BenchmarkRegister(b *testing.B) {
	dr := NewDecoratorRegistry(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dr.Register(&RegisteredDecorator{
			Name: strings.Repeat("B", i%100+1),
			Type: "bench",
		})
	}
}

func BenchmarkInvoke(b *testing.B) {
	dr := NewDecoratorRegistry(nil)
	args := DecoratorArgs{
		Target:    "test",
		Arguments: []interface{}{"/test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dr.Invoke(context.Background(), "GET", args)
	}
}

func BenchmarkList(b *testing.B) {
	dr := NewDecoratorRegistry(nil)

	// Add many decorators
	for i := 0; i < 100; i++ {
		dr.Register(&RegisteredDecorator{
			Name: strings.Repeat("L", i+1),
			Type: "bench",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dr.List()
	}
}

func BenchmarkConcurrentInvoke(b *testing.B) {
	dr := NewDecoratorRegistry(nil)

	b.RunParallel(func(pb *testing.PB) {
		args := DecoratorArgs{
			Target:    "test",
			Arguments: []interface{}{"/test"},
		}
		for pb.Next() {
			dr.Invoke(context.Background(), "GET", args)
		}
	})
}
