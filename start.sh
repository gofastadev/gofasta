#!/usr/bin/env bash

for script in ./scripts/*.sh; do
  if [ -f "$script" ]; then
    chmod +x "$script"
    echo "Made $script executable"
  fi
done

if [ -z "$1" ]; then
  echo "No argument supplied"
else
  if [[ "$1" == "dev" || "$1" == "qa" || "$1" == "prod" ]]; then
    echo "Starting gofasta in: $1 environment..."
    docker compose -p gofasta_helpers -f "./compose-$1.yml" --profile db up -d
    docker compose -p gofasta_full_setup -f "./compose-$1.yml" --profile main up
  else
    echo "Invalid environment: $1, valid environments are: dev, qa, prod"
  fi
fi
