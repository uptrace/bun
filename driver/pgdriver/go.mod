module github.com/uptrace/bun/driver/pgdriver

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v1.0.18
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871 // indirect
	mellium.im/sasl v0.2.1
)
