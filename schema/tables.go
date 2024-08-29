package schema

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/puzpuzpuz/xsync/v3"
)

type Tables struct {
	dialect Dialect

	mu     sync.Mutex
	tables *xsync.MapOf[reflect.Type, *Table]

	inProgress map[reflect.Type]*Table
}

func NewTables(dialect Dialect) *Tables {
	return &Tables{
		dialect:    dialect,
		tables:     xsync.NewMapOf[reflect.Type, *Table](),
		inProgress: make(map[reflect.Type]*Table),
	}
}

func (reg *Tables) Register(models ...interface{}) {
	for _, model := range models {
		_ = reg.Get(reflect.TypeOf(model).Elem())
	}
}

func (reg *Tables) Get(typ reflect.Type) *Table {
	typ = indirectType(typ)
	if typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("got %s, wanted %s", typ.Kind(), reflect.Struct))
	}

	if v, ok := reg.tables.Load(typ); ok {
		return v
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	if v, ok := reg.tables.Load(typ); ok {
		reg.mu.Unlock()
		return v
	}

	table := reg.InProgress(typ)
	table.initRelations()

	reg.dialect.OnTable(table)
	for _, field := range table.FieldMap {
		if field.UserSQLType == "" {
			field.UserSQLType = field.DiscoveredSQLType
		}
		if field.CreateTableSQLType == "" {
			field.CreateTableSQLType = field.UserSQLType
		}
	}

	reg.tables.Store(typ, table)
	return table
}

func (reg *Tables) InProgress(typ reflect.Type) *Table {
	if table, ok := reg.inProgress[typ]; ok {
		return table
	}

	table := new(Table)
	reg.inProgress[typ] = table
	table.init(reg.dialect, typ, false)

	return table
}

func (t *Tables) ByModel(name string) *Table {
	var found *Table
	t.tables.Range(func(typ reflect.Type, table *Table) bool {
		if table.TypeName == name {
			found = table
			return false
		}
		return true
	})
	return found
}

func (t *Tables) ByName(name string) *Table {
	var found *Table
	t.tables.Range(func(typ reflect.Type, table *Table) bool {
		if table.Name == name {
			found = table
			return false
		}
		return true
	})
	return found
}
