#!/bin/sh

# cd /gofasta

source .env

docker exec -it ${PROJECT_NAME}_web_container_dev sh
