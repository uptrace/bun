package bunotel

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/schema"
)

// otelsqlInstrumName is the instrumentation name otelsql uses for the meter it
// creates in ReportDBStatsMetrics.
const otelsqlInstrumName = "github.com/uptrace/opentelemetry-go-extra/otelsql"

// TestQueryHookInitUsesConfiguredMeterProvider ensures that DB stats metrics are
// reported to the meter provider passed to WithMeterProvider instead of the
// global one. See https://github.com/uptrace/bun/issues/1270.
func TestQueryHookInitUsesConfiguredMeterProvider(t *testing.T) {
	global := setGlobalMeterProvider(t)

	configured := new(recordingMeterProvider)
	NewQueryHook(WithMeterProvider(configured)).Init(newTestDB(t))

	if !configured.hasMeter(otelsqlInstrumName) {
		t.Error("DB stats metrics were not reported to the configured meter provider")
	}
	if global.hasMeter(otelsqlInstrumName) {
		t.Error("DB stats metrics were reported to the global meter provider")
	}
}

// TestQueryHookInitFallsBackToGlobalMeterProvider ensures that hooks created
// without WithMeterProvider keep reporting to the global meter provider.
func TestQueryHookInitFallsBackToGlobalMeterProvider(t *testing.T) {
	global := setGlobalMeterProvider(t)

	NewQueryHook().Init(newTestDB(t))

	if !global.hasMeter(otelsqlInstrumName) {
		t.Error("DB stats metrics were not reported to the global meter provider")
	}
}

type narrowModel struct {
	bun.BaseModel `bun:"table:narrow,alias:n"`

	A, B, C int
}

type wideModel struct {
	bun.BaseModel `bun:"table:wide,alias:w"`

	A, B, C, D, E, F, G, H, I, J, K, L int
}

func TestWithColumnLimit(t *testing.T) {
	db := newTestDB(t)
	query := db.NewSelect().Model((*narrowModel)(nil))

	const listed = `SELECT "n"."a", "n"."b", "n"."c" FROM "narrow" AS "n"`
	const collapsed = `SELECT "n".'3 columns' FROM "narrow" AS "n"`

	tests := []struct {
		name  string
		limit int
		want  string
	}{
		{"below column count", 2, collapsed},
		{"equal to column count", 3, listed},
		{"above column count", 4, listed},
		{"zero lists every column", 0, listed},
		{"negative lists every column", -1, listed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewQueryHook(WithColumnLimit(tt.limit))
			if got := h.eventQuery(&bun.QueryEvent{IQuery: query}); got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestDefaultColumnLimit(t *testing.T) {
	db := newTestDB(t)
	event := &bun.QueryEvent{IQuery: db.NewSelect().Model((*wideModel)(nil))}

	if got := NewQueryHook().eventQuery(event); !strings.Contains(got, `'12 columns'`) {
		t.Errorf("default hook did not collapse 12 columns: %s", got)
	}
	if got := NewQueryHook(WithColumnLimit(0)).eventQuery(event); strings.Contains(got, "columns") {
		t.Errorf("WithColumnLimit(0) collapsed the column list: %s", got)
	}
}

// setGlobalMeterProvider installs a recording meter provider as the global one
// and restores the previous provider when the test finishes.
func setGlobalMeterProvider(t *testing.T) *recordingMeterProvider {
	t.Helper()

	prev := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(prev) })

	mp := new(recordingMeterProvider)
	otel.SetMeterProvider(mp)
	return mp
}

// recordingMeterProvider is a no-op meter provider that remembers which
// instrumentation names asked it for a meter.
type recordingMeterProvider struct {
	noop.MeterProvider

	names []string
}

func (p *recordingMeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	p.names = append(p.names, name)
	return p.MeterProvider.Meter(name, opts...)
}

func (p *recordingMeterProvider) hasMeter(name string) bool {
	for _, n := range p.names {
		if n == name {
			return true
		}
	}
	return false
}

// newTestDB returns a bun.DB that never connects to a database: reporting DB
// stats only requires sql.DB.Stats.
func newTestDB(t *testing.T) *bun.DB {
	t.Helper()

	db := bun.NewDB(sql.OpenDB(nopConnector{}), newNopDialect())
	t.Cleanup(func() { _ = db.Close() })
	return db
}

var errNotImplemented = errors.New("bunotel: not implemented")

type nopConnector struct{}

func (nopConnector) Connect(context.Context) (driver.Conn, error) { return nil, errNotImplemented }

func (c nopConnector) Driver() driver.Driver { return nopDriver{} }

type nopDriver struct{}

func (nopDriver) Open(string) (driver.Conn, error) { return nil, errNotImplemented }

type nopDialect struct {
	schema.BaseDialect

	tables *schema.Tables
}

var _ schema.Dialect = (*nopDialect)(nil)

func newNopDialect() *nopDialect {
	d := new(nopDialect)
	d.tables = schema.NewTables(d)
	return d
}

func (*nopDialect) Init(*sql.DB) {}

func (*nopDialect) Name() dialect.Name { return dialect.SQLite }

func (*nopDialect) Features() feature.Feature { return 0 }

func (d *nopDialect) Tables() *schema.Tables { return d.tables }

func (*nopDialect) OnTable(*schema.Table) {}

func (*nopDialect) IdentQuote() byte { return '"' }

func (*nopDialect) AppendSequence(b []byte, _ *schema.Table, _ *schema.Field) []byte { return b }

func (*nopDialect) DefaultVarcharLen() int { return 0 }

func (*nopDialect) DefaultSchema() string { return "" }
