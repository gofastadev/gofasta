// Package fault_tolerance provides fault tolerance decorators for GoFasta
package fault_tolerance

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// SupervisorStrategy defines the restart strategy for supervised actors
type SupervisorStrategy int

const (
	OneForOne SupervisorStrategy = iota
	OneForAll
	RestForOne
)

// SupervisorConfig holds configuration for a supervisor
type SupervisorConfig struct {
	Strategy      SupervisorStrategy
	MaxRetries    int
	RetryInterval time.Duration
	Name          string
}

// SupervisorState tracks the state of supervised actors
type SupervisorState struct {
	config       SupervisorConfig
	children     map[string]*ChildState
	mu           sync.RWMutex
	restarts     int64
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
	config, err := parseSupevisorArgs(args)
	if err != nil {
		return core.DecoratorResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse supervisor arguments: %v", err),
		}, nil
	}

	state := &SupervisorState{
		config:   config,
		children: make(map[string]*ChildState),
	}

	// Wrap the target with supervisor functionality
	wrappedTarget := &SupervisedTarget{
		original: args.Target,
		state:    state,
	}

	// Build metadata based on strategy
	metadata := map[string]interface{}{
		"supervisor_name":      config.Name,
		"strategy":            config.Strategy,
		"max_retries":         config.MaxRetries,
		"supervision_strategy": strategyToString(config.Strategy),
		"escalate_failures":   true, // From properties
		"fast_initialization": true,
		"memory_pool_size":    10,
	}

	// Add restart policy metadata
	metadata["restart_policy"] = map[string]interface{}{
		"max_retries":     config.MaxRetries,
		"retry_interval":  config.RetryInterval.String(),
		"backoff_policy":  "exponential",
		"max_backoff":     "30s",
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

// strategyToString converts strategy enum to string
func strategyToString(strategy SupervisorStrategy) string {
	switch strategy {
	case OneForOne:
		return "OneForOne"
	case OneForAll:
		return "OneForAll" 
	case RestForOne:
		return "RestForOne"
	default:
		return "OneForOne"
	}
}

// parseSupevisorArgs parses supervisor decorator arguments
func parseSupevisorArgs(args core.DecoratorArgs) (SupervisorConfig, error) {
	config := SupervisorConfig{
		Strategy:      OneForOne,
		MaxRetries:    3,
		RetryInterval: 100 * time.Millisecond,
		Name:          "DefaultSupervisor",
	}

	// Parse strategy from arguments
	if len(args.Arguments) > 0 {
		if strategyStr, ok := args.Arguments[0].(string); ok {
			switch strings.ToLower(strategyStr) {
			case "oneforone":
				config.Strategy = OneForOne
			case "oneforall":
				config.Strategy = OneForAll
			case "restforone":
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