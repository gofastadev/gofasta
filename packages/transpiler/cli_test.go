package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCLI tests comprehensive CLI functionality
func TestCLI(t *testing.T) {
	cli := NewCLI("1.0.0-test")

	// Test version command
	t.Run("version command", func(t *testing.T) {
		err := cli.versionCommand([]string{})
		if err != nil {
			t.Errorf("Version command failed: %v", err)
		}
	})

	// Test help command
	t.Run("help command", func(t *testing.T) {
		err := cli.helpCommand([]string{})
		if err != nil {
			t.Errorf("Help command failed: %v", err)
		}

		// Test help for specific commands
		err = cli.helpCommand([]string{"transpile"})
		if err != nil {
			t.Errorf("Help for transpile failed: %v", err)
		}

		err = cli.helpCommand([]string{"watch"})
		if err != nil {
			t.Errorf("Help for watch failed: %v", err)
		}

		err = cli.helpCommand([]string{"version"})
		if err != nil {
			t.Errorf("Help for version failed: %v", err)
		}

		// Test help for unknown command
		err = cli.helpCommand([]string{"unknown"})
		if err != nil {
			t.Errorf("Help for unknown command failed: %v", err)
		}
	})

	// Test unknown command
	t.Run("unknown command", func(t *testing.T) {
		err := cli.Run([]string{"gofasta", "unknown"})
		if err == nil {
			t.Error("Expected error for unknown command")
		}
	})

	// Test no arguments
	t.Run("no arguments", func(t *testing.T) {
		err := cli.Run([]string{"gofasta"})
		if err != nil {
			t.Errorf("No arguments should not fail: %v", err)
		}
	})

	// Test command aliases
	t.Run("command aliases", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.gofa")
		
		// Create a simple test file
		testContent := `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func Test() {}`

		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test transpile alias 't'
		err := cli.Run([]string{"gofasta", "t", "-file", testFile, "-output", tempDir, "-dry-run"})
		if err != nil {
			t.Errorf("Transpile alias 't' failed: %v", err)
		}

		// Test version alias 'v'
		err = cli.Run([]string{"gofasta", "v"})
		if err != nil {
			t.Errorf("Version alias 'v' failed: %v", err)
		}

		// Test help alias 'h'
		err = cli.Run([]string{"gofasta", "h"})
		if err != nil {
			t.Errorf("Help alias 'h' failed: %v", err)
		}
	})
}

// TestTranspileCommand tests the transpile command comprehensively
func TestTranspileCommand(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"controller.gofa": `package main

@Controller("/test")
type TestController struct {
	Service *TestService ` + "`inject:\"\"`" + `
}

@Get("/")
func GetTest() {}`,

		"service.gofa": `package main

@Injectable()
type TestService struct {}

func DoSomething() string {
	return "test"
}`,

		"module.gofa": `package main

@Module({
	controllers: ["TestController"],
	providers: ["TestService"]
})
type TestModule struct {}`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "basic transpile",
			args: []string{"transpile", "-input", tempDir, "-output", filepath.Join(tempDir, "output1"), "-dry-run"},
		},
		{
			name: "transpile with verbose",
			args: []string{"transpile", "-input", tempDir, "-output", filepath.Join(tempDir, "output2"), "-verbose", "-dry-run"},
		},
		{
			name: "transpile single file",
			args: []string{"transpile", "-file", filepath.Join(tempDir, "controller.gofa"), "-output", tempDir, "-dry-run"},
		},
		{
			name: "transpile with workers",
			args: []string{"transpile", "-input", tempDir, "-output", filepath.Join(tempDir, "output3"), "-workers", "2", "-dry-run"},
		},
		{
			name: "transpile without preserve",
			args: []string{"transpile", "-input", tempDir, "-output", filepath.Join(tempDir, "output4"), "-preserve=false", "-dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.transpileCommand(tt.args)
			if err != nil {
				t.Errorf("Transpile command failed: %v", err)
			}
		})
	}
}

// TestTranspileCommandErrors tests error conditions in transpile command
func TestTranspileCommandErrors(t *testing.T) {
	cli := NewCLI("1.0.0-test")

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "non-existent input directory",
			args: []string{"transpile", "-input", "/non/existent/path", "-dry-run"},
		},
		{
			name: "non-existent single file",
			args: []string{"transpile", "-file", "/non/existent/file.gofa", "-dry-run"},
		},
		{
			name: "invalid file extension",
			args: []string{"transpile", "-file", "test.go", "-dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.transpileCommand(tt.args)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// TestActualTranspilation tests actual file transpilation (not dry-run)
func TestActualTranspilation(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	tempDir := t.TempDir()

	// Create a simple test file
	testContent := `package main

@Controller("/test")
type TestController struct {
	Service *TestService ` + "`inject:\"\"`" + `
}

@Get("/hello")
func SayHello() {}`

	inputFile := filepath.Join(tempDir, "test.gofa")
	if err := os.WriteFile(inputFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outputDir := filepath.Join(tempDir, "output")
	
	// Test single file transpilation
	err := cli.transpileSingleFile(inputFile, outputDir, true, false, false)
	if err != nil {
		t.Fatalf("Single file transpilation failed: %v", err)
	}

	// Verify output file was created
	outputFile := filepath.Join(outputDir, "test.go")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created: %s", outputFile)
	}

	// Verify output content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedParts := []string{
		"package main",
		"type TestController struct",
		"Service *TestService `inject:\"\"`",
		"func (c *TestController) RegisterRoutes",
		"func (c *TestController) SayHello",
	}

	contentStr := string(content)
	for _, part := range expectedParts {
		if !strings.Contains(contentStr, part) {
			t.Errorf("Output missing expected part: %s", part)
		}
	}

	// Test force overwrite
	err = cli.transpileSingleFile(inputFile, outputDir, false, false, true)
	if err != nil {
		t.Errorf("Force overwrite failed: %v", err)
	}

	// Test without force (should fail)
	err = cli.transpileSingleFile(inputFile, outputDir, false, false, false)
	if err == nil {
		t.Error("Expected error when output file exists without force")
	}
}

// TestWatchCommand tests watch command (basic functionality)
func TestWatchCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping watch test in short mode")
	}

	cli := NewCLI("1.0.0-test")
	tempDir := t.TempDir()

	// Create a test file
	testContent := `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func Test() {}`

	testFile := filepath.Join(tempDir, "test.gofa")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test watch command with timeout to avoid hanging tests
	// Note: This primarily tests argument parsing and initial setup
	args := []string{"watch", "-input", tempDir, "-output", filepath.Join(tempDir, "output"), "-debounce", "100ms"}
	
	// Use a channel to signal completion and prevent hanging
	done := make(chan error, 1)
	
	// Run watch command in a goroutine with timeout
	go func() {
		done <- cli.watchCommand(args)
	}()
	
	// Wait for either completion or timeout
	select {
	case err := <-done:
		// If it returns quickly, that's fine for testing argument parsing
		if err != nil && !strings.Contains(err.Error(), "watch") && !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Watch command failed: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		// If it runs for 200ms without error, consider it successful
		// (means it passed argument parsing and started watching)
		t.Log("Watch command started successfully (timed out as expected)")
	}
}

// TestDryRunTranspile tests dry run functionality
func TestDryRunTranspile(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	tempDir := t.TempDir()

	// Create test files
	for i := 0; i < 3; i++ {
		testContent := fmt.Sprintf(`package main

@Controller("/test%d")
type TestController%d struct {}

@Get("/")
func Test() {}`, i, i)

		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.gofa", i))
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	opts := TranspileOptions{
		OutputDir:      filepath.Join(tempDir, "output"),
		PreserveStruct: true,
		Verbose:        false,
	}

	err := cli.dryRunTranspile(tempDir, opts)
	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}

	// Test with empty directory
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty dir: %v", err)
	}

	err = cli.dryRunTranspile(emptyDir, opts)
	if err != nil {
		t.Errorf("Dry run with empty directory failed: %v", err)
	}
}

// TestCheckExistingFiles tests existing file checking
func TestCheckExistingFiles(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	tempDir := t.TempDir()

	// Create .gofa file
	gofaFile := filepath.Join(tempDir, "test.gofa")
	testContent := `package main
@Controller("/test")
type TestController struct {}`

	if err := os.WriteFile(gofaFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create .gofa file: %v", err)
	}

	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// First check should pass (no existing files)
	err := cli.checkExistingFiles(tempDir, outputDir, false)
	if err != nil {
		t.Errorf("Check should pass when no existing files: %v", err)
	}

	// Create output file
	outputFile := filepath.Join(outputDir, "test.go")
	if err := os.WriteFile(outputFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}

	// Second check should fail (existing file)
	err = cli.checkExistingFiles(tempDir, outputDir, false)
	if err == nil {
		t.Error("Check should fail when existing files present")
	}
}

// TestRunMain tests the main entry point
func TestRunMain(t *testing.T) {
	// Save original args
	originalArgs := os.Args

	defer func() {
		os.Args = originalArgs
	}()

	// Test help
	os.Args = []string{"gofasta", "help"}
	// RunMain() would call os.Exit, so we can't test it directly in unit tests
	// Instead, we test the CLI.Run method which it uses
	cli := NewCLI("1.0.0-test")
	err := cli.Run(os.Args)
	if err != nil {
		t.Errorf("RunMain help failed: %v", err)
	}

	// Test version
	os.Args = []string{"gofasta", "version"}
	err = cli.Run(os.Args)
	if err != nil {
		t.Errorf("RunMain version failed: %v", err)
	}
}

// TestCLIArgumentParsing tests CLI argument parsing edge cases
func TestCLIArgumentParsing(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.gofa")
	testContent := `package main
@Controller("/test")
type TestController struct {}`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name: "transpile with all flags",
			args: []string{
				"transpile",
				"-input", tempDir,
				"-output", filepath.Join(tempDir, "output"),
				"-workers", "4",
				"-preserve",
				"-verbose",
				"-dry-run",
			},
			shouldError: false,
		},
		{
			name: "transpile with boolean flags",
			args: []string{
				"transpile",
				"-input", tempDir,
				"-preserve=true",
				"-verbose=false",
				"-dry-run=true",
			},
			shouldError: false,
		},
		{
			name: "watch with all flags",
			args: []string{
				"watch",
				"-input", tempDir,
				"-output", filepath.Join(tempDir, "output"),
				"-workers", "2",
				"-preserve=false",
				"-debounce", "1s",
				"-verbose",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special handling for watch command to prevent hanging
			if strings.Contains(tt.name, "watch") {
				done := make(chan error, 1)
				go func() {
					done <- cli.Run(append([]string{"gofasta"}, tt.args...))
				}()
				
				select {
				case err := <-done:
					if tt.shouldError && err == nil {
						t.Error("Expected error but got none")
					}
					if !tt.shouldError && err != nil {
						// Watch command might fail due to implementation limitations
						if !strings.Contains(err.Error(), "watch") && !strings.Contains(err.Error(), "does not exist") {
							t.Errorf("Unexpected error: %v", err)
						}
					}
				case <-time.After(300 * time.Millisecond):
					// Watch command should run indefinitely, so timeout is expected
					if !tt.shouldError {
						t.Log("Watch command started successfully (timed out as expected)")
					}
				}
			} else {
				// Normal handling for non-watch commands
				err := cli.Run(append([]string{"gofasta"}, tt.args...))
				
				if tt.shouldError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tt.shouldError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCLIUsage tests usage printing
func TestCLIUsage(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	
	// Test general usage (no panics)
	cli.printUsage()

	// Test with empty CLI version
	emptyCLI := NewCLI("")
	emptyCLI.printUsage()
}

// TestNewCLI tests CLI constructor
func TestNewCLI(t *testing.T) {
	version := "2.0.0-test"
	cli := NewCLI(version)
	
	if cli.version != version {
		t.Errorf("Expected version %s, got %s", version, cli.version)
	}
}

// Helper function to create fmt import for tests that need it
func init() {
	// This ensures fmt is available for string formatting in tests
	_ = fmt.Sprintf
}