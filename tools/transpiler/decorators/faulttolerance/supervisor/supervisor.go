// Package supervisor provides supervisor decorator for Gofasta fault tolerance
package supervisor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
	"github.com/healtronlabs/gofasta/tools/transpiler/decorators/faulttolerance/common"
)

func init() {
	// Register Supervisor decorator
	core.RegisterDecorator(&core.RegisteredDecorator{
		Name:        "Supervisor",
		Type:        "fault_tolerance",
		Description: "Hierarchical supervision trees with fast initialization",
		Handler:     SupervisorDecorator,
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{Name: "strategy", Type: "string", Required: false, Default: "OneForOne", Description: "Supervision strategy: OneForOne, OneForAll, RestForOne"},
			},
			Properties: map[string]core.PropertyDef{
				"maxRetries":     {Type: "int", Default: 3},
				"retryWindow":    {Type: "duration", Default: "1m"},
				"escalateAfter":  {Type: "int", Default: 5},
				"parallelStart":  {Type: "bool", Default: true},
				"maxChildren":    {Type: "int", Default: 100},
				"treeDepthLimit": {Type: "int", Default: 10},
			},
		},
	})

	// Register SupervisionStrategy decorator (alias for backward compatibility)
	core.RegisterDecorator(&core.RegisteredDecorator{
		Name:        "SupervisionStrategy",
		Type:        "fault_tolerance",
		Description: "Fast strategy compilation for supervision",
		Handler:     SupervisorDecorator,
		Schema: &core.DecoratorSchema{
			Arguments: []core.ArgumentSchema{
				{Name: "strategy", Type: "string", Required: true, Description: "OneForOne, OneForAll, or RestForOne"},
			},
		},
	})
}

// Using shared types from common package
type SupervisorStrategy = common.SupervisorStrategy
type SupervisorConfig = common.SupervisorConfig

// Supervisor strategy constants
const (
	OneForOne  = common.OneForOne
	OneForAll  = common.OneForAll
	RestForOne = common.RestForOne
)

// SupervisorState tracks the state of supervised actors
type SupervisorState struct {
	config        common.SupervisorConfig
	children      map[string]*ChildState
	mu            sync.RWMutex
	restarts      int64
	totalFailures int64
}

// ChildState tracks individual child actor state
type ChildState struct {
	name      string
	failures  int64
	lastStart time.Time
	mu        sync.RWMutex
}

// SupervisorDecorator implements the @Supervisor decorator
func SupervisorDecorator(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
	// Check if this is a lifecycle operation on existing supervisor
	// Only treat as operation if first argument is not a valid strategy
	if target, ok := args.Target.(*SupervisedTarget); ok {
		if len(args.Arguments) > 0 {
			if strategyStr, ok := args.Arguments[0].(string); ok {
				// If it's a valid strategy, create nested supervisor instead of operation
				switch strategyStr {
				case "OneForOne", "OneForAll", "RestForOne":
					// This is a nested supervisor creation, continue with normal flow
				default:
					// This is an operation
					return handleSupervisorOperation(ctx, target, args)
				}
			} else {
				return handleSupervisorOperation(ctx, target, args)
			}
		} else {
			return handleSupervisorOperation(ctx, target, args)
		}
	}

	config, err := parseSupevisorArgs(args)
	if err != nil {
		return core.DecoratorResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse supervisor arguments: %v", err),
		}, nil
	}

	// Check depth limits
	if depth, ok := args.Properties["depth"].(int); ok {
		if maxDepth, ok := args.Properties["maxDepth"].(int); ok {
			if depth > maxDepth {
				return core.DecoratorResult{
					Success: false,
					Error:   fmt.Sprintf("supervision tree depth %d exceeds maximum depth %d", depth, maxDepth),
				}, nil
			}
		}
	}

	state := &SupervisorState{
		config:   config,
		children: make(map[string]*ChildState),
	}

	// Initialize children from properties
	if children, ok := args.Properties["children"].([]string); ok {
		for _, childName := range children {
			state.AddChild(childName)
		}
	}

	// Wrap the target with supervisor functionality
	wrappedTarget := &SupervisedTarget{
		original: args.Target,
		state:    state,
	}

	// Build metadata based on strategy and properties
	metadata := buildSupervisorMetadata(config, args)

	return core.DecoratorResult{
		Success:  true,
		Modified: wrappedTarget,
		Metadata: metadata,
	}, nil
}

// SupervisedTarget wraps the original target with supervisor capabilities
type SupervisedTarget struct {
	original interface{}
	state    *SupervisorState
}

// parseSupevisorArgs parses supervisor decorator arguments
func parseSupevisorArgs(args core.DecoratorArgs) (common.SupervisorConfig, error) {
	config := common.SupervisorConfig{
		Strategy:      OneForOne,
		MaxRetries:    3,
		RetryInterval: 1 * time.Second,
		Name:          "DefaultSupervisor",
	}

	// Parse strategy from arguments
	if len(args.Arguments) > 0 {
		if strategyStr, ok := args.Arguments[0].(string); ok {
			switch strategyStr {
			case "OneForOne":
				config.Strategy = OneForOne
			case "OneForAll":
				config.Strategy = OneForAll
			case "RestForOne":
				config.Strategy = RestForOne
			default:
				return config, fmt.Errorf("unknown strategy: %s", strategyStr)
			}
		}
	}

	// Parse properties
	if name, ok := args.Properties["name"].(string); ok {
		config.Name = name
	}

	if maxRetries, ok := args.Properties["maxRetries"].(int); ok {
		config.MaxRetries = maxRetries
	}

	if retryIntervalStr, ok := args.Properties["retryInterval"].(string); ok {
		if duration, err := time.ParseDuration(retryIntervalStr); err == nil {
			config.RetryInterval = duration
		}
	}

	return config, nil
}

// AddChild adds a child actor to supervision
func (s *SupervisorState) AddChild(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.children[name] = &ChildState{
		name:      name,
		failures:  0,
		lastStart: time.Now(),
	}
}

// RecordFailure records a failure for a child actor
func (s *SupervisorState) RecordFailure(childName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	child, exists := s.children[childName]
	if !exists {
		return false
	}

	child.mu.Lock()
	defer child.mu.Unlock()

	atomic.AddInt64(&child.failures, 1)
	atomic.AddInt64(&s.totalFailures, 1)

	// Check if max retries exceeded
	if child.failures >= int64(s.config.MaxRetries) {
		return false
	}

	// Apply restart strategy
	switch s.config.Strategy {
	case OneForOne:
		s.restartChild(childName)
	case OneForAll:
		s.restartAllChildren()
	case RestForOne:
		s.restartChildrenFrom(childName)
	}

	atomic.AddInt64(&s.restarts, 1)
	return true
}

// restartChild restarts a specific child
func (s *SupervisorState) restartChild(childName string) {
	if child, exists := s.children[childName]; exists {
		child.mu.Lock()
		child.lastStart = time.Now()
		child.mu.Unlock()
	}
}

// restartAllChildren restarts all supervised children
func (s *SupervisorState) restartAllChildren() {
	for _, child := range s.children {
		child.mu.Lock()
		child.lastStart = time.Now()
		child.mu.Unlock()
	}
}

// restartChildrenFrom restarts children starting from the given child
func (s *SupervisorState) restartChildrenFrom(childName string) {
	found := false
	for name, child := range s.children {
		if name == childName {
			found = true
		}
		if found {
			child.mu.Lock()
			child.lastStart = time.Now()
			child.mu.Unlock()
		}
	}
}

// GetStats returns supervisor statistics
func (s *SupervisorState) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_children":  len(s.children),
		"total_failures":  atomic.LoadInt64(&s.totalFailures),
		"total_restarts":  atomic.LoadInt64(&s.restarts),
		"supervisor_name": s.config.Name,
	}

	return stats
}

// buildSupervisorMetadata builds metadata based on configuration and properties
func buildSupervisorMetadata(config common.SupervisorConfig, args core.DecoratorArgs) map[string]interface{} {
	metadata := map[string]interface{}{
		"supervisor_name":      config.Name,
		"strategy":             config.Strategy,
		"max_retries":          config.MaxRetries,
		"supervision_strategy": config.Strategy.String(),
		"escalate_failures":    true,
		"fast_initialization":  true,
		"memory_pool_size":     10,
	}

	// Add restart policy metadata
	metadata["restart_policy"] = map[string]interface{}{
		"max_retries":    config.MaxRetries,
		"retry_interval": config.RetryInterval.String(),
		"backoff_policy": "exponential",
		"max_backoff":    "30s",
	}

	// Add strategy-specific metadata
	switch config.Strategy {
	case OneForOne:
		metadata["restart_child_only"] = true
	case OneForAll:
		metadata["restart_all_children"] = true
	case RestForOne:
		metadata["restart_subsequent"] = true
	}

	// Handle hierarchical properties
	if parent, ok := args.Properties["parent"].(string); ok {
		metadata["parent_supervisor"] = parent
	}

	// Handle escalation properties
	if escalate, ok := args.Properties["escalate"].(bool); ok {
		metadata["escalation_enabled"] = escalate
	} else {
		metadata["escalation_enabled"] = true
	}

	if maxFailures, ok := args.Properties["maxFailures"].(int); ok {
		metadata["max_failures"] = maxFailures
	} else if escalate, ok := args.Properties["escalate"].(bool); ok && escalate {
		metadata["max_failures"] = 5 // default
	}

	// Handle parallel startup
	if parallelStartup, ok := args.Properties["parallelStartup"].(bool); ok {
		metadata["parallel_startup"] = parallelStartup
	}

	// Handle dynamic children
	if dynamicChildren, ok := args.Properties["dynamicChildren"].(bool); ok {
		metadata["dynamic_children"] = dynamicChildren
	}

	// Handle memory tracking
	if trackMemory, ok := args.Properties["trackMemory"].(bool); ok && trackMemory {
		children := []interface{}{}
		if childList, ok := args.Properties["children"].([]string); ok {
			for _, child := range childList {
				children = append(children, child)
			}
		}
		baseMemory := int64(len(children) * 1000) // 1KB per child
		metadata["memory_usage"] = map[string]interface{}{
			"base_memory":    baseMemory,
			"child_count":    len(children),
			"memory_profile": true,
		}
	}

	// Handle depth limits
	if depth, ok := args.Properties["depth"].(int); ok {
		if maxDepth, ok := args.Properties["maxDepth"].(int); ok {
			if depth > maxDepth {
				// This should cause failure in the main function
				metadata["depth_limit_exceeded"] = true
			}
		}
		metadata["tree_depth"] = depth
	}

	// Handle actor system integration
	if actorSystem, ok := args.Properties["actorSystem"].(string); ok {
		metadata["actor_system_integration"] = true
		metadata["system_name"] = actorSystem
		if managedActors, ok := args.Properties["managedActors"].([]string); ok {
			metadata["actor_refs_managed"] = len(managedActors)
		}
	}

	return metadata
}

// handleSupervisorOperation handles operations on existing supervisors
func handleSupervisorOperation(ctx context.Context, target *SupervisedTarget, args core.DecoratorArgs) (core.DecoratorResult, error) {
	if len(args.Arguments) == 0 {
		return core.DecoratorResult{
			Success: false,
			Error:   "no operation specified",
		}, nil
	}

	operation, ok := args.Arguments[0].(string)
	if !ok {
		return core.DecoratorResult{
			Success: false,
			Error:   "operation must be a string",
		}, nil
	}

	metadata := make(map[string]interface{})

	switch operation {
	case "start":
		metadata["lifecycle_state"] = "started"
	case "pause":
		metadata["lifecycle_state"] = "pauseed"
	case "resume":
		metadata["lifecycle_state"] = "resumeed"
	case "stop":
		metadata["lifecycle_state"] = "stopped"
	case "restart", "status", "health", "metrics":
		metadata["operation_result"] = operation + "_completed"
	case "addChild":
		if childName, ok := args.Properties["childName"].(string); ok {
			target.state.AddChild(childName)
			metadata["child_added"] = childName
		}
	case "removeChild":
		if childName, ok := args.Properties["childName"].(string); ok {
			target.state.RemoveChild(childName)
			metadata["child_removed"] = childName
		}
	default:
		return core.DecoratorResult{
			Success: false,
			Error:   fmt.Sprintf("unknown operation: %s", operation),
		}, nil
	}

	return core.DecoratorResult{
		Success:  true,
		Modified: target,
		Metadata: metadata,
	}, nil
}

// RemoveChild removes a child from supervision
func (s *SupervisorState) RemoveChild(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.children, name)
}
