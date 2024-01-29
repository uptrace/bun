package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTable(t *testing.T) {
	dialect := newNopDialect()
	tables := NewTables(dialect)

	t.Run("simple", func(t *testing.T) {
		type Model struct {
			ID  int `bun:",pk"`
			Foo string
			Bar string
		}

		table := tables.Get(reflect.TypeOf((*Model)(nil)))

		require.Len(t, table.allFields, 3)
		require.Len(t, table.Fields, 3)
		require.Len(t, table.PKs, 1)
		require.Len(t, table.DataFields, 2)
	})

	type Model struct {
		Foo string
		Bar string
	}

	t.Run("model1", func(t *testing.T) {
		type Model1 struct {
			Model
			Foo string
		}

		table := tables.Get(reflect.TypeOf((*Model1)(nil)))

		foo, ok := table.FieldMap["foo"]
		require.True(t, ok)
		require.Equal(t, []int{1}, foo.Index)

		bar, ok := table.FieldMap["bar"]
		require.True(t, ok)
		require.Equal(t, []int{0, 1}, bar.Index)
	})

	t.Run("model2", func(t *testing.T) {
		type Model2 struct {
			Foo string
			Model
		}

		table := tables.Get(reflect.TypeOf((*Model2)(nil)))

		foo, ok := table.FieldMap["foo"]
		require.True(t, ok)
		require.Equal(t, []int{0}, foo.Index)

		bar, ok := table.FieldMap["bar"]
		require.True(t, ok)
		require.Equal(t, []int{1, 1}, bar.Index)
	})

	t.Run("table name", func(t *testing.T) {
		type Model struct {
			BaseModel `bun:"custom_name,alias:custom_alias"`
		}

		table := tables.Get(reflect.TypeOf((*Model)(nil)))
		require.Equal(t, "custom_name", table.Name)
		require.Equal(t, "custom_alias", table.Alias)
	})

	t.Run("extend", func(t *testing.T) {
		type Model1 struct {
			BaseModel `bun:"custom_name,alias:custom_alias"`
		}
		type Model2 struct {
			Model1 `bun:",extend"`
		}

		table := tables.Get(reflect.TypeOf((*Model2)(nil)))
		require.Equal(t, "custom_name", table.Name)
		require.Equal(t, "custom_alias", table.Alias)
	})

	t.Run("embed", func(t *testing.T) {
		type Perms struct {
			View   bool
			Create bool
		}

		type Role struct {
			Foo Perms `bun:"embed:foo_"`
			Bar Perms `bun:"embed:bar_"`
		}

		table := tables.Get(reflect.TypeOf((*Role)(nil)))
		require.Nil(t, table.StructMap["foo"])
		require.Nil(t, table.StructMap["bar"])

		fooView, ok := table.FieldMap["foo_view"]
		require.True(t, ok)
		require.Equal(t, []int{0, 0}, fooView.Index)

		barView, ok := table.FieldMap["bar_view"]
		require.True(t, ok)
		require.Equal(t, []int{1, 0}, barView.Index)
	})

	t.Run("scanonly", func(t *testing.T) {
		type Model1 struct {
			Foo string
			Bar string
		}

		type Model2 struct {
			XXX Model1 `bun:",scanonly"`
		}

		table := tables.Get(reflect.TypeOf((*Model2)(nil)))
		require.NotNil(t, table.StructMap["xxx"])

		foo := table.LookupField("xxx__foo")
		require.NotNil(t, foo)
		require.Equal(t, []int{0, 0}, foo.Index)

		bar := table.LookupField("xxx__bar")
		require.NotNil(t, foo)
		require.Equal(t, []int{0, 1}, bar.Index)
	})

	t.Run("recursive", func(t *testing.T) {
		type Model struct {
			*Model

			Foo string
			Bar string
		}

		table := tables.Get(reflect.TypeOf((*Model)(nil)))

		foo, ok := table.FieldMap["foo"]
		require.True(t, ok)
		require.Equal(t, []int{1}, foo.Index)

		bar, ok := table.FieldMap["bar"]
		require.True(t, ok)
		require.Equal(t, []int{2}, bar.Index)
	})

	t.Run("recursive relation", func(t *testing.T) {
		type Item struct {
			ID     int64 `bun:",pk"`
			ItemID int64
			Item   *Item `bun:"rel:belongs-to,join:item_id=id"`
		}

		table := tables.Get(reflect.TypeOf((*Item)(nil)))

		rel, ok := table.Relations["Item"]
		require.True(t, ok)
		require.Equal(t, BelongsToRelation, rel.Type)

		{
			require.NotNil(t, table.StructMap["item"])

			id := table.LookupField("item__id")
			require.NotNil(t, id)
			require.Equal(t, []int{2, 0}, id.Index)
		}
	})
}
