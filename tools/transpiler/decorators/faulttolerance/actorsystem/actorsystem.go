// Package actor provides fault tolerance decorators for GoFasta
package actorsystem

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
	"github.com/healtronlabs/gofasta/tools/transpiler/decorators/faulttolerance/actor"
	"github.com/healtronlabs/gofasta/tools/transpiler/decorators/faulttolerance/actorref"
	"github.com/healtronlabs/gofasta/tools/transpiler/decorators/faulttolerance/common"
	"github.com/healtronlabs/gofasta/tools/transpiler/decorators/faulttolerance/supervisor"
)

// ActorSystemConfig holds configuration for an actor system
type ActorSystemConfig struct {
	Name           string
	MaxActors      int
	ClusterEnabled bool
	NodeID         string
	Discovery      DiscoveryConfig
	Persistence    PersistenceConfig
}

// DiscoveryConfig holds service discovery configuration
type DiscoveryConfig struct {
	Provider   string
	Endpoints  []string
	Interval   time.Duration
	TTL        time.Duration
}

// PersistenceConfig holds persistence configuration
type PersistenceConfig struct {
	Enabled    bool
	Provider   string
	Connection string
	BatchSize  int
}

// ActorSystemRuntime manages the actor system
type ActorSystemRuntime struct {
	config         ActorSystemConfig
	actors         map[string]common.ActorInterface      // Actor implementations
	supervisors    map[string]common.SupervisorInterface // Supervisor implementations  
	actorRefs      map[string]common.ActorRefInterface   // ActorRef implementations
	totalMessages  int64
	totalActors    int64
	totalErrors    int64
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	discovery      *ServiceDiscovery
	persistence    *PersistenceManager
}

// ServiceDiscovery manages service discovery
type ServiceDiscovery struct {
	config    DiscoveryConfig
	nodes     map[string]*NodeInfo
	mu        sync.RWMutex
	ticker    *time.Ticker
	stopChan  chan struct{}
}

// NodeInfo holds information about a cluster node
type NodeInfo struct {
	ID        string
	Address   string
	LastSeen  time.Time
	Active    bool
	Load      float64
}

// PersistenceManager manages persistence operations
type PersistenceManager struct {
	config    PersistenceConfig
	events    chan PersistenceEvent
	snapshots map[string][]byte
	mu        sync.RWMutex
}

// PersistenceEvent represents an event to be persisted
type PersistenceEvent struct {
	ActorID   string
	EventType string
	Data      []byte
	Timestamp time.Time
}

// ActorSystemDecorator implements the @ActorSystem decorator
func ActorSystemDecorator(ctx context.Context, args core.DecoratorArgs) (core.DecoratorResult, error) {
	config, err := parseActorSystemArgs(args)
	if err != nil {
		return core.DecoratorResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse actor system arguments: %v", err),
		}, nil
	}

	systemCtx, cancel := context.WithCancel(ctx)
	runtime := &ActorSystemRuntime{
		config:      config,
		actors:      make(map[string]common.ActorInterface),
		supervisors: make(map[string]common.SupervisorInterface),
		actorRefs:   make(map[string]common.ActorRefInterface),
		ctx:         systemCtx,
		cancel:      cancel,
	}

	// Initialize discovery if cluster is enabled
	if config.ClusterEnabled {
		runtime.discovery = NewServiceDiscovery(config.Discovery)
		runtime.discovery.Start(systemCtx)
	}

	// Initialize persistence if enabled
	if config.Persistence.Enabled {
		runtime.persistence = NewPersistenceManager(config.Persistence)
		runtime.persistence.Start(systemCtx)
	}

	// Wrap the target with actor system functionality
	wrappedTarget := &ActorSystemTarget{
		original: args.Target,
		runtime:  runtime,
	}

	return core.DecoratorResult{
		Success:  true,
		Modified: wrappedTarget,
		Metadata: map[string]interface{}{
			"system_name":      config.Name,
			"max_actors":       config.MaxActors,
			"cluster_enabled":  config.ClusterEnabled,
			"node_id":          config.NodeID,
		},
	}, nil
}

// ActorSystemTarget wraps the original target with actor system capabilities
type ActorSystemTarget struct {
	original interface{}
	runtime  *ActorSystemRuntime
}

// parseActorSystemArgs parses actor system decorator arguments
func parseActorSystemArgs(args core.DecoratorArgs) (ActorSystemConfig, error) {
	config := ActorSystemConfig{
		Name:           "DefaultActorSystem",
		MaxActors:      10000,
		ClusterEnabled: false,
		NodeID:         fmt.Sprintf("node-%d", time.Now().UnixNano()),
		Discovery: DiscoveryConfig{
			Provider:  "consul",
			Endpoints: []string{"localhost:8500"},
			Interval:  30 * time.Second,
			TTL:       60 * time.Second,
		},
		Persistence: PersistenceConfig{
			Enabled:    false,
			Provider:   "redis",
			Connection: "localhost:6379",
			BatchSize:  100,
		},
	}

	// Parse properties
	if name, ok := args.Properties["name"].(string); ok {
		config.Name = name
	}

	if maxActors, ok := args.Properties["maxActors"].(int); ok {
		config.MaxActors = maxActors
	}

	if clusterEnabled, ok := args.Properties["clusterEnabled"].(bool); ok {
		config.ClusterEnabled = clusterEnabled
	}

	if nodeID, ok := args.Properties["nodeId"].(string); ok {
		config.NodeID = nodeID
	}

	if persistenceEnabled, ok := args.Properties["persistenceEnabled"].(bool); ok {
		config.Persistence.Enabled = persistenceEnabled
	}

	return config, nil
}

// RegisterActor registers an actor in the system
func (r *ActorSystemRuntime) RegisterActor(id string, actor *actor.ActorRuntime) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.actors) >= r.config.MaxActors {
		return fmt.Errorf("maximum actors limit reached")
	}

	r.actors[id] = actor
	atomic.AddInt64(&r.totalActors, 1)

	return nil
}

// RegisterSupervisor registers a supervisor in the system
func (r *ActorSystemRuntime) RegisterSupervisor(name string, supervisor *supervisor.SupervisorState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.supervisors[name] = supervisor
}

// RegisterActorRef registers an actor ref in the system
func (r *ActorSystemRuntime) RegisterActorRef(id string, actorRef *actorref.ActorRefRuntime) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.actorRefs[id] = actorRef
}

// RouteMessage routes a message to the appropriate actor
func (r *ActorSystemRuntime) RouteMessage(actorID string, msg common.Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if actor, exists := r.actors[actorID]; exists {
		atomic.AddInt64(&r.totalMessages, 1)
		return actor.SendMessage(msg)
	}

	if actorRef, exists := r.actorRefs[actorID]; exists {
		atomic.AddInt64(&r.totalMessages, 1)
		return actorRef.SendMessage(msg)
	}

	return fmt.Errorf("actor not found: %s", actorID)
}

// Shutdown gracefully shuts down the actor system
func (r *ActorSystemRuntime) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop all actors
	for _, actor := range r.actors {
		actor.Stop()
	}

	// Stop discovery
	if r.discovery != nil {
		r.discovery.Stop()
	}

	// Stop persistence
	if r.persistence != nil {
		r.persistence.Stop()
	}

	// Cancel context
	if r.cancel != nil {
		r.cancel()
	}
}

// GetStats returns actor system statistics
func (r *ActorSystemRuntime) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"system_name":     r.config.Name,
		"total_actors":    len(r.actors),
		"total_supervisors": len(r.supervisors),
		"total_actor_refs": len(r.actorRefs),
		"total_messages":  atomic.LoadInt64(&r.totalMessages),
		"total_errors":    atomic.LoadInt64(&r.totalErrors),
		"cluster_enabled": r.config.ClusterEnabled,
		"node_id":         r.config.NodeID,
	}
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(config DiscoveryConfig) *ServiceDiscovery {
	return &ServiceDiscovery{
		config:   config,
		nodes:    make(map[string]*NodeInfo),
		ticker:   time.NewTicker(config.Interval),
		stopChan: make(chan struct{}),
	}
}

// Start starts the service discovery
func (sd *ServiceDiscovery) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-sd.ticker.C:
				sd.discover()
			case <-sd.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the service discovery
func (sd *ServiceDiscovery) Stop() {
	sd.ticker.Stop()
	close(sd.stopChan)
}

// discover performs service discovery
func (sd *ServiceDiscovery) discover() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Simulate discovery process
	now := time.Now()
	for _, node := range sd.nodes {
		if now.Sub(node.LastSeen) > sd.config.TTL {
			node.Active = false
		}
	}
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(config PersistenceConfig) *PersistenceManager {
	return &PersistenceManager{
		config:    config,
		events:    make(chan PersistenceEvent, config.BatchSize*2),
		snapshots: make(map[string][]byte),
	}
}

// Start starts the persistence manager
func (pm *PersistenceManager) Start(ctx context.Context) {
	go pm.eventLoop(ctx)
}

// Stop stops the persistence manager
func (pm *PersistenceManager) Stop() {
	close(pm.events)
}

// eventLoop processes persistence events
func (pm *PersistenceManager) eventLoop(ctx context.Context) {
	batch := make([]PersistenceEvent, 0, pm.config.BatchSize)

	for {
		select {
		case event, ok := <-pm.events:
			if !ok {
				return
			}
			batch = append(batch, event)
			if len(batch) >= pm.config.BatchSize {
				pm.processBatch(batch)
				batch = batch[:0]
			}
		case <-ctx.Done():
			return
		}
	}
}

// processBatch processes a batch of persistence events
func (pm *PersistenceManager) processBatch(batch []PersistenceEvent) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Simulate batch processing
	for _, event := range batch {
		pm.snapshots[event.ActorID] = event.Data
	}
}