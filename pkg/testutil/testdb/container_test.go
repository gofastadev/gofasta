package testdb

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"gorm.io/driver/sqlite"
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

// fakeContainer is the in-memory dbContainer used by the unit
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

func swapPostgresRun(t *testing.T, fn func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (dbContainer, error)) {
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
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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
	swapPostgresRun(t, func(_ context.Context, _ string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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
	swapPostgresRun(t, func(_ context.Context, image string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
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

// observingContainer wraps a dbContainer and records when
// Terminate fires. Used to verify the cleanup contract in each error
// test.
type observingContainer struct {
	inner       dbContainer
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

// swapDriver sets the package Driver for one test.
func swapDriver(t *testing.T, driver string) {
	t.Helper()
	orig := Driver
	Driver = driver
	t.Cleanup(func() { Driver = orig })
}

// TestSetupTestDB_UnsupportedDriverErrors — an unknown Driver value
// must fail with a descriptive error, not fall through to postgres.
func TestSetupTestDB_UnsupportedDriverErrors(t *testing.T) {
	swapDriver(t, "oracle")
	_, _, err := setupTestDB(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported driver "oracle"`)
}

// TestSetupTestDB_SQLiteInProcess — the sqlite driver needs no Docker:
// it opens a temp-file database, runs (no-op) migrations, and cleanup
// removes the directory.
func TestSetupTestDB_SQLiteInProcess(t *testing.T) {
	swapDriver(t, "sqlite")
	db, cleanup, err := setupTestDB(context.Background())
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NotNil(t, db)
	assert.Equal(t, "sqlite", db.Name())

	type row struct{ ID uint }
	require.NoError(t, db.AutoMigrate(&row{}))
	require.NoError(t, db.Create(&row{}).Error)
}

// TestDefaultImageFor — each container driver resolves its own default
// image; Image (when set) overrides via startContainer's caller.
func TestDefaultImageFor(t *testing.T) {
	assert.Equal(t, DefaultImage, defaultImageFor("postgres"))
	assert.Equal(t, "mysql:8.4", defaultImageFor("mysql"))
	assert.Equal(t, "mcr.microsoft.com/mssql/server:2022-latest", defaultImageFor("sqlserver"))
	assert.Equal(t, "clickhouse/clickhouse-server:24.8-alpine", defaultImageFor("clickhouse"))
}

// TestDialectorFor — driver names map to the matching GORM dialector.
func TestDialectorFor(t *testing.T) {
	cases := map[string]string{
		"postgres":   "postgres",
		"mysql":      "mysql",
		"sqlserver":  "sqlserver",
		"clickhouse": "clickhouse",
	}
	for driver, want := range cases {
		assert.Equal(t, want, dialectorFor(driver, "dsn").Name(), "driver=%s", driver)
	}
}

// TestStartContainer_DispatchesPerDriver — each driver hits its own run
// seam with that driver's default image.
func TestStartContainer_DispatchesPerDriver(t *testing.T) {
	errStop := errors.New("stop before docker")
	var gotImage string
	record := func(_ context.Context, image string, _ ...testcontainers.ContainerCustomizer) (dbContainer, error) {
		gotImage = image
		return nil, errStop
	}

	origMy, origMs, origCh := mysqlRunFn, mssqlRunFn, clickhouseRunFn
	mysqlRunFn, mssqlRunFn, clickhouseRunFn = record, record, record
	t.Cleanup(func() { mysqlRunFn, mssqlRunFn, clickhouseRunFn = origMy, origMs, origCh })

	for driver, wantImage := range map[string]string{
		"mysql":      "mysql:8.4",
		"sqlserver":  "mcr.microsoft.com/mssql/server:2022-latest",
		"clickhouse": "clickhouse/clickhouse-server:24.8-alpine",
	} {
		gotImage = ""
		_, err := startContainer(context.Background(), driver)
		require.ErrorIs(t, err, errStop, "driver=%s", driver)
		assert.Equal(t, wantImage, gotImage, "driver=%s", driver)
	}
}

// TestRunMigrations_NonPostgresIsNoop — only postgres has a shared
// baseline; other drivers' scaffolded migrations are self-contained.
func TestRunMigrations_NonPostgresIsNoop(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "x.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, RunMigrations(db))
}

// --- sqlite error branches (driven through the package seams) ---

func TestSetupSQLiteTestDB_MkdirTempError(t *testing.T) {
	swapDriver(t, "sqlite")
	orig := osMkdirTempFn
	osMkdirTempFn = func(dir, pattern string) (string, error) {
		return "", errors.New("disk full")
	}
	t.Cleanup(func() { osMkdirTempFn = orig })

	_, _, err := setupTestDB(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite temp dir")
}

func TestSetupSQLiteTestDB_OpenError(t *testing.T) {
	swapDriver(t, "sqlite")
	swapGormOpen(t, func(_ gorm.Dialector, _ ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	})

	_, _, err := setupTestDB(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite test database")
}

func TestSetupSQLiteTestDB_MigrationsError(t *testing.T) {
	swapDriver(t, "sqlite")
	origRun := runMigrationsFn
	runMigrationsFn = func(_ *gorm.DB) error { return errors.New("boom") }
	t.Cleanup(func() { runMigrationsFn = origRun })

	_, _, err := setupTestDB(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run migrations")
}

// TestSetupSQLiteTestDB_ErrorCleanupsAreNoops — the cleanup funcs
// returned on error paths must be callable no-ops (callers register
// them unconditionally via t.Cleanup).
func TestSetupSQLiteTestDB_ErrorCleanupsAreNoops(t *testing.T) {
	swapDriver(t, "sqlite")

	orig := osMkdirTempFn
	osMkdirTempFn = func(dir, pattern string) (string, error) { return "", errors.New("x") }
	_, cleanup, err := setupTestDB(context.Background())
	osMkdirTempFn = orig
	require.Error(t, err)
	cleanup() // must not panic

	origOpen := gormOpenFn
	gormOpenFn = func(_ gorm.Dialector, _ ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}
	_, cleanup, err = setupTestDB(context.Background())
	gormOpenFn = origOpen
	require.Error(t, err)
	cleanup()

	origRun := runMigrationsFn
	runMigrationsFn = func(_ *gorm.DB) error { return errors.New("boom") }
	_, cleanup, err = setupTestDB(context.Background())
	runMigrationsFn = origRun
	require.Error(t, err)
	cleanup()
}

// TestConnectionString_NonPostgresPath — non-postgres drivers call
// ConnectionString without the sslmode arg.
func TestConnectionString_NonPostgresPath(t *testing.T) {
	c := &fakeConnStringContainer{}
	_, err := connectionString(context.Background(), c, "mysql")
	require.NoError(t, err)
	assert.Zero(t, c.gotArgs, "non-postgres drivers pass no extra args")

	_, err = connectionString(context.Background(), c, "postgres")
	require.NoError(t, err)
	assert.Equal(t, 1, c.gotArgs, "postgres passes sslmode=disable")
}

type fakeConnStringContainer struct{ gotArgs int }

func (f *fakeConnStringContainer) Terminate(context.Context, ...testcontainers.TerminateOption) error {
	return nil
}
func (f *fakeConnStringContainer) ConnectionString(_ context.Context, args ...string) (string, error) {
	f.gotArgs = len(args)
	return "dsn", nil
}

// TestProductionRunSeams_DelegateToTestcontainers — invoke the real
// mysql/mssql/clickhouse run closures with an already-canceled context:
// they must reach the testcontainers module (returning its error)
// rather than being dead wiring. Requires a Docker daemon like the rest
// of the non-short suite; the postgres closure is covered by the
// end-to-end test.
func TestProductionRunSeams_DelegateToTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, run := range map[string]func(context.Context, string, ...testcontainers.ContainerCustomizer) (dbContainer, error){
		"mysql":      mysqlRunFn,
		"mssql":      mssqlRunFn,
		"clickhouse": clickhouseRunFn,
	} {
		_, err := run(ctx, defaultImageFor(name))
		assert.Error(t, err, "%s closure must reach testcontainers and surface its context error", name)
	}
}
