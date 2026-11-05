package db2dialect

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
)

func TestDialect(t *testing.T) {
	d := New()

	if d.Name() != dialect.DB2 {
		t.Fatalf("expected dialect.DB2, got %v", d.Name())
	}
	if d.IdentQuote() != '"' {
		t.Fatalf("expected identifier quote to be a double quote")
	}
	if !d.Features().Has(feature.OffsetFetch) {
		t.Fatal("expected OffsetFetch feature to be enabled")
	}
	if !d.Features().Has(feature.Identity) {
		t.Fatal("expected Identity feature to be enabled")
	}
	if d.DefaultVarcharLen() != 255 {
		t.Fatalf("expected default varchar length 255, got %d", d.DefaultVarcharLen())
	}
	if d.DefaultSchema() != "" {
		t.Fatalf("expected empty default schema (current authorization ID), got %q", d.DefaultSchema())
	}
}

func TestAppendBool(t *testing.T) {
	d := New()

	if got := string(d.AppendBool(nil, true)); got != "1" {
		t.Fatalf("expected %q, got %q", "1", got)
	}
	if got := string(d.AppendBool(nil, false)); got != "0" {
		t.Fatalf("expected %q, got %q", "0", got)
	}
}

func TestAppendTime(t *testing.T) {
	d := New()

	if got := string(d.AppendTime(nil, time.Time{})); got != "NULL" {
		t.Fatalf("expected zero time to append NULL, got %q", got)
	}

	tm := time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.FixedZone("test", 2*60*60))
	if got := string(d.AppendTime(nil, tm)); got != "'2026-03-04 03:06:07.123456'" {
		t.Fatalf("expected UTC timestamp, got %q", got)
	}
}

func TestExplicitConstructors(t *testing.T) {
	tests := []struct {
		name   string
		new    func() *Dialect
		target TargetPlatform
	}{
		{"LUW", NewLUW, TargetLUW},
		{"z/OS", NewZOS, TargetZOS},
		{"IBM i", NewIBMi, TargetIBMi},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := test.new()
			if d.Target() != test.target {
				t.Fatalf("expected target %v, got %v", test.target, d.Target())
			}
			if !d.targetSetExplicitly {
				t.Fatal("explicit constructor must disable target auto-detection")
			}
			if d.autoDetected {
				t.Fatal("explicit constructor should not mark target as auto-detected")
			}
		})
	}
}

func TestClassifyDBMSName(t *testing.T) {
	tests := []struct {
		name string
		want TargetPlatform
	}{
		{"DB2/LINUXX8664", TargetLUW},
		{"DB2", TargetZOS},
		{"DSN11015", TargetZOS},
		{"AS/400", TargetIBMi},
		{"IDS", TargetLUW},
		{"SOMETHING_ELSE", TargetLUW},
		{"", TargetLUW},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyDBMSName(test.name); got != test.want {
				t.Fatalf("classifyDBMSName(%q) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}

// infoType mirrors the driver's named api.SQLUSMALLINT type to verify
// getDBMSName's reflection-based call works against a non-uint16 named type.
type infoType uint16

type fakeGetInfoConn struct {
	name string
	err  error
}

func (c fakeGetInfoConn) GetInfo(t infoType) (string, error) {
	if t != sqlDBMSName {
		return "", fmt.Errorf("unexpected infoType %d", t)
	}
	return c.name, c.err
}

func TestGetDBMSName(t *testing.T) {
	name, err := getDBMSName(fakeGetInfoConn{name: "DB2/LINUXX8664"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "DB2/LINUXX8664" {
		t.Fatalf("expected %q, got %q", "DB2/LINUXX8664", name)
	}

	if _, err := getDBMSName(fakeGetInfoConn{err: errors.New("boom")}); err == nil {
		t.Fatal("expected error to propagate")
	}

	if _, err := getDBMSName(struct{}{}); !errors.Is(err, errUnsupportedDriverConn) {
		t.Fatalf("expected errUnsupportedDriverConn, got %v", err)
	}
}

func TestCatalogSchema(t *testing.T) {
	tests := []struct {
		target TargetPlatform
		want   string
	}{
		{TargetLUW, "SYSCAT"},
		{TargetZOS, "SYSIBM"},
		{TargetIBMi, "QSYS2"},
	}

	for _, test := range tests {
		d := New(WithTarget(test.target))
		if got := d.CatalogSchema(); got != test.want {
			t.Errorf("target %v: expected catalog %q, got %q", test.target, test.want, got)
		}
	}
}

// TestAppendOffsetLimit verifies pagination SQL generation
func TestAppendOffsetLimit(t *testing.T) {
	tests := []struct {
		name     string
		offset   int64
		limit    int64
		expected string
	}{
		{"no pagination", 0, 0, ""},
		{"limit only", 0, 10, " FETCH NEXT 10 ROWS ONLY"},
		{"offset only", 20, 0, " OFFSET 20 ROWS"},
		{"offset and limit", 20, 10, " OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := make([]byte, 0, 64)
			result := AppendOffsetLimit(b, test.offset, test.limit)
			resultStr := string(result)

			if resultStr != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, resultStr)
			}
		})
	}
}

// TestAppendOffsetLimitNegativeValuesPanic verifies negative offset/limit are rejected
func TestAppendOffsetLimitNegativeValuesPanic(t *testing.T) {
	tests := []struct {
		name   string
		offset int64
		limit  int64
	}{
		{"negative offset", -1, 10},
		{"negative limit", 20, -1},
		{"min int64 offset", math.MinInt64, 10},
		{"min int64 limit", 20, math.MinInt64},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for offset=%d, limit=%d", test.offset, test.limit)
				}
			}()
			b := make([]byte, 0, 64)
			AppendOffsetLimit(b, test.offset, test.limit)
		})
	}
}
