package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gofastadev/gofasta/configs"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start development server with hot reload, auto-migration",
	Long: `Start the gofasta development server on your host machine.
This command:
  1. Runs database migrations (if DB is reachable)
  2. Starts air for hot reload
  3. Rebuilds on every file change

Prerequisites: Go installed, database running (use 'docker compose up db -d' for Docker DB)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDev()
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}

func runDev() error {
	fmt.Println("Starting gofasta development server...")

	// Try running migrations (don't fail if DB is unreachable)
	fmt.Println("🗄  Running migrations...")
	cfg, err := configs.LoadConfig()
	if err == nil {
		dbURL := buildMigrationURL(cfg)
		migrateCmd := exec.Command("migrate", "-path", "db/migrations", "-database", dbURL, "up")
		migrateCmd.Stdout = os.Stdout
		migrateCmd.Stderr = os.Stderr
		if err := migrateCmd.Run(); err != nil {
			fmt.Println("   ⚠ Migrations skipped (database may not be running)")
		}
	}

	fmt.Println("\n🚀 Starting air (hot reload)...")
	fmt.Printf("   REST API:    http://localhost:%s\n", getPort(cfg))
	fmt.Printf("   GraphQL:     http://localhost:%s/graphql\n", getPort(cfg))
	fmt.Printf("   Playground:  http://localhost:%s/graphql-playground\n\n", getPort(cfg))

	// Start air
	airCmd := exec.Command("go", "tool", "air")
	airCmd.Stdout = os.Stdout
	airCmd.Stderr = os.Stderr
	airCmd.Stdin = os.Stdin

	// Forward signals so air shuts down cleanly
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if airCmd.Process != nil {
			airCmd.Process.Signal(os.Interrupt)
		}
	}()

	return airCmd.Run()
}

func getPort(cfg *configs.AppConfig) string {
	if cfg != nil && cfg.Server.Port != "" {
		return cfg.Server.Port
	}
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
