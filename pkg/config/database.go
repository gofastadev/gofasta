package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// SetupDB creates a GORM database connection based on the configured driver.
// Supported drivers: postgres, mysql, sqlite, sqlserver, clickhouse.
func SetupDB(cfg *DatabaseConfig) *gorm.DB {
	dialector, err := buildDialector(cfg)
	if err != nil {
		slog.Error("failed to configure database", "error", err)
		panic(fmt.Sprintf("failed to configure database: %v", err))
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "driver", cfg.Driver, "error", err)
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	slog.Info("successfully connected to the database", "driver", cfg.Driver)

	if cfg.Driver != "sqlite" {
		sqlDB, _ := db.DB()
		pool := poolOrDefaults(cfg)
		sqlDB.SetMaxIdleConns(pool.idle)
		sqlDB.SetMaxOpenConns(pool.open)
		sqlDB.SetConnMaxLifetime(pool.life)
	}

	return db
}

// BuildDSN returns the connection string SetupDB composes for cfg, with
// credentials and identifiers safely escaped per driver:
//
//   - postgres: libpq key=value form; values are single-quoted with
//     backslash escaping, so passwords containing spaces or quotes work.
//   - mysql: go-sql-driver's Config.FormatDSN — its parser explicitly
//     allows any character in the password, no escaping needed.
//   - sqlite: the Name field verbatim (a file path or :memory:).
//   - sqlserver / clickhouse: URL form via net/url, credentials through
//     url.UserPassword so @ : / % and spaces survive.
//
// An explicit cfg.DSN wins and is returned verbatim.
func BuildDSN(cfg *DatabaseConfig) (string, error) {
	if cfg.DSN != "" {
		return cfg.DSN, nil
	}
	switch cfg.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			quotePGValue(cfg.Host), quotePGValue(cfg.User), quotePGValue(cfg.Password),
			quotePGValue(cfg.Name), quotePGValue(cfg.Port), quotePGValue(cfg.SSLMode),
		), nil

	case "mysql":
		mc := mysqldrv.NewConfig()
		mc.User = cfg.User
		mc.Passwd = cfg.Password
		mc.Net = "tcp"
		mc.Addr = cfg.Host + ":" + cfg.Port
		mc.DBName = cfg.Name
		mc.Params = map[string]string{"charset": "utf8mb4"}
		mc.ParseTime = true
		mc.Loc = time.UTC // driver default; kept explicit to match the old loc=UTC DSN
		return mc.FormatDSN(), nil

	case "sqlite":
		return cfg.Name, nil

	case "sqlserver":
		u := url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     cfg.Host + ":" + cfg.Port,
			RawQuery: "database=" + url.QueryEscape(cfg.Name),
		}
		return u.String(), nil

	case "clickhouse":
		u := url.URL{
			Scheme: "clickhouse",
			User:   url.UserPassword(cfg.User, cfg.Password),
			Host:   cfg.Host + ":" + cfg.Port,
			Path:   "/" + cfg.Name,
		}
		return u.String(), nil

	default:
		return "", fmt.Errorf("unsupported database driver: %q (supported: postgres, mysql, sqlite, sqlserver, clickhouse)", cfg.Driver)
	}
}

// BuildMigrateURL returns the golang-migrate database URL for cfg —
// the scheme-prefixed form the `migrate` CLI expects, distinct from the
// dialector DSN (postgres uses key=value there but postgres:// here).
// Credentials are escaped the same way as BuildDSN. Exported so the
// scaffolded cmd/migrate.go builds its URL from the same source of
// truth instead of hand-rolled Sprintf calls.
func BuildMigrateURL(cfg *DatabaseConfig) (string, error) {
	switch cfg.Driver {
	case "postgres":
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     cfg.Host + ":" + cfg.Port,
			Path:     "/" + cfg.Name,
			RawQuery: "sslmode=" + url.QueryEscape(cfg.SSLMode),
		}
		return u.String(), nil

	case "mysql":
		// golang-migrate strips the scheme and hands the rest to
		// go-sql-driver's ParseDSN, which allows any password character.
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name), nil

	case "sqlite":
		return "sqlite3://" + cfg.Name, nil

	case "sqlserver":
		u := url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     cfg.Host + ":" + cfg.Port,
			RawQuery: "database=" + url.QueryEscape(cfg.Name),
		}
		return u.String(), nil

	case "clickhouse":
		u := url.URL{
			Scheme: "clickhouse",
			User:   url.UserPassword(cfg.User, cfg.Password),
			Host:   cfg.Host + ":" + cfg.Port,
			Path:   "/" + cfg.Name,
		}
		return u.String(), nil

	default:
		return "", fmt.Errorf("unsupported database driver: %q (supported: postgres, mysql, sqlite, sqlserver, clickhouse)", cfg.Driver)
	}
}

// quotePGValue makes a value safe for libpq's key=value DSN form: empty
// values, and values containing spaces, quotes, or backslashes are
// single-quoted with backslash escaping (per the libpq connection-string
// rules). Plain values pass through untouched for readability.
func quotePGValue(v string) string {
	if v != "" && !strings.ContainsAny(v, ` '\`) {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

func buildDialector(cfg *DatabaseConfig) (gorm.Dialector, error) {
	// An explicit DSN wins over the composed form. The templates below cannot
	// express connection-string-only options (search_path, TimeZone overrides,
	// PgBouncer parameters, multi-host targets), so without this a project
	// needing any of them would have to bypass SetupDB entirely.
	dsn, err := BuildDSN(cfg)
	if err != nil {
		return nil, err
	}
	return dialectorForDSN(cfg.Driver, dsn)
}

// dialectorForDSN opens the driver's dialector against a caller-supplied
// connection string. Driver selection still comes from cfg.Driver because a
// DSN's scheme is not a reliable discriminator — "postgres://" and
// "postgresql://" are both valid, and the key=value form has no scheme at all.
func dialectorForDSN(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "postgres":
		return postgres.Open(dsn), nil
	case "mysql":
		return mysql.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	case "sqlserver":
		return sqlserver.Open(dsn), nil
	case "clickhouse":
		return clickhouse.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %q (supported: postgres, mysql, sqlite, sqlserver, clickhouse)", driver)
	}
}

// poolOrDefaults fills in pool limits the caller left unset.
//
// LoadConfig applies these when reading config.yaml, but a caller that builds
// a DatabaseConfig literal — passing a DSN directly, say — gets the zero
// value, and database/sql reads zero MaxOpenConns as *unlimited*. A service
// under load then opens connections until PostgreSQL refuses them, which
// surfaces as "too many clients" on an unrelated request rather than anywhere
// near the code that configured the pool.
func poolOrDefaults(cfg *DatabaseConfig) struct {
	idle, open int
	life       time.Duration
} {
	pool := struct {
		idle, open int
		life       time.Duration
	}{idle: cfg.MaxIdle, open: cfg.MaxOpen, life: cfg.MaxLife}

	if pool.idle == 0 {
		pool.idle = defaultMaxIdleConns
	}
	if pool.open == 0 {
		pool.open = defaultMaxOpenConns
	}
	if pool.life == 0 {
		pool.life = defaultConnMaxLifetime
	}
	return pool
}

// Pool defaults, matching the ones LoadConfig applies.
const (
	defaultMaxIdleConns    = 10
	defaultMaxOpenConns    = 100
	defaultConnMaxLifetime = time.Hour
)
