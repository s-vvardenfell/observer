#!/usr/bin/env sh

docker compose -f docker-compose-with-observ.yaml down --remove-orphans --volumes && docker container prune -f && docker ps -a