# Gofasta — Missing Features Roadmap

## Already Built

- REST API (gorilla/mux) + GraphQL (gqlgen) dual API
- Google Wire compile-time dependency injection
- Repository pattern with interfaces
- Service layer with interfaces
- GORM multi-database support (postgres, mysql, sqlite, sqlserver, clickhouse)
- koanf configuration management (YAML + env vars)
- slog structured logging
- Custom error types (AppError) with HTTP status mapping
- GraphQL error sanitization
- Middleware chain (recovery, logging, CORS, request ID, content type, auth skeleton)
- Error-returning HTTP handler adapter
- Graceful shutdown with signal handling
- Context propagation through all layers
- Email service (SMTP, SendGrid, Brevo) with HTML templates
- Cron job scheduler (robfig/cron)
- Cobra CLI with modular scaffolding generators
- testify mocks + testcontainers-go
- Multi-platform deployment (Docker, Kubernetes, systemd, CI/CD for AWS/GCP/Azure/VPS)
- Hot reload with air
- Multi-database migration generation (per-driver SQL)
- Go version pinning (toolchain directive + .go-version)

---

## Tier 1: Must-Have

These are features that backend developers expect out of the box. Every major framework (Laravel, Rails, NestJS, Spring Boot, Django) ships with them.

### 1. JWT Authentication

- Token generation, validation, refresh token rotation, blacklisting
- Library: `github.com/golang-jwt/jwt/v5`
- Includes: login/register endpoints, token middleware, refresh flow
- Integrate with existing auth middleware skeleton in `pkg/middleware/auth.go`

### 2. Role-Based Authorization (RBAC/ABAC)

- Define who can access what — roles, permissions, policies
- Library: `github.com/casbin/casbin/v2`
- Supports: ACL, RBAC, ABAC, multi-tenancy
- Middleware that checks permissions before handler execution

### 3. Rate Limiting

- Per-client request throttling to prevent abuse
- Library: `github.com/ulule/limiter` (supports gorilla/mux, in-memory + Redis backends)
- Alternative: `golang.org/x/time/rate` (in-memory only, stdlib-adjacent)
- Add as middleware in the chain, configurable per-route or global

### 4. Caching Layer (Redis + In-Memory)

- Cache frequently accessed data to reduce DB load
- Libraries:
  - Redis client: `github.com/redis/go-redis/v9`
  - Cache abstraction: `github.com/eko/gocache` (chain cache: local + Redis, tags, metrics, invalidation)
  - In-memory fallback: `github.com/dgraph-io/ristretto` (high-performance LFU)
- Add Redis config to `config.yaml`, wire into DI container
- Provide a `CacheService` interface for services to use

### 5. API Documentation (OpenAPI/Swagger)

- Auto-generate interactive API docs from code comments
- Libraries:
  - `github.com/swaggo/swag` (parses Go comments into OpenAPI 3 spec)
  - `github.com/swaggo/http-swagger` (serves Swagger UI, works with gorilla/mux)
- Add `// @Summary`, `// @Param`, `// @Success` annotations to controllers
- Serve at `/swagger/` endpoint

### 6. Deep Health Checks

- Production-grade health endpoints for Kubernetes and load balancers
- Library: `github.com/alexliesenfeld/health` (composable checks, caching, timeout)
- Endpoints: `/health` (basic), `/healthz` (liveness), `/readyz` (readiness)
- Checks: database ping, Redis ping, disk space, memory usage, version/uptime
- Replace current basic health controller with comprehensive checks

### 7. Database Seeding

- Populate development/test databases with sample data
- Implement as Cobra command: `gofasta seed`
- Convention: seed files in `db/seeds/` as Go functions
- Support: `gofasta seed --fresh` (drop + migrate + seed)
- Similar to: Laravel seeders, Rails `db:seed`, Django fixtures

---

## Tier 2: Nice-to-Have

Significant developer experience improvements that differentiate the framework.

### 8. Request Binding Helper

- Auto-bind JSON body, query params, and form data into typed structs with validation
- Build a generic `Bind[T](r *http.Request) (T, error)` helper
- Combines: `encoding/json` + `gorilla/schema` (already a dep) + `validator/v10` (already a dep)
- Reduces boilerplate in every controller method

### 9. WebSocket Support

- Real-time bidirectional communication for notifications, live updates, chat
- Library: `github.com/gorilla/websocket` (already an indirect dep — promote to direct)
- Alternative: `github.com/coder/websocket` (nhooyr, stdlib-compatible)
- Build a WebSocket hub/manager abstraction with room support
- Wire into the middleware chain and DI container

### 10. File Upload and Cloud Storage

- Upload, store, and serve files with a unified interface
- Library: `github.com/minio/minio-go/v7` (works with AWS S3, GCS, MinIO, DigitalOcean Spaces)
- Build a `StorageService` interface with implementations: local filesystem, S3-compatible
- Add multipart upload middleware with file size and type validation
- Config-driven: `storage.driver: local` or `storage.driver: s3`

### 11. Async Task Queue

- Process background jobs with retries, priorities, and scheduling
- Library: `github.com/hibiken/asynq` (Redis-based, closest to Laravel Queues)
- Features: task retries, priority queues, unique tasks, scheduled tasks, dead letter queue
- Dashboard: `github.com/hibiken/asynqmon` (web UI for monitoring)
- Alternative: `github.com/ThreeDotsLabs/watermill` (supports Kafka, RabbitMQ, NATS, Redis Streams)

### 12. Security Headers Middleware

- Protect against common web vulnerabilities
- Libraries:
  - `github.com/unrolled/secure` (Helmet.js equivalent — HSTS, CSP, X-Frame-Options, X-Content-Type-Options)
  - `github.com/gorilla/csrf` (CSRF protection, pairs with gorilla/mux)
- Add to middleware chain, configurable via `config.yaml`

### 13. Observability (Metrics + Distributed Tracing)

- Monitor application performance and trace requests across services
- Libraries:
  - Metrics: `github.com/prometheus/client_golang` (expose `/metrics` for Prometheus)
  - Tracing: `go.opentelemetry.io/otel` (already indirect dep — promote)
  - Exporter: `go.opentelemetry.io/otel/exporters/otlp/otlptrace` (Jaeger, Zipkin, Tempo)
- Build middleware that records: request duration, status codes, active connections
- Add trace IDs to structured logs (correlate logs with traces)

### 14. API Versioning

- Support multiple API versions simultaneously
- Implement via gorilla/mux subrouters: `/api/v1/...`, `/api/v2/...`
- Alternative: header-based versioning (`Accept: application/vnd.api+json;version=2`)
- Version-specific route registration in the route config

---

## Tier 3: Advanced

Enterprise-grade features that differentiate gofasta from basic frameworks.

### 15. Notification System

- Unified multi-channel notification abstraction (like Laravel Notifications)
- Channels:
  - Email (already built — reuse mailer)
  - SMS: `github.com/twilio/twilio-go`
  - Slack: webhook HTTP calls
  - Push notifications: `github.com/sideshow/apns2` (iOS), `github.com/appleboy/go-fcm` (Android)
  - Database: store notifications in DB for in-app notification center
- Interface: `Notifier.Send(ctx, user, notification)` — routes to configured channels

### 16. Circuit Breaker / Resilience

- Protect against cascading failures in microservice architectures
- Library: `github.com/failsafe-go/failsafe-go` (circuit breaker, retry, timeout, fallback, rate limiter — all composable)
- Alternative: `github.com/sony/gobreaker` (simple circuit breaker)
- Retry with backoff: `github.com/avast/retry-go`
- Wrap external service calls (APIs, databases, message queues)

### 17. Internationalization (i18n)

- Translate API responses, error messages, and email templates
- Library: `github.com/nicksnyder/go-i18n/v2` (message catalogs, pluralization, TOML/JSON/YAML)
- Extend existing go-playground/universal-translator (already used in validators)
- Convention: locale files in `locales/en.yaml`, `locales/fr.yaml`, etc.
- Detect locale from `Accept-Language` header

### 18. Feature Flags

- Gradual rollouts, A/B testing, kill switches
- Library: `github.com/thomaspoignant/go-feature-flag` (file, S3, GitHub, HTTP backends)
- Features: percentage rollouts, user targeting, scheduled flags
- Alternative: integrate with LaunchDarkly, Unleash, or Flagsmith

### 19. GraphQL Subscriptions

- Real-time data streaming over GraphQL (WebSocket transport)
- gqlgen already supports subscriptions — just needs to be wired up
- Use cases: live notifications, real-time dashboards, chat
- Requires WebSocket support (#9) as foundation

### 20. Data Encryption at Rest

- Encrypt sensitive fields (PII, tokens, credentials) before storing in DB
- Use `golang.org/x/crypto` (already present) for AES-GCM encryption
- Build an `Encrypter` service with `Encrypt(plaintext) -> ciphertext` and `Decrypt(ciphertext) -> plaintext`
- Transparent field-level encryption via GORM hooks

### 21. Session Management

- Server-side sessions for hybrid/full-stack applications
- Library: `github.com/gorilla/sessions` (works with gorilla/mux)
- Stores: cookie, filesystem, Redis (`github.com/rbcervilla/redisstore`)
- Use case: admin panels, server-rendered pages

### 22. Worker Dashboard

- Web UI for monitoring background jobs and async tasks
- Library: `github.com/hibiken/asynqmon` (if using asynq for task queue)
- Shows: active workers, pending/completed/failed tasks, retry history
- Serve at `/admin/workers/` behind auth middleware

---

## Implementation Order (Recommended)

1. JWT Authentication + Authorization (Tier 1, #1 + #2)
2. Rate Limiting (Tier 1, #3)
3. Caching Layer (Tier 1, #4)
4. Request Binding Helper (Tier 2, #8)
5. API Documentation / Swagger (Tier 1, #5)
6. Deep Health Checks (Tier 1, #6)
7. Database Seeding (Tier 1, #7)
8. Security Headers (Tier 2, #12)
9. File/Cloud Storage (Tier 2, #10)
10. Async Task Queue (Tier 2, #11)
11. WebSocket Support (Tier 2, #9)
12. Observability (Tier 2, #13)
13. Notification System (Tier 3, #15)
14. Everything else from Tier 3
