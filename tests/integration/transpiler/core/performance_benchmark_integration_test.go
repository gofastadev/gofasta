package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

// TestPerformanceBenchmarkIntegration tests realistic performance scenarios and benchmarks
func TestPerformanceBenchmarkIntegration(t *testing.T) {
	t.Run("RealisticPerformanceScenarios", testRealisticPerformanceScenarios)
	t.Run("LargeCodebaseProcessing", testLargeCodebaseProcessing)
	t.Run("MemoryUsageProfiling", testMemoryUsageProfiling)
	t.Run("ProcessingSpeedBenchmarks", testProcessingSpeedBenchmarks)
	t.Run("ScalabilityTesting", testScalabilityTesting)
	t.Run("PerformanceRegressionDetection", testPerformanceRegressionDetection)
	t.Run("ComponentPerformanceIsolation", testComponentPerformanceIsolation)
}

// Test 1: Realistic performance scenarios
func testRealisticPerformanceScenarios(t *testing.T) {
	testDir := createTestDir(t, "realistic_perf_test")
	defer os.RemoveAll(testDir)

	// Create realistic project structures with varying complexity
	scenarios := []struct {
		name           string
		fileCount      int
		avgFileSize    int
		decoratorCount int
		expectMinRate  float64 // minimum files per second
	}{
		{"SmallProject", 50, 500, 3, 1000},
		{"MediumProject", 200, 1500, 8, 500},
		{"LargeProject", 500, 3000, 15, 200},
		{"EnterpriseProject", 1000, 5000, 25, 100},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenarioDir := filepath.Join(testDir, scenario.name)
			if err := os.MkdirAll(scenarioDir, 0755); err != nil {
				t.Fatalf("Failed to create scenario directory: %v", err)
			}

			var filePaths []string

			// Create realistic file structure
			for i := 0; i < scenario.fileCount; i++ {
				// Distribute files across subdirectories
				subDir := fmt.Sprintf("module%d", i/20)
				moduleDir := filepath.Join(scenarioDir, subDir)
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					t.Fatalf("Failed to create module directory: %v", err)
				}

				filename := fmt.Sprintf("service_%03d.gofa", i)
				filepath := filepath.Join(moduleDir, filename)

				// Generate realistic content
				content := generateRealisticContent(i, scenario.avgFileSize, scenario.decoratorCount)

				if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create realistic test file: %v", err)
				}
				filePaths = append(filePaths, filepath)
			}

			// Benchmark parsing performance
			config := core.DefaultConfig()
			config.MaxWorkers = runtime.NumCPU()
			parser := core.NewParallelParser(config)

			// Run multiple iterations for stable measurements
			const iterations = 3
			var totalDuration time.Duration
			var successfulRuns int

			for iter := 0; iter < iterations; iter++ {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

				start := time.Now()
				results, err := parser.ParseFiles(ctx, filePaths)
				duration := time.Since(start)
				cancel()

				if err != nil {
					t.Logf("Iteration %d failed: %v", iter+1, err)
					continue
				}

				// Validate results
				successCount := 0
				for _, result := range results {
					if result.Error == nil {
						successCount++
					}
				}

				successRate := float64(successCount) / float64(len(filePaths))
				if successRate < 0.95 {
					t.Errorf("Low success rate in %s iteration %d: %.1f%%", scenario.name, iter+1, successRate*100)
					continue
				}

				totalDuration += duration
				successfulRuns++
			}

			if successfulRuns == 0 {
				t.Fatalf("No successful benchmark runs for %s", scenario.name)
			}

			avgDuration := totalDuration / time.Duration(successfulRuns)
			filesPerSec := float64(scenario.fileCount) / avgDuration.Seconds()

			// Performance validation
			if filesPerSec < scenario.expectMinRate {
				t.Errorf("%s performance below expected: %.2f < %.2f files/sec",
					scenario.name, filesPerSec, scenario.expectMinRate)
			}

			// Get detailed statistics
			stats := parser.GetStatistics()
			actualFilesPerSec, _ := stats["files_per_second"].(float64)

			t.Logf("%s Performance: %.2f files/sec (measured), %.2f files/sec (calculated), %v avg duration, %d files",
				scenario.name, actualFilesPerSec, filesPerSec, avgDuration, scenario.fileCount)
		})
	}
}

// Test 2: Large codebase processing
func testLargeCodebaseProcessing(t *testing.T) {
	// Skip if running in short mode or limited resources
	if testing.Short() {
		t.Skip("Skipping large codebase test in short mode")
	}

	testDir := createTestDir(t, "large_codebase_test")
	defer os.RemoveAll(testDir)

	// Create a large codebase simulation
	const (
		totalFiles      = 2000
		packagesCount   = 50
		filesPerPackage = totalFiles / packagesCount
	)

	var allFilePaths []string

	// Create package structure
	for pkg := 0; pkg < packagesCount; pkg++ {
		packageDir := filepath.Join(testDir, fmt.Sprintf("pkg%03d", pkg))
		if err := os.MkdirAll(packageDir, 0755); err != nil {
			t.Fatalf("Failed to create package directory: %v", err)
		}

		for file := 0; file < filesPerPackage; file++ {
			filename := fmt.Sprintf("file_%03d.gofa", file)
			filepath := filepath.Join(packageDir, filename)

			// Generate varied content sizes
			contentSize := 1000 + (file*100)%5000 // 1KB to 6KB files
			decorators := 5 + (file*2)%20         // 5 to 25 decorators
			content := generateRealisticContent(pkg*filesPerPackage+file, contentSize, decorators)

			if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create large codebase file: %v", err)
			}
			allFilePaths = append(allFilePaths, filepath)
		}
	}

	t.Logf("Created large codebase: %d files across %d packages", totalFiles, packagesCount)

	// Test with different worker configurations
	workerConfigs := []int{
		1,
		runtime.NumCPU() / 2,
		runtime.NumCPU(),
		runtime.NumCPU() * 2,
	}

	for _, workerCount := range workerConfigs {
		t.Run(fmt.Sprintf("Workers_%d", workerCount), func(t *testing.T) {
			config := core.DefaultConfig()
			config.MaxWorkers = workerCount
			parser := core.NewParallelParser(config)

			// Extended timeout for large codebase
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, allFilePaths)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Large codebase processing failed with %d workers: %v", workerCount, err)
			}

			// Validate results
			successCount := 0
			var totalBytes int64
			for _, result := range results {
				if result.Error == nil {
					successCount++
					totalBytes += result.Size
				}
			}

			successRate := float64(successCount) / float64(totalFiles)
			if successRate < 0.99 {
				t.Errorf("Large codebase success rate too low with %d workers: %.1f%%",
					workerCount, successRate*100)
			}

			filesPerSec := float64(totalFiles) / duration.Seconds()
			bytesPerSec := float64(totalBytes) / duration.Seconds()

			// Performance thresholds for large codebases
			minFilesPerSec := 50.0 // Conservative for large datasets
			if filesPerSec < minFilesPerSec {
				t.Errorf("Large codebase processing too slow with %d workers: %.2f files/sec",
					workerCount, filesPerSec)
			}

			t.Logf("Large codebase with %d workers: %.2f files/sec, %.2f MB/sec, %v duration, %.1f%% success",
				workerCount, filesPerSec, bytesPerSec/1024/1024, duration, successRate*100)
		})
	}
}

// Test 3: Memory usage profiling
func testMemoryUsageProfiling(t *testing.T) {
	testDir := createTestDir(t, "memory_profile_test")
	defer os.RemoveAll(testDir)

	// Create files of varying sizes to test memory behavior
	fileSizes := []struct {
		name  string
		size  int
		count int
	}{
		{"Small", 500, 100},  // 100 small files
		{"Medium", 5000, 50}, // 50 medium files
		{"Large", 50000, 10}, // 10 large files
		{"Huge", 100000, 5},  // 5 huge files
	}

	var allFiles []string
	for _, sizeConfig := range fileSizes {
		for i := 0; i < sizeConfig.count; i++ {
			filename := fmt.Sprintf("%s_%03d.gofa", sizeConfig.name, i)
			filepath := filepath.Join(testDir, filename)

			decoratorCount := sizeConfig.size / 200 // Scale decorators with size
			content := generateRealisticContent(i, sizeConfig.size, decoratorCount)

			if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create memory profile test file: %v", err)
			}
			allFiles = append(allFiles, filepath)
		}
	}

	// Test memory usage with different configurations
	configs := []struct {
		name       string
		maxWorkers int
		cacheSize  int
	}{
		{"LowMemory", 2, 50},
		{"Balanced", runtime.NumCPU(), 200},
		{"HighPerf", runtime.NumCPU() * 2, 500},
	}

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			// Measure memory before
			var memBefore runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&memBefore)

			// Configure parser
			parserConfig := core.DefaultConfig()
			parserConfig.MaxWorkers = config.maxWorkers
			// Note: CacheSize configuration would be handled by individual components
			parser := core.NewParallelParser(parserConfig)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, allFiles)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Memory profile test failed: %v", err)
			}

			// Measure memory after
			var memAfter runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&memAfter)

			// Calculate memory usage
			memUsedMB := float64(memAfter.Sys-memBefore.Sys) / 1024 / 1024
			allocsMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / 1024 / 1024

			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}

			filesPerSec := float64(successCount) / duration.Seconds()
			memPerFile := memUsedMB / float64(successCount)

			// Memory usage validation
			maxMemPerFileMB := 0.5 // 500KB per file seems reasonable
			if memPerFile > maxMemPerFileMB {
				t.Errorf("High memory usage per file in %s: %.2f MB/file", config.name, memPerFile)
			}

			t.Logf("Memory profile %s: %.2f MB sys, %.2f MB allocs, %.3f MB/file, %.2f files/sec, %d workers",
				config.name, memUsedMB, allocsMB, memPerFile, filesPerSec, config.maxWorkers)
		})
	}
}

// Test 4: Processing speed benchmarks
func testProcessingSpeedBenchmarks(t *testing.T) {
	testDir := createTestDir(t, "speed_benchmark_test")
	defer os.RemoveAll(testDir)

	// Create benchmark datasets
	benchmarks := []struct {
		name           string
		fileCount      int
		fileSize       int
		complexity     string
		minFilesPerSec float64
	}{
		{"SimpleFiles", 500, 800, "simple", 2000},
		{"ComplexFiles", 200, 2000, "complex", 800},
		{"MixedComplexity", 300, 1500, "mixed", 1000},
		{"DecoratorHeavy", 100, 3000, "decorators", 400},
	}

	for _, benchmark := range benchmarks {
		t.Run(benchmark.name, func(t *testing.T) {
			benchmarkDir := filepath.Join(testDir, benchmark.name)
			if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
				t.Fatalf("Failed to create benchmark directory: %v", err)
			}

			var filePaths []string
			for i := 0; i < benchmark.fileCount; i++ {
				filename := fmt.Sprintf("benchmark_%03d.gofa", i)
				filepath := filepath.Join(benchmarkDir, filename)

				var content string
				switch benchmark.complexity {
				case "simple":
					content = generateSimpleContent(i, benchmark.fileSize)
				case "complex":
					content = generateComplexContent(i, benchmark.fileSize)
				case "mixed":
					if i%2 == 0 {
						content = generateSimpleContent(i, benchmark.fileSize)
					} else {
						content = generateComplexContent(i, benchmark.fileSize)
					}
				case "decorators":
					decoratorCount := benchmark.fileSize / 100
					content = generateRealisticContent(i, benchmark.fileSize, decoratorCount)
				}

				if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create benchmark file: %v", err)
				}
				filePaths = append(filePaths, filepath)
			}

			// Run benchmark with optimal settings
			config := core.DefaultConfig()
			config.MaxWorkers = runtime.NumCPU()
			parser := core.NewParallelParser(config)

			// Multiple runs for stable measurements
			const runs = 5
			var totalDuration time.Duration
			var successfulRuns int

			for run := 0; run < runs; run++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

				start := time.Now()
				results, err := parser.ParseFiles(ctx, filePaths)
				duration := time.Since(start)
				cancel()

				if err != nil {
					t.Logf("Run %d failed: %v", run+1, err)
					continue
				}

				successCount := 0
				for _, result := range results {
					if result.Error == nil {
						successCount++
					}
				}

				if successCount < benchmark.fileCount*95/100 {
					t.Logf("Run %d had low success rate: %d/%d", run+1, successCount, benchmark.fileCount)
					continue
				}

				totalDuration += duration
				successfulRuns++
			}

			if successfulRuns == 0 {
				t.Fatalf("No successful benchmark runs for %s", benchmark.name)
			}

			avgDuration := totalDuration / time.Duration(successfulRuns)
			filesPerSec := float64(benchmark.fileCount) / avgDuration.Seconds()

			// Performance validation
			if filesPerSec < benchmark.minFilesPerSec {
				t.Errorf("%s benchmark below threshold: %.2f < %.2f files/sec",
					benchmark.name, filesPerSec, benchmark.minFilesPerSec)
			}

			stats := parser.GetStatistics()
			actualFilesPerSec, _ := stats["files_per_second"].(float64)

			t.Logf("%s Benchmark: %.2f files/sec avg (%.2f measured), %v avg duration, %d files, %d successful runs",
				benchmark.name, filesPerSec, actualFilesPerSec, avgDuration, benchmark.fileCount, successfulRuns)
		})
	}
}

// Test 5: Scalability testing
func testScalabilityTesting(t *testing.T) {
	testDir := createTestDir(t, "scalability_test")
	defer os.RemoveAll(testDir)

	// Test scalability with increasing file counts
	fileCounts := []int{10, 50, 100, 250, 500, 1000}

	for _, fileCount := range fileCounts {
		t.Run(fmt.Sprintf("Files_%d", fileCount), func(t *testing.T) {
			// Create test files
			var filePaths []string
			for i := 0; i < fileCount; i++ {
				filename := fmt.Sprintf("scale_%04d.gofa", i)
				filepath := filepath.Join(testDir, filename)

				content := generateRealisticContent(i, 1500, 8) // Standard complexity
				if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create scalability test file: %v", err)
				}
				filePaths = append(filePaths, filepath)
			}

			// Test with optimal worker configuration
			config := core.DefaultConfig()
			config.MaxWorkers = runtime.NumCPU()
			parser := core.NewParallelParser(config)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, filePaths)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Scalability test failed for %d files: %v", fileCount, err)
			}

			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}

			successRate := float64(successCount) / float64(fileCount)
			filesPerSec := float64(successCount) / duration.Seconds()

			// Scalability validation - should maintain reasonable performance
			expectedMinRate := 100.0 // Conservative minimum
			if filesPerSec < expectedMinRate && fileCount > 100 {
				t.Errorf("Scalability degraded with %d files: %.2f files/sec", fileCount, filesPerSec)
			}

			if successRate < 0.98 {
				t.Errorf("Low success rate with %d files: %.1f%%", fileCount, successRate*100)
			}

			t.Logf("Scalability %d files: %.2f files/sec, %v duration, %.1f%% success",
				fileCount, filesPerSec, duration, successRate*100)

			// Clean up files for next iteration
			for _, path := range filePaths {
				os.Remove(path)
			}
		})
	}
}

// Test 6: Performance regression detection
func testPerformanceRegressionDetection(t *testing.T) {
	testDir := createTestDir(t, "regression_test")
	defer os.RemoveAll(testDir)

	// Create a standard test dataset for regression testing
	const standardFileCount = 200
	var filePaths []string

	for i := 0; i < standardFileCount; i++ {
		filename := fmt.Sprintf("regression_%03d.gofa", i)
		filepath := filepath.Join(testDir, filename)

		// Use consistent content for reproducible performance
		content := generateRealisticContent(i, 2000, 10)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create regression test file: %v", err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Establish baseline performance
	config := core.DefaultConfig()
	config.MaxWorkers = runtime.NumCPU()
	parser := core.NewParallelParser(config)

	const baselineRuns = 10
	var baselineResults []float64

	for run := 0; run < baselineRuns; run++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		start := time.Now()
		results, err := parser.ParseFiles(ctx, filePaths)
		duration := time.Since(start)
		cancel()

		if err != nil {
			t.Fatalf("Baseline run %d failed: %v", run+1, err)
		}

		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}

		if successCount < standardFileCount*95/100 {
			t.Logf("Baseline run %d had low success rate, retrying", run+1)
			run-- // Retry this run
			continue
		}

		filesPerSec := float64(successCount) / duration.Seconds()
		baselineResults = append(baselineResults, filesPerSec)
	}

	// Calculate baseline statistics
	var sum, min, max float64
	min = baselineResults[0]
	max = baselineResults[0]

	for _, result := range baselineResults {
		sum += result
		if result < min {
			min = result
		}
		if result > max {
			max = result
		}
	}

	avgBaseline := sum / float64(len(baselineResults))

	// Calculate variance
	var variance float64
	for _, result := range baselineResults {
		diff := result - avgBaseline
		variance += diff * diff
	}
	variance /= float64(len(baselineResults))
	stdDev := fmt.Sprintf("%.2f", variance)

	// Performance regression thresholds
	regressionThreshold := avgBaseline * 0.8  // 20% degradation threshold
	improvementThreshold := avgBaseline * 1.2 // 20% improvement detection

	t.Logf("Performance baseline established: avg=%.2f files/sec, min=%.2f, max=%.2f, stddev=%s",
		avgBaseline, min, max, stdDev)
	t.Logf("Regression threshold: %.2f files/sec, Improvement threshold: %.2f files/sec",
		regressionThreshold, improvementThreshold)

	// Test current performance against baseline
	const currentRuns = 5
	var currentResults []float64

	for run := 0; run < currentRuns; run++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		start := time.Now()
		results, err := parser.ParseFiles(ctx, filePaths)
		duration := time.Since(start)
		cancel()

		if err != nil {
			t.Errorf("Current performance run %d failed: %v", run+1, err)
			continue
		}

		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}

		filesPerSec := float64(successCount) / duration.Seconds()
		currentResults = append(currentResults, filesPerSec)
	}

	if len(currentResults) == 0 {
		t.Fatal("No successful current performance runs")
	}

	// Calculate current average
	var currentSum float64
	for _, result := range currentResults {
		currentSum += result
	}
	avgCurrent := currentSum / float64(len(currentResults))

	// Regression detection
	if avgCurrent < regressionThreshold {
		t.Errorf("PERFORMANCE REGRESSION DETECTED: current %.2f files/sec < threshold %.2f files/sec (%.1f%% degradation)",
			avgCurrent, regressionThreshold, (avgBaseline-avgCurrent)/avgBaseline*100)
	} else if avgCurrent > improvementThreshold {
		t.Logf("PERFORMANCE IMPROVEMENT DETECTED: current %.2f files/sec > baseline %.2f files/sec (%.1f%% improvement)",
			avgCurrent, avgBaseline, (avgCurrent-avgBaseline)/avgBaseline*100)
	} else {
		t.Logf("Performance stable: current %.2f files/sec vs baseline %.2f files/sec (%.1f%% change)",
			avgCurrent, avgBaseline, (avgCurrent-avgBaseline)/avgBaseline*100)
	}
}

// Test 7: Component performance isolation
func testComponentPerformanceIsolation(t *testing.T) {
	testDir := createTestDir(t, "component_perf_test")
	defer os.RemoveAll(testDir)

	// Create test content for component isolation
	const testFileCount = 100
	var testContent [][]byte
	var filePaths []string

	for i := 0; i < testFileCount; i++ {
		filename := fmt.Sprintf("component_%03d.gofa", i)
		filepath := filepath.Join(testDir, filename)

		content := generateRealisticContent(i, 2000, 12)
		testContent = append(testContent, []byte(content))

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create component performance test file: %v", err)
		}
		filePaths = append(filePaths, filepath)
	}

	// Test individual component performance
	components := []struct {
		name     string
		testFunc func() (float64, error)
	}{
		{"Parser", func() (float64, error) {
			config := core.DefaultConfig()
			parser := core.NewParallelParser(config)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			start := time.Now()
			results, err := parser.ParseFiles(ctx, filePaths)
			duration := time.Since(start)

			if err != nil {
				return 0, err
			}

			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}

			return float64(successCount) / duration.Seconds(), nil
		}},
		{"DecoratorExtractor", func() (float64, error) {
			extractor := core.NewDecoratorExtractor(core.DefaultExtractorConfig())

			start := time.Now()
			successCount := 0

			for _, content := range testContent {
				_, err := extractor.Extract(content)
				if err == nil {
					successCount++
				}
			}

			duration := time.Since(start)
			return float64(successCount) / duration.Seconds(), nil
		}},
		{"CodeGenerator", func() (float64, error) {
			generator := core.NewCodeGenerator(core.DefaultGeneratorConfig())

			start := time.Now()
			successCount := 0

			for i := 0; i < testFileCount; i++ {
				typeDef := core.TypeDefinition{
					Name: fmt.Sprintf("Generated%d", i),
					Kind: "struct",
				}

				_, err := generator.GenerateStruct(typeDef)
				if err == nil {
					successCount++
				}
			}

			duration := time.Since(start)
			return float64(successCount) / duration.Seconds(), nil
		}},
	}

	// Benchmark each component in isolation
	for _, component := range components {
		t.Run(component.name, func(t *testing.T) {
			const runs = 5
			var results []float64

			for run := 0; run < runs; run++ {
				result, err := component.testFunc()
				if err != nil {
					t.Errorf("Component %s run %d failed: %v", component.name, run+1, err)
					continue
				}
				results = append(results, result)
			}

			if len(results) == 0 {
				t.Fatalf("No successful runs for component %s", component.name)
			}

			var sum, min, max float64
			min = results[0]
			max = results[0]

			for _, result := range results {
				sum += result
				if result < min {
					min = result
				}
				if result > max {
					max = result
				}
			}

			avg := sum / float64(len(results))

			// Component-specific performance expectations
			var expectedMin float64
			switch component.name {
			case "Parser":
				expectedMin = 200.0 // files per second
			case "DecoratorExtractor":
				expectedMin = 500.0 // extractions per second
			case "CodeGenerator":
				expectedMin = 1000.0 // generations per second
			}

			if avg < expectedMin {
				t.Errorf("Component %s performance below expected: %.2f < %.2f ops/sec",
					component.name, avg, expectedMin)
			}

			t.Logf("Component %s performance: avg=%.2f ops/sec, min=%.2f, max=%.2f, %d runs",
				component.name, avg, min, max, len(results))
		})
	}
}

// Helper functions for generating test content

func generateRealisticContent(id int, targetSize int, decoratorCount int) string {
	var builder strings.Builder

	packageName := fmt.Sprintf("realistic%d", id)
	builder.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Add imports
	builder.WriteString("import (\n")
	builder.WriteString("\t\"context\"\n")
	builder.WriteString("\t\"database/sql\"\n")
	builder.WriteString("\t\"net/http\"\n")
	builder.WriteString("\t\"time\"\n")
	builder.WriteString(")\n\n")

	// Add decorators
	for d := 0; d < decoratorCount; d++ {
		decorator := fmt.Sprintf("// @Decorator%d(\"param%d\", %d)\n", d, d, d*10)
		builder.WriteString(decorator)
	}

	// Add struct
	builder.WriteString(fmt.Sprintf("type Service%d struct {\n", id))
	builder.WriteString("\tdb *sql.DB\n")
	builder.WriteString("\tcache map[string]interface{}\n")
	builder.WriteString("\tlogger Logger\n")
	builder.WriteString("}\n\n")

	// Add methods to reach target size
	methodCount := targetSize / 200 // Rough estimate
	for m := 0; m < methodCount; m++ {
		builder.WriteString(fmt.Sprintf("// @Method%d(\"endpoint/%d\")\n", m, m))
		builder.WriteString(fmt.Sprintf("func (s *Service%d) Method%d(ctx context.Context) error {\n", id, m))
		builder.WriteString("\t// Implementation details\n")
		builder.WriteString("\treturn nil\n")
		builder.WriteString("}\n\n")
	}

	return builder.String()
}

func generateSimpleContent(id int, targetSize int) string {
	var builder strings.Builder

	packageName := fmt.Sprintf("simple%d", id)
	builder.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Simple function
	functionCount := targetSize / 100
	for f := 0; f < functionCount; f++ {
		builder.WriteString(fmt.Sprintf("func Function%d() {\n", f))
		builder.WriteString("\t// Simple implementation\n")
		builder.WriteString("}\n\n")
	}

	return builder.String()
}

func generateComplexContent(id int, targetSize int) string {
	var builder strings.Builder

	packageName := fmt.Sprintf("complex%d", id)
	builder.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Complex imports
	builder.WriteString("import (\n")
	builder.WriteString("\t\"context\"\n")
	builder.WriteString("\t\"database/sql\"\n")
	builder.WriteString("\t\"encoding/json\"\n")
	builder.WriteString("\t\"net/http\"\n")
	builder.WriteString("\t\"sync\"\n")
	builder.WriteString("\t\"time\"\n")
	builder.WriteString(")\n\n")

	// Complex structs and interfaces
	builder.WriteString("type ComplexInterface interface {\n")
	builder.WriteString("\tProcess(ctx context.Context, data []byte) (interface{}, error)\n")
	builder.WriteString("\tValidate(input map[string]interface{}) bool\n")
	builder.WriteString("}\n\n")

	// Add content to reach target size
	contentPerMethod := 300
	methodCount := targetSize / contentPerMethod

	for m := 0; m < methodCount; m++ {
		builder.WriteString(fmt.Sprintf("func ComplexMethod%d(ctx context.Context, params map[string]interface{}) (interface{}, error) {\n", m))
		builder.WriteString("\tvar mu sync.RWMutex\n")
		builder.WriteString("\tmu.Lock()\n")
		builder.WriteString("\tdefer mu.Unlock()\n")
		builder.WriteString("\t\n")
		builder.WriteString("\tselect {\n")
		builder.WriteString("\tcase <-ctx.Done():\n")
		builder.WriteString("\t\treturn nil, ctx.Err()\n")
		builder.WriteString("\tdefault:\n")
		builder.WriteString("\t\t// Complex processing logic\n")
		builder.WriteString("\t}\n")
		builder.WriteString("\treturn nil, nil\n")
		builder.WriteString("}\n\n")
	}

	return builder.String()
}
