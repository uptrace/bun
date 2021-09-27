module github.com/uptrace/bun/example/placeholders

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

require (
	github.com/uptrace/bun v1.0.9
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.9
	github.com/uptrace/bun/driver/sqliteshim v1.0.9
	github.com/uptrace/bun/extra/bundebug v1.0.9
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
