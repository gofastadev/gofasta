#!/bin/sh

cd /go-gql-template

source .env

docker exec -it ${PROJECT_NAME}_db_container_dev bash
