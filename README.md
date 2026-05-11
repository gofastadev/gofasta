# Gofasta

[![CI](https://github.com/gofastadev/gofasta/actions/workflows/ci.yml/badge.svg)](https://github.com/gofastadev/gofasta/actions/workflows/ci.yml) [![CodeQL](https://github.com/gofastadev/gofasta/actions/workflows/codeql.yml/badge.svg)](https://github.com/gofastadev/gofasta/actions/workflows/codeql.yml) [![codecov](https://codecov.io/gh/gofastadev/gofasta/graph/badge.svg)](https://codecov.io/gh/gofastadev/gofasta) [![Go Reference](https://pkg.go.dev/badge/github.com/gofastadev/gofasta.svg)](https://pkg.go.dev/github.com/gofastadev/gofasta) [![Go Report Card](https://goreportcard.com/badge/github.com/gofastadev/gofasta)](https://goreportcard.com/report/github.com/gofastadev/gofasta) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/github/go-mod/go-version/gofastadev/gofasta)](https://github.com/gofastadev/gofasta/blob/main/go.mod) [![Release](https://img.shields.io/github/v/release/gofastadev/gofasta)](https://github.com/gofastadev/gofasta/releases)

Gofasta is a Go backend toolkit. This repo is the **library** your project imports — every package under `pkg/*` is self-contained and exposes a clear interface for one concern (auth, caching, database setup, email, middleware, observability, …) so you focus on business logic.

To create a new project, use the [Gofasta CLI](https://github.com/gofastadev/cli) — `gofasta new <project>` scaffolds a working project with this library wired in.

## The Gofasta project

Gofasta is split across three independent repositories. Each has its own release cycle and `go.mod` / `package.json`.

| Repo | Role |
|------|------|
| [`gofastadev/gofasta`](https://github.com/gofastadev/gofasta) | **You are here.** The library — every `pkg/*` your project imports. |
| [`gofastadev/cli`](https://github.com/gofastadev/cli) | The `gofasta` binary — `gofasta new`, code generation, and the dev loop. |
| [`gofastadev/website`](https://github.com/gofastadev/website) | The docs site at **[gofasta.dev](https://gofasta.dev)**. |

For full documentation — guides, CLI reference, and a per-package API reference for everything below — visit **[gofasta.dev](https://gofasta.dev)**. This README covers library installation and the package overview; everything else is on the website.

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
| `pkg/graphql` | Shared GraphQL schema fragments (`.gql` files) for pagination, sorting, errors, and a default health Query/Mutation. Consumed by gqlgen at build time, not imported as Go code — keeps REST and GraphQL response shapes aligned. |

### Authentication & Security

| Package | What it does |
|---------|-------------|
| `pkg/auth` | JWT token management (generate, validate, refresh) and RBAC (Role-Based Access Control) using Casbin. Includes middleware for extracting JWT from requests and enforcing role-based permissions. |
| `pkg/encryption` | AES-256-GCM encryption and decryption for sensitive data at rest. |
| `pkg/session` | Server-side session management wrapping gorilla/sessions. Supports cookie-based and filesystem-based session stores. |

### Data & Storage

| Package | What it does |
|---------|-------------|
| `pkg/cache` | A caching interface with two implementations: in-memory and Redis. Methods: `Get`, `Set`, `Delete`, `Exists`, `Flush`, `Close`. |
| `pkg/storage` | File storage abstraction with two backends: local filesystem and S3-compatible storage (AWS S3, MinIO, etc.). Methods: `Upload`, `Download`, `Delete`, `URL`. |
| `pkg/seeds` | A seeder registry for populating databases with test/development data. Call `Register()` to add seed functions and `RunAll()` to execute them. |

### Communication

| Package | What it does |
|---------|-------------|
| `pkg/mailer` | Email sending with three providers: SMTP, SendGrid, and Brevo (Sendinblue). Includes a template renderer that processes Go HTML templates for email bodies. |
| `pkg/slack` | Outbound Slack messaging. Two delivery modes: incoming-webhook and bot-token (`api`). Supports threading, Block Kit, and `files.uploadV2`. |
| `pkg/whatsapp` | Outbound WhatsApp messaging. Three providers: UltraMsg, Twilio Programmable Messaging, and Meta WhatsApp Cloud API. Media attachments, threaded replies. |
| `pkg/push` | Outbound mobile push notifications. Firebase Cloud Messaging ships in the standard build; the `Sender` interface is provider-agnostic for swap-in alternatives. |
| `pkg/notify` | A multi-channel notification orchestrator that delivers messages over email, SMS (Twilio), Slack, and a database channel that persists notifications to a table. |
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
| `pkg/validators` | Input validation package wrapping go-playground/validator. Provides `AppValidator` with `ValidateStruct()` that returns structured error DTOs. Includes common validators: UUID validation, record existence checks, URL validation, and record deletability checks. Projects register their own custom validators on top. |
| `pkg/i18n` | Internationalization using go-i18n. Loads translation files from a `locales/` directory and translates messages based on the request's language. |
| `pkg/utils` | Small standalone helpers used across the library — string-case conversion, slice helpers, time formatting, etc. Imported as needed; nothing here depends on the rest of the library. |

### Observability

| Package | What it does |
|---------|-------------|
| `pkg/observability` | Prometheus metrics (request count, duration, in-flight requests) exposed at `/metrics`, and distributed tracing with OpenTelemetry. Both available as HTTP middleware. |
| `pkg/featureflag` | Thin wrapper around the OpenFeature Go SDK for evaluating feature flags. Works with any OpenFeature provider — in-memory, Flagd, LaunchDarkly, go-feature-flag, or custom — registered via `openfeature.SetProvider`. |

### `pkg/config.JSONSchema()`

`pkg/config` additionally exports a `JSONSchema()` function that reflects over `AppConfig` and returns a JSON Schema (Draft 7) describing the complete config shape. Fields, types, durations, enums (via `validate:"oneof=..."` tags), and required fields (via `validate:"required"`) are all honored. Used by the scaffold's `cmd/schema` helper binary, which the CLI's `gofasta config schema` command shells out to — keeps the emitted schema always in sync with the exact library version pinned by the project.

### Testing

| Package | What it does |
|---------|-------------|
| `pkg/testutil/testdb` | Spins up a PostgreSQL container using testcontainers-go for integration tests. Call `SetupTestDB(t)` in your test and get a real `*gorm.DB` — the container is automatically cleaned up when the test finishes. **Requires Docker** on the machine running the tests (install: <https://docs.docker.com/get-docker/>). |

> **Per-package reference.** Every `pkg/*` has a dedicated page on [gofasta.dev/docs/api-reference](https://gofasta.dev/docs/api-reference) with types, functions, configuration, and usage examples. The table above is a one-line tour; the website is the source of truth for API detail.

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

## Maintenance and sustainability

Gofasta is currently maintained by one person; sustainability planning — release cadence, security SLOs, the solo-to-team transition, and the automation arc that retires manual steps as the project matures — is documented in the [release coordination repo](https://github.com/gofastadev/release), specifically in [`CADENCE.md`](https://github.com/gofastadev/release/blob/main/CADENCE.md), [`RELEASING.md`](https://github.com/gofastadev/release/blob/main/RELEASING.md), and [`COMMUNITY.md`](https://github.com/gofastadev/release/blob/main/COMMUNITY.md). Read those three together for the full picture.

## License

MIT
