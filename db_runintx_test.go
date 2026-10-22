package bun

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/schema"
)

// Regression test for https://github.com/uptrace/bun/issues/1458: nested Tx.RunInTx
// must pass the savepoint's context (from the caller) into the callback, not the
// outer transaction's context.

type runInTxCtxKey struct{}

type runInTxStubDriver struct{}

func (runInTxStubDriver) Open(string) (driver.Conn, error) {
	return runInTxStubConn{}, nil
}

type runInTxStubConn struct{}

func (runInTxStubConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (runInTxStubConn) Close() error { return nil }

func (runInTxStubConn) Begin() (driver.Tx, error) {
	return runInTxStubTx{}, nil
}

func (runInTxStubConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return runInTxStubTx{}, nil
}

func (runInTxStubConn) ExecContext(
	context.Context, string, []driver.NamedValue,
) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (runInTxStubConn) QueryContext(
	context.Context, string, []driver.NamedValue,
) (driver.Rows, error) {
	return runInTxStubRows{}, nil
}

type runInTxStubTx struct{}

func (runInTxStubTx) Commit() error   { return nil }
func (runInTxStubTx) Rollback() error { return nil }

type runInTxStubRows struct{}

func (runInTxStubRows) Columns() []string              { return nil }
func (runInTxStubRows) Close() error                   { return nil }
func (runInTxStubRows) Next(dest []driver.Value) error { return io.EOF }

func init() {
	sql.Register("bun-nested-runintx-stub", runInTxStubDriver{})
}

type runInTxTestDialect struct {
	schema.BaseDialect
	tables *schema.Tables
}

func newRunInTxTestDialect() *runInTxTestDialect {
	d := new(runInTxTestDialect)
	d.tables = schema.NewTables(d)
	return d
}

func (*runInTxTestDialect) Init(*sql.DB)              {}
func (*runInTxTestDialect) Name() dialect.Name        { return dialect.PG }
func (*runInTxTestDialect) Features() feature.Feature { return 0 }
func (d *runInTxTestDialect) Tables() *schema.Tables  { return d.tables }
func (*runInTxTestDialect) OnField(*schema.Field)     {}
func (*runInTxTestDialect) OnTable(*schema.Table)     {}
func (*runInTxTestDialect) IdentQuote() byte          { return '"' }
func (*runInTxTestDialect) DefaultVarcharLen() int    { return 0 }
func (*runInTxTestDialect) DefaultSchema() string     { return "public" }
func (*runInTxTestDialect) AppendSequence([]byte, *schema.Table, *schema.Field) []byte {
	return nil
}

func TestTxRunInTxPassesCallerContext(t *testing.T) {
	sqldb, err := sql.Open("bun-nested-runintx-stub", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })

	db := NewDB(sqldb, newRunInTxTestDialect())
	t.Cleanup(func() { _ = db.Close() })

	outerCtx := context.Background()

	err = db.RunInTx(outerCtx, nil, func(_ context.Context, tx Tx) error {
		c, cancel := context.WithTimeout(
			context.WithValue(outerCtx, runInTxCtxKey{}, "v"),
			time.Minute,
		)
		defer cancel()

		return tx.RunInTx(c, nil, func(inner context.Context, _ Tx) error {
			require.Equal(t, "v", inner.Value(runInTxCtxKey{}))
			_, hasDeadline := inner.Deadline()
			require.True(t, hasDeadline)
			return nil
		})
	})
	require.NoError(t, err)
}
