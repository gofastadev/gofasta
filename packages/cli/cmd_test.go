package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func testCmd() *cobra.Command {
	var coverage bool
	var verbose bool
	var race bool
	var benchmarks bool
	var packages []string
	var outputFormat string
	var tags string

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests with Gofasta testing utilities",
		Long: `Run tests with Gofasta's enhanced testing capabilities.

Features:
- Dependency injection testing
- Database test utilities
- HTTP endpoint testing
- Coverage reporting
- Parallel test execution`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(TestConfig{
				Coverage:     coverage,
				Verbose:      verbose,
				Race:         race,
				Benchmarks:   benchmarks,
				Packages:     packages,
				OutputFormat: outputFormat,
				Tags:         tags,
			})
		},
	}

	cmd.Flags().BoolVarP(&coverage, "coverage", "c", false, "Generate coverage report")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose test output")
	cmd.Flags().BoolVar(&race, "race", false, "Enable race detection")
	cmd.Flags().BoolVarP(&benchmarks, "bench", "b", false, "Run benchmarks")
	cmd.Flags().StringSliceVarP(&packages, "packages", "p", []string{"./..."}, "Packages to test")
	cmd.Flags().StringVar(&outputFormat, "format", "default", "Output format (default, json, junit)")
	cmd.Flags().StringVar(&tags, "tags", "", "Build tags for tests")

	return cmd
}

type TestConfig struct {
	Coverage     bool
	Verbose      bool
	Race         bool
	Benchmarks   bool
	Packages     []string
	OutputFormat string
	Tags         string
}

func runTests(config TestConfig) error {
	fmt.Println("🧪 Running Gofasta tests...")

	// Create test output directory
	if err := os.MkdirAll("test-results", 0755); err != nil {
		return fmt.Errorf("failed to create test-results directory: %w", err)
	}

	if config.Benchmarks {
		return runBenchmarks(config)
	}

	return runUnitTests(config)
}

func runUnitTests(config TestConfig) error {
	args := []string{"test"}

	// Add verbose flag
	if config.Verbose {
		args = append(args, "-v")
	}

	// Add race detection
	if config.Race {
		args = append(args, "-race")
	}

	// Add build tags
	if config.Tags != "" {
		args = append(args, "-tags", config.Tags)
	}

	// Add coverage
	if config.Coverage {
		args = append(args, "-coverprofile=test-results/coverage.out")
		args = append(args, "-covermode=atomic")
	}

	// Add output format
	switch config.OutputFormat {
	case "json":
		args = append(args, "-json")
	case "junit":
		// This would require additional tooling like go-junit-report
		fmt.Println("💡 JUnit format requires go-junit-report tool")
	}

	// Add packages
	args = append(args, config.Packages...)

	fmt.Printf("🔧 Command: go %s\n", strings.Join(args, " "))

	// Execute tests
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Tests failed: %v\n", err)
		return err
	}

	fmt.Println("✅ All tests passed!")

	// Generate coverage report if requested
	if config.Coverage {
		if err := generateCoverageReport(); err != nil {
			fmt.Printf("⚠️  Failed to generate coverage report: %v\n", err)
		}
	}

	return nil
}

func runBenchmarks(config TestConfig) error {
	fmt.Println("📊 Running benchmarks...")

	args := []string{"test", "-bench=."}

	if config.Verbose {
		args = append(args, "-v")
	}

	if config.Tags != "" {
		args = append(args, "-tags", config.Tags)
	}

	// Benchmark-specific flags
	args = append(args, "-benchmem")
	args = append(args, "-benchtime=3s")

	// Add packages
	args = append(args, config.Packages...)

	fmt.Printf("🔧 Command: go %s\n", strings.Join(args, " "))

	// Execute benchmarks
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Benchmarks failed: %v\n", err)
		return err
	}

	fmt.Println("✅ Benchmarks completed!")
	return nil
}

func generateCoverageReport() error {
	coverageFile := "test-results/coverage.out"

	// Check if coverage file exists
	if _, err := os.Stat(coverageFile); os.IsNotExist(err) {
		return fmt.Errorf("coverage file not found: %s", coverageFile)
	}

	// Generate HTML coverage report
	htmlFile := "test-results/coverage.html"
	cmd := exec.Command("go", "tool", "cover", "-html="+coverageFile, "-o", htmlFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML coverage report: %w", err)
	}

	// Generate text coverage summary
	cmd = exec.Command("go", "tool", "cover", "-func="+coverageFile)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate coverage summary: %w", err)
	}

	// Write summary to file
	summaryFile := "test-results/coverage.txt"
	if err := os.WriteFile(summaryFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write coverage summary: %w", err)
	}

	// Parse total coverage
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			fmt.Printf("📊 Coverage: %s\n", strings.TrimSpace(line))
			break
		}
	}

	fmt.Printf("📄 Coverage reports generated:\n")
	fmt.Printf("   HTML: %s\n", htmlFile)
	fmt.Printf("   Text: %s\n", summaryFile)

	// Open HTML report in browser (optional)
	absPath, _ := filepath.Abs(htmlFile)
	fmt.Printf("💡 Open in browser: file://%s\n", absPath)

	return nil
}
