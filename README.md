# Gofasta

[![CI](https://github.com/gofastadev/gofasta/actions/workflows/ci.yml/badge.svg)](https://github.com/gofastadev/gofasta/actions/workflows/ci.yml) [![CodeQL](https://github.com/gofastadev/gofasta/actions/workflows/codeql.yml/badge.svg)](https://github.com/gofastadev/gofasta/actions/workflows/codeql.yml) [![codecov](https://codecov.io/gh/gofastadev/gofasta/graph/badge.svg)](https://codecov.io/gh/gofastadev/gofasta) [![Go Reference](https://pkg.go.dev/badge/github.com/gofastadev/gofasta.svg)](https://pkg.go.dev/github.com/gofastadev/gofasta) [![Go Report Card](https://goreportcard.com/badge/github.com/gofastadev/gofasta)](https://goreportcard.com/report/github.com/gofastadev/gofasta) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/github/go-mod/go-version/gofastadev/gofasta)](https://github.com/gofastadev/gofasta/blob/main/go.mod) [![Release](https://img.shields.io/github/v/release/gofastadev/gofasta)](https://github.com/gofastadev/gofasta/releases)

Gofasta is a Go library that provides production-ready building blocks for backend services. It handles the infrastructure plumbing — authentication, caching, database setup, email, logging, middleware, and more — so you can focus on writing business logic.

This repo is the **library** that your project imports. To create a new project, use the [Gofasta CLI](https://github.com/gofastadev/cli).

## Quick Start

```bash
# Install the CLI tool
go install github.com/gofastadev/cli/cmd/gofasta@latest

# Create a new project
gofasta new myapp
cd myapp

# Start developing
make dev
```

Your project's `go.mod` will contain:

```
require github.com/gofastadev/gofasta v0.1.0
```

You then import packages like this in your code:

```go
import (
    "github.com/gofastadev/gofasta/pkg/config"
    "github.com/gofastadev/gofasta/pkg/middleware"
    "github.com/gofastadev/gofasta/pkg/errors"
)
```

## What's in the Box

Every package lives under `pkg/`. Here's what each one does:

### Core Infrastructure

| Package | What it does |
|---------|-------------|
| `pkg/config` | Loads configuration from `config.yaml` and environment variables. Provides `LoadConfig()` to get an `AppConfig` struct with all settings, and `SetupDB()` to create a database connection. Supports Postgres, MySQL, SQLite, SQL Server, and ClickHouse. |
| `pkg/logger` | Creates a structured logger using Go's `slog` package. Configurable output format (text or JSON) and log level. |
| `pkg/errors` | Defines application error types (`NotFound`, `BadRequest`, `Conflict`, `Internal`, etc.) and maps them to HTTP status codes. Also provides a GraphQL error presenter that formats errors for GraphQL responses. |
| `pkg/models` | Provides `BaseModelImpl` — a struct you embed in your domain models to get standard fields: `ID` (UUID), `CreatedAt`, `UpdatedAt`, `DeletedAt`, `RecordVersion`, `IsActive`, `IsDeletable`. Includes a GORM `BeforeCreate` hook that auto-generates UUIDs and timestamps. |
| `pkg/types` | Common DTO (Data Transfer Object) types used across projects: `TPaginationInputDto`, `TSortingInputDto`, `TPaginationObjectDto`, `TCommonAPIErrorDto`, `TCommonResponseDto`, and the `SortOrientation` enum. |

### HTTP & API

| Package | What it does |
|---------|-------------|
| `pkg/httputil` | Three helpers for HTTP handlers: `Bind()` parses and validates request bodies, `Handle()` wraps handler functions that return errors into standard `http.Handler`, and `OK()`/`Created()`/`JSON()` write JSON responses. |
| `pkg/middleware` | A collection of HTTP middleware: request logging, panic recovery, CORS, security headers (HSTS, CSP, X-Frame-Options), rate limiting, request ID generation, and content-type validation. Compose them with `middleware.Chain()`. |
| `pkg/health` | A health check controller with three endpoints: `/health` (basic liveness), `/health/live` (process alive), `/health/ready` (checks database and cache connectivity). |

### Authentication & Security

| Package | What it does |
|---------|-------------|
| `pkg/auth` | JWT token management (generate, validate, refresh) and RBAC (Role-Based Access Control) using Casbin. Includes middleware for extracting JWT from requests and enforcing role-based permissions. |
| `pkg/encryption` | AES-256-GCM encryption and decryption for sensitive data at rest. |
| `pkg/session` | Server-side session management wrapping gorilla/sessions. Supports cookie-based and filesystem-based session stores. |

### Data & Storage

| Package | What it does |
|---------|-------------|
| `pkg/cache` | A caching interface with two implementations: in-memory and Redis. Methods: `Get`, `Set`, `Delete`, `Flush`, `Ping`. |
| `pkg/storage` | File storage abstraction with two backends: local filesystem and S3-compatible storage (AWS S3, MinIO, etc.). Methods: `Upload`, `Download`, `Delete`, `URL`. |
| `pkg/seeds` | A seeder registry for populating databases with test/development data. Call `Register()` to add seed functions and `RunAll()` to execute them. |

### Communication

| Package | What it does |
|---------|-------------|
| `pkg/mailer` | Email sending with three providers: SMTP, SendGrid, and Brevo (Sendinblue). Includes a template renderer that processes Go HTML templates for email bodies. |
| `pkg/notify` | A notification system that sends messages through multiple channels: email, SMS (Twilio), Slack, and database (stores notifications in a table). |
| `pkg/websocket` | WebSocket support with a hub that manages connections, rooms, and message broadcasting. |

### Background Processing

| Package | What it does |
|---------|-------------|
| `pkg/scheduler` | Cron job scheduling using robfig/cron. Register jobs with cron expressions (6-field format with seconds). |
| `pkg/queue` | Async task queue backed by Redis using hibiken/asynq. Enqueue tasks for background processing with configurable concurrency. |
| `pkg/resilience` | Retry policies with exponential backoff using failsafe-go. Wrap unreliable operations with automatic retry logic. |

### Validation & i18n

| Package | What it does |
|---------|-------------|
| `pkg/validators` | Input validation framework wrapping go-playground/validator. Provides `AppValidator` with `ValidateStruct()` that returns structured error DTOs. Includes common validators: UUID validation, record existence checks, URL validation, and record deletability checks. Projects register their own custom validators on top. |
| `pkg/i18n` | Internationalization using go-i18n. Loads translation files from a `locales/` directory and translates messages based on the request's language. |

### Observability

| Package | What it does |
|---------|-------------|
| `pkg/observability` | Prometheus metrics (request count, duration, in-flight requests) exposed at `/metrics`, and distributed tracing with OpenTelemetry. Both available as HTTP middleware. |
| `pkg/featureflag` | Thin wrapper around the OpenFeature Go SDK for evaluating feature flags. Works with any OpenFeature provider — in-memory, Flagd, LaunchDarkly, go-feature-flag, or custom — registered via `openfeature.SetProvider`. |

### Testing

| Package | What it does |
|---------|-------------|
| `pkg/testutil/testdb` | Spins up a PostgreSQL container using testcontainers-go for integration tests. Call `SetupTestDB(t)` in your test and get a real `*gorm.DB` — the container is automatically cleaned up when the test finishes. |

## Architecture

This library follows a convention: every package is self-contained and exposes a clear interface. Your project imports only what it needs.

```
Your Project (myapp/)
    │
    ├── imports pkg/config      → loads config.yaml
    ├── imports pkg/middleware   → wraps your HTTP server
    ├── imports pkg/auth        → JWT + RBAC
    ├── imports pkg/models      → base model for your domain entities
    ├── imports pkg/errors      → structured error handling
    └── ... any other pkg/ you need
```

The library never imports your project code — the dependency flows one way.

## Database Support

The `pkg/config` package supports five database drivers through GORM:

| Driver | Config value | Notes |
|--------|-------------|-------|
| PostgreSQL | `postgres` | Default. Full feature support including CITEXT. |
| MySQL | `mysql` | UTF-8, timezone-aware. |
| SQLite | `sqlite` | File-based or `:memory:`. No connection pooling. |
| SQL Server | `sqlserver` | Microsoft SQL Server. |
| ClickHouse | `clickhouse` | Column-oriented, for analytics workloads. |

Set the driver in `config.yaml`:

```yaml
database:
  driver: postgres
  host: localhost
  port: "5432"
  user: myapp
  password: myapp
  name: myapp_dev
```

## License

MIT
