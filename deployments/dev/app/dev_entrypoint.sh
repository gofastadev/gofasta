#!/bin/sh
set -e

# If arguments provided, run them instead (e.g. docker compose run app sh)
if [ $# -gt 0 ]; then
  exec "$@"
fi

echo "==> Installing dependencies..."
go mod download

echo "==> Running database migrations..."
# Build DSN from environment variables
DSN="${GOFASTA_DATABASE_DRIVER:-postgres}://${GOFASTA_DATABASE_USER}:${GOFASTA_DATABASE_PASSWORD}@${GOFASTA_DATABASE_HOST}:${GOFASTA_DATABASE_PORT}/${GOFASTA_DATABASE_NAME}?sslmode=${GOFASTA_DATABASE_SSLMODE:-disable}"
migrate -database "$DSN" -path db/migrations up 2>&1 || echo "==> Migrations: nothing to apply (or already up-to-date)"

echo "==> Starting dev server with hot reload..."
echo "    REST API:    http://localhost:${PORT:-8080}"
echo "    GraphQL:     http://localhost:${PORT:-8080}/graphql"
echo "    Playground:  http://localhost:${PORT:-8080}/graphql-playground"
echo ""
exec go tool air
