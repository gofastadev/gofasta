#!/bin/sh

# Enable strict error handling
set -euo pipefail

# Function for error handling and logging
warn_on_fail() {
  echo "Error: $*" >&2
  exit 1
}

# Ensure the .env file exists and load it
[ -f .env.production ] || warn_on_fail ".env.production file not found"
. .env.production || warn_on_fail "Failed to load .env.production file"

# Validate that PROJECT_NAME is set
[ -z "${PROJECT_NAME:-}" ] && warn_on_fail "PROJECT_NAME is not set in the .env file"

# Variables
REPO_URL="git@github.com:healtronlabs/${PROJECT_NAME}.git"
CLONE_DIR="${PROJECT_NAME}"
DOCKERFILE_PATH="./deployments/${ENV}/app/dockerfile"
IMAGE_TAG="healtron/${PROJECT_NAME}:${ENV}"
ENV_FILE=".env.${ENV}"
PM2_PROCESS_NAME="${PROJECT_NAME}-${ENV}-process"

# Remove any existing repository clone
if [ -d "${CLONE_DIR}" ]; then
  echo "Removing existing ${CLONE_DIR} directory..."
  rm -rf "${CLONE_DIR}"
fi

# Clone the repository
echo "Cloning repository from ${REPO_URL}..."
git clone "${REPO_URL}" "${CLONE_DIR}" || warn_on_fail "Failed to clone repository"

# Copy the environment file
if [ -f "${ENV_FILE}" ]; then
  echo "Copying environment file to ${CLONE_DIR}/.env..."
  cp "${ENV_FILE}" "${CLONE_DIR}/.env"
else
  warn_on_fail "Environment file ${ENV_FILE} not found"
fi

# Build the Docker image
echo "Building Docker image ${IMAGE_TAG}..."
(
  cd "${CLONE_DIR}" || warn_on_fail "Failed to enter ${CLONE_DIR} directory"
  BUILD_ARGS=$(grep -v '^#' ${ENV_FILE}.${ENV} | sed 's/^/--build-arg /' | tr '\n' ' ')
  docker build --no-cache --file "${DOCKERFILE_PATH}" --tag "${IMAGE_TAG}" $BUILD_ARGS .
)

# Restart the PM2 process
echo "Restarting PM2 process: ${PM2_PROCESS_NAME}..."
if pm2 describe "${PM2_PROCESS_NAME}" > /dev/null; then
  pm2 restart "${PM2_PROCESS_NAME}" || warn_on_fail "Failed to restart PM2 process"
else
  warn_on_fail "PM2 process ${PM2_PROCESS_NAME} not found"
fi

# Clean up unused Docker images and build cache
echo "Pruning unused Docker images and build cache..."
docker image prune -f || warn_on_fail "Failed to prune Docker images"
docker builder prune -a -f || warn_on_fail "Failed to prune Docker build cache"

echo "Deployment completed successfully!"
