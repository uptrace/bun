package dbtest_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var errFormatErrorValue = errors.New("dbtest: value cannot be formatted")

type formatErrorValuer struct{}

func (formatErrorValuer) Value() (driver.Value, error) { return nil, errFormatErrorValue }

// A value bun cannot format must fail the executor with a *schema.FormatError
// before anything is sent, instead of reaching the database as a marker.
func TestFormatError(t *testing.T) {
	testEachDB(t, testFormatError)
}

func testFormatError(t *testing.T, dbName string, db *bun.DB) {
	type FormatErrorModel struct {
		ID   int64 `bun:",pk,autoincrement"`
		Name string
	}

	mustResetModel(t, ctx, db, (*FormatErrorModel)(nil))

	var sent []string
	db = db.WithQueryHook(&queryHook{
		beforeQuery: func(ctx context.Context, event *bun.QueryEvent) context.Context {
			sent = append(sent, event.Query)
			return ctx
		},
	})

	requireNotSent := func(t *testing.T, err, want error) {
		t.Helper()
		require.ErrorIs(t, err, want)
		var fe *schema.FormatError
		require.ErrorAs(t, err, &fe)
		require.Empty(t, sent, "the statement reached the database")
	}

	// testExecutors runs every executor with a value it cannot format.
	testExecutors := func(t *testing.T, value any, want error) {
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Close()) }()

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer func() { require.NoError(t, tx.Rollback()) }()

		executors := []struct {
			name string
			run  func() error
		}{
			{"SelectQuery.Scan", func() error {
				var models []FormatErrorModel
				return db.NewSelect().Model(&models).Where("name = ?", value).Scan(ctx)
			}},
			{"SelectQuery.Rows", func() error {
				rows, err := db.NewSelect().Model((*FormatErrorModel)(nil)).Where("name = ?", value).Rows(ctx)
				if err != nil {
					return err
				}
				return rows.Close()
			}},
			{"SelectQuery.Count", func() error {
				_, err := db.NewSelect().Model((*FormatErrorModel)(nil)).Where("name = ?", value).Count(ctx)
				return err
			}},
			{"SelectQuery.Exists", func() error {
				_, err := db.NewSelect().Model((*FormatErrorModel)(nil)).Where("name = ?", value).Exists(ctx)
				return err
			}},
			{"InsertQuery.Exec", func() error {
				values := map[string]any{"name": value}
				_, err := db.NewInsert().Model(&values).TableExpr("format_error_models").Exec(ctx)
				return err
			}},
			{"UpdateQuery.Exec", func() error {
				_, err := db.NewUpdate().Model((*FormatErrorModel)(nil)).Set("name = ?", value).Where("id = 1").Exec(ctx)
				return err
			}},
			{"DeleteQuery.Exec", func() error {
				_, err := db.NewDelete().Model((*FormatErrorModel)(nil)).Where("name = ?", value).Exec(ctx)
				return err
			}},
			{"RawQuery.Scan", func() error {
				var name string
				return db.NewRaw("SELECT name FROM format_error_models WHERE name = ?", value).Scan(ctx, &name)
			}},
			{"DB.ExecContext", func() error {
				_, err := db.ExecContext(ctx, "DELETE FROM format_error_models WHERE name = ?", value)
				return err
			}},
			{"DB.QueryContext", func() error {
				rows, err := db.QueryContext(ctx, "SELECT name FROM format_error_models WHERE name = ?", value)
				if err != nil {
					return err
				}
				return rows.Close()
			}},
			{"Conn.ExecContext", func() error {
				_, err := conn.ExecContext(ctx, "DELETE FROM format_error_models WHERE name = ?", value)
				return err
			}},
			{"Tx.ExecContext", func() error {
				_, err := tx.ExecContext(ctx, "DELETE FROM format_error_models WHERE name = ?", value)
				return err
			}},
		}
		for _, e := range executors {
			t.Run(e.name, func(t *testing.T) {
				sent = sent[:0]
				requireNotSent(t, e.run(), want)
			})
		}
	}

	t.Run("driver.Valuer error", func(t *testing.T) {
		testExecutors(t, formatErrorValuer{}, errFormatErrorValue)
	})

	t.Run("NUL byte", func(t *testing.T) {
		if db.Dialect().Name() == dialect.MySQL {
			// MySQL stores NUL: mysqldialect escapes it as \0 (#1443).
			model := &FormatErrorModel{Name: "a\x00b\\\x000"}
			_, err := db.NewInsert().Model(model).Exec(ctx)
			require.NoError(t, err)

			got := new(FormatErrorModel)
			err = db.NewSelect().Model(got).Where("name = ?", model.Name).Scan(ctx)
			require.NoError(t, err)
			require.Equal(t, model.Name, got.Name)
			return
		}

		testExecutors(t, "a\x00b", schema.ErrStringNul)

		sent = sent[:0]
		_, err := db.NewInsert().Model(&FormatErrorModel{Name: "a\x00b"}).Exec(ctx)
		requireNotSent(t, err, schema.ErrStringNul)

		q := db.NewSelect().Model((*FormatErrorModel)(nil)).Where("name = ?", "a\x00b")
		require.Contains(t, q.String(), "?!(bun: string contains a NUL byte (0x00))")
	})

	t.Run("value that looks like a marker", func(t *testing.T) {
		model := &FormatErrorModel{Name: "x?!(y)"}
		_, err := db.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)

		got := new(FormatErrorModel)
		err = db.NewSelect().Model(got).Where("name = ?", model.Name).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, model.Name, got.Name)
	})
}
