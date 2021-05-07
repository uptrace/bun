package bun

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/uptrace/bun/schema"
)

type structTableModel struct {
	db    *DB
	table *schema.Table
	rel   *schema.Relation
	joins []join

	root  reflect.Value
	index []int

	strct         reflect.Value
	structInited  bool
	structInitErr error

	columns   []string
	scanIndex int
}

var _ tableModel = (*structTableModel)(nil)

func newStructTableModel(db *DB, table *schema.Table) *structTableModel {
	return &structTableModel{
		db:    db,
		table: table,
	}
}

func newStructTableModelValue(db *DB, v reflect.Value) *structTableModel {
	return &structTableModel{
		db:    db,
		table: db.Table(v.Type()),
		root:  v,
		strct: v,
	}
}

func (m *structTableModel) String() string {
	return m.table.String()
}

func (m *structTableModel) IsNil() bool {
	return !m.strct.IsValid()
}

func (m *structTableModel) Table() *schema.Table {
	return m.table
}

func (m *structTableModel) Relation() *schema.Relation {
	return m.rel
}

func (m *structTableModel) Root() reflect.Value {
	return m.root
}

func (m *structTableModel) Index() []int {
	return m.index
}

func (m *structTableModel) ParentIndex() []int {
	return m.index[:len(m.index)-len(m.rel.Field.Index)]
}

func (m *structTableModel) Kind() reflect.Kind {
	return reflect.Struct
}

func (m *structTableModel) Value() reflect.Value {
	return m.strct
}

func (m *structTableModel) Mount(host reflect.Value) {
	m.strct = host.FieldByIndex(m.rel.Field.Index)
	m.structInited = false
}

func (m *structTableModel) initStruct() error {
	if m.structInited {
		return m.structInitErr
	}
	m.structInited = true

	switch m.strct.Kind() {
	case reflect.Invalid:
		m.structInitErr = errModelNil
		return m.structInitErr
	case reflect.Interface:
		m.strct = m.strct.Elem()
	}

	if m.strct.Kind() == reflect.Ptr {
		if m.strct.IsNil() {
			m.strct.Set(reflect.New(m.strct.Type().Elem()))
			m.strct = m.strct.Elem()
		} else {
			m.strct = m.strct.Elem()
		}
	}

	m.mountJoins()

	return nil
}

func (m *structTableModel) mountJoins() {
	for i := range m.joins {
		j := &m.joins[i]
		switch j.Relation.Type {
		case schema.HasOneRelation, schema.BelongsToRelation:
			j.JoinModel.Mount(m.strct)
		}
	}
}

var _ schema.BeforeScanHook = (*structTableModel)(nil)

func (m *structTableModel) BeforeScan(ctx context.Context) error {
	if !m.table.HasBeforeScanHook() {
		return nil
	}
	return callBeforeScanHook(ctx, m.strct.Addr())
}

var _ schema.AfterScanHook = (*structTableModel)(nil)

func (m *structTableModel) AfterScan(ctx context.Context) error {
	if !m.table.HasAfterScanHook() || !m.structInited {
		return nil
	}

	var firstErr error

	if err := callAfterScanHook(ctx, m.strct.Addr()); err != nil && firstErr == nil {
		firstErr = err
	}

	for _, j := range m.joins {
		switch j.Relation.Type {
		case schema.HasOneRelation, schema.BelongsToRelation:
			if err := j.JoinModel.AfterScan(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func (m *structTableModel) AfterSelect(ctx context.Context) error {
	if m.table.HasAfterSelectHook() {
		return callAfterSelectHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) BeforeInsert(ctx context.Context) error {
	if m.table.HasBeforeInsertHook() {
		return callBeforeInsertHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) AfterInsert(ctx context.Context) error {
	if m.table.HasAfterInsertHook() {
		return callAfterInsertHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) BeforeUpdate(ctx context.Context) error {
	if m.table.HasBeforeUpdateHook() && !m.IsNil() {
		return callBeforeUpdateHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) AfterUpdate(ctx context.Context) error {
	if m.table.HasAfterUpdateHook() && !m.IsNil() {
		return callAfterUpdateHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) BeforeDelete(ctx context.Context) error {
	if m.table.HasBeforeDeleteHook() && !m.IsNil() {
		return callBeforeDeleteHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) AfterDelete(ctx context.Context) error {
	if m.table.HasAfterDeleteHook() && !m.IsNil() {
		return callAfterDeleteHook(ctx, m.strct.Addr())
	}
	return nil
}

func (m *structTableModel) GetJoin(name string) *join {
	for i := range m.joins {
		j := &m.joins[i]
		if j.Relation.Field.Name == name || j.Relation.Field.GoName == name {
			return j
		}
	}
	return nil
}

func (m *structTableModel) GetJoins() []join {
	return m.joins
}

func (m *structTableModel) AddJoin(j join) *join {
	m.joins = append(m.joins, j)
	return &m.joins[len(m.joins)-1]
}

func (m *structTableModel) Join(name string, apply func(*SelectQuery) *SelectQuery) *join {
	return m.join(m.Value(), name, apply)
}

func (m *structTableModel) join(
	bind reflect.Value, name string, apply func(*SelectQuery) *SelectQuery,
) *join {
	path := strings.Split(name, ".")
	index := make([]int, 0, len(path))

	currJoin := join{
		BaseModel: m,
		JoinModel: m,
	}
	var lastJoin *join

	for _, name := range path {
		relation, ok := currJoin.JoinModel.Table().Relations[name]
		if !ok {
			return nil
		}

		currJoin.Relation = relation
		index = append(index, relation.Field.Index...)

		if j := currJoin.JoinModel.GetJoin(name); j != nil {
			currJoin.BaseModel = j.BaseModel
			currJoin.JoinModel = j.JoinModel

			lastJoin = j
		} else {
			model, err := newTableModelIndex(m.db, m.table, bind, index, relation)
			if err != nil {
				return nil
			}

			currJoin.Parent = lastJoin
			currJoin.BaseModel = currJoin.JoinModel
			currJoin.JoinModel = model

			lastJoin = currJoin.BaseModel.AddJoin(currJoin)
		}
	}

	// No joins with such name.
	if lastJoin == nil {
		return nil
	}
	if apply != nil {
		lastJoin.ApplyQueryFunc = apply
	}

	return lastJoin
}

func (m *structTableModel) updateSoftDeleteField() error {
	fv := m.table.SoftDeleteField.Value(m.strct)
	return m.table.UpdateSoftDeleteField(fv)
}

func (m *structTableModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	if !rows.Next() {
		return 0, sql.ErrNoRows
	}

	if err := m.ScanRow(ctx, rows); err != nil {
		return 0, err
	}

	return 1, nil
}

func (m *structTableModel) ScanRow(ctx context.Context, rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	m.columns = columns
	dest := makeDest(m, len(columns))

	return m.scanRow(ctx, rows, dest)
}

func (m *structTableModel) scanRow(ctx context.Context, rows *sql.Rows, dest []interface{}) error {
	if err := m.BeforeScan(ctx); err != nil {
		return err
	}

	m.scanIndex = 0
	if err := rows.Scan(dest...); err != nil {
		return err
	}

	if err := m.AfterScan(ctx); err != nil {
		return err
	}

	return nil
}

func (m *structTableModel) Scan(src interface{}) error {
	column := m.columns[m.scanIndex]
	m.scanIndex++

	return m.ScanColumn(unquote(column), src)
}

func (m *structTableModel) ScanColumn(column string, src interface{}) error {
	if ok, err := m.scanColumn(column, src); ok {
		return err
	}
	if m.db.flags.Has(discardUnknownColumns) {
		return nil
	}
	return fmt.Errorf("bun: %s does not have column %q", m.table.TypeName, column)
}

func (m *structTableModel) scanColumn(column string, src interface{}) (bool, error) {
	if src != nil {
		if err := m.initStruct(); err != nil {
			return true, err
		}
	}

	if field, ok := m.table.FieldMap[column]; ok {
		return true, field.ScanValue(m.strct, src)
	}

	if joinName, column := splitColumn(column); joinName != "" {
		if join := m.GetJoin(joinName); join != nil {
			return true, join.JoinModel.ScanColumn(column, src)
		}
		if m.table.ModelName == joinName {
			return true, m.ScanColumn(column, src)
		}
	}

	return false, nil
}

// sqlite3 sometimes does not unquote columns.
func unquote(s string) string {
	if s == "" {
		return s
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func splitColumn(s string) (string, string) {
	if i := strings.Index(s, "__"); i >= 0 {
		return s[:i], s[i+2:]
	}
	return "", s
}
