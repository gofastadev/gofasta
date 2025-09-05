package actorsystem

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

// Tests for @ActorSystem decorator (2.1d from roadmap)

func TestActorSystemDecorator_Registration(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:        "ActorSystem",
		Type:        "fault_tolerance",
		Description: "Actor system management with parallel startup",
		Handler:     actorSystemHandler,
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{Name: "systemName", Type: "string", Required: true},
			},
			Properties: map[string]core.PropertyDef{
				"parallelStartup": {Type: "bool", Default: true},
				"maxActors":       {Type: "int", Default: 10000},
				"clustering":      {Type: "bool", Default: false},
				"remoting":        {Type: "bool", Default: false},
			},
		},
	}

	err := registry.Register(actorSystem)
	if err != nil {
		t.Errorf("failed to register ActorSystem decorator: %v", err)
		return
	}

	// Verify registration
	registered, err := registry.Get("ActorSystem")
	if err != nil {
		t.Errorf("failed to get registered ActorSystem decorator: %v", err)
		return
	}

	if registered.Name != "ActorSystem" {
		t.Errorf("expected name 'ActorSystem', got %q", registered.Name)
	}
}

func TestActorSystemDecorator_ParallelStartup(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Test parallel startup of actor system
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"TestActorSystem"},
		Properties: map[string]interface{}{
			"parallelStartup": true,
			"startupWorkers":  8,
			"actorCount":      100,
			"startupTimeout":  "10s",
		},
	}

	start := time.Now()
	result, err := registry.Invoke(ctx, "ActorSystem", args)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Parallel startup should be fast
	if duration > 5*time.Second {
		t.Errorf("parallel startup took too long: %v (expected < 5s)", duration)
	}

	// Verify parallel startup metadata
	expectedMetadata := map[string]interface{}{
		"parallel_startup": true,
		"startup_workers":  8,
		"system_name":      "TestActorSystem",
		"actors_started":   100,
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := result.Metadata[key]; !exists {
			t.Errorf("missing metadata key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorSystemDecorator_ActorManagement(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Create actor system
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"ActorManagementSystem"},
		Properties: map[string]interface{}{
			"maxActors":      1000,
			"actorPooling":   true,
			"actorRecycling": true,
		},
	}

	result, err := registry.Invoke(ctx, "ActorSystem", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor system: %v, %s", err, result.Error)
	}

	actorSystemInstance := result.Modified

	// Test actor lifecycle operations
	operations := []struct {
		operation string
		actorPath string
		actorType string
	}{
		{"createActor", "/user/worker1", "WorkerActor"},
		{"createActor", "/user/supervisor1", "SupervisorActor"},
		{"createActor", "/user/batch1", "BatchActor"},
		{"stopActor", "/user/worker1", ""},
		{"restartActor", "/user/supervisor1", ""},
		{"terminateActor", "/user/batch1", ""},
	}

	for _, op := range operations {
		opArgs := core.DecoratorArgs{
			Target:    actorSystemInstance,
			Arguments: []interface{}{op.operation},
			Properties: map[string]interface{}{
				"actorPath": op.actorPath,
				"actorType": op.actorType,
			},
		}

		opResult, err := registry.Invoke(ctx, "ActorSystem", opArgs)
		if err != nil {
			t.Errorf("%s operation error: %v", op.operation, err)
			continue
		}

		if !opResult.Success {
			t.Errorf("%s operation failed: %s", op.operation, opResult.Error)
			continue
		}

		// Verify operation completed
		if opCompleted, exists := opResult.Metadata["operation_completed"]; !exists {
			t.Errorf("missing operation_completed for %s", op.operation)
		} else if opCompleted != true {
			t.Errorf("expected operation_completed to be true for %s, got %v", op.operation, opCompleted)
		}
	}
}

func TestActorSystemDecorator_ConcurrentOperations(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Create actor system
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"ConcurrentSystem"},
		Properties: map[string]interface{}{
			"maxActors":  5000,
			"concurrent": true,
			"threadSafe": true,
		},
	}

	result, err := registry.Invoke(ctx, "ActorSystem", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor system: %v, %s", err, result.Error)
	}

	actorSystemInstance := result.Modified

	// Test concurrent operations
	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			opArgs := core.DecoratorArgs{
				Target:    actorSystemInstance,
				Arguments: []interface{}{"createActor"},
				Properties: map[string]interface{}{
					"actorPath":   fmt.Sprintf("/user/concurrent_%d", id),
					"actorType":   "ConcurrentWorker",
					"goroutineId": id,
				},
			}

			opResult, err := registry.Invoke(ctx, "ActorSystem", opArgs)
			if err != nil {
				errors <- err
				return
			}

			if !opResult.Success {
				errors <- fmt.Errorf("operation failed: %s", opResult.Error)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("had %d concurrent operation errors out of %d", errorCount, numGoroutines)
	}
}

func TestActorSystemDecorator_SystemShutdown(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Create actor system with actors
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"ShutdownTestSystem"},
		Properties: map[string]interface{}{
			"gracefulShutdown": true,
			"shutdownTimeout":  "30s",
			"actorCount":       50,
		},
	}

	result, err := registry.Invoke(ctx, "ActorSystem", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor system: %v, %s", err, result.Error)
	}

	actorSystemInstance := result.Modified

	// Test graceful shutdown
	shutdownArgs := core.DecoratorArgs{
		Target:    actorSystemInstance,
		Arguments: []interface{}{"shutdown"},
		Properties: map[string]interface{}{
			"graceful": true,
			"reason":   "test completed",
		},
	}

	start := time.Now()
	shutdownResult, err := registry.Invoke(ctx, "ActorSystem", shutdownArgs)
	shutdownDuration := time.Since(start)

	if err != nil {
		t.Errorf("shutdown error: %v", err)
		return
	}

	if !shutdownResult.Success {
		t.Errorf("shutdown failed: %s", shutdownResult.Error)
		return
	}

	// Graceful shutdown should complete in reasonable time
	if shutdownDuration > 10*time.Second {
		t.Errorf("shutdown took too long: %v (expected < 10s)", shutdownDuration)
	}

	// Verify shutdown metadata
	expectedMetadata := map[string]interface{}{
		"shutdown_completed": true,
		"graceful_shutdown":  true,
		"actors_terminated":  50,
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := shutdownResult.Metadata[key]; !exists {
			t.Errorf("missing shutdown metadata key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected shutdown metadata[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorSystemDecorator_Clustering(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Create clustered actor system
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"ClusteredSystem"},
		Properties: map[string]interface{}{
			"clustering":   true,
			"clusterNodes": []string{"node1", "node2", "node3"},
			"seedNodes":    []string{"node1"},
			"remoting":     true,
			"remotingPort": 2552,
		},
	}

	result, err := registry.Invoke(ctx, "ActorSystem", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify clustering configuration
	clusterConfig, exists := result.Metadata["cluster_config"]
	if !exists {
		t.Errorf("missing cluster_config metadata")
		return
	}

	configMap, ok := clusterConfig.(map[string]interface{})
	if !ok {
		t.Errorf("cluster_config is not a map")
		return
	}

	expectedConfig := map[string]interface{}{
		"clustering_enabled": true,
		"remoting_enabled":   true,
		"node_count":         3,
		"seed_nodes":         1,
	}

	for key, expectedValue := range expectedConfig {
		if actualValue, exists := configMap[key]; !exists {
			t.Errorf("missing cluster config key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected cluster[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorSystemDecorator_Performance(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Performance test with large number of actors
	const numActors = 10000
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"PerformanceTestSystem"},
		Properties: map[string]interface{}{
			"parallelStartup": true,
			"maxActors":       numActors,
			"pooling":         true,
			"fastInit":        true,
		},
	}

	start := time.Now()
	result, err := registry.Invoke(ctx, "ActorSystem", args)
	creationDuration := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	actorSystemInstance := result.Modified

	// Test actor creation performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		createArgs := core.DecoratorArgs{
			Target:    actorSystemInstance,
			Arguments: []interface{}{"createActor"},
			Properties: map[string]interface{}{
				"actorPath": fmt.Sprintf("/user/perf_%d", i),
				"actorType": "PerformanceActor",
			},
		}

		createResult, err := registry.Invoke(ctx, "ActorSystem", createArgs)
		if err != nil || !createResult.Success {
			t.Errorf("failed to create actor %d: %v, %s", i, err, createResult.Error)
			return
		}
	}
	actorCreationDuration := time.Since(start)

	// Performance expectations
	avgSystemCreation := creationDuration
	avgActorCreation := actorCreationDuration / 1000

	if avgSystemCreation > 1*time.Second {
		t.Errorf("ActorSystem creation too slow: %v (expected < 1s)", avgSystemCreation)
	}

	if avgActorCreation > 1*time.Millisecond {
		t.Errorf("Actor creation too slow: %v per actor (expected < 1ms)", avgActorCreation)
	}

	t.Logf("ActorSystem performance - System: %v, Actor: %v per creation",
		avgSystemCreation, avgActorCreation)
}

func TestActorSystemDecorator_ActorDiscovery(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorSystem := &core.RegisteredDecorator{
		Name:    "ActorSystem",
		Type:    "fault_tolerance",
		Handler: actorSystemHandler,
	}
	registry.Register(actorSystem)

	ctx := context.Background()

	// Create actor system with discovery
	args := core.DecoratorArgs{
		Target:    mockActorSystemTarget(),
		Arguments: []interface{}{"DiscoverySystem"},
		Properties: map[string]interface{}{
			"actorDiscovery":  true,
			"registryEnabled": true,
			"pathResolution":  true,
		},
	}

	result, err := registry.Invoke(ctx, "ActorSystem", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor system: %v, %s", err, result.Error)
	}

	actorSystemInstance := result.Modified

	// Create some actors for discovery testing
	actorPaths := []string{
		"/user/worker1",
		"/user/worker2",
		"/user/supervisor1",
		"/system/logger",
	}

	for _, path := range actorPaths {
		createArgs := core.DecoratorArgs{
			Target:    actorSystemInstance,
			Arguments: []interface{}{"createActor"},
			Properties: map[string]interface{}{
				"actorPath": path,
				"actorType": "TestActor",
			},
		}

		createResult, err := registry.Invoke(ctx, "ActorSystem", createArgs)
		if err != nil || !createResult.Success {
			t.Errorf("failed to create actor %s: %v, %s", path, err, createResult.Error)
			continue
		}
	}

	// Test actor discovery
	discoveryTests := []struct {
		pattern       string
		expectedCount int
	}{
		{"/user/*", 3},   // worker1, worker2, supervisor1
		{"/system/*", 1}, // logger
		{"*/worker*", 2}, // worker1, worker2
		{"*", 4},         // all actors
	}

	for _, test := range discoveryTests {
		discoverArgs := core.DecoratorArgs{
			Target:    actorSystemInstance,
			Arguments: []interface{}{"discover"},
			Properties: map[string]interface{}{
				"pattern": test.pattern,
			},
		}

		discoverResult, err := registry.Invoke(ctx, "ActorSystem", discoverArgs)
		if err != nil {
			t.Errorf("discovery error for pattern %s: %v", test.pattern, err)
			continue
		}

		if !discoverResult.Success {
			t.Errorf("discovery failed for pattern %s: %s", test.pattern, discoverResult.Error)
			continue
		}

		// Verify discovery results
		if foundCount, exists := discoverResult.Metadata["actors_found"]; !exists {
			t.Errorf("missing actors_found for pattern %s", test.pattern)
		} else if foundCount != test.expectedCount {
			t.Errorf("expected %d actors for pattern %s, got %v", test.expectedCount, test.pattern, foundCount)
		}
	}
}

// Mock ActorSystem target
func mockActorSystemTarget() interface{} {
	return map[string]interface{}{
		"type":       "actorSystem",
		"systemName": "",
		"source":     "test_file.go",
	}
}

// ActorSystem handler placeholder
var actorSystemHandler = ActorSystemDecorator
