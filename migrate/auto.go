package migrate

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/schema/inspector"
)

type AutoMigrator struct {
	db *bun.DB

	// models limit the set of tables considered for the migration.
	models []interface{}

	// dbInspector creates the current state for the target database.
	dbInspector schema.Inspector

	// modelInspector creates the desired state based on the model definitions.
	modelInspector schema.Inspector
}

func NewAutoMigrator(db *bun.DB) (*AutoMigrator, error) {
	dialect := db.Dialect()
	withInspector, ok := dialect.(inspector.Dialect)
	if !ok {
		return nil, fmt.Errorf("%q dialect does not implement inspector.Dialect", dialect.Name())
	}

	return &AutoMigrator{
		db:          db,
		dbInspector: withInspector.Inspector(db),
	}, nil
}

func (am *AutoMigrator) SetModels(models ...interface{}) {
	am.models = models
}

func (am *AutoMigrator) diff(ctx context.Context) (Changeset, error) {
	var changes Changeset
	var err error

	// TODO: do on "SetModels"
	am.modelInspector = schema.NewInspector(am.db.Dialect(), am.models...)

	_, err = am.dbInspector.Inspect(ctx)
	if err != nil {
		return changes, err
	}

	_, err = am.modelInspector.Inspect(ctx)
	if err != nil {
		return changes, err
	}
	return changes, nil
}

func (am *AutoMigrator) Migrate(ctx context.Context) error {
	return nil
}

// INTERNAL -------------------------------------------------------------------

// Operation is an abstraction a level above a MigrationFunc.
// Apart from storing the function to execute the change,
// it knows how to *write* the corresponding code, and what the reverse operation is.
type Operation interface {
	Func() MigrationFunc
}

type RenameTable struct {
	From string
	To   string
}

func (rt *RenameTable) Func() MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		db.Dialect()
		return nil
	}
}

// Changeset is a set of changes that alter database state.
type Changeset struct {
	operations []Operation
}

func (c Changeset) Operations() []Operation {
	return c.operations
}

func (c *Changeset) Add(op Operation) {
	c.operations = append(c.operations, op)
}

type Detector struct{}

func (d *Detector) Diff(got, want schema.State) Changeset {
	var changes Changeset

	// Detect renamed models
	oldModels := newTableSet(got.Tables...)
	newModels := newTableSet(want.Tables...)

	addedModels := newModels.Sub(oldModels)
	for _, added := range addedModels.Values() {
		removedModels := oldModels.Sub(newModels)
		for _, removed := range removedModels.Values() {
			if !haveSameSignature(added, removed) {
				continue
			}
			changes.Add(&RenameTable{
				From: removed.Name,
				To:   added.Name,
			})
		}
	}

	return changes
}

// haveSameSignature determines if two tables have the same "signature".
func haveSameSignature(t1, t2 schema.TableDef) bool {
	sig1 := newSignature(t1)
	sig2 := newSignature(t2)
	return sig1.Equals(sig2)
}

// tableSet stores unique table definitions.
type tableSet struct {
	underlying map[string]schema.TableDef
}

func newTableSet(initial ...schema.TableDef) tableSet {
	set := tableSet{
		underlying: make(map[string]schema.TableDef),
	}
	for _, t := range initial {
		set.Add(t)
	}
	return set
}

func (set tableSet) Add(t schema.TableDef) {
	set.underlying[t.Name] = t
}

func (set tableSet) Remove(s string) {
	delete(set.underlying, s)
}

func (set tableSet) Values() (tables []schema.TableDef) {
	for _, t := range set.underlying {
		tables = append(tables, t)
	}
	return
}

func (set tableSet) Sub(other tableSet) tableSet {
	res := set.clone()
	for v := range other.underlying {
		if _, ok := set.underlying[v]; ok {
			res.Remove(v)
		}
	}
	return res
}

func (set tableSet) clone() tableSet {
	res := newTableSet()
	for _, t := range set.underlying {
		res.Add(t)
	}
	return res
}

// signature is a set of column definitions, which allows "relation/name-agnostic" comparison between them;
// meaning that two columns are considered equal if their types are the same.
type signature struct {

	// underlying stores the number of occurences for each unique column type.
	// It helps to account for the fact that a table might have multiple columns that have the same type.
	underlying map[schema.ColumnDef]int
}

func newSignature(t schema.TableDef) signature {
	s := signature{
		underlying: make(map[schema.ColumnDef]int),
	}
	s.scan(t)
	return s
}

// scan iterates over table's field and counts occurrences of each unique column definition.
func (s *signature) scan(t schema.TableDef) {
	for _, c := range t.Columns {
		s.underlying[c]++
	}
}

// Equals returns true if 2 signatures share an identical set of columns.
func (s *signature) Equals(other signature) bool {
	for k, count := range s.underlying {
		if countOther, ok := other.underlying[k]; !ok || countOther != count {
			return false
		}
	}
	return true
}
