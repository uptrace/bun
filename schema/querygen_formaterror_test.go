package schema

import (
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const nulMarker = "?!(bun: string contains a NUL byte (0x00))"

// strippingDialect accepts every string and strips NUL, like mysqldialect.
type strippingDialect struct {
	lcTestDialect
}

func (strippingDialect) CheckString(string) error { return nil }

func (strippingDialect) AppendString(b []byte, s string) []byte {
	return BaseDialect{}.AppendString(b, strings.ReplaceAll(s, "\x00", ""))
}

type failingValuer struct{ err error }

func (v failingValuer) Value() (driver.Value, error) { return nil, v.err }

type valuerValuer struct{}

func (valuerValuer) Value() (driver.Value, error) { return failingValuer{}, nil }

type failingAppender struct{ err error }

func (a failingAppender) AppendQuery(gen QueryGen, b []byte) ([]byte, error) { return nil, a.err }

func TestQueryGen_ErrorSink(t *testing.T) {
	errValue := errors.New("bun-test: value failed")
	gen := NewQueryGen(lcTestDialect{newNopDialect()})

	tests := []struct {
		name  string
		gen   QueryGen
		query string
		args  []any
		want  error // nil: no formatting error
	}{
		{"NUL string", gen, "name = ?", []any{"a\x00b"}, ErrStringNul},
		{"failing driver.Valuer", gen, "v = ?", []any{failingValuer{errValue}}, errValue},
		{"failing QueryAppender", gen, "v = ?", []any{failingAppender{errValue}}, errValue},
		{"first error wins", gen, "a = ? AND b = ?", []any{failingValuer{errValue}, "a\x00b"}, errValue},
		{"value that looks like a marker", gen, "v = ?", []any{"x?!(y)"}, nil},
		{"marker text without NUL", gen, "v = ?", []any{nulMarker}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := tt.gen.WithErrorSink()
			sink.FormatQuery(tt.query, tt.args...)
			err := sink.Err()
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
			var fe *FormatError
			require.ErrorAs(t, err, &fe)
			require.Equal(t, tt.want.Error(), err.Error())
		})
	}
}

// Without a sink, errors are only rendered: String() output is unchanged.
func TestQueryGen_NoErrorSink(t *testing.T) {
	gen := NewQueryGen(lcTestDialect{newNopDialect()})

	require.Equal(t, "name = "+nulMarker, gen.FormatQuery("name = ?", "a\x00b"))
	require.NoError(t, gen.Err())

	b := gen.AppendError(nil, errors.New("bun-test: failed"))
	require.Equal(t, "?!(bun-test: failed)", string(b))
	require.NoError(t, gen.Err())
}

func TestQueryGen_ErrorSinkKeepsMarker(t *testing.T) {
	sink := NewQueryGen(lcTestDialect{newNopDialect()}).WithErrorSink()

	require.Equal(t, "name = "+nulMarker, sink.FormatQuery("name = ?", "a\x00b"))
	require.ErrorIs(t, sink.Err(), ErrStringNul)
}

// Regression test: WithArg and WithNamedArg used to build a new QueryGen and
// drop the sink, and every select, insert, update and delete renders through
// WithArg.
func TestQueryGen_ErrorSinkSurvivesWithArg(t *testing.T) {
	gen := NewQueryGen(lcTestDialect{newNopDialect()})

	sink := gen.WithErrorSink()
	sink.WithNamedArg("name", "a\x00b").FormatQuery("name = ?name")
	require.ErrorIs(t, sink.Err(), ErrStringNul)

	sink = gen.WithErrorSink()
	sink.WithArg(&namedArg{name: "name", value: "a\x00b"}).FormatQuery("name = ?name")
	require.ErrorIs(t, sink.Err(), ErrStringNul)
}

// The dialect decides which strings it refuses: an override of CheckString
// wins over the one promoted from BaseDialect.
func TestQueryGen_ErrorSinkAsksTheDialect(t *testing.T) {
	sink := NewQueryGen(strippingDialect{lcTestDialect{newNopDialect()}}).WithErrorSink()

	require.Equal(t, "name = 'ab'", sink.FormatQuery("name = ?", "a\x00b"))
	require.NoError(t, sink.Err())
}

func TestQueryGen_ErrorSinksAreSeparate(t *testing.T) {
	gen := NewQueryGen(lcTestDialect{newNopDialect()})
	first, second := gen.WithErrorSink(), gen.WithErrorSink()

	first.FormatQuery("name = ?", "a\x00b")
	require.ErrorIs(t, first.Err(), ErrStringNul)
	require.NoError(t, second.Err())
	require.NoError(t, gen.Err())
}

func TestQueryGen_ErrorSinkAppendPaths(t *testing.T) {
	errValue := errors.New("bun-test: value failed")
	sink := NewQueryGen(lcTestDialect{newNopDialect()}).WithErrorSink()

	b := sink.Append(nil, failingAppender{errValue})
	require.Equal(t, "?!(bun-test: value failed)", string(b))
	require.ErrorIs(t, sink.Err(), errValue)

	sink = NewQueryGen(lcTestDialect{newNopDialect()}).WithErrorSink()
	sink.Append(nil, valuerValuer{})
	require.EqualError(t, sink.Err(), "driver.Valuer returns unsupported type schema.failingValuer")

	sink = NewQueryGen(lcTestDialect{newNopDialect()}).WithErrorSink()
	sink.AppendError(nil, errValue)
	require.ErrorIs(t, sink.Err(), errValue)
}
