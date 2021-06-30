#!/bin/sh -eux
trap 'docker-compose down -v' EXIT
docker-compose down -v
docker-compose up -d
sleep 15
CGO_ENABLED=0 go test -run=^$ -bench=. -benchmem -count=10 | tee bench-purego.txt
CGO_ENABLED=1 go test -run=^$ -bench=. -benchmem -count=10 -tags cgosqlite | tee bench-cgo.txt
benchstat bench-cgo.txt bench-purego.txt
