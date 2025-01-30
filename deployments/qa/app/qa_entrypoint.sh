#!/bin/sh

# Load environment variables from .env file
. .env

set -euo pipefail

# This is the entrypoint script used for qa docker workflows.
# By default it will:
#  - Install dependencies.
#  - Run migrations.
#  - Start the gofasta qa server.
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

echo "Waiting for postgres to be available..."
sh deployments/${ENV}/app/wait-for-it.sh -t 30 -q ${PROJECT_NAME}_db_container_${ENV}:5432

if [ -z "${DATABASE_URL:-}" ]; then
  warnfail "DATABASE_URL is not set"
fi

if ! psql -d "${DATABASE_URL}" -c '\d schema_migrations' >/dev/null 2>&1; then
  echo "Running migrations..."
  sh ./scripts/migrate-up.sh
fi

echo "Starting ${PROJECT_NAME} server in ${ENV} environment..."
./${PROJECT_NAME}
