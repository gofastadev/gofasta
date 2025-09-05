package supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
	"github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/common"
)

func TestSupervisorDecorator_Registration(t *testing.T) {
	tests := []struct {
		name        string
		decorator   *core.RegisteredDecorator
		wantErr     bool
		errContains string
	}{
		{
			name: "valid supervisor decorator",
			decorator: &core.RegisteredDecorator{
				Name:        "Supervisor",
				Type:        "fault_tolerance",
				Description: "Hierarchical supervision trees with fast initialization",
				Handler:     supervisorHandler,
				Schema: &core.DecoratorSchema{
					Arguments: []core.ArgumentSchema{
						{Name: "strategy", Type: "string", Required: true},
					},
					Properties: map[string]core.PropertyDef{
						"maxRetries":    {Type: "int", Default: 3},
						"retryInterval": {Type: "duration", Default: "1s"},
						"escalate":      {Type: "bool", Default: true},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "supervisor without name should fail",
			decorator: &core.RegisteredDecorator{
				Type:    "fault_tolerance",
				Handler: supervisorHandler,
			},
			wantErr:     true,
			errContains: "decorator name is required",
		},
		{
			name: "supervisor without handler should succeed but not be invokable",
			decorator: &core.RegisteredDecorator{
				Name: "Supervisor",
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
			registered, err := registry.Get("Supervisor")
			if err != nil {
				t.Errorf("failed to get registered decorator: %v", err)
				return
			}

			if registered.Name != "Supervisor" {
				t.Errorf("expected name 'Supervisor', got %q", registered.Name)
			}
			if registered.Type != "fault_tolerance" {
				t.Errorf("expected type 'fault_tolerance', got %q", registered.Type)
			}
		})
	}
}

func TestSupervisorDecorator_Strategies(t *testing.T) {
	tests := []struct {
		name       string
		strategy   string
		wantErr    bool
		wantOutput map[string]interface{}
	}{
		{
			name:     "OneForOne strategy",
			strategy: "OneForOne",
			wantErr:  false,
			wantOutput: map[string]interface{}{
				"supervision_strategy": "OneForOne",
				"restart_child_only":   true,
				"escalate_failures":    true,
			},
		},
		{
			name:     "OneForAll strategy",
			strategy: "OneForAll",
			wantErr:  false,
			wantOutput: map[string]interface{}{
				"supervision_strategy": "OneForAll",
				"restart_all_children": true,
				"escalate_failures":    true,
			},
		},
		{
			name:     "RestForOne strategy",
			strategy: "RestForOne",
			wantErr:  false,
			wantOutput: map[string]interface{}{
				"supervision_strategy": "RestForOne",
				"restart_subsequent":   true,
				"escalate_failures":    true,
			},
		},
		{
			name:     "invalid strategy should fail",
			strategy: "InvalidStrategy",
			wantErr:  true,
		},
		{
			name:     "empty strategy should fail",
			strategy: "",
			wantErr:  true,
		},
	}

	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:    mockSupervisorTarget(),
				Arguments: []interface{}{tt.strategy},
				Properties: map[string]interface{}{
					"maxRetries":    3,
					"retryInterval": "1s",
					"escalate":      true,
				},
			}

			result, err := registry.Invoke(ctx, "Supervisor", args)
			if err != nil {
				t.Errorf("unexpected error during invocation: %v", err)
				return
			}

			if tt.wantErr {
				if result.Success {
					t.Errorf("expected failure but got success")
				}
				if result.Error == "" {
					t.Errorf("expected error message but got empty string")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success but got failure: %s", result.Error)
				return
			}

			// Verify output contains expected strategy configuration
			for key, expectedValue := range tt.wantOutput {
				if actualValue, exists := result.Metadata[key]; !exists {
					t.Errorf("expected metadata key %q not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected metadata[%q] = %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestSupervisorDecorator_HierarchicalTrees(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	tests := []struct {
		name       string
		parentName string
		childName  string
		wantErr    bool
	}{
		{
			name:       "create supervisor hierarchy",
			parentName: "ParentSupervisor",
			childName:  "ChildSupervisor",
			wantErr:    false,
		},
		{
			name:       "deep hierarchy levels",
			parentName: "Level1Supervisor",
			childName:  "Level2Supervisor",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create parent supervisor
			parentArgs := core.DecoratorArgs{
				Target:    mockSupervisorTarget(),
				Arguments: []interface{}{"OneForOne"},
				Properties: map[string]interface{}{
					"name":     tt.parentName,
					"children": []string{tt.childName},
				},
			}

			parentResult, err := registry.Invoke(ctx, "Supervisor", parentArgs)
			if err != nil {
				t.Errorf("unexpected error creating parent: %v", err)
				return
			}

			if !parentResult.Success {
				t.Errorf("failed to create parent supervisor: %s", parentResult.Error)
				return
			}

			// Create child supervisor
			childArgs := core.DecoratorArgs{
				Target:    mockSupervisorTarget(),
				Arguments: []interface{}{"OneForAll"},
				Properties: map[string]interface{}{
					"name":   tt.childName,
					"parent": tt.parentName,
				},
			}

			childResult, err := registry.Invoke(ctx, "Supervisor", childArgs)
			if err != nil {
				t.Errorf("unexpected error creating child: %v", err)
				return
			}

			if !childResult.Success {
				t.Errorf("failed to create child supervisor: %s", childResult.Error)
				return
			}

			// Verify hierarchy structure
			if parentName, exists := childResult.Metadata["parent_supervisor"]; !exists {
				t.Errorf("child supervisor missing parent reference")
			} else if parentName != tt.parentName {
				t.Errorf("expected parent name %q, got %q", tt.parentName, parentName)
			}
		})
	}
}

func TestSupervisorDecorator_FastInitialization(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	ctx := context.Background()
	args := core.DecoratorArgs{
		Target:    mockSupervisorTarget(),
		Arguments: []interface{}{"OneForOne"},
		Properties: map[string]interface{}{
			"fastInit": true,
			"poolSize": 10,
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

	// Fast initialization should complete within 10ms
	if duration > 10*time.Millisecond {
		t.Errorf("initialization took too long: %v (expected < 10ms)", duration)
	}

	// Verify fast initialization metadata
	if fastInit, exists := result.Metadata["fast_initialization"]; !exists {
		t.Errorf("missing fast_initialization metadata")
	} else if fastInit != true {
		t.Errorf("expected fast_initialization to be true, got %v", fastInit)
	}

	if poolSize, exists := result.Metadata["memory_pool_size"]; !exists {
		t.Errorf("missing memory_pool_size metadata")
	} else if poolSize != 10 {
		t.Errorf("expected memory_pool_size to be 10, got %v", poolSize)
	}
}

func TestSupervisorDecorator_RestartPolicies(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	tests := []struct {
		name           string
		maxRetries     int
		retryInterval  string
		expectedPolicy map[string]interface{}
	}{
		{
			name:          "default restart policy",
			maxRetries:    3,
			retryInterval: "1s",
			expectedPolicy: map[string]interface{}{
				"max_retries":    3,
				"retry_interval": "1s",
				"backoff_policy": "exponential",
				"max_backoff":    "30s",
			},
		},
		{
			name:          "aggressive restart policy",
			maxRetries:    10,
			retryInterval: "100ms",
			expectedPolicy: map[string]interface{}{
				"max_retries":    10,
				"retry_interval": "100ms",
				"backoff_policy": "exponential",
				"max_backoff":    "30s",
			},
		},
		{
			name:          "conservative restart policy",
			maxRetries:    1,
			retryInterval: "5s",
			expectedPolicy: map[string]interface{}{
				"max_retries":    1,
				"retry_interval": "5s",
				"backoff_policy": "exponential",
				"max_backoff":    "30s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:    mockSupervisorTarget(),
				Arguments: []interface{}{"OneForOne"},
				Properties: map[string]interface{}{
					"maxRetries":    tt.maxRetries,
					"retryInterval": tt.retryInterval,
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

			// Verify restart policy configuration
			restartPolicy, exists := result.Metadata["restart_policy"]
			if !exists {
				t.Errorf("missing restart_policy metadata")
				return
			}

			policyMap, ok := restartPolicy.(map[string]interface{})
			if !ok {
				t.Errorf("restart_policy is not a map")
				return
			}

			for key, expectedValue := range tt.expectedPolicy {
				if actualValue, exists := policyMap[key]; !exists {
					t.Errorf("expected policy key %q not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected policy[%q] = %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestSupervisorDecorator_FailureEscalation(t *testing.T) {
	registry := core.NewDecoratorRegistry(nil)
	supervisor := &core.RegisteredDecorator{
		Name:    "Supervisor",
		Type:    "fault_tolerance",
		Handler: supervisorHandler,
	}
	registry.Register(supervisor)

	tests := []struct {
		name           string
		escalate       bool
		maxFailures    int
		wantEscalation bool
	}{
		{
			name:           "escalation enabled",
			escalate:       true,
			maxFailures:    5,
			wantEscalation: true,
		},
		{
			name:           "escalation disabled",
			escalate:       false,
			maxFailures:    5,
			wantEscalation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := core.DecoratorArgs{
				Target:    mockSupervisorTarget(),
				Arguments: []interface{}{"OneForOne"},
				Properties: map[string]interface{}{
					"escalate":    tt.escalate,
					"maxFailures": tt.maxFailures,
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

			// Verify escalation configuration
			if escalationEnabled, exists := result.Metadata["escalation_enabled"]; !exists {
				t.Errorf("missing escalation_enabled metadata")
			} else if escalationEnabled != tt.wantEscalation {
				t.Errorf("expected escalation_enabled to be %v, got %v", tt.wantEscalation, escalationEnabled)
			}

			if maxFailures, exists := result.Metadata["max_failures"]; !exists {
				t.Errorf("missing max_failures metadata")
			} else if maxFailures != tt.maxFailures {
				t.Errorf("expected max_failures to be %v, got %v", tt.maxFailures, maxFailures)
			}
		})
	}
}

// Mock supervisor target for testing
func mockSupervisorTarget() interface{} {
	return map[string]interface{}{
		"type":     "supervisor",
		"function": "handleRequests",
		"source":   "test_file.go",
	}
}

// supervisorHandler will be implemented in supervisor.go
var supervisorHandler = SupervisorDecorator
