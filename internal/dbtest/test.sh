#!/bin/sh -eux
trap 'docker-compose down -v' EXIT
docker-compose down -v
docker-compose up -d
sleep 30
CGO_ENABLED=0 TZ= go test
CGO_ENABLED=1 TZ= go test -tags cgosqlite
