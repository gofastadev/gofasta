package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "1.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "gofasta",
		Short: "Gofasta CLI - Enterprise Go Framework Tooling",
		Long: `Gofasta CLI provides code generation, project scaffolding, and development tools
for the Gofasta enterprise framework.

Features:
- Project scaffolding with enterprise patterns
- Code generation for modules, controllers, services
- Database migration management
- Development server with hot reload
- Build optimization and deployment tools`,
		Version: Version,
	}

	// Add subcommands
	rootCmd.AddCommand(newCmd())
	rootCmd.AddCommand(generateCmd())
	rootCmd.AddCommand(migrationCmd())
	rootCmd.AddCommand(devCmd())
	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(testCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Gofasta CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Gofasta CLI v%s\n", Version)
		},
	}
}
