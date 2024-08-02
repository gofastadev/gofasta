#!/bin/sh

cd /go_gql_template

source .env

docker exec -it ${PROJECT_NAME}_db_container_dev bash
