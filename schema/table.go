package schema

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/inflection"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/tagparser"
)

const (
	beforeScanHookFlag internal.Flag = 1 << iota
	afterScanHookFlag
)

var tableNameInflector = inflection.Plural

// SetTableNameInflector overrides the default func that pluralizes
// model name to get table name, e.g. my_article becomes my_articles.
func SetTableNameInflector(fn func(string) string) {
	tableNameInflector = fn
}

// Table represents a SQL table created from Go struct.
type Table struct {
	dialect Dialect

	Type      reflect.Type
	ZeroValue reflect.Value // reflect.Struct
	ZeroIface interface{}   // struct pointer

	TypeName  string
	ModelName string

	Name              string
	SQLName           Safe
	SQLNameForSelects Safe
	Alias             string
	SQLAlias          Safe

	Fields     []*Field // PKs + DataFields
	PKs        []*Field
	DataFields []*Field

	fieldsMapMu sync.RWMutex
	FieldMap    map[string]*Field

	Relations map[string]*Relation
	Unique    map[string][]*Field

	SoftDeleteField       *Field
	UpdateSoftDeleteField func(fv reflect.Value) error

	allFields     []*Field // read only
	skippedFields []*Field

	flags internal.Flag
}

func newTable(dialect Dialect, typ reflect.Type) *Table {
	t := new(Table)
	t.dialect = dialect
	t.Type = typ
	t.ZeroValue = reflect.New(t.Type).Elem()
	t.ZeroIface = reflect.New(t.Type).Interface()
	t.TypeName = internal.ToExported(t.Type.Name())
	t.ModelName = internal.Underscore(t.Type.Name())
	tableName := tableNameInflector(t.ModelName)
	t.setName(tableName)
	t.Alias = t.ModelName
	t.SQLAlias = t.quoteIdent(t.ModelName)

	hooks := []struct {
		typ  reflect.Type
		flag internal.Flag
	}{
		{beforeScanHookType, beforeScanHookFlag},
		{afterScanHookType, afterScanHookFlag},
	}

	typ = reflect.PtrTo(t.Type)
	for _, hook := range hooks {
		if typ.Implements(hook.typ) {
			t.flags = t.flags.Set(hook.flag)
		}
	}

	return t
}

func (t *Table) init1() {
	t.initFields()
}

func (t *Table) init2() {
	t.initInlines()
	t.initRelations()
	t.skippedFields = nil
}

func (t *Table) setName(name string) {
	t.Name = name
	t.SQLName = t.quoteIdent(name)
	t.SQLNameForSelects = t.quoteIdent(name)
	if t.SQLAlias == "" {
		t.Alias = name
		t.SQLAlias = t.quoteIdent(name)
	}
}

func (t *Table) String() string {
	return "model=" + t.TypeName
}

func (t *Table) CheckPKs() error {
	if len(t.PKs) == 0 {
		return fmt.Errorf("bun: %s does not have primary keys", t)
	}
	return nil
}

func (t *Table) addField(field *Field) {
	t.Fields = append(t.Fields, field)
	if field.IsPK {
		t.PKs = append(t.PKs, field)
	} else {
		t.DataFields = append(t.DataFields, field)
	}
	t.FieldMap[field.Name] = field
}

func (t *Table) removeField(field *Field) {
	t.Fields = removeField(t.Fields, field)
	if field.IsPK {
		t.PKs = removeField(t.PKs, field)
	} else {
		t.DataFields = removeField(t.DataFields, field)
	}
	delete(t.FieldMap, field.Name)
}

func (t *Table) fieldWithLock(name string) *Field {
	t.fieldsMapMu.RLock()
	field := t.FieldMap[name]
	t.fieldsMapMu.RUnlock()
	return field
}

func (t *Table) HasField(name string) bool {
	_, ok := t.FieldMap[name]
	return ok
}

func (t *Table) Field(name string) (*Field, error) {
	field, ok := t.FieldMap[name]
	if !ok {
		return nil, fmt.Errorf("bun: %s does not have column=%s", t, name)
	}
	return field, nil
}

func (t *Table) fieldByGoName(name string) *Field {
	for _, f := range t.allFields {
		if f.GoName == name {
			return f
		}
	}
	return nil
}

func (t *Table) initFields() {
	t.Fields = make([]*Field, 0, t.Type.NumField())
	t.FieldMap = make(map[string]*Field, t.Type.NumField())
	t.addFields(t.Type, nil)

	if len(t.PKs) > 0 {
		return
	}
	for _, name := range []string{"id", "uuid", "pk_" + t.ModelName} {
		if field, ok := t.FieldMap[name]; ok {
			field.markAsPK()
			t.PKs = []*Field{field}
			t.DataFields = removeField(t.DataFields, field)
			break
		}
	}
	if len(t.PKs) == 1 {
		t.PKs[0].AutoIncrement = true
	}
}

func (t *Table) addFields(typ reflect.Type, baseIndex []int) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		// Make a copy so slice is not shared between fields.
		index := make([]int, len(baseIndex))
		copy(index, baseIndex)

		if f.Anonymous {
			if f.Tag.Get("bun") == "-" {
				continue
			}
			if f.Name == "BaseModel" {
				if len(index) == 0 {
					t.processBaseModelField(f)
				}
				continue
			}

			fieldType := indirectType(f.Type)
			if fieldType.Kind() != reflect.Struct {
				continue
			}
			t.addFields(fieldType, append(index, f.Index...))

			tag := tagparser.Parse(f.Tag.Get("bun"))
			if _, inherit := tag.Options["inherit"]; inherit {
				embeddedTable := t.dialect.Tables().Ref(fieldType)
				t.TypeName = embeddedTable.TypeName
				t.SQLName = embeddedTable.SQLName
				t.SQLNameForSelects = embeddedTable.SQLNameForSelects
				t.Alias = embeddedTable.Alias
				t.SQLAlias = embeddedTable.SQLAlias
				t.ModelName = embeddedTable.ModelName
			}

			continue
		}

		field := t.newField(f, index)
		if field != nil {
			t.addField(field)
		}
	}
}

func (t *Table) processBaseModelField(f reflect.StructField) {
	tag := tagparser.Parse(f.Tag.Get("bun"))

	if isKnownTableOption(tag.Name) {
		internal.Warn.Printf(
			"%s.%s tag name %q is also an option name; is it a mistake?",
			t.TypeName, f.Name, tag.Name,
		)
	}

	for name := range tag.Options {
		if !isKnownTableOption(name) {
			internal.Warn.Printf("%s.%s has unknown tag option: %q", t.TypeName, f.Name, name)
		}
	}

	if tag.Name != "" {
		t.setName(tag.Name)
	}

	if s, ok := tag.Options["select"]; ok {
		t.SQLNameForSelects = t.quoteTableName(s)
	}

	if s, ok := tag.Options["alias"]; ok {
		t.Alias = s
		t.SQLAlias = t.quoteIdent(s)
	}
}

//nolint
func (t *Table) newField(f reflect.StructField, index []int) *Field {
	tag := tagparser.Parse(f.Tag.Get("bun"))

	if f.PkgPath != "" {
		return nil
	}

	sqlName := internal.Underscore(f.Name)

	if tag.Name != sqlName && isKnownFieldOption(tag.Name) {
		internal.Warn.Printf(
			"%s.%s tag name %q is also an option name; is it a mistake?",
			t.TypeName, f.Name, tag.Name,
		)
	}

	for name := range tag.Options {
		if !isKnownFieldOption(name) {
			internal.Warn.Printf("%s.%s has unknown tag option: %q", t.TypeName, f.Name, name)
		}
	}

	skip := tag.Name == "-"
	if !skip && tag.Name != "" {
		sqlName = tag.Name
	}

	index = append(index, f.Index...)
	if field := t.fieldWithLock(sqlName); field != nil {
		if indexEqual(field.Index, index) {
			return field
		}
		t.removeField(field)
	}

	field := &Field{
		StructField: f,

		Tag:   tag,
		Type:  indirectType(f.Type),
		Index: index,

		Name:    sqlName,
		GoName:  f.Name,
		SQLName: t.quoteIdent(sqlName),
	}

	field.NotNull = tag.HasOption("notnull")
	field.NullZero = tag.HasOption("nullzero")
	field.AutoIncrement = tag.HasOption("autoincrement")
	if tag.HasOption("pk") {
		field.markAsPK()
	}

	if v, ok := tag.Options["unique"]; ok {
		// Split the value by comma, this will allow multiple names to be specified.
		// We can use this to create multiple named unique constraints where a single column
		// might be included in multiple constraints.
		for _, uniqueName := range strings.Split(v, ",") {
			if t.Unique == nil {
				t.Unique = make(map[string][]*Field)
			}
			t.Unique[uniqueName] = append(t.Unique[uniqueName], field)
		}
	}
	if s, ok := tag.Options["default"]; ok {
		field.SQLDefault = s
	}
	if s, ok := field.Tag.Options["type"]; ok {
		field.UserSQLType = s
	}
	field.DiscoveredSQLType = DiscoverSQLType(field.Type)
	field.Append = FieldAppender(t.dialect, field)
	field.Scan = FieldScanner(t.dialect, field)
	field.IsZero = FieldZeroChecker(field)

	if v, ok := tag.Options["alias"]; ok {
		t.FieldMap[v] = field
	}

	t.allFields = append(t.allFields, field)
	if skip {
		t.skippedFields = append(t.skippedFields, field)
		t.FieldMap[field.Name] = field
		return nil
	}

	if _, ok := tag.Options["soft_delete"]; ok {
		field.NullZero = true
		t.SoftDeleteField = field
		t.UpdateSoftDeleteField = softDeleteFieldUpdater(field)
	}

	return field
}

func (t *Table) initInlines() {
	for _, f := range t.skippedFields {
		if f.Type.Kind() == reflect.Struct {
			t.inlineFields(f, nil)
		}
	}
}

//---------------------------------------------------------------------------------------

func (t *Table) initRelations() {
	for i := 0; i < len(t.Fields); {
		f := t.Fields[i]
		if t.tryRelation(f) {
			t.Fields = removeField(t.Fields, f)
			t.DataFields = removeField(t.DataFields, f)
		} else {
			i++
		}

		if f.Type.Kind() == reflect.Struct {
			t.inlineFields(f, nil)
		}
	}
}

func (t *Table) tryRelation(field *Field) bool {
	if rel, ok := field.Tag.Options["rel"]; ok {
		t.initRelation(field, rel)
		return true
	}
	if field.Tag.HasOption("m2m") {
		t.addRelation(t.m2mRelation(field))
		return true
	}

	if field.Tag.HasOption("join") {
		internal.Warn.Printf(
			`%s.%s option "join" requires a relation type`,
			t.TypeName, field.GoName,
		)
	}

	return false
}

func (t *Table) initRelation(field *Field, rel string) {
	switch rel {
	case "belongs-to":
		t.addRelation(t.belongsToRelation(field))
	case "has-one":
		t.addRelation(t.hasOneRelation(field))
	case "has-many":
		t.addRelation(t.hasManyRelation(field))
	default:
		panic(fmt.Errorf("bun: unknown relation=%s on field=%s", rel, field.GoName))
	}
}

func (t *Table) addRelation(rel *Relation) {
	if t.Relations == nil {
		t.Relations = make(map[string]*Relation)
	}
	_, ok := t.Relations[rel.Field.GoName]
	if ok {
		panic(fmt.Errorf("%s already has %s", t, rel))
	}
	t.Relations[rel.Field.GoName] = rel
}

func (t *Table) belongsToRelation(field *Field) *Relation {
	joinTable := t.dialect.Tables().Ref(field.Type)
	if err := joinTable.CheckPKs(); err != nil {
		panic(err)
	}

	rel := &Relation{
		Type:      HasOneRelation,
		Field:     field,
		JoinTable: joinTable,
	}

	if join, ok := field.Tag.Options["join"]; ok {
		baseColumns, joinColumns := parseRelationJoin(join)
		for i, baseColumn := range baseColumns {
			joinColumn := joinColumns[i]

			if f := t.fieldWithLock(baseColumn); f != nil {
				rel.BaseFields = append(rel.BaseFields, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s belongs-to %s: %s must have column %s",
					t.TypeName, field.GoName, t.TypeName, baseColumn,
				))
			}

			if f := joinTable.fieldWithLock(joinColumn); f != nil {
				rel.JoinFields = append(rel.JoinFields, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s belongs-to %s: %s must have column %s",
					t.TypeName, field.GoName, t.TypeName, baseColumn,
				))
			}
		}
		return rel
	}

	rel.JoinFields = joinTable.PKs
	fkPrefix := internal.Underscore(field.GoName) + "_"
	for _, joinPK := range joinTable.PKs {
		fkName := fkPrefix + joinPK.Name
		if fk := t.fieldWithLock(fkName); fk != nil {
			rel.BaseFields = append(rel.BaseFields, fk)
			continue
		}

		if fk := t.fieldWithLock(joinPK.Name); fk != nil {
			rel.BaseFields = append(rel.BaseFields, fk)
			continue
		}

		panic(fmt.Errorf(
			"bun: %s belongs-to %s: %s must have column %s "+
				"(to override, use join:base_column=join_column tag on %s field)",
			t.TypeName, field.GoName, t.TypeName, fkName, field.GoName,
		))
	}
	return rel
}

func (t *Table) hasOneRelation(field *Field) *Relation {
	if err := t.CheckPKs(); err != nil {
		panic(err)
	}

	joinTable := t.dialect.Tables().Ref(field.Type)
	rel := &Relation{
		Type:      BelongsToRelation,
		Field:     field,
		JoinTable: joinTable,
	}

	if join, ok := field.Tag.Options["join"]; ok {
		baseColumns, joinColumns := parseRelationJoin(join)
		for i, baseColumn := range baseColumns {
			if f := t.fieldWithLock(baseColumn); f != nil {
				rel.BaseFields = append(rel.BaseFields, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-one %s: %s must have column %s",
					field.GoName, t.TypeName, joinTable.TypeName, baseColumn,
				))
			}

			joinColumn := joinColumns[i]
			if f := joinTable.fieldWithLock(joinColumn); f != nil {
				rel.JoinFields = append(rel.JoinFields, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-one %s: %s must have column %s",
					field.GoName, t.TypeName, joinTable.TypeName, baseColumn,
				))
			}
		}
		return rel
	}

	rel.BaseFields = t.PKs
	fkPrefix := internal.Underscore(t.ModelName) + "_"
	for _, pk := range t.PKs {
		fkName := fkPrefix + pk.Name
		if f := joinTable.fieldWithLock(fkName); f != nil {
			rel.JoinFields = append(rel.JoinFields, f)
			continue
		}

		if f := joinTable.fieldWithLock(pk.Name); f != nil {
			rel.JoinFields = append(rel.JoinFields, f)
			continue
		}

		panic(fmt.Errorf(
			"bun: %s has-one %s: %s must have column %s "+
				"(to override, use join:base_column=join_column tag on %s field)",
			field.GoName, t.TypeName, joinTable.TypeName, fkName, field.GoName,
		))
	}
	return rel
}

func (t *Table) hasManyRelation(field *Field) *Relation {
	if err := t.CheckPKs(); err != nil {
		panic(err)
	}
	if field.Type.Kind() != reflect.Slice {
		panic(fmt.Errorf(
			"bun: %s.%s has-many relation requires slice, got %q",
			t.TypeName, field.GoName, field.Type.Kind(),
		))
	}

	joinTable := t.dialect.Tables().Ref(indirectType(field.Type.Elem()))
	polymorphicValue, isPolymorphic := field.Tag.Options["polymorphic"]
	rel := &Relation{
		Type:      HasManyRelation,
		Field:     field,
		JoinTable: joinTable,
	}
	var polymorphicColumn string

	if join, ok := field.Tag.Options["join"]; ok {
		baseColumns, joinColumns := parseRelationJoin(join)
		for i, baseColumn := range baseColumns {
			joinColumn := joinColumns[i]

			if isPolymorphic && baseColumn == "type" {
				polymorphicColumn = joinColumn
				continue
			}

			if f := t.fieldWithLock(baseColumn); f != nil {
				rel.BaseFields = append(rel.BaseFields, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-one %s: %s must have column %s",
					t.TypeName, field.GoName, t.TypeName, baseColumn,
				))
			}

			if f := joinTable.fieldWithLock(joinColumn); f != nil {
				rel.JoinFields = append(rel.JoinFields, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-one %s: %s must have column %s",
					t.TypeName, field.GoName, t.TypeName, baseColumn,
				))
			}
		}
	} else {
		rel.BaseFields = t.PKs
		fkPrefix := internal.Underscore(t.ModelName) + "_"
		if isPolymorphic {
			polymorphicColumn = fkPrefix + "type"
		}

		for _, pk := range t.PKs {
			joinColumn := fkPrefix + pk.Name
			if fk := joinTable.fieldWithLock(joinColumn); fk != nil {
				rel.JoinFields = append(rel.JoinFields, fk)
				continue
			}

			if fk := joinTable.fieldWithLock(pk.Name); fk != nil {
				rel.JoinFields = append(rel.JoinFields, fk)
				continue
			}

			panic(fmt.Errorf(
				"bun: %s has-many %s: %s must have column %s "+
					"(to override, use join:base_column=join_column tag on the field %s)",
				t.TypeName, field.GoName, joinTable.TypeName, joinColumn, field.GoName,
			))
		}
	}

	if isPolymorphic {
		rel.PolymorphicField = joinTable.fieldWithLock(polymorphicColumn)
		if rel.PolymorphicField == nil {
			panic(fmt.Errorf(
				"bun: %s has-many %s: %s must have polymorphic column %s",
				t.TypeName, field.GoName, joinTable.TypeName, polymorphicColumn,
			))
		}

		if polymorphicValue == "" {
			polymorphicValue = t.ModelName
		}
		rel.PolymorphicValue = polymorphicValue
	}

	return rel
}

func (t *Table) m2mRelation(field *Field) *Relation {
	if field.Type.Kind() != reflect.Slice {
		panic(fmt.Errorf(
			"bun: %s.%s m2m relation requires slice, got %q",
			t.TypeName, field.GoName, field.Type.Kind(),
		))
	}
	joinTable := t.dialect.Tables().Ref(indirectType(field.Type.Elem()))

	if err := t.CheckPKs(); err != nil {
		panic(err)
	}
	if err := joinTable.CheckPKs(); err != nil {
		panic(err)
	}

	m2mTableName, ok := field.Tag.Options["m2m"]
	if !ok {
		panic(fmt.Errorf("bun: %s must have m2m tag option", field.GoName))
	}

	m2mTable := t.dialect.Tables().ByName(m2mTableName)
	if m2mTable == nil {
		panic(fmt.Errorf(
			"bun: can't find m2m %s table (use db.RegisterModel)",
			m2mTableName,
		))
	}

	rel := &Relation{
		Type:      ManyToManyRelation,
		Field:     field,
		JoinTable: joinTable,
		M2MTable:  m2mTable,
	}
	var leftColumn, rightColumn string

	if join, ok := field.Tag.Options["join"]; ok {
		left, right := parseRelationJoin(join)
		leftColumn = left[0]
		rightColumn = right[0]
	} else {
		leftColumn = t.TypeName
		rightColumn = joinTable.TypeName
	}

	leftField := m2mTable.fieldByGoName(leftColumn)
	if leftField == nil {
		panic(fmt.Errorf(
			"bun: %s many-to-many %s: %s must have field %s "+
				"(to override, use tag join:LeftField=RightField on field %s.%s",
			t.TypeName, field.GoName, m2mTable.TypeName, leftColumn, t.TypeName, field.GoName,
		))
	}

	rightField := m2mTable.fieldByGoName(rightColumn)
	if rightField == nil {
		panic(fmt.Errorf(
			"bun: %s many-to-many %s: %s must have field %s "+
				"(to override, use tag join:LeftField=RightField on field %s.%s",
			t.TypeName, field.GoName, m2mTable.TypeName, rightColumn, t.TypeName, field.GoName,
		))
	}

	leftRel := m2mTable.belongsToRelation(leftField)
	rel.BaseFields = leftRel.JoinFields
	rel.M2MBaseFields = leftRel.BaseFields

	rightRel := m2mTable.belongsToRelation(rightField)
	rel.JoinFields = rightRel.JoinFields
	rel.M2MJoinFields = rightRel.BaseFields

	return rel
}

func (t *Table) inlineFields(strct *Field, path map[reflect.Type]struct{}) {
	if path == nil {
		path = map[reflect.Type]struct{}{
			t.Type: {},
		}
	}

	if _, ok := path[strct.Type]; ok {
		return
	}
	path[strct.Type] = struct{}{}

	joinTable := t.dialect.Tables().Ref(strct.Type)
	for _, f := range joinTable.allFields {
		f = f.Clone()
		f.GoName = strct.GoName + "_" + f.GoName
		f.Name = strct.Name + "__" + f.Name
		f.SQLName = t.quoteIdent(f.Name)
		f.Index = appendNew(strct.Index, f.Index...)

		t.fieldsMapMu.Lock()
		if _, ok := t.FieldMap[f.Name]; !ok {
			t.FieldMap[f.Name] = f
		}
		t.fieldsMapMu.Unlock()

		if f.Type.Kind() != reflect.Struct {
			continue
		}

		if _, ok := path[f.Type]; !ok {
			t.inlineFields(f, path)
		}
	}
}

//------------------------------------------------------------------------------

func (t *Table) Dialect() Dialect { return t.dialect }

//------------------------------------------------------------------------------

func (t *Table) HasBeforeScanHook() bool { return t.flags.Has(beforeScanHookFlag) }
func (t *Table) HasAfterScanHook() bool  { return t.flags.Has(afterScanHookFlag) }

//------------------------------------------------------------------------------

func (t *Table) quoteTableName(s string) Safe {
	// Don't quote if table name contains placeholder (?) or parentheses.
	if strings.IndexByte(s, '?') >= 0 ||
		strings.IndexByte(s, '(') >= 0 ||
		strings.IndexByte(s, ')') >= 0 {
		return Safe(s)
	}
	return t.quoteIdent(s)
}

func (t *Table) quoteIdent(s string) Safe {
	return Safe(NewFormatter(t.dialect).AppendIdent(nil, s))
}

func appendNew(dst []int, src ...int) []int {
	cp := make([]int, len(dst)+len(src))
	copy(cp, dst)
	copy(cp[len(dst):], src)
	return cp
}

func isKnownTableOption(name string) bool {
	switch name {
	case "alias", "select":
		return true
	}
	return false
}

func isKnownFieldOption(name string) bool {
	switch name {
	case "alias",
		"type",
		"array",
		"hstore",
		"composite",
		"json_use_number",
		"msgpack",
		"notnull",
		"nullzero",
		"default",
		"unique",
		"soft_delete",

		"pk",
		"autoincrement",
		"rel",
		"join",
		"m2m",
		"polymorphic":
		return true
	}
	return false
}

func removeField(fields []*Field, field *Field) []*Field {
	for i, f := range fields {
		if f == field {
			return append(fields[:i], fields[i+1:]...)
		}
	}
	return fields
}

func parseRelationJoin(join string) ([]string, []string) {
	ss := strings.Split(join, ",")
	baseColumns := make([]string, len(ss))
	joinColumns := make([]string, len(ss))
	for i, s := range ss {
		ss := strings.Split(strings.TrimSpace(s), "=")
		if len(ss) != 2 {
			panic(fmt.Errorf("can't parse relation join: %q", join))
		}
		baseColumns[i] = ss[0]
		joinColumns[i] = ss[1]
	}
	return baseColumns, joinColumns
}

//------------------------------------------------------------------------------

func softDeleteFieldUpdater(field *Field) func(fv reflect.Value) error {
	switch field.Type {
	case timeType:
		return func(fv reflect.Value) error {
			ptr := fv.Addr().Interface().(*time.Time)
			*ptr = time.Now()
			return nil
		}
	case nullTimeType:
		return func(fv reflect.Value) error {
			ptr := fv.Addr().Interface().(*sql.NullTime)
			*ptr = sql.NullTime{Time: time.Now()}
			return nil
		}
	case nullIntType:
		return func(fv reflect.Value) error {
			ptr := fv.Addr().Interface().(*sql.NullInt64)
			*ptr = sql.NullInt64{Int64: time.Now().UnixNano()}
			return nil
		}
	}

	switch field.Type.Kind() {
	case reflect.Int64:
		return func(fv reflect.Value) error {
			ptr := fv.Addr().Interface().(*int64)
			*ptr = time.Now().UnixNano()
			return nil
		}
	case reflect.Ptr:
		break
	default:
		return softDeleteFieldUpdaterFallback(field)
	}

	typ := field.Type.Elem()

	switch typ { //nolint:gocritic
	case timeType:
		return func(fv reflect.Value) error {
			now := time.Now()
			fv.Set(reflect.ValueOf(&now))
			return nil
		}
	}

	switch typ.Kind() { //nolint:gocritic
	case reflect.Int64:
		return func(fv reflect.Value) error {
			utime := time.Now().UnixNano()
			fv.Set(reflect.ValueOf(&utime))
			return nil
		}
	}

	return softDeleteFieldUpdaterFallback(field)
}

func softDeleteFieldUpdaterFallback(field *Field) func(fv reflect.Value) error {
	return func(fv reflect.Value) error {
		return field.ScanWithCheck(fv, time.Now())
	}
}
