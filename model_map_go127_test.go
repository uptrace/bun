//go:build go1.27

package bun_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/schema"
)

// Regression test for #1434: on Go 1.27 database/sql holds the driver
// connection lock while calling driver.RowsColumnScanner.ScanColumn, so
// scanning into a map must not call rows.ColumnTypes from mapModel.Scan.
func TestScanMapWithRowsColumnScanner(t *testing.T) {
	db := bun.NewDB(sql.OpenDB(columnScannerConnector{}), newTestDialect())
	defer db.Close()

	t.Run("map", func(t *testing.T) {
		m := make(map[string]any)
		runWithTimeout(t, func() error {
			return db.NewSelect().ColumnExpr("1").Scan(context.Background(), &m)
		})
		require.Equal(t, map[string]any{"id": int64(1), "data": []byte("row-1")}, m)
	})

	t.Run("map slice", func(t *testing.T) {
		var ms []map[string]any
		runWithTimeout(t, func() error {
			return db.NewSelect().ColumnExpr("1").Scan(context.Background(), &ms)
		})
		require.Equal(t, []map[string]any{
			{"id": int64(1), "data": []byte("row-1")},
			{"id": int64(2), "data": []byte("row-2")},
		}, ms)
	})
}

func runWithTimeout(t *testing.T, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not return (deadlock?)")
	}
}

type testDialect struct {
	schema.BaseDialect
	tables *schema.Tables
}

func newTestDialect() *testDialect {
	d := new(testDialect)
	d.tables = schema.NewTables(d)
	return d
}

func (d *testDialect) Init(*sql.DB) {}

func (d *testDialect) Name() dialect.Name {
	return dialect.Invalid
}

func (d *testDialect) Features() feature.Feature {
	return 0
}

func (d *testDialect) Tables() *schema.Tables {
	return d.tables
}

func (d *testDialect) OnTable(*schema.Table) {}

func (d *testDialect) IdentQuote() byte {
	return '"'
}

func (d *testDialect) AppendSequence(b []byte, _ *schema.Table, _ *schema.Field) []byte {
	return b
}

func (d *testDialect) DefaultVarcharLen() int {
	return 0
}

func (d *testDialect) DefaultSchema() string {
	return ""
}

type columnScannerConnector struct{}

func (c columnScannerConnector) Connect(context.Context) (driver.Conn, error) {
	return columnScannerConn{}, nil
}

func (c columnScannerConnector) Driver() driver.Driver { return nil }

type columnScannerConn struct{}

func (columnScannerConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (columnScannerConn) Close() error                        { return nil }
func (columnScannerConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }

func (columnScannerConn) QueryContext(
	context.Context, string, []driver.NamedValue,
) (driver.Rows, error) {
	return &columnScannerRows{
		data: [][]driver.Value{
			{int64(1), []byte("row-1")},
			{int64(2), []byte("row-2")},
		},
		row: -1,
	}, nil
}

// columnScannerRows implements driver.RowsColumnScanner the way pgx v5.11 does
// for sql.Scanner destinations: it hands the value to sql.ConvertAssign, which
// calls the destination's Scan method.
type columnScannerRows struct {
	data [][]driver.Value
	row  int
}

var _ driver.RowsColumnScanner = (*columnScannerRows)(nil)

func (r *columnScannerRows) Columns() []string { return []string{"id", "data"} }
func (r *columnScannerRows) Close() error      { return nil }

func (r *columnScannerRows) ColumnTypeScanType(index int) reflect.Type {
	if index == 0 {
		return reflect.TypeFor[int64]()
	}
	return reflect.TypeFor[[]byte]()
}

func (r *columnScannerRows) Next(dest []driver.Value) error {
	if err := r.NextRow(); err != nil {
		return err
	}
	copy(dest, r.data[r.row])
	return nil
}

func (r *columnScannerRows) NextRow() error {
	if r.row+1 >= len(r.data) {
		return io.EOF
	}
	r.row++
	return nil
}

func (r *columnScannerRows) ScanColumn(scanCtx driver.ScanContext, index int, dest any) error {
	return sql.ConvertAssign(scanCtx, dest, r.data[r.row][index])
}
