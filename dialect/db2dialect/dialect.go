package db2dialect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

var errUnsupportedDriverConn = errors.New("db2dialect: driver connection does not support GetInfo() detection")

// sqlDBMSName is the standard ODBC SQLGetInfo infoType requesting the DBMS
// name (SQL_DBMS_NAME), mirrored here to avoid a hard dependency on the
// go_ibm_db driver's cgo-only api package.
const sqlDBMSName = 17

// getDBMSName calls the driver connection's GetInfo(infoType) (string, error)
// method (as implemented by *go_ibm_db.Conn) via reflection. Reflection is
// used, rather than a static interface, because the method's infoType
// parameter is a named cgo type (api.SQLUSMALLINT) that db2dialect cannot
// reference without pulling in the driver's cgo build requirements.
func getDBMSName(driverConn any) (string, error) {
	v := reflect.ValueOf(driverConn)
	m := v.MethodByName("GetInfo")
	if !m.IsValid() {
		return "", errUnsupportedDriverConn
	}
	mt := m.Type()
	if mt.NumIn() != 1 || mt.NumOut() != 2 {
		return "", errUnsupportedDriverConn
	}
	argType := mt.In(0)
	if argType.Kind() != reflect.Uint16 {
		return "", errUnsupportedDriverConn
	}
	if !mt.Out(0).AssignableTo(reflect.TypeFor[string]()) || !mt.Out(1).Implements(reflect.TypeFor[error]()) {
		return "", errUnsupportedDriverConn
	}

	arg := reflect.New(argType).Elem()
	arg.SetUint(sqlDBMSName)

	out := m.Call([]reflect.Value{arg})
	if errVal := out[1].Interface(); errVal != nil {
		return "", errVal.(error)
	}
	return out[0].String(), nil
}

func init() {
	if Version() != bun.Version() {
		panic(fmt.Errorf("db2dialect and Bun must have the same version: v%s != v%s",
			Version(), bun.Version()))
	}
}

type Dialect struct {
	schema.BaseDialect

	tables              *schema.Tables
	features            feature.Feature
	target              TargetPlatform
	targetSetExplicitly bool
	autoDetected        bool
}

func New(opts ...DialectOption) *Dialect {
	d := &Dialect{target: TargetLUW}
	d.tables = schema.NewTables(d)
	d.features = feature.CTE |
		feature.WithValues |
		feature.SelectExists |
		feature.CompositeIn |
		feature.OffsetFetch |
		feature.Identity

	for _, opt := range opts {
		opt(d)
	}

	return d
}

type DialectOption func(*Dialect)

// NewLUW creates a dialect explicitly configured for DB2 for LUW.
func NewLUW() *Dialect {
	return New(WithTarget(TargetLUW))
}

// NewZOS creates a dialect explicitly configured for DB2 for z/OS.
func NewZOS() *Dialect {
	return New(WithTarget(TargetZOS))
}

// NewIBMi creates a dialect explicitly configured for DB2 for IBM i.
func NewIBMi() *Dialect {
	return New(WithTarget(TargetIBMi))
}

// TargetPlatform represents the target DB2 platform flavor.
type TargetPlatform int

const (
	TargetLUW TargetPlatform = iota
	TargetZOS
	TargetIBMi
)

func (t TargetPlatform) String() string {
	switch t {
	case TargetZOS:
		return "z/OS"
	case TargetIBMi:
		return "IBM i"
	default:
		return "LUW"
	}
}

// WithTarget sets the target DB2 platform flavor.
func WithTarget(target TargetPlatform) DialectOption {
	return func(d *Dialect) {
		d.target = target
		d.targetSetExplicitly = true
	}
}

func WithoutFeature(other feature.Feature) DialectOption {
	return func(d *Dialect) {
		d.features = d.features.Remove(other)
	}
}

func classifyDBMSName(name string) TargetPlatform {
	upper := strings.ToUpper(name)
	switch {
	case upper == "DB2" || strings.HasPrefix(upper, "DSN"):
		return TargetZOS
	case strings.HasPrefix(upper, "AS"):
		return TargetIBMi
	case strings.HasPrefix(upper, "DB2/"):
		return TargetLUW
	default:
		log.Printf("db2dialect: WARNING: unable to deduce DB2 platform from DBMS_NAME %q, defaulting to LUW", name)
		return TargetLUW
	}
}

func (d *Dialect) Init(db *sql.DB) {
	if db == nil || d.targetSetExplicitly || d.autoDetected {
		return
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Printf("db2dialect: WARNING: unable to deduce DB2 platform (%v), defaulting to LUW", err)
		d.target = TargetLUW
		d.autoDetected = true
		return
	}
	defer conn.Close()

	var name string
	rawErr := conn.Raw(func(driverConn any) error {
		var callErr error
		name, callErr = getDBMSName(driverConn)
		return callErr
	})
	if rawErr != nil {
		log.Printf("db2dialect: WARNING: unable to deduce DB2 platform (%v), defaulting to LUW", rawErr)
		d.target = TargetLUW
		d.autoDetected = true
		return
	}

	d.target = classifyDBMSName(name)
	d.autoDetected = true
}

func (d *Dialect) Name() dialect.Name {
	return dialect.DB2
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

// Target returns the configured DB2 platform flavor.
func (d *Dialect) Target() TargetPlatform {
	return d.target
}

// DummyTable returns the system dummy table required by z/OS for SELECTs
// without an application table.
func (d *Dialect) DummyTable() string {
	return "SYSIBM.SYSDUMMY1"
}

// CatalogSchema returns the system catalog schema for the target platform.
func (d *Dialect) CatalogSchema() string {
	switch d.target {
	case TargetZOS:
		return "SYSIBM"
	case TargetIBMi:
		return "QSYS2"
	default:
		return "SYSCAT"
	}
}

func (d *Dialect) OnTable(table *schema.Table) {
	for _, field := range table.FieldMap {
		if d.target == TargetZOS && (field.SQLDefault == "current_timestamp" || field.SQLDefault == "CURRENT_TIMESTAMP" || field.SQLDefault == "WITH DEFAULT") {
			field.SQLDefault = ""
		}
		// Bun resolves CreateTableSQLType from UserSQLType/DiscoveredSQLType only
		// after OnTable returns, so an explicit `type:` tag must be preserved here.
		if field.UserSQLType != "" || field.CreateTableSQLType != "" {
			continue
		}
		field.DiscoveredSQLType = fieldSQLType(field)
	}
}

func (d *Dialect) IdentQuote() byte {
	return '"'
}

func (d *Dialect) DefaultVarcharLen() int {
	return 255
}

// DefaultSchema returns an empty schema so DB2 uses the current authorization ID.
func (d *Dialect) DefaultSchema() string {
	return ""
}

// AppendSequence emits the DB2 IDENTITY clause for autoincrement primary keys.
func (d *Dialect) AppendSequence(b []byte, _ *schema.Table, field *schema.Field) []byte {
	if !field.IsPK || field.CreateTableSQLType != "BIGINT" && field.CreateTableSQLType != "INTEGER" {
		return b
	}
	return append(b, " GENERATED BY DEFAULT AS IDENTITY (START WITH 1 INCREMENT BY 1)"...)
}

func (*Dialect) AppendTime(b []byte, tm time.Time) []byte {
	if tm.IsZero() {
		return append(b, "NULL"...)
	}
	b = append(b, '\'')
	b = tm.UTC().AppendFormat(b, "2006-01-02 15:04:05.999999")
	b = append(b, '\'')
	return b
}

func (*Dialect) AppendBool(b []byte, v bool) []byte {
	if v {
		return append(b, '1')
	}
	return append(b, '0')
}

var (
	dateType      = reflect.TypeFor[Date]()
	timeOfDayType = reflect.TypeFor[TimeOfDay]()
	timestampType = reflect.TypeFor[Timestamp]()
	timeType      = reflect.TypeFor[time.Time]()

	smallIntType         = reflect.TypeFor[SmallInt]()
	smallIntBoolType     = reflect.TypeFor[SmallIntBool]()
	nullSmallIntType     = reflect.TypeFor[NullSmallInt]()
	nullSmallIntBoolType = reflect.TypeFor[NullSmallIntBool]()
	nullBoolType         = reflect.TypeFor[sql.NullBool]()
	nullStringType       = reflect.TypeFor[sql.NullString]()
	nullInt64Type        = reflect.TypeFor[sql.NullInt64]()
	nullInt32Type        = reflect.TypeFor[sql.NullInt32]()
	nullInt16Type        = reflect.TypeFor[sql.NullInt16]()
	nullByteType         = reflect.TypeFor[sql.NullByte]()
	nullFloat64Type      = reflect.TypeFor[sql.NullFloat64]()
	nullTimeType         = reflect.TypeFor[sql.NullTime]()
)

// fieldSQLType maps the SQL type bun discovers from the Go field type to the
// closest DB2-native type. DB2 has no usable BOOLEAN column type on older
// versions, so booleans are stored as SMALLINT.
func fieldSQLType(field *schema.Field) string {
	// The helper types in types.go carry DB2 semantics that bun cannot infer:
	// the SMALLINT ones would be discovered as INTEGER/VARCHAR, and the
	// time-based ones as VARCHAR because they are named types, not time.Time.
	switch field.IndirectType {
	case smallIntType, smallIntBoolType, nullSmallIntType, nullSmallIntBoolType,
		nullBoolType, nullInt16Type, nullByteType:
		return sqltype.SmallInt
	case nullStringType:
		return sqltype.VarChar
	case nullInt64Type:
		return sqltype.BigInt
	case nullInt32Type:
		return sqltype.Integer
	case nullFloat64Type:
		return sqltype.DoublePrecision
	case nullTimeType:
		return sqltype.Timestamp
	case dateType:
		return "DATE"
	case timeOfDayType:
		return "TIME"
	case timestampType:
		return sqltype.Timestamp
	}

	if field.IndirectType.Kind() == reflect.Struct && field.IndirectType.ConvertibleTo(timeType) {
		return sqltype.Timestamp
	}

	switch field.DiscoveredSQLType {
	case sqltype.Boolean:
		return sqltype.SmallInt
	default:
		return field.DiscoveredSQLType
	}
}
