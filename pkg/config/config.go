package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Development placeholders for security-critical secrets. applyDefaults fills
// empty secrets with these so a project boots out of the box during local
// development; ValidateSecrets then rejects them so they can never reach
// production unnoticed.
const (
	PlaceholderJWTSecret     = "change-me-in-production"
	PlaceholderSessionSecret = "change-me-in-production-32bytes!"
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
	Push          PushConfig          `koanf:"push"`
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
	Driver string `koanf:"driver"`
	// DSN, when set, is used verbatim and every field below except Driver and
	// the pool settings is ignored.
	//
	// The composed form cannot express driver options that live in the
	// connection string — PostgreSQL's search_path, a non-UTC TimeZone,
	// PgBouncer parameters, multi-host failover targets — and a project that
	// needs one of those otherwise has no way to reach the dialector. Setting
	// it here (typically from a DATABASE_URL environment variable) keeps that
	// escape hatch open without every project reimplementing SetupDB.
	DSN      string        `koanf:"dsn"`
	Host     string        `koanf:"host"`
	Port     string        `koanf:"port"`
	User     string        `koanf:"user"`
	Password string        `koanf:"password"`
	Name     string        `koanf:"name"`
	SSLMode  string        `koanf:"sslmode"`
	MaxIdle  int           `koanf:"max_idle"`
	MaxOpen  int           `koanf:"max_open"`
	MaxLife  time.Duration `koanf:"max_life"`
	// DegradedFallback, when true, lets the application start against an
	// in-memory stand-in database when the configured one is unreachable
	// at boot. The zero value (false) is the safe default: an unreachable
	// database refuses to start rather than silently serving requests
	// against a throwaway store that health checks would report as up.
	DegradedFallback bool `koanf:"degraded_fallback"`
}

// GraphQLConfig configures the GraphQL routes and server posture.
//
// PlaygroundEnabled and IntrospectionEnabled are plain booleans whose
// zero value is false (safe-by-default); the scaffold's config.yaml
// ships them as true for local development and its production manifests
// override them to false via env vars. ComplexityLimit bounds the cost
// of a single query (gqlgen's FixedComplexityLimit); 0 means "use the
// default" (200 via applyDefaults), negative disables the limit.
type GraphQLConfig struct {
	PlaygroundRoute      string `koanf:"playground_route"`
	GeneralRoute         string `koanf:"general_route"`
	PlaygroundEnabled    bool   `koanf:"playground_enabled"`
	IntrospectionEnabled bool   `koanf:"introspection_enabled"`
	ComplexityLimit      int    `koanf:"complexity_limit"`
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
	Resend      ResendConfig   `koanf:"resend"`
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

// ResendConfig configures the Resend email provider.
type ResendConfig struct {
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

// PushConfig configures outbound mobile push notifications via
// pkg/push.
//
// Provider selects the delivery backend:
//   - ""    → disabled (the noop sender — Send returns ErrNotConfigured)
//   - "fcm" → Firebase Cloud Messaging via the official Admin SDK
//
// Each provider reads its own subsection. Switching providers is a
// config-only change (plus restart).
type PushConfig struct {
	Provider string        `koanf:"provider"`
	FCM      PushFCMConfig `koanf:"fcm"`
}

// PushFCMConfig — credentials for Firebase Cloud Messaging.
//
// Provide ONE of CredentialsJSON (raw service-account JSON inline in
// env, useful for containerized deployments) or CredentialsFilePath
// (a file on disk). CredentialsJSON wins when both are set.
//
// ProjectID is read from the credentials JSON when not set
// explicitly; the override is mostly useful for tests against a fake
// project.
type PushFCMConfig struct {
	CredentialsJSON     string `koanf:"credentials_json"`
	CredentialsFilePath string `koanf:"credentials_file_path"`
	ProjectID           string `koanf:"project_id"`
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
	// Redis is the backing store when Store is "redis". Required in that
	// case: a rate limit is only global if every replica counts against the
	// same store, and a memory store silently multiplies the effective limit
	// by the replica count.
	Redis RedisConfig `koanf:"redis"`
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
	// Cookie attributes for the session cookie, mirroring pkg/session's
	// Options. Plain booleans (zero = false); the scaffold's config.yaml
	// ships cookie_http_only: true and its production manifests set
	// cookie_secure via env. cookie_same_site accepts lax|strict|none
	// ("" → lax). cookie_max_age is seconds (0 → 30 days).
	CookieSecure   bool   `koanf:"cookie_secure"`
	CookieHTTPOnly bool   `koanf:"cookie_http_only"`
	CookieSameSite string `koanf:"cookie_same_site"`
	CookieMaxAge   int    `koanf:"cookie_max_age"`
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
// variables using the given prefix. The prefix is stripped and the
// remainder lowercased, then resolved against the known config keys
// derived from AppConfig's koanf tags — so prefix="MYAPP_" maps
// MYAPP_DATABASE_HOST → database.host AND MYAPP_AUTH_JWT_SECRET →
// auth.jwt_secret (the leaf itself contains an underscore; a naive
// underscores-become-dots mapping would produce auth.jwt.secret and
// silently drop the override — the documented env override for every
// multi-word key was broken this way). Names that don't match a known
// key fall back to the legacy all-underscores-become-dots mapping, so
// dynamic/map-typed keys keep working.
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
		flat := strings.ToLower(strings.TrimPrefix(s, prefix))
		if dotted, ok := knownKoanfKeys()[flat]; ok {
			return dotted
		}
		return strings.ReplaceAll(flat, "_", ".")
	}), nil)

	cfg := &AppConfig{}
	_ = k.Unmarshal("", cfg)

	applyDefaults(cfg)
	return cfg, nil
}

// knownKoanfKeyIndex maps the underscore-flattened form of every dotted
// koanf leaf path in AppConfig ("auth_jwt_secret") to its dotted form
// ("auth.jwt_secret"). Built once by reflection over the koanf struct
// tags — reflection rather than koanf's Keys() because the YAML file may
// omit whole sections whose env overrides must still resolve, and
// applyDefaults runs after unmarshal so koanf never sees defaults.
var (
	knownKoanfKeysOnce sync.Once
	knownKoanfKeyIndex map[string]string
)

func knownKoanfKeys() map[string]string {
	knownKoanfKeysOnce.Do(func() {
		knownKoanfKeyIndex = map[string]string{}
		collectKoanfKeys(reflect.TypeOf(AppConfig{}), "", knownKoanfKeyIndex)
	})
	return knownKoanfKeyIndex
}

// collectKoanfKeys walks a config struct type, recording every leaf
// field's dotted path keyed by its underscore-flattened form. Nested
// config structs recurse; slices, maps, and scalar kinds (including
// time.Duration) are leaves — env override of composite values goes
// through the legacy mapping.
func collectKoanfKeys(t reflect.Type, prefix string, out map[string]string) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("koanf")
		if tag == "" || tag == "-" {
			continue
		}
		dotted := tag
		if prefix != "" {
			dotted = prefix + "." + tag
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			collectKoanfKeys(ft, dotted, out)
			continue
		}
		out[strings.ReplaceAll(dotted, ".", "_")] = dotted
	}
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
	if cfg.GraphQL.ComplexityLimit == 0 {
		cfg.GraphQL.ComplexityLimit = 200
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
		cfg.Auth.JWTSecret = PlaceholderJWTSecret
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
		cfg.Session.Secret = PlaceholderSessionSecret
	}
	if cfg.Session.SessionName == "" {
		cfg.Session.SessionName = "app_session"
	}
	if cfg.Session.FilesystemPath == "" {
		cfg.Session.FilesystemPath = "./sessions"
	}
	if cfg.Session.CookieSameSite == "" {
		cfg.Session.CookieSameSite = "lax"
	}
	if cfg.Session.CookieMaxAge == 0 {
		cfg.Session.CookieMaxAge = 30 * 24 * 60 * 60 // 30 days, in seconds
	}
	if cfg.Observability.MetricsPath == "" {
		cfg.Observability.MetricsPath = "/metrics"
	}
	if cfg.Observability.ServiceName == "" {
		cfg.Observability.ServiceName = "app"
	}
}

// ValidateSecrets returns an error when a security-critical secret is still
// empty or set to a known development placeholder. Generated projects call this
// at startup so a deployment that forgot to set real secrets fails loudly
// instead of silently shipping forgeable JWTs and session cookies signed with a
// publicly-known key.
//
// It is intentionally separate from LoadConfig: loading must still succeed with
// placeholder defaults so local development works out of the box, while the
// boot path decides when to enforce real secrets (typically always).
func (cfg *AppConfig) ValidateSecrets() error {
	var problems []string
	if cfg.Auth.JWTSecret == "" || cfg.Auth.JWTSecret == PlaceholderJWTSecret {
		problems = append(problems, "auth.jwt_secret")
	}
	if cfg.Session.Secret == "" || cfg.Session.Secret == PlaceholderSessionSecret {
		problems = append(problems, "session.secret")
	}
	if len(problems) > 0 {
		return fmt.Errorf(
			"insecure default secret(s) in use: %s — set real values in config.yaml "+
				"or via the corresponding *_AUTH_JWT_SECRET / *_SESSION_SECRET env vars",
			strings.Join(problems, ", "),
		)
	}
	return nil
}
