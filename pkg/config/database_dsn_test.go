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

// TestBuildDialector_DSNAcrossSupportedDrivers walks every driver through the
// DSN path. Driver selection cannot be inferred from the connection string —
// "postgres://" and "postgresql://" are both valid and the key=value form has
// no scheme at all — so each driver needs its own arm exercised.
func TestBuildDialector_DSNAcrossSupportedDrivers(t *testing.T) {
	cases := []struct {
		driver string
		dsn    string
	}{
		{"postgres", "postgresql://u:p@db:5432/app?sslmode=disable"},
		{"mysql", "u:p@tcp(db:3306)/app?parseTime=true"},
		{"sqlite", "file:app.db?cache=shared"},
		{"sqlserver", "sqlserver://u:p@db:1433?database=app"},
		{"clickhouse", "clickhouse://u:p@db:9000/app"},
	}

	for _, tc := range cases {
		t.Run(tc.driver, func(t *testing.T) {
			d, err := buildDialector(&DatabaseConfig{Driver: tc.driver, DSN: tc.dsn})
			if err != nil {
				t.Fatalf("buildDialector(%s): %v", tc.driver, err)
			}
			if d == nil {
				t.Fatalf("buildDialector(%s): nil dialector", tc.driver)
			}
		})
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

// --- BuildDSN / BuildMigrateURL escaping ---

// nastyPassword exercises every character class that used to break the
// Sprintf-composed connection strings: URL delimiters, spaces, quotes,
// percent signs.
const nastyPassword = `p@ss:w/o?r#d %25 'quote' \back`

func TestBuildDSN_PostgresEscapesSpacesAndQuotes(t *testing.T) {
	cfg := &DatabaseConfig{Driver: "postgres", Host: "db", Port: "5432",
		User: "app user", Password: nastyPassword, Name: "myapp", SSLMode: "disable"}
	dsn, err := BuildDSN(cfg)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	if !strings.Contains(dsn, `user='app user'`) {
		t.Errorf("user with space must be quoted, got %q", dsn)
	}
	if !strings.Contains(dsn, `password='`) || !strings.Contains(dsn, `\'quote\'`) {
		t.Errorf("password quotes must be escaped, got %q", dsn)
	}
}

func TestBuildDSN_MySQLUsesFormatDSN(t *testing.T) {
	cfg := &DatabaseConfig{Driver: "mysql", Host: "db", Port: "3306",
		User: "u", Password: nastyPassword, Name: "myapp"}
	dsn, err := BuildDSN(cfg)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	if !strings.Contains(dsn, "tcp(db:3306)") || !strings.Contains(dsn, "/myapp") {
		t.Errorf("unexpected mysql DSN shape: %q", dsn)
	}
	if !strings.Contains(dsn, "parseTime=true") {
		t.Errorf("parseTime must be set, got %q", dsn)
	}
}

func TestBuildDSN_URLDriversEscapeCredentials(t *testing.T) {
	for _, driver := range []string{"sqlserver", "clickhouse"} {
		cfg := &DatabaseConfig{Driver: driver, Host: "db", Port: "1433",
			User: "u@corp", Password: nastyPassword, Name: "my app"}
		dsn, err := BuildDSN(cfg)
		if err != nil {
			t.Fatalf("BuildDSN(%s): %v", driver, err)
		}
		if strings.Count(dsn, "@") != 1 {
			t.Errorf("%s: credentials must be escaped so exactly one @ separates userinfo from host, got %q", driver, dsn)
		}
		if strings.Contains(dsn, " ") {
			t.Errorf("%s: spaces must be escaped, got %q", driver, dsn)
		}
	}
}

func TestBuildDSN_SQLiteIsName(t *testing.T) {
	dsn, err := BuildDSN(&DatabaseConfig{Driver: "sqlite", Name: "app.db"})
	if err != nil || dsn != "app.db" {
		t.Errorf("sqlite DSN = %q, %v; want app.db", dsn, err)
	}
}

func TestBuildDSN_ExplicitDSNWins(t *testing.T) {
	dsn, err := BuildDSN(&DatabaseConfig{Driver: "postgres", DSN: "verbatim", Host: "x"})
	if err != nil || dsn != "verbatim" {
		t.Errorf("explicit DSN must win, got %q, %v", dsn, err)
	}
}

func TestBuildDSN_UnsupportedDriver(t *testing.T) {
	_, err := BuildDSN(&DatabaseConfig{Driver: "oracle"})
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestBuildMigrateURL_EscapesPerDriver(t *testing.T) {
	pg, err := BuildMigrateURL(&DatabaseConfig{Driver: "postgres", Host: "db", Port: "5432",
		User: "u", Password: nastyPassword, Name: "myapp", SSLMode: "disable"})
	if err != nil {
		t.Fatalf("postgres: %v", err)
	}
	if !strings.HasPrefix(pg, "postgres://") || strings.Count(pg, "@") != 1 {
		t.Errorf("postgres migrate URL must escape credentials, got %q", pg)
	}

	lite, err := BuildMigrateURL(&DatabaseConfig{Driver: "sqlite", Name: "app.db"})
	if err != nil || lite != "sqlite3://app.db" {
		t.Errorf("sqlite migrate URL = %q, %v", lite, err)
	}

	my, err := BuildMigrateURL(&DatabaseConfig{Driver: "mysql", Host: "db", Port: "3306",
		User: "u", Password: "simple", Name: "myapp"})
	if err != nil || my != "mysql://u:simple@tcp(db:3306)/myapp" {
		t.Errorf("mysql migrate URL = %q, %v", my, err)
	}

	if _, err := BuildMigrateURL(&DatabaseConfig{Driver: "oracle"}); err == nil {
		t.Error("expected error for unsupported driver")
	}
}

func TestQuotePGValue(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"":           "''",
		"two words":  "'two words'",
		`with'quote`: `'with\'quote'`,
		`back\slash`: `'back\\slash'`,
	}
	for in, want := range cases {
		if got := quotePGValue(in); got != want {
			t.Errorf("quotePGValue(%q) = %q, want %q", in, got, want)
		}
	}
}
