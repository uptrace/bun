// Use modernc.org/sqlite on all supported platforms unless Cgo driver
// was explicitly requested.
//
// See also https://pkg.go.dev/modernc.org/sqlite#hdr-Supported_platforms_and_architectures

// +build !cgosqlite
// +build darwin,amd64 darwin,arm64 linux,386 linux,amd64 linux,arm linux,arm64 windows,amd64

package sqliteshim

import "modernc.org/sqlite"

const (
	needsCgo   = false
	usesCgo    = false
	driverName = "sqlite"
)

var shimDriver = (*sqlite.Driver)(nil)
