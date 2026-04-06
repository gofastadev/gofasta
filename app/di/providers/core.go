package providers

import (
	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/validators"
	"github.com/healtronlabs/gofasta/configs"
	"github.com/healtronlabs/gofasta/pkg/logger"
	"github.com/healtronlabs/gofasta/pkg/mailer"
)

// CoreSet provides core infrastructure: config, database, logger, validator, email.
var CoreSet = wire.NewSet(
	configs.LoadConfig,
	ProvideDBConfig,
	ProvideLogConfig,
	ProvideEmailConfig,
	configs.SetupDB,
	logger.NewLogger,
	validators.NewAppValidator,
	ProvideTemplateRenderer,
	mailer.NewEmailSender,
)

// ProvideDBConfig extracts DatabaseConfig from AppConfig.
func ProvideDBConfig(cfg *configs.AppConfig) *configs.DatabaseConfig {
	return &cfg.Database
}

// ProvideLogConfig extracts LogConfig from AppConfig.
func ProvideLogConfig(cfg *configs.AppConfig) *configs.LogConfig {
	return &cfg.Log
}

// ProvideEmailConfig extracts EmailConfig from AppConfig.
func ProvideEmailConfig(cfg *configs.AppConfig) *configs.EmailConfig {
	return &cfg.Email
}

// ProvideTemplateRenderer creates the email template renderer.
func ProvideTemplateRenderer(cfg *configs.AppConfig) *mailer.TemplateRenderer {
	return mailer.NewTemplateRenderer("templates/emails", cfg.Email.FromName)
}
