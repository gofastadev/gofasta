package config

import (
	"fmt"
	"log/slog"

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
		sqlDB.SetMaxIdleConns(cfg.MaxIdle)
		sqlDB.SetMaxOpenConns(cfg.MaxOpen)
		sqlDB.SetConnMaxLifetime(cfg.MaxLife)
	}

	return db
}

func buildDialector(cfg *DatabaseConfig) (gorm.Dialector, error) {
	// An explicit DSN wins over the composed form. The templates below cannot
	// express connection-string-only options (search_path, TimeZone overrides,
	// PgBouncer parameters, multi-host targets), so without this a project
	// needing any of them would have to bypass SetupDB entirely.
	if cfg.DSN != "" {
		return dialectorForDSN(cfg.Driver, cfg.DSN)
	}

	switch cfg.Driver {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode,
		)
		return postgres.Open(dsn), nil

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		)
		return mysql.Open(dsn), nil

	case "sqlite":
		return sqlite.Open(cfg.Name), nil

	case "sqlserver":
		dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		)
		return sqlserver.Open(dsn), nil

	case "clickhouse":
		dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		)
		return clickhouse.Open(dsn), nil

	default:
		return nil, fmt.Errorf("unsupported database driver: %q (supported: postgres, mysql, sqlite, sqlserver, clickhouse)", cfg.Driver)
	}
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
