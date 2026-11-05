package db2dialect

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/uptrace/bun/schema"
)

func TestFieldSQLType(t *testing.T) {
	tests := []struct {
		name   string
		typ    reflect.Type
		want   string
		userIn string
	}{
		{name: "bool", typ: reflect.TypeFor[bool](), want: "SMALLINT"},
		{name: "int32", typ: reflect.TypeFor[int32](), want: "INTEGER"},
		{name: "int64", typ: reflect.TypeFor[int64](), want: "BIGINT"},
		{name: "float32", typ: reflect.TypeFor[float32](), want: "REAL"},
		{name: "float64", typ: reflect.TypeFor[float64](), want: "DOUBLE PRECISION"},
		{name: "string", typ: reflect.TypeFor[string](), want: "VARCHAR"},
		{name: "[]byte", typ: reflect.TypeFor[[]byte](), want: "BLOB"},
		{name: "time.Time", typ: reflect.TypeFor[time.Time](), want: "TIMESTAMP"},
		{name: "Date", typ: reflect.TypeFor[Date](), want: "DATE"},
		{name: "TimeOfDay", typ: reflect.TypeFor[TimeOfDay](), want: "TIME"},
		{name: "Timestamp", typ: reflect.TypeFor[Timestamp](), want: "TIMESTAMP"},
		{name: "SmallInt", typ: reflect.TypeFor[SmallInt](), want: "SMALLINT"},
		{name: "SmallIntBool", typ: reflect.TypeFor[SmallIntBool](), want: "SMALLINT"},
		{name: "NullSmallInt", typ: reflect.TypeFor[NullSmallInt](), want: "SMALLINT"},
		{name: "NullSmallIntBool", typ: reflect.TypeFor[NullSmallIntBool](), want: "SMALLINT"},
		{name: "sql.NullBool", typ: reflect.TypeFor[sql.NullBool](), want: "SMALLINT"},
		{name: "sql.NullString", typ: reflect.TypeFor[sql.NullString](), want: "VARCHAR"},
		{name: "sql.NullInt64", typ: reflect.TypeFor[sql.NullInt64](), want: "BIGINT"},
		{name: "sql.NullInt32", typ: reflect.TypeFor[sql.NullInt32](), want: "INTEGER"},
		{name: "sql.NullInt16", typ: reflect.TypeFor[sql.NullInt16](), want: "SMALLINT"},
		{name: "sql.NullByte", typ: reflect.TypeFor[sql.NullByte](), want: "SMALLINT"},
		{name: "sql.NullFloat64", typ: reflect.TypeFor[sql.NullFloat64](), want: "DOUBLE PRECISION"},
		{name: "sql.NullTime", typ: reflect.TypeFor[sql.NullTime](), want: "TIMESTAMP"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := &schema.Field{
				IndirectType:      test.typ,
				DiscoveredSQLType: schema.DiscoverSQLType(test.typ),
			}
			if got := fieldSQLType(field); got != test.want {
				t.Errorf("expected %q, got %q", test.want, got)
			}
		})
	}
}

// TestOnTablePreservesUserSQLType guards against overriding an explicit
// bun:"type:..." tag, which must always win over the discovered type.
func TestOnTablePreservesUserSQLType(t *testing.T) {
	type model struct {
		Name string `bun:"name,type:VARCHAR(100)"`
	}

	d := New()
	table := d.Tables().Get(reflect.TypeFor[model]())

	field := table.FieldMap["name"]
	if field == nil {
		t.Fatal("expected field \"name\" to be registered")
	}
	if field.CreateTableSQLType != "VARCHAR(100)" {
		t.Errorf("expected VARCHAR(100), got %q", field.CreateTableSQLType)
	}
}

func TestSmallIntBoolScan(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want SmallIntBool
	}{
		{"nil", nil, 0},
		{"int64", int64(1), 1},
		{"int32", int32(1), 1},
		{"float64", float64(1), 1},
		{"numeric string", "1", 1},
		{"bool string", "true", 1},
		{"false string", "false", 0},
		{"bytes", []byte("1"), 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got SmallIntBool
			if err := got.Scan(test.in); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got != test.want {
				t.Errorf("expected %d, got %d", test.want, got)
			}
		})
	}

	var s SmallIntBool
	if err := s.Scan("not a bool"); err == nil {
		t.Error("expected an error for an unparsable string")
	}
}

// TestNullScansRemainInvalidAfterError guards against a failed Scan leaving a
// stale Valid=true (with a previously scanned value) on the Null* wrappers.
func TestNullScansRemainInvalidAfterError(t *testing.T) {
	boolValue := NullSmallIntBool{SmallIntBool: 1}
	if err := boolValue.Scan("not-a-number"); err == nil {
		t.Fatal("expected invalid boolean scan to fail")
	}
	if boolValue.Valid {
		t.Fatal("boolean value should remain invalid after a failed scan")
	}
	if got, err := boolValue.Value(); err != nil || got != nil {
		t.Fatalf("failed boolean scan should emit NULL, got value=%v err=%v", got, err)
	}

	intValue := NullSmallInt{SmallInt: 7}
	if err := intValue.Scan("not-a-number"); err == nil {
		t.Fatal("expected invalid integer scan to fail")
	}
	if intValue.Valid {
		t.Fatal("integer value should remain invalid after a failed scan")
	}
	if got, err := intValue.Value(); err != nil || got != nil {
		t.Fatalf("failed integer scan should emit NULL, got value=%v err=%v", got, err)
	}
}

// TestValuersReturnDriverValueTypes guards the database/sql/driver contract,
// which permits int64 but not int32.
func TestValuersReturnDriverValueTypes(t *testing.T) {
	v, err := SmallInt(7).Value()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, ok := v.(int64); !ok {
		t.Errorf("expected int64, got %T", v)
	}

	v, err = SmallIntBool(1).Value()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, ok := v.(int64); !ok {
		t.Errorf("expected int64, got %T", v)
	}
}

func TestNullSmallIntValue(t *testing.T) {
	v, err := NullSmallInt{}.Value()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if v != nil {
		t.Errorf("expected nil for an invalid value, got %v", v)
	}

	v, err = NullSmallInt{SmallInt: 3, Valid: true}.Value()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if v != int64(3) {
		t.Errorf("expected 3, got %v", v)
	}
}

func TestTemporalValuers(t *testing.T) {
	tm := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	if v, _ := Date(tm).Value(); v != "2026-03-04" {
		t.Errorf("expected 2026-03-04, got %v", v)
	}
	if v, _ := TimeOfDay(tm).Value(); v != "05:06:07" {
		t.Errorf("expected 05:06:07, got %v", v)
	}
	if v, _ := Timestamp(tm).Value(); v != "2026-03-04 05:06:07.000000" {
		t.Errorf("expected 2026-03-04 05:06:07.000000, got %v", v)
	}
}

func TestTemporalScanners(t *testing.T) {
	var d Date
	if err := d.Scan("2026-03-04"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !d.Time().Equal(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected date: %s", d.Time())
	}

	var tod TimeOfDay
	if err := tod.Scan([]byte("05:06:07")); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if tod.Time().Hour() != 5 {
		t.Errorf("unexpected hour: %d", tod.Time().Hour())
	}

	var ts Timestamp
	if err := ts.Scan("2026-03-04 05:06:07"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ts.Time().Equal(time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)) {
		t.Errorf("unexpected timestamp: %s", ts.Time())
	}
}
