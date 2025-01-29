#!/bin/sh

# cd /gofasta

source .env

migrate -database ${DATABASE_URL} -path db/migrations up
