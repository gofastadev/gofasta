package supervisor

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

// Additional advanced tests for @Supervisor decorator covering missing functionality

func TestSupervisorDecorator_ParallelStartup(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Test parallel startup of multiple children
	children := []string{"child1", "child2", "child3", "child4", "child5"}

	args := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"parallelStartup": true,
			"children":        children,
			"startupTimeout":  "5s",
		},
	}

	start := time.Now()
	result, err := registry.Invoke(ctx, "Supervisor", args)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Parallel startup should be faster than sequential
	expectedMaxDuration := time.Duration(len(children)) * 50 * time.Millisecond
	if duration > expectedMaxDuration {
		t.Errorf("parallel startup took too long: %v (expected < %v)", duration, expectedMaxDuration)
	}

	// Verify parallel startup metadata
	if parallelStartup, exists := result.Metadata["parallel_startup"]; !exists {
		t.Errorf("missing parallel_startup metadata")
	} else if parallelStartup != true {
		t.Errorf("expected parallel_startup to be true, got %v", parallelStartup)
	}
}

func TestSupervisorDecorator_PerformanceBenchmark(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Test transpilation performance target (< 300ms)
	const numIterations = 100
	args := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"maxRetries":    5,
			"retryInterval": "1s",
			"fastInit":      true,
		},
	}

	start := time.Now()
	for i := 0; i < numIterations; i++ {
		result, err := registry.Invoke(ctx, "Supervisor", args)
		if err != nil {
			t.Errorf("iteration %d error: %v", i, err)
			return
		}
		if !result.Success {
			t.Errorf("iteration %d failed: %s", i, result.Error)
			return
		}
	}
	duration := time.Since(start)

	avgDuration := duration / numIterations

	// Each supervisor creation should be well under 300ms
	maxExpectedDuration := 10 * time.Millisecond
	if avgDuration > maxExpectedDuration {
		t.Errorf("supervisor creation too slow: %v per creation (expected < %v)", avgDuration, maxExpectedDuration)
	}

	t.Logf("Supervisor performance: %v per creation (%d iterations)", avgDuration, numIterations)
}

func TestSupervisorDecorator_DynamicSupervision(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Test adding children dynamically
	initialChildren := []string{"child1", "child2"}
	args := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"children":        initialChildren,
			"dynamicChildren": true,
			"maxChildren":     10,
		},
	}

	result, err := registry.Invoke(ctx, "Supervisor", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify dynamic children support
	if dynamicSupported, exists := result.Metadata["dynamic_children"]; !exists {
		t.Errorf("missing dynamic_children metadata")
	} else if dynamicSupported != true {
		t.Errorf("expected dynamic_children to be true, got %v", dynamicSupported)
	}

	// Test adding new child
	addChildArgs := core.DecoratorArgs{
		Target:    result.Modified,
		Arguments: []interface{}{"addChild"},
		Properties: map[string]interface{}{
			"childName": "child3",
			"childType": "worker",
		},
	}

	addResult, err := registry.Invoke(ctx, "Supervisor", addChildArgs)
	if err != nil {
		t.Errorf("add child error: %v", err)
		return
	}

	if !addResult.Success {
		t.Errorf("add child failed: %s", addResult.Error)
		return
	}

	// Test removing child
	removeChildArgs := core.DecoratorArgs{
		Target:    result.Modified,
		Arguments: []interface{}{"removeChild"},
		Properties: map[string]interface{}{
			"childName": "child1",
		},
	}

	removeResult, err := registry.Invoke(ctx, "Supervisor", removeChildArgs)
	if err != nil {
		t.Errorf("remove child error: %v", err)
		return
	}

	if !removeResult.Success {
		t.Errorf("remove child failed: %s", removeResult.Error)
		return
	}
}

func TestSupervisorDecorator_LifecycleManagement(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Create supervisor
	createArgs := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"lifecycle": true,
		},
	}

	result, err := registry.Invoke(ctx, "Supervisor", createArgs)
	if err != nil || !result.Success {
		t.Fatalf("failed to create supervisor: %v, %s", err, result.Error)
	}

	supervisorInstance := result.Modified

	// Test lifecycle operations
	lifecycleOps := []string{"start", "pause", "resume", "stop"}

	for _, op := range lifecycleOps {
		opArgs := core.DecoratorArgs{
			Target:    supervisorInstance,
			Arguments: []interface{}{op},
		}

		opResult, err := registry.Invoke(ctx, "Supervisor", opArgs)
		if err != nil {
			t.Errorf("lifecycle operation %s error: %v", op, err)
			continue
		}

		if !opResult.Success {
			t.Errorf("lifecycle operation %s failed: %s", op, opResult.Error)
			continue
		}

		// Verify operation was recorded
		if state, exists := opResult.Metadata["lifecycle_state"]; !exists {
			t.Errorf("missing lifecycle_state after %s operation", op)
		} else if state != op+"ed" && op != "stop" {
			// "started", "paused", "resumed", but "stopped" not "stoped"
			expectedState := op + "ed"
			if op == "stop" {
				expectedState = "stopped"
			}
			if state != expectedState {
				t.Errorf("expected lifecycle_state %s, got %v", expectedState, state)
			}
		}
	}
}

func TestSupervisorDecorator_ConcurrentOperations(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Create supervisor
	createArgs := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"concurrent": true,
			"children":   []string{"child1", "child2", "child3"},
		},
	}

	result, err := registry.Invoke(ctx, "Supervisor", createArgs)
	if err != nil || !result.Success {
		t.Fatalf("failed to create supervisor: %v, %s", err, result.Error)
	}

	supervisorInstance := result.Modified

	// Test concurrent operations on supervisor
	const numGoroutines = 50
	operations := []string{"restart", "status", "health", "metrics"}

	var wg sync.WaitGroup
	errorsChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			op := operations[id%len(operations)]
			opArgs := core.DecoratorArgs{
				Target:    supervisorInstance,
				Arguments: []interface{}{op},
				Properties: map[string]interface{}{
					"operationId": id,
				},
			}

			opResult, err := registry.Invoke(ctx, "Supervisor", opArgs)
			if err != nil {
				errorsChan <- err
				return
			}

			if !opResult.Success {
				errorsChan <- fmt.Errorf("operation %s failed: %s", op, opResult.Error)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errorsChan)

	// Check for errors
	for err := range errorsChan {
		t.Errorf("concurrent operation error: %v", err)
	}
}

func TestSupervisorDecorator_MemoryUsageTracking(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Test memory tracking with different numbers of children
	childCounts := []int{1, 10, 50, 100}

	for _, childCount := range childCounts {
		children := make([]string, childCount)
		for i := 0; i < childCount; i++ {
			children[i] = fmt.Sprintf("child_%d", i)
		}

		args := core.DecoratorArgs{
			Target:    mockSupervisorTarget(),
			Arguments: []interface{}{"OneForOne"},
			Properties: map[string]interface{}{
				"children":      children,
				"trackMemory":   true,
				"memoryProfile": true,
			},
		}

		result, err := registry.Invoke(ctx, "Supervisor", args)
		if err != nil {
			t.Errorf("memory tracking test with %d children error: %v", childCount, err)
			continue
		}

		if !result.Success {
			t.Errorf("memory tracking test with %d children failed: %s", childCount, result.Error)
			continue
		}

		// Verify memory tracking metadata
		memoryUsage, exists := result.Metadata["memory_usage"]
		if !exists {
			t.Errorf("missing memory_usage metadata for %d children", childCount)
			continue
		}

		usage, ok := memoryUsage.(map[string]interface{})
		if !ok {
			t.Errorf("memory_usage is not a map for %d children", childCount)
			continue
		}

		// Check that memory usage increases with more children
		if baseMemory, exists := usage["base_memory"]; exists {
			if baseValue, ok := baseMemory.(int64); ok {
				expectedMinMemory := int64(childCount * 100) // 100 bytes per child minimum
				if baseValue < expectedMinMemory {
					t.Errorf("unexpected low memory usage %d for %d children (expected > %d)",
						baseValue, childCount, expectedMinMemory)
				}
			}
		}

		t.Logf("Supervisor with %d children - memory usage: %v", childCount, usage)
	}
}

func TestSupervisorDecorator_TreeDepthLimits(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Test supervision tree depth limits
	maxDepth := 10
	var currentParent interface{} = mockSupervisorTarget()

	for depth := 1; depth <= maxDepth+1; depth++ {
		args := core.DecoratorArgs{
			Target:    currentParent,
			Arguments: []interface{}{"OneForOne"},
			Properties: map[string]interface{}{
				"name":     fmt.Sprintf("supervisor_level_%d", depth),
				"depth":    depth,
				"maxDepth": maxDepth,
			},
		}

		result, err := registry.Invoke(ctx, "Supervisor", args)
		if err != nil {
			t.Errorf("depth %d error: %v", depth, err)
			break
		}

		if depth > maxDepth {
			// Should fail when exceeding max depth
			if result.Success {
				t.Errorf("expected failure at depth %d (max: %d), but got success", depth, maxDepth)
			}
			break
		} else {
			// Should succeed within max depth
			if !result.Success {
				t.Errorf("expected success at depth %d, but got failure: %s", depth, result.Error)
				break
			}
			currentParent = result.Modified
		}
	}
}

func TestSupervisorDecorator_ActorSystemIntegration(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()

	// Test supervisor within actor system context
	args := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"actorSystem":    "TestActorSystem",
			"systemAddress":  "akka://TestSystem",
			"managedActors":  []string{"actor1", "actor2", "actor3"},
			"actorRefsTable": true,
		},
	}

	result, err := registry.Invoke(ctx, "Supervisor", args)
	if err != nil {
		t.Errorf("actor system integration error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("actor system integration failed: %s", result.Error)
		return
	}

	// Verify actor system integration metadata
	expectedMetadata := map[string]interface{}{
		"actor_system_integration": true,
		"system_name":              "TestActorSystem",
		"actor_refs_managed":       3,
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := result.Metadata[key]; !exists {
			t.Errorf("missing metadata key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}
