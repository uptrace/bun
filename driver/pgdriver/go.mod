module github.com/uptrace/bun/driver/pgdriver

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v1.0.16
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	mellium.im/sasl v0.2.1
)
