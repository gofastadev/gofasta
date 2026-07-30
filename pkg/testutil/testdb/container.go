// Package testdb provides test database setup helpers using testcontainers.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormch "gorm.io/driver/clickhouse"
	gormmysql "gorm.io/driver/mysql"
	gormpg "gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
	gormmssql "gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// dbContainer is the subset of a testcontainers module container this
// package depends on. Captured as an interface so tests can substitute
// an in-memory fake; production passes the real container returned by
// the module's Run function.
type dbContainer interface {
	Terminate(ctx context.Context, opts ...testcontainers.TerminateOption) error
	ConnectionString(ctx context.Context, args ...string) (string, error)
}

// Package-level seams over the upstream constructors. Production
// assigns the real functions at init; tests reassign + restore via
// t.Cleanup so the unreachable error branches in setupTestDB and
// RunMigrations can be exercised without spinning up Docker.
var (
	postgresRunFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (dbContainer, error) {
		return postgres.Run(ctx, image, opts...)
	}
	mysqlRunFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (dbContainer, error) {
		return mysql.Run(ctx, image, opts...)
	}
	mssqlRunFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (dbContainer, error) {
		return mssql.Run(ctx, image, opts...)
	}
	clickhouseRunFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (dbContainer, error) {
		return clickhouse.Run(ctx, image, opts...)
	}
	gormOpenFn      = gorm.Open
	runMigrationsFn = RunMigrations
	osMkdirTempFn   = os.MkdirTemp
)

// SetupTestDB provisions a test database for the configured Driver,
// runs the driver's baseline migrations, and returns a connected
// *gorm.DB. Container-backed drivers are cleaned up when the test
// finishes; sqlite runs in-process against a temp file and needs no
// Docker. Each failure mode causes t.Fatalf with a descriptive message.
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

// Driver selects which database SetupTestDB provisions. Empty means
// "postgres". Recognized values: postgres, mysql, sqlserver,
// clickhouse (each starts a container) and sqlite (in-process temp
// file, no Docker). Set it once in TestMain, typically from the same
// source the app reads:
//
//	func TestMain(m *testing.M) {
//	    testdb.Driver = os.Getenv("MYAPP_DATABASE_DRIVER")
//	    os.Exit(m.Run())
//	}
var Driver string

// Image selects the container image SetupTestDB starts, overriding the
// driver's default.
//
// Overridable because the defaults carry no extensions: a project storing
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
var Image = ""

// defaultImageFor returns the per-driver container image used when
// Image is unset. Kept aligned with the compose manifests the CLI
// scaffolds per driver.
func defaultImageFor(driver string) string {
	switch driver {
	case "mysql":
		return "mysql:8.4"
	case "sqlserver":
		return "mcr.microsoft.com/mssql/server:2022-latest"
	case "clickhouse":
		return "clickhouse/clickhouse-server:24.8-alpine"
	default:
		return DefaultImage
	}
}

func driverOrDefault() string {
	if Driver == "" {
		return "postgres"
	}
	return Driver
}

// setupTestDB is the error-returning core of SetupTestDB. Split out
// so unit tests can drive each failure branch without a real test
// harness — SetupTestDB itself is a one-liner shim that translates
// errors into t.Fatalf.
//
// Returns the connected *gorm.DB, a cleanup function the caller must
// register (via t.Cleanup or similar), and any error encountered.
func setupTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	driver := driverOrDefault()
	if driver == "sqlite" {
		return setupSQLiteTestDB()
	}

	container, err := startContainer(ctx, driver)
	if err != nil {
		return nil, func() {}, err
	}

	cleanup := func() {
		// Terminate's error is logged in production via the real
		// SetupTestDB shim; setupTestDB itself returns nothing from
		// cleanup so callers don't have to handle a doubly-rare path.
		_ = container.Terminate(ctx)
	}

	connStr, err := connectionString(ctx, container, driver)
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := gormOpenFn(dialectorFor(driver, connStr), &gorm.Config{})
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

// setupSQLiteTestDB opens an in-process sqlite database backed by a
// temp file — no container, no Docker requirement.
func setupSQLiteTestDB() (*gorm.DB, func(), error) {
	dir, err := osMkdirTempFn("", "gofasta-testdb-*")
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create sqlite temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	db, err := gormOpenFn(gormsqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{})
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to open sqlite test database: %w", err)
	}
	if err := runMigrationsFn(db); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to run migrations: %w", err)
	}
	return db, cleanup, nil
}

// startContainer launches the driver's testcontainers module with the
// package's canonical credentials.
func startContainer(ctx context.Context, driver string) (dbContainer, error) {
	image := Image
	if image == "" {
		image = defaultImageFor(driver)
	}
	var (
		container dbContainer
		err       error
	)
	switch driver {
	case "postgres":
		container, err = postgresRunFn(ctx, image,
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
	case "mysql":
		container, err = mysqlRunFn(ctx, image,
			mysql.WithDatabase("testdb"),
			mysql.WithUsername("testuser"),
			mysql.WithPassword("testpass"),
		)
	case "sqlserver":
		container, err = mssqlRunFn(ctx, image,
			mssql.WithAcceptEULA(),
			mssql.WithPassword("TestPass123!"),
		)
	case "clickhouse":
		container, err = clickhouseRunFn(ctx, image,
			clickhouse.WithDatabase("testdb"),
			clickhouse.WithUsername("testuser"),
			clickhouse.WithPassword("testpass"),
		)
	default:
		return nil, fmt.Errorf("testdb: unsupported driver %q (supported: postgres, mysql, sqlite, sqlserver, clickhouse)", driver)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to start %s container: %w", driver, err)
	}
	return container, nil
}

// connectionString asks the container for its DSN, with per-driver
// extra args where the module supports them.
func connectionString(ctx context.Context, c dbContainer, driver string) (string, error) {
	if driver == "postgres" {
		return c.ConnectionString(ctx, "sslmode=disable")
	}
	return c.ConnectionString(ctx)
}

// dialectorFor maps a driver name + DSN onto the GORM dialector.
func dialectorFor(driver, dsn string) gorm.Dialector {
	switch driver {
	case "mysql":
		return gormmysql.Open(dsn)
	case "sqlserver":
		return gormmssql.Open(dsn)
	case "clickhouse":
		return gormch.Open(dsn)
	default:
		return gormpg.Open(dsn)
	}
}

// defaultMigrations is the canonical Postgres baseline schema applied to
// every Postgres test database. Exported as a slice (not inline) so unit
// tests can reuse the production set or substitute their own to exercise
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
//
// The other drivers have NO shared baseline: their scaffolded 000001
// migrations inline everything per-table (mysql/sqlite/sqlserver
// triggers, clickhouse none) — kept in step with
// cli/internal/skeleton/migrations/<driver>/.
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

// RunMigrations applies the driver-appropriate base migrations to the
// test database. Postgres gets the shared extension + trigger
// functions; every other driver's baseline is empty by design (their
// scaffolded migrations are self-contained per table).
func RunMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	if db.Dialector == nil || db.Name() != "postgres" {
		return nil
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
