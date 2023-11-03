package sqlschema

type State struct {
	Tables []Table
}

type Table struct {
	Schema  string
	Name    string
	Model   interface{}
	Columns map[string]Column
}

// Column stores attributes of a database column.
type Column struct {
	SQLType         string
	DefaultValue    string
	IsPK            bool
	IsNullable      bool
	IsAutoIncrement bool
	IsIdentity      bool
}

// EqualSignatures determines if two tables have the same "signature".
func EqualSignatures(t1, t2 Table) bool {
	sig1 := newSignature(t1)
	sig2 := newSignature(t2)
	return sig1.Equals(sig2)
}

// signature is a set of column definitions, which allows "relation/name-agnostic" comparison between them;
// meaning that two columns are considered equal if their types are the same.
type signature struct {

	// underlying stores the number of occurences for each unique column type.
	// It helps to account for the fact that a table might have multiple columns that have the same type.
	underlying map[Column]int
}

func newSignature(t Table) signature {
	s := signature{
		underlying: make(map[Column]int),
	}
	s.scan(t)
	return s
}

// scan iterates over table's field and counts occurrences of each unique column definition.
func (s *signature) scan(t Table) {
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
