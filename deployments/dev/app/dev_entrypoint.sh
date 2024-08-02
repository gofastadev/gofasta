#!/bin/sh

# Load environment variables from .env file
. .env

set -euo pipefail

# This is the entrypoint script used for development docker workflows.
# By default it will:
#  - Install dependencies.
#  - Run migrations.
#  - Start the dev server.
# It also accepts any commands to be run instead.

warnfail() {
  echo "$@" >&2
  exit 1
}

case ${1:-} in
"") # If no arguments are provided, start air dev server.
  ;;
*) # If any arguments are provided, execute them instead.
  exec "$@" ;;
esac

if [ ! -f go.mod ]; then
  echo "Initializing module"
  go mod init github.com/$GITHUB_USERNAME/$PROJECT_NAME
fi

echo "Installing all dependencies..."
go mod tidy

echo "Waiting for postgres to be available..."
sh deployments/dev/app/wait-for-it.sh -t 30 -q go_gql_template_db_container_dev:5432

if [ -z "${DATABASE_URL:-}" ]; then
  warnfail "DATABASE_URL is not set"
fi

if ! psql -d "$DATABASE_URL" -c '\d schema_migrations' >/dev/null 2>&1; then
  echo "Running migrations..."
  ./scripts/migrate-up.sh
fi

echo "Starting air dev server..."
exec go run github.com/air-verse/air
