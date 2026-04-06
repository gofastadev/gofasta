package configs

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

	// SQLite doesn't support connection pooling
	if cfg.Driver != "sqlite" {
		sqlDB, err := db.DB()
		if err != nil {
			slog.Error("failed to get database instance", "error", err)
			panic(fmt.Sprintf("failed to get database instance: %v", err))
		}
		sqlDB.SetMaxIdleConns(cfg.MaxIdle)
		sqlDB.SetMaxOpenConns(cfg.MaxOpen)
		sqlDB.SetConnMaxLifetime(cfg.MaxLife)
	}

	return db
}

func buildDialector(cfg *DatabaseConfig) (gorm.Dialector, error) {
	switch cfg.Driver {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode,
		)
		return postgres.Open(dsn), nil

	case "mysql":
		// DSN format: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=UTC
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		)
		return mysql.Open(dsn), nil

	case "sqlite":
		// Name is the file path for SQLite, e.g. "gofasta.db" or ":memory:"
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
