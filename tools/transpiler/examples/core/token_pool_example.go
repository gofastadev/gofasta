// Example demonstrating Phase 1.1c: Token pooling for memory efficiency
package main

import (
	"fmt"
	"go/token"
	"sync"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	// Initialize token pool with custom configuration
	config := &core.TokenPoolConfig{
		InitialSize:   5,
		MaxSize:       20,
		EnableMetrics: true,
	}
	
	tokenPool := core.NewTokenPool(config)
	
	fmt.Println("=== Token Pool Example ===")
	fmt.Printf("Initial pool configuration:\n")
	fmt.Printf("  Initial Size: %d\n", config.InitialSize)
	fmt.Printf("  Max Size: %d\n", config.MaxSize)
	
	// Warm up the pool
	fmt.Printf("\nWarming up pool to 10 entries...\n")
	tokenPool.WarmUp(10)
	
	initialStats := tokenPool.GetStatistics()
	fmt.Printf("Pool after warmup:\n")
	fmt.Printf("  Pool Size: %v\n", initialStats["pool_size"])
	fmt.Printf("  Created: %v\n", initialStats["created"])
	
	// Demonstrate pool usage in a concurrent scenario
	fmt.Printf("\nConcurrent usage demonstration:\n")
	
	var wg sync.WaitGroup
	numWorkers := 8
	operationsPerWorker := 100
	
	start := time.Now()
	
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerWorker; j++ {
				// Get a FileSet from the pool
				fset := tokenPool.Get()
				
				// Simulate work with the FileSet
				pos := fset.AddFile(fmt.Sprintf("worker%d_file%d.go", workerID, j), -1, 100)
				_ = pos // Use the position
				
				// Return the FileSet to the pool
				tokenPool.Put(fset)
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	// Show performance results
	finalStats := tokenPool.GetStatistics()
	totalOps := numWorkers * operationsPerWorker
	
	fmt.Printf("\nPerformance Results:\n")
	fmt.Printf("  Workers: %d\n", numWorkers)
	fmt.Printf("  Operations per worker: %d\n", operationsPerWorker)
	fmt.Printf("  Total operations: %d\n", totalOps)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Operations per second: %.0f\n", float64(totalOps)/duration.Seconds())
	
	fmt.Printf("\nFinal Pool Statistics:\n")
	fmt.Printf("  Pool Size: %v\n", finalStats["pool_size"])
	fmt.Printf("  Created: %v\n", finalStats["created"])
	fmt.Printf("  Reused: %v\n", finalStats["reused"])
	fmt.Printf("  Discarded: %v\n", finalStats["discarded"])
	
	reused := finalStats["reused"].(int64)
	created := finalStats["created"].(int64)
	if created > 0 {
		reuseRatio := float64(reused) / float64(created) * 100
		fmt.Printf("  Reuse Ratio: %.1f%%\n", reuseRatio)
	}
	
	// Demonstrate pool resizing
	fmt.Printf("\nDemonstrating pool resizing:\n")
	fmt.Printf("Resizing pool from %d to 15...\n", config.MaxSize)
	tokenPool.Resize(15)
	
	resizeStats := tokenPool.GetStatistics()
	fmt.Printf("Pool after resize:\n")
	fmt.Printf("  Max Size: %v\n", resizeStats["max_size"])
	fmt.Printf("  Pool Size: %v\n", resizeStats["pool_size"])
	
	// Demonstrate pool draining
	fmt.Printf("\nDraining pool...\n")
	drained := tokenPool.Drain()
	fmt.Printf("Drained %d FileSets from pool\n", drained)
	
	drainStats := tokenPool.GetStatistics()
	fmt.Printf("Pool after drain:\n")
	fmt.Printf("  Pool Size: %v\n", drainStats["pool_size"])
	
	fmt.Printf("\n✓ Token pool example completed successfully!\n")
}

// demonstratePoolEfficiency shows the difference between pooled and non-pooled allocations
func demonstratePoolEfficiency() {
	fmt.Printf("\n=== Pool Efficiency Comparison ===\n")
	
	iterations := 10000
	
	// Test without pool (allocating new FileSets each time)
	fmt.Printf("Testing without pool...\n")
	start := time.Now()
	for i := 0; i < iterations; i++ {
		fset := token.NewFileSet()
		fset.AddFile("test.go", -1, 100)
	}
	withoutPool := time.Since(start)
	
	// Test with pool
	fmt.Printf("Testing with pool...\n")
	tokenPool := core.NewTokenPool(&core.TokenPoolConfig{
		InitialSize:   100,
		MaxSize:       200,
		EnableMetrics: true,
	})
	tokenPool.WarmUp(100)
	
	start = time.Now()
	for i := 0; i < iterations; i++ {
		fset := tokenPool.Get()
		fset.AddFile("test.go", -1, 100)
		tokenPool.Put(fset)
	}
	withPool := time.Since(start)
	
	fmt.Printf("\nPerformance Comparison:\n")
	fmt.Printf("  Without pool: %v\n", withoutPool)
	fmt.Printf("  With pool: %v\n", withPool)
	
	if withoutPool > withPool {
		improvement := float64(withoutPool-withPool) / float64(withoutPool) * 100
		fmt.Printf("  Improvement: %.1f%% faster with pool\n", improvement)
	}
	
	poolStats := tokenPool.GetStatistics()
	fmt.Printf("  Pool reuse ratio: %.1f%%\n", 
		float64(poolStats["reused"].(int64))/float64(poolStats["created"].(int64))*100)
}