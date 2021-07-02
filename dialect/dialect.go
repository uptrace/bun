package dialect

type Name int

func (n Name) String() string {
	switch n {
	case PG:
		return "pg"
	case SQLite:
		return "sqlite"
	case MySQL5:
		return "mysql5"
	case MySQL8:
		return "mysql8"
	default:
		return "invalid"
	}
}

const (
	Invalid Name = iota
	PG
	SQLite
	MySQL5
	MySQL8
)
