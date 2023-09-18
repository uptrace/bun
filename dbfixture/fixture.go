package dbfixture

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

var (
	funcNameRE = regexp.MustCompile(`^\{\{ (\w+) \}\}$`)
	tplRE      = regexp.MustCompile(`\{\{ .+ \}\}`)
)

type FixtureOption func(l *Fixture)

func WithRecreateTables() FixtureOption {
	return func(l *Fixture) {
		if l.truncateTables {
			panic("don't use WithDropTables together with WithTruncateTables")
		}
		l.recreateTables = true
		l.seenTables = make(map[string]struct{})
	}
}

func WithTruncateTables() FixtureOption {
	return func(l *Fixture) {
		if l.recreateTables {
			panic("don't use WithTruncateTables together with WithRecreateTables")
		}
		l.truncateTables = true
		l.seenTables = make(map[string]struct{})
	}
}

func WithTemplateFuncs(funcMap template.FuncMap) FixtureOption {
	return func(l *Fixture) {
		for k, v := range funcMap {
			l.funcMap[k] = v
		}
	}
}

type BeforeInsertData struct {
	Query *bun.InsertQuery
	Model interface{}
}

type BeforeInsertFunc func(ctx context.Context, data *BeforeInsertData) error

func WithBeforeInsert(fn BeforeInsertFunc) FixtureOption {
	return func(f *Fixture) {
		f.beforeInsert = append(f.beforeInsert, fn)
	}
}

type Fixture struct {
	db bun.IDB

	recreateTables bool
	truncateTables bool
	beforeInsert   []BeforeInsertFunc

	seenTables map[string]struct{}

	funcMap   template.FuncMap
	modelRows map[string]map[string]interface{}
}

func New(db bun.IDB, opts ...FixtureOption) *Fixture {
	f := &Fixture{
		db: db,

		funcMap:   defaultFuncs(),
		modelRows: make(map[string]map[string]interface{}),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *Fixture) Row(id string) (interface{}, error) {
	ss := strings.Split(id, ".")
	if len(ss) != 2 {
		return nil, fmt.Errorf("fixture: invalid row id: %q", id)
	}
	model, rowID := ss[0], ss[1]

	rows, ok := f.modelRows[model]
	if !ok {
		return nil, fmt.Errorf("fixture: unknown model=%q", model)
	}

	row, ok := rows[rowID]
	if !ok {
		return nil, fmt.Errorf("fixture: can't find row=%q for model=%q", rowID, model)
	}

	return row, nil
}

func (f *Fixture) MustRow(id string) interface{} {
	row, err := f.Row(id)
	if err != nil {
		panic(err)
	}
	return row
}

func (f *Fixture) Load(ctx context.Context, fsys fs.FS, names ...string) error {
	for _, name := range names {
		if err := f.load(ctx, fsys, name); err != nil {
			return err
		}
	}
	return nil
}

func (f *Fixture) load(ctx context.Context, fsys fs.FS, name string) error {
	fh, err := fsys.Open(name)
	if err != nil {
		return err
	}

	var fixtures []fixtureData

	dec := yaml.NewDecoder(fh)
	if err := dec.Decode(&fixtures); err != nil {
		return err
	}

	for i := range fixtures {
		if err := f.addFixture(ctx, &fixtures[i]); err != nil {
			return err
		}
	}

	return nil
}

func (f *Fixture) addFixture(ctx context.Context, data *fixtureData) error {
	table := f.db.Dialect().Tables().ByModel(data.Model)
	if table == nil {
		return fmt.Errorf("fixture: can't find model=%q (use db.RegisterModel)", data.Model)
	}

	if f.recreateTables {
		if err := f.dropTable(ctx, table); err != nil {
			return err
		}
	} else if f.truncateTables {
		if err := f.truncateTable(ctx, table); err != nil {
			return err
		}
	}

	for _, row := range data.Rows {
		if err := f.addRow(ctx, table, row); err != nil {
			return err
		}
	}

	return nil
}

func (f *Fixture) addRow(ctx context.Context, table *schema.Table, row row) error {
	var rowID string
	strct := reflect.New(table.Type).Elem()

	for key, value := range row {
		if key == "_id" {
			if err := value.Decode(&rowID); err != nil {
				return err
			}
			continue
		}

		field, err := table.Field(key)
		if err != nil {
			return err
		}

		if err := f.decodeField(strct, field, &value); err != nil {
			return fmt.Errorf("dbfixture: decoding %s failed: %w", key, err)
		}
	}

	model := strct.Addr().Interface()
	q := f.db.NewInsert().Model(model)

	data := &BeforeInsertData{
		Query: q,
		Model: model,
	}
	for _, fn := range f.beforeInsert {
		if err := fn(ctx, data); err != nil {
			return err
		}
	}

	if _, err := q.Exec(ctx); err != nil {
		return err
	}

	if rowID == "" && len(table.PKs) == 1 {
		pk := table.PKs[0]
		fv := pk.Value(strct)
		rowID = "pk" + asString(fv)
	}

	if rowID != "" {
		rows, ok := f.modelRows[table.TypeName]
		if !ok {
			rows = make(map[string]interface{})
			f.modelRows[table.TypeName] = rows
		}
		rows[rowID] = model
	}

	return nil
}

func (f *Fixture) decodeField(strct reflect.Value, field *schema.Field, value *yaml.Node) error {
	fv := field.Value(strct)
	iface := fv.Addr().Interface()

	if value.Tag != "!!str" {
		return value.Decode(iface)
	}

	if ss := funcNameRE.FindStringSubmatch(value.Value); len(ss) > 0 {
		if fn, ok := f.funcMap[ss[1]].(func() interface{}); ok {
			return scanFieldValue(strct, field, fn())
		}
	}

	if tplRE.MatchString(value.Value) {
		src, err := f.eval(value.Value)
		if err != nil {
			return err
		}
		return scanFieldValue(strct, field, src)
	}

	if v, ok := iface.(yaml.Unmarshaler); ok {
		return v.UnmarshalYAML(value)
	}

	if _, ok := iface.(sql.Scanner); ok {
		var str string
		if err := value.Decode(&str); err != nil {
			return err
		}
		return field.ScanValue(strct, str)
	}

	return value.Decode(iface)
}

func (f *Fixture) dropTable(ctx context.Context, table *schema.Table) error {
	if _, ok := f.seenTables[table.Name]; ok {
		return nil
	}
	f.seenTables[table.Name] = struct{}{}

	if _, err := f.db.NewDropTable().
		Model(table.ZeroIface).
		IfExists().
		Cascade().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := f.db.NewCreateTable().
		Model(table.ZeroIface).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (f *Fixture) truncateTable(ctx context.Context, table *schema.Table) error {
	if _, ok := f.seenTables[table.Name]; ok {
		return nil
	}
	f.seenTables[table.Name] = struct{}{}

	if _, err := f.db.NewTruncateTable().
		Model(table.ZeroIface).
		Cascade().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (f *Fixture) eval(templ string) (interface{}, error) {
	if v, ok := f.evalFuncCall(templ); ok {
		return v, nil
	}

	tpl, err := template.New("").Funcs(f.funcMap).Parse(templ)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err := tpl.Execute(&buf, f.modelRows); err != nil {
		return nil, err
	}

	return buf.String(), nil
}

func (f *Fixture) evalFuncCall(templ string) (interface{}, bool) {
	tree, err := parse.Parse("", templ, "{{", "}}", f.funcMap)
	if err != nil {
		return nil, false
	}

	root := tree[""].Root
	if len(root.Nodes) != 1 {
		return nil, false
	}

	action, ok := root.Nodes[0].(*parse.ActionNode)
	if !ok {
		return nil, false
	}

	if len(action.Pipe.Cmds) != 1 {
		return nil, false
	}

	args := action.Pipe.Cmds[0].Args
	if len(args) == 0 {
		return nil, false
	}

	funcName, ok := args[0].(*parse.IdentifierNode)
	if !ok {
		return nil, false
	}

	fn, ok := f.funcMap[funcName.Ident]
	if !ok {
		return nil, false
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	if fnType.NumOut() != 1 {
		return nil, false
	}

	args = args[1:]
	if len(args) != fnType.NumIn() {
		return nil, false
	}
	argValues := make([]reflect.Value, len(args))

	for i, node := range args {
		switch node := node.(type) {
		case *parse.StringNode:
			argValues[i] = reflect.ValueOf(node.Text)
		case *parse.NumberNode:
			switch {
			case node.IsInt:
				argValues[i] = reflect.ValueOf(node.Int64)
			case node.IsUint:
				argValues[i] = reflect.ValueOf(node.Uint64)
			case node.IsFloat:
				argValues[i] = reflect.ValueOf(node.Float64)
			case node.IsComplex:
				argValues[i] = reflect.ValueOf(node.Complex128)
			default:
				argValues[i] = reflect.ValueOf(node.Text)
			}
		case *parse.BoolNode:
			argValues[i] = reflect.ValueOf(node.True)
		default:
			return nil, false
		}
	}

	out := fnValue.Call(argValues)
	return out[0].Interface(), true
}

type fixtureData struct {
	Model string `yaml:"model"`
	Rows  []row  `yaml:"rows"`
}

type row map[string]yaml.Node

func asString(rv reflect.Value) string {
	switch rv.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	}
	return fmt.Sprintf("%v", rv.Interface())
}

func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"now": func() interface{} {
			return time.Now()
		},
	}
}

func scanFieldValue(strct reflect.Value, field *schema.Field, value interface{}) error {
	if v := reflect.ValueOf(value); v.CanConvert(field.StructField.Type) {
		field.Value(strct).Set(v.Convert(field.StructField.Type))
		return nil
	}
	return field.ScanValue(strct, value)
}
