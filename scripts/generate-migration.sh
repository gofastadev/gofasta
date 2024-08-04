#!/bin/sh
if [ -z "$1" ]; then
  echo "No argument supplied, you should pass the migration name"
else
  # Check if a migration file with the same name exists
  if ls db/migrations/*_"$1".up.sql 1> /dev/null 2>&1 || ls db/migrations/*_"$1".down.sql 1> /dev/null 2>&1; then
    echo "Migration file with the name '$1' already exists."
  else
    migrate create -ext sql -dir db/migrations -seq "$1"
  fi
fi
