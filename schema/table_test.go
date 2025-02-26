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

		table := tables.Get(reflect.TypeFor[*Model]())

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

		table := tables.Get(reflect.TypeFor[*Model1]())

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

		table := tables.Get(reflect.TypeFor[*Model2]())

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

		table := tables.Get(reflect.TypeFor[*Model]())
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

		table := tables.Get(reflect.TypeFor[*Model2]())
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

		table := tables.Get(reflect.TypeFor[*Role]())
		require.Nil(t, table.StructMap["foo"])
		require.Nil(t, table.StructMap["bar"])

		fooView, ok := table.FieldMap["foo_view"]
		require.True(t, ok)
		require.Equal(t, []int{0, 0}, fooView.Index)

		barView, ok := table.FieldMap["bar_view"]
		require.True(t, ok)
		require.Equal(t, []int{1, 0}, barView.Index)
	})

	t.Run("unambiguous embed field", func(t *testing.T) {
		type Perms struct {
			View   bool
			Create bool
		}
		type Role struct {
			Perms        // should be ignore
			Foo    Perms `bun:"embed:foo_"`
			View   bool
			Create bool
		}

		table := tables.Get(reflect.TypeFor[*Role]())

		view, ok := table.FieldMap["view"]
		require.True(t, ok)
		require.Equal(t, []int{2}, view.Index)

		fooView, ok := table.FieldMap["foo_view"]
		require.True(t, ok)
		require.Equal(t, []int{1, 0}, fooView.Index)
	})

	t.Run("embedWithUnique", func(t *testing.T) {
		type Perms struct {
			View          bool
			Create        bool
			UniqueID      int `bun:",unique"`
			UniqueGroupID int `bun:",unique:groupa"`
		}

		type Role struct {
			Foo Perms `bun:"embed:foo_"`
			Perms
		}

		table := tables.Get(reflect.TypeFor[*Role]())
		require.Nil(t, table.StructMap["foo"])
		require.Nil(t, table.StructMap["bar"])

		fooView, ok := table.FieldMap["foo_view"]
		require.True(t, ok)
		require.Equal(t, []int{0, 0}, fooView.Index)

		barView, ok := table.FieldMap["view"]
		require.True(t, ok)
		require.Equal(t, []int{1, 0}, barView.Index)

		require.Equal(t, 3, len(table.Unique))
		require.Equal(t, 2, len(table.Unique[""]))
		require.Equal(t, "foo_unique_id", table.Unique[""][0].Name)
		require.Equal(t, "unique_id", table.Unique[""][1].Name)
		require.Equal(t, 1, len(table.Unique["groupa"]))
		require.Equal(t, "unique_group_id", table.Unique["groupa"][0].Name)
		require.Equal(t, 1, len(table.Unique["foo_groupa"]))
		require.Equal(t, "foo_unique_group_id", table.Unique["foo_groupa"][0].Name)
	})

	t.Run("embedWithRelation", func(t *testing.T) {
		type Profile struct {
			ID     string `bun:",pk"`
			UserID string
		}
		type User struct {
			ID      string   `bun:",pk"`
			Profile *Profile `bun:"rel:has-one,join:id=user_id"`
		}
		type Embeded struct {
			User
			Extra string `bun:"-"`
		}

		table := tables.Get(reflect.TypeFor[*Embeded]())
		require.Contains(t, table.StructMap, "profile")
	})

	t.Run("embed scanonly", func(t *testing.T) {
		type Model1 struct {
			Foo string
			Bar string `bun:",scanonly"`
		}

		type Model2 struct {
			Model1
		}

		table := tables.Get(reflect.TypeFor[*Model2]())
		require.Len(t, table.FieldMap, 2)

		foo, ok := table.FieldMap["foo"]
		require.True(t, ok)
		require.Equal(t, []int{0, 0}, foo.Index)

		bar, ok := table.FieldMap["bar"]
		require.True(t, ok)
		require.Equal(t, []int{0, 1}, bar.Index)
	})

	t.Run("embed scanonly prefix", func(t *testing.T) {
		type Model1 struct {
			Foo string `bun:",scanonly"`
			Bar string `bun:",scanonly"`
		}

		type Model2 struct {
			Baz Model1 `bun:"embed:baz_"`
		}

		table := tables.Get(reflect.TypeFor[*Model2]())
		require.Len(t, table.FieldMap, 2)

		foo, ok := table.FieldMap["baz_foo"]
		require.True(t, ok)
		require.Equal(t, []int{0, 0}, foo.Index)

		bar, ok := table.FieldMap["baz_bar"]
		require.True(t, ok)
		require.Equal(t, []int{0, 1}, bar.Index)
	})

	t.Run("scanonly", func(t *testing.T) {
		type Model1 struct {
			Foo string
			Bar string
		}

		type Model2 struct {
			XXX Model1 `bun:",scanonly"`
			Baz string `bun:",scanonly"`
		}

		table := tables.Get(reflect.TypeFor[*Model2]())

		require.Len(t, table.StructMap, 1)
		require.NotNil(t, table.StructMap["xxx"])

		require.Len(t, table.FieldMap, 2)
		baz := table.FieldMap["baz"]
		require.NotNil(t, baz)
		require.Equal(t, []int{1}, baz.Index)

		foo := table.LookupField("xxx__foo")
		require.NotNil(t, foo)
		require.Equal(t, []int{0, 0}, foo.Index)

		bar := table.LookupField("xxx__bar")
		require.NotNil(t, bar)
		require.Equal(t, []int{0, 1}, bar.Index)
	})

	t.Run("recursive", func(t *testing.T) {
		type Model struct {
			*Model

			Foo string
			Bar string
		}

		table := tables.Get(reflect.TypeFor[*Model]())

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

		table := tables.Get(reflect.TypeFor[*Item]())

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

	t.Run("alternative name", func(t *testing.T) {
		type ModelTest struct {
			Model
			Foo string `bun:"alt:alt_name"`
		}

		table := tables.Get(reflect.TypeFor[*ModelTest]())

		foo, ok := table.FieldMap["foo"]
		require.True(t, ok)
		require.Equal(t, []int{1}, foo.Index)

		foo2, ok := table.FieldMap["alt_name"]
		require.True(t, ok)
		require.Equal(t, []int{1}, foo2.Index)

		require.Equal(t, table.FieldMap["foo"].SQLName, table.FieldMap["alt_name"].SQLName)
	})
}
