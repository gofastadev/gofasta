package providers

import (
	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/validators"
	"github.com/healtronlabs/gofasta/configs"
	"github.com/healtronlabs/gofasta/pkg/logger"
)

// CoreSet provides core infrastructure: config, database, logger, validator.
var CoreSet = wire.NewSet(
	configs.LoadConfig,
	ProvideDBConfig,
	ProvideLogConfig,
	configs.SetupDB,
	logger.NewLogger,
	validators.NewAppValidator,
)

// ProvideDBConfig extracts DatabaseConfig from AppConfig.
func ProvideDBConfig(cfg *configs.AppConfig) *configs.DatabaseConfig {
	return &cfg.Database
}

// ProvideLogConfig extracts LogConfig from AppConfig.
func ProvideLogConfig(cfg *configs.AppConfig) *configs.LogConfig {
	return &cfg.Log
}
