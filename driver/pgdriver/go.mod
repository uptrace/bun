module github.com/uptrace/bun/driver/pgdriver

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

require (
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v0.0.0-20210507075305-2e91d2c5c8de
	github.com/uptrace/bun/dialect/pgdialect v0.0.0-00010101000000-000000000000
)
