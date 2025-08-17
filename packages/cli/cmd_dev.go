package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

func devCmd() *cobra.Command {
	var port int
	var watchPaths []string
	var excludePaths []string
	var buildCmd string

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start development server with hot reload",
		Long: `Start the development server with automatic reloading when files change.

Features:
- Hot reload on file changes
- Automatic build and restart
- Live reload for web assets
- Environment variable loading`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startDevServer(port, watchPaths, excludePaths, buildCmd)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
	cmd.Flags().StringSliceVarP(&watchPaths, "watch", "w", []string{"."}, "Paths to watch for changes")
	cmd.Flags().StringSliceVar(&excludePaths, "exclude", []string{"tmp", "dist", "node_modules", ".git"}, "Paths to exclude from watching")
	cmd.Flags().StringVar(&buildCmd, "build", "go build -o tmp/main", "Build command")

	return cmd
}

func startDevServer(port int, watchPaths, excludePaths []string, buildCommand string) error {
	fmt.Println("🚀 Starting Gofasta development server...")
	fmt.Printf("   Port: %d\n", port)
	fmt.Printf("   Watching: %v\n", watchPaths)
	fmt.Printf("   Build: %s\n", buildCommand)
	fmt.Println()

	// Create tmp directory for builds
	if err := os.MkdirAll("tmp", 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	// Initialize file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Add watch paths
	for _, path := range watchPaths {
		if err := addWatchPath(watcher, path, excludePaths); err != nil {
			log.Printf("Warning: Failed to watch %s: %v", path, err)
		}
	}

	// Channel to control application restarts
	restartChan := make(chan bool, 1)
	stopChan := make(chan bool, 1)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the application
	go runApplication(buildCommand, restartChan, stopChan)

	// Initial build and start
	restartChan <- true

	// Watch for file changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if shouldReload(event, excludePaths) {
				fmt.Printf("📝 File changed: %s\n", event.Name)
				select {
				case restartChan <- true:
				default:
					// Non-blocking send to avoid duplicate restarts
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)

		case <-sigChan:
			fmt.Println("\n🛑 Shutting down development server...")
			stopChan <- true
			return nil
		}
	}
}

func runApplication(buildCommand string, restartChan, stopChan chan bool) {
	var currentProcess *os.Process

	for {
		select {
		case <-restartChan:
			// Stop current process if running
			if currentProcess != nil {
				fmt.Println("🔄 Stopping current process...")
				if err := currentProcess.Kill(); err != nil {
					log.Printf("Warning: Failed to kill process: %v", err)
				}
				currentProcess.Wait()
			}

			// Build the application
			fmt.Println("🔨 Building application...")
			if err := buildApp(buildCommand); err != nil {
				fmt.Printf("❌ Build failed: %v\n", err)
				continue
			}

			// Start the new process
			fmt.Println("▶️  Starting application...")
			cmd := exec.Command("./tmp/main")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = os.Environ()

			if err := cmd.Start(); err != nil {
				fmt.Printf("❌ Failed to start process: %v\n", err)
				continue
			}

			currentProcess = cmd.Process
			fmt.Printf("✅ Application started (PID: %d)\n", currentProcess.Pid)

			// Wait for process in background
			go func() {
				cmd.Wait()
			}()

		case <-stopChan:
			if currentProcess != nil {
				currentProcess.Kill()
				currentProcess.Wait()
			}
			return
		}
	}
}

func buildApp(buildCommand string) error {
	args := parseCommand(buildCommand)
	if len(args) == 0 {
		return fmt.Errorf("invalid build command")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func addWatchPath(watcher *fsnotify.Watcher, root string, excludePaths []string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded paths
		for _, exclude := range excludePaths {
			if matched, _ := filepath.Match(exclude, info.Name()); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Only watch directories
		if info.IsDir() {
			return watcher.Add(path)
		}

		return nil
	})
}

func shouldReload(event fsnotify.Event, excludePaths []string) bool {
	// Only trigger on write events
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return false
	}

	// Check file extension (only Go files and config files)
	ext := filepath.Ext(event.Name)
	if ext != ".go" && ext != ".env" && ext != ".yaml" && ext != ".yml" && ext != ".json" {
		return false
	}

	// Check if path is excluded
	for _, exclude := range excludePaths {
		if matched, _ := filepath.Match(exclude, filepath.Base(event.Name)); matched {
			return false
		}
	}

	return true
}

func parseCommand(command string) []string {
	// Simple command parsing - in production use a proper shell parser
	return []string{"sh", "-c", command}
}
