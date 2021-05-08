package fixture

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"strconv"
	"text/template"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"gopkg.in/yaml.v3"
)

type LoaderOption func(f *Loader)

func WithDropTables() LoaderOption {
	return func(f *Loader) {
		if f.truncateTables {
			panic("don't use WithDropTables together with WithTruncateTables")
		}
		f.dropTables = true
		f.seenTables = make(map[string]struct{})
	}
}

func WithTruncateTables() LoaderOption {
	return func(f *Loader) {
		if f.truncateTables {
			panic("don't use WithTruncateTables together with WithDropTables")
		}
		f.truncateTables = true
		f.seenTables = make(map[string]struct{})
	}
}

type Loader struct {
	db *bun.DB

	dropTables     bool
	truncateTables bool
	seenTables     map[string]struct{}

	modelRows map[string]map[string]interface{}
}

func NewLoader(db *bun.DB, opts ...LoaderOption) *Loader {
	f := &Loader{
		db: db,

		modelRows: make(map[string]map[string]interface{}),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *Loader) Get(model, rowID string) (interface{}, error) {
	rows, ok := f.modelRows[model]
	if !ok {
		return nil, fmt.Errorf("fixture: unknown model=%q", model)
	}

	row, ok := rows[rowID]
	if !ok {
		return nil, fmt.Errorf("fixture: unknown row=%q for model=%q", row, model)
	}

	return row, nil
}

func (f *Loader) MustGet(table, rowID string) interface{} {
	row, err := f.Get(table, rowID)
	if err != nil {
		panic(err)
	}
	return row
}

func (f *Loader) Load(ctx context.Context, fsys fs.FS, names ...string) error {
	for _, name := range names {
		if err := f.load(ctx, fsys, name); err != nil {
			return err
		}
	}
	return nil
}

func (f *Loader) load(ctx context.Context, fsys fs.FS, name string) error {
	fh, err := fsys.Open(name)
	if err != nil {
		return err
	}

	var fixtures []Fixture

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

func (f *Loader) addFixture(ctx context.Context, fixture *Fixture) error {
	table := f.db.Dialect().Tables().ByModel(fixture.Model)
	if table == nil {
		return fmt.Errorf("fixture: can't find model=%q (use db.RegisterModel)", fixture.Model)
	}

	if f.dropTables {
		if err := f.dropTable(ctx, table); err != nil {
			return err
		}
	} else if f.truncateTables {
		if err := f.truncateTable(ctx, table); err != nil {
			return err
		}
	}

	for _, row := range fixture.Rows {
		if err := f.addRow(ctx, table, row); err != nil {
			return err
		}
	}

	return nil
}

func (f *Loader) addRow(ctx context.Context, table *schema.Table, row row) error {
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

		if value.Tag == "!!str" && isTemplate(value.Value) {
			res, err := f.eval(value.Value)
			if err != nil {
				return err
			}

			if res != value.Value {
				if err := field.ScanValue(strct, res); err != nil {
					return err
				}
				continue
			}
		}

		fv := field.Value(strct)
		if err := value.Decode(fv.Addr().Interface()); err != nil {
			return err
		}
	}

	iface := strct.Addr().Interface()
	if _, err := f.db.NewInsert().
		Model(iface).
		Exec(ctx); err != nil {
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
		rows[rowID] = iface
	}

	return nil
}

func (f *Loader) dropTable(ctx context.Context, table *schema.Table) error {
	if _, ok := f.seenTables[table.Name]; ok {
		return nil
	}
	f.seenTables[table.Name] = struct{}{}

	if _, err := f.db.NewDropTable().
		Model(table.ZeroIface).
		IfExists().
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

func (f *Loader) truncateTable(ctx context.Context, table *schema.Table) error {
	if _, ok := f.seenTables[table.Name]; ok {
		return nil
	}
	f.seenTables[table.Name] = struct{}{}

	if _, err := f.db.NewTruncateTable().
		Model(table.ZeroIface).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (f *Loader) eval(templ string) (string, error) {
	tpl, err := template.New("").Parse(templ)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err := tpl.Execute(&buf, f.modelRows); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type Fixture struct {
	Model string `yaml:"model"`
	Rows  []row  `yaml:"rows"`
}

type row map[string]yaml.Node

var tplRE = regexp.MustCompile(`\{\{ .+ \}\}`)

func isTemplate(s string) bool {
	return tplRE.MatchString(s)
}

func asString(rv reflect.Value) string {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	}
	return fmt.Sprintf("%v", rv.Interface())
}
