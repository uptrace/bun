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

func (t *Tables) Register(models ...any) {
	for _, model := range models {
		_ = t.Get(reflect.TypeOf(model).Elem())
	}
}

func (t *Tables) Get(typ reflect.Type) *Table {
	typ = indirectType(typ)
	if typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("got %s, wanted %s", typ.Kind(), reflect.Struct))
	}

	if v, ok := t.tables.Load(typ); ok {
		return v
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if v, ok := t.tables.Load(typ); ok {
		return v
	}

	table := t.InProgress(typ)
	table.initRelations()

	t.dialect.OnTable(table)
	for _, field := range table.FieldMap {
		if field.UserSQLType == "" {
			field.UserSQLType = field.DiscoveredSQLType
		}
		if field.CreateTableSQLType == "" {
			field.CreateTableSQLType = field.UserSQLType
		}
	}

	t.tables.Store(typ, table)
	return table
}

// InProgress returns the in-progress table for typ, initializing it
// if it has not been initialized yet. The Placeholder + init() split
// lets callers register StructMap entries with the table pointer
// before triggering recursive initialization, preventing missing
// entries in circular dependency chains.
func (t *Tables) InProgress(typ reflect.Type) *Table {
	table := t.Placeholder(typ)
	table.init(t.dialect, typ)
	return table
}

// Placeholder returns an existing in-progress table or creates a new
// uninitialized placeholder entry.
func (t *Tables) Placeholder(typ reflect.Type) *Table {
	table, ok := t.inProgress[typ]
	if !ok {
		table = new(Table)
		t.inProgress[typ] = table
	}
	return table
}

// ByModel gets the table by its Go name.
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

// ByName gets the table by its SQL name.
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

// All returns all registered tables.
func (t *Tables) All() []*Table {
	var found []*Table
	t.tables.Range(func(typ reflect.Type, table *Table) bool {
		found = append(found, table)
		return true
	})
	return found
}
