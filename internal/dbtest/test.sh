#!/bin/sh -eux
cd mssql-docker
docker build -t mssql-local .
cd ..
trap 'docker-compose down -v' EXIT
docker-compose down -v
docker-compose up -d
sleep 30
CGO_ENABLED=0 TZ= go test "$@"
CGO_ENABLED=1 TZ= go test -tags cgosqlite "$@"
