package pgdialect

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/alt"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

func (d *Dialect) Migrator(db *bun.DB) sqlschema.Migrator {
	return &Migrator{db: db, BaseMigrator: sqlschema.NewBaseMigrator(db)}
}

type Migrator struct {
	*sqlschema.BaseMigrator

	db *bun.DB
}

var _ sqlschema.Migrator = (*Migrator)(nil)

func (m *Migrator) execRaw(ctx context.Context, q *bun.RawQuery) error {
	if _, err := q.Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) RenameTable(ctx context.Context, oldName, newName string) error {
	q := m.db.NewRaw("ALTER TABLE ? RENAME TO ?", bun.Ident(oldName), bun.Ident(newName))
	return m.execRaw(ctx, q)
}

func (m *Migrator) AddContraint(ctx context.Context, fk sqlschema.FK, name string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? ADD CONSTRAINT ? FOREIGN KEY (?) REFERENCES ?.? (?)",
		bun.Safe(fk.From.Schema), bun.Safe(fk.From.Table), bun.Safe(name),
		bun.Safe(fk.From.Column.String()),
		bun.Safe(fk.To.Schema), bun.Safe(fk.To.Table),
		bun.Safe(fk.To.Column.String()),
	)
	return m.execRaw(ctx, q)
}

func (m *Migrator) DropContraint(ctx context.Context, schema, table, name string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? DROP CONSTRAINT ?",
		bun.Ident(schema), bun.Ident(table), bun.Ident(name),
	)
	return m.execRaw(ctx, q)
}

func (m *Migrator) RenameConstraint(ctx context.Context, schema, table, oldName, newName string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? RENAME CONSTRAINT ? TO ?",
		bun.Ident(schema), bun.Ident(table), bun.Ident(oldName), bun.Ident(newName),
	)
	return m.execRaw(ctx, q)
}

func (m *Migrator) RenameColumn(ctx context.Context, schema, table, oldName, newName string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? RENAME COLUMN ? TO ?",
		bun.Ident(schema), bun.Ident(table), bun.Ident(oldName), bun.Ident(newName),
	)
	return m.execRaw(ctx, q)
}

// -------------

func (m *Migrator) Apply(ctx context.Context, changes ...sqlschema.Operation) error {
	if len(changes) == 0 {
		return nil
	}

	queries, err := m.buildQueries(changes...)
	if err != nil {
		return fmt.Errorf("apply database schema changes: %w", err)
	}

	for _, query := range queries {
		var b []byte
		if b, err = query.AppendQuery(m.db.Formatter(), b); err != nil {
			return err
		}
		m.execRaw(ctx, m.db.NewRaw(string(b)))
	}

	return nil
}

// buildQueries combines schema changes to a number of ALTER TABLE queries.
func (m *Migrator) buildQueries(changes ...sqlschema.Operation) ([]*AlterTableQuery, error) {
	var queries []*AlterTableQuery

	chain := func(change sqlschema.Operation) error {
		for _, query := range queries {
			if err := query.Chain(change); err != errCannotChain {
				return err // either nil (successful) or non-nil (failed)
			}
		}

		// Create a new query for this change, since it cannot be chained to any of the existing ones.
		q, err := newAlterTableQuery(change)
		if err != nil {
			return err
		}
		queries = append(queries, q.Sep())
		return nil
	}

	for _, change := range changes {
		if err := chain(change); err != nil {
			return nil, err
		}
	}
	return queries, nil
}

type AlterTableQuery struct {
	FQN schema.FQN

	RenameTable      sqlschema.Operation
	RenameColumn     sqlschema.Operation
	RenameConstraint sqlschema.Operation
	Actions          Actions

	separate bool
}

type Actions []*Action

var _ schema.QueryAppender = (*Actions)(nil)

type Action struct {
	AddColumn      sqlschema.Operation
	DropColumn     sqlschema.Operation
	AlterColumn    sqlschema.Operation
	AlterType      sqlschema.Operation
	SetDefault     sqlschema.Operation
	DropDefault    sqlschema.Operation
	SetNotNull     sqlschema.Operation
	DropNotNull    sqlschema.Operation
	AddGenerated   sqlschema.Operation
	AddConstraint  sqlschema.Operation
	DropConstraint sqlschema.Operation
	Custom         sqlschema.Operation
}

var _ schema.QueryAppender = (*Action)(nil)

func newAlterTableQuery(op sqlschema.Operation) (*AlterTableQuery, error) {
	q := AlterTableQuery{
		FQN: op.FQN(),
	}
	switch op.(type) {
	case *alt.RenameTable:
		q.RenameTable = op
	case *alt.RenameColumn:
		q.RenameColumn = op
	case *alt.RenameConstraint:
		q.RenameConstraint = op
	default:
		q.Actions = append(q.Actions, newAction(op))
	}
	return &q, nil
}

func newAction(op sqlschema.Operation) *Action {
	var a Action
	return &a
}

// errCannotChain is a sentinel error. To apply the change, callers should
// create a new AlterTableQuery instead and include it there.
var errCannotChain = errors.New("cannot chain change to the current query")

func (q *AlterTableQuery) Chain(op sqlschema.Operation) error {
	if op.FQN() != q.FQN {
		return errCannotChain
	}

	switch op.(type) {
	default:
		return fmt.Errorf("unsupported operation %T", op)
	}
}

func (q *AlterTableQuery) isEmpty() bool {
	return q.RenameTable == nil && q.RenameColumn == nil && q.RenameConstraint == nil && len(q.Actions) == 0
}

// Sep appends a ";" separator at the end of the query.
func (q *AlterTableQuery) Sep() *AlterTableQuery {
	q.separate = true
	return q
}

func (q *AlterTableQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	var op schema.QueryAppender
	switch true {
	case q.RenameTable != nil:
		op = q.RenameTable
	case q.RenameColumn != nil:
		op = q.RenameColumn
	case q.RenameConstraint != nil:
		op = q.RenameConstraint
	case len(q.Actions) > 0:
		op = q.Actions
	default:
		return b, nil
	}
	b = append(b, "ALTER TABLE "...)
	b, _ = q.FQN.AppendQuery(fmter, b)
	b = append(b, " "...)
	if b, err = op.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	if q.separate {
		b = append(b, ";"...)
	}
	return b, nil
}

func (actions Actions) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	for i, a := range actions {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = a.AppendQuery(fmter, b)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func (a *Action) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	var op schema.QueryAppender
	switch true {
	case a.AddColumn != nil:
		op = a.AddColumn
	case a.DropColumn != nil:
		op = a.DropColumn
	case a.AlterColumn != nil:
		op = a.AlterColumn
	case a.AlterType != nil:
		op = a.AlterType
	case a.SetDefault != nil:
		op = a.SetDefault
	case a.DropDefault != nil:
		op = a.DropDefault
	case a.SetNotNull != nil:
		op = a.SetNotNull
	case a.DropNotNull != nil:
		op = a.DropNotNull
	case a.AddGenerated != nil:
		op = a.AddGenerated
	case a.AddConstraint != nil:
		op = a.AddConstraint
	case a.DropConstraint != nil:
		op = a.DropConstraint
	default:
		return b, nil
	}
	return op.AppendQuery(fmter, b)
}
