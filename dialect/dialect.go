package dialect

type Name int

func (n Name) String() string {
	switch n {
	case PG:
		return "pg"
	case SQLite:
		return "sqlite"
	case MySQL:
		return "mysql"
	default:
		return "invalid"
	}
}

const (
	Invalid Name = iota
	PG
	SQLite
	MySQL
)
