#!/bin/sh

cd /go-gql-template

source .env

migrate -database ${DATABASE_URL} -path database/migrations down
