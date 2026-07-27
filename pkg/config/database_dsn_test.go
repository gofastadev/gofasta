package config

import (
	"strings"
	"testing"
)

// TestBuildDialector_DSNWinsOverComposedFields covers the escape hatch that
// lets a project supply a connection string the composed form cannot express —
// PostgreSQL's search_path, a non-UTC TimeZone, PgBouncer parameters. Without
// it, such a project has no way to reach the dialector through SetupDB.
func TestBuildDialector_DSNWinsOverComposedFields(t *testing.T) {
	cfg := &DatabaseConfig{
		Driver: "postgres",
		DSN:    "postgresql://u:p@db:5432/app?search_path=public&sslmode=disable",
		// Deliberately contradictory: if these were used instead of DSN the
		// dialector would point at a different server entirely.
		Host: "ignored-host", Port: "9999", User: "ignored", Name: "ignored",
	}

	d, err := buildDialector(cfg)
	if err != nil {
		t.Fatalf("buildDialector: %v", err)
	}
	if d == nil {
		t.Fatal("nil dialector")
	}
	if got := d.Name(); got != "postgres" {
		t.Errorf("dialector name = %q, want postgres", got)
	}
}

func TestBuildDialector_FallsBackToComposedFields(t *testing.T) {
	cfg := &DatabaseConfig{
		Driver: "postgres", Host: "h", Port: "5432",
		User: "u", Password: "p", Name: "n", SSLMode: "disable",
	}
	if _, err := buildDialector(cfg); err != nil {
		t.Fatalf("buildDialector without DSN: %v", err)
	}
}

func TestBuildDialector_DSNWithUnsupportedDriver(t *testing.T) {
	_, err := buildDialector(&DatabaseConfig{Driver: "oracle", DSN: "whatever"})
	if err == nil {
		t.Fatal("expected an error for an unsupported driver")
	}
	if !strings.Contains(err.Error(), "unsupported database driver") {
		t.Errorf("error = %q, want it to name the unsupported driver", err)
	}
}
