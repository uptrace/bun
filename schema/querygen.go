package schema

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/parser"
)

var nopQueryGen = QueryGen{
	dialect: newNopDialect(),
}

type QueryGen struct {
	dialect Dialect
	args    *namedArgList
	errs    *formatErrors // nil: formatting errors are only rendered as markers
}

// FormatError reports a value that could not be formatted into a query.
// Executors return it before anything is sent to the database. Unwrap gives
// the cause, e.g. ErrStringNul or the error of a driver.Valuer.
type FormatError struct {
	Err error
}

func (e *FormatError) Error() string { return e.Err.Error() }
func (e *FormatError) Unwrap() error { return e.Err }

// formatErrors is shared by every copy of a QueryGen made after WithErrorSink.
type formatErrors struct {
	first error
}

func NewQueryGen(dialect Dialect) QueryGen {
	return QueryGen{
		dialect: dialect,
	}
}

func NewNopQueryGen() QueryGen {
	return nopQueryGen
}

func (f QueryGen) IsNop() bool {
	return f.dialect.Name() == dialect.Invalid
}

func (f QueryGen) Dialect() Dialect {
	return f.dialect
}

func (f QueryGen) IdentQuote() byte {
	return f.dialect.IdentQuote()
}

// guardLineComment inserts a space when a negative numeric literal is about to
// be appended immediately after a '-'. Without it, formatting an expression
// such as "col - ?" with a negative argument produces "col --1", where the "--"
// starts a SQL line comment that can be abused for SQL injection. This mirrors
// the fix applied to driver/pgdriver (CVE-2024-44906).
func guardLineComment(b []byte, negative bool) []byte {
	if negative && len(b) > 0 && b[len(b)-1] == '-' {
		return append(b, ' ')
	}
	return b
}

func (gen QueryGen) Append(b []byte, v any) []byte {
	switch v := v.(type) {
	case nil:
		return dialect.AppendNull(b)
	case bool:
		return gen.Dialect().AppendBool(b, v)
	case int:
		return strconv.AppendInt(guardLineComment(b, v < 0), int64(v), 10)
	case int32:
		return strconv.AppendInt(guardLineComment(b, v < 0), int64(v), 10)
	case int64:
		return strconv.AppendInt(guardLineComment(b, v < 0), v, 10)
	case uint:
		return gen.Dialect().AppendUint64(b, uint64(v))
	case uint32:
		return gen.Dialect().AppendUint32(b, v)
	case uint64:
		return gen.Dialect().AppendUint64(b, v)
	case float32:
		return dialect.AppendFloat32(guardLineComment(b, v < 0), v)
	case float64:
		return dialect.AppendFloat64(guardLineComment(b, v < 0), v)
	case string:
		return gen.appendString(b, v)
	case time.Time:
		return gen.Dialect().AppendTime(b, v)
	case []byte:
		return gen.Dialect().AppendBytes(b, v)
	case QueryAppender:
		return AppendQueryAppender(gen, b, v)
	default:
		vv := reflect.ValueOf(v)
		if vv.Kind() == reflect.Pointer && vv.IsNil() {
			return dialect.AppendNull(b)
		}
		appender := Appender(gen.Dialect(), vv.Type())
		return appender(gen, b, vv)
	}
}

func (f QueryGen) AppendName(b []byte, name string) []byte {
	return dialect.AppendName(b, name, f.IdentQuote())
}

func (f QueryGen) AppendIdent(b []byte, ident string) []byte {
	return dialect.AppendIdent(b, ident, f.IdentQuote())
}

func (f QueryGen) AppendValue(b []byte, v reflect.Value) []byte {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return dialect.AppendNull(b)
	}
	appender := Appender(f.dialect, v.Type())
	return appender(f, b, v)
}

func (f QueryGen) HasFeature(feature feature.Feature) bool {
	return f.dialect.Features().Has(feature)
}

// WithArg returns a copy that also resolves named arguments from arg. The
// copy keeps the error sink, if any.
func (f QueryGen) WithArg(arg NamedArgAppender) QueryGen {
	f.args = f.args.WithArg(arg)
	return f
}

// WithNamedArg returns a copy that resolves ?name to value. The copy keeps
// the error sink, if any.
func (f QueryGen) WithNamedArg(name string, value any) QueryGen {
	f.args = f.args.WithArg(&namedArg{name: name, value: value})
	return f
}

// WithErrorSink returns a copy that records formatting errors; Err reports
// the first one. Copies made from the result share the record, so a whole
// statement can be checked once after rendering it. Without a sink, errors
// are only rendered as markers, which keeps String() output unchanged.
func (gen QueryGen) WithErrorSink() QueryGen {
	gen.errs = new(formatErrors)
	return gen
}

// Err returns the first formatting error recorded since WithErrorSink as a
// *FormatError, or nil.
func (gen QueryGen) Err() error {
	if gen.errs == nil || gen.errs.first == nil {
		return nil
	}
	return &FormatError{Err: gen.errs.first}
}

// AppendError renders err as a marker, like dialect.AppendError, and records
// it on the error sink.
func (gen QueryGen) AppendError(b []byte, err error) []byte {
	if gen.errs != nil && gen.errs.first == nil {
		gen.errs.first = err
	}
	return dialect.AppendError(b, err)
}

// appendString formats s through the dialect. With an error sink, a string
// the dialect refuses (StringChecker) is recorded instead of formatted.
func (gen QueryGen) appendString(b []byte, s string) []byte {
	if gen.errs != nil {
		if c, ok := gen.dialect.(StringChecker); ok {
			if err := c.CheckString(s); err != nil {
				return gen.AppendError(b, err)
			}
		}
	}
	return gen.dialect.AppendString(b, s)
}

func (f QueryGen) FormatQuery(query string, args ...any) string {
	if f.IsNop() || (args == nil && f.args == nil) || strings.IndexByte(query, '?') == -1 {
		return query
	}
	return internal.String(f.AppendQuery(nil, query, args...))
}

func (f QueryGen) AppendQuery(dst []byte, query string, args ...any) []byte {
	if f.IsNop() || (args == nil && f.args == nil) || strings.IndexByte(query, '?') == -1 {
		return append(dst, query...)
	}
	return f.append(dst, parser.NewString(query), args)
}

func (f QueryGen) append(dst []byte, p *parser.Parser, args []any) []byte {
	var namedArgs NamedArgAppender
	if len(args) == 1 {
		if v, ok := args[0].(NamedArgAppender); ok {
			namedArgs = v
		} else if v, ok := newStructArgs(f, args[0]); ok {
			namedArgs = v
		}
	}

	var argIndex int
	for p.Valid() {
		b, ok := p.ReadSep('?')
		if !ok {
			dst = append(dst, b...)
			continue
		}
		if len(b) > 0 && b[len(b)-1] == '\\' {
			dst = append(dst, b[:len(b)-1]...)
			dst = append(dst, '?')
			continue
		}
		dst = append(dst, b...)

		name, numeric := p.ReadIdentifier()
		if name != "" {
			if numeric {
				idx, err := strconv.Atoi(name)
				if err != nil {
					goto restore_arg
				}

				if idx >= len(args) {
					goto restore_arg
				}

				dst = f.appendArg(dst, args[idx])
				continue
			}

			if namedArgs != nil {
				dst, ok = namedArgs.AppendNamedArg(f, dst, name)
				if ok {
					continue
				}
			}

			dst, ok = f.args.AppendNamedArg(f, dst, name)
			if ok {
				continue
			}

		restore_arg:
			dst = append(dst, '?')
			dst = append(dst, name...)
			continue
		}

		if argIndex >= len(args) {
			dst = append(dst, '?')
			continue
		}

		arg := args[argIndex]
		argIndex++

		dst = f.appendArg(dst, arg)
	}

	return dst
}

func (gen QueryGen) appendArg(b []byte, arg any) []byte {
	switch arg := arg.(type) {
	case QueryAppender:
		bb, err := arg.AppendQuery(gen, b)
		if err != nil {
			return gen.AppendError(b, err)
		}
		return bb
	default:
		return gen.Append(b, arg)
	}
}

//------------------------------------------------------------------------------

type NamedArgAppender interface {
	AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool)
}

type namedArgList struct {
	arg  NamedArgAppender
	next *namedArgList
}

func (l *namedArgList) WithArg(arg NamedArgAppender) *namedArgList {
	return &namedArgList{
		arg:  arg,
		next: l,
	}
}

func (l *namedArgList) AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool) {
	for l != nil && l.arg != nil {
		if b, ok := l.arg.AppendNamedArg(gen, b, name); ok {
			return b, true
		}
		l = l.next
	}
	return b, false
}

//------------------------------------------------------------------------------

type namedArg struct {
	name  string
	value any
}

var _ NamedArgAppender = (*namedArg)(nil)

func (a *namedArg) AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool) {
	if a.name == name {
		return gen.appendArg(b, a.value), true
	}
	return b, false
}

//------------------------------------------------------------------------------

type structArgs struct {
	table *Table
	strct reflect.Value
}

var _ NamedArgAppender = (*structArgs)(nil)

func newStructArgs(gen QueryGen, strct any) (*structArgs, bool) {
	v := reflect.ValueOf(strct)
	if !v.IsValid() {
		return nil, false
	}

	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	return &structArgs{
		table: gen.Dialect().Tables().Get(v.Type()),
		strct: v,
	}, true
}

func (m *structArgs) AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool) {
	return m.table.AppendNamedArg(gen, b, name, m.strct)
}
