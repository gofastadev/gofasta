// Package testdb provides test database setup helpers using testcontainers.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// postgresContainer is the subset of *postgres.PostgresContainer this
// package depends on. Captured as an interface so tests can substitute
// an in-memory fake; production passes the real container returned by
// postgres.Run.
type postgresContainer interface {
	Terminate(ctx context.Context, opts ...testcontainers.TerminateOption) error
	ConnectionString(ctx context.Context, args ...string) (string, error)
}

// Package-level seams over the upstream constructors. Production
// assigns the real functions at init; tests reassign + restore via
// t.Cleanup so the unreachable error branches in setupTestDB and
// RunMigrations can be exercised without spinning up Docker.
var (
	postgresRunFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return postgres.Run(ctx, image, opts...)
	}
	gormOpenFn      = gorm.Open
	runMigrationsFn = RunMigrations
)

// SetupTestDB spins up a PostgreSQL container, runs migrations, and
// returns a connected *gorm.DB. The container is automatically
// cleaned up when the test finishes. Each failure mode causes
// t.Fatalf with a descriptive message.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, cleanup, err := setupTestDB(context.Background())
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Cleanup(cleanup)
	return db
}

// DefaultImage is the Postgres image used when none is configured.
const DefaultImage = "postgres:18-alpine"

// Image selects the container image SetupTestDB starts.
//
// Overridable because the default carries no extensions: a project storing
// embeddings needs pgvector/pgvector:pgNN, one using geometry needs PostGIS,
// and the extension has to exist in the image before any migration can CREATE
// it. Set it once in TestMain:
//
//	func TestMain(m *testing.M) {
//	    testdb.Image = "pgvector/pgvector:pg17"
//	    os.Exit(m.Run())
//	}
//
// Prefer the image the project actually deploys, so the test schema exercises
// the same server version as production.
var Image = DefaultImage

// setupTestDB is the error-returning core of SetupTestDB. Split out
// so unit tests can drive each failure branch without a real test
// harness — SetupTestDB itself is a one-liner shim that translates
// errors into t.Fatalf.
//
// Returns the connected *gorm.DB, a cleanup function the caller must
// register (via t.Cleanup or similar), and any error encountered.
func setupTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	image := Image
	if image == "" {
		image = DefaultImage
	}
	pgContainer, err := postgresRunFn(ctx,
		image,
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
		return nil, func() {}, fmt.Errorf("failed to start postgres container: %w", err)
	}

	cleanup := func() {
		// Terminate's error is logged in production via the real
		// SetupTestDB shim; setupTestDB itself returns nothing from
		// cleanup so callers don't have to handle a doubly-rare path.
		_ = pgContainer.Terminate(ctx)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := gormOpenFn(gormpg.Open(connStr), &gorm.Config{})
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to connect to test database: %w", err)
	}

	if err := runMigrationsFn(db); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, cleanup, nil
}

// defaultMigrations is the canonical baseline schema applied to every
// test database. Exported as a slice (not inline) so unit tests can
// reuse the production set or substitute their own to exercise
// per-migration error paths in runMigrationsOn.
// The function NAMES here must match the ones a scaffolded project's own
// migrations create, because that project's CREATE TRIGGER statements
// reference them by name. They previously did not — this package created
// update_updated_at_column / prevent_delete_non_deletable /
// increment_record_version while the generated migrations create
// update_updated_at_column_function /
// avoid_deleting_record_with_is_deletable_equal_to_false_function /
// increment_record_version_column_function — so applying a real migration set
// against a database prepared by this helper failed with "function ... does
// not exist". Keep these in step with
// cli/internal/skeleton/migrations/postgres/00000{2,3,4}_*.up.sql.
var defaultMigrations = []string{
	`CREATE EXTENSION IF NOT EXISTS citext;`,
	`CREATE OR REPLACE FUNCTION update_updated_at_column_function()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at := NOW();
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;`,
	`CREATE OR REPLACE FUNCTION avoid_deleting_record_with_is_deletable_equal_to_false_function()
	RETURNS TRIGGER AS $$
	BEGIN
		IF OLD.is_deletable = false THEN
			RAISE EXCEPTION 'This record is not deletable';
		END IF;
		RETURN OLD;
	END;
	$$ LANGUAGE plpgsql;`,
	`CREATE OR REPLACE FUNCTION increment_record_version_column_function()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.record_version := OLD.record_version + 1;
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;`,
}

// RunMigrations applies the base SQL migrations to the test database.
func RunMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return runMigrationsOn(sqlDB, defaultMigrations)
}

// runMigrationsOn is the testable core of RunMigrations. Split out so
// unit tests can drive the per-migration exec-failure branch by
// injecting a deliberately-bad SQL statement against a real test
// database — without going through gorm.
func runMigrationsOn(sqlDB *sql.DB, migrations []string) error {
	for _, migration := range migrations {
		if _, err := sqlDB.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
