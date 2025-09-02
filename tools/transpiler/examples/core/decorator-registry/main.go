// Package main demonstrates usage of the decorator registry.
// This example shows how to register, invoke, and manage decorators.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	fmt.Println("=== Decorator Registry Example ===")
	fmt.Println()

	// Example 1: Using built-in decorators
	builtinExample()

	// Example 2: Custom decorator registration
	customDecoratorExample()

	// Example 3: Schema validation
	schemaValidationExample()

	// Example 4: Decorator composition
	compositionExample()

	// Example 5: Registry management
	registryManagementExample()
}

func builtinExample() {
	fmt.Println("Example 1: Built-in Decorators")
	fmt.Println(strings.Repeat("-", 40))

	// Create registry with default config
	registry := core.NewDecoratorRegistry(nil)

	// List all built-in decorators
	decorators := registry.List()
	fmt.Printf("Registry has %d built-in decorators:\n", len(decorators))

	// Group by type
	byType := make(map[string][]string)
	for _, dec := range decorators {
		byType[dec.Type] = append(byType[dec.Type], dec.Name)
	}

	for dtype, names := range byType {
		fmt.Printf("  %s: %s\n", dtype, strings.Join(names, ", "))
	}

	// Invoke REST decorator
	fmt.Println("\nInvoking GET decorator:")
	args := core.DecoratorArgs{
		Target:    "GetUsers",
		Arguments: []interface{}{"/api/users"},
		Properties: map[string]interface{}{
			"auth":     true,
			"version":  "v1",
		},
	}

	result, err := registry.Invoke(context.Background(), "GET", args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Success: %v\n", result.Success)
	fmt.Printf("  Duration: %v\n", result.Duration)
	fmt.Printf("  Metadata: %+v\n", result.Metadata)
	fmt.Println()
}

func customDecoratorExample() {
	fmt.Println("Example 2: Custom Decorators")
	fmt.Println(strings.Repeat("-", 40))

	registry := core.NewDecoratorRegistry(nil)

	// Register a custom decorator
	customDecorator := &core.RegisteredDecorator{
		Name:        "Retry",
		Type:        "resilience",
		Description: "Retry operation on failure",
		Version:     "1.0.0",
		Author:      "example",
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{
					Name:        "attempts",
					Type:        "int",
					Required:    true,
					Description: "Number of retry attempts",
				},
				{
					Name:        "delay",
					Type:        "duration",
					Required:    false,
					Default:     "1s",
					Description: "Delay between retries",
				},
			},
		},
		Handler: func(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
			start := time.Now()

			// Extract arguments
			attempts := 3
			delay := time.Second

			if len(args.Arguments) > 0 {
				if a, ok := args.Arguments[0].(int); ok {
					attempts = a
				}
			}
			if len(args.Arguments) > 1 {
				if d, ok := args.Arguments[1].(string); ok {
					if duration, err := time.ParseDuration(d); err == nil {
						delay = duration
					}
				}
			}

			return core.DecoratorResult{
				Success:  true,
				Modified: args.Target,
				Metadata: map[string]interface{}{
					"retry_attempts": attempts,
					"retry_delay":    delay.String(),
					"target":         args.Target,
				},
				Duration: time.Since(start),
			}, nil
		},
	}

	err := registry.Register(customDecorator)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Registered custom Retry decorator")

	// Invoke custom decorator
	args := core.DecoratorArgs{
		Target:    "ProcessPayment",
		Arguments: []interface{}{5, "2s"},
	}

	result, err := registry.Invoke(context.Background(), "Retry", args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Invoked Retry decorator:\n")
	fmt.Printf("  Metadata: %+v\n", result.Metadata)
	fmt.Println()
}

func schemaValidationExample() {
	fmt.Println("Example 3: Schema Validation")
	fmt.Println(strings.Repeat("-", 40))

	config := &core.RegistryConfig{
		ValidateOnRegister: true,
	}
	registry := core.NewDecoratorRegistry(config)

	// Try to register decorator with invalid schema
	invalidDecorator := &core.RegisteredDecorator{
		Name: "Invalid",
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{
					// Missing required Name field
					Type:     "string",
					Required: true,
				},
			},
		},
	}

	err := registry.Register(invalidDecorator)
	if err != nil {
		fmt.Printf("Validation failed (expected): %v\n", err)
	}

	// Register decorator with valid schema
	validDecorator := &core.RegisteredDecorator{
		Name: "Cache",
		Type: "performance",
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{
					Name:        "ttl",
					Type:        "duration",
					Required:    true,
					Description: "Time to live",
				},
			},
			Properties: map[string]core.PropertyDef{
				"strategy": {
					Type:        "string",
					Required:    false,
					Default:     "lru",
					Enum:        []string{"lru", "lfu", "fifo"},
					Description: "Cache eviction strategy",
				},
			},
		},
		Handler: func(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
			return core.DecoratorResult{Success: true}, nil
		},
	}

	err = registry.Register(validDecorator)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully registered Cache decorator with schema")

	// Export schema
	var builder strings.Builder
	err = registry.ExportSchema(&builder)
	if err != nil {
		log.Fatal(err)
	}

	var schema []core.RegisteredDecorator
	json.Unmarshal([]byte(builder.String()), &schema)

	// Find our Cache decorator
	for _, dec := range schema {
		if dec.Name == "Cache" {
			fmt.Printf("\nCache decorator schema:\n")
			fmt.Printf("  Arguments: %d\n", len(dec.Schema.Arguments))
			fmt.Printf("  Properties: %d\n", len(dec.Schema.Properties))
			break
		}
	}
	fmt.Println()
}

func compositionExample() {
	fmt.Println("Example 4: Decorator Composition")
	fmt.Println(strings.Repeat("-", 40))

	registry := core.NewDecoratorRegistry(nil)

	// Create a composite decorator that combines multiple decorators
	compositeDecorator := &core.RegisteredDecorator{
		Name:        "SecureEndpoint",
		Type:        "composite",
		Description: "Combines Auth, RateLimit, and Logging",
		Handler: func(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
			start := time.Now()
			metadata := make(map[string]interface{})

			// Apply Auth
			authResult, _ := registry.Invoke(ctx, "Auth", args)
			metadata["auth"] = authResult.Success

			// Apply RateLimit
			rateLimitArgs := args
			rateLimitArgs.Arguments = []interface{}{100, "1m"}
			rateLimitResult, _ := registry.Invoke(ctx, "RateLimit", rateLimitArgs)
			metadata["rate_limit"] = rateLimitResult.Metadata

			// Add logging
			metadata["logged_at"] = time.Now().Format(time.RFC3339)
			metadata["endpoint"] = args.Target

			return core.DecoratorResult{
				Success:  true,
				Modified: args.Target,
				Metadata: metadata,
				Duration: time.Since(start),
			}, nil
		},
	}

	err := registry.Register(compositeDecorator)
	if err != nil {
		log.Fatal(err)
	}

	// Use composite decorator
	args := core.DecoratorArgs{
		Target: "AdminDashboard",
	}

	result, err := registry.Invoke(context.Background(), "SecureEndpoint", args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Applied SecureEndpoint composite decorator:")
	fmt.Printf("  Success: %v\n", result.Success)
	fmt.Printf("  Metadata keys: ")
	for key := range result.Metadata {
		fmt.Printf("%s ", key)
	}
	fmt.Println("\n")
}

func registryManagementExample() {
	fmt.Println("Example 5: Registry Management")
	fmt.Println(strings.Repeat("-", 40))

	config := &core.RegistryConfig{
		MaxDecorators: 100,
		EnableMetrics: true,
	}
	registry := core.NewDecoratorRegistry(config)

	// Register multiple decorators
	for i := 0; i < 10; i++ {
		decorator := &core.RegisteredDecorator{
			Name: fmt.Sprintf("Custom%d", i),
			Type: "test",
			Handler: func(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
				return core.DecoratorResult{Success: true}, nil
			},
		}
		registry.Register(decorator)
	}

	// Perform some invocations
	for i := 0; i < 20; i++ {
		registry.Invoke(context.Background(), "GET", core.DecoratorArgs{
			Target:    fmt.Sprintf("endpoint%d", i),
			Arguments: []interface{}{"/api/test"},
		})
	}

	// Get statistics
	stats := registry.GetStatistics()
	fmt.Println("Registry Statistics:")
	fmt.Printf("  Total decorators: %d\n", stats["decorators"])
	fmt.Printf("  Total handlers: %d\n", stats["handlers"])
	fmt.Printf("  Registrations: %d\n", stats["registrations"])
	fmt.Printf("  Invocations: %d\n", stats["invocations"])

	// List decorators by type
	testDecorators := registry.ListByType("test")
	fmt.Printf("\nTest decorators: %d\n", len(testDecorators))

	restDecorators := registry.ListByType("rest")
	fmt.Printf("REST decorators: %d\n", len(restDecorators))

	// Export and display a sample
	var builder strings.Builder
	registry.ExportSchema(&builder)
	schemaJSON := builder.String()
	fmt.Printf("\nExported schema size: %d bytes\n", len(schemaJSON))

	// Shutdown
	err := registry.Shutdown()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nRegistry shutdown complete")
}
