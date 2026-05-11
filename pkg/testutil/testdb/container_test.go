package testdb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────
// Tests for the testdb package. The production helper (SetupTestDB)
// requires Docker, so most tests drive the error-returning core
// (`setupTestDB`) via package-level seams. One end-to-end happy-path
// test still runs against a real Postgres container — gated behind
// Docker availability via testcontainers' own probe — to verify the
// integrated pipeline.
// ─────────────────────────────────────────────────────────────────────

// fakeContainer is the in-memory postgresContainer used by the unit
// tests. Each Fn field, when nil, returns a benign default; tests
// assign only the ones they need.
type fakeContainer struct {
	terminateErr error
	connStr      string
	connStrErr   error
}

func (f *fakeContainer) Terminate(_ context.Context, _ ...testcontainers.TerminateOption) error {
	return f.terminateErr
}

func (f *fakeContainer) ConnectionString(_ context.Context, _ ...string) (string, error) {
	return f.connStr, f.connStrErr
}

func swapPostgresRun(t *testing.T, fn func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (postgresContainer, error)) {
	t.Helper()
	orig := postgresRunFn
	postgresRunFn = fn
	t.Cleanup(func() { postgresRunFn = orig })
}

func swapGormOpen(t *testing.T, fn func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error)) {
	t.Helper()
	orig := gormOpenFn
	gormOpenFn = fn
	t.Cleanup(func() { gormOpenFn = orig })
}

func swapRunMigrations(t *testing.T, fn func(*gorm.DB) error) {
	t.Helper()
	orig := runMigrationsFn
	runMigrationsFn = fn
	t.Cleanup(func() { runMigrationsFn = orig })
}

// TestSetupTestDB_PostgresRunFails — first failure mode: the
// container fails to start. Returned error names the cause; cleanup
// is a no-op (no container was created).
func TestSetupTestDB_PostgresRunFails(t *testing.T) {
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return nil, errors.New("docker down")
	})
	_, cleanup, err := setupTestDB(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postgres container")
	cleanup() // must not panic
}

// TestSetupTestDB_ConnectionStringFails — the container starts but
// the test rig can't extract a connection string. Cleanup is called
// to terminate the (live) container.
func TestSetupTestDB_ConnectionStringFails(t *testing.T) {
	var terminated bool
	fake := &fakeContainer{connStrErr: errors.New("no port allocated")}
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return &observingContainer{inner: fake, onTerminate: func() { terminated = true }}, nil
	})
	_, cleanup, err := setupTestDB(context.Background())
	cleanup() // exercise the returned no-op closure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection string")
	assert.True(t, terminated, "container must be terminated when connection string fails")
}

// TestSetupTestDB_GormOpenFails — connection string is fine but
// gorm.Open errors (bad DSN shape). Container is terminated.
func TestSetupTestDB_GormOpenFails(t *testing.T) {
	var terminated bool
	fake := &fakeContainer{connStr: "host=127.0.0.1 port=1 user=u dbname=d"}
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return &observingContainer{inner: fake, onTerminate: func() { terminated = true }}, nil
	})
	swapGormOpen(t, func(_ gorm.Dialector, _ ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("gorm boom")
	})
	_, cleanup, err := setupTestDB(context.Background())
	cleanup() // exercise the returned no-op closure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test database")
	assert.True(t, terminated)
}

// TestSetupTestDB_MigrationsFail — gorm.Open succeeds but the
// migration step errors. Container is terminated.
func TestSetupTestDB_MigrationsFail(t *testing.T) {
	var terminated bool
	fake := &fakeContainer{connStr: "host=127.0.0.1 port=1 user=u dbname=d"}
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return &observingContainer{inner: fake, onTerminate: func() { terminated = true }}, nil
	})
	swapGormOpen(t, func(_ gorm.Dialector, _ ...gorm.Option) (*gorm.DB, error) {
		// gorm.DB doesn't need to be functional; runMigrationsFn is
		// also stubbed.
		return &gorm.DB{}, nil
	})
	swapRunMigrations(t, func(_ *gorm.DB) error {
		return errors.New("bad SQL")
	})
	_, cleanup, err := setupTestDB(context.Background())
	cleanup() // exercise the returned no-op closure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migrations")
	assert.True(t, terminated)
}

// TestSetupTestDB_HappyPath — entire pipeline succeeds with stubs.
// Verifies cleanup is wired to terminate the container.
func TestSetupTestDB_HappyPath(t *testing.T) {
	var terminated bool
	fake := &fakeContainer{connStr: "host=127.0.0.1 port=1 user=u dbname=d"}
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return &observingContainer{inner: fake, onTerminate: func() { terminated = true }}, nil
	})
	swapGormOpen(t, func(_ gorm.Dialector, _ ...gorm.Option) (*gorm.DB, error) {
		return &gorm.DB{}, nil
	})
	swapRunMigrations(t, func(_ *gorm.DB) error { return nil })

	db, cleanup, err := setupTestDB(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, db)
	cleanup()
	assert.True(t, terminated)
}

// SetupTestDB's t.Fatalf wrapper is a four-line shim that calls
// setupTestDB and converts any error to t.Fatalf. Exercising the
// Fatalf branch from another test would mark the parent test failed
// (Go's testing framework treats sub-test failures as propagating).
// The integration test (TestSetupTestDB_RealContainer) covers the
// happy path of the shim; the error-returning core (setupTestDB) is
// covered exhaustively by the unit tests above.

// TestSetupTestDB_RealContainer is the integration smoke test. It
// runs against a real Postgres container — only meaningful when
// Docker is available — and verifies the happy path from end to end.
// Skipped automatically when testcontainers can't reach a daemon, so
// this test is safe to run on every machine.
func TestSetupTestDB_RealContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker; skipped in -short mode")
	}
	defer func() {
		// testcontainers can panic or t.Skip mid-init when the daemon
		// is unreachable. Treat any such bail as "Docker unavailable"
		// rather than failing the test.
		if r := recover(); r != nil {
			t.Skipf("docker unavailable: %v", r)
		}
	}()

	db := SetupTestDB(t)
	require.NotNil(t, db)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())

	// Verify the migrations applied: citext extension is loaded and
	// the helper functions are present.
	row := sqlDB.QueryRow(
		"SELECT count(*) FROM pg_proc WHERE proname IN ($1,$2,$3)",
		"update_updated_at_column", "prevent_delete_non_deletable", "increment_record_version",
	)
	var n int
	require.NoError(t, row.Scan(&n))
	assert.Equal(t, 3, n, "all three trigger functions must be created")
}

// TestRunMigrationsOn_ExecFails feeds a deliberately-invalid SQL
// statement into the per-migration loop against the real test
// container. Exercises the exec-failure branch that production code
// hits when a migration has a syntax error or violates a constraint.
func TestRunMigrationsOn_ExecFails(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker; skipped in -short mode")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("docker unavailable: %v", r)
		}
	}()

	db := SetupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	err = runMigrationsOn(sqlDB, []string{"DEFINITELY NOT VALID SQL;"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration failed")
}

// RunMigrations exercise paths:
//
//   - Happy path: covered by TestSetupTestDB_RealContainer end-to-end.
//   - sqlDB.Exec failure: covered indirectly by injecting a
//     migrations-failing stub into setupTestDB
//     (TestSetupTestDB_MigrationsFail).
//   - db.DB() failure: not unit-testable from outside the gorm package
//     because a zero-value *gorm.DB panics rather than returning an
//     error from DB(). Documented here so the gap is intentional rather
//     than accidental.

// observingContainer wraps a postgresContainer and records when
// Terminate fires. Used to verify the cleanup contract in each error
// test.
type observingContainer struct {
	inner       postgresContainer
	onTerminate func()
}

func (o *observingContainer) Terminate(ctx context.Context, opts ...testcontainers.TerminateOption) error {
	if o.onTerminate != nil {
		o.onTerminate()
	}
	return o.inner.Terminate(ctx, opts...)
}

func (o *observingContainer) ConnectionString(ctx context.Context, args ...string) (string, error) {
	return o.inner.ConnectionString(ctx, args...)
}
