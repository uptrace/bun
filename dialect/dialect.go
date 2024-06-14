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
	case MSSQL:
		return "mssql"
	case Oracle:
		return "oracle"
	default:
		return "invalid"
	}
}

const (
	Invalid Name = iota
	PG
	SQLite
	MySQL
	MSSQL
	Oracle
)
