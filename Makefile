ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

test:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go test in $${dir}"; \
	  (cd "$${dir}" && \
	    go test ./... && \
	    go vet); \
	done

tag:
	git tag $(VERSION)
	git tag dialect/pgdialect/$(VERSION)
	git tag dialect/mysqldialect/$(VERSION)
	git tag dialect/sqlitedialect/$(VERSION)
	git tag driver/pgdriver/$(VERSION)
	git tag dbfixture/$(VERSION)
	git tag extra/bundebug/$(VERSION)
	git tag extra/bunotel/$(VERSION)

go_mod_tidy:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go get -d ./... && \
	    go mod tidy); \
	done

fmt:
	gofmt -w -s ./
	goimports -w  -local github.com/uptrace/bun ./
