package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func buildCmd() *cobra.Command {
	var output string
	var platform string
	var arch string
	var ldflags string
	var tags string
	var optimize bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the Gofasta application",
		Long: `Build the Gofasta application with optimizations and cross-platform support.

Features:
- Cross-platform compilation
- Build optimizations
- Custom build tags
- Embedded assets
- Binary size optimization`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildApplication(BuildConfig{
				Output:   output,
				Platform: platform,
				Arch:     arch,
				LDFlags:  ldflags,
				Tags:     tags,
				Optimize: optimize,
				Verbose:  verbose,
			})
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output binary name")
	cmd.Flags().StringVar(&platform, "platform", runtime.GOOS, "Target platform (linux, darwin, windows)")
	cmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Target architecture (amd64, arm64)")
	cmd.Flags().StringVar(&ldflags, "ldflags", "", "Linker flags")
	cmd.Flags().StringVar(&tags, "tags", "", "Build tags")
	cmd.Flags().BoolVar(&optimize, "optimize", true, "Enable build optimizations")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	return cmd
}

type BuildConfig struct {
	Output   string
	Platform string
	Arch     string
	LDFlags  string
	Tags     string
	Optimize bool
	Verbose  bool
}

func buildApplication(config BuildConfig) error {
	fmt.Println("🔨 Building Gofasta application...")

	// Determine output name
	if config.Output == "" {
		config.Output = "app"
		if config.Platform == "windows" {
			config.Output += ".exe"
		}
	}

	// Create dist directory
	distDir := "dist"
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	outputPath := filepath.Join(distDir, config.Output)

	// Build ldflags
	ldflags := buildLDFlags(config.LDFlags, config.Optimize)

	// Prepare build command
	args := []string{"build"}

	if config.Verbose {
		args = append(args, "-v")
	}

	if config.Tags != "" {
		args = append(args, "-tags", config.Tags)
	}

	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}

	args = append(args, "-o", outputPath)

	// Find main package
	mainPackage, err := findMainPackage()
	if err != nil {
		return fmt.Errorf("failed to find main package: %w", err)
	}

	args = append(args, mainPackage)

	// Set environment variables for cross-compilation
	env := os.Environ()
	env = append(env, "GOOS="+config.Platform)
	env = append(env, "GOARCH="+config.Arch)

	if config.Optimize {
		env = append(env, "CGO_ENABLED=0") // Disable CGO for better optimization
	}

	fmt.Printf("📦 Target: %s/%s\n", config.Platform, config.Arch)
	fmt.Printf("📁 Output: %s\n", outputPath)

	if config.Verbose {
		fmt.Printf("🔧 Command: go %s\n", strings.Join(args, " "))
	}

	// Execute build
	cmd := exec.Command("go", args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Get binary size
	if stat, err := os.Stat(outputPath); err == nil {
		size := formatBytes(stat.Size())
		fmt.Printf("✅ Build completed successfully!\n")
		fmt.Printf("📏 Binary size: %s\n", size)
	}

	// Show next steps
	fmt.Println("\n🚀 Next steps:")
	fmt.Printf("   ./%s\n", outputPath)

	if config.Platform != runtime.GOOS {
		fmt.Printf("\n💡 Cross-compiled for %s/%s\n", config.Platform, config.Arch)
	}

	return nil
}

func buildLDFlags(customFlags string, optimize bool) string {
	var flags []string

	// Add version information
	flags = append(flags, "-X main.Version="+Version)
	flags = append(flags, "-X main.BuildTime="+getCurrentTime())

	// Add optimization flags
	if optimize {
		flags = append(flags, "-s", "-w") // Strip debug info and symbol table
	}

	// Add custom flags
	if customFlags != "" {
		flags = append(flags, customFlags)
	}

	return strings.Join(flags, " ")
}

func findMainPackage() (string, error) {
	// Look for main.go in common locations
	candidates := []string{
		".",
		"cmd",
		"cmd/main.go",
		"main.go",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			// Check if it contains package main
			if isMainPackage(candidate) {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("no main package found")
}

func isMainPackage(path string) bool {
	// Simple check - in production, you'd parse the Go file
	if filepath.Ext(path) == ".go" {
		content, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		return strings.Contains(string(content), "package main")
	}

	// Check if directory contains main.go
	mainFile := filepath.Join(path, "main.go")
	if _, err := os.Stat(mainFile); err == nil {
		return isMainPackage(mainFile)
	}

	return false
}

func getCurrentTime() string {
	return "2024-01-01T00:00:00Z" // Simplified for example
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
