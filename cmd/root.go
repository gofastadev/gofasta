package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gofasta",
	Short: "Gofasta - A scalable Go web framework",
	Long:  "Gofasta is a production-grade Go web framework with GraphQL + REST, DI, middleware, and more.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
