// Package config provides configuration loading for gofasta applications.
package config

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
	Server    ServerConfig    `koanf:"server"`
	Database  DatabaseConfig  `koanf:"database"`
	GraphQL   GraphQLConfig   `koanf:"graphql"`
	Log       LogConfig       `koanf:"log"`
	Email     EmailConfig     `koanf:"email"`
	Jobs      []JobConfig     `koanf:"jobs"`
	Auth      AuthConfig      `koanf:"auth"`
	RateLimit RateLimitConfig `koanf:"rate_limit"`
	Cache     CacheConfig     `koanf:"cache"`
	Security  SecurityConfig  `koanf:"security"`
	Storage       StorageConfig       `koanf:"storage"`
	Queue         QueueConfig         `koanf:"queue"`
	WebSocket     WebSocketConfig     `koanf:"websocket"`
	I18n          I18nConfig          `koanf:"i18n"`
	FeatureFlag   FeatureFlagConfig   `koanf:"feature_flag"`
	Encryption    EncryptionConfig    `koanf:"encryption"`
	Session       SessionConfig       `koanf:"session"`
	Observability ObservabilityConfig `koanf:"observability"`
}

type JobConfig struct {
	Name     string `koanf:"name"`
	Schedule string `koanf:"schedule"`
	Enabled  bool   `koanf:"enabled"`
}

type ServerConfig struct {
	Port            string        `koanf:"port"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
	AllowedOrigins  []string      `koanf:"allowed_origins"`
}

type DatabaseConfig struct {
	Driver   string        `koanf:"driver"`
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

type GraphQLConfig struct {
	PlaygroundRoute string `koanf:"playground_route"`
	GeneralRoute    string `koanf:"general_route"`
}

type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

type EmailConfig struct {
	Provider    string         `koanf:"provider"`
	FromName    string         `koanf:"from_name"`
	FromAddress string         `koanf:"from_address"`
	SMTP        SMTPConfig     `koanf:"smtp"`
	SendGrid    SendGridConfig `koanf:"sendgrid"`
	Brevo       BrevoConfig    `koanf:"brevo"`
}

type SMTPConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	UseTLS   bool   `koanf:"use_tls"`
}

type SendGridConfig struct {
	APIKey string `koanf:"api_key"`
}

type BrevoConfig struct {
	APIKey string `koanf:"api_key"`
}

type AuthConfig struct {
	JWTSecret         string        `koanf:"jwt_secret"`
	AccessTokenExpiry time.Duration `koanf:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `koanf:"refresh_token_expiry"`
	RBACModelPath     string        `koanf:"rbac_model"`
	RBACPolicyPath    string        `koanf:"rbac_policy"`
}

type RateLimitConfig struct {
	Enabled bool   `koanf:"enabled"`
	Rate    string `koanf:"rate"`
	Store   string `koanf:"store"`
}

type CacheConfig struct {
	Driver string      `koanf:"driver"`
	Redis  RedisConfig `koanf:"redis"`
}

type RedisConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

type SecurityConfig struct {
	HSTS                  bool   `koanf:"hsts"`
	HSTSMaxAge            int    `koanf:"hsts_max_age"`
	FrameDeny             bool   `koanf:"frame_deny"`
	ContentTypeNosniff    bool   `koanf:"content_type_nosniff"`
	BrowserXSSFilter      bool   `koanf:"browser_xss_filter"`
	ContentSecurityPolicy string `koanf:"content_security_policy"`
	ReferrerPolicy        string `koanf:"referrer_policy"`
}

type StorageConfig struct {
	Driver string             `koanf:"driver"`
	Local  LocalStorageConfig `koanf:"local"`
	S3     S3Config           `koanf:"s3"`
}

type LocalStorageConfig struct {
	Path string `koanf:"path"`
}

type S3Config struct {
	Endpoint  string `koanf:"endpoint"`
	Bucket    string `koanf:"bucket"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Region    string `koanf:"region"`
	UseSSL    bool   `koanf:"use_ssl"`
}

type QueueConfig struct {
	Enabled     bool            `koanf:"enabled"`
	Concurrency int             `koanf:"concurrency"`
	Queues      map[string]int  `koanf:"queues"`
	Redis       QueueRedisConfig `koanf:"redis"`
}

type QueueRedisConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

type WebSocketConfig struct {
	Enabled bool `koanf:"enabled"`
}

type I18nConfig struct {
	DefaultLanguage string `koanf:"default_language"`
	LocalesDir      string `koanf:"locales_dir"`
}

type FeatureFlagConfig struct {
	Enabled    bool   `koanf:"enabled"`
	ConfigPath string `koanf:"config_path"`
}

type EncryptionConfig struct {
	Key string `koanf:"key"`
}

type SessionConfig struct {
	Driver         string `koanf:"driver"`
	Secret         string `koanf:"secret"`
	SessionName    string `koanf:"session_name"`
	FilesystemPath string `koanf:"filesystem_path"`
}

type ObservabilityConfig struct {
	MetricsEnabled bool   `koanf:"metrics_enabled"`
	TracingEnabled bool   `koanf:"tracing_enabled"`
	MetricsPath    string `koanf:"metrics_path"`
	ServiceName    string `koanf:"service_name"`
}

// LoadConfig loads configuration from config.yaml (if present), then overlays
// environment variables prefixed with GOFASTA_ (e.g., GOFASTA_DATABASE_HOST).
func LoadConfig() (*AppConfig, error) {
	k := koanf.New(".")

	if _, err := os.Stat("config.yaml"); err == nil {
		if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
			return nil, err
		}
	}

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
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
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
		cfg.Email.FromName = "App"
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
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "change-me-in-production"
	}
	if cfg.Auth.AccessTokenExpiry == 0 {
		cfg.Auth.AccessTokenExpiry = 15 * time.Minute
	}
	if cfg.Auth.RefreshTokenExpiry == 0 {
		cfg.Auth.RefreshTokenExpiry = 168 * time.Hour
	}
	if cfg.Auth.RBACModelPath == "" {
		cfg.Auth.RBACModelPath = "configs/rbac_model.conf"
	}
	if cfg.Auth.RBACPolicyPath == "" {
		cfg.Auth.RBACPolicyPath = "configs/rbac_policy.csv"
	}
	if cfg.RateLimit.Rate == "" {
		cfg.RateLimit.Rate = "100-S"
	}
	if cfg.RateLimit.Store == "" {
		cfg.RateLimit.Store = "memory"
	}
	if cfg.Cache.Driver == "" {
		cfg.Cache.Driver = "memory"
	}
	if cfg.Cache.Redis.Host == "" {
		cfg.Cache.Redis.Host = "localhost"
	}
	if cfg.Cache.Redis.Port == "" {
		cfg.Cache.Redis.Port = "6379"
	}
	if cfg.Security.HSTSMaxAge == 0 {
		cfg.Security.HSTSMaxAge = 31536000
	}
	if cfg.Security.ReferrerPolicy == "" {
		cfg.Security.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = "local"
	}
	if cfg.Storage.Local.Path == "" {
		cfg.Storage.Local.Path = "./uploads"
	}
	if cfg.Queue.Concurrency == 0 {
		cfg.Queue.Concurrency = 10
	}
	if cfg.Queue.Redis.Host == "" {
		cfg.Queue.Redis.Host = "localhost"
	}
	if cfg.Queue.Redis.Port == "" {
		cfg.Queue.Redis.Port = "6379"
	}
	if cfg.Queue.Redis.DB == 0 {
		cfg.Queue.Redis.DB = 1
	}
	if cfg.I18n.DefaultLanguage == "" {
		cfg.I18n.DefaultLanguage = "en"
	}
	if cfg.I18n.LocalesDir == "" {
		cfg.I18n.LocalesDir = "locales"
	}
	if cfg.FeatureFlag.ConfigPath == "" {
		cfg.FeatureFlag.ConfigPath = "configs/features.yaml"
	}
	if cfg.Session.Driver == "" {
		cfg.Session.Driver = "cookie"
	}
	if cfg.Session.Secret == "" {
		cfg.Session.Secret = "change-me-in-production-32bytes!"
	}
	if cfg.Session.SessionName == "" {
		cfg.Session.SessionName = "app_session"
	}
	if cfg.Session.FilesystemPath == "" {
		cfg.Session.FilesystemPath = "./sessions"
	}
	if cfg.Observability.MetricsPath == "" {
		cfg.Observability.MetricsPath = "/metrics"
	}
	if cfg.Observability.ServiceName == "" {
		cfg.Observability.ServiceName = "app"
	}
}
