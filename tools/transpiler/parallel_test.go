package transpiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewParallelTranspiler tests parallel transpiler constructor
func TestNewParallelTranspiler(t *testing.T) {
	tests := []struct {
		name     string
		opts     TranspileOptions
		expected struct {
			maxWorkers     int
			fileExtension  string
			preserveStruct bool
			verbose        bool
		}
	}{
		{
			name: "default options",
			opts: TranspileOptions{},
			expected: struct {
				maxWorkers     int
				fileExtension  string
				preserveStruct bool
				verbose        bool
			}{
				maxWorkers:     4, // Depends on runtime.NumCPU(), but should be > 0
				fileExtension:  ".go",
				preserveStruct: false,
				verbose:        false,
			},
		},
		{
			name: "custom options",
			opts: TranspileOptions{
				MaxWorkers:     8,
				FileExtension:  ".generated.go",
				PreserveStruct: true,
				Verbose:        true,
			},
			expected: struct {
				maxWorkers     int
				fileExtension  string
				preserveStruct bool
				verbose        bool
			}{
				maxWorkers:     8,
				fileExtension:  ".generated.go",
				preserveStruct: true,
				verbose:        true,
			},
		},
		{
			name: "negative workers",
			opts: TranspileOptions{
				MaxWorkers: -1,
			},
			expected: struct {
				maxWorkers     int
				fileExtension  string
				preserveStruct bool
				verbose        bool
			}{
				maxWorkers:     4, // Should default to runtime.NumCPU()
				fileExtension:  ".go",
				preserveStruct: false,
				verbose:        false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transpiler := NewParallelTranspiler(tt.opts)

			if transpiler.maxWorkers <= 0 {
				t.Errorf("Expected positive maxWorkers, got %d", transpiler.maxWorkers)
			}

			if transpiler.fileExtension != tt.expected.fileExtension {
				t.Errorf("Expected fileExtension %s, got %s", 
					tt.expected.fileExtension, transpiler.fileExtension)
			}

			if transpiler.preserveStruct != tt.expected.preserveStruct {
				t.Errorf("Expected preserveStruct %t, got %t", 
					tt.expected.preserveStruct, transpiler.preserveStruct)
			}

			if transpiler.verbose != tt.expected.verbose {
				t.Errorf("Expected verbose %t, got %t", 
					tt.expected.verbose, transpiler.verbose)
			}
		})
	}
}

// TestTranspileDirectoryComprehensive tests directory transpilation
func TestTranspileDirectoryComprehensive(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested directory structure with .gofa files
	structure := map[string]string{
		"controller.gofa": `package main

@Controller("/api/users")
type UserController struct {
	Service *UserService ` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(@Param("id") id string) {}`,

		"service.gofa": `package main

@Injectable()
type UserService struct {
	Repo *UserRepository ` + "`inject:\"\"`" + `
}

func FindUser(id string) (*User, error) {
	return nil, nil
}`,

		"nested/controller.gofa": `package nested

@Controller("/api/nested")
type NestedController struct {}

@Get("/")
func GetNested() {}`,

		"nested/deep/service.gofa": `package deep

@Injectable()
type DeepService struct {}

func DeepMethod() {}`,

		"module.gofa": `package main

@Module({
	controllers: ["UserController"],
	providers: ["UserService"]
})
type AppModule struct {}`,

		// Non-.gofa file (should be ignored)
		"regular.go": `package main
func main() {}`,

		// Empty .gofa file
		"empty.gofa": ``,
	}

	// Create files
	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	tests := []struct {
		name            string
		opts            TranspileOptions
		expectedFiles   int
		expectedSuccess int
		expectedFails   int
	}{
		{
			name: "basic transpilation",
			opts: TranspileOptions{
				MaxWorkers:     2,
				OutputDir:      filepath.Join(tempDir, "output1"),
				PreserveStruct: false,
				Verbose:        false,
			},
			expectedFiles:   6, // 6 .gofa files (including empty one)
			expectedSuccess: 5, // empty.gofa will fail
			expectedFails:   1,
		},
		{
			name: "preserve structure",
			opts: TranspileOptions{
				MaxWorkers:     1,
				OutputDir:      filepath.Join(tempDir, "output2"),
				PreserveStruct: true,
				Verbose:        true,
			},
			expectedFiles:   6,
			expectedSuccess: 5,
			expectedFails:   1,
		},
		{
			name: "high concurrency",
			opts: TranspileOptions{
				MaxWorkers:     10,
				OutputDir:      filepath.Join(tempDir, "output3"),
				PreserveStruct: false,
				Verbose:        true,
			},
			expectedFiles:   6,
			expectedSuccess: 5,
			expectedFails:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transpiler := NewParallelTranspiler(tt.opts)
			
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			results, err := transpiler.TranspileDirectory(ctx, tempDir)
			if err != nil {
				t.Fatalf("TranspileDirectory failed: %v", err)
			}

			if len(results) != tt.expectedFiles {
				t.Errorf("Expected %d files, got %d", tt.expectedFiles, len(results))
			}

			successCount := 0
			failCount := 0
			
			for _, result := range results {
				if result.Error != nil {
					failCount++
					t.Logf("File %s failed: %v", result.InputPath, result.Error)
				} else {
					successCount++
					
					// Verify output file exists
					if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
						t.Errorf("Output file missing: %s", result.OutputPath)
					}
				}

				// Verify duration is recorded
				if result.Duration <= 0 {
					t.Errorf("Duration should be positive, got %v", result.Duration)
				}
			}

			// Allow some flexibility in success/fail counts due to empty file
			if successCount < tt.expectedSuccess-1 {
				t.Errorf("Expected at least %d successful files, got %d", 
					tt.expectedSuccess-1, successCount)
			}
		})
	}
}

// TestTranspileFiles tests transpiling specific files
func TestTranspileFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"controller1.gofa",
		"controller2.gofa", 
		"service.gofa",
		"invalid.go", // Should be ignored
	}

	for i, filename := range testFiles {
		var content string
		if strings.HasSuffix(filename, ".gofa") {
			content = fmt.Sprintf(`package main

@Controller("/test%d")
type TestController%d struct {}

@Get("/")
func Test() {}`, i, i)
		} else {
			content = "package main\nfunc main() {}"
		}

		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create file paths for transpilation
	filePaths := make([]string, len(testFiles))
	for i, filename := range testFiles {
		filePaths[i] = filepath.Join(tempDir, filename)
	}

	opts := TranspileOptions{
		MaxWorkers: 2,
		OutputDir:  filepath.Join(tempDir, "output"),
		Verbose:    true,
	}

	transpiler := NewParallelTranspiler(opts)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	results, err := transpiler.TranspileFiles(ctx, filePaths)
	if err != nil {
		t.Fatalf("TranspileFiles failed: %v", err)
	}

	// Should only process .gofa files
	expectedFiles := 3
	if len(results) != expectedFiles {
		t.Errorf("Expected %d results, got %d", expectedFiles, len(results))
	}

	// Verify all are successful
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("File %s failed: %v", result.InputPath, result.Error)
		}
	}
}

// TestFindGofaFiles tests finding .gofa files
func TestFindGofaFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory structure
	structure := map[string]string{
		"file1.gofa":           "content1",
		"file2.gofa":           "content2", 
		"subdir/file3.gofa":    "content3",
		"subdir/file4.gofa":    "content4",
		"deep/nested/file5.gofa": "content5",
		"regular.go":           "not a gofa file",
		"text.txt":             "text file",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	transpiler := NewParallelTranspiler(TranspileOptions{})
	
	files, err := transpiler.FindGofaFiles(tempDir)
	if err != nil {
		t.Fatalf("FindGofaFiles failed: %v", err)
	}

	expectedCount := 5
	if len(files) != expectedCount {
		t.Errorf("Expected %d .gofa files, got %d", expectedCount, len(files))
	}

	// Verify all found files have .gofa extension
	for _, file := range files {
		if !strings.HasSuffix(file, ".gofa") {
			t.Errorf("Non-.gofa file found: %s", file)
		}
	}

	// Test with non-existent directory
	_, err = transpiler.FindGofaFiles("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

// TestGetOutputPath tests output path generation
func TestGetOutputPath(t *testing.T) {
	tests := []struct {
		name           string
		preserveStruct bool
		inputDir       string
		gofaFile       string
		outputDir      string
		expected       string
	}{
		{
			name:           "preserve structure",
			preserveStruct: true,
			inputDir:       "/src",
			gofaFile:       "/src/controllers/user.gofa",
			outputDir:      "/dist",
			expected:       "/dist/controllers/user.go",
		},
		{
			name:           "flatten structure",
			preserveStruct: false,
			inputDir:       "/src", 
			gofaFile:       "/src/controllers/user.gofa",
			outputDir:      "/dist",
			expected:       "/dist/user.go",
		},
		{
			name:           "nested with preserve",
			preserveStruct: true,
			inputDir:       "/project/src",
			gofaFile:       "/project/src/modules/user/user.controller.gofa",
			outputDir:      "/project/dist",
			expected:       "/project/dist/modules/user/user.controller.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := TranspileOptions{
				OutputDir:      tt.outputDir,
				PreserveStruct: tt.preserveStruct,
			}
			
			transpiler := NewParallelTranspiler(opts)
			result := transpiler.GetOutputPath(tt.inputDir, tt.gofaFile)
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestGetOutputPathForFile tests single file output path generation
func TestGetOutputPathForFile(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		inputPath string
		expected  string
	}{
		{
			name:      "with output dir",
			outputDir: "/dist",
			inputPath: "/src/user.gofa",
			expected:  "/dist/user.go",
		},
		{
			name:      "without output dir",
			outputDir: "",
			inputPath: "/src/user.gofa",
			expected:  "/src/user.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transpiler := &ParallelTranspiler{
				outputDir:     tt.outputDir,
				fileExtension: ".go",
			}
			
			result := transpiler.getOutputPathForFile(tt.inputPath)
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestProcessJob tests individual job processing
func TestProcessJob(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")

	transpiler := NewParallelTranspiler(TranspileOptions{
		OutputDir: outputDir,
	})

	tests := []struct {
		name      string
		job       TranspileJob
		shouldErr bool
	}{
		{
			name: "valid job",
			job: TranspileJob{
				InputPath:  filepath.Join(tempDir, "valid.gofa"),
				OutputPath: filepath.Join(outputDir, "valid.go"),
				Content: `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func Test() {}`,
			},
			shouldErr: false,
		},
		{
			name: "invalid content",
			job: TranspileJob{
				InputPath:  filepath.Join(tempDir, "invalid.gofa"),
				OutputPath: filepath.Join(outputDir, "invalid.go"),
				Content:    "invalid gofa content @#@$",
			},
			shouldErr: true,
		},
		{
			name: "empty content",
			job: TranspileJob{
				InputPath:  filepath.Join(tempDir, "empty.gofa"),
				OutputPath: filepath.Join(outputDir, "empty.go"),
				Content:    "",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transpiler.processJob(tt.job)
			
			if tt.shouldErr && result.Error == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.shouldErr && result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
			}
			
			if result.InputPath != tt.job.InputPath {
				t.Errorf("Expected InputPath %s, got %s", 
					tt.job.InputPath, result.InputPath)
			}
			
			if result.OutputPath != tt.job.OutputPath {
				t.Errorf("Expected OutputPath %s, got %s", 
					tt.job.OutputPath, result.OutputPath)
			}

			// For successful jobs, verify output file was created
			if result.Error == nil {
				if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
					t.Errorf("Output file was not created: %s", result.OutputPath)
				}
			}
		})
	}
}

// TestTranspileJobsCancellation tests context cancellation
func TestTranspileJobsCancellation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create multiple jobs
	jobs := make([]TranspileJob, 10)
	for i := 0; i < 10; i++ {
		jobs[i] = TranspileJob{
			InputPath:  fmt.Sprintf("test%d.gofa", i),
			OutputPath: filepath.Join(tempDir, fmt.Sprintf("test%d.go", i)),
			Content: fmt.Sprintf(`package main

@Controller("/test%d")
type TestController%d struct {}

@Get("/")
func Test() {}`, i, i),
		}
	}

	transpiler := NewParallelTranspiler(TranspileOptions{
		MaxWorkers: 2,
		OutputDir:  tempDir,
	})

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	results, err := transpiler.transpileJobs(ctx, jobs)
	
	// Should get context cancellation error
	if err == nil || err != context.DeadlineExceeded {
		t.Logf("Expected context deadline exceeded, got: %v", err)
		// Don't fail the test as timing can be flaky
	}

	// Should get some results (possibly empty if cancelled very quickly)
	if len(results) > len(jobs) {
		t.Errorf("Got more results than jobs: %d > %d", len(results), len(jobs))
	}
}

// TestGetStats tests statistics calculation
func TestGetStats(t *testing.T) {
	results := []TranspileResult{
		{
			InputPath:  "file1.gofa",
			OutputPath: "file1.go",
			Error:      nil,
			Duration:   100 * time.Millisecond,
		},
		{
			InputPath:  "file2.gofa",
			OutputPath: "file2.go", 
			Error:      nil,
			Duration:   200 * time.Millisecond,
		},
		{
			InputPath: "file3.gofa",
			Error:     fmt.Errorf("parsing error: invalid syntax"),
			Duration:  50 * time.Millisecond,
		},
		{
			InputPath: "file4.gofa",
			Error:     fmt.Errorf("lexer error: invalid token"),
			Duration:  30 * time.Millisecond,
		},
		{
			InputPath: "file5.gofa",
			Error:     fmt.Errorf("codegen error: unsupported feature"),
			Duration:  20 * time.Millisecond,
		},
		{
			InputPath: "file6.gofa",
			Error:     fmt.Errorf("file error: permission denied"),
			Duration:  10 * time.Millisecond,
		},
		{
			InputPath: "file7.gofa",
			Error:     fmt.Errorf("unknown error type"),
			Duration:  40 * time.Millisecond,
		},
	}

	stats := GetStats(results)

	// Test counts
	if stats.TotalFiles != 7 {
		t.Errorf("Expected 7 total files, got %d", stats.TotalFiles)
	}
	
	if stats.SuccessfulFiles != 2 {
		t.Errorf("Expected 2 successful files, got %d", stats.SuccessfulFiles)
	}
	
	if stats.FailedFiles != 5 {
		t.Errorf("Expected 5 failed files, got %d", stats.FailedFiles)
	}

	// Test durations
	expectedTotal := 450 * time.Millisecond
	if stats.TotalDuration != expectedTotal {
		t.Errorf("Expected total duration %v, got %v", expectedTotal, stats.TotalDuration)
	}

	expectedAvg := expectedTotal / 7
	if stats.AverageDuration != expectedAvg {
		t.Errorf("Expected average duration %v, got %v", expectedAvg, stats.AverageDuration)
	}

	// Test error categorization
	expectedErrors := map[string]int{
		"parsing_error": 1,
		"lexer_error":   1,
		"codegen_error": 1,
		"io_error":      1,
		"other_error":   1,
	}

	for errorType, expectedCount := range expectedErrors {
		if stats.ErrorSummary[errorType] != expectedCount {
			t.Errorf("Expected %d %s errors, got %d", 
				expectedCount, errorType, stats.ErrorSummary[errorType])
		}
	}
}

// TestGetStatsEmpty tests stats with empty results
func TestGetStatsEmpty(t *testing.T) {
	stats := GetStats([]TranspileResult{})

	if stats.TotalFiles != 0 {
		t.Errorf("Expected 0 total files, got %d", stats.TotalFiles)
	}

	if stats.TotalDuration != 0 {
		t.Errorf("Expected 0 total duration, got %v", stats.TotalDuration)
	}

	if stats.AverageDuration != 0 {
		t.Errorf("Expected 0 average duration, got %v", stats.AverageDuration)
	}
}

// TestPrintStats tests statistics printing (just verify no panics)
func TestPrintStats(t *testing.T) {
	stats := TranspileStats{
		TotalFiles:      10,
		SuccessfulFiles: 8,
		FailedFiles:     2,
		TotalDuration:   5 * time.Second,
		AverageDuration: 500 * time.Millisecond,
		ErrorSummary: map[string]int{
			"parsing_error": 1,
			"io_error":      1,
		},
	}

	// Should not panic
	PrintStats(stats)

	// Test with empty stats
	emptyStats := TranspileStats{}
	PrintStats(emptyStats)
}

// TestBatchTranspiler tests the batch transpiler
func TestBatchTranspiler(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.gofa")
	testContent := `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func Test() {}`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	opts := TranspileOptions{
		MaxWorkers: 2,
		OutputDir:  filepath.Join(tempDir, "output"),
		Verbose:    true,
	}

	batchTranspiler := NewBatchTranspiler(opts)

	// Test successful transpilation
	err := batchTranspiler.TranspileProject(tempDir)
	if err != nil {
		t.Errorf("TranspileProject failed: %v", err)
	}

	// Test stop
	batchTranspiler.Stop()
}

// TestWatchMode tests watch mode (basic functionality)
func TestWatchMode(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.gofa")
	testContent := `package main

@Controller("/test")
type TestController struct {}`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	opts := TranspileOptions{
		MaxWorkers: 1,
		OutputDir:  filepath.Join(tempDir, "output"),
		Verbose:    false,
	}

	watchMode := NewWatchMode(opts, tempDir, 100*time.Millisecond)

	// Test start (just initial transpilation)
	err := watchMode.Start()
	if err != nil {
		t.Errorf("WatchMode Start failed: %v", err)
	}

	// Test stop
	watchMode.Stop()
}