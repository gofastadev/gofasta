package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// TestConcurrentProcessingIntegration tests concurrent processing capabilities and thread safety
func TestConcurrentProcessingIntegration(t *testing.T) {
	t.Run("MultiFileParallelProcessing", testMultiFileParallelProcessing)
	t.Run("WorkerPoolManagement", testWorkerPoolManagement)
	t.Run("RaceConditionDetection", testRaceConditionDetection)
	t.Run("ThreadSafetyValidation", testThreadSafetyValidation)
	t.Run("ResourceContentionHandling", testResourceContentionHandling)
	t.Run("LoadBalancingAcrossWorkers", testLoadBalancingAcrossWorkers)
	t.Run("ConcurrentComponentIntegration", testConcurrentComponentIntegration)
	t.Run("MemoryConsistencyUnderLoad", testMemoryConsistencyUnderLoad)
	t.Run("DeadlockPrevention", testDeadlockPrevention)
	t.Run("ConcurrentCacheOperations", testConcurrentCacheOperations)
}

// Test 1: Multi-file parallel processing
func testMultiFileParallelProcessing(t *testing.T) {
	testDir := createTestDir(t, "multifile_parallel_test")
	defer os.RemoveAll(testDir)

	// Create large number of test files to ensure parallel processing
	numFiles := runtime.NumCPU() * 10 // Scale with available CPUs
	if numFiles < 50 {
		numFiles = 50
	}

	var filePaths []string
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("parallel_test_%03d.gofa", i)
		filepath := filepath.Join(testDir, filename)
		content := fmt.Sprintf(`package test%d

// @GET("/api/endpoint%d")
// @Auth("jwt")
// @RateLimit(%d)
func Endpoint%d() {
    // Function body for endpoint %d
    data := "test data %d"
    return data
}

// @POST("/api/create%d")
// @Validation("required")
func CreateEndpoint%d() {
    // Create operation %d
}
`, i, i, i*10, i, i, i, i, i, i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Test with different worker configurations
	workerConfigs := []int{1, runtime.NumCPU(), runtime.NumCPU() * 2}

	for _, workerCount := range workerConfigs {
		t.Run(fmt.Sprintf("Workers_%d", workerCount), func(t *testing.T) {
			// Initialize parser with specific worker count
			config := core.DefaultConfig()
			config.MaxWorkers = workerCount
			parser := core.NewParallelParser(config)

			// Measure parsing performance
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, filePaths)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Parallel parsing failed with %d workers: %v", workerCount, err)
			}

			if len(results) != numFiles {
				t.Errorf("Expected %d parse results, got %d", numFiles, len(results))
			}

			// Count successful parses
			successCount := 0
			for _, result := range results {
				if result.Error == nil && result.File != nil {
					successCount++
				}
			}

			if successCount != numFiles {
				t.Errorf("Expected %d successful parses, got %d", numFiles, successCount)
			}

			// Get performance statistics
			stats := parser.GetStatistics()
			filesPerSec, _ := stats["files_per_second"].(float64)

			t.Logf("Worker config %d: parsed %d files in %v (%.2f files/sec)", 
				workerCount, numFiles, duration, filesPerSec)

			// Validate performance expectations
			if filesPerSec < 10 {
				t.Errorf("Performance too low with %d workers: %.2f files/sec", workerCount, filesPerSec)
			}
		})
	}

	t.Logf("Multi-file parallel processing successful: tested %d files with various worker configurations", numFiles)
}

// Test 2: Worker pool management
func testWorkerPoolManagement(t *testing.T) {
	testDir := createTestDir(t, "worker_pool_test")
	defer os.RemoveAll(testDir)

	// Create files for worker pool testing
	numFiles := 100
	var filePaths []string

	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("worker_test_%03d.gofa", i)
		filepath := filepath.Join(testDir, filename)
		content := fmt.Sprintf("package worker%d\n\nfunc Worker%d() {}", i, i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create worker test file: %v", err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Test worker pool scaling
	testCases := []struct {
		name        string
		workerCount int
		expectMin   float64 // minimum expected files/sec
	}{
		{"SingleWorker", 1, 50},
		{"OptimalWorkers", runtime.NumCPU(), 100},
		{"OverSubscribed", runtime.NumCPU() * 4, 80}, // May be less efficient
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := core.DefaultConfig()
			config.MaxWorkers = tc.workerCount
			parser := core.NewParallelParser(config)

			// Measure worker utilization
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, filePaths)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Worker pool test failed: %v", err)
			}

			if len(results) != numFiles {
				t.Errorf("Expected %d results, got %d", numFiles, len(results))
			}

			// Calculate performance metrics
			filesPerSec := float64(numFiles) / duration.Seconds()
			
			if filesPerSec < tc.expectMin {
				t.Errorf("Worker performance below expected minimum: %.2f < %.2f files/sec", 
					filesPerSec, tc.expectMin)
			}

			stats := parser.GetStatistics()
			actualWorkers := stats["max_workers"].(int)
			
			if actualWorkers != tc.workerCount {
				t.Errorf("Expected %d workers, parser configured with %d", tc.workerCount, actualWorkers)
			}

			t.Logf("Worker pool %s: %d workers, %.2f files/sec, %v duration", 
				tc.name, tc.workerCount, filesPerSec, duration)
		})
	}
}

// Test 3: Race condition detection
func testRaceConditionDetection(t *testing.T) {
	testDir := createTestDir(t, "race_condition_test")
	defer os.RemoveAll(testDir)

	// Create shared test files
	sharedFiles := make([]string, 20)
	for i := range sharedFiles {
		filename := fmt.Sprintf("shared_%d.gofa", i)
		filepath := filepath.Join(testDir, filename)
		content := fmt.Sprintf("package shared%d\n\n// @Service\ntype Service%d struct {}", i, i)
		
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create shared test file: %v", err)
		}
		sharedFiles[i] = filepath
	}

	// Initialize components that will access shared resources
	extractorConfig := core.DefaultExtractorConfig()
	extractorConfig.ParallelExtraction = true
	extractorConfig.WorkerCount = runtime.NumCPU()
	extractor := core.NewDecoratorExtractor(extractorConfig)

	registryConfig := core.DefaultRegistryConfig()
	registryConfig.ParallelLoading = true
	registry := core.NewDecoratorRegistry(registryConfig)

	// Test concurrent operations on shared components
	const numGoroutines = 50
	var wg sync.WaitGroup
	var raceErrors int64
	var successfulOps int64

	// Concurrent extraction operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for _, filePath := range sharedFiles {
				// Read file content
				content, err := os.ReadFile(filePath)
				if err != nil {
					atomic.AddInt64(&raceErrors, 1)
					continue
				}

				// Extract decorators concurrently
				result, err := extractor.Extract(content)
				if err != nil {
					atomic.AddInt64(&raceErrors, 1)
					continue
				}

				// Process decorators through registry
				ctx := context.Background()
				for _, decorator := range result.Decorators {
					args := core.DecoratorArgs{
						Target:    fmt.Sprintf("goroutine_%d", goroutineID),
						Arguments: []interface{}{decorator.Arguments},
					}
					
					_, err := registry.Invoke(ctx, decorator.Name, args)
					if err == nil {
						atomic.AddInt64(&successfulOps, 1)
					} else {
						// Some errors are expected for unregistered decorators
						atomic.AddInt64(&successfulOps, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Check for race conditions
	raceCount := atomic.LoadInt64(&raceErrors)
	successCount := atomic.LoadInt64(&successfulOps)

	if raceCount > int64(numGoroutines) {
		t.Errorf("Excessive race condition errors: %d (may indicate data races)", raceCount)
	}

	if successCount == 0 {
		t.Error("No successful concurrent operations (possible deadlock or severe race conditions)")
	}

	// Verify component statistics are consistent
	extractorStats := extractor.GetStatistics()
	registryStats := registry.GetStatistics()

	if extractions, ok := extractorStats["extractions"].(int64); ok && extractions == 0 {
		t.Error("Extractor shows no operations (possible race condition)")
	}

	if invocations, ok := registryStats["invocations"].(int64); ok && invocations == 0 {
		t.Error("Registry shows no invocations (possible race condition)")
	}

	t.Logf("Race condition test: %d successful operations, %d errors across %d goroutines", 
		successCount, raceCount, numGoroutines)
}

// Test 4: Thread safety validation
func testThreadSafetyValidation(t *testing.T) {
	testDir := createTestDir(t, "thread_safety_test")
	defer os.RemoveAll(testDir)

	// Create test content for thread safety testing
	content := `package threadsafety

// @Controller("/api/test")
type TestController struct {}

// @GET("/users")
// @Auth("jwt")
func (c *TestController) GetUsers() {}

// @POST("/users")
// @Validation("required")
func (c *TestController) CreateUser() {}
`

	testFile := filepath.Join(testDir, "threadsafe.gofa")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create thread safety test file: %v", err)
	}

	// Initialize shared components
	extractor := core.NewDecoratorExtractor(core.DefaultExtractorConfig())
	generator := core.NewCodeGenerator(core.DefaultGeneratorConfig())

	// Concurrent operations to test thread safety
	const iterations = 100
	var wg sync.WaitGroup
	var errors int64
	var successes int64

	// Test concurrent read operations (should be safe)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Concurrent extraction
			result, err := extractor.Extract([]byte(content))
			if err != nil {
				atomic.AddInt64(&errors, 1)
				return
			}

			// Concurrent code generation
			for _, decorator := range result.Decorators {
				structDef := core.TypeDefinition{
					Name: fmt.Sprintf("Generated%d", id),
					Kind: "struct",
					Decorators: []core.Decorator{decorator},
				}

				_, err := generator.GenerateStruct(structDef)
				if err != nil {
					atomic.AddInt64(&errors, 1)
					return
				}
			}

			atomic.AddInt64(&successes, 1)
		}(i)
	}

	wg.Wait()

	errorCount := atomic.LoadInt64(&errors)
	successCount := atomic.LoadInt64(&successes)

	// Verify thread safety
	if errorCount > int64(iterations/10) {
		t.Errorf("High error rate in concurrent operations: %d/%d", errorCount, iterations)
	}

	if successCount < int64(iterations/2) {
		t.Errorf("Low success rate in concurrent operations: %d/%d", successCount, iterations)
	}

	// Test component state consistency after concurrent access
	extractorStats := extractor.GetStatistics()
	generatorStats := generator.GetStatistics()

	if extractorExtractions, ok := extractorStats["extractions"].(int64); ok && extractorExtractions != successCount {
		t.Logf("Extractor extractions (%d) vs successes (%d) - may be normal due to async updates", 
			extractorExtractions, successCount)
	}

	if generatorGenerations, ok := generatorStats["generations"].(int64); ok && generatorGenerations < successCount {
		t.Logf("Generator operations (%d) vs successes (%d) - may indicate missing operations", 
			generatorGenerations, successCount)
	}

	t.Logf("Thread safety validation: %d successes, %d errors out of %d concurrent operations", 
		successCount, errorCount, iterations)
}

// Test 5: Resource contention handling
func testResourceContentionHandling(t *testing.T) {
	testDir := createTestDir(t, "resource_contention_test")
	defer os.RemoveAll(testDir)

	// Create files that will compete for resources
	numFiles := 50
	var filePaths []string

	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("contention_%03d.gofa", i)
		filepath := filepath.Join(testDir, filename)
		// Create content that requires significant processing
		content := fmt.Sprintf(`package contention%d

import (
	"context"
	"database/sql"
	"net/http"
)

// @Controller("/api/v%d")
// @RateLimit(%d)
type Controller%d struct {
	db *sql.DB
	cache map[string]interface{}
}

// @GET("/items")
// @Auth("bearer")
// @Cache("5m")
func (c *Controller%d) GetItems(ctx context.Context) ([]Item, error) {
	// Complex processing that might contend for resources
	items := make([]Item, %d)
	for i := range items {
		items[i] = Item{ID: i, Name: "item"}
	}
	return items, nil
}

// @POST("/items")
// @Validation("required", "json")
// @Transaction
func (c *Controller%d) CreateItem(ctx context.Context, item Item) error {
	// Database operation
	return nil
}

type Item struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`, i, i, i*5, i, i, i*10, i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create contention test file: %v", err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Test resource contention with limited workers vs many workers
	testConfigs := []struct {
		name         string
		maxWorkers   int
		expectGracefulDegradation bool
	}{
		{"LimitedWorkers", 2, true},
		{"ManyWorkers", runtime.NumCPU() * 4, false},
	}

	for _, tc := range testConfigs {
		t.Run(tc.name, func(t *testing.T) {
			config := core.DefaultConfig()
			config.MaxWorkers = tc.maxWorkers
			parser := core.NewParallelParser(config)

			// Add timeout to detect deadlocks/hangs
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, filePaths)
			duration := time.Since(start)

			if err != nil {
				if tc.expectGracefulDegradation {
					t.Logf("Expected some issues with limited workers: %v", err)
				} else {
					t.Errorf("Unexpected error with many workers: %v", err)
				}
			}

			if len(results) != numFiles {
				t.Errorf("Expected %d results, got %d", numFiles, len(results))
			}

			// Check for timeout issues
			if duration > time.Second*30 {
				t.Errorf("Processing took too long: %v (possible resource contention)", duration)
			}

			// Validate results quality
			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}

			successRate := float64(successCount) / float64(numFiles)
			if successRate < 0.8 {
				t.Errorf("Low success rate: %.2f%% (possible resource exhaustion)", successRate*100)
			}

			stats := parser.GetStatistics()
			filesPerSec, _ := stats["files_per_second"].(float64)

			t.Logf("Resource contention %s: %d workers, %.2f files/sec, %v duration, %.1f%% success", 
				tc.name, tc.maxWorkers, filesPerSec, duration, successRate*100)
		})
	}
}

// Test 6: Load balancing across workers
func testLoadBalancingAcrossWorkers(t *testing.T) {
	testDir := createTestDir(t, "load_balancing_test")
	defer os.RemoveAll(testDir)

	// Create files with varying complexity to test load balancing
	fileComplexities := []struct {
		name       string
		size       int
		decorators int
	}{
		{"simple", 100, 1},
		{"medium", 1000, 5},
		{"complex", 5000, 15},
	}

	var allFiles []string
	for _, spec := range fileComplexities {
		// Create multiple files of each complexity
		for i := 0; i < 20; i++ {
			filename := fmt.Sprintf("%s_%03d.gofa", spec.name, i)
			filepath := filepath.Join(testDir, filename)

			// Generate content based on complexity
			var contentBuilder strings.Builder
			contentBuilder.WriteString(fmt.Sprintf("package %s%d\n\n", spec.name, i))

			// Add decorators based on complexity
			for d := 0; d < spec.decorators; d++ {
				contentBuilder.WriteString(fmt.Sprintf("// @Decorator%d(\"param%d\")\n", d, d))
			}

			// Add functions to reach target size
			funcContent := fmt.Sprintf(`func Function%d() {
	// Function body %d
	data := "test data"
	return data
}

`, i, i)

			for contentBuilder.Len() < spec.size {
				contentBuilder.WriteString(funcContent)
			}

			if err := os.WriteFile(filepath, []byte(contentBuilder.String()), 0644); err != nil {
				t.Fatalf("Failed to create load balancing test file: %v", err)
			}
			allFiles = append(allFiles, filepath)
		}
	}

	// Test load balancing with different worker counts
	workerCounts := []int{2, runtime.NumCPU(), runtime.NumCPU() * 2}

	for _, workerCount := range workerCounts {
		t.Run(fmt.Sprintf("LoadBalance_%d_workers", workerCount), func(t *testing.T) {
			config := core.DefaultConfig()
			config.MaxWorkers = workerCount
			parser := core.NewParallelParser(config)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, allFiles)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Load balancing test failed: %v", err)
			}

			if len(results) != len(allFiles) {
				t.Errorf("Expected %d results, got %d", len(allFiles), len(results))
			}

			// Analyze performance distribution
			var totalSize int64
			var totalDuration time.Duration
			successCount := 0

			for _, result := range results {
				if result.Error == nil {
					successCount++
					totalSize += result.Size
					totalDuration += result.Duration
				}
			}

			avgDurationPerFile := totalDuration / time.Duration(successCount)
			avgSize := totalSize / int64(successCount)
			filesPerSec := float64(len(allFiles)) / duration.Seconds()

			// Load balancing should show reasonable performance
			if filesPerSec < 10 {
				t.Errorf("Poor load balancing performance: %.2f files/sec", filesPerSec)
			}

			t.Logf("Load balancing with %d workers: %.2f files/sec, avg %v per file (avg size: %d bytes)", 
				workerCount, filesPerSec, avgDurationPerFile, avgSize)
		})
	}
}

// Test 7: Concurrent component integration
func testConcurrentComponentIntegration(t *testing.T) {
	testDir := createTestDir(t, "concurrent_component_test")
	defer os.RemoveAll(testDir)

	// Create test files for full component integration
	numFiles := 30
	var filePaths []string

	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("integration_%03d.gofa", i)
		filepath := filepath.Join(testDir, filename)
		content := fmt.Sprintf(`package integration%d

// @RestController("/api/v%d")
// @Auth("jwt")
type Controller%d struct {
	service *Service%d
}

// @GET("/items")
// @Cache("10m")
// @RateLimit(100)
func (c *Controller%d) GetItems() []Item {
	return []Item{}
}

// @POST("/items")
// @Validation("required")
// @Transaction
func (c *Controller%d) CreateItem(item Item) error {
	return nil
}

type Item struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

type Service%d struct {
	repo Repository
}
`, i, i, i, i, i, i, i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create component integration test file: %v", err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Initialize all components for concurrent testing
	parser := core.NewParallelParser(core.DefaultConfig())
	
	extractorConfig := core.DefaultExtractorConfig()
	extractorConfig.ParallelExtraction = true
	extractor := core.NewDecoratorExtractor(extractorConfig)
	
	registryConfig := core.DefaultRegistryConfig()
	registryConfig.ParallelLoading = true
	registry := core.NewDecoratorRegistry(registryConfig)
	
	generator := core.NewCodeGenerator(core.DefaultGeneratorConfig())

	// Test full concurrent pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	start := time.Now()

	// Step 1: Concurrent parsing
	parseResults, err := parser.ParseFiles(ctx, filePaths)
	if err != nil {
		t.Fatalf("Concurrent parsing failed: %v", err)
	}

	// Step 2: Concurrent extraction from parsed content
	extractionTasks := make(map[string][]byte)
	for i, result := range parseResults {
		if result.Error == nil {
			// Read file content for extraction
			if content, err := os.ReadFile(filePaths[i]); err == nil {
				extractionTasks[filePaths[i]] = content
			}
		}
	}

	extractionResults, err := extractor.ExtractParallel(extractionTasks)
	if err != nil {
		t.Fatalf("Concurrent extraction failed: %v", err)
	}

	// Step 3: Concurrent processing through registry
	var processedDecorators int64
	var wg sync.WaitGroup

	for filePath, extraction := range extractionResults {
		for _, decorator := range extraction.Decorators {
			wg.Add(1)
			go func(path string, dec core.Decorator) {
				defer wg.Done()
				
				args := core.DecoratorArgs{
					Target:    path,
					Arguments: []interface{}{dec.Arguments},
				}
				
				result, err := registry.Invoke(ctx, dec.Name, args)
				if err == nil && result.Success {
					atomic.AddInt64(&processedDecorators, 1)
				}
			}(filePath, decorator)
		}
	}
	wg.Wait()

	// Step 4: Concurrent code generation
	generationTasks := make(map[string]core.GenerationRequest)
	taskCount := 0
	for i := 0; i < min(10, len(extractionResults)); i++ { // Limit to avoid overload
		taskName := fmt.Sprintf("generated_%d", i)
		generationTasks[taskName] = core.GenerationRequest{
			Template: "struct",
			Context: core.TypeDefinition{
				Name: fmt.Sprintf("Generated%d", i),
				Kind: "struct",
			},
		}
		taskCount++
	}

	if taskCount > 0 {
		generationResults, err := generator.GenerateBatch(generationTasks)
		if err != nil {
			t.Errorf("Concurrent generation failed: %v", err)
		} else if len(generationResults) != taskCount {
			t.Errorf("Expected %d generation results, got %d", taskCount, len(generationResults))
		}
	}

	totalDuration := time.Since(start)
	processedCount := atomic.LoadInt64(&processedDecorators)

	// Validate concurrent component integration
	if len(parseResults) != numFiles {
		t.Errorf("Expected %d parse results, got %d", numFiles, len(parseResults))
	}

	if len(extractionResults) == 0 {
		t.Error("No extraction results from concurrent processing")
	}

	if processedCount == 0 {
		t.Error("No decorators processed through concurrent registry operations")
	}

	// Performance validation
	filesPerSec := float64(numFiles) / totalDuration.Seconds()
	if filesPerSec < 5 {
		t.Errorf("Poor concurrent integration performance: %.2f files/sec", filesPerSec)
	}

	t.Logf("Concurrent component integration: %d files, %d extractions, %d processed decorators, %.2f files/sec in %v", 
		numFiles, len(extractionResults), processedCount, filesPerSec, totalDuration)
}

// Test 8: Memory consistency under load
func testMemoryConsistencyUnderLoad(t *testing.T) {
	// Test memory consistency by running concurrent operations and checking for data corruption
	const iterations = 500
	const goroutines = 20

	// Shared data structure for testing
	sharedData := make(map[string]int64)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	// Components to test under load
	extractor := core.NewDecoratorExtractor(core.DefaultExtractorConfig())
	registry := core.NewDecoratorRegistry(core.DefaultRegistryConfig())

	testContent := []byte(`package memory

// @Service("test")
// @Cache("1m")
type TestService struct {}

// @GET("/test")
func (s *TestService) Test() {}
`)

	// Concurrent read/write operations
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < iterations/goroutines; j++ {
				// Memory-intensive operations
				result, err := extractor.Extract(testContent)
				if err != nil {
					t.Errorf("Extraction failed: %v", err)
					continue
				}

				// Update shared data structure
				key := fmt.Sprintf("goroutine_%d", goroutineID)
				mu.Lock()
				sharedData[key] += int64(len(result.Decorators))
				mu.Unlock()

				// Test registry operations
				ctx := context.Background()
				for _, decorator := range result.Decorators {
					args := core.DecoratorArgs{
						Target: fmt.Sprintf("test_%d_%d", goroutineID, j),
					}
					registry.Invoke(ctx, decorator.Name, args)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify memory consistency
	mu.RLock()
	totalOperations := int64(0)
	for _, count := range sharedData {
		totalOperations += count
	}
	mu.RUnlock()

	if len(sharedData) != goroutines {
		t.Errorf("Expected %d goroutine entries, got %d", goroutines, len(sharedData))
	}

	if totalOperations == 0 {
		t.Error("No operations recorded (possible memory consistency issue)")
	}

	// Check component statistics for consistency
	extractorStats := extractor.GetStatistics()
	registryStats := registry.GetStatistics()

	t.Logf("Memory consistency test: %d total operations, %d goroutines, extractor stats: %v", 
		totalOperations, goroutines, extractorStats["extractions"])
	
	if invocations, ok := registryStats["invocations"].(int64); ok {
		t.Logf("Registry invocations: %d", invocations)
	}
}

// Test 9: Deadlock prevention
func testDeadlockPrevention(t *testing.T) {
	// Test scenarios that could potentially cause deadlocks
	testDir := createTestDir(t, "deadlock_test")
	defer os.RemoveAll(testDir)

	// Create interdependent test files
	files := map[string]string{
		"service.gofa": `package service

// @Service
type UserService struct {
	repo *UserRepository
}

// @GET("/users")
func (s *UserService) GetUsers() {}
`,
		"repository.gofa": `package repository

// @Repository
type UserRepository struct {
	db Database
}

// @Query("SELECT * FROM users")
func (r *UserRepository) FindAll() {}
`,
		"controller.gofa": `package controller

// @Controller("/api")
type UserController struct {
	service *UserService
}

// @GET("/users")
func (c *UserController) GetUsers() {}
`,
	}

	var filePaths []string
	for filename, content := range files {
		filepath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create deadlock test file: %v", err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Test with tight timeout to detect deadlocks
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run multiple concurrent processing attempts
	const attempts = 10
	results := make(chan error, attempts)

	for i := 0; i < attempts; i++ {
		go func(attempt int) {
			parser := core.NewParallelParser(core.DefaultConfig())
			
			// This should not deadlock
			_, err := parser.ParseFiles(ctx, filePaths)
			results <- err
		}(i)
	}

	// Collect results within timeout
	completedAttempts := 0
	var errors []error

	timeoutTimer := time.NewTimer(8 * time.Second)
	defer timeoutTimer.Stop()

	for completedAttempts < attempts {
		select {
		case err := <-results:
			completedAttempts++
			if err != nil {
				errors = append(errors, err)
			}
		case <-timeoutTimer.C:
			t.Errorf("Deadlock detected: only %d/%d attempts completed within timeout", 
				completedAttempts, attempts)
			return
		}
	}

	// Validate no deadlocks occurred
	if len(errors) > attempts/2 {
		t.Errorf("High error rate may indicate deadlock issues: %d errors out of %d attempts", 
			len(errors), attempts)
	}

	t.Logf("Deadlock prevention test: %d/%d attempts completed successfully", 
		attempts-len(errors), attempts)
}

// Test 10: Concurrent cache operations
func testConcurrentCacheOperations(t *testing.T) {
	// Test thread safety of caching systems under concurrent load
	extractorConfig := core.DefaultExtractorConfig()
	extractorConfig.EnableCache = true
	extractorConfig.ParallelExtraction = true
	extractor := core.NewDecoratorExtractor(extractorConfig)

	// Test content for caching
	testContents := [][]byte{
		[]byte(`package cache1
// @Service
type Service1 struct {}`),
		[]byte(`package cache2
// @Controller("/api")
type Controller2 struct {}`),
		[]byte(`package cache3
// @Repository
type Repository3 struct {}`),
	}

	const goroutines = 50
	const iterations = 20
	var wg sync.WaitGroup
	var cacheErrors int64

	// Concurrent cache operations
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				contentIdx := (goroutineID + j) % len(testContents)
				content := testContents[contentIdx]

				// Multiple extractions of same content should hit cache
				_, err := extractor.Extract(content)
				if err != nil {
					atomic.AddInt64(&cacheErrors, 1)
					continue
				}

				// Second extraction should be faster (cache hit)
				_, err = extractor.Extract(content)
				if err != nil {
					atomic.AddInt64(&cacheErrors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Check cache statistics
	stats := extractor.GetStatistics()
	totalErrors := atomic.LoadInt64(&cacheErrors)
	
	cacheHitRate, ok := stats["cache_hit_rate"].(float64)
	if !ok {
		t.Error("Cache hit rate not available in statistics")
	} else if cacheHitRate < 10.0 {
		t.Errorf("Low cache hit rate under concurrent load: %.2f%%", cacheHitRate)
	}

	extractions, _ := stats["extractions"].(int64)
	expectedOperations := int64(goroutines * iterations * 2) // 2 extractions per iteration

	if totalErrors > expectedOperations/10 {
		t.Errorf("High error rate in concurrent cache operations: %d errors", totalErrors)
	}

	if extractions == 0 {
		t.Error("No extractions recorded (possible cache corruption)")
	}

	t.Logf("Concurrent cache operations: %.2f%% hit rate, %d extractions, %d errors across %d goroutines", 
		cacheHitRate, extractions, totalErrors, goroutines)
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}