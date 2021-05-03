package sqltype

import (
	"database/sql"
	"reflect"
	"time"
)

const (
	Boolean         = "BOOLEAN"
	SmallInt        = "SMALLINT"
	Integer         = "INTEGER"
	BigInt          = "BIGINT"
	Real            = "REAL"
	DoublePrecision = "DOUBLE PRECISION"
	VarChar         = "VARCHAR"
	Timestamp       = "TIMESTAMP"
)

var (
	timeType        = reflect.TypeOf((*time.Time)(nil)).Elem()
	sqlNullTimeType = reflect.TypeOf((*sql.NullTime)(nil)).Elem()
	nullBoolType    = reflect.TypeOf((*sql.NullBool)(nil)).Elem()
	nullFloatType   = reflect.TypeOf((*sql.NullFloat64)(nil)).Elem()
	nullIntType     = reflect.TypeOf((*sql.NullInt64)(nil)).Elem()
	nullStringType  = reflect.TypeOf((*sql.NullString)(nil)).Elem()
)

var types = []string{
	reflect.Bool:          Boolean,
	reflect.Int:           BigInt,
	reflect.Int8:          SmallInt,
	reflect.Int16:         SmallInt,
	reflect.Int32:         Integer,
	reflect.Int64:         BigInt,
	reflect.Uint:          BigInt,
	reflect.Uint8:         SmallInt,
	reflect.Uint16:        SmallInt,
	reflect.Uint32:        Integer,
	reflect.Uint64:        BigInt,
	reflect.Uintptr:       BigInt,
	reflect.Float32:       Real,
	reflect.Float64:       DoublePrecision,
	reflect.Complex64:     "",
	reflect.Complex128:    "",
	reflect.Array:         "",
	reflect.Chan:          "",
	reflect.Func:          "",
	reflect.Interface:     "",
	reflect.Map:           VarChar,
	reflect.Ptr:           "",
	reflect.Slice:         VarChar,
	reflect.String:        VarChar,
	reflect.Struct:        VarChar,
	reflect.UnsafePointer: "",
}

func Detect(typ reflect.Type) string {
	switch typ {
	case timeType, sqlNullTimeType:
		return Timestamp
	case nullBoolType:
		return Boolean
	case nullFloatType:
		return DoublePrecision
	case nullIntType:
		return BigInt
	case nullStringType:
		return VarChar
	}
	return types[typ.Kind()]
}
