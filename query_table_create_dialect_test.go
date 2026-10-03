package bun

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/schema"
)

type dialectTypeModel struct {
	Level   int16   `bun:"type:tinyint;pg=smallint;mssql=tinyint;oracle=NUMBER(3)"`
	Amount  float64 `bun:"type:decimal(10,2);pg=numeric(12,4);sqlite=REAL"`
	Payload string  `bun:"type:text;mysql=json;oracle=CLOB"`
	Extra   string  `bun:"type:varchar(64)"`
}

func TestCreateTableDialectSQLTypes(t *testing.T) {
	tests := []struct {
		name        string // dialect name, e.g. pg, mysql, sqlite
		dialect     dialect.Name
		levelType   string // type used for the dialect-specific column
		amountType  string
		payloadType string
	}{
		{name: "pg", dialect: dialect.PG, levelType: "smallint", amountType: "numeric(12,4)", payloadType: "text"},
		{name: "mysql", dialect: dialect.MySQL, levelType: "tinyint", amountType: "decimal(10,2)", payloadType: "json"},
		{name: "sqlite", dialect: dialect.SQLite, levelType: "tinyint", amountType: "REAL", payloadType: "text"},
		{name: "mssql", dialect: dialect.MSSQL, levelType: "tinyint", amountType: "decimal(10,2)", payloadType: "text"},
		{name: "oracle", dialect: dialect.Oracle, levelType: "NUMBER(3)", amountType: "decimal(10,2)", payloadType: "CLOB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newDialectTypeTestDB(tt.dialect)

			query := db.NewCreateTable().Model(&dialectTypeModel{})
			sqlStr := query.String()

			sqlLower := strings.ToLower(sqlStr)
			require.Contains(t, sqlLower, "\"level\" "+strings.ToLower(tt.levelType))
			require.Contains(t, sqlLower, "\"amount\" "+strings.ToLower(tt.amountType))
			require.Contains(t, sqlLower, "\"payload\" "+strings.ToLower(tt.payloadType))
			require.Contains(t, sqlLower, "\"extra\" varchar(64)")
		})
	}
}

func TestCreateTableDialectSQLTypesFallback(t *testing.T) {
	// A dialect without a matching tag option falls back to the generic type,
	// and the generic type alone behaves exactly as before.
	db := newDialectTypeTestDB(dialect.SQLite)

	query := db.NewCreateTable().Model(&fallbackModel{})
	sqlStr := query.String()

	sqlLower := strings.ToLower(sqlStr)
	require.Contains(t, sqlLower, "\"level\" tinyint")
	require.Contains(t, sqlLower, "\"payload\" jsonb")
	require.Contains(t, sqlLower, "\"extra\" varchar(64)")
}

type fallbackModel struct {
	Level   int16  `bun:"type:tinyint;pg=smallint"`
	Payload string `bun:"type:jsonb;mysql=json"`
	Extra   string `bun:"type:varchar(64)"`
}

func newDialectTypeTestDB(name dialect.Name) *DB {
	d := &testDialect{name: name}
	d.tables = schema.NewTables(d)
	return NewDB(sql.OpenDB(nopConnector{}), d)
}

// testDialect implements schema.Dialect with just enough behavior to build
// create-table queries without a real connection.
type testDialect struct {
	schema.BaseDialect

	name   dialect.Name
	tables *schema.Tables
}

var _ schema.Dialect = (*testDialect)(nil)

func (d *testDialect) Init(*sql.DB) {}

func (d *testDialect) Name() dialect.Name { return d.name }

func (d *testDialect) Features() feature.Feature { return 0 }

func (d *testDialect) Tables() *schema.Tables { return d.tables }

func (d *testDialect) OnTable(*schema.Table) {}

func (d *testDialect) IdentQuote() byte { return '"' }

func (d *testDialect) AppendSequence(b []byte, _ *schema.Table, _ *schema.Field) []byte { return b }

func (d *testDialect) DefaultVarcharLen() int { return 0 }

func (d *testDialect) DefaultSchema() string { return "" }

type nopConnector struct{}

var errNotImplemented = errors.New("bun: not implemented")

func (nopConnector) Connect(context.Context) (driver.Conn, error) { return nil, errNotImplemented }

func (c nopConnector) Driver() driver.Driver { return nopDriver{} }

type nopDriver struct{}

func (nopDriver) Open(string) (driver.Conn, error) { return nil, errNotImplemented }
