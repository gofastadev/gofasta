package providers

import (
	"log/slog"

	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/validators"
	"github.com/healtronlabs/gofasta/configs"
	"github.com/healtronlabs/gofasta/pkg/auth"
	"github.com/healtronlabs/gofasta/pkg/cache"
	"github.com/healtronlabs/gofasta/pkg/encryption"
	"github.com/healtronlabs/gofasta/pkg/i18n"
	"github.com/healtronlabs/gofasta/pkg/logger"
	"github.com/healtronlabs/gofasta/pkg/mailer"
	"github.com/healtronlabs/gofasta/pkg/notify"
	"github.com/healtronlabs/gofasta/pkg/queue"
	"github.com/healtronlabs/gofasta/pkg/session"
	"github.com/healtronlabs/gofasta/pkg/storage"
	"github.com/healtronlabs/gofasta/pkg/websocket"
	"gorm.io/gorm"
)

// CoreSet provides all core infrastructure.
var CoreSet = wire.NewSet(
	// Config
	configs.LoadConfig,
	ProvideDBConfig,
	ProvideLogConfig,
	ProvideEmailConfig,
	ProvideAuthConfig,
	ProvideCacheConfig,
	ProvideStorageConfig,
	ProvideQueueConfig,
	// Infrastructure
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
	// Newly wired
	ProvideWebSocketHub,
	ProvideNotifier,
	ProvideI18nService,
	ProvideEncrypter,
	ProvideSessionStore,
)

// --- Config extractors ---

func ProvideDBConfig(cfg *configs.AppConfig) *configs.DatabaseConfig       { return &cfg.Database }
func ProvideLogConfig(cfg *configs.AppConfig) *configs.LogConfig           { return &cfg.Log }
func ProvideEmailConfig(cfg *configs.AppConfig) *configs.EmailConfig       { return &cfg.Email }
func ProvideAuthConfig(cfg *configs.AppConfig) *configs.AuthConfig         { return &cfg.Auth }
func ProvideCacheConfig(cfg *configs.AppConfig) *configs.CacheConfig       { return &cfg.Cache }
func ProvideStorageConfig(cfg *configs.AppConfig) *configs.StorageConfig   { return &cfg.Storage }
func ProvideQueueConfig(cfg *configs.AppConfig) *configs.QueueConfig       { return &cfg.Queue }

// --- Complex providers ---

func ProvideTemplateRenderer(cfg *configs.AppConfig) *mailer.TemplateRenderer {
	return mailer.NewTemplateRenderer("templates/emails", cfg.Email.FromName)
}

func ProvideWebSocketHub(logger *slog.Logger) *websocket.Hub {
	return websocket.NewHub(logger)
}

func ProvideNotifier(logger *slog.Logger, emailSender mailer.EmailSender, db *gorm.DB) *notify.Notifier {
	return notify.NewNotifier(logger,
		notify.NewEmailChannel(emailSender),
		notify.NewDatabaseChannel(db),
	)
}

func ProvideI18nService(cfg *configs.AppConfig) *i18n.I18nService {
	return i18n.NewI18nService(cfg.I18n.LocalesDir, cfg.I18n.DefaultLanguage)
}

func ProvideEncrypter(cfg *configs.AppConfig) *encryption.Encrypter {
	if cfg.Encryption.Key == "" {
		return nil
	}
	enc, err := encryption.NewEncrypter(cfg.Encryption.Key)
	if err != nil {
		return nil
	}
	return enc
}

func ProvideSessionStore(cfg *configs.AppConfig) *session.Store {
	sc := cfg.Session
	if sc.Driver == "filesystem" {
		return session.NewFilesystemStore(sc.FilesystemPath, sc.Secret, sc.SessionName)
	}
	return session.NewCookieStore(sc.Secret, sc.SessionName)
}
