package schema

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/uptrace/bun/dialect/sqltype"
)

type xDialect struct {
	*nopDialect
}

func (xDialect) AppendBytes(b, bs []byte) []byte {
	b = append(b, "X'"...)
	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)
	b = append(b, '\'')
	return b
}

type msgpackPayload struct {
	Something int    `msgpack:"something"`
	Name      string `msgpack:"name"`
}

type msgpackModel struct {
	ID      int64          `bun:",pk,autoincrement"`
	Encoded msgpackPayload `bun:",msgpack"`
}

func TestAppendMsgpack_Dialect(t *testing.T) {
	d := xDialect{newNopDialect()}
	gen := NewQueryGen(d)
	tables := NewTables(d)
	table := tables.Get(reflect.TypeOf(msgpackModel{}))

	field, err := table.Field("encoded")
	require.NoError(t, err)
	require.Equal(t, sqltype.Blob, field.DiscoveredSQLType, "msgpack field must discover sqltype.Blob")

	payload := msgpackPayload{Something: 42, Name: "hello"}
	b, err := msgpack.Marshal(payload)
	require.NoError(t, err)

	val := reflect.ValueOf(payload)
	appended := string(field.Append(gen, nil, val))

	require.True(t, strings.HasPrefix(appended, "X'"), "expected X'...' prefix for xDialect, got: %s", appended)
	want := string(d.AppendBytes(nil, b))
	require.Equal(t, want, appended)
}
