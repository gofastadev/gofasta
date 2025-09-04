package actor

import (
	"context"
	"testing"
	"time"
	"sync"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
	"github.com/healtronlabs/gofasta/tools/transpiler/decorators/faulttolerance/common"
)

func TestActorDecorator_Registration(t *testing.T) {
	tests := []struct {
		name        string
		decorator   *core.RegisteredDecorator
		wantErr     bool
		errContains string
	}{
		{
			name: "valid actor decorator",
			decorator: &core.RegisteredDecorator{
				Name:        "Actor",
				Type:        "fault_tolerance",
				Description: "Actor model implementation with memory pooling",
				Handler:     actorHandler,
				Schema: &core.DecoratorSchema{
					Properties: map[string]core.PropertyDef{
						"mailboxSize":     {Type: "int", Default: 1000},
						"poolSize":        {Type: "int", Default: 10},
						"timeout":         {Type: "duration", Default: "30s"},
						"supervised":      {Type: "bool", Default: true},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "actor without name should fail",
			decorator: &core.RegisteredDecorator{
				Type:    "fault_tolerance",
				Handler: actorHandler,
			},
			wantErr:     true,
			errContains: "decorator name is required",
		},
		{
			name: "actor without handler should succeed but not be invokable",
			decorator: &core.RegisteredDecorator{
				Name: "Actor",
				Type: "fault_tolerance",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := core.NewDecoratorRegistry(nil)
			err := registry.Register(tt.decorator)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tt.errContains != "" && !common.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify registration
			registered, err := registry.Get("Actor")
			if err != nil {
				t.Errorf("failed to get registered decorator: %v", err)
				return
			}

			if registered.Name != "Actor" {
				t.Errorf("expected name 'Actor', got %q", registered.Name)
			}
			if registered.Type != "fault_tolerance" {
				t.Errorf("expected type 'fault_tolerance', got %q", registered.Type)
			}
		})
	}
}

func TestActorDecorator_BasicCreation(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	tests := []struct {
		name         string
		properties   map[string]interface{}
		wantErr      bool
		wantMetadata map[string]interface{}
	}{
		{
			name: "default actor configuration",
			properties: map[string]interface{}{},
			wantErr: false,
			wantMetadata: map[string]interface{}{
				"actor_type":     "basic",
				"mailbox_size":   1000,
				"memory_pooled":  true,
				"supervised":     true,
			},
		},
		{
			name: "custom mailbox size",
			properties: map[string]interface{}{
				"mailboxSize": 5000,
			},
			wantErr: false,
			wantMetadata: map[string]interface{}{
				"mailbox_size": 5000,
				"memory_pooled": true,
			},
		},
		{
			name: "unsupervised actor",
			properties: map[string]interface{}{
				"supervised": false,
			},
			wantErr: false,
			wantMetadata: map[string]interface{}{
				"supervised": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:     mockActorTarget(),
				Arguments:  []interface{}{},
				Properties: tt.properties,
			}

			result, err := registry.Invoke(ctx, "Actor", args)
			if err != nil {
				t.Errorf("unexpected error during invocation: %v", err)
				return
			}

			if tt.wantErr {
				if result.Success {
					t.Errorf("expected failure but got success")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success but got failure: %s", result.Error)
				return
			}

			// Verify expected metadata
			for key, expectedValue := range tt.wantMetadata {
				if actualValue, exists := result.Metadata[key]; !exists {
					t.Errorf("expected metadata key %q not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestActorDecorator_MemoryPooling(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	tests := []struct {
		name         string
		poolSize     int
		expectedPool map[string]interface{}
	}{
		{
			name:     "small pool",
			poolSize: 5,
			expectedPool: map[string]interface{}{
				"initial_size": 5,
				"max_size":     50,
				"growth_factor": 2.0,
			},
		},
		{
			name:     "large pool",
			poolSize: 100,
			expectedPool: map[string]interface{}{
				"initial_size": 100,
				"max_size":     1000,
				"growth_factor": 1.5,
			},
		},
		{
			name:     "default pool",
			poolSize: 10,
			expectedPool: map[string]interface{}{
				"initial_size": 10,
				"max_size":     100,
				"growth_factor": 2.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:    mockActorTarget(),
				Arguments: []interface{}{},
				Properties: map[string]interface{}{
					"poolSize": tt.poolSize,
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

			// Verify memory pool configuration
			memoryPool, exists := result.Metadata["memory_pool"]
			if !exists {
				t.Errorf("missing memory_pool metadata")
				return
			}

			poolMap, ok := memoryPool.(map[string]interface{})
			if !ok {
				t.Errorf("memory_pool is not a map")
				return
			}

			for key, expectedValue := range tt.expectedPool {
				if actualValue, exists := poolMap[key]; !exists {
					t.Errorf("expected pool key %q not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected pool[%q] = %v, got %v", key, expectedValue, actualValue)
				}
			}

			// Verify pooling is enabled
			if memoryPooled, exists := result.Metadata["memory_pooled"]; !exists {
				t.Errorf("missing memory_pooled metadata")
			} else if memoryPooled != true {
				t.Errorf("expected memory_pooled to be true, got %v", memoryPooled)
			}
		})
	}
}

func TestActorDecorator_MessageHandling(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	tests := []struct {
		name            string
		mailboxSize     int
		messageTimeout  string
		expectedHandling map[string]interface{}
	}{
		{
			name:           "high throughput configuration",
			mailboxSize:    10000,
			messageTimeout: "100ms",
			expectedHandling: map[string]interface{}{
				"mailbox_type":     "bounded",
				"overflow_policy":  "drop_oldest",
				"timeout_policy":   "fail_fast",
			},
		},
		{
			name:           "low latency configuration",
			mailboxSize:    100,
			messageTimeout: "1ms",
			expectedHandling: map[string]interface{}{
				"mailbox_type":     "bounded",
				"overflow_policy":  "backpressure",
				"timeout_policy":   "fail_fast",
			},
		},
		{
			name:           "reliable configuration",
			mailboxSize:    1000,
			messageTimeout: "5s",
			expectedHandling: map[string]interface{}{
				"mailbox_type":     "bounded",
				"overflow_policy":  "block",
				"timeout_policy":   "retry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:    mockActorTarget(),
				Arguments: []interface{}{},
				Properties: map[string]interface{}{
					"mailboxSize":    tt.mailboxSize,
					"messageTimeout": tt.messageTimeout,
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

			// Verify message handling configuration
			messageHandling, exists := result.Metadata["message_handling"]
			if !exists {
				t.Errorf("missing message_handling metadata")
				return
			}

			handlingMap, ok := messageHandling.(map[string]interface{})
			if !ok {
				t.Errorf("message_handling is not a map")
				return
			}

			for key, expectedValue := range tt.expectedHandling {
				if actualValue, exists := handlingMap[key]; !exists {
					t.Errorf("expected handling key %q not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected handling[%q] = %v, got %v", key, expectedValue, actualValue)
				}
			}

			// Verify mailbox size
			if mailboxSize, exists := result.Metadata["mailbox_size"]; !exists {
				t.Errorf("missing mailbox_size metadata")
			} else if mailboxSize != tt.mailboxSize {
				t.Errorf("expected mailbox_size to be %v, got %v", tt.mailboxSize, mailboxSize)
			}
		})
	}
}

func TestActorDecorator_Supervision(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	tests := []struct {
		name           string
		supervised     bool
		supervisorName string
		expectSupervised bool
	}{
		{
			name:             "supervised actor with named supervisor",
			supervised:       true,
			supervisorName:   "MainSupervisor",
			expectSupervised: true,
		},
		{
			name:             "supervised actor with default supervisor",
			supervised:       true,
			supervisorName:   "",
			expectSupervised: true,
		},
		{
			name:             "unsupervised actor",
			supervised:       false,
			supervisorName:   "",
			expectSupervised: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			properties := map[string]interface{}{
				"supervised": tt.supervised,
			}
			if tt.supervisorName != "" {
				properties["supervisor"] = tt.supervisorName
			}

			args := core.DecoratorArgs{
				Target:     mockActorTarget(),
				Arguments:  []interface{}{},
				Properties: properties,
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

			// Verify supervision status
			if supervised, exists := result.Metadata["supervised"]; !exists {
				t.Errorf("missing supervised metadata")
			} else if supervised != tt.expectSupervised {
				t.Errorf("expected supervised to be %v, got %v", tt.expectSupervised, supervised)
			}

			// Verify supervisor name if supervised
			if tt.expectSupervised {
				supervisorName, exists := result.Metadata["supervisor_name"]
				if !exists {
					t.Errorf("missing supervisor_name metadata for supervised actor")
				} else {
					expectedName := tt.supervisorName
					if expectedName == "" {
						expectedName = "default_supervisor"
					}
					if supervisorName != expectedName {
						t.Errorf("expected supervisor_name to be %q, got %q", expectedName, supervisorName)
					}
				}
			}
		})
	}
}

func TestActorDecorator_ConcurrentAccess(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	// Test concurrent actor creation
	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make(chan core.DecoratorResult, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:    mockActorTarget(),
				Arguments: []interface{}{},
				Properties: map[string]interface{}{
					"actorId": id,
					"poolSize": 10,
				},
			}

			result, err := registry.Invoke(ctx, "Actor", args)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}

	// Verify all actors were created successfully
	successCount := 0
	for result := range results {
		if result.Success {
			successCount++
		} else {
			t.Errorf("actor creation failed: %s", result.Error)
		}
	}

	if successCount != numGoroutines {
		t.Errorf("expected %d successful actor creations, got %d", numGoroutines, successCount)
	}
}

func TestActorDecorator_Performance(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	// Test actor creation performance
	ctx := context.Background()
	args := core.DecoratorArgs{
		Target:    mockActorTarget(),
		Arguments: []interface{}{},
		Properties: map[string]interface{}{
			"poolSize":    100,
			"mailboxSize": 10000,
		},
	}

	const numIterations = 1000
	start := time.Now()
	
	for i := 0; i < numIterations; i++ {
		result, err := registry.Invoke(ctx, "Actor", args)
		if err != nil {
			t.Errorf("performance test error: %v", err)
			return
		}
		if !result.Success {
			t.Errorf("performance test failure: %s", result.Error)
			return
		}
	}
	
	duration := time.Since(start)
	avgDuration := duration / numIterations

	// Actor creation should be very fast (< 1ms per creation)
	if avgDuration > time.Millisecond {
		t.Errorf("actor creation too slow: %v per creation (expected < 1ms)", avgDuration)
	}

	t.Logf("Actor creation performance: %v per creation (%d iterations in %v)", 
		avgDuration, numIterations, duration)
}

func TestActorDecorator_ErrorHandling(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	actor := &core.RegisteredDecorator{
		Name:    "Actor",
		Type:    "fault_tolerance",
		Handler: actorHandler,
	}
	registry.Register(actor)

	tests := []struct {
		name       string
		properties map[string]interface{}
		wantErr    bool
		errMsg     string
	}{
		{
			name: "negative mailbox size",
			properties: map[string]interface{}{
				"mailboxSize": -100,
			},
			wantErr: true,
			errMsg:  "mailbox size must be positive",
		},
		{
			name: "invalid timeout format",
			properties: map[string]interface{}{
				"timeout": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name: "zero pool size",
			properties: map[string]interface{}{
				"poolSize": 0,
			},
			wantErr: true,
			errMsg:  "pool size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:     mockActorTarget(),
				Arguments:  []interface{}{},
				Properties: tt.properties,
			}

			result, err := registry.Invoke(ctx, "Actor", args)
			if err != nil {
				t.Errorf("unexpected error during invocation: %v", err)
				return
			}

			if tt.wantErr {
				if result.Success {
					t.Errorf("expected failure but got success")
					return
				}
				if result.Error == "" {
					t.Errorf("expected error message but got empty string")
					return
				}
				if !common.Contains(result.Error, tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, result.Error)
				}
			} else {
				if !result.Success {
					t.Errorf("expected success but got failure: %s", result.Error)
				}
			}
		})
	}
}

// Mock actor target for testing
func mockActorTarget() interface{} {
	return map[string]interface{}{
		"type":     "actor",
		"function": "processMessages",
		"source":   "test_file.go",
	}
}


// actorHandler will be implemented in actor.go
var actorHandler = ActorDecorator