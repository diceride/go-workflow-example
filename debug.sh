#!/bin/bash

set -e

export $(grep -v '^#' .env | xargs)

cd $(mktemp -d)

curl -O https://raw.githubusercontent.com/uber/cadence/master/docker/docker-compose.yml && curl -O https://raw.githubusercontent.com/uber/cadence/master/docker/prometheus_config.yml
docker-compose up -d

sleep 30

docker run --network=host --rm ubercadence/cli:master --do "$CADENCE_DOMAIN" domain register -rd 1

cd -
