package configs

import (
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// AppConfig holds the complete application configuration.
type AppConfig struct {
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	GraphQL  GraphQLConfig  `koanf:"graphql"`
	Log      LogConfig      `koanf:"log"`
	Email    EmailConfig    `koanf:"email"`
	Jobs     []JobConfig    `koanf:"jobs"`
}

// JobConfig defines a single cron job schedule.
type JobConfig struct {
	Name     string `koanf:"name"`
	Schedule string `koanf:"schedule"`
	Enabled  bool   `koanf:"enabled"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            string        `koanf:"port"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
	AllowedOrigins  []string      `koanf:"allowed_origins"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host     string        `koanf:"host"`
	Port     string        `koanf:"port"`
	User     string        `koanf:"user"`
	Password string        `koanf:"password"`
	Name     string        `koanf:"name"`
	SSLMode  string        `koanf:"sslmode"`
	MaxIdle  int           `koanf:"max_idle"`
	MaxOpen  int           `koanf:"max_open"`
	MaxLife  time.Duration `koanf:"max_life"`
}

// GraphQLConfig holds GraphQL endpoint settings.
type GraphQLConfig struct {
	PlaygroundRoute string `koanf:"playground_route"`
	GeneralRoute    string `koanf:"general_route"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// EmailConfig holds email provider settings.
type EmailConfig struct {
	Provider    string         `koanf:"provider"`
	FromName    string         `koanf:"from_name"`
	FromAddress string         `koanf:"from_address"`
	SMTP        SMTPConfig     `koanf:"smtp"`
	SendGrid    SendGridConfig `koanf:"sendgrid"`
	Brevo       BrevoConfig    `koanf:"brevo"`
}

// SMTPConfig holds SMTP server settings.
type SMTPConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	UseTLS   bool   `koanf:"use_tls"`
}

// SendGridConfig holds SendGrid API settings.
type SendGridConfig struct {
	APIKey string `koanf:"api_key"`
}

// BrevoConfig holds Brevo (Sendinblue) API settings.
type BrevoConfig struct {
	APIKey string `koanf:"api_key"`
}

// LoadConfig loads configuration from config.yaml (if present), then overlays
// environment variables prefixed with GOFASTA_ (e.g., GOFASTA_DATABASE_HOST).
func LoadConfig() (*AppConfig, error) {
	k := koanf.New(".")

	// Load from config.yaml if it exists
	if _, err := os.Stat("config.yaml"); err == nil {
		if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
			return nil, err
		}
	}

	// Overlay with env vars prefixed with GOFASTA_ (e.g., GOFASTA_DATABASE_HOST)
	if err := k.Load(env.Provider("GOFASTA_", ".", func(s string) string {
		return strings.Replace(
			strings.ToLower(strings.TrimPrefix(s, "GOFASTA_")),
			"_", ".", -1,
		)
	}), nil); err != nil {
		return nil, err
	}

	cfg := &AppConfig{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, err
	}

	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *AppConfig) {
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 15 * time.Second
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		cfg.Server.AllowedOrigins = []string{"*"}
	}
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == "" {
		cfg.Database.Port = "5432"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxIdle == 0 {
		cfg.Database.MaxIdle = 10
	}
	if cfg.Database.MaxOpen == 0 {
		cfg.Database.MaxOpen = 100
	}
	if cfg.Database.MaxLife == 0 {
		cfg.Database.MaxLife = time.Hour
	}
	if cfg.GraphQL.PlaygroundRoute == "" {
		cfg.GraphQL.PlaygroundRoute = "/graphql-playground"
	}
	if cfg.GraphQL.GeneralRoute == "" {
		cfg.GraphQL.GeneralRoute = "/graphql"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "text"
	}
	if cfg.Email.Provider == "" {
		cfg.Email.Provider = "smtp"
	}
	if cfg.Email.FromName == "" {
		cfg.Email.FromName = "Gofasta App"
	}
	if cfg.Email.FromAddress == "" {
		cfg.Email.FromAddress = "noreply@example.com"
	}
	if cfg.Email.SMTP.Host == "" {
		cfg.Email.SMTP.Host = "localhost"
	}
	if cfg.Email.SMTP.Port == 0 {
		cfg.Email.SMTP.Port = 587
	}
}
