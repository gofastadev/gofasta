// Package fault_tolerance provides fault tolerance decorators for GoFasta
package fault_tolerance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

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
	config        ActorRefConfig
	sentMessages  int64
	failedSends   int64
	lastActivity  time.Time
	mu            sync.RWMutex
	connections   []*Connection
	currentIndex  int64
}

// Connection represents a connection to an actor
type Connection struct {
	address   string
	active    bool
	lastUsed  time.Time
	failures  int64
	mu        sync.RWMutex
}

// ActorRefDecorator implements the @ActorRef decorator
func ActorRefDecorator(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
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

	return core.DecoratorResult{
		Success:  true,
		Modified: wrappedTarget,
		Metadata: map[string]interface{}{
			"actor_id":       config.ActorID,
			"remote_address": config.RemoteAddress,
			"timeout":        config.Timeout,
		},
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

	// Parse arguments
	if len(args.Arguments) > 0 {
		if actorID, ok := args.Arguments[0].(string); ok {
			config.ActorID = actorID
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
func (r *ActorRefRuntime) SendMessage(msg Message) error {
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
func (r *ActorRefRuntime) attemptSend(conn *Connection, msg Message) error {
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
		"actor_id":        r.config.ActorID,
		"sent_messages":   atomic.LoadInt64(&r.sentMessages),
		"failed_sends":    atomic.LoadInt64(&r.failedSends),
		"connections":     len(r.connections),
		"last_activity":   r.lastActivity,
		"load_balancing":  r.config.LoadBalancing,
	}
}