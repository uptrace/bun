package pgdriver

import (
	"database/sql/driver"
	"reflect"
	"time"
)

// PostgreSQL type OIDs (from pg_type).
// See https://www.postgresql.org/docs/current/datatype-oid.html and system catalog.
const (
	pgTypeBool = 16

	pgTypeInt2   = 21
	pgTypeInt4   = 23
	pgTypeInt8   = 20
	pgTypeOID    = 26
	pgTypeXID    = 28
	pgTypeCID    = 29
	pgTypeJSON   = 114
	pgTypeXML    = 142
	pgTypeUUID   = 2950
	pgTypeJSONB  = 3802
	pgTypeBigSerial = 20 // same as int8
	pgTypeSerial = 23   // same as int4

	pgTypeFloat4 = 700
	pgTypeFloat8 = 701
	pgTypeNumeric = 1700

	pgTypeChar    = 18
	pgTypeName    = 19
	pgTypeText    = 25
	pgTypeBPChar  = 1042
	pgTypeVarchar = 1043

	pgTypeBytea = 17

	pgTypeDate        = 1082
	pgTypeTime        = 1083
	pgTypeTimestamp   = 1114
	pgTypeTimestamptz = 1184
	pgTypeInterval    = 1186

	pgTypeInet   = 869
	pgTypeCidr   = 650
	pgTypeMacAddr = 829

	pgTypeBoolArray   = 1000
	pgTypeInt2Array   = 1005
	pgTypeInt4Array   = 1007
	pgTypeInt8Array   = 1016
	pgTypeFloat4Array = 1021
	pgTypeFloat8Array = 1022
	pgTypeTextArray   = 1009
	pgTypeVarcharArray = 1015
	pgTypeByteaArray  = 1001
	pgTypeTimestampArray = 1115
	pgTypeTimestamptzArray = 1185
	pgTypeUUIDArray   = 2951
	pgTypeJSONBArray  = 3807
)

// columnTypeInfo holds database type name and Go scan type for a PostgreSQL OID.
type columnTypeInfo struct {
	dbTypeName string
	scanType   reflect.Type
}

var oidToColumnType = map[int32]columnTypeInfo{
	pgTypeBool:   {"BOOL", reflect.TypeOf(false)},
	pgTypeInt2:   {"INT2", reflect.TypeOf(int64(0))},
	pgTypeInt4:   {"INT4", reflect.TypeOf(int64(0))},
	pgTypeInt8:   {"INT8", reflect.TypeOf(int64(0))},
	pgTypeOID:    {"OID", reflect.TypeOf(int64(0))},
	pgTypeXID:    {"XID", reflect.TypeOf(int64(0))},
	pgTypeCID:    {"CID", reflect.TypeOf(int64(0))},
	pgTypeFloat4: {"FLOAT4", reflect.TypeOf(float64(0))},
	pgTypeFloat8: {"FLOAT8", reflect.TypeOf(float64(0))},
	pgTypeNumeric: {"NUMERIC", reflect.TypeOf(float64(0))},
	pgTypeChar:   {"CHAR", reflect.TypeOf("")},
	pgTypeName:   {"NAME", reflect.TypeOf("")},
	pgTypeText:   {"TEXT", reflect.TypeOf("")},
	pgTypeBPChar: {"BPCHAR", reflect.TypeOf("")},
	pgTypeVarchar: {"VARCHAR", reflect.TypeOf("")},
	pgTypeBytea:  {"BYTEA", reflect.TypeOf([]byte(nil))},
	pgTypeDate:   {"DATE", reflect.TypeOf(time.Time{})},
	pgTypeTime:   {"TIME", reflect.TypeOf(time.Time{})},
	pgTypeTimestamp:   {"TIMESTAMP", reflect.TypeOf(time.Time{})},
	pgTypeTimestamptz:  {"TIMESTAMPTZ", reflect.TypeOf(time.Time{})},
	pgTypeInterval:     {"INTERVAL", reflect.TypeOf("")},
	pgTypeJSON:   {"JSON", reflect.TypeOf([]byte(nil))},
	pgTypeJSONB:  {"JSONB", reflect.TypeOf([]byte(nil))},
	pgTypeXML:    {"XML", reflect.TypeOf("")},
	pgTypeUUID:   {"UUID", reflect.TypeOf("")},
	pgTypeInet:   {"INET", reflect.TypeOf("")},
	pgTypeCidr:   {"CIDR", reflect.TypeOf("")},
	pgTypeMacAddr: {"MACADDR", reflect.TypeOf("")},
	// arrays: scan as []byte for now (driver.Value does not have slice type)
	pgTypeBoolArray:       {"BOOL[]", reflect.TypeOf([]byte(nil))},
	pgTypeInt2Array:       {"INT2[]", reflect.TypeOf([]byte(nil))},
	pgTypeInt4Array:       {"INT4[]", reflect.TypeOf([]byte(nil))},
	pgTypeInt8Array:       {"INT8[]", reflect.TypeOf([]byte(nil))},
	pgTypeFloat4Array:     {"FLOAT4[]", reflect.TypeOf([]byte(nil))},
	pgTypeFloat8Array:     {"FLOAT8[]", reflect.TypeOf([]byte(nil))},
	pgTypeTextArray:       {"TEXT[]", reflect.TypeOf([]byte(nil))},
	pgTypeVarcharArray:    {"VARCHAR[]", reflect.TypeOf([]byte(nil))},
	pgTypeByteaArray:     {"BYTEA[]", reflect.TypeOf([]byte(nil))},
	pgTypeTimestampArray: {"TIMESTAMP[]", reflect.TypeOf([]byte(nil))},
	pgTypeTimestamptzArray: {"TIMESTAMPTZ[]", reflect.TypeOf([]byte(nil))},
	pgTypeUUIDArray:       {"UUID[]", reflect.TypeOf([]byte(nil))},
	pgTypeJSONBArray:      {"JSONB[]", reflect.TypeOf([]byte(nil))},
}

var scanTypeBytes = reflect.TypeOf([]byte(nil))

func columnTypeByOID(oid int32) (dbTypeName string, scanType reflect.Type) {
	if info, ok := oidToColumnType[oid]; ok {
		return info.dbTypeName, info.scanType
	}
	return "BYTEA", scanTypeBytes
}

// Ensure rows implements driver column type interfaces.
var (
	_ driver.RowsColumnTypeScanType           = (*rows)(nil)
	_ driver.RowsColumnTypeDatabaseTypeName   = (*rows)(nil)
)
