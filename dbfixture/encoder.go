package dbfixture

import (
	"fmt"
	"io"
	"reflect"

	"github.com/uptrace/bun"
	"gopkg.in/yaml.v3"
)

type fixtureRows struct {
	Model string      `yaml:"model"`
	Rows  interface{} `yaml:"rows"`
}

type Encoder struct {
	db  bun.IDB
	enc *yaml.Encoder
}

func NewEncoder(db bun.IDB, w io.Writer) *Encoder {
	return &Encoder{
		db:  db,
		enc: yaml.NewEncoder(w),
	}
}

func (e *Encoder) Encode(multiRows ...interface{}) error {
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
