# Phase 1.1 Examples

This directory contains comprehensive examples demonstrating all Phase 1.1 components of the GoFasta transpiler.

## Overview

Phase 1.1 implements high-performance Go language processing with caching and parallel processing:

- **Phase 1.1a**: Parallel parser (already implemented)
- **Phase 1.1b**: AST caching system
- **Phase 1.1c**: Token memory pooling  
- **Phase 1.1d**: Incremental type checking
- **Phase 1.1e**: Batched formatting
- **Phase 1.1f**: Import caching

## Examples

### 1. AST Cache Example (`ast_cache_example.go`)

Demonstrates the AST caching system from Phase 1.1b:

```bash
go run ast_cache_example.go
```

**Features shown:**
- AST cache configuration and initialization
- Caching parsed ASTs with modification time validation
- Cache hit/miss scenarios
- Performance metrics and statistics
- Cache cleanup and management
- Performance comparison with 1000+ lookups

**Expected output:**
- Cache hit confirmation
- Performance statistics
- Hit ratio improvements
- Memory usage tracking

### 2. Token Pool Example (`token_pool_example.go`)

Demonstrates the token pooling system from Phase 1.1c:

```bash
go run token_pool_example.go
```

**Features shown:**
- Token pool configuration and warm-up
- Concurrent usage with 8 workers and 100 operations each
- Pool resizing and draining
- Performance metrics and reuse ratios
- Memory efficiency comparison

**Expected output:**
- Concurrent operation statistics
- Pool reuse efficiency metrics
- Performance improvements from pooling

### 3. Type Checker Example (`type_checker_example.go`)

Demonstrates the incremental type checking from Phase 1.1d:

```bash
go run type_checker_example.go
```

**Features shown:**
- Incremental type checker configuration
- Single package type checking with caching
- Parallel type checking across multiple packages
- Cache hit performance improvements
- Cache invalidation mechanisms

**Expected output:**
- Type checking performance comparisons
- Cache efficiency statistics
- Parallel processing results

### 4. Formatter Example (`formatter_example.go`)

Demonstrates the batched formatting system from Phase 1.1e:

```bash
go run formatter_example.go
```

**Features shown:**
- Batch formatter configuration
- Single and batch file formatting
- Formatting options (tabs vs spaces, import sorting)
- Performance metrics and throughput
- Memory usage estimation

**Expected output:**
- Formatted code output
- Formatting performance statistics
- Before/after comparisons

### 5. Import Cache Example (`import_cache_example.go`)

Demonstrates the import caching system from Phase 1.1f:

```bash
go run import_cache_example.go
```

**Features shown:**
- Import cache configuration
- Standard library package caching
- Cache hit performance improvements
- Preloading and warm-up strategies
- Fallback import mechanisms

**Expected output:**
- Import performance comparisons
- Cache efficiency metrics
- Memory usage statistics

### 6. Complete Pipeline Example (`complete_pipeline_example.go`)

Demonstrates all Phase 1.1 components working together:

```bash
go run complete_pipeline_example.go
```

**Features shown:**
- Complete GoFasta processing pipeline
- All components working in harmony
- Performance optimization suggestions
- Comprehensive statistics and metrics
- Real-world project processing simulation

**Expected output:**
- End-to-end processing results
- Component performance breakdown
- Optimization recommendations
- Overall system efficiency metrics

## Performance Expectations

Based on the test results, Phase 1.1 achieves:

- **Parser**: 20,000+ files/second
- **Formatter**: 20,000+ files/second  
- **AST Cache**: 100% hit ratio for repeated access
- **Type Checker**: 50%+ cache hit ratio
- **Import Cache**: 50%+ hit ratio for standard packages
- **Token Pool**: High reuse ratios reducing memory allocation

## Configuration Options

Each component provides extensive configuration options:

### AST Cache Configuration
```go
&core.ASTCacheConfig{
    MaxEntries:    1000,     // Maximum cached ASTs
    TTL:           time.Hour, // Time-to-live for entries
    MaxMemoryMB:   512,      // Memory limit in MB
    EnableMetrics: true,     // Enable performance tracking
}
```

### Token Pool Configuration
```go
&core.TokenPoolConfig{
    InitialSize:   10,  // Initial pool size
    MaxSize:       50,  // Maximum pool size
    EnableMetrics: true, // Enable metrics tracking
}
```

### Type Checker Configuration
```go
&core.TypeCheckerConfig{
    EnableCaching:    true,               // Enable result caching
    CacheTTL:         30 * time.Minute,   // Cache TTL
    MaxCacheEntries:  500,                // Max cached results
    ParallelChecking: true,               // Enable parallel processing
    MaxWorkers:       4,                  // Worker count
    EnableMetrics:    true,               // Enable metrics
}
```

### Formatter Configuration
```go
&core.BatchFormatterConfig{
    BatchSize:     10,    // Files per batch
    MaxWorkers:    4,     // Parallel workers
    EnableMetrics: true,  // Enable metrics
    FormatOptions: &core.FormatOptions{
        TabWidth:    8,     // Tab width
        UseSpaces:   false, // Use tabs vs spaces
        SortImports: true,  // Sort import statements
    },
}
```

### Import Cache Configuration
```go
&core.ImportCacheConfig{
    MaxEntries:    1000,     // Maximum cached imports
    TTL:           time.Hour, // Cache TTL
    EnableMetrics: true,     // Enable metrics
    MaxMemoryMB:   256,      // Memory limit
}
```

## Integration with GoFasta Transpiler

These components integrate seamlessly with the main GoFasta transpiler pipeline, providing:

1. **High-throughput parsing** with parallel file processing
2. **Memory efficiency** through AST caching and token pooling
3. **Fast type checking** with incremental caching
4. **Optimized formatting** with batched parallel processing
5. **Efficient imports** with smart caching strategies

## Next Steps

- Phase 1.2: Advanced AST transformations
- Phase 1.3: Code generation optimizations
- Phase 2.1: GOFA language parsing
- Phase 2.2: GOFA to Go transpilation

All examples include comprehensive error handling, performance monitoring, and best practices for production use.