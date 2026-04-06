// Package testdb provides test database setup helpers using testcontainers.
package testdb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupTestDB spins up a PostgreSQL container, runs migrations, and returns
// a connected *gorm.DB. The container is automatically cleaned up when the test finishes.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := gorm.Open(gormpg.Open(connStr), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// RunMigrations applies the base SQL migrations to the test database.
func RunMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS citext;`,
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = now();
			RETURN NEW;
		END;
		$$ language 'plpgsql';`,
		`CREATE OR REPLACE FUNCTION prevent_delete_non_deletable()
		RETURNS TRIGGER AS $$
		BEGIN
			IF OLD.is_deletable = false THEN
				RAISE EXCEPTION 'Cannot delete non-deletable record';
			END IF;
			RETURN OLD;
		END;
		$$ language 'plpgsql';`,
		`CREATE OR REPLACE FUNCTION increment_record_version()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.record_version = OLD.record_version + 1;
			RETURN NEW;
		END;
		$$ language 'plpgsql';`,
	}

	for _, migration := range migrations {
		if _, err := sqlDB.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
