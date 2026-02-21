package dbfixture

import (
	"fmt"
	"io"
	"reflect"

	"github.com/uptrace/bun"
	"gopkg.in/yaml.v3"
)

type fixtureRows struct {
	Model string `yaml:"model"`
	Rows  any    `yaml:"rows"`
}

// Encoder writes fixture data as YAML.
type Encoder struct {
	db  bun.IDB
	enc *yaml.Encoder
}

// NewEncoder creates an Encoder that writes YAML to w.
func NewEncoder(db bun.IDB, w io.Writer) *Encoder {
	return &Encoder{
		db:  db,
		enc: yaml.NewEncoder(w),
	}
}

// Encode encodes the given model slices as YAML fixture data.
func (e *Encoder) Encode(multiRows ...any) error {
	fixtures := make([]fixtureRows, len(multiRows))

	for i, rows := range multiRows {
		v := reflect.ValueOf(rows)
		if v.Kind() != reflect.Slice {
			return fmt.Errorf("dbfixture: got %T, wanted a slice", rows)
		}

		table := e.db.Dialect().Tables().Get(v.Type().Elem())
		fixtures[i] = fixtureRows{
			Model: table.TypeName,
			Rows:  rows,
		}
	}

	return e.enc.Encode(fixtures)
}
