#!/bin/sh

# cd /go_gql_template

source .env

migrate -database ${DATABASE_URL} -path db/migrations down
