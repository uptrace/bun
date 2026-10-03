//go:build go1.27

package internal

import (
	"reflect"
	"uuid"
)

// TypeUUID is the standard library uuid.UUID type. It is nil when the
// toolchain does not provide the uuid package (Go < 1.27).
var TypeUUID = reflect.TypeFor[uuid.UUID]()

// TypeUUIDPtr is the *uuid.UUID type, or nil when TypeUUID is nil.
var TypeUUIDPtr = reflect.TypeFor[*uuid.UUID]()
