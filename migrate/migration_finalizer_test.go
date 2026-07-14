package migrate

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
	"testing/fstest"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/schema"
)

// Regression test for https://github.com/uptrace/bun/issues/1389: SQL migration
// finalizer errors (tx.Commit / tx.Rollback / conn.Close) were discarded
// because the migration ran with an unnamed return.

var (
	errCommitFailed = errors.New("commit failed")
	errExecFailed   = errors.New("exec failed")
)

// failingDriver is a database/sql driver whose transactions fail to commit, and
// which optionally fails ExecContext, so we can observe how the migration func
// reports finalizer errors.
type failingDriver struct{ failExec bool }

func (d failingDriver) Open(string) (driver.Conn, error) {
	return failingConn(d), nil
}

type failingConn struct{ failExec bool }

func (failingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (failingConn) Close() error              { return nil }
func (failingConn) Begin() (driver.Tx, error) { return failingTx{}, nil }
func (failingConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return failingTx{}, nil
}

func (c failingConn) ExecContext(
	context.Context, string, []driver.NamedValue,
) (driver.Result, error) {
	if c.failExec {
		return nil, errExecFailed
	}
	return driver.ResultNoRows, nil
}

func (failingConn) QueryContext(
	context.Context, string, []driver.NamedValue,
) (driver.Rows, error) {
	return emptyRows{}, nil
}

type failingTx struct{}

func (failingTx) Commit() error   { return errCommitFailed }
func (failingTx) Rollback() error { return nil }

type emptyRows struct{}

func (emptyRows) Columns() []string              { return nil }
func (emptyRows) Close() error                   { return nil }
func (emptyRows) Next(dest []driver.Value) error { return io.EOF }

func init() {
	sql.Register("bun-migrate-failcommit", failingDriver{})
	sql.Register("bun-migrate-failexec", failingDriver{failExec: true})
}

// testDialect is a minimal dialect (mirroring bun's internal nopDialect) so the
// migrator can be constructed without pulling in a dialect module.
type testDialect struct {
	schema.BaseDialect
	tables *schema.Tables
}

func newTestDialect() *testDialect {
	d := new(testDialect)
	d.tables = schema.NewTables(d)
	return d
}

func (*testDialect) Init(*sql.DB)              {}
func (*testDialect) Name() dialect.Name        { return dialect.SQLite }
func (*testDialect) Features() feature.Feature { return 0 }
func (d *testDialect) Tables() *schema.Tables  { return d.tables }
func (*testDialect) OnField(*schema.Field)     {}
func (*testDialect) OnTable(*schema.Table)     {}
func (*testDialect) IdentQuote() byte          { return '"' }
func (*testDialect) DefaultVarcharLen() int    { return 0 }
func (*testDialect) DefaultSchema() string     { return "main" }
func (*testDialect) AppendSequence([]byte, *schema.Table, *schema.Field) []byte {
	return nil
}

func newTestMigrator(t *testing.T, driverName string) *Migrator {
	t.Helper()

	sqldb, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })

	db := bun.NewDB(sqldb, newTestDialect())
	return NewMigrator(db, NewMigrations())
}

func runSQLMigration(t *testing.T, migrator *Migrator) error {
	t.Helper()

	const name = "20260101000000_test.tx.up.sql"
	fsys := fstest.MapFS{
		name: &fstest.MapFile{Data: []byte("SELECT 1;")},
	}

	fn := newSQLMigrationFunc(fsys, name)
	return fn(context.Background(), migrator, &Migration{})
}

func TestSQLMigrationReturnsCommitError(t *testing.T) {
	migrator := newTestMigrator(t, "bun-migrate-failcommit")

	err := runSQLMigration(t, migrator)

	if !errors.Is(err, errCommitFailed) {
		t.Fatalf("expected the commit error to be returned, got: %v", err)
	}
}

func TestSQLMigrationPreservesExecError(t *testing.T) {
	migrator := newTestMigrator(t, "bun-migrate-failexec")

	err := runSQLMigration(t, migrator)

	// The execution error must still be reported when the body fails.
	if !errors.Is(err, errExecFailed) {
		t.Fatalf("expected the exec error to be returned, got: %v", err)
	}
}
