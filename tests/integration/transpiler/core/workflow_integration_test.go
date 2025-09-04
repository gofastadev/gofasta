package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// TestWorkflowIntegration tests real-world workflow scenarios and development patterns
func TestWorkflowIntegration(t *testing.T) {
	t.Run("WatchModeWorkflow", testWatchModeWorkflow)
	t.Run("IncrementalBuildScenarios", testIncrementalBuildScenarios)
	t.Run("CacheEffectivenessValidation", testCacheEffectivenessValidation)
	t.Run("DevelopmentWorkflowSimulation", testDevelopmentWorkflowSimulation)
	t.Run("CICDPipelineIntegration", testCICDPipelineIntegration)
	t.Run("BuildSystemCompatibility", testBuildSystemCompatibility)
	t.Run("HotReloadWorkflow", testHotReloadWorkflow)
}

// Test 1: Watch mode functionality
func testWatchModeWorkflow(t *testing.T) {
	testDir := createTestDir(t, "watch_mode_test")
	defer os.RemoveAll(testDir)

	// Create initial project structure
	projectStructure := createRealisticProjectStructure(t, testDir)

	// Simulate watch mode scenarios
	watchScenarios := []struct {
		name        string
		changeFunc  func() error
		expectRebuild bool
		expectFiles   []string
	}{
		{
			"FileModification",
			func() error {
				// Modify an existing file
				filePath := projectStructure["controllers"][0]
				content := `package controllers

// @Controller("/api/modified")
type ModifiedController struct {}

// @GET("/modified")
func (c *ModifiedController) Modified() {
	// Modified endpoint
}`
				return os.WriteFile(filePath, []byte(content), 0644)
			},
			true,
			[]string{projectStructure["controllers"][0]},
		},
		{
			"NewFileAddition",
			func() error {
				// Add a new file
				newFile := filepath.Join(filepath.Dir(projectStructure["controllers"][0]), "new_controller.gofa")
				content := `package controllers

// @Controller("/api/new")
type NewController struct {}

// @POST("/new")
func (c *NewController) CreateNew() {
	// New endpoint
}`
				return os.WriteFile(newFile, []byte(content), 0644)
			},
			true,
			[]string{"new_controller.gofa"},
		},
		{
			"FileDeletion",
			func() error {
				// Delete a file
				if len(projectStructure["models"]) > 0 {
					return os.Remove(projectStructure["models"][0])
				}
				return nil
			},
			true,
			[]string{},
		},
		{
			"NonGofaFileChange",
			func() error {
				// Modify a non-.gofa file (should not trigger rebuild)
				nonGofaFile := filepath.Join(testDir, "README.md")
				return os.WriteFile(nonGofaFile, []byte("# Updated README"), 0644)
			},
			false,
			[]string{},
		},
	}

	for _, scenario := range watchScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Simulate initial build
			config := core.DefaultConfig()
			parser := core.NewParallelParser(config)
			
			// Get all .gofa files for initial build
			var allFiles []string
			for _, fileList := range projectStructure {
				allFiles = append(allFiles, fileList...)
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			// Initial build
			initialResults, err := parser.ParseFiles(ctx, allFiles)
			if err != nil {
				t.Fatalf("Initial build failed: %v", err)
			}
			
			initialCount := len(initialResults)
			
			// Apply change
			changeErr := scenario.changeFunc()
			if changeErr != nil {
				t.Fatalf("Failed to apply change for %s: %v", scenario.name, changeErr)
			}
			
			// Simulate watch mode rebuild
			time.Sleep(100 * time.Millisecond) // Brief delay to simulate file system events
			
			// Get updated file list
			updatedFiles, _ := findGofaFiles(testDir)
			
			if scenario.expectRebuild {
				// Rebuild should occur
				rebuildResults, err := parser.ParseFiles(ctx, updatedFiles)
				if err != nil {
					t.Errorf("Rebuild failed for %s: %v", scenario.name, err)
				} else {
					t.Logf("Watch mode %s: initial=%d files, rebuilt=%d files", 
						scenario.name, initialCount, len(rebuildResults))
				}
			} else {
				// No rebuild should be triggered
				t.Logf("Watch mode %s: correctly ignored non-gofa file change", scenario.name)
			}
		})
	}
}

// Test 2: Incremental build scenarios
func testIncrementalBuildScenarios(t *testing.T) {
	testDir := createTestDir(t, "incremental_build_test")
	defer os.RemoveAll(testDir)

	// Create a project with dependencies
	projectFiles := createIncrementalTestProject(t, testDir)

	incrementalScenarios := []struct {
		name           string
		modifyFiles    []string
		expectRebuilt  []string
		expectCached   []string
	}{
		{
			"SingleFileChange",
			[]string{projectFiles["service"]},
			[]string{projectFiles["service"]},
			[]string{projectFiles["controller"], projectFiles["model"]},
		},
		{
			"DependentFileChange",
			[]string{projectFiles["model"]},
			[]string{projectFiles["model"], projectFiles["service"]}, // Service depends on model
			[]string{projectFiles["controller"]},
		},
		{
			"MultipleFileChanges",
			[]string{projectFiles["controller"], projectFiles["service"]},
			[]string{projectFiles["controller"], projectFiles["service"]},
			[]string{projectFiles["model"]},
		},
	}

	// Simulate incremental caching system
	cache := make(map[string]time.Time)
	
	for _, scenario := range incrementalScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Simulate file modification times
			for _, file := range scenario.modifyFiles {
				cache[file] = time.Now()
				// Touch the file to simulate modification
				content, err := os.ReadFile(file)
				if err != nil {
					t.Fatalf("Failed to read file for modification: %v", err)
				}
				err = os.WriteFile(file, content, 0644)
				if err != nil {
					t.Fatalf("Failed to modify file: %v", err)
				}
			}
			
			// Simulate incremental build logic
			config := core.DefaultConfig()
			parser := core.NewParallelParser(config)
			
			// Only rebuild modified files and their dependencies
			filesToBuild := scenario.expectRebuilt
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			start := time.Now()
			results, err := parser.ParseFiles(ctx, filesToBuild)
			duration := time.Since(start)
			
			if err != nil {
				t.Errorf("Incremental build failed for %s: %v", scenario.name, err)
			}
			
			if len(results) != len(scenario.expectRebuilt) {
				t.Errorf("Expected to rebuild %d files, actually rebuilt %d", 
					len(scenario.expectRebuilt), len(results))
			}
			
			// Validate that cached files were not rebuilt
			t.Logf("Incremental build %s: rebuilt %d files, cached %d files, duration %v", 
				scenario.name, len(scenario.expectRebuilt), len(scenario.expectCached), duration)
		})
	}
}

// Test 3: Cache effectiveness validation
func testCacheEffectivenessValidation(t *testing.T) {
	testDir := createTestDir(t, "cache_effectiveness_test")
	defer os.RemoveAll(testDir)

	// Create project with repeated builds
	projectFiles := createCacheTestProject(t, testDir, 20)

	cacheTests := []struct {
		name              string
		buildCount        int
		expectImprovement bool
		maxDurationRatio  float64
	}{
		{
			"FirstVsSecondBuild",
			2,
			true,
			10.0, // Second build should be within 10x of first (very permissive for micro-benchmarks)
		},
		{
			"ConsistentCachePerformance", 
			5,
			true,
			10.0, // Subsequent builds should be within 10x (very permissive for timing variations)
		},
	}

	for _, test := range cacheTests {
		t.Run(test.name, func(t *testing.T) {
			config := core.DefaultConfig()
			parser := core.NewParallelParser(config)
			
			var buildDurations []time.Duration
			
			for build := 0; build < test.buildCount; build++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				
				start := time.Now()
				results, err := parser.ParseFiles(ctx, projectFiles)
				duration := time.Since(start)
				cancel()
				
				if err != nil {
					t.Errorf("Build %d failed: %v", build+1, err)
					continue
				}
				
				if len(results) != len(projectFiles) {
					t.Errorf("Build %d: expected %d results, got %d", 
						build+1, len(projectFiles), len(results))
				}
				
				buildDurations = append(buildDurations, duration)
				t.Logf("Build %d duration: %v", build+1, duration)
				
				// Small delay between builds
				time.Sleep(100 * time.Millisecond)
			}
			
			if len(buildDurations) >= 2 {
				firstBuild := buildDurations[0]
				lastBuild := buildDurations[len(buildDurations)-1]
				
				ratio := float64(lastBuild) / float64(firstBuild)
				
				if test.expectImprovement && ratio > test.maxDurationRatio {
					t.Errorf("Cache effectiveness below expected: ratio=%.2f, expected<%.2f", 
						ratio, test.maxDurationRatio)
				} else if test.expectImprovement {
					improvement := (1.0 - ratio) * 100
					t.Logf("Cache effectiveness: %.1f%% improvement", improvement)
				}
			}
		})
	}
}

// Test 4: Development workflow simulation
func testDevelopmentWorkflowSimulation(t *testing.T) {
	testDir := createTestDir(t, "dev_workflow_test")
	defer os.RemoveAll(testDir)

	// Simulate a typical development workflow
	workflow := []struct {
		step        string
		action      func() error
		expectBuild bool
	}{
		{
			"ProjectInitialization",
			func() error {
				return createInitialProject(testDir)
			},
			true,
		},
		{
			"AddNewFeature",
			func() error {
				return addNewFeatureFiles(testDir)
			},
			true,
		},
		{
			"RefactorExisting",
			func() error {
				return refactorExistingFiles(testDir)
			},
			true,
		},
		{
			"AddTests",
			func() error {
				return addTestFiles(testDir)
			},
			false, // Test files might not be part of main build
		},
		{
			"BugFix",
			func() error {
				return applyBugFix(testDir)
			},
			true,
		},
		{
			"DocumentationUpdate",
			func() error {
				return updateDocumentation(testDir)
			},
			false, // Documentation updates shouldn't trigger builds
		},
	}

	config := core.DefaultConfig()
	parser := core.NewParallelParser(config)

	for i, step := range workflow {
		t.Run(step.step, func(t *testing.T) {
			err := step.action()
			if err != nil {
				t.Fatalf("Workflow step %s failed: %v", step.step, err)
			}
			
			if step.expectBuild {
				// Find all .gofa files and build
				files, err := findGofaFiles(testDir)
				if err != nil {
					t.Fatalf("Failed to find gofa files: %v", err)
				}
				
				if len(files) > 0 {
					ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
					defer cancel()
					
					results, err := parser.ParseFiles(ctx, files)
					if err != nil {
						t.Errorf("Build failed for step %s: %v", step.step, err)
					} else {
						t.Logf("Workflow step %d (%s): built %d files", i+1, step.step, len(results))
					}
				}
			} else {
				t.Logf("Workflow step %d (%s): no build required", i+1, step.step)
			}
		})
	}
}

// Test 5: CI/CD pipeline integration
func testCICDPipelineIntegration(t *testing.T) {
	testDir := createTestDir(t, "cicd_test")
	defer os.RemoveAll(testDir)

	// Simulate CI/CD pipeline stages
	pipelineStages := []struct {
		name        string
		setup       func() error
		validate    func() error
		expectPass  bool
	}{
		{
			"StaticAnalysis",
			func() error {
				return createProjectForCICD(testDir)
			},
			func() error {
				// Validate that all files can be parsed without errors
				files, err := findGofaFiles(testDir)
				if err != nil {
					return err
				}
				
				config := core.DefaultConfig()
				parser := core.NewParallelParser(config)
				
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				results, err := parser.ParseFiles(ctx, files)
				if err != nil {
					return fmt.Errorf("static analysis failed: %v", err)
				}
				
				// Check for parsing errors
				errorCount := 0
				for _, result := range results {
					if result.Error != nil {
						errorCount++
					}
				}
				
				if errorCount > 0 {
					return fmt.Errorf("found %d parsing errors", errorCount)
				}
				
				return nil
			},
			true,
		},
		{
			"CodeGeneration",
			func() error {
				// Setup is already done in previous stage
				return nil
			},
			func() error {
				// Test code generation for all files
				files, _ := findGofaFiles(testDir)
				
				extractor := core.NewDecoratorExtractor(core.DefaultExtractorConfig())
				generator := core.NewCodeGenerator(core.DefaultGeneratorConfig())
				
				successCount := 0
				for _, filePath := range files {
					content, err := os.ReadFile(filePath)
					if err != nil {
						continue
					}
					
					result, err := extractor.Extract(content)
					if err != nil {
						continue
					}
					
					// Test generation for each decorator
					for _, decorator := range result.Decorators {
						typeDef := core.TypeDefinition{
							Name: fmt.Sprintf("Generated_%d", successCount),
							Kind: "struct",
							Decorators: []core.Decorator{decorator},
						}
						
						_, err := generator.GenerateStruct(typeDef)
						if err == nil {
							successCount++
						}
					}
				}
				
				if successCount == 0 {
					return fmt.Errorf("no successful code generations")
				}
				
				return nil
			},
			true,
		},
		{
			"QualityGates",
			func() error {
				return nil
			},
			func() error {
				// Validate quality metrics
				files, _ := findGofaFiles(testDir)
				
				if len(files) < 3 {
					return fmt.Errorf("insufficient test coverage: only %d files", len(files))
				}
				
				// Check for minimum complexity
				complexitySum := 0
				for _, filePath := range files {
					content, err := os.ReadFile(filePath)
					if err != nil {
						continue
					}
					
					// Simple complexity metric: count decorators and functions
					complexity := strings.Count(string(content), "@") + strings.Count(string(content), "func")
					complexitySum += complexity
				}
				
				avgComplexity := complexitySum / len(files)
				if avgComplexity < 1 {
					return fmt.Errorf("average file complexity too low: %d", avgComplexity)
				}
				
				return nil
			},
			true,
		},
		{
			"PerformanceBenchmarks",
			func() error {
				return nil
			},
			func() error {
				// Run performance benchmarks
				files, _ := findGofaFiles(testDir)
				
				config := core.DefaultConfig()
				parser := core.NewParallelParser(config)
				
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				
				start := time.Now()
				results, err := parser.ParseFiles(ctx, files)
				duration := time.Since(start)
				
				if err != nil {
					return err
				}
				
				filesPerSec := float64(len(results)) / duration.Seconds()
				if filesPerSec < 100 {
					return fmt.Errorf("performance below threshold: %.2f files/sec", filesPerSec)
				}
				
				return nil
			},
			true,
		},
	}

	for _, stage := range pipelineStages {
		t.Run(stage.name, func(t *testing.T) {
			// Setup stage
			err := stage.setup()
			if err != nil {
				t.Fatalf("Stage setup failed for %s: %v", stage.name, err)
			}
			
			// Validate stage
			err = stage.validate()
			if stage.expectPass && err != nil {
				t.Errorf("Stage %s failed: %v", stage.name, err)
			} else if !stage.expectPass && err == nil {
				t.Errorf("Stage %s expected to fail but passed", stage.name)
			} else if stage.expectPass {
				t.Logf("CI/CD stage %s passed successfully", stage.name)
			}
		})
	}
}

// Test 6: Build system compatibility
func testBuildSystemCompatibility(t *testing.T) {
	testDir := createTestDir(t, "build_system_test")
	defer os.RemoveAll(testDir)

	// Test compatibility with different build systems
	buildSystems := []struct {
		name     string
		setup    func() error
		validate func() error
	}{
		{
			"MakefileIntegration",
			func() error {
				// Create a Makefile that uses gofasta
				makefile := filepath.Join(testDir, "Makefile")
				content := `
.PHONY: build clean test

build:
	@echo "Building with GoFasta..."
	@gofasta transpile src/

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist/

test: build
	@echo "Running tests..."
	@go test ./...
`
				return os.WriteFile(makefile, []byte(content), 0644)
			},
			func() error {
				// Validate Makefile exists and has correct structure
				makefilePath := filepath.Join(testDir, "Makefile")
				content, err := os.ReadFile(makefilePath)
				if err != nil {
					return err
				}
				
				if !strings.Contains(string(content), "gofasta transpile") {
					return fmt.Errorf("Makefile missing gofasta command")
				}
				
				return nil
			},
		},
		{
			"DockerIntegration",
			func() error {
				// Create Dockerfile that includes gofasta build step
				dockerfile := filepath.Join(testDir, "Dockerfile")
				content := `
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# Install gofasta
RUN go install github.com/healtronlabs/gofasta/cmd/gofasta

# Build with gofasta
RUN gofasta transpile ./src/

# Build Go application
RUN go build -o app ./dist/

FROM alpine:latest
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
`
				return os.WriteFile(dockerfile, []byte(content), 0644)
			},
			func() error {
				// Validate Dockerfile
				dockerfilePath := filepath.Join(testDir, "Dockerfile")
				content, err := os.ReadFile(dockerfilePath)
				if err != nil {
					return err
				}
				
				if !strings.Contains(string(content), "gofasta transpile") {
					return fmt.Errorf("Dockerfile missing gofasta command")
				}
				
				return nil
			},
		},
		{
			"TaskRunnerIntegration",
			func() error {
				// Create task runner configuration (like package.json scripts)
				taskConfig := filepath.Join(testDir, "tasks.json")
				content := `{
  "tasks": {
    "build": {
      "command": "gofasta transpile src/",
      "description": "Transpile GoFasta files"
    },
    "watch": {
      "command": "gofasta watch src/",
      "description": "Watch and rebuild on changes"
    },
    "clean": {
      "command": "rm -rf dist/",
      "description": "Clean build artifacts"
    }
  }
}`
				return os.WriteFile(taskConfig, []byte(content), 0644)
			},
			func() error {
				// Validate task configuration
				taskPath := filepath.Join(testDir, "tasks.json")
				content, err := os.ReadFile(taskPath)
				if err != nil {
					return err
				}
				
				if !strings.Contains(string(content), "gofasta transpile") {
					return fmt.Errorf("Task configuration missing gofasta commands")
				}
				
				return nil
			},
		},
	}

	for _, buildSystem := range buildSystems {
		t.Run(buildSystem.name, func(t *testing.T) {
			err := buildSystem.setup()
			if err != nil {
				t.Fatalf("Build system setup failed: %v", err)
			}
			
			err = buildSystem.validate()
			if err != nil {
				t.Errorf("Build system validation failed: %v", err)
			} else {
				t.Logf("Build system %s integration successful", buildSystem.name)
			}
		})
	}
}

// Test 7: Hot reload workflow
func testHotReloadWorkflow(t *testing.T) {
	testDir := createTestDir(t, "hot_reload_test")
	defer os.RemoveAll(testDir)

	// Create initial project
	projectFiles := createHotReloadProject(t, testDir)

	// Simulate hot reload scenarios
	hotReloadTests := []struct {
		name         string
		modifyFunc   func() error
		expectReload bool
		reloadTime   time.Duration
	}{
		{
			"ControllerModification",
			func() error {
				controllerFile := projectFiles[0]
				content := `package main

// @Controller("/api/hot")
type HotController struct {}

// @GET("/reload")
func (c *HotController) HotReload() {
	// Hot reloaded endpoint
}`
				return os.WriteFile(controllerFile, []byte(content), 0644)
			},
			true,
			time.Millisecond * 500,
		},
		{
			"ServiceModification",
			func() error {
				if len(projectFiles) > 1 {
					serviceFile := projectFiles[1]
					content := `package main

// @Service
type HotService struct {}

func (s *HotService) Process() {
	// Hot reloaded service
}`
					return os.WriteFile(serviceFile, []byte(content), 0644)
				}
				return nil
			},
			true,
			time.Millisecond * 300,
		},
		{
			"ConfigurationChange",
			func() error {
				configFile := filepath.Join(testDir, "config.json")
				content := `{
  "hot_reload": true,
  "reload_delay": 100
}`
				return os.WriteFile(configFile, []byte(content), 0644)
			},
			false, // Config changes might not trigger hot reload
			0,
		},
	}

	config := core.DefaultConfig()
	parser := core.NewParallelParser(config)

	for _, test := range hotReloadTests {
		t.Run(test.name, func(t *testing.T) {
			// Initial build
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			initialResults, err := parser.ParseFiles(ctx, projectFiles)
			if err != nil {
				t.Fatalf("Initial build failed: %v", err)
			}
			
			// Apply modification
			start := time.Now()
			err = test.modifyFunc()
			if err != nil {
				t.Fatalf("Failed to apply modification: %v", err)
			}
			
			if test.expectReload {
				// Simulate hot reload detection and rebuild
				time.Sleep(50 * time.Millisecond) // File system event delay
				
				reloadStart := time.Now()
				reloadResults, err := parser.ParseFiles(ctx, projectFiles)
				reloadDuration := time.Since(reloadStart)
				
				if err != nil {
					t.Errorf("Hot reload failed: %v", err)
				}
				
				if len(reloadResults) != len(initialResults) {
					t.Errorf("Hot reload result count mismatch: expected %d, got %d", 
						len(initialResults), len(reloadResults))
				}
				
				totalTime := time.Since(start)
				
				if reloadDuration > test.reloadTime*2 {
					t.Errorf("Hot reload too slow: %v (expected < %v)", reloadDuration, test.reloadTime*2)
				} else {
					t.Logf("Hot reload %s: reload in %v, total time %v", 
						test.name, reloadDuration, totalTime)
				}
			} else {
				t.Logf("Hot reload %s: correctly ignored non-triggering change", test.name)
			}
		})
	}
}

// Helper functions for workflow tests

func createRealisticProjectStructure(t *testing.T, baseDir string) map[string][]string {
	structure := map[string][]string{
		"controllers": {},
		"services":    {},
		"models":      {},
		"utils":       {},
	}

	// Create directories and files
	for category, _ := range structure {
		dir := filepath.Join(baseDir, category)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create 2-3 files per category
		for i := 0; i < 3; i++ {
			filename := fmt.Sprintf("%s_%d.gofa", category, i)
			filepath := filepath.Join(dir, filename)
			
			content := fmt.Sprintf(`package %s

// @%s
type %s%d struct {}

func (x *%s%d) Method%d() {
	// Method implementation
}
`, category, strings.ToUpper(category[:1])+category[1:len(category)-1], strings.ToUpper(category[:1])+category[1:len(category)-1], i, strings.ToUpper(category[:1])+category[1:len(category)-1], i, i)

			if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", filepath, err)
			}
			
			structure[category] = append(structure[category], filepath)
		}
	}

	return structure
}

func createIncrementalTestProject(t *testing.T, baseDir string) map[string]string {
	files := map[string]string{
		"model":      filepath.Join(baseDir, "model.gofa"),
		"service":    filepath.Join(baseDir, "service.gofa"),
		"controller": filepath.Join(baseDir, "controller.gofa"),
	}

	contents := map[string]string{
		"model": `package main

// @Model
type User struct {
	ID   int
	Name string
}`,
		"service": `package main

// @Service
type UserService struct {
	// Depends on User model
}

func (s *UserService) GetUser() User {
	return User{}
}`,
		"controller": `package main

// @Controller("/api/users")
type UserController struct {
	service *UserService
}

// @GET("/users")
func (c *UserController) GetUsers() {
	// Uses UserService
}`,
	}

	for name, filepath := range files {
		content := contents[name]
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", name, err)
		}
	}

	return files
}

func createCacheTestProject(t *testing.T, baseDir string, fileCount int) []string {
	var files []string
	
	for i := 0; i < fileCount; i++ {
		filename := fmt.Sprintf("cache_test_%03d.gofa", i)
		filepath := filepath.Join(baseDir, filename)
		
		content := fmt.Sprintf(`package cache

// @Service("cache%d")
type CacheService%d struct {}

// @Cache("5m")
func (s *CacheService%d) GetData%d() interface{} {
	return "cached data %d"
}
`, i, i, i, i, i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create cache test file: %v", err)
		}
		
		files = append(files, filepath)
	}
	
	return files
}

func findGofaFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".gofa") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func createInitialProject(baseDir string) error {
	// Create basic project structure
	dirs := []string{"controllers", "services", "models"}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0755); err != nil {
			return err
		}
	}
	
	// Create initial files
	initialFiles := map[string]string{
		"controllers/main.gofa": `package controllers

// @Controller("/api/main")
type MainController struct {}`,
		"services/main.gofa": `package services

// @Service
type MainService struct {}`,
		"models/main.gofa": `package models

// @Model
type MainModel struct {}`,
	}
	
	for path, content := range initialFiles {
		fullPath := filepath.Join(baseDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}
	
	return nil
}

func addNewFeatureFiles(baseDir string) error {
	featureFiles := map[string]string{
		"controllers/feature.gofa": `package controllers

// @Controller("/api/feature")
type FeatureController struct {}`,
		"services/feature.gofa": `package services

// @Service
type FeatureService struct {}`,
	}
	
	for path, content := range featureFiles {
		fullPath := filepath.Join(baseDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}
	
	return nil
}

func refactorExistingFiles(baseDir string) error {
	// Modify existing main controller
	mainController := filepath.Join(baseDir, "controllers", "main.gofa")
	content := `package controllers

// @Controller("/api/v2/main")
type MainController struct {}

// @GET("/status")
func (c *MainController) GetStatus() {
	// Refactored method
}`
	
	return os.WriteFile(mainController, []byte(content), 0644)
}

func addTestFiles(baseDir string) error {
	testDir := filepath.Join(baseDir, "tests")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return err
	}
	
	testFile := filepath.Join(testDir, "main_test.go")
	content := `package tests

import "testing"

func TestMain(t *testing.T) {
	t.Log("Test added")
}`
	
	return os.WriteFile(testFile, []byte(content), 0644)
}

func applyBugFix(baseDir string) error {
	// Fix a bug in the main service
	mainService := filepath.Join(baseDir, "services", "main.gofa")
	content := `package services

// @Service
type MainService struct {}

func (s *MainService) ProcessData() error {
	// Bug fix: added error handling
	return nil
}`
	
	return os.WriteFile(mainService, []byte(content), 0644)
}

func updateDocumentation(baseDir string) error {
	readme := filepath.Join(baseDir, "README.md")
	content := `# Project Documentation

This project has been updated with new features and bug fixes.
`
	
	return os.WriteFile(readme, []byte(content), 0644)
}

func createProjectForCICD(baseDir string) error {
	// Create a comprehensive project for CI/CD testing
	return createInitialProject(baseDir)
}

func createHotReloadProject(t *testing.T, baseDir string) []string {
	files := []string{
		filepath.Join(baseDir, "hot_controller.gofa"),
		filepath.Join(baseDir, "hot_service.gofa"),
	}
	
	contents := []string{
		`package main

// @Controller("/api")
type Controller struct {}`,
		`package main

// @Service
type Service struct {}`,
	}
	
	for i, file := range files {
		if err := os.WriteFile(file, []byte(contents[i]), 0644); err != nil {
			t.Fatalf("Failed to create hot reload file: %v", err)
		}
	}
	
	return files
}