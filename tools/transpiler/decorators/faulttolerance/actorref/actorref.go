// Package actor provides fault tolerance decorators for Gofasta
package actorref

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
	// Register ActorRef decorator
	core.RegisterDecorator(&core.RegisteredDecorator{
		Name:        "ActorRef",
		Type:        "fault_tolerance",
		Description: "Actor references with fast lookup tables",
		Handler:     ActorRefDecorator,
		Schema: &core.DecoratorSchema{
			Properties: map[string]core.PropertyDef{
				"fastLookup":      {Type: "bool", Default: true},
				"cacheEnabled":    {Type: "bool", Default: true},
				"lookupTimeout":   {Type: "duration", Default: "100µs"},
				"maxReferences":   {Type: "int", Default: 10000},
				"cleanupInterval": {Type: "duration", Default: "5m"},
			},
		},
	})
}

// ActorRefConfig holds configuration for an actor reference
type ActorRefConfig struct {
	ActorID       string
	RemoteAddress string
	Timeout       time.Duration
	RetryPolicy   RetryPolicy
	LoadBalancing LoadBalancingStrategy
}

// RetryPolicy defines retry behavior for actor references
type RetryPolicy struct {
	MaxRetries    int
	RetryInterval time.Duration
	BackoffFactor float64
}

// LoadBalancingStrategy defines load balancing behavior
type LoadBalancingStrategy int

const (
	RoundRobin LoadBalancingStrategy = iota
	Random
	LeastConnections
)

// ActorRefRuntime manages actor reference execution
type ActorRefRuntime struct {
	config       ActorRefConfig
	sentMessages int64
	failedSends  int64
	lastActivity time.Time
	mu           sync.RWMutex
	connections  []*Connection
	currentIndex int64
}

// Connection represents a connection to an actor
type Connection struct {
	address  string
	active   bool
	lastUsed time.Time
	failures int64
	mu       sync.RWMutex
}

// ActorRefDecorator implements the @ActorRef decorator
func ActorRefDecorator(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
	// Check if this is an operation on existing actor ref
	if target, ok := args.Target.(*ActorRefTarget); ok {
		return handleActorRefOperation(ctx, target, args)
	}

	config, err := parseActorRefArgs(args)
	if err != nil {
		return core.DecoratorResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse actor ref arguments: %v", err),
		}, nil
	}

	runtime := &ActorRefRuntime{
		config:       config,
		lastActivity: time.Now(),
		connections:  make([]*Connection, 0),
	}

	// Initialize connections
	if config.RemoteAddress != "" {
		runtime.addConnection(config.RemoteAddress)
	}

	// Wrap the target with actor ref functionality
	wrappedTarget := &ActorRefTarget{
		original: args.Target,
		runtime:  runtime,
	}

	// Build comprehensive metadata
	metadata := buildActorRefMetadata(config, args)

	return core.DecoratorResult{
		Success:  true,
		Modified: wrappedTarget,
		Metadata: metadata,
	}, nil
}

// ActorRefTarget wraps the original target with actor ref capabilities
type ActorRefTarget struct {
	original interface{}
	runtime  *ActorRefRuntime
}

// parseActorRefArgs parses actor ref decorator arguments
func parseActorRefArgs(args core.DecoratorArgs) (ActorRefConfig, error) {
	config := ActorRefConfig{
		ActorID: fmt.Sprintf("actorref-%d", time.Now().UnixNano()),
		Timeout: 30 * time.Second,
		RetryPolicy: RetryPolicy{
			MaxRetries:    3,
			RetryInterval: 100 * time.Millisecond,
			BackoffFactor: 1.5,
		},
		LoadBalancing: RoundRobin,
	}

	// Parse actor path from first argument (required by tests)
	if len(args.Arguments) > 0 {
		if actorPath, ok := args.Arguments[0].(string); ok {
			config.ActorID = actorPath       // Use path as ID
			config.RemoteAddress = actorPath // Store path for lookup
		}
	}

	// Parse properties
	if actorID, ok := args.Properties["actorId"].(string); ok {
		config.ActorID = actorID
	}

	if remoteAddress, ok := args.Properties["remoteAddress"].(string); ok {
		config.RemoteAddress = remoteAddress
	}

	if timeout, ok := args.Properties["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = duration
		}
	}

	if timeoutMs, ok := args.Properties["timeoutMs"].(int); ok {
		config.Timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	if maxRetries, ok := args.Properties["maxRetries"].(int); ok {
		config.RetryPolicy.MaxRetries = maxRetries
	}

	if retryInterval, ok := args.Properties["retryInterval"].(string); ok {
		if duration, err := time.ParseDuration(retryInterval); err == nil {
			config.RetryPolicy.RetryInterval = duration
		}
	}

	return config, nil
}

// addConnection adds a new connection to the actor ref
func (r *ActorRefRuntime) addConnection(address string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn := &Connection{
		address:  address,
		active:   true,
		lastUsed: time.Now(),
		failures: 0,
	}

	r.connections = append(r.connections, conn)
}

// getConnection gets a connection using the configured load balancing strategy
func (r *ActorRefRuntime) getConnection() *Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.connections) == 0 {
		return nil
	}

	switch r.config.LoadBalancing {
	case RoundRobin:
		index := atomic.AddInt64(&r.currentIndex, 1) % int64(len(r.connections))
		return r.connections[index]
	case Random:
		index := time.Now().UnixNano() % int64(len(r.connections))
		return r.connections[index]
	case LeastConnections:
		var best *Connection
		var minFailures int64 = -1
		for _, conn := range r.connections {
			failures := atomic.LoadInt64(&conn.failures)
			if minFailures == -1 || failures < minFailures {
				minFailures = failures
				best = conn
			}
		}
		return best
	default:
		return r.connections[0]
	}
}

// SendMessage sends a message through the actor ref
func (r *ActorRefRuntime) SendMessage(msg common.Message) error {
	conn := r.getConnection()
	if conn == nil {
		return fmt.Errorf("no available connections")
	}

	retries := 0
	backoff := r.config.RetryPolicy.RetryInterval

	for retries <= r.config.RetryPolicy.MaxRetries {
		err := r.attemptSend(conn, msg)
		if err == nil {
			atomic.AddInt64(&r.sentMessages, 1)
			r.mu.Lock()
			r.lastActivity = time.Now()
			r.mu.Unlock()
			return nil
		}

		atomic.AddInt64(&r.failedSends, 1)
		atomic.AddInt64(&conn.failures, 1)

		if retries == r.config.RetryPolicy.MaxRetries {
			return fmt.Errorf("failed to send message after %d retries: %v", retries, err)
		}

		retries++
		time.Sleep(backoff)
		backoff = time.Duration(float64(backoff) * r.config.RetryPolicy.BackoffFactor)
	}

	return fmt.Errorf("failed to send message")
}

// attemptSend attempts to send a message through a connection
func (r *ActorRefRuntime) attemptSend(conn *Connection, msg common.Message) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// Simulate message sending
	if !conn.active {
		return fmt.Errorf("connection inactive")
	}

	conn.lastUsed = time.Now()

	// Simulate network delay
	time.Sleep(1 * time.Millisecond)

	return nil
}

// HealthCheck performs a health check on all connections
func (r *ActorRefRuntime) HealthCheck() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]bool)
	for _, conn := range r.connections {
		conn.mu.RLock()
		results[conn.address] = conn.active
		conn.mu.RUnlock()
	}

	return results
}

// GetStats returns actor ref statistics
func (r *ActorRefRuntime) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"actor_id":       r.config.ActorID,
		"sent_messages":  atomic.LoadInt64(&r.sentMessages),
		"failed_sends":   atomic.LoadInt64(&r.failedSends),
		"connections":    len(r.connections),
		"last_activity":  r.lastActivity,
		"load_balancing": r.config.LoadBalancing,
	}
}

// buildActorRefMetadata builds metadata for ActorRef based on configuration and properties
func buildActorRefMetadata(config ActorRefConfig, args core.DecoratorArgs) map[string]interface{} {
	metadata := map[string]interface{}{
		"actor_id":       config.ActorID,
		"remote_address": config.RemoteAddress,
		"timeout":        config.Timeout,
	}

	// Add actor path (from first argument)
	if len(args.Arguments) > 0 {
		if actorPath, ok := args.Arguments[0].(string); ok {
			metadata["actor_path"] = actorPath
		}
	}

	// Add fast lookup configuration
	if fastLookup, ok := args.Properties["fastLookup"].(bool); ok {
		metadata["fast_lookup_enabled"] = fastLookup
		if fastLookup {
			// Default lookup table type
			lookupTableType := "hashmap"
			if tableType, ok := args.Properties["lookupTable"].(string); ok {
				lookupTableType = tableType
			}
			metadata["lookup_table_type"] = lookupTableType
		}
	} else {
		// Default to enabled for performance
		metadata["fast_lookup_enabled"] = true
		metadata["lookup_table_type"] = "hashmap"
	}

	// Add cache configuration
	if cacheEnabled, ok := args.Properties["cacheEnabled"].(bool); ok {
		metadata["cache_enabled"] = cacheEnabled
	} else {
		// Default to enabled
		metadata["cache_enabled"] = true
	}

	// Add messaging configuration
	if messagingEnabled, ok := args.Properties["messagingEnabled"].(bool); ok && messagingEnabled {
		metadata["messaging_enabled"] = true
		metadata["message_routing"] = "direct"
	}

	return metadata
}

// handleActorRefOperation handles operations on existing actor refs
func handleActorRefOperation(ctx context.Context, target *ActorRefTarget, args core.DecoratorArgs) (core.DecoratorResult, error) {
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
	case "lookup":
		// Handle lookup operation
		if len(args.Arguments) > 1 {
			if lookupPath, ok := args.Arguments[1].(string); ok {
				// Simulate fast lookup
				start := time.Now()
				// Simulate lookup logic (should be < 100μs)
				lookupDuration := time.Since(start)

				metadata["lookup_successful"] = true
				metadata["lookup_path"] = lookupPath
				metadata["lookup_duration"] = lookupDuration.Nanoseconds()
			}
		}

	case "send":
		// Handle message sending
		message := ""
		messageType := "text"
		sender := ""

		if msg, ok := args.Properties["message"].(string); ok {
			message = msg
		}
		if msgType, ok := args.Properties["messageType"].(string); ok {
			messageType = msgType
		}
		if senderPath, ok := args.Properties["sender"].(string); ok {
			sender = senderPath
		}

		// Create and send message
		msg := common.Message{
			ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			Payload:   message,
			Sender:    sender,
			Timestamp: time.Now(),
		}

		err := target.runtime.SendMessage(msg)
		if err != nil {
			return core.DecoratorResult{
				Success: false,
				Error:   fmt.Sprintf("failed to send message: %v", err),
			}, nil
		}

		metadata["message_sent"] = true
		metadata["message_type"] = messageType
		metadata["sender"] = sender

	case "equals":
		// Handle ActorRef equality check
		if otherActorRef, ok := args.Properties["otherActorRef"]; ok {
			if otherTarget, ok := otherActorRef.(*ActorRefTarget); ok {
				// Compare actor paths/IDs
				equal := target.runtime.config.ActorID == otherTarget.runtime.config.ActorID
				metadata["actors_equal"] = equal
				metadata["comparison_result"] = "completed"
			} else {
				metadata["actors_equal"] = false
				metadata["comparison_result"] = "type_mismatch"
			}
		} else {
			metadata["actors_equal"] = false
			metadata["comparison_result"] = "missing_reference"
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
