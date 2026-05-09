package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestLoadConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "8080")
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want %q (loopback default avoids macOS firewall prompts on Air rebuilds)", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.ShutdownTimeout != 15*time.Second {
		t.Errorf("Server.ShutdownTimeout = %v, want %v", cfg.Server.ShutdownTimeout, 15*time.Second)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "postgres")
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("Database.Port = %q, want %q", cfg.Database.Port, "5432")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	yamlContent := `server:
  port: "9090"
database:
  driver: "mysql"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9090")
	}
	if cfg.Database.Driver != "mysql" {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "mysql")
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	t.Setenv("GOFASTA_SERVER_PORT", "3000")
	t.Setenv("GOFASTA_DATABASE_DRIVER", "sqlite")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Server.Port != "3000" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "3000")
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "sqlite")
	}
}

// TestLoadConfig_ServerHostEnvOverride asserts that production / Docker
// deployments can flip the Server.Host default (loopback) to 0.0.0.0 via
// an env var alone, without editing config.yaml. This is the production
// path the Dockerfile and compose manifests rely on; if it ever broke,
// scaffolded apps would silently start binding to loopback inside a
// container and become unreachable from the host.
func TestLoadConfig_ServerHostEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	t.Setenv("GOFASTA_SERVER_HOST", "0.0.0.0")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
}

// LoadConfigWithPrefix must honor a project-specific env prefix so a
// scaffolded project named "myapp" can use MYAPP_DATABASE_HOST etc.
// without colliding with other toolkit-scaffolded projects on the same
// host that happen to also use GOFASTA_ as a prefix.
func TestLoadConfigWithPrefix_ProjectPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	t.Setenv("MYAPP_SERVER_PORT", "4000")
	t.Setenv("MYAPP_DATABASE_DRIVER", "mysql")
	t.Setenv("MYAPP_DATABASE_HOST", "db.internal")
	t.Setenv("MYAPP_DATABASE_PORT", "3306")

	cfg, err := LoadConfigWithPrefix("MYAPP_")
	if err != nil {
		t.Fatalf("LoadConfigWithPrefix error: %v", err)
	}
	if cfg.Server.Port != "4000" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "4000")
	}
	if cfg.Database.Driver != "mysql" {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "mysql")
	}
	if cfg.Database.Host != "db.internal" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.internal")
	}
	if cfg.Database.Port != "3306" {
		t.Errorf("Database.Port = %q, want %q", cfg.Database.Port, "3306")
	}
}

// A prefix mismatch must be ignored — MYAPP_* vars should not leak into
// a LoadConfigWithPrefix("OTHER_") call on the same process.
func TestLoadConfigWithPrefix_IgnoresOtherPrefixes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	t.Setenv("MYAPP_DATABASE_DRIVER", "mysql")

	cfg, err := LoadConfigWithPrefix("OTHER_")
	if err != nil {
		t.Fatalf("LoadConfigWithPrefix error: %v", err)
	}
	// MYAPP_DATABASE_DRIVER must NOT bleed through.
	if cfg.Database.Driver != "postgres" {
		t.Errorf("Database.Driver = %q, want %q (the default, since OTHER_ prefix matched nothing)",
			cfg.Database.Driver, "postgres")
	}
}

// LoadConfig() must remain equivalent to LoadConfigWithPrefix("GOFASTA_")
// so existing callers keep working without modification.
func TestLoadConfig_IsAliasForGofastaPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	t.Setenv("GOFASTA_DATABASE_DRIVER", "sqlite")

	cfgA, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	cfgB, err := LoadConfigWithPrefix("GOFASTA_")
	if err != nil {
		t.Fatalf("LoadConfigWithPrefix error: %v", err)
	}
	if cfgA.Database.Driver != cfgB.Database.Driver {
		t.Errorf("LoadConfig and LoadConfigWithPrefix(\"GOFASTA_\") diverged: %q vs %q",
			cfgA.Database.Driver, cfgB.Database.Driver)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &AppConfig{}
	applyDefaults(cfg)

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Server.Host", cfg.Server.Host, "127.0.0.1"},
		{"Server.Port", cfg.Server.Port, "8080"},
		{"Server.ShutdownTimeout", cfg.Server.ShutdownTimeout, 15 * time.Second},
		{"Database.Driver", cfg.Database.Driver, "postgres"},
		{"Database.Host", cfg.Database.Host, "localhost"},
		{"Database.Port", cfg.Database.Port, "5432"},
		{"Database.SSLMode", cfg.Database.SSLMode, "disable"},
		{"Database.MaxIdle", cfg.Database.MaxIdle, 10},
		{"Database.MaxOpen", cfg.Database.MaxOpen, 100},
		{"Database.MaxLife", cfg.Database.MaxLife, time.Hour},
		{"GraphQL.PlaygroundRoute", cfg.GraphQL.PlaygroundRoute, "/graphql-playground"},
		{"GraphQL.GeneralRoute", cfg.GraphQL.GeneralRoute, "/graphql"},
		{"Log.Level", cfg.Log.Level, "info"},
		{"Log.Format", cfg.Log.Format, "text"},
		{"Email.Provider", cfg.Email.Provider, "smtp"},
		{"Email.FromName", cfg.Email.FromName, "App"},
		{"Email.FromAddress", cfg.Email.FromAddress, "noreply@example.com"},
		{"Email.SMTP.Host", cfg.Email.SMTP.Host, "localhost"},
		{"Email.SMTP.Port", cfg.Email.SMTP.Port, 587},
		{"Auth.JWTSecret", cfg.Auth.JWTSecret, "change-me-in-production"},
		{"Auth.AccessTokenExpiry", cfg.Auth.AccessTokenExpiry, 15 * time.Minute},
		{"Auth.RefreshTokenExpiry", cfg.Auth.RefreshTokenExpiry, 168 * time.Hour},
		{"Auth.RBACModelPath", cfg.Auth.RBACModelPath, "configs/rbac_model.conf"},
		{"Auth.RBACPolicyPath", cfg.Auth.RBACPolicyPath, "configs/rbac_policy.csv"},
		{"RateLimit.Rate", cfg.RateLimit.Rate, "100-S"},
		{"RateLimit.Store", cfg.RateLimit.Store, "memory"},
		{"Cache.Driver", cfg.Cache.Driver, "memory"},
		{"Cache.Redis.Host", cfg.Cache.Redis.Host, "localhost"},
		{"Cache.Redis.Port", cfg.Cache.Redis.Port, "6379"},
		{"Security.HSTSMaxAge", cfg.Security.HSTSMaxAge, 31536000},
		{"Security.ReferrerPolicy", cfg.Security.ReferrerPolicy, "strict-origin-when-cross-origin"},
		{"Storage.Driver", cfg.Storage.Driver, "local"},
		{"Storage.Local.Path", cfg.Storage.Local.Path, "./uploads"},
		{"Queue.Concurrency", cfg.Queue.Concurrency, 10},
		{"Queue.Redis.Host", cfg.Queue.Redis.Host, "localhost"},
		{"Queue.Redis.Port", cfg.Queue.Redis.Port, "6379"},
		{"Queue.Redis.DB", cfg.Queue.Redis.DB, 1},
		{"I18n.DefaultLanguage", cfg.I18n.DefaultLanguage, "en"},
		{"I18n.LocalesDir", cfg.I18n.LocalesDir, "locales"},
		{"FeatureFlag.ConfigPath", cfg.FeatureFlag.ConfigPath, "configs/features.yaml"},
		{"Session.Driver", cfg.Session.Driver, "cookie"},
		{"Session.Secret", cfg.Session.Secret, "change-me-in-production-32bytes!"},
		{"Session.SessionName", cfg.Session.SessionName, "app_session"},
		{"Session.FilesystemPath", cfg.Session.FilesystemPath, "./sessions"},
		{"Observability.MetricsPath", cfg.Observability.MetricsPath, "/metrics"},
		{"Observability.ServiceName", cfg.Observability.ServiceName, "app"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	if len(cfg.Server.AllowedOrigins) != 1 || cfg.Server.AllowedOrigins[0] != "*" {
		t.Errorf("Server.AllowedOrigins = %v, want [*]", cfg.Server.AllowedOrigins)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Write invalid YAML content
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("{{invalid yaml content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestBuildDialector_AllDrivers(t *testing.T) {
	tests := []struct {
		driver  string
		wantErr bool
	}{
		{"postgres", false},
		{"mysql", false},
		{"sqlite", false},
		{"sqlserver", false},
		{"clickhouse", false},
		{"unsupported", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			cfg := &DatabaseConfig{
				Driver: tt.driver,
				Host:   "localhost",
				Port:   "5432",
				User:   "user",
				Name:   ":memory:",
			}
			d, err := buildDialector(cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if d == nil {
					t.Error("expected non-nil dialector")
				}
			}
		})
	}
}

func TestSetupDB_SQLite(t *testing.T) {
	cfg := &DatabaseConfig{
		Driver: "sqlite",
		Name:   ":memory:",
	}
	db := SetupDB(cfg)
	if db == nil {
		t.Fatal("expected non-nil *gorm.DB")
	}
}

func TestSetupDB_ConnectionFailure_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unreachable database")
		}
	}()
	SetupDB(&DatabaseConfig{
		Driver:  "postgres",
		Host:    "255.255.255.255",
		Port:    "5432",
		User:    "nobody",
		Name:    "nodb",
		SSLMode: "disable",
	})
}

func TestSetupDB_UnsupportedDriver(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unsupported driver, got none")
		}
	}()

	cfg := &DatabaseConfig{
		Driver: "unsupported",
	}
	SetupDB(cfg)
}

func TestSetupDB_Postgres_PoolConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker")
	}
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
	require.NoError(t, err)
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)
	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	cfg := &DatabaseConfig{
		Driver:   "postgres",
		Host:     host,
		Port:     port.Port(),
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		SSLMode:  "disable",
		MaxIdle:  5,
		MaxOpen:  10,
		MaxLife:  time.Hour,
	}

	db := SetupDB(cfg)
	require.NotNil(t, db)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Verify pool config was applied (MaxOpenConnections is exposed in Stats)
	assert.Equal(t, 10, sqlDB.Stats().MaxOpenConnections)

	// Verify the DB works
	err = sqlDB.Ping()
	require.NoError(t, err)
}
