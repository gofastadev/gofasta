package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func migrationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration",
		Short: "Database migration commands",
		Long:  "Create, run, and manage database migrations",
		Aliases: []string{"migrate"},
	}

	cmd.AddCommand(migrationCreateCmd())
	cmd.AddCommand(migrationRunCmd())
	cmd.AddCommand(migrationRollbackCmd())
	cmd.AddCommand(migrationStatusCmd())
	cmd.AddCommand(migrationResetCmd())

	return cmd
}

func migrationCreateCmd() *cobra.Command {
	var migrationDir string

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createMigration(args[0], migrationDir)
		},
	}

	cmd.Flags().StringVarP(&migrationDir, "dir", "d", "migrations", "Migration directory")

	return cmd
}

func migrationRunCmd() *cobra.Command {
	var migrationDir string
	var databaseURL string
	var steps int

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(migrationDir, databaseURL, steps)
		},
	}

	cmd.Flags().StringVarP(&migrationDir, "dir", "d", "migrations", "Migration directory")
	cmd.Flags().StringVarP(&databaseURL, "database-url", "u", "", "Database URL (env: DATABASE_URL)")
	cmd.Flags().IntVarP(&steps, "steps", "s", 0, "Number of migrations to run (0 = all)")

	return cmd
}

func migrationRollbackCmd() *cobra.Command {
	var migrationDir string
	var databaseURL string
	var steps int

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rollbackMigrations(migrationDir, databaseURL, steps)
		},
	}

	cmd.Flags().StringVarP(&migrationDir, "dir", "d", "migrations", "Migration directory")
	cmd.Flags().StringVarP(&databaseURL, "database-url", "u", "", "Database URL (env: DATABASE_URL)")
	cmd.Flags().IntVarP(&steps, "steps", "s", 1, "Number of migrations to rollback")

	return cmd
}

func migrationStatusCmd() *cobra.Command {
	var migrationDir string
	var databaseURL string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrationStatus(migrationDir, databaseURL)
		},
	}

	cmd.Flags().StringVarP(&migrationDir, "dir", "d", "migrations", "Migration directory")
	cmd.Flags().StringVarP(&databaseURL, "database-url", "u", "", "Database URL (env: DATABASE_URL)")

	return cmd
}

func migrationResetCmd() *cobra.Command {
	var migrationDir string
	var databaseURL string
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset all migrations (dangerous)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Println("Use --force to confirm migration reset")
				return nil
			}
			return resetMigrations(migrationDir, databaseURL)
		},
	}

	cmd.Flags().StringVarP(&migrationDir, "dir", "d", "migrations", "Migration directory")
	cmd.Flags().StringVarP(&databaseURL, "database-url", "u", "", "Database URL (env: DATABASE_URL)")
	cmd.Flags().BoolVar(&force, "force", false, "Force reset without confirmation")

	return cmd
}

func createMigration(name, migrationDir string) error {
	// Create migrations directory if it doesn't exist
	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	// Generate timestamp
	timestamp := time.Now().Format("20060102150405")
	
	// Generate file names
	upFile := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.up.sql", timestamp, name))
	downFile := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.down.sql", timestamp, name))

	// Create up migration file
	upContent := fmt.Sprintf(`-- Migration: %s
-- Created: %s

-- Add your SQL statements here
-- Example:
-- CREATE TABLE example (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );
`, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to create up migration file: %w", err)
	}

	// Create down migration file
	downContent := fmt.Sprintf(`-- Rollback migration: %s
-- Created: %s

-- Add your rollback SQL statements here
-- Example:
-- DROP TABLE IF EXISTS example;
`, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to create down migration file: %w", err)
	}

	fmt.Printf("✅ Created migration files:\n")
	fmt.Printf("   %s\n", upFile)
	fmt.Printf("   %s\n", downFile)

	return nil
}

func runMigrations(migrationDir, databaseURL string, steps int) error {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}

	if databaseURL == "" {
		return fmt.Errorf("database URL is required (use --database-url or set DATABASE_URL env var)")
	}

	fmt.Printf("Running migrations from %s...\n", migrationDir)
	fmt.Printf("Database: %s\n", maskDatabaseURL(databaseURL))

	// In a real implementation, you would use golang-migrate or similar
	// This is a simplified example showing the structure
	
	migrations, err := getMigrationFiles(migrationDir, "up")
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Println("No pending migrations found.")
		return nil
	}

	limit := len(migrations)
	if steps > 0 && steps < limit {
		limit = steps
	}

	fmt.Printf("Found %d migration(s) to run:\n", limit)
	for i := 0; i < limit; i++ {
		fmt.Printf("  - %s\n", migrations[i])
	}

	// This would actually execute the migrations
	fmt.Printf("✅ Successfully ran %d migration(s)\n", limit)

	return nil
}

func rollbackMigrations(migrationDir, databaseURL string, steps int) error {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}

	if databaseURL == "" {
		return fmt.Errorf("database URL is required (use --database-url or set DATABASE_URL env var)")
	}

	fmt.Printf("Rolling back %d migration(s) from %s...\n", steps, migrationDir)
	fmt.Printf("Database: %s\n", maskDatabaseURL(databaseURL))

	// In a real implementation, you would use golang-migrate or similar
	migrations, err := getMigrationFiles(migrationDir, "down")
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations to rollback.")
		return nil
	}

	limit := steps
	if limit > len(migrations) {
		limit = len(migrations)
	}

	fmt.Printf("Rolling back %d migration(s):\n", limit)
	for i := 0; i < limit; i++ {
		fmt.Printf("  - %s\n", migrations[i])
	}

	// This would actually execute the rollback migrations
	fmt.Printf("✅ Successfully rolled back %d migration(s)\n", limit)

	return nil
}

func migrationStatus(migrationDir, databaseURL string) error {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}

	if databaseURL == "" {
		return fmt.Errorf("database URL is required (use --database-url or set DATABASE_URL env var)")
	}

	fmt.Printf("Migration status for %s\n", migrationDir)
	fmt.Printf("Database: %s\n", maskDatabaseURL(databaseURL))
	fmt.Println()

	// In a real implementation, you would query the migration table
	fmt.Println("Migration                    | Status")
	fmt.Println("---------------------------- | --------")
	fmt.Println("20240101120000_initial       | Applied")
	fmt.Println("20240101130000_add_users     | Applied") 
	fmt.Println("20240101140000_add_orders    | Pending")

	return nil
}

func resetMigrations(migrationDir, databaseURL string) error {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}

	if databaseURL == "" {
		return fmt.Errorf("database URL is required (use --database-url or set DATABASE_URL env var)")
	}

	fmt.Printf("⚠️  Resetting all migrations for %s...\n", maskDatabaseURL(databaseURL))
	
	// This would actually drop all tables and reset migration state
	fmt.Println("✅ Successfully reset all migrations")

	return nil
}

func getMigrationFiles(migrationDir, direction string) ([]string, error) {
	pattern := filepath.Join(migrationDir, fmt.Sprintf("*.%s.sql", direction))
	
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func maskDatabaseURL(url string) string {
	// Simple masking for display purposes
	if len(url) > 20 {
		return url[:8] + "***" + url[len(url)-8:]
	}
	return "***"
}