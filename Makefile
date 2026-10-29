# example/db2 requires IBM's proprietary DB2 CLI/ODBC client driver (cgo + libdb2)
# and is tested separately by the DB2 integration job.
ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | grep -v '^\./example/db2$$' | sort)
EXAMPLE_GO_MOD_DIRS := $(shell find ./example/ -type f -name 'go.mod' -exec dirname {} \; | grep -v '^\./example/db2$$' | sort)

test:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go test in $${dir}"; \
	  (cd "$${dir}" && \
	    go test && \
	    go test -race && \
	    env GOOS=linux GOARCH=386 TZ= go test && \
	    go vet); \
	done

go_mod_tidy:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go get -u ./... && \
	    go mod tidy); \
	done

fmt:
	gofmt -w -s ./
	goimports -w  -local github.com/uptrace/bun ./

run-examples:
	set -e; for dir in $(EXAMPLE_GO_MOD_DIRS); do \
	  echo "go run . in $${dir}"; \
	  (cd "$${dir}" && go run .); \
	done
