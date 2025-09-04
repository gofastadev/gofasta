package fault_tolerance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// Tests for @ActorRef decorator (2.1c from roadmap)

func TestActorRefDecorator_Registration(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorRef := &core.RegisteredDecorator{
		Name:        "ActorRef",
		Type:        "fault_tolerance",
		Description: "Actor references with fast lookup tables",
		Handler:     actorRefHandler,
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{Name: "actorPath", Type: "string", Required: true},
			},
			Properties: map[string]core.PropertyDef{
				"fastLookup":    {Type: "bool", Default: true},
				"cacheEnabled":  {Type: "bool", Default: true},
				"timeoutMs":     {Type: "int", Default: 5000},
			},
		},
	}

	err := registry.Register(actorRef)
	if err != nil {
		t.Errorf("failed to register ActorRef decorator: %v", err)
		return
	}

	// Verify registration
	registered, err := registry.Get("ActorRef")
	if err != nil {
		t.Errorf("failed to get registered ActorRef decorator: %v", err)
		return
	}

	if registered.Name != "ActorRef" {
		t.Errorf("expected name 'ActorRef', got %q", registered.Name)
	}
}

func TestActorRefDecorator_FastLookupTables(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorRef := &core.RegisteredDecorator{
		Name:    "ActorRef",
		Type:    "fault_tolerance",
		Handler: actorRefHandler,
	}
	registry.Register(actorRef)

	ctx := context.Background()
	
	// Test creating ActorRef with fast lookup
	args := core.DecoratorArgs{
		Target:    mockActorRefTarget(),
		Arguments: []interface{}{"/user/testActor"},
		Properties: map[string]interface{}{
			"fastLookup":   true,
			"cacheEnabled": true,
			"lookupTable":  "hashmap",
		},
	}

	result, err := registry.Invoke(ctx, "ActorRef", args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
		return
	}

	// Verify fast lookup configuration
	expectedMetadata := map[string]interface{}{
		"fast_lookup_enabled": true,
		"lookup_table_type":   "hashmap",
		"cache_enabled":       true,
		"actor_path":         "/user/testActor",
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := result.Metadata[key]; !exists {
			t.Errorf("missing metadata key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestActorRefDecorator_LookupPerformance(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorRef := &core.RegisteredDecorator{
		Name:    "ActorRef",
		Type:    "fault_tolerance",
		Handler: actorRefHandler,
	}
	registry.Register(actorRef)

	ctx := context.Background()
	
	// Create multiple ActorRefs for performance testing
	const numActorRefs = 1000
	actorPaths := make([]string, numActorRefs)
	for i := 0; i < numActorRefs; i++ {
		actorPaths[i] = fmt.Sprintf("/user/actor_%d", i)
	}

	// Create ActorRefs
	start := time.Now()
	for _, path := range actorPaths {
		args := core.DecoratorArgs{
			Target:    mockActorRefTarget(),
			Arguments: []interface{}{path},
			Properties: map[string]interface{}{
				"fastLookup": true,
			},
		}

		result, err := registry.Invoke(ctx, "ActorRef", args)
		if err != nil || !result.Success {
			t.Errorf("failed to create ActorRef for %s: %v, %s", path, err, result.Error)
			return
		}
	}
	creationDuration := time.Since(start)

	// Test lookup performance
	start = time.Now()
	for i := 0; i < numActorRefs*10; i++ { // 10x lookups
		path := actorPaths[i%numActorRefs]
		lookupArgs := core.DecoratorArgs{
			Target:    mockActorRefTarget(),
			Arguments: []interface{}{"lookup", path},
		}

		result, err := registry.Invoke(ctx, "ActorRef", lookupArgs)
		if err != nil || !result.Success {
			t.Errorf("lookup failed for %s: %v, %s", path, err, result.Error)
			return
		}
	}
	lookupDuration := time.Since(start)

	avgCreation := creationDuration / numActorRefs
	avgLookup := lookupDuration / (numActorRefs * 10)

	// Performance expectations
	if avgCreation > 1*time.Millisecond {
		t.Errorf("ActorRef creation too slow: %v (expected < 1ms)", avgCreation)
	}

	if avgLookup > 100*time.Microsecond {
		t.Errorf("ActorRef lookup too slow: %v (expected < 100μs)", avgLookup)
	}

	t.Logf("ActorRef performance - Creation: %v, Lookup: %v", avgCreation, avgLookup)
}

func TestActorRefDecorator_ActorMessaging(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorRef := &core.RegisteredDecorator{
		Name:    "ActorRef",
		Type:    "fault_tolerance",
		Handler: actorRefHandler,
	}
	registry.Register(actorRef)

	ctx := context.Background()
	
	// Create ActorRef
	args := core.DecoratorArgs{
		Target:    mockActorRefTarget(),
		Arguments: []interface{}{"/user/messageActor"},
		Properties: map[string]interface{}{
			"fastLookup":      true,
			"messagingEnabled": true,
		},
	}

	result, err := registry.Invoke(ctx, "ActorRef", args)
	if err != nil || !result.Success {
		t.Fatalf("failed to create ActorRef: %v, %s", err, result.Error)
	}

	actorRefInstance := result.Modified

	// Test message sending
	sendArgs := core.DecoratorArgs{
		Target:    actorRefInstance,
		Arguments: []interface{}{"send"},
		Properties: map[string]interface{}{
			"message":     "Hello Actor!",
			"messageType": "text",
			"sender":      "/user/senderActor",
		},
	}

	sendResult, err := registry.Invoke(ctx, "ActorRef", sendArgs)
	if err != nil {
		t.Errorf("send message error: %v", err)
		return
	}

	if !sendResult.Success {
		t.Errorf("send message failed: %s", sendResult.Error)
		return
	}

	// Verify messaging metadata
	if messageSent, exists := sendResult.Metadata["message_sent"]; !exists {
		t.Errorf("missing message_sent metadata")
	} else if messageSent != true {
		t.Errorf("expected message_sent to be true, got %v", messageSent)
	}
}

func TestActorRefDecorator_ActorRefEquality(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actorRef := &core.RegisteredDecorator{
		Name:    "ActorRef",
		Type:    "fault_tolerance",
		Handler: actorRefHandler,
	}
	registry.Register(actorRef)

	ctx := context.Background()
	path := "/user/testActor"
	
	// Create two ActorRefs to the same path
	args1 := core.DecoratorArgs{
		Target:    mockActorRefTarget(),
		Arguments: []interface{}{path},
		Properties: map[string]interface{}{"fastLookup": true},
	}

	result1, err := registry.Invoke(ctx, "ActorRef", args1)
	if err != nil || !result1.Success {
		t.Fatalf("failed to create first ActorRef: %v, %s", err, result1.Error)
	}

	args2 := core.DecoratorArgs{
		Target:    mockActorRefTarget(),
		Arguments: []interface{}{path},
		Properties: map[string]interface{}{"fastLookup": true},
	}

	result2, err := registry.Invoke(ctx, "ActorRef", args2)
	if err != nil || !result2.Success {
		t.Fatalf("failed to create second ActorRef: %v, %s", err, result2.Error)
	}

	// Test equality
	equalityArgs := core.DecoratorArgs{
		Target:    result1.Modified,
		Arguments: []interface{}{"equals"},
		Properties: map[string]interface{}{
			"otherActorRef": result2.Modified,
		},
	}

	equalityResult, err := registry.Invoke(ctx, "ActorRef", equalityArgs)
	if err != nil {
		t.Errorf("equality check error: %v", err)
		return
	}

	if !equalityResult.Success {
		t.Errorf("equality check failed: %s", equalityResult.Error)
		return
	}

	// ActorRefs to same path should be equal
	if equal, exists := equalityResult.Metadata["actors_equal"]; !exists {
		t.Errorf("missing actors_equal metadata")
	} else if equal != true {
		t.Errorf("expected actors_equal to be true, got %v", equal)
	}
}

// Mock ActorRef target
func mockActorRefTarget() interface{} {
	return map[string]interface{}{
		"type":       "actorRef",
		"targetPath": "",
		"source":     "test_file.go",
	}
}

// ActorRef handler placeholder
var actorRefHandler = ActorRefDecorator

