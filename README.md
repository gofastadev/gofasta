# Gofasta

[![CI](https://github.com/gofastadev/gofasta/actions/workflows/ci.yml/badge.svg)](https://github.com/gofastadev/gofasta/actions/workflows/ci.yml) [![CodeQL](https://github.com/gofastadev/gofasta/actions/workflows/codeql.yml/badge.svg)](https://github.com/gofastadev/gofasta/actions/workflows/codeql.yml) [![codecov](https://codecov.io/gh/gofastadev/gofasta/graph/badge.svg)](https://codecov.io/gh/gofastadev/gofasta) [![Go Reference](https://pkg.go.dev/badge/github.com/gofastadev/gofasta.svg)](https://pkg.go.dev/github.com/gofastadev/gofasta) [![Go Report Card](https://goreportcard.com/badge/github.com/gofastadev/gofasta)](https://goreportcard.com/report/github.com/gofastadev/gofasta) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/github/go-mod/go-version/gofastadev/gofasta)](https://github.com/gofastadev/gofasta/blob/main/go.mod) [![Release](https://img.shields.io/github/v/release/gofastadev/gofasta)](https://github.com/gofastadev/gofasta/releases)

Gofasta is a Go backend toolkit. This repo is the **library** your project imports — every package under `pkg/*` is self-contained and exposes a clear interface for one concern (auth, caching, database setup, email, middleware, observability, …) so you focus on business logic.

To create a new project, use the [Gofasta CLI](https://github.com/gofastadev/cli) — `gofasta new <project>` scaffolds a working project with this library wired in.

> **Status:** pre-1.0 (`v0.1.x`). The API may change before `v1.0.0` — pin the exact version in your `go.mod` and read release notes when upgrading.

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

<!-- gofasta:begin pkg-table:core -->
| Package | What it does |
|---------|-------------|
| `pkg/config` | Package config loads configuration from config.yaml and environment variables. It provides LoadConfig() to get an AppConfig struct with all settings, and SetupDB() to create a database connection. Supports Postgres, MySQL, SQLite, SQL Server, and ClickHouse. |
| `pkg/logger` | Package logger creates a structured logger using Go's slog package. Output format (text or JSON) and log level are configurable. |
| `pkg/errors` | Package apperrors defines application error types (NotFound, BadRequest, Conflict, Internal, etc.) and maps them to HTTP status codes. It also provides a GraphQL error presenter that formats errors for GraphQL responses. |
| `pkg/models` | Package models provides BaseModelImpl, a struct you embed in your domain models to get standard fields: ID (UUID), CreatedAt, UpdatedAt, DeletedAt, RecordVersion, IsActive, IsDeletable. It includes a GORM BeforeCreate hook that auto-generates UUIDs and timestamps. |
| `pkg/types` | Package types provides common DTO (Data Transfer Object) types used across gofasta applications: TPaginationInputDto, TSortingInputDto, TPaginationObjectDto, TCommonAPIErrorDto, TCommonResponseDto, and the SortOrientation enum. |
| `pkg/database` | Package database provides transaction propagation for the repository layer. |
<!-- gofasta:end pkg-table:core -->

### HTTP & API

<!-- gofasta:begin pkg-table:http -->
| Package | What it does |
|---------|-------------|
| `pkg/httputil` | Package httputil provides three helpers for HTTP handlers: Bind() parses and validates request bodies, Handle() wraps handler functions that return errors into standard http.Handler, and OK()/Created()/JSON() write JSON responses. |
| `pkg/middleware` | Package middleware is a collection of HTTP middleware: request logging, panic recovery, CORS, security headers (HSTS, CSP, X-Frame-Options), rate limiting, request ID generation, and content-type validation. Compose them with Chain(). |
| `pkg/health` | Package health provides a health check controller with three endpoints: /health (basic liveness), /health/live (process alive), and /health/ready (checks database and cache connectivity). |
| `pkg/graphql` | Shared GraphQL schema fragments (`.gql` files) for pagination, sorting, errors, and a default health Query/Mutation. Consumed by gqlgen at build time, not imported as Go code — keeps REST and GraphQL response shapes aligned. |
<!-- gofasta:end pkg-table:http -->

### Authentication & Security

<!-- gofasta:begin pkg-table:security -->
| Package | What it does |
|---------|-------------|
| `pkg/auth` | Package auth provides JWT token management (generate, validate, refresh) and RBAC (Role-Based Access Control) using Casbin. It includes middleware for extracting JWT from requests and enforcing role-based permissions. |
| `pkg/encryption` | Package encryption provides AES-256-GCM encryption and decryption for sensitive data at rest. |
| `pkg/session` | Package session provides server-side session management wrapping gorilla/sessions. It supports cookie-based and filesystem-based session stores. |
<!-- gofasta:end pkg-table:security -->

### Data & Storage

<!-- gofasta:begin pkg-table:data -->
| Package | What it does |
|---------|-------------|
| `pkg/cache` | Package cache provides a caching interface with two implementations: in-memory and Redis. Methods: Get, Set, Delete, Exists, Flush, Close. |
| `pkg/storage` | Package storage provides a file storage abstraction with two backends: local filesystem and S3-compatible storage (AWS S3, MinIO, etc.). Methods: Upload, Download, Delete, URL. |
| `pkg/seeds` | Package seeds provides a seeder registry for populating databases with test/development data. Call Register() to add seed functions and RunAll() to execute them. |
<!-- gofasta:end pkg-table:data -->

### Communication

<!-- gofasta:begin pkg-table:communication -->
| Package | What it does |
|---------|-------------|
| `pkg/mailer` | Package mailer sends email with three providers: SMTP, SendGrid, and Brevo (Sendinblue). It includes a template renderer that processes Go HTML templates for email bodies. |
| `pkg/slack` | Package slack provides outbound Slack messaging primitives. It is the counterpart to pkg/mailer for chat: a SlackSender interface and one or more concrete implementations selected by configuration. |
| `pkg/whatsapp` | Package whatsapp provides outbound WhatsApp messaging primitives. It is the chat counterpart to pkg/mailer and pkg/slack. |
| `pkg/push` | Package push provides outbound mobile push notification primitives. It is the mobile counterpart to pkg/mailer, pkg/slack and pkg/whatsapp. |
| `pkg/notify` | Package notify is a multi-channel notification orchestrator that delivers messages over email, SMS (Twilio), Slack, and a database channel that persists notifications to a table. |
| `pkg/websocket` | Package websocket provides WebSocket support with a hub that manages connections, rooms, and message broadcasting. |
<!-- gofasta:end pkg-table:communication -->

### Background Processing

<!-- gofasta:begin pkg-table:background -->
| Package | What it does |
|---------|-------------|
| `pkg/scheduler` | Package scheduler provides cron job scheduling using robfig/cron. Register jobs with cron expressions (6-field format with seconds). |
| `pkg/queue` | Package queue provides an async task queue backed by Redis using hibiken/asynq. Enqueue tasks for background processing with configurable concurrency. |
| `pkg/resilience` | Package resilience provides retry policies with exponential backoff using failsafe-go. Wrap unreliable operations with automatic retry logic. |
<!-- gofasta:end pkg-table:background -->

### Validation & i18n

<!-- gofasta:begin pkg-table:validation -->
| Package | What it does |
|---------|-------------|
| `pkg/validators` | Package validators wraps go-playground/validator for input validation. It provides AppValidator with ValidateStruct() that returns structured error DTOs, plus common validators: UUID validation, record existence checks, URL validation, and record deletability checks. Projects register their own custom validators on top. |
| `pkg/i18n` | Package i18n provides internationalization using go-i18n. It loads translation files from a locales/ directory and translates messages based on the request's language. |
| `pkg/utils` | Package utils provides small standalone helpers used across the library: string-case conversion, slice helpers, time formatting, and similar. Imported as needed; nothing here depends on the rest of the library. |
<!-- gofasta:end pkg-table:validation -->

### Observability

<!-- gofasta:begin pkg-table:observability -->
| Package | What it does |
|---------|-------------|
| `pkg/observability` | Package observability provides Prometheus metrics (request count, duration, in-flight requests) exposed at /metrics, and distributed tracing with OpenTelemetry. Both are available as HTTP middleware. |
| `pkg/featureflag` | Package featureflag wraps the OpenFeature Go SDK so callers can evaluate feature flags through a stable interface while remaining free to swap the underlying provider (in-memory, Flagd, LaunchDarkly, go-feature-flag, ConfigCat, or a custom implementation) at application startup via openfeature.SetProvider. |
<!-- gofasta:end pkg-table:observability -->

### `pkg/config.JSONSchema()`

`pkg/config` additionally exports a `JSONSchema()` function that reflects over `AppConfig` and returns a JSON Schema (Draft 7) describing the complete config shape. Fields, types, durations, enums (via `validate:"oneof=..."` tags), and required fields (via `validate:"required"`) are all honored. Used by the scaffold's `cmd/schema` helper binary, which the CLI's `gofasta config schema` command shells out to — keeps the emitted schema always in sync with the exact library version pinned by the project.

### Testing

<!-- gofasta:begin pkg-table:testing -->
| Package | What it does |
|---------|-------------|
| `pkg/testutil/testdb` | Package testdb provides test database setup helpers using testcontainers. |
<!-- gofasta:end pkg-table:testing -->

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

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT
