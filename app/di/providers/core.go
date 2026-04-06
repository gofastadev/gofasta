package providers

import (
	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/validators"
	"github.com/healtronlabs/gofasta/configs"
	"github.com/healtronlabs/gofasta/pkg/auth"
	"github.com/healtronlabs/gofasta/pkg/cache"
	"github.com/healtronlabs/gofasta/pkg/logger"
	"github.com/healtronlabs/gofasta/pkg/mailer"
	"github.com/healtronlabs/gofasta/pkg/queue"
	"github.com/healtronlabs/gofasta/pkg/storage"
)

// CoreSet provides core infrastructure.
var CoreSet = wire.NewSet(
	configs.LoadConfig,
	ProvideDBConfig,
	ProvideLogConfig,
	ProvideEmailConfig,
	ProvideAuthConfig,
	ProvideCacheConfig,
	ProvideStorageConfig,
	ProvideQueueConfig,
	configs.SetupDB,
	logger.NewLogger,
	validators.NewAppValidator,
	ProvideTemplateRenderer,
	mailer.NewEmailSender,
	auth.NewJWTService,
	auth.NewRBACService,
	cache.NewCacheService,
	storage.NewStorageService,
	queue.NewQueueService,
)

func ProvideDBConfig(cfg *configs.AppConfig) *configs.DatabaseConfig {
	return &cfg.Database
}

func ProvideLogConfig(cfg *configs.AppConfig) *configs.LogConfig {
	return &cfg.Log
}

func ProvideEmailConfig(cfg *configs.AppConfig) *configs.EmailConfig {
	return &cfg.Email
}

func ProvideAuthConfig(cfg *configs.AppConfig) *configs.AuthConfig {
	return &cfg.Auth
}

func ProvideCacheConfig(cfg *configs.AppConfig) *configs.CacheConfig {
	return &cfg.Cache
}

func ProvideStorageConfig(cfg *configs.AppConfig) *configs.StorageConfig {
	return &cfg.Storage
}

func ProvideQueueConfig(cfg *configs.AppConfig) *configs.QueueConfig {
	return &cfg.Queue
}

func ProvideTemplateRenderer(cfg *configs.AppConfig) *mailer.TemplateRenderer {
	return mailer.NewTemplateRenderer("templates/emails", cfg.Email.FromName)
}
