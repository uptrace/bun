#!/bin/sh -eux
trap 'docker-compose down -v' EXIT
docker-compose down -v
docker-compose up -d
sleep 20
CGO_ENABLED=0 go test
CGO_ENABLED=1 go test -tags cgosqlite
