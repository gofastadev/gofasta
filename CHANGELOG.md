# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Driver-aware test databases** — `pkg/testutil/testdb` gains a package-level `Driver` selector: `postgres` (default), `mysql`, `sqlserver`, and `clickhouse` provision containers; `sqlite` runs in-process against a temp file with no Docker requirement. `Image` now overrides the selected driver's default image.
- **`config.BuildDSN` / `config.BuildMigrateURL`** — exported, per-driver connection-string builders with proper credential escaping (libpq quoting for postgres, `mysql.Config.FormatDSN`, `net/url` userinfo for sqlserver/clickhouse). `SetupDB` composes its dialector through them, and scaffolded projects can build their golang-migrate URL from the same source of truth. Passwords containing `@ : / % space '` no longer produce malformed connection strings.
- **`session.Options`** — both session-store constructors accept optional cookie attributes (`Secure`, `HttpOnly`, `SameSite`, `MaxAge`, `Path`). With no options, safe defaults apply: `HttpOnly: true`, `SameSite: Lax`, 30-day lifetime (gorilla v1.2.x's own defaults left session cookies JS-readable with no SameSite).
- **`health.NewControllerForDriver`** — readiness can now verify the live connection's dialector matches the configured driver, so an app that booted against a degraded fallback store reports `503` from `/health/ready` instead of "up".
- `GraphQLConfig` gains `playground_enabled`, `introspection_enabled`, and `complexity_limit` (default 200); `DatabaseConfig` gains `degraded_fallback` (default false — refuse to start when the database is unreachable); `SessionConfig` gains `cookie_secure`, `cookie_http_only`, `cookie_same_site`, `cookie_max_age`.

### Fixed
- **List filtering actually filters** — `utils.BuildQueryForAnyModel` previously acted only on `*string` values, so the dereferenced values every generated caller passes were silently ignored and list endpoints returned unfiltered tables. It now handles strings (case-insensitive substring match, driver-aware: `ILIKE` on postgres/clickhouse, `LOWER(...) LIKE LOWER(?)` on mysql/sqlserver, `LIKE` on sqlite) and exact-match equality for other types, takes a caller-supplied column allow-list (closing the raw column-name interpolation), and escapes LIKE metacharacters so `%`/`_` match literally. The previous `ILIKE`-everywhere form would have errored on every non-postgres driver.
- **Env-var overrides for multi-word config keys** — the env key mapper turned every `_` into `.`, so `MYAPP_AUTH_JWT_SECRET` resolved to the non-existent `auth.jwt.secret` and the documented override silently did nothing (same for `rate_limit.*`, `server.allowed_origins`, `session.session_name`, and every other multi-word leaf). Env names now resolve against the known config keys derived from `AppConfig`'s koanf tags, with the legacy mapping as fallback — all previously-working overrides keep working.
- **`/health/ready` no longer panics or lies** — a nil `*gorm.DB` or a connection-pool-less fallback DB reports `down` instead of nil-dereferencing, and the previously discarded `DB()` error is handled.

### Added (earlier in this cycle)
- **`config.JSONSchema()`** — emits a JSON Schema (Draft 7) describing the `AppConfig` struct by reflecting over its fields and struct tags. Honors `koanf:"name"` for field naming, `validate:"required"` for required fields, `validate:"oneof=a b c"` for enums, and `desc:"..."` for descriptions. Consumed by the CLI's new `gofasta config schema` command (which shells out to a project-local helper) so the emitted schema always matches the exact library version pinned in `go.mod`. Editor extensions (VS Code YAML, JetBrains) and CI pipelines can consume the output directly for autocomplete and config validation.
- Initial public release of the Gofasta library — a collection of independent Go packages under `pkg/*`
- `pkg/auth` — JWT authentication and RBAC authorization
- `pkg/cache` — In-memory and Redis cache backends
- `pkg/config` — Configuration loading via YAML and environment variables
- `pkg/encryption` — AES-256-GCM encryption helpers
- `pkg/errors` — Structured application errors with HTTP and GraphQL mapping
- `pkg/featureflag` — Runtime feature flag evaluation
- `pkg/health` — HTTP health check controller and GraphQL resolver
- `pkg/httputil` — Request binding, response helpers, and handler utilities
- `pkg/i18n` — Internationalization and translation support
- `pkg/logger` — Structured logging with `slog`
- `pkg/mailer` — Email delivery via SMTP, SendGrid, and Brevo
- `pkg/middleware` — CORS, recovery, and request logging middleware
- `pkg/models` — Base model with UUID, timestamps, and soft delete
- `pkg/notify` — Multi-channel notifications (email, SMS, push)
- `pkg/observability` — Prometheus metrics and OpenTelemetry tracing
- `pkg/queue` — Background task queue powered by Asynq
- `pkg/resilience` — Retry logic and circuit breaker
- `pkg/scheduler` — Cron-based job scheduling
- `pkg/seeds` — Database seeder registry
- `pkg/session` — Session management
- `pkg/storage` — Local filesystem and S3-compatible object storage
- `pkg/testutil` — Test helpers including a Dockerized database container
- `pkg/types` — Shared DTO types for pagination, sorting, and responses
- `pkg/utils` — String utilities, paginator, and search query builder
- `pkg/validators` — Input validation with custom rules and error messages
- `pkg/websocket` — WebSocket hub and client management
