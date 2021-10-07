package bun

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

const (
	discardUnknownColumns internal.Flag = 1 << iota
)

type DBStats struct {
	Queries uint32
	Errors  uint32
}

type DBOption func(db *DB)

func WithDiscardUnknownColumns() DBOption {
	return func(db *DB) {
		db.flags = db.flags.Set(discardUnknownColumns)
	}
}

type DB struct {
	*sql.DB
	dialect  schema.Dialect
	features feature.Feature

	queryHooks []QueryHook

	fmter schema.Formatter
	flags internal.Flag

	stats DBStats
}

func NewDB(sqldb *sql.DB, dialect schema.Dialect, opts ...DBOption) *DB {
	dialect.Init(sqldb)

	db := &DB{
		DB:       sqldb,
		dialect:  dialect,
		features: dialect.Features(),
		fmter:    schema.NewFormatter(dialect),
	}

	for _, opt := range opts {
		opt(db)
	}

	return db
}

func (db *DB) String() string {
	var b strings.Builder
	b.WriteString("DB<dialect=")
	b.WriteString(db.dialect.Name().String())
	b.WriteString(">")
	return b.String()
}

func (db *DB) DBStats() DBStats {
	return DBStats{
		Queries: atomic.LoadUint32(&db.stats.Queries),
		Errors:  atomic.LoadUint32(&db.stats.Errors),
	}
}

func (db *DB) NewValues(model interface{}) *ValuesQuery {
	return NewValuesQuery(db, model)
}

func (db *DB) NewSelect() *SelectQuery {
	return NewSelectQuery(db)
}

func (db *DB) NewInsert() *InsertQuery {
	return NewInsertQuery(db)
}

func (db *DB) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(db)
}

func (db *DB) NewDelete() *DeleteQuery {
	return NewDeleteQuery(db)
}

func (db *DB) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(db)
}

func (db *DB) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(db)
}

func (db *DB) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(db)
}

func (db *DB) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(db)
}

func (db *DB) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(db)
}

func (db *DB) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(db)
}

func (db *DB) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(db)
}

func (db *DB) ResetModel(ctx context.Context, models ...interface{}) error {
	for _, model := range models {
		if _, err := db.NewDropTable().Model(model).IfExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model(model).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) Dialect() schema.Dialect {
	return db.dialect
}

func (db *DB) ScanRows(ctx context.Context, rows *sql.Rows, dest ...interface{}) error {
	model, err := newModel(db, dest)
	if err != nil {
		return err
	}

	_, err = model.ScanRows(ctx, rows)
	return err
}

func (db *DB) ScanRow(ctx context.Context, rows *sql.Rows, dest ...interface{}) error {
	model, err := newModel(db, dest)
	if err != nil {
		return err
	}

	rs, ok := model.(rowScanner)
	if !ok {
		return fmt.Errorf("bun: %T does not support ScanRow", model)
	}

	return rs.ScanRow(ctx, rows)
}

type queryHookIniter interface {
	Init(db *DB)
}

func (db *DB) AddQueryHook(hook QueryHook) {
	if initer, ok := hook.(queryHookIniter); ok {
		initer.Init(db)
	}
	db.queryHooks = append(db.queryHooks, hook)
}

func (db *DB) Table(typ reflect.Type) *schema.Table {
	return db.dialect.Tables().Get(typ)
}

func (db *DB) RegisterModel(models ...interface{}) {
	db.dialect.Tables().Register(models...)
}

func (db *DB) clone() *DB {
	clone := *db

	l := len(clone.queryHooks)
	clone.queryHooks = clone.queryHooks[:l:l]

	return &clone
}

func (db *DB) WithNamedArg(name string, value interface{}) *DB {
	clone := db.clone()
	clone.fmter = clone.fmter.WithNamedArg(name, value)
	return clone
}

func (db *DB) Formatter() schema.Formatter {
	return db.fmter
}

//------------------------------------------------------------------------------

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

func (db *DB) ExecContext(
	ctx context.Context, query string, args ...interface{},
) (sql.Result, error) {
	ctx, event := db.beforeQuery(ctx, nil, query, args, nil)
	res, err := db.DB.ExecContext(ctx, db.format(query, args))
	db.afterQuery(ctx, event, res, err)
	return res, err
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

func (db *DB) QueryContext(
	ctx context.Context, query string, args ...interface{},
) (*sql.Rows, error) {
	ctx, event := db.beforeQuery(ctx, nil, query, args, nil)
	rows, err := db.DB.QueryContext(ctx, db.format(query, args))
	db.afterQuery(ctx, event, nil, err)
	return rows, err
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, event := db.beforeQuery(ctx, nil, query, args, nil)
	row := db.DB.QueryRowContext(ctx, db.format(query, args))
	db.afterQuery(ctx, event, nil, row.Err())
	return row
}

func (db *DB) format(query string, args []interface{}) string {
	return db.fmter.FormatQuery(query, args...)
}

//------------------------------------------------------------------------------

type Conn struct {
	db *DB
	*sql.Conn
}

func (db *DB) Conn(ctx context.Context) (Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return Conn{}, err
	}
	return Conn{
		db:   db,
		Conn: conn,
	}, nil
}

func (c Conn) ExecContext(
	ctx context.Context, query string, args ...interface{},
) (sql.Result, error) {
	ctx, event := c.db.beforeQuery(ctx, nil, query, args, nil)
	res, err := c.Conn.ExecContext(ctx, c.db.format(query, args))
	c.db.afterQuery(ctx, event, res, err)
	return res, err
}

func (c Conn) QueryContext(
	ctx context.Context, query string, args ...interface{},
) (*sql.Rows, error) {
	ctx, event := c.db.beforeQuery(ctx, nil, query, args, nil)
	rows, err := c.Conn.QueryContext(ctx, c.db.format(query, args))
	c.db.afterQuery(ctx, event, nil, err)
	return rows, err
}

func (c Conn) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, event := c.db.beforeQuery(ctx, nil, query, args, nil)
	row := c.Conn.QueryRowContext(ctx, c.db.format(query, args))
	c.db.afterQuery(ctx, event, nil, row.Err())
	return row
}

func (c Conn) NewValues(model interface{}) *ValuesQuery {
	return NewValuesQuery(c.db, model).Conn(c)
}

func (c Conn) NewSelect() *SelectQuery {
	return NewSelectQuery(c.db).Conn(c)
}

func (c Conn) NewInsert() *InsertQuery {
	return NewInsertQuery(c.db).Conn(c)
}

func (c Conn) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(c.db).Conn(c)
}

func (c Conn) NewDelete() *DeleteQuery {
	return NewDeleteQuery(c.db).Conn(c)
}

func (c Conn) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(c.db).Conn(c)
}

func (c Conn) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(c.db).Conn(c)
}

func (c Conn) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(c.db).Conn(c)
}

func (c Conn) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(c.db).Conn(c)
}

func (c Conn) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(c.db).Conn(c)
}

func (c Conn) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(c.db).Conn(c)
}

func (c Conn) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(c.db).Conn(c)
}

//------------------------------------------------------------------------------

type Stmt struct {
	*sql.Stmt
}

func (db *DB) Prepare(query string) (Stmt, error) {
	return db.PrepareContext(context.Background(), query)
}

func (db *DB) PrepareContext(ctx context.Context, query string) (Stmt, error) {
	stmt, err := db.DB.PrepareContext(ctx, query)
	if err != nil {
		return Stmt{}, err
	}
	return Stmt{Stmt: stmt}, nil
}

//------------------------------------------------------------------------------

type Tx struct {
	db *DB
	*sql.Tx
}

// RunInTx runs the function in a transaction. If the function returns an error,
// the transaction is rolled back. Otherwise, the transaction is committed.
func (db *DB) RunInTx(
	ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context, tx Tx) error,
) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) Begin() (Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}
	return Tx{
		db: db,
		Tx: tx,
	}, nil
}

func (tx Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	return tx.ExecContext(context.TODO(), query, args...)
}

func (tx Tx) ExecContext(
	ctx context.Context, query string, args ...interface{},
) (sql.Result, error) {
	ctx, event := tx.db.beforeQuery(ctx, nil, query, args, nil)
	res, err := tx.Tx.ExecContext(ctx, tx.db.format(query, args))
	tx.db.afterQuery(ctx, event, res, err)
	return res, err
}

func (tx Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return tx.QueryContext(context.TODO(), query, args...)
}

func (tx Tx) QueryContext(
	ctx context.Context, query string, args ...interface{},
) (*sql.Rows, error) {
	ctx, event := tx.db.beforeQuery(ctx, nil, query, args, nil)
	rows, err := tx.Tx.QueryContext(ctx, tx.db.format(query, args))
	tx.db.afterQuery(ctx, event, nil, err)
	return rows, err
}

func (tx Tx) QueryRow(query string, args ...interface{}) *sql.Row {
	return tx.QueryRowContext(context.TODO(), query, args...)
}

func (tx Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, event := tx.db.beforeQuery(ctx, nil, query, args, nil)
	row := tx.Tx.QueryRowContext(ctx, tx.db.format(query, args))
	tx.db.afterQuery(ctx, event, nil, row.Err())
	return row
}

//------------------------------------------------------------------------------

func (tx Tx) NewValues(model interface{}) *ValuesQuery {
	return NewValuesQuery(tx.db, model).Conn(tx)
}

func (tx Tx) NewSelect() *SelectQuery {
	return NewSelectQuery(tx.db).Conn(tx)
}

func (tx Tx) NewInsert() *InsertQuery {
	return NewInsertQuery(tx.db).Conn(tx)
}

func (tx Tx) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(tx.db).Conn(tx)
}

func (tx Tx) NewDelete() *DeleteQuery {
	return NewDeleteQuery(tx.db).Conn(tx)
}

func (tx Tx) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(tx.db).Conn(tx)
}

func (tx Tx) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(tx.db).Conn(tx)
}

func (tx Tx) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(tx.db).Conn(tx)
}

func (tx Tx) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(tx.db).Conn(tx)
}

func (tx Tx) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(tx.db).Conn(tx)
}

func (tx Tx) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(tx.db).Conn(tx)
}

func (tx Tx) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(tx.db).Conn(tx)
}

//------------------------------------------------------------------------------

func (db *DB) makeQueryBytes() []byte {
	// TODO: make this configurable?
	return make([]byte, 0, 4096)
}
