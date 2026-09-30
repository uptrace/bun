package schema

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/dialect/sqltype"
)

// This file guards against behavior changes for types that are *not* the
// standard library uuid.UUID. Every case below is expected to behave exactly
// as it did before uuid support was added.

// jsonDocWithText implements encoding/json's interfaces *and* the encoding
// text interfaces, and is stored in a jsonb column.
type jsonDocWithText struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (d jsonDocWithText) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		A int    `json:"a"`
		B string `json:"b"`
	}{d.A, d.B})
}

func (d *jsonDocWithText) UnmarshalJSON(b []byte) error {
	var v struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	d.A, d.B = v.A, v.B
	return nil
}

func (d jsonDocWithText) MarshalText() ([]byte, error) { return []byte("text-form"), nil }

func (d *jsonDocWithText) UnmarshalText(b []byte) error {
	d.A, d.B = -1, string(b)
	return nil
}

// binaryOnlyJSON implements json.Unmarshaler and encoding.BinaryUnmarshaler
// only, with no text interface.
type binaryOnlyJSON struct {
	A int `json:"a"`
}

func (d *binaryOnlyJSON) UnmarshalBinary(b []byte) error {
	d.A = 99
	return nil
}

func (d *binaryOnlyJSON) UnmarshalJSON(b []byte) error {
	var v struct {
		A int `json:"a"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	d.A = v.A
	return nil
}

// valuerWithText has a value-receiver text encoder and a pointer-receiver
// driver.Valuer. The valuer must keep winning.
type valuerWithText struct{ n int }

func (v valuerWithText) MarshalText() ([]byte, error)  { return []byte("TEXT"), nil }
func (v *valuerWithText) Value() (driver.Value, error) { return int64(v.n), nil }

// appenderWithText has a value-receiver text encoder and a pointer-receiver
// QueryAppender. The custom appender must keep winning.
type appenderWithText struct{ n int }

func (v appenderWithText) MarshalText() ([]byte, error) { return []byte("TEXT"), nil }
func (v *appenderWithText) AppendQuery(gen QueryGen, b []byte) ([]byte, error) {
	return append(b, "CUSTOM"...), nil
}

// token16 is an unrelated 16-byte type that implements the encoding text
// interfaces. It is not a UUID and must not be treated as one.
type token16 [16]byte

func (t token16) MarshalText() ([]byte, error)        { return []byte("token"), nil }
func (t token16) AppendText(b []byte) ([]byte, error) { return append(b, "token"...), nil }
func (t *token16) UnmarshalText(b []byte) error       { return nil }

// other16 is a 16-byte type with no methods at all.
type other16 [16]byte

func TestCompatJSONDocScan(t *testing.T) {
	dialect := newNopDialect()
	table := NewTables(dialect).Get(reflect.TypeFor[*struct {
		ID  int             `bun:",pk"`
		Doc jsonDocWithText `bun:",type:jsonb"`
	}]())

	var m struct {
		ID  int             `bun:",pk"`
		Doc jsonDocWithText `bun:",type:jsonb"`
	}
	v := reflect.ValueOf(&m).Elem()

	// postgres returns the stored jsonb value as text.
	require.NoError(t, table.FieldMap["doc"].ScanValue(v, `{"a":1,"b":"hello"}`))
	require.Equal(t, jsonDocWithText{A: 1, B: "hello"}, m.Doc,
		"jsonb columns must still decode as JSON, not via UnmarshalText")
}

func TestCompatJSONDocAppend(t *testing.T) {
	dialect := newNopDialect()
	gen := NewQueryGen(dialect)
	table := NewTables(dialect).Get(reflect.TypeFor[*struct {
		ID  int             `bun:",pk"`
		Doc jsonDocWithText `bun:",type:jsonb"`
	}]())

	m := struct {
		ID  int             `bun:",pk"`
		Doc jsonDocWithText `bun:",type:jsonb"`
	}{Doc: jsonDocWithText{A: 1, B: "hello"}}
	v := reflect.ValueOf(&m).Elem()

	require.Equal(t, `'{"a":1,"b":"hello"}'`,
		string(table.FieldMap["doc"].AppendValue(gen, nil, v)))

	// Same rules for a plain query argument.
	require.Equal(t, `'{"a":1,"b":"hello"}'`, string(gen.Append(nil, m.Doc)))
}

func TestCompatBinaryUnmarshalerIgnored(t *testing.T) {
	// A type with only json and binary unmarshalers must keep using JSON;
	// the removed BinaryUnmarshaler support must not resurface.
	var m struct {
		Doc binaryOnlyJSON `bun:",type:jsonb"`
	}
	scanner := Scanner(reflect.TypeFor[binaryOnlyJSON]())
	require.NoError(t, scanner(reflect.ValueOf(&m).Elem().Field(0), `{"a":7}`))
	require.Equal(t, binaryOnlyJSON{A: 7}, m.Doc)
}

func TestCompatCustomInterfacePrecedence(t *testing.T) {
	dialect := newNopDialect()
	gen := NewQueryGen(dialect)

	// Addressable values, as a struct field or slice element would be: the
	// pointer-receiver custom interface must win over the value-receiver
	// text encoder.
	v := valuerWithText{n: 7}
	require.Equal(t, "7",
		string(Appender(dialect, reflect.TypeFor[valuerWithText]())(gen, nil, reflect.ValueOf(&v).Elem())))

	a := appenderWithText{n: 7}
	require.Equal(t, "CUSTOM",
		string(Appender(dialect, reflect.TypeFor[appenderWithText]())(gen, nil, reflect.ValueOf(&a).Elem())))

	// Pointer values.
	require.Equal(t, "7",
		string(Appender(dialect, reflect.TypeFor[*valuerWithText]())(gen, nil, reflect.ValueOf(&v))))
	require.Equal(t, "CUSTOM",
		string(Appender(dialect, reflect.TypeFor[*appenderWithText]())(gen, nil, reflect.ValueOf(&a))))
}

func TestCompatUnrelatedByteArrays(t *testing.T) {
	dialect := newNopDialect()
	gen := NewQueryGen(dialect)

	// An unrelated [16]byte type with text methods is not a UUID.
	tok := token16{1, 2, 3}
	require.Equal(t, `'\x01020300000000000000000000000000'`,
		string(Appender(dialect, reflect.TypeFor[token16]())(gen, nil, reflect.ValueOf(tok))))
	require.Equal(t, "", DiscoverSQLType(reflect.TypeFor[token16]()))

	// Plain byte arrays keep the default array scanner (which is nil).
	for _, typ := range []reflect.Type{
		reflect.TypeFor[other16](),
		reflect.TypeFor[[8]byte](),
		reflect.TypeFor[[32]byte](),
	} {
		require.Nil(t, Scanner(typ), "Scanner(%s) must stay nil", typ)
		require.Equal(t, "", DiscoverSQLType(typ))
	}

	// And the shared toBytes helper must not accept arbitrary byte arrays.
	require.Equal(t, `'\x01020300000000000000000000000000'`,
		string(Appender(dialect, reflect.TypeFor[other16]())(gen, nil, reflect.ValueOf(other16{1, 2, 3}))))
}

// TestCompatDiscoverSQLType pins the discovery rules that existed before uuid
// support, so the uuid special case cannot quietly displace them.
func TestCompatDiscoverSQLType(t *testing.T) {
	require.Equal(t, sqltype.Blob, DiscoverSQLType(reflect.TypeFor[[]byte]()))
	require.Equal(t, sqltype.Blob, DiscoverSQLType(reflect.TypeFor[sql.RawBytes]()))
	require.Equal(t, sqltype.JSON, DiscoverSQLType(reflect.TypeFor[json.RawMessage]()))
	require.Equal(t, sqltype.VarChar, DiscoverSQLType(reflect.TypeFor[string]()))
	require.Equal(t, sqltype.VarChar, DiscoverSQLType(reflect.TypeFor[[]string]()))
	require.Equal(t, sqltype.VarChar, DiscoverSQLType(reflect.TypeFor[map[string]string]()))
	require.Equal(t, sqltype.Timestamp, DiscoverSQLType(reflect.TypeFor[time.Time]()))

	// Byte arrays keep the empty discovery they had on master.
	require.Equal(t, "", DiscoverSQLType(reflect.TypeFor[[16]byte]()))
	require.Equal(t, "", DiscoverSQLType(reflect.TypeFor[[32]byte]()))
}
