package fault_tolerance

import (
	"context"
	"testing"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// Additional advanced tests for @Actor decorator covering missing functionality

func TestActorDecorator_ActorReferences(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with ActorRef support
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"actorRef":     true,
			"actorPath":    "/user/testActor",
			"actorAddress": "akka://TestSystem/user/testActor",
			"fastLookup":   true,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify ActorRef metadata
	expectedMetadata := map[string]interface{}{
		"actor_ref_enabled": true,
		"actor_path":        "/user/testActor",
		"fast_lookup":       true,
		"addressable":       true,
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := result.Metadata[key]; !exists {
			t.Errorf("missing metadata key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorDecorator_LifecycleManagement(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with lifecycle management
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"lifecycle":     true,
			"preStart":      true,
			"postStop":      true,
			"preRestart":    true,
			"postRestart":   true,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor: %v, %s", err, result.Error)
	}

	actorInstance := result.Modified

	// Test lifecycle operations
	lifecycleOps := []string{"start", "stop", "restart", "pause", "resume"}
	
	for _, op := range lifecycleOps {
		opArgs := core.DecoratorArgs{
			Target:    actorInstance,
			Arguments: []interface{}{op},
			Properties: map[string]interface{}{
				"operation": op,
			},
		}

		opResult, err := registry.Invoke(ctx, "Actor", opArgs)
		if err != nil {
			t.Errorf("lifecycle operation %s error: %v", op, err)
			continue
		}

		if !opResult.Success {
			t.Errorf("lifecycle operation %s failed: %s", op, opResult.Error)
			continue
		}

		// Verify lifecycle state
		if state, exists := opResult.Metadata["lifecycle_state"]; !exists {
			t.Errorf("missing lifecycle_state after %s operation", op)
		} else {
			expectedStates := map[string]string{
				"start":   "started",
				"stop":    "stopped",
				"restart": "restarted",
				"pause":   "paused", 
				"resume":  "resumed",
			}
			if expectedState, ok := expectedStates[op]; ok && state != expectedState {
				t.Errorf("expected lifecycle_state %s after %s, got %v", expectedState, op, state)
			}
		}
	}
}

func TestActorDecorator_MessageRouting(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with message routing
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"messageRouting":   true,
			"routingStrategy":  "roundRobin",
			"deliveryGuarantee": "atLeastOnce",
			"messageOrdering":   "fifo",
			"deadLetterQueue":   true,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify message routing configuration
	messageRouting, exists := result.Metadata["message_routing"]
	if !exists {
		t.Errorf("missing message_routing metadata")
		return
	}

	routingConfig, ok := messageRouting.(map[string]interface{})
	if !ok {
		t.Errorf("message_routing is not a map")
		return
	}

	expectedConfig := map[string]interface{}{
		"strategy":           "roundRobin",
		"delivery_guarantee": "atLeastOnce",
		"message_ordering":   "fifo",
		"dead_letter_queue":  true,
	}

	for key, expectedValue := range expectedConfig {
		if actualValue, exists := routingConfig[key]; !exists {
			t.Errorf("missing routing config key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected routing[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorDecorator_ActorToActorCommunication(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create sender actor
	senderArgs := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"actorName":     "senderActor",
			"canSend":       true,
			"actorRef":      true,
		},
	}

	senderResult, err := registry.Invoke(ctx, "Actor", senderArgs)
	if err != nil || !senderResult.Success {
		t.Fatalf("failed to create sender actor: %v, %s", err, senderResult.Error)
	}

	// Create receiver actor
	receiverArgs := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"actorName":     "receiverActor",
			"canReceive":    true,
			"actorRef":      true,
		},
	}

	receiverResult, err := registry.Invoke(ctx, "Actor", receiverArgs)
	if err != nil || !receiverResult.Success {
		t.Fatalf("failed to create receiver actor: %v, %s", err, receiverResult.Error)
	}

	// Test message sending between actors
	sendArgs := core.DecoratorArgs{
		Target:    senderResult.Modified,
		Arguments: []interface{}{"sendMessage"},
		Properties: map[string]interface{}{
			"targetActor": "receiverActor",
			"message":     "Hello from sender!",
			"messageType": "greeting",
		},
	}

	sendResult, err := registry.Invoke(ctx, "Actor", sendArgs)
	if err != nil {
		t.Errorf("send message error: %v", err)
		return
	}

	if !sendResult.Success {
		t.Errorf("send message failed: %s", sendResult.Error)
		return
	}

	// Verify message sending metadata
	if messageSent, exists := sendResult.Metadata["message_sent"]; !exists {
		t.Errorf("missing message_sent metadata")
	} else if messageSent != true {
		t.Errorf("expected message_sent to be true, got %v", messageSent)
	}
}

func TestActorDecorator_StateManagement(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with state management
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"stateManagement": true,
			"statePersistence": true,
			"stateRecovery":   true,
			"snapshotting":    true,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor with state: %v, %s", err, result.Error)
	}

	actorInstance := result.Modified

	// Test state operations
	stateOps := []struct {
		operation string
		data      map[string]interface{}
	}{
		{"setState", map[string]interface{}{"key": "value1", "counter": 1}},
		{"updateState", map[string]interface{}{"counter": 2}},
		{"getState", map[string]interface{}{"key": "key"}},
		{"snapshot", map[string]interface{}{"reason": "periodic"}},
		{"recover", map[string]interface{}{"fromSnapshot": true}},
	}

	for _, stateOp := range stateOps {
		opArgs := core.DecoratorArgs{
			Target:    actorInstance,
			Arguments: []interface{}{stateOp.operation},
			Properties: stateOp.data,
		}

		opResult, err := registry.Invoke(ctx, "Actor", opArgs)
		if err != nil {
			t.Errorf("state operation %s error: %v", stateOp.operation, err)
			continue
		}

		if !opResult.Success {
			t.Errorf("state operation %s failed: %s", stateOp.operation, opResult.Error)
			continue
		}

		// Verify state operation completed
		if opCompleted, exists := opResult.Metadata["state_operation_completed"]; !exists {
			t.Errorf("missing state_operation_completed for %s", stateOp.operation)
		} else if opCompleted != true {
			t.Errorf("expected state_operation_completed to be true for %s, got %v", stateOp.operation, opCompleted)
		}
	}
}

func TestActorDecorator_BackpressureHandling(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with backpressure handling
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"mailboxSize":        100, // Small mailbox to trigger backpressure
			"backpressure":       true,
			"backpressureStrategy": "dropOldest",
			"overflowPolicy":     "reject",
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify backpressure configuration
	backpressureConfig, exists := result.Metadata["backpressure"]
	if !exists {
		t.Errorf("missing backpressure metadata")
		return
	}

	configMap, ok := backpressureConfig.(map[string]interface{})
	if !ok {
		t.Errorf("backpressure config is not a map")
		return
	}

	expectedConfig := map[string]interface{}{
		"enabled":       true,
		"strategy":      "dropOldest",
		"overflow_policy": "reject",
		"mailbox_limit": 100,
	}

	for key, expectedValue := range expectedConfig {
		if actualValue, exists := configMap[key]; !exists {
			t.Errorf("missing backpressure config key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected backpressure[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorDecorator_ActorHierarchy(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create parent actor
	parentArgs := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"actorName":    "parentActor",
			"canSpawn":     true,
			"hierarchy":    true,
			"maxChildren":  5,
		},
	}

	parentResult, err := registry.Invoke(ctx, "Actor", parentArgs)
	if err != nil || !parentResult.Success {
		t.Fatalf("failed to create parent actor: %v, %s", err, parentResult.Error)
	}

	// Create child actors
	childNames := []string{"child1", "child2", "child3"}
	for _, childName := range childNames {
		childArgs := core.DecoratorArgs{
			Target:    mockActorTarget(),
			Arguments: []interface{}{},
			Properties: map[string]interface{}{
				"actorName": childName,
				"parent":    "parentActor",
				"hierarchy": true,
			},
		}

		childResult, err := registry.Invoke(ctx, "Actor", childArgs)
		if err != nil {
			t.Errorf("failed to create child %s: %v", childName, err)
			continue
		}

		if !childResult.Success {
			t.Errorf("failed to create child %s: %s", childName, childResult.Error)
			continue
		}

		// Verify parent-child relationship
		if parent, exists := childResult.Metadata["parent_actor"]; !exists {
			t.Errorf("missing parent_actor metadata for child %s", childName)
		} else if parent != "parentActor" {
			t.Errorf("expected parent_actor to be 'parentActor' for child %s, got %v", childName, parent)
		}
	}

	// Test spawning child through parent
	spawnArgs := core.DecoratorArgs{
		Target:    parentResult.Modified,
		Arguments: []interface{}{"spawnChild"},
		Properties: map[string]interface{}{
			"childName": "spawnedChild",
			"childType": "worker",
		},
	}

	spawnResult, err := registry.Invoke(ctx, "Actor", spawnArgs)
	if err != nil {
		t.Errorf("spawn child error: %v", err)
		return
	}

	if !spawnResult.Success {
		t.Errorf("spawn child failed: %s", spawnResult.Error)
		return
	}

	// Verify spawning metadata
	if childSpawned, exists := spawnResult.Metadata["child_spawned"]; !exists {
		t.Errorf("missing child_spawned metadata")
	} else if childSpawned != true {
		t.Errorf("expected child_spawned to be true, got %v", childSpawned)
	}
}

func TestActorDecorator_HotSwapBehavior(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with hot-swap capability
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"hotSwap":           true,
			"behaviorStack":     true,
			"defaultBehavior":   "normal",
			"allowBehaviorSwap": true,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create hot-swap actor: %v, %s", err, result.Error)
	}

	actorInstance := result.Modified

	// Test behavior swapping
	behaviors := []string{"aggressive", "conservative", "maintenance", "normal"}
	
	for _, behavior := range behaviors {
		swapArgs := core.DecoratorArgs{
			Target:    actorInstance,
			Arguments: []interface{}{"swapBehavior"},
			Properties: map[string]interface{}{
				"newBehavior": behavior,
				"pushToStack": true,
			},
		}

		swapResult, err := registry.Invoke(ctx, "Actor", swapArgs)
		if err != nil {
			t.Errorf("behavior swap to %s error: %v", behavior, err)
			continue
		}

		if !swapResult.Success {
			t.Errorf("behavior swap to %s failed: %s", behavior, swapResult.Error)
			continue
		}

		// Verify behavior swap
		if currentBehavior, exists := swapResult.Metadata["current_behavior"]; !exists {
			t.Errorf("missing current_behavior metadata")
		} else if currentBehavior != behavior {
			t.Errorf("expected current_behavior to be %s, got %v", behavior, currentBehavior)
		}
	}

	// Test behavior stack popping
	popArgs := core.DecoratorArgs{
		Target:    actorInstance,
		Arguments: []interface{}{"popBehavior"},
	}

	popResult, err := registry.Invoke(ctx, "Actor", popArgs)
	if err != nil {
		t.Errorf("behavior pop error: %v", err)
		return
	}

	if !popResult.Success {
		t.Errorf("behavior pop failed: %s", popResult.Error)
		return
	}

	// Should now be back to previous behavior
	if stackSize, exists := popResult.Metadata["behavior_stack_size"]; !exists {
		t.Errorf("missing behavior_stack_size metadata")
	} else if stackSize.(int) < len(behaviors)-1 {
		t.Errorf("expected behavior stack size to be reduced")
	}
}

func TestActorDecorator_ResourceCleanup(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor with resource management
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"resourceManagement": true,
			"autoCleanup":        true,
			"cleanupTimeout":     "10s",
			"resourceTracking":   true,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create actor with resources: %v, %s", err, result.Error)
	}

	actorInstance := result.Modified

	// Simulate resource allocation
	allocateArgs := core.DecoratorArgs{
		Target:    actorInstance,
		Arguments: []interface{}{"allocateResources"},
		Properties: map[string]interface{}{
			"resources": []string{"database", "file", "network"},
		},
	}

	allocateResult, err := registry.Invoke(ctx, "Actor", allocateArgs)
	if err != nil || !allocateResult.Success {
		t.Fatalf("resource allocation failed: %v, %s", err, allocateResult.Error)
	}

	// Test cleanup on termination
	cleanupArgs := core.DecoratorArgs{
		Target:    actorInstance,
		Arguments: []interface{}{"terminate"},
		Properties: map[string]interface{}{
			"reason": "testing cleanup",
		},
	}

	cleanupResult, err := registry.Invoke(ctx, "Actor", cleanupArgs)
	if err != nil {
		t.Errorf("cleanup error: %v", err)
		return
	}

	if !cleanupResult.Success {
		t.Errorf("cleanup failed: %s", cleanupResult.Error)
		return
	}

	// Verify cleanup metadata
	if cleanupCompleted, exists := cleanupResult.Metadata["cleanup_completed"]; !exists {
		t.Errorf("missing cleanup_completed metadata")
	} else if cleanupCompleted != true {
		t.Errorf("expected cleanup_completed to be true, got %v", cleanupCompleted)
	}

	if resourcesReleased, exists := cleanupResult.Metadata["resources_released"]; !exists {
		t.Errorf("missing resources_released metadata")
	} else if resourcesReleased.(int) != 3 {
		t.Errorf("expected 3 resources to be released, got %v", resourcesReleased)
	}
}

func TestActorDecorator_ActorSystemIntegration(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	ctx := context.Background()

	// Create actor integrated with actor system
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"actorSystem":     "TestActorSystem",
			"systemAddress":   "akka://TestSystem",
			"parallelStartup": true,
			"systemManaged":   true,
			"clusteringEnabled": false,
		},
	}

	result, err := registry.Invoke(ctx, "Actor", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify actor system integration
	expectedMetadata := map[string]interface{}{
		"actor_system_integration": true,
		"system_managed":           true,
		"parallel_startup":         true,
		"system_name":             "TestActorSystem",
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := result.Metadata[key]; !exists {
			t.Errorf("missing metadata key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}