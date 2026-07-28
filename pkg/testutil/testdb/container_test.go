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

// TestSetupTestDB_FatalsOnSetupError covers the shim's error branch.
//
// The branch cannot be driven through a sub-test, because a failure there
// propagates to the parent. It can be driven on a throwaway *testing.T with no
// parent: t.Fatalf records the failure on that value and then calls
// runtime.Goexit, so running the call in its own goroutine confines the abort
// to that goroutine and leaves this test's own T untouched.
func TestSetupTestDB_FatalsOnSetupError(t *testing.T) {
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		return nil, errors.New("docker down")
	})

	var (
		sink     testing.T
		returned bool
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		SetupTestDB(&sink)
		returned = true // not reached: Fatalf ends this goroutine
	}()
	<-done

	assert.False(t, returned, "SetupTestDB must abort the test rather than return a nil DB")
	assert.True(t, sink.Failed(), "the setup failure must be reported through the caller's *testing.T")
}

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

	// Verify the migrations applied: citext extension is loaded and the helper
	// functions are present under the SAME names the scaffold's generated
	// migrations reference in their CREATE TRIGGER statements. The names below
	// are the contract — this package used to create shorter ones, so applying
	// a generated migration set against a database prepared here failed with
	// "function ... does not exist".
	row := sqlDB.QueryRow(
		"SELECT count(*) FROM pg_proc WHERE proname IN ($1,$2,$3)",
		"update_updated_at_column_function",
		"avoid_deleting_record_with_is_deletable_equal_to_false_function",
		"increment_record_version_column_function",
	)
	var n int
	require.NoError(t, row.Scan(&n))
	assert.Equal(t, 3, n, "all three trigger functions must be created under their scaffold-migration names")
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
//   - db.DB() failure: covered by TestRunMigrations_UnusableDBErrors.

// TestRunMigrations_UnusableDBErrors covers the db.DB() failure branch.
//
// The Config must be non-nil: gorm.DB embeds *gorm.Config and DB() reads
// ConnPool through it, so a fully zero-value &gorm.DB{} segfaults instead of
// returning an error. With Config set and no pool assigned, DB() reports
// ErrInvalidDB and the branch is reachable without a container.
func TestRunMigrations_UnusableDBErrors(t *testing.T) {
	err := RunMigrations(&gorm.DB{Config: &gorm.Config{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get sql.DB")
	assert.ErrorIs(t, err, gorm.ErrInvalidDB)
}

// TestSetupTestDB_EmptyImageFallsBackToDefault covers the Image fallback. A
// project that clears the override — or assigns it from an unset env var —
// must still get a working image rather than an empty container reference.
func TestSetupTestDB_EmptyImageFallsBackToDefault(t *testing.T) {
	var requestedImage string
	swapPostgresRun(t, func(_ context.Context, image string, _ ...testcontainers.ContainerCustomizer) (postgresContainer, error) {
		requestedImage = image
		return nil, errors.New("not started; the image argument is what matters here")
	})

	orig := Image
	Image = ""
	t.Cleanup(func() { Image = orig })

	_, cleanup, err := setupTestDB(context.Background())
	cleanup()
	require.Error(t, err)
	assert.Equal(t, DefaultImage, requestedImage)
}

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
