//go:build !go1.27

package schema

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/dialect"
)

// uuidRequiresGo127 explains that the standard library uuid.UUID type is only
// available on Go 1.27 or newer. Exact-type guards make these handlers
// unreachable on older toolchains; they exist so both build tags share one
// signature.
func uuidRequiresGo127() error {
	return fmt.Errorf("bun: standard library uuid.UUID requires Go 1.27 or newer")
}

// ScanUUIDText reports that textual UUID scanning needs Go 1.27 or newer.
func ScanUUIDText(dest reflect.Value, src any) error {
	return uuidRequiresGo127()
}

func scanUUID(dest reflect.Value, src any) error {
	return uuidRequiresGo127()
}

func appendUUIDText(gen QueryGen, b []byte, v reflect.Value) []byte {
	return dialect.AppendError(b, uuidRequiresGo127())
}

func isZeroUUID(v reflect.Value) bool {
	panic(uuidRequiresGo127())
}
