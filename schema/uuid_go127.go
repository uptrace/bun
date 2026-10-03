//go:build go1.27

package schema

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/internal"
)

// scanUUID scans into a uuid.UUID. Scanner(uuid.UUID) passes the addressable
// field value and Scanner(*uuid.UUID) is wrapped by addrScanner, which passes
// the addressable pointer, so dest is an addressable uuid.UUID value here.
// NULL is handled locally rather than by the shared scanNull helper so that a
// NULL column is scanned as the nil UUID.
func scanUUID(dest reflect.Value, src any) error {
	if !dest.CanAddr() {
		return fmt.Errorf("bun: Scan(nonaddressable %s)", dest.Type())
	}

	if src == nil {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}

	u := dest.Addr().Interface().(encoding.TextUnmarshaler)

	switch src := src.(type) {
	case [16]byte:
		// Raw UUID bytes are accepted from [16]byte sources, mirroring
		// database/sql, which copies driver.Value bytes of matching length.
		reflect.Copy(dest, reflect.ValueOf(src))
		return nil
	case []byte:
		if len(src) == 16 {
			reflect.Copy(dest, reflect.ValueOf(src))
			return nil
		}
		return u.UnmarshalText(src)
	case string:
		return u.UnmarshalText(internal.Bytes(src))
	default:
		return scanError(dest.Type(), src)
	}
}

// ScanUUIDText parses a UUID from its textual form only.
//
// Null is handled here because a quoted empty string and an unquoted NULL both
// arrive as an empty slice; callers that can tell them apart must do so before
// calling. Array elements arrive as text even for a uuid[] column, so a
// 16-character value such as "0123456789abcdef" must be rejected instead of
// being mistaken for raw UUID bytes by scanUUID.
func ScanUUIDText(dest reflect.Value, src any) error {
	if !dest.CanAddr() {
		return fmt.Errorf("bun: Scan(nonaddressable %s)", dest.Type())
	}

	if src == nil {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}

	u := dest.Addr().Interface().(encoding.TextUnmarshaler)

	switch src := src.(type) {
	case string:
		return u.UnmarshalText(internal.Bytes(src))
	case []byte:
		// Convert rather than alias: UnmarshalText must not observe a buffer
		// the caller can still mutate.
		return u.UnmarshalText(internal.Bytes(string(src)))
	default:
		return scanError(dest.Type(), src)
	}
}

func appendUUIDText(gen QueryGen, b []byte, v reflect.Value) []byte {
	text, err := v.Interface().(encoding.TextAppender).AppendText(nil)
	if err != nil {
		return dialect.AppendError(b, err)
	}
	return gen.Dialect().AppendString(b, internal.String(text))
}

func isZeroUUID(v reflect.Value) bool {
	for i := 0; i < v.Len(); i++ {
		if v.Index(i).Uint() != 0 {
			return false
		}
	}
	return true
}
