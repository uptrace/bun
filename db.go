package bun

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/sqlfmt"
)

const (
	discardUnknownColumns internal.Flag = 1 << iota
)

type config struct{}

type ConfigOption func(cfg *config)

type DB struct {
	*sql.DB
	dialect  schema.Dialect
	features feature.Feature
	cfg      config

	queryHooks []QueryHook

	fmter sqlfmt.Formatter
	flags internal.Flag
}

func Open(sqldb *sql.DB, dialect schema.Dialect, opts ...ConfigOption) *DB {
	db := &DB{
		DB:       sqldb,
		dialect:  dialect,
		features: dialect.Features(),
		fmter:    sqlfmt.NewFormatter(dialect.Features()),
	}

	for _, opt := range opts {
		opt(&db.cfg)
	}

	return db
}

func (db *DB) DiscardUnknownColumns() {
	db.flags = db.flags.Set(discardUnknownColumns)
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

func (db *DB) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(db)
}

func (db *DB) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(db)
}

func (db *DB) Dialect() schema.Dialect {
	return db.dialect
}

func (db *DB) ScanRows(ctx context.Context, rows *sql.Rows, dest ...interface{}) error {
	defer rows.Close()

	model, err := newModel(db, dest)
	if err != nil {
		return err
	}

	_, err = model.ScanRows(ctx, rows)
	return err
}

func (db *DB) AddQueryHook(hook QueryHook) {
	db.queryHooks = append(db.queryHooks, hook)
}

func (db *DB) Table(typ reflect.Type) *schema.Table {
	return db.dialect.Tables().Get(typ)
}

func (db *DB) WithArg(name string, value interface{}) *DB {
	clone := db.clone()
	clone.fmter = clone.fmter.WithArg(name, value)
	return clone
}

func (db *DB) clone() *DB {
	clone := *db

	l := len(clone.queryHooks)
	clone.queryHooks = clone.queryHooks[:l:l]

	return &clone
}

func (db *DB) Formatter() sqlfmt.Formatter {
	return db.fmter
}

//------------------------------------------------------------------------------

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

func (db *DB) ExecContext(
	ctx context.Context, query string, args ...interface{},
) (sql.Result, error) {
	ctx, event := db.beforeQuery(ctx, nil, query, args)
	res, err := db.DB.ExecContext(ctx, query, args...)
	db.afterQuery(ctx, event, res, err)

	return res, err
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

func (db *DB) QueryContext(
	ctx context.Context, query string, args ...interface{},
) (*sql.Rows, error) {
	ctx, event := db.beforeQuery(ctx, nil, query, args)
	rows, err := db.DB.QueryContext(ctx, query, args...)
	db.afterQuery(ctx, event, nil, err)

	return rows, err
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, event := db.beforeQuery(ctx, nil, query, args)
	row := db.DB.QueryRowContext(ctx, query, args...)
	db.afterQuery(ctx, event, nil, row.Err())
	return row
}

type Conn struct {
	*sql.Conn
}

func (db *DB) Conn(ctx context.Context) (Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return Conn{}, err
	}
	return Conn{Conn: conn}, nil
}

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

type Tx struct {
	*sql.Tx
}

func (db *DB) Begin() (Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}
	return Tx{Tx: tx}, nil
}

//------------------------------------------------------------------------------

type Result struct {
	r sql.Result
	n int
}

func (r Result) RowsAffected() (int64, error) {
	if r.r != nil {
		return r.r.RowsAffected()
	}
	return int64(r.n), nil
}

func (r Result) LastInsertId() (int64, error) {
	if r.r != nil {
		return r.r.LastInsertId()
	}
	return 0, errors.New("LastInsertId is not available")
}
