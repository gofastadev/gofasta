# Fault Tolerance Integration Tests

This directory contains comprehensive integration tests for the GoFasta fault tolerance framework, covering all aspects of the @Supervisor, @Actor, @ActorRef, and @ActorSystem decorators working together in real-world scenarios.

## Test Structure

### Core Integration Tests

1. **`integration_test.go`** - Original comprehensive integration test covering basic decorator interactions
2. **`hierarchical_supervision_integration_test.go`** - Multi-level supervision trees and escalation patterns
3. **`actor_lifecycle_integration_test.go`** - Complete actor lifecycle with supervision, state management, and hierarchy
4. **`actorsystem_clustering_integration_test.go`** - Distributed ActorSystem behavior, clustering, and remoting
5. **`performance_stress_integration_test.go`** - High-volume performance and stress testing scenarios
6. **`circuit_breaker_integration_test.go`** - Circuit breaker patterns with supervision and adaptive behavior
7. **`end_to_end_integration_test.go`** - Complete system integration, real-world scenarios, and disaster recovery

## Test Categories

### Hierarchical Supervision Integration (`hierarchical_supervision_integration_test.go`)

Tests multi-level supervision trees and escalation patterns:

- **Three-level OneForAll supervision** with cascading restarts
- **Mixed strategy hierarchies** (OneForAll→OneForOne→RestForOne)
- **Deep supervision trees** (5+ levels) with complex escalation paths
- **Performance under load** (50 supervisors, 10K actors, 2% failure rate)
- **Escalation policies** (immediate, retry-first, circuit-breaker)
- **Memory pooling effectiveness** with 100 actors over multiple restart cycles

Key Features Tested:
- Hierarchical fault tolerance with different strategies at each level
- Escalation policies and failure propagation control
- Performance characteristics under high supervision load
- Circuit breaker integration with supervision trees
- Memory efficiency during restart cycles

### Actor Lifecycle Integration (`actor_lifecycle_integration_test.go`)

Tests complete actor lifecycle management with supervision:

- **Full lifecycle with supervision** (create→initialize→start→process→fail→restart→stop→terminate)
- **Stateful actors with persistence** (state management, snapshots, recovery)
- **Hot-swappable behavior** (normal→degraded→emergency behavior transitions)
- **Parent-child actor hierarchies** with dynamic spawning and resource management
- **Backpressure handling** (dropOldest, dropNewest, reject strategies)
- **ActorRef communication** (high-volume messaging, failure recovery, broadcast patterns)
- **Persistence integration** (event sourcing, snapshots, distributed persistence)

Key Features Tested:
- Actor state preservation across restarts
- Hot behavior swapping with <10ms latency
- Hierarchical resource management and cleanup
- Backpressure strategies under overload conditions
- High-volume ActorRef messaging (10K messages, >95% delivery rate)
- Distributed persistence with eventual consistency

### ActorSystem Clustering Integration (`actorsystem_clustering_integration_test.go`)

Tests distributed ActorSystem behavior and clustering:

- **Cluster formation** (3-node clusters with leader election)
- **Network partition handling** (split-brain prevention, automatic healing)
- **Cross-node supervision** (OneForAll supervision across cluster nodes)
- **Node isolation handling** (actor migration, supervision maintenance)
- **High-throughput remoting** (10K messages, compression, connection pooling)
- **Network resilience** (connection recovery, message ordering, duplicate detection)
- **Graceful shutdown** (coordinated vs. forced shutdown strategies)
- **Load balancing** (round-robin, capacity-aware distribution)

Key Features Tested:
- Cluster consensus and leader election
- Cross-node actor supervision and migration
- Remoting with compression and connection pooling
- Network partition detection and recovery
- Load balancing across heterogeneous nodes
- Graceful system shutdown with message draining

### Performance Stress Integration (`performance_stress_integration_test.go`)

Tests system behavior under extreme load and stress conditions:

- **High-volume performance** (50 supervisors, 10K actors, 100K msg/sec, 0.1% failure rate)
- **High-failure stress** (5% failure rate with >98% restart success)
- **Memory pressure** (10K actors with memory pooling, GC pressure control)
- **Concurrency stress** (1K concurrent actors, 10K reads, 1K writes)
- **Memory leak detection** (100 cycles × 1K actors with <10% memory growth)
- **Failure recovery stress** (rapid failure/recovery cycles, >99% success rate)
- **Resource exhaustion** (memory/connection pool exhaustion with recovery)

Key Features Tested:
- System stability under extreme load (100K operations/sec)
- Memory pooling effectiveness (>90% reuse, <10% GC pressure)
- Concurrent safety with high contention levels
- Memory leak prevention over extended cycles
- Resource exhaustion detection and recovery
- Sustained performance under continuous failures

### Circuit Breaker Integration (`circuit_breaker_integration_test.go`)

Tests circuit breaker patterns integrated with fault tolerance:

- **Circuit breaker + OneForAll supervision** (coordinated recovery)
- **Cascade prevention** with circuit breaker isolation
- **Adaptive thresholds** (learning from traffic patterns, 85% confidence)
- **Predictive circuit opening** (trend analysis, early failure detection)
- **Bulkhead patterns** (resource pool isolation, thread pool separation)
- **Timeout patterns** (adaptive timeout adjustment, cascade prevention)

Key Features Tested:
- Circuit breaker state coordination with supervision
- Adaptive threshold adjustment based on traffic analysis
- Bulkhead isolation effectiveness (>90% resource utilization)
- Predictive failure detection and early protection
- Hierarchical timeout management with cascade prevention
- Integration of multiple resilience patterns

### End-to-End Integration (`end_to_end_integration_test.go`)

Tests complete system integration in real-world scenarios:

- **Production-like workload** (20 supervisors, 1K actors, 10K req/sec, >99.9% availability)
- **Chaos engineering** (high chaos level, random failures, network partitions)
- **Microservice mesh** (6 services with cascading failure prevention)
- **High-frequency trading simulation** (<1ms latency, >1M ops/sec, zero message loss)
- **IoT device management** (100K+ device connections, 50K+ msg/sec)
- **Long-running stability** (24-hour and week-long endurance tests)
- **Disaster recovery** (datacenter failure, split-brain scenarios)

Key Features Tested:
- End-to-end system availability and performance
- Chaos engineering resilience testing
- Real-world scenario simulation (microservices, trading, IoT)
- Long-term stability and consistency
- Disaster recovery capabilities and RTO/RPO compliance
- Complete system behavior under realistic workloads

## Performance Targets

Based on the roadmap requirements, these integration tests verify:

### Supervisor Performance
- **Creation time**: < 10ms per supervisor
- **Restart time**: < 100ms for simple restarts
- **Hierarchical overhead**: < 5ms per supervision level
- **Memory overhead**: < 1MB per supervisor

### Actor Performance  
- **Message throughput**: 10,000+ messages/second per actor
- **Memory pooling**: 90% reduction in GC pressure
- **Startup time**: < 1ms per actor
- **Concurrent safety**: Full thread safety for all operations

### ActorRef Performance
- **Lookup time**: < 100μs per ActorRef lookup
- **Message delivery**: > 95% delivery rate under normal conditions
- **Batch operations**: Efficient broadcast to 100+ actors
- **Cache effectiveness**: > 90% cache hit rate

### ActorSystem Performance
- **Startup time**: < 1s for 10,000+ actors
- **Clustering latency**: < 100ms for cross-node operations
- **Remoting throughput**: > 10,000 messages/sec with compression
- **Graceful shutdown**: < 30s for coordinated cluster shutdown

## Test Execution

### Running Individual Test Suites

```bash
# Run hierarchical supervision tests
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestHierarchicalSupervisionScenarios

# Run actor lifecycle tests  
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestActorLifecycleIntegrationScenarios

# Run clustering tests
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestActorSystemClusteringScenarios

# Run performance stress tests
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestHighVolumePerformanceScenarios

# Run circuit breaker tests
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestCircuitBreakerIntegrationWithSupervision

# Run end-to-end tests
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestCompleteSystemIntegration
```

### Running All Integration Tests

```bash
# Run all integration tests
go test -v ./tests/integration/transpiler/decorators/fault_tolerance

# Run with race detection
go test -race -v ./tests/integration/transpiler/decorators/fault_tolerance

# Run with coverage
go test -cover -v ./tests/integration/transpiler/decorators/fault_tolerance
```

### Running Long-Duration Tests

```bash
# Run 24-hour stability test
go test -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestLongRunningStabilityTest

# Run with extended timeout
go test -timeout 25h -v ./tests/integration/transpiler/decorators/fault_tolerance -run TestLongRunningStabilityTest

# Skip long tests in CI
go test -short -v ./tests/integration/transpiler/decorators/fault_tolerance
```

## Test Architecture

### Helper Types

Each integration test file defines helper types that simulate real-world components:

- **Supervisors**: `TestSupervisor`, `DistributedSupervisor`, `LoadTestSupervisor`
- **Actors**: `TestActor`, `LifecycleActor`, `HierarchicalActor`, `LoadTestActor`
- **Systems**: `ActorSystem`, `ClusteredActorSystem`, `RealWorldScenarioSystem`
- **Monitors**: `PerformanceMonitor`, `ConcurrencyMonitor`, `ResourceMonitor`

### Monitoring and Metrics

All integration tests include comprehensive monitoring:

- **Performance metrics**: Throughput, latency, error rates
- **Resource tracking**: Memory, CPU, network, storage usage  
- **Failure analysis**: Failure patterns, recovery times, success rates
- **System health**: Component status, dependencies, availability

### Failure Injection

Tests include sophisticated failure injection mechanisms:

- **Failure patterns**: Burst, continuous, intermittent, cascading
- **Failure types**: Network, memory, disk, service, dependency
- **Recovery testing**: Automatic recovery, manual intervention, partial recovery
- **Chaos engineering**: Random failures, resource exhaustion, network partitions

## Integration with TDD Approach

These integration tests are designed to work with the Test-Driven Development approach:

1. **Tests written first** - All integration scenarios defined before implementation
2. **Red-Green-Refactor** - Tests initially fail, then pass after implementation
3. **Comprehensive coverage** - All decorator interactions and edge cases covered
4. **Performance validation** - Tests verify roadmap performance targets
5. **Real-world scenarios** - Tests simulate actual production use cases

## Expected Test Results

When the decorator implementations are complete, these integration tests should demonstrate:

1. **100% test pass rate** - All integration scenarios working correctly
2. **Performance targets met** - All latency and throughput requirements satisfied
3. **Zero data loss** - Complete fault tolerance under all failure scenarios
4. **Linear scalability** - Performance scales linearly with resources
5. **Production readiness** - System ready for real-world deployment

## Next Steps

After running these integration tests:

1. **Analyze results** - Identify any performance bottlenecks or failure modes
2. **Optimize implementation** - Tune decorator implementations based on test results  
3. **Extend scenarios** - Add additional integration tests for specific use cases
4. **Production validation** - Deploy to staging environment for final validation
5. **Documentation** - Update examples and documentation based on test insights

These integration tests provide comprehensive validation of the GoFasta fault tolerance framework, ensuring that all decorators work together seamlessly to provide production-ready resilience capabilities.