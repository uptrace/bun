package mysqldialect

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"
)

type timeModel struct {
	bun.BaseModel `bun:"table:time_model"`

	ID        int64        `bun:"id,pk"`
	At        time.Time    `bun:"at,notnull"`
	Opt       bun.NullTime `bun:"opt"`
	Ptr       *time.Time   `bun:"ptr"`
	CreatedAt time.Time    `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	SeenAt    time.Time    `bun:"seen_at,nullzero,notnull,default:NOW()"`
	Typed     time.Time    `bun:"typed,type:datetime,notnull,default:current_timestamp"`
	Name      string       `bun:"name,notnull,default:'x'"`
}

// noServer lets bun.NewDB build DDL without a database; Init logs that it
// cannot discover the server version.
type noServer struct{}

func (noServer) Connect(context.Context) (driver.Conn, error) { return nil, errors.New("no server") }
func (noServer) Driver() driver.Driver                        { return nil }

func createTableSQL(d *Dialect) string {
	db := bun.NewDB(sql.OpenDB(noServer{}), d)
	return db.NewCreateTable().Model((*timeModel)(nil)).String()
}

func TestCreateTableTimePrecision(t *testing.T) {
	tests := []struct {
		name string
		d    *Dialect
		want []string
	}{
		{"whole seconds by default", New(), []string{
			"`at` DATETIME NOT NULL",
			"`opt` DATETIME,",
			"`ptr` DATETIME,",
			"`created_at` DATETIME NOT NULL DEFAULT current_timestamp,",
			"`seen_at` DATETIME NOT NULL DEFAULT NOW(),",
			"`typed` datetime NOT NULL DEFAULT current_timestamp,",
		}},
		{"microseconds", New(WithTimePrecision(6)), []string{
			"`at` DATETIME(6) NOT NULL",
			"`opt` DATETIME(6),",
			"`ptr` DATETIME(6),",
			// MySQL refuses DATETIME(6) DEFAULT current_timestamp (error 1067).
			"`created_at` DATETIME(6) NOT NULL DEFAULT current_timestamp(6),",
			"`seen_at` DATETIME(6) NOT NULL DEFAULT NOW(6),",
			// An explicit type is the model's decision, and so is its default.
			"`typed` datetime NOT NULL DEFAULT current_timestamp,",
			"`name` VARCHAR(255) NOT NULL DEFAULT 'x'",
		}},
		{"zero is whole seconds", New(WithTimePrecision(0)), []string{
			"`at` DATETIME NOT NULL",
			"`created_at` DATETIME NOT NULL DEFAULT current_timestamp,",
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := createTableSQL(test.d)
			for _, want := range test.want {
				if !strings.Contains(got, want) {
					t.Errorf("CREATE TABLE lacks %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestCurrentTimeWithPrecision(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"current_timestamp", "current_timestamp(6)"},
		{"CURRENT_TIMESTAMP()", "CURRENT_TIMESTAMP(6)"},
		{"now()", "now(6)"},
		{"localtime", "localtime(6)"},
		{"LOCALTIMESTAMP", "LOCALTIMESTAMP(6)"},
		{"current_timestamp(3)", "current_timestamp(3)"},
		{"'2026-01-01 00:00:00'", "'2026-01-01 00:00:00'"},
		{"", ""},
	}
	for _, test := range tests {
		if got := currentTimeWithPrecision(test.in, 6); got != test.want {
			t.Errorf("currentTimeWithPrecision(%q, 6) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestWithTimePrecisionOutOfRange(t *testing.T) {
	for _, fsp := range []int{-1, 7} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("WithTimePrecision(%d) did not panic", fsp)
				}
			}()
			New(WithTimePrecision(fsp))
		}()
	}
}
