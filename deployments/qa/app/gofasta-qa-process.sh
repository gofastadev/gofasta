#!/bin/sh

# Enable strict error handling
set -euo pipefail

# Function for error handling and logging
warn_on_fail() {
  echo "Error: $*" >&2
  exit 1
}

# Log messages for better visibility
log_info() {
  echo "Info: $*"
}

# Ensure the .env file exists and load it
[ -f .env.qa ] || warn_on_fail ".env.qa file not found"
. .env.qa || warn_on_fail "Failed to load .env.qa file"

# Validate required environment variables
validate_env_vars() {
  local missing_vars=()
  for var in PROJECT_NAME ENV; do
    [ -z "${!var:-}" ] && missing_vars+=("$var")
  done
  if [ "${#missing_vars[@]}" -ne 0 ]; then
    warn_on_fail "Missing required environment variables: ${missing_vars[*]}"
  fi
}
validate_env_vars

# Define the Docker Compose file path
DOCKER_COMPOSE_FILE="./${PROJECT_NAME}/deployments/${ENV}/app/compose-${ENV}.yml"

# Ensure the Docker Compose file exists
[ -f "$DOCKER_COMPOSE_FILE" ] || warn_on_fail "Docker Compose file not found at $DOCKER_COMPOSE_FILE"

# Stop and start the Docker Compose services
log_info "Stopping existing services using $DOCKER_COMPOSE_FILE..."
docker compose -p ${PROJECT_NAME}_${ENV}_api_process -f "$DOCKER_COMPOSE_FILE" --env-file .env.${ENV} --profile main down || warn_on_fail "Failed to stop Docker Compose services"

log_info "Starting services using $DOCKER_COMPOSE_FILE..."
docker compose -p ${PROJECT_NAME}_${ENV}_api_process -f "$DOCKER_COMPOSE_FILE" --env-file .env.${ENV} --profile main up || warn_on_fail "Failed to start Docker Compose services"

log_info "Deployment completed successfully!"
