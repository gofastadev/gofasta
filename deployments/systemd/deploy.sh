#!/usr/bin/env bash
# Bare metal deployment script for VPS (no Docker).
# Usage: ./deployments/systemd/deploy.sh user@your-server.com
#
# Prerequisites on server:
#   - PostgreSQL (or your chosen DB) installed and running
#   - Go 1.25+ installed (for building) OR build locally and SCP the binary

set -euo pipefail

SERVER=${1:?"Usage: $0 user@server"}
APP_NAME="gofasta"
REMOTE_DIR="/etc/${APP_NAME}"
BINARY="/usr/local/bin/${APP_NAME}"

echo "==> Building binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./tmp/${APP_NAME} ./app/main

echo "==> Deploying to ${SERVER}..."

# Create remote directory structure
ssh "${SERVER}" "sudo mkdir -p ${REMOTE_DIR}/migrations ${REMOTE_DIR}/templates && sudo chown -R \$(whoami) ${REMOTE_DIR}"

# Copy files
scp ./tmp/${APP_NAME} "${SERVER}:${BINARY}.new"
scp config.yaml "${SERVER}:${REMOTE_DIR}/config.yaml"
scp -r db/migrations/* "${SERVER}:${REMOTE_DIR}/migrations/" 2>/dev/null || true
scp -r templates/* "${SERVER}:${REMOTE_DIR}/templates/" 2>/dev/null || true
scp deployments/systemd/gofasta.service "${SERVER}:/tmp/gofasta.service"

# Install and restart
ssh "${SERVER}" bash <<'REMOTE'
    sudo mv /usr/local/bin/gofasta.new /usr/local/bin/gofasta
    sudo chmod +x /usr/local/bin/gofasta
    sudo cp /tmp/gofasta.service /etc/systemd/system/gofasta.service

    # Create service user if not exists
    id -u gofasta &>/dev/null || sudo useradd -r -s /bin/false gofasta
    sudo chown -R gofasta:gofasta /etc/gofasta

    # Run migrations
    cd /etc/gofasta && sudo -u gofasta /usr/local/bin/gofasta migrate up || echo "Migrations: nothing to apply"

    # Restart service
    sudo systemctl daemon-reload
    sudo systemctl enable gofasta
    sudo systemctl restart gofasta

    echo "==> Deployed! Status:"
    sudo systemctl status gofasta --no-pager
REMOTE

echo "==> Done!"
