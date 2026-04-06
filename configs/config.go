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
	Storage   StorageConfig   `koanf:"storage"`
	Queue     QueueConfig     `koanf:"queue"`
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
// Supported drivers: postgres, mysql, sqlite, sqlserver, clickhouse
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

// AuthConfig holds JWT and RBAC settings.
type AuthConfig struct {
	JWTSecret         string        `koanf:"jwt_secret"`
	AccessTokenExpiry time.Duration `koanf:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `koanf:"refresh_token_expiry"`
	RBACModelPath     string        `koanf:"rbac_model"`
	RBACPolicyPath    string        `koanf:"rbac_policy"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Enabled bool   `koanf:"enabled"`
	Rate    string `koanf:"rate"`
	Store   string `koanf:"store"`
}

// CacheConfig holds caching settings.
type CacheConfig struct {
	Driver string      `koanf:"driver"`
	Redis  RedisConfig `koanf:"redis"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

// SecurityConfig holds security header settings.
type SecurityConfig struct {
	HSTS                  bool   `koanf:"hsts"`
	HSTSMaxAge            int    `koanf:"hsts_max_age"`
	FrameDeny             bool   `koanf:"frame_deny"`
	ContentTypeNosniff    bool   `koanf:"content_type_nosniff"`
	BrowserXSSFilter      bool   `koanf:"browser_xss_filter"`
	ContentSecurityPolicy string `koanf:"content_security_policy"`
	ReferrerPolicy        string `koanf:"referrer_policy"`
}

// StorageConfig holds file storage settings.
type StorageConfig struct {
	Driver string             `koanf:"driver"`
	Local  LocalStorageConfig `koanf:"local"`
	S3     S3Config           `koanf:"s3"`
}

// LocalStorageConfig holds local filesystem storage settings.
type LocalStorageConfig struct {
	Path string `koanf:"path"`
}

// S3Config holds S3-compatible storage settings.
type S3Config struct {
	Endpoint  string `koanf:"endpoint"`
	Bucket    string `koanf:"bucket"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Region    string `koanf:"region"`
	UseSSL    bool   `koanf:"use_ssl"`
}

// QueueConfig holds async task queue settings.
type QueueConfig struct {
	Enabled     bool            `koanf:"enabled"`
	Concurrency int             `koanf:"concurrency"`
	Queues      map[string]int  `koanf:"queues"`
	Redis       QueueRedisConfig `koanf:"redis"`
}

// QueueRedisConfig holds Redis settings for the task queue.
type QueueRedisConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
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
	// Auth defaults
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
	// Rate limit defaults
	if cfg.RateLimit.Rate == "" {
		cfg.RateLimit.Rate = "100-S"
	}
	if cfg.RateLimit.Store == "" {
		cfg.RateLimit.Store = "memory"
	}
	// Cache defaults
	if cfg.Cache.Driver == "" {
		cfg.Cache.Driver = "memory"
	}
	if cfg.Cache.Redis.Host == "" {
		cfg.Cache.Redis.Host = "localhost"
	}
	if cfg.Cache.Redis.Port == "" {
		cfg.Cache.Redis.Port = "6379"
	}
	// Security defaults
	if cfg.Security.HSTSMaxAge == 0 {
		cfg.Security.HSTSMaxAge = 31536000
	}
	if cfg.Security.ReferrerPolicy == "" {
		cfg.Security.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
	// Storage defaults
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = "local"
	}
	if cfg.Storage.Local.Path == "" {
		cfg.Storage.Local.Path = "./uploads"
	}
	// Queue defaults
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
}
