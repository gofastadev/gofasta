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
	Server        ServerConfig        `koanf:"server"`
	Database      DatabaseConfig      `koanf:"database"`
	GraphQL       GraphQLConfig       `koanf:"graphql"`
	Log           LogConfig           `koanf:"log"`
	Email         EmailConfig         `koanf:"email"`
	Slack         SlackConfig         `koanf:"slack"`
	WhatsApp      WhatsAppConfig      `koanf:"whatsapp"`
	Jobs          []JobConfig         `koanf:"jobs"`
	Auth          AuthConfig          `koanf:"auth"`
	RateLimit     RateLimitConfig     `koanf:"rate_limit"`
	Cache         CacheConfig         `koanf:"cache"`
	Security      SecurityConfig      `koanf:"security"`
	Storage       StorageConfig       `koanf:"storage"`
	Queue         QueueConfig         `koanf:"queue"`
	WebSocket     WebSocketConfig     `koanf:"websocket"`
	I18n          I18nConfig          `koanf:"i18n"`
	FeatureFlag   FeatureFlagConfig   `koanf:"feature_flag"`
	Encryption    EncryptionConfig    `koanf:"encryption"`
	Session       SessionConfig       `koanf:"session"`
	Observability ObservabilityConfig `koanf:"observability"`
}

// JobConfig is a single cron-scheduled job entry.
type JobConfig struct {
	Name     string `koanf:"name"`
	Schedule string `koanf:"schedule"`
	Enabled  bool   `koanf:"enabled"`
}

// ServerConfig configures the HTTP server.
//
// Host is the bind address. The default is 127.0.0.1 (loopback) so a
// freshly-scaffolded project runs cleanly on macOS without the firewall
// prompting for permission on every Air rebuild — macOS only fires that
// dialog for processes listening on non-loopback interfaces, so binding
// to 127.0.0.1 sidesteps it entirely. For production / containers, set
// the host to 0.0.0.0 (or the desired interface) via env var, e.g.
// MYAPP_SERVER_HOST=0.0.0.0. The scaffold's Dockerfile and compose
// manifests already do this.
type ServerConfig struct {
	Host            string        `koanf:"host"`
	Port            string        `koanf:"port"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
	AllowedOrigins  []string      `koanf:"allowed_origins"`
}

// DatabaseConfig configures the GORM database connection.
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

// GraphQLConfig configures the GraphQL routes.
type GraphQLConfig struct {
	PlaygroundRoute string `koanf:"playground_route"`
	GeneralRoute    string `koanf:"general_route"`
}

// LogConfig configures the logger.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// EmailConfig configures the email subsystem and its providers.
type EmailConfig struct {
	Provider    string         `koanf:"provider"`
	FromName    string         `koanf:"from_name"`
	FromAddress string         `koanf:"from_address"`
	SMTP        SMTPConfig     `koanf:"smtp"`
	SendGrid    SendGridConfig `koanf:"sendgrid"`
	Brevo       BrevoConfig    `koanf:"brevo"`
}

// SMTPConfig configures the SMTP email provider.
type SMTPConfig struct {
	Host               string `koanf:"host"`
	Port               int    `koanf:"port"`
	Username           string `koanf:"username"`
	Password           string `koanf:"password"`
	UseTLS             bool   `koanf:"use_tls"`
	InsecureSkipVerify bool   `koanf:"insecure_skip_verify"`
}

// SendGridConfig configures the SendGrid email provider.
type SendGridConfig struct {
	APIKey string `koanf:"api_key"`
}

// BrevoConfig configures the Brevo (ex-Sendinblue) email provider.
type BrevoConfig struct {
	APIKey string `koanf:"api_key"`
}

// SlackConfig configures outbound Slack messaging via pkg/slack.
//
// Provider selects the delivery mode:
//   - ""        → disabled (no-op sender; Send returns "not configured")
//   - "webhook" → POST to a single Incoming Webhook URL
//   - "api"     → use a bot token against api.slack.com (chat.postMessage,
//     files.uploadV2)
//
// Token + WebhookURL are mutually independent — populate the one your
// chosen provider needs.
type SlackConfig struct {
	Provider   string `koanf:"provider"`
	BotToken   string `koanf:"bot_token"`
	WebhookURL string `koanf:"webhook_url"`
	// SigningSecret is consumed by app code that handles INBOUND
	// interactivity callbacks (button clicks, slash commands). pkg/slack
	// itself does outbound only — but the value lives here because the
	// rest of slack config does, so the inbound handler can reach it
	// via the same config service.
	SigningSecret string `koanf:"signing_secret"`
}

// WhatsAppConfig configures outbound WhatsApp messaging via pkg/whatsapp.
//
// Provider selects the delivery backend:
//   - ""         → disabled
//   - "ultramsg" → UltraMsg instance API
//   - "twilio"   → Twilio WhatsApp Business
//   - "meta"     → Meta WhatsApp Cloud API
//
// Each provider reads its own subsection. Switching providers is a
// config-only change (plus restart).
type WhatsAppConfig struct {
	Provider string                 `koanf:"provider"`
	UltraMsg WhatsAppUltraMsgConfig `koanf:"ultramsg"`
	Twilio   WhatsAppTwilioConfig   `koanf:"twilio"`
	Meta     WhatsAppMetaConfig     `koanf:"meta"`
}

// WhatsAppUltraMsgConfig — UltraMsg uses an instance-scoped API. URL
// shape: https://api.ultramsg.com/instance{ID}/. Token authenticates
// every call; instance + token together identify a single WhatsApp
// session.
type WhatsAppUltraMsgConfig struct {
	BaseURL    string `koanf:"base_url"`    // e.g. "https://api.ultramsg.com"
	InstanceID string `koanf:"instance_id"` // e.g. "instance60301"
	Token      string `koanf:"token"`
}

// WhatsAppTwilioConfig — Twilio Programmable Messaging WhatsApp. Auth
// is HTTP Basic (Account SID + Auth Token). Sender numbers are
// `whatsapp:+E164` and must be approved in the Twilio console.
type WhatsAppTwilioConfig struct {
	AccountSID string `koanf:"account_sid"`
	AuthToken  string `koanf:"auth_token"`
	FromNumber string `koanf:"from_number"` // e.g. "+14155238886"
}

// WhatsAppMetaConfig — Meta WhatsApp Cloud API (Graph). Bearer auth,
// per-WABA phone-number IDs. Templates and interactive messages are
// supported but you must have an approved WhatsApp Business Account.
type WhatsAppMetaConfig struct {
	AccessToken   string `koanf:"access_token"`
	PhoneNumberID string `koanf:"phone_number_id"`
	APIVersion    string `koanf:"api_version"` // e.g. "v20.0"; defaults to v20.0 when empty
}

// AuthConfig configures JWT + RBAC.
type AuthConfig struct {
	JWTSecret          string        `koanf:"jwt_secret"`
	AccessTokenExpiry  time.Duration `koanf:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `koanf:"refresh_token_expiry"`
	RBACModelPath      string        `koanf:"rbac_model"`
	RBACPolicyPath     string        `koanf:"rbac_policy"`
}

// RateLimitConfig configures the HTTP rate limiter.
type RateLimitConfig struct {
	Enabled bool   `koanf:"enabled"`
	Rate    string `koanf:"rate"`
	Store   string `koanf:"store"`
}

// CacheConfig configures the cache backend.
type CacheConfig struct {
	Driver string      `koanf:"driver"`
	Redis  RedisConfig `koanf:"redis"`
}

// RedisConfig describes a Redis connection.
type RedisConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

// SecurityConfig toggles security middleware options.
type SecurityConfig struct {
	HSTS                  bool     `koanf:"hsts"`
	HSTSMaxAge            int      `koanf:"hsts_max_age"`
	FrameDeny             bool     `koanf:"frame_deny"`
	ContentTypeNosniff    bool     `koanf:"content_type_nosniff"`
	BrowserXSSFilter      bool     `koanf:"browser_xss_filter"`
	ContentSecurityPolicy string   `koanf:"content_security_policy"`
	ReferrerPolicy        string   `koanf:"referrer_policy"`
	AllowedHosts          []string `koanf:"allowed_hosts"`
}

// StorageConfig configures the file storage backend.
type StorageConfig struct {
	Driver string             `koanf:"driver"`
	Local  LocalStorageConfig `koanf:"local"`
	S3     S3Config           `koanf:"s3"`
}

// LocalStorageConfig configures local-filesystem storage.
type LocalStorageConfig struct {
	Path string `koanf:"path"`
}

// S3Config configures S3-compatible object storage.
type S3Config struct {
	Endpoint  string `koanf:"endpoint"`
	Bucket    string `koanf:"bucket"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Region    string `koanf:"region"`
	UseSSL    bool   `koanf:"use_ssl"`
}

// QueueConfig configures the asynq-backed task queue.
type QueueConfig struct {
	Enabled     bool             `koanf:"enabled"`
	Concurrency int              `koanf:"concurrency"`
	Queues      map[string]int   `koanf:"queues"`
	Redis       QueueRedisConfig `koanf:"redis"`
}

// QueueRedisConfig describes the Redis instance dedicated to the queue.
type QueueRedisConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

// WebSocketConfig toggles the WebSocket hub.
type WebSocketConfig struct {
	Enabled bool `koanf:"enabled"`
}

// I18nConfig configures the translations loader.
type I18nConfig struct {
	DefaultLanguage string `koanf:"default_language"`
	LocalesDir      string `koanf:"locales_dir"`
}

// FeatureFlagConfig configures the feature-flag provider.
type FeatureFlagConfig struct {
	Enabled    bool   `koanf:"enabled"`
	ConfigPath string `koanf:"config_path"`
}

// EncryptionConfig holds the at-rest encryption key.
type EncryptionConfig struct {
	Key string `koanf:"key"`
}

// SessionConfig configures the session store.
type SessionConfig struct {
	Driver         string `koanf:"driver"`
	Secret         string `koanf:"secret"`
	SessionName    string `koanf:"session_name"`
	FilesystemPath string `koanf:"filesystem_path"`
}

// ObservabilityConfig toggles metrics and tracing.
type ObservabilityConfig struct {
	MetricsEnabled bool   `koanf:"metrics_enabled"`
	TracingEnabled bool   `koanf:"tracing_enabled"`
	MetricsPath    string `koanf:"metrics_path"`
	ServiceName    string `koanf:"service_name"`
}

// LoadConfig loads configuration from config.yaml (if present), then overlays
// environment variables prefixed with GOFASTA_ (e.g., GOFASTA_DATABASE_HOST).
// Equivalent to LoadConfigWithPrefix("GOFASTA_") — kept for backwards
// compatibility. Callers that want a project-specific env var prefix
// (e.g. MYAPP_DATABASE_HOST) should use LoadConfigWithPrefix directly.
func LoadConfig() (*AppConfig, error) {
	return LoadConfigWithPrefix("GOFASTA_")
}

// LoadConfigWithPrefix is the same as LoadConfig but reads environment
// variables using the given prefix. The prefix is stripped, lowercased,
// and underscores become dots before the key is looked up on koanf — so
// prefix="MYAPP_" maps MYAPP_DATABASE_HOST → database.host.
//
// Projects scaffolded by `gofasta new` call this with their project-
// specific prefix (e.g. "MYAPP_") so env vars match the project name in
// .env files, Dockerfiles, CI configs, and k8s manifests. This keeps the
// generated project's configuration surface named after the project
// itself, not the toolkit — and lets a developer swap pkg/config for a
// different loader later without having to rename every env var across
// their infrastructure files.
//
// An empty prefix is valid and causes every env var to be considered —
// typically you don't want this because unrelated vars like PATH would
// be parsed as config keys. Pass a non-empty prefix like "GOFASTA_" or
// "MYAPP_".
func LoadConfigWithPrefix(prefix string) (*AppConfig, error) {
	k := koanf.New(".")

	if _, err := os.Stat("config.yaml"); err == nil {
		if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
			return nil, err
		}
	}

	_ = k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.ReplaceAll(
			strings.ToLower(strings.TrimPrefix(s, prefix)),
			"_", ".",
		)
	}), nil)

	cfg := &AppConfig{}
	_ = k.Unmarshal("", cfg)

	applyDefaults(cfg)
	return cfg, nil
}

// applyDefaults fills in zero-valued fields of cfg with sane defaults for
// every subsystem. This is intentionally a flat sequence of cheap zero-checks
// — refactoring it into helpers would add indirection without reducing total
// cognitive load.
//
//nolint:gocognit,gocyclo // flat sequence of field defaults; see doc above.
func applyDefaults(cfg *AppConfig) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
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
