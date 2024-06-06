package schema

import (
	"fmt"
	"reflect"
	"sync"
)

type Tables struct {
	dialect Dialect
	tables  sync.Map

	mu         sync.RWMutex
	seen       map[reflect.Type]*Table
	inProgress map[reflect.Type]*tableInProgress
}

func NewTables(dialect Dialect) *Tables {
	return &Tables{
		dialect:    dialect,
		seen:       make(map[reflect.Type]*Table),
		inProgress: make(map[reflect.Type]*tableInProgress),
	}
}

func (t *Tables) Register(models ...interface{}) {
	for _, model := range models {
		_ = t.Get(reflect.TypeOf(model).Elem())
	}
}

func (t *Tables) Get(typ reflect.Type) *Table {
	return t.table(typ, false)
}

func (t *Tables) InProgress(typ reflect.Type) *Table {
	return t.table(typ, true)
}

func (t *Tables) table(typ reflect.Type, allowInProgress bool) *Table {
	typ = indirectType(typ)
	if typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("got %s, wanted %s", typ.Kind(), reflect.Struct))
	}

	if v, ok := t.tables.Load(typ); ok {
		return v.(*Table)
	}

	t.mu.Lock()

	if v, ok := t.tables.Load(typ); ok {
		t.mu.Unlock()
		return v.(*Table)
	}

	var table *Table

	inProgress := t.inProgress[typ]
	if inProgress == nil {
		table = newTable(t.dialect, typ, t.seen, false)
		inProgress = newTableInProgress(table)
		t.inProgress[typ] = inProgress
	} else {
		table = inProgress.table
	}

	t.mu.Unlock()

	if allowInProgress {
		return table
	}

	if !inProgress.init() {
		return table
	}

	t.mu.Lock()
	delete(t.inProgress, typ)
	t.tables.Store(typ, table)
	t.mu.Unlock()

	t.dialect.OnTable(table)

	for _, field := range table.FieldMap {
		if field.UserSQLType == "" {
			field.UserSQLType = field.DiscoveredSQLType
		}
		if field.CreateTableSQLType == "" {
			field.CreateTableSQLType = field.UserSQLType
		}
	}

	return table
}

func (t *Tables) ByModel(name string) *Table {
	var found *Table
	t.tables.Range(func(key, value interface{}) bool {
		t := value.(*Table)
		if t.TypeName == name {
			found = t
			return false
		}
		return true
	})
	return found
}

func (t *Tables) ByName(name string) *Table {
	var found *Table
	t.tables.Range(func(key, value interface{}) bool {
		t := value.(*Table)
		if t.Name == name {
			found = t
			return false
		}
		return true
	})
	return found
}

type tableInProgress struct {
	table *Table

	initOnce sync.Once
}

func newTableInProgress(table *Table) *tableInProgress {
	return &tableInProgress{
		table: table,
	}
}

func (inp *tableInProgress) init() bool {
	var inited bool
	inp.initOnce.Do(func() {
		inp.table.init()
		inited = true
	})
	return inited
}
