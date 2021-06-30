#!/bin/sh -eux
trap 'docker-compose down -v' EXIT
docker-compose down -v
docker-compose up -d
sleep 15
CGO_ENABLED=0 go test -count=1
CGO_ENABLED=1 go test -count=1 -tags cgosqlite
