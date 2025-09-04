# Fault Tolerance Examples

This directory contains examples demonstrating the GoFasta fault tolerance framework with `@Supervisor` and `@Actor` decorators.

## Overview

The fault tolerance framework provides Akka-style supervision trees and actor model implementation with memory pooling for building resilient Go applications.

## Examples

### 1. Basic Supervisor (`basic-supervisor.gofa`)

Demonstrates fundamental supervisor usage with different restart strategies:

- **OneForOne Strategy**: Restart only the failed child
- **OneForAll Strategy**: Restart all children when one fails  
- **RestForOne Strategy**: Restart the failed child and subsequent children

**Key Features:**
- Basic supervisor configuration
- Error handling and restart policies
- Simple worker supervision

**Run Example:**
```bash
./gofasta transpile basic-supervisor.gofa
go run basic-supervisor.go
```

### 2. Advanced Supervisor (`advanced-supervisor.gofa`)

Shows hierarchical supervision trees with complex fault tolerance patterns:

- **Hierarchical Supervisors**: Multi-level supervision architecture
- **Different Strategies**: Each supervisor can use different restart strategies
- **Escalation Policies**: Failures can escalate up the supervision tree
- **Fast Initialization**: Optimized startup for critical components

**Architecture:**
```
RootSupervisor (OneForAll)
├── DatabaseSupervisor (RestForOne)
│   ├── PrimaryDB
│   ├── SecondaryDB
│   └── Cache
├── WebServerSupervisor (OneForOne)
│   ├── HTTP Server
│   ├── WebSocket Server
│   └── Health Check
└── BackgroundJobSupervisor (OneForOne)
    ├── Email Worker
    ├── Report Generator
    └── Data Cleanup
```

### 3. Basic Actor (`basic-actor.gofa`)

Demonstrates actor model implementation with memory pooling:

- **Memory Pooling**: Efficient memory management for high-throughput actors
- **Mailbox Configuration**: Different mailbox sizes for different use cases
- **Message Processing**: Async message handling patterns
- **Supervision Integration**: Actors can be supervised

**Actor Types:**
- **UserActor**: Standard user request processing (1000 mailbox, 10 pool)
- **PaymentActor**: High-throughput payment processing (10000 mailbox, 50 pool)
- **NotificationActor**: Low-latency notifications (100 mailbox, 5 pool)
- **AuditActor**: Reliable audit logging (5000 mailbox, 20 pool)
- **BatchProcessorActor**: Batch processing with periodic flushing

### 4. Advanced Actor System (`advanced-actor-system.gofa`)

Complex actor system simulating an e-commerce platform:

- **Multiple Actor Types**: Different actors for different business domains
- **Actor Interactions**: Actors processing different message types
- **Memory Optimization**: Tailored memory pooling for each actor type
- **Performance Characteristics**: Different timeout and throughput settings

**Business Domains:**
- **Order Processing**: High-volume order processing
- **Inventory Management**: Thread-safe inventory tracking  
- **Payment Processing**: Secure payment handling
- **Notifications**: High-volume notification delivery
- **Analytics**: Batch analytics processing
- **Fraud Detection**: Real-time fraud analysis
- **Customer Service**: Human-scale request handling

### 5. Supervisor-Actor Integration (`supervisor-actor-integration.gofa`)

Complete fault-tolerant system combining supervisors and actors:

- **Tiered Architecture**: Application, Web, Data, and Processing tiers
- **Supervised Actors**: All actors are supervised for fault tolerance
- **Cascading Restarts**: Failures can cascade through supervision hierarchy
- **Independent Critical Systems**: Critical monitoring runs independently

**System Architecture:**
```
ApplicationSupervisor (OneForAll)
├── WebTierSupervisor (OneForOne)
│   ├── HTTPHandlerActor
│   ├── WebSocketActor
│   └── SessionActor
├── DataTierSupervisor (RestForOne)
│   ├── DatabaseActor
│   ├── CacheActor
│   └── SearchIndexActor
└── ProcessingTierSupervisor (OneForOne)
    ├── OrderProcessingActor
    ├── PaymentProcessingActor
    ├── EmailActor
    └── ReportGeneratorActor

CriticalMonitoringSupervisor (OneForOne) [Independent]
├── SystemHealthActor
└── SecurityAuditActor
```

### 6. ActorRef Messaging (`actorref-messaging.gofa`)

Demonstrates fast actor-to-actor communication using ActorRefs:

- **Fast Lookup Tables**: Hash-based ActorRef lookup (< 100μs)
- **Actor Messaging**: Direct messaging between actors via references
- **Batch Sending**: Efficient broadcast messaging to multiple actors
- **Chat System Example**: Real-world messaging system demonstration

**Key Features:**
- Fast ActorRef lookups for chat rooms and users
- Message routing between chat server, rooms, and user sessions
- Demonstrates ActorRef performance optimization
- Shows practical actor communication patterns

### 7. ActorSystem Management (`actorsystem-management.gofa`)

Shows comprehensive ActorSystem management with parallel startup:

- **Parallel Actor Startup**: Concurrent initialization of thousands of actors
- **Actor Discovery**: Path-based actor lookup and discovery
- **Clustering Support**: Multi-node distributed actor systems
- **Graceful Shutdown**: Clean termination of entire actor systems
- **Game Server Example**: Real-world game server architecture

**Key Features:**
- Single-node and clustered ActorSystems
- Actor lifecycle management (create, stop, restart, terminate)
- System-level operations and monitoring
- Performance: < 1s startup for 10,000+ actors

### 8. Actor Advanced Features (`actor-advanced-features.gofa`)

Demonstrates advanced actor capabilities:

- **Hot-Swapping Behavior**: Runtime behavior changes based on conditions
- **State Management**: Persistent actor state with snapshots and recovery
- **Actor Hierarchy**: Parent actors spawning and managing children
- **Resource Cleanup**: Automatic resource management and cleanup
- **Backpressure Handling**: Managing mailbox overflow with different strategies

**Key Features:**
- Trading bot with behavior swapping based on market conditions
- Portfolio manager with persistent state across restarts
- Task manager spawning child worker actors
- Resource manager with automatic cleanup
- Data processor demonstrating backpressure patterns

### 9. Performance Benchmarks (`performance-benchmarks.gofa`)

Performance testing and benchmarking for supervisors and actors:

- **Supervisor Creation**: Benchmarking supervisor initialization speed
- **Actor Creation**: Testing actor startup performance
- **Message Throughput**: High-frequency message processing tests
- **Low Latency**: Minimal response time measurements
- **Concurrent Access**: Thread-safety under high concurrency
- **Memory Usage**: Memory pooling effectiveness demonstration

**Performance Targets:**
- Supervisor creation: < 10ms per supervisor
- Actor creation: < 1ms per actor
- Message throughput: > 10,000 messages/second
- Low latency: < 100μs response time
- Concurrent access: > 100,000 operations/second

## Decorator Configuration

### @Supervisor Parameters

- **strategy**: `"OneForOne"`, `"OneForAll"`, or `"RestForOne"`
- **maxRetries**: Maximum retry attempts before escalation (default: 3)
- **retryInterval**: Time between restart attempts (default: "1s")
- **escalate**: Whether to escalate failures to parent (default: true)
- **parent**: Name of parent supervisor for hierarchical trees
- **fastInit**: Enable fast initialization for critical components

### @Actor Parameters

- **mailboxSize**: Size of the actor's message mailbox (default: 1000)
- **poolSize**: Size of the memory pool (default: 10)
- **timeout**: Message processing timeout (default: "30s")
- **supervised**: Whether the actor should be supervised (default: true)
- **supervisor**: Name of the supervising supervisor
- **hotSwap**: Enable hot-swapping of actor behavior (default: false)
- **stateManagement**: Enable state management and persistence (default: false)
- **hierarchy**: Enable parent-child actor relationships (default: false)
- **resourceManagement**: Enable automatic resource cleanup (default: false)
- **backpressure**: Enable backpressure handling (default: false)
- **backpressureStrategy**: Strategy for handling mailbox overflow ("dropOldest", "dropNewest", "reject")

### @ActorRef Parameters

- **actorPath**: Path to the target actor (required)
- **fastLookup**: Enable fast lookup tables (default: true)
- **cacheEnabled**: Enable ActorRef caching (default: true)
- **timeoutMs**: Timeout for ActorRef operations in milliseconds (default: 5000)
- **lookupTable**: Type of lookup table ("hashmap", "btree", "radix")

### @ActorSystem Parameters

- **systemName**: Name of the actor system (required)
- **parallelStartup**: Enable parallel actor startup (default: true)
- **maxActors**: Maximum number of actors in the system (default: 10000)
- **clustering**: Enable clustering support (default: false)
- **remoting**: Enable remoting for distributed systems (default: false)
- **clusterNodes**: List of cluster node names
- **seedNodes**: List of seed nodes for cluster bootstrap
- **remotingPort**: Port for remoting communication (default: 2552)
- **gracefulShutdown**: Enable graceful shutdown (default: true)

## Performance Characteristics

### Supervisor Performance
- **Fast Initialization**: < 10ms for basic supervisors
- **Restart Time**: < 100ms for simple restarts
- **Hierarchical Overhead**: < 5ms per supervision level
- **Memory Overhead**: < 1MB per supervisor

### Actor Performance  
- **Message Throughput**: 10,000+ messages/second per actor
- **Memory Pooling**: 90% reduction in GC pressure
- **Startup Time**: < 1ms per actor
- **Concurrent Safety**: Full thread safety for all operations

## Error Simulation

All examples include simulated failures to demonstrate fault tolerance:

- **Database Failures**: Connection timeouts, query failures
- **Network Failures**: Service unavailable, connection refused
- **Memory Issues**: Out of memory, resource exhaustion
- **Processing Errors**: Validation failures, business logic errors
- **External Service Failures**: Payment gateways, email services

## Best Practices

1. **Supervision Strategy Selection**:
   - Use `OneForOne` for independent workers
   - Use `OneForAll` for tightly coupled components
   - Use `RestForOne` for ordered dependencies

2. **Actor Configuration**:
   - Size mailboxes based on expected message volume
   - Configure memory pools based on concurrent load
   - Set appropriate timeouts for business requirements

3. **Error Handling**:
   - Use defer/recover for graceful failure handling
   - Log failures for debugging and monitoring
   - Design for failure - assume components will fail

4. **Performance Tuning**:
   - Monitor memory usage and adjust pool sizes
   - Profile message processing times
   - Tune restart intervals based on failure patterns

## Building and Running

To transpile and run any example:

```bash
# Transpile the .gofa file to .go
./gofasta transpile example-name.gofa

# Run the generated Go code
go run example-name.go
```

To run all examples:

```bash
# Build the transpiler
cd tools/transpiler && go build -o gofasta ./cmd

# Run each example
./gofasta transpile basic-supervisor.gofa && go run basic-supervisor.go
./gofasta transpile advanced-supervisor.gofa && go run advanced-supervisor.go  
./gofasta transpile basic-actor.gofa && go run basic-actor.go
./gofasta transpile advanced-actor-system.gofa && go run advanced-actor-system.go
./gofasta transpile supervisor-actor-integration.gofa && go run supervisor-actor-integration.go
./gofasta transpile actorref-messaging.gofa && go run actorref-messaging.go
./gofasta transpile actorsystem-management.gofa && go run actorsystem-management.go
./gofasta transpile actor-advanced-features.gofa && go run actor-advanced-features.go
./gofasta transpile performance-benchmarks.gofa && go run performance-benchmarks.go
```

## Expected Output

When running the examples, you should see:

1. **Supervisor Startup**: Messages showing supervisor initialization
2. **Actor Creation**: Messages showing actors starting under supervision  
3. **Normal Processing**: Messages showing normal work being performed
4. **Failure Simulation**: Error messages when components fail
5. **Automatic Restart**: Messages showing components being restarted
6. **Graceful Shutdown**: Messages showing clean shutdown when examples complete

The fault tolerance framework ensures that temporary failures don't bring down the entire system, demonstrating the resilience patterns essential for production systems.