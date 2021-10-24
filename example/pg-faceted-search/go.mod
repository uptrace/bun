module github.com/uptrace/bun/example/pg-faceted-search

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/uptrace/bun v1.0.14
	github.com/uptrace/bun/dbfixture v1.0.14
	github.com/uptrace/bun/dialect/pgdialect v1.0.14
	github.com/uptrace/bun/driver/pgdriver v1.0.14
	github.com/uptrace/bun/extra/bundebug v1.0.14
)
