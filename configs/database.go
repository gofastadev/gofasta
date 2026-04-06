package configs

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDB(cfg *DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.Port,
		cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	slog.Info("successfully connected to the database")

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("failed to get database instance", "error", err)
		panic(fmt.Sprintf("failed to get database instance: %v", err))
	}
	slog.Info("successfully got the database instance")

	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetConnMaxLifetime(cfg.MaxLife)

	return db
}
