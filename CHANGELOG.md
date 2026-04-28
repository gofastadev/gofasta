# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
