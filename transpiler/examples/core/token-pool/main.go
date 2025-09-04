// Example demonstrating Phase 1.1c: Token pooling for memory efficiency
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	// Initialize token pool with custom configuration
	config := &core.TokenPoolConfig{
		InitialSize:   5,
		MaxSize:       20,
		EnableMetrics: true,
	}

	pool := core.NewTokenPool(config)

	fmt.Println("=== Token Pool Example ===")
	fmt.Printf("Initial pool configuration:\n")
	fmt.Printf("  Initial Size: %d\n", config.InitialSize)
	fmt.Printf("  Max Size: %d\n", config.MaxSize)

	// Warm up the pool
	fmt.Printf("\nWarming up pool to 10 entries...\n")
	pool.WarmUp(10)

	stats := pool.GetStatistics()
	fmt.Printf("Pool after warmup:\n")
	fmt.Printf("  Pool Size: %v\n", stats["pool_size"])
	fmt.Printf("  Created: %v\n", stats["created"])

	// Demonstrate concurrent usage
	fmt.Printf("\nConcurrent usage demonstration:\n")

	numWorkers := 8
	operationsPerWorker := 100
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Get a FileSet from the pool
				fset := pool.Get()

				// Use the FileSet (simulate some work)
				fset.AddFile("worker_file.go", -1, 100)

				// Return it to the pool
				pool.Put(fset)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("\nPerformance Results:\n")
	fmt.Printf("  Workers: %d\n", numWorkers)
	fmt.Printf("  Operations per worker: %d\n", operationsPerWorker)
	fmt.Printf("  Total operations: %d\n", numWorkers*operationsPerWorker)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Operations per second: %.0f\n", float64(numWorkers*operationsPerWorker)/duration.Seconds())

	// Show final statistics
	finalStats := pool.GetStatistics()
	fmt.Printf("\nFinal Pool Statistics:\n")
	fmt.Printf("  Pool Size: %v\n", finalStats["pool_size"])
	fmt.Printf("  Created: %v\n", finalStats["created"])
	fmt.Printf("  Reused: %v\n", finalStats["reused"])
	fmt.Printf("  Discarded: %v\n", finalStats["discarded"])

	// Calculate reuse ratio
	if finalStats["created"].(int64) > 0 {
		reuseRatio := float64(finalStats["reused"].(int64)) / float64(finalStats["created"].(int64)) * 100
		fmt.Printf("  Reuse Ratio: %.1f%%\n", reuseRatio)
	}

	// Demonstrate pool resizing
	fmt.Printf("\nDemonstrating pool resizing:\n")
	fmt.Printf("Resizing pool from %d to 15...\n", config.MaxSize)
	pool.Resize(15)

	resizeStats := pool.GetStatistics()
	fmt.Printf("Pool after resize:\n")
	fmt.Printf("  Max Size: %d\n", 15)
	fmt.Printf("  Pool Size: %v\n", resizeStats["pool_size"])

	// Demonstrate draining
	fmt.Printf("\nDraining pool...\n")
	drained := pool.Drain()
	fmt.Printf("Drained %d FileSets from pool\n", drained)

	drainStats := pool.GetStatistics()
	fmt.Printf("Pool after drain:\n")
	fmt.Printf("  Pool Size: %v\n", drainStats["pool_size"])

	fmt.Printf("\n✓ Token pool example completed successfully!\n")
}
