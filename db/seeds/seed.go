package seeds

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

var seeders []func(*gorm.DB) error

// Register adds a seed function to the registry.
func Register(fn func(*gorm.DB) error) {
	seeders = append(seeders, fn)
}

// RunAll executes all registered seed functions.
func RunAll(db *gorm.DB) error {
	slog.Info("running database seeds", "count", len(seeders))
	for i, fn := range seeders {
		if err := fn(db); err != nil {
			return fmt.Errorf("seed #%d failed: %w", i+1, err)
		}
	}
	slog.Info("all seeds completed")
	return nil
}

func init() {
	// Register example seed — remove or replace for your project
	Register(func(db *gorm.DB) error {
		slog.Info("example seed: no data to insert (customize db/seeds/seed.go)")
		return nil
	})
}
