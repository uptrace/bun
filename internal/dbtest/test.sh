#!/bin/sh -eux

if command -v docker compose >/dev/null 2>/dev/null; then
  DOCKER_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>/dev/null; then
  DOCKER_CMD="docker-compose"
else
  echo "Error: Neither 'docker compose' nor 'docker-compose' is available."
  exit 1
fi

cd mssql-docker
docker build -t mssql-local .
cd ..
# shellcheck disable=SC2064
trap "${DOCKER_CMD} down -v" EXIT
${DOCKER_CMD} down -v
${DOCKER_CMD} up -d
sleep 30
CGO_ENABLED=0 TZ= go test "$@"
CGO_ENABLED=1 TZ= go test -tags cgosqlite "$@"
