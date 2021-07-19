package schema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/parser"
)

type ArgAppender interface {
	AppendArg(fmter Formatter, b []byte, name string) ([]byte, bool)
}

type namedArg struct {
	name  string
	value interface{}
}

// TODO: try linked list
type namedArgs []namedArg

func (args namedArgs) Get(name string) (interface{}, bool) {
	for _, arg := range args {
		if arg.name == name {
			return arg.value, true
		}
	}
	return nil, false
}

//------------------------------------------------------------------------------

var nopFormatter = Formatter{
	dialect: newNopDialect(),
}

type Formatter struct {
	dialect   Dialect
	model     ArgAppender
	namedArgs namedArgs
}

func NewFormatter(dialect Dialect) Formatter {
	return Formatter{
		dialect: dialect,
	}
}

func NewNopFormatter() Formatter {
	return nopFormatter
}

func (f Formatter) String() string {
	if len(f.namedArgs) == 0 {
		return ""
	}

	ss := make([]string, len(f.namedArgs))
	for i, arg := range f.namedArgs {
		ss[i] = fmt.Sprintf("%s=%v", arg.name, arg.value)
	}
	return strings.Join(ss, " ")
}

func (f Formatter) IsNop() bool {
	return f.dialect.Name() == dialect.Invalid
}

func (f Formatter) Dialect() Dialect {
	return f.dialect
}

func (f Formatter) IdentQuote() byte {
	return f.dialect.IdentQuote()
}

func (f Formatter) AppendIdent(b []byte, ident string) []byte {
	return dialect.AppendIdent(b, ident, f.IdentQuote())
}

func (f Formatter) AppendValue(b []byte, v reflect.Value) []byte {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return dialect.AppendNull(b)
	}
	appender := f.dialect.Appender(v.Type())
	return appender(f, b, v)
}

func (f Formatter) HasFeature(feature feature.Feature) bool {
	return f.dialect.Features().Has(feature)
}

func (f Formatter) clone() Formatter {
	clone := f
	l := len(clone.namedArgs)
	clone.namedArgs = clone.namedArgs[:l:l]
	return clone
}

func (f Formatter) WithModel(model ArgAppender) Formatter {
	clone := f.clone()
	clone.model = model
	return clone
}

func (f Formatter) WithArg(name string, value interface{}) Formatter {
	clone := f.clone()
	clone.namedArgs = append(clone.namedArgs, namedArg{name: name, value: value})
	return clone
}

func (f Formatter) Arg(name string) interface{} {
	value, _ := f.namedArgs.Get(name)
	return value
}

func (f Formatter) FormatQuery(query string, args ...interface{}) string {
	if f.IsNop() || (args == nil && f.hasNoArgs()) || strings.IndexByte(query, '?') == -1 {
		return query
	}
	return internal.String(f.AppendQuery(nil, query, args...))
}

func (f Formatter) AppendQuery(dst []byte, query string, args ...interface{}) []byte {
	if f.IsNop() || (args == nil && f.hasNoArgs()) || strings.IndexByte(query, '?') == -1 {
		return append(dst, query...)
	}
	return f.append(dst, parser.NewString(query), args)
}

func (f Formatter) hasNoArgs() bool {
	return f.namedArgs == nil && f.model == nil
}

func (f Formatter) append(dst []byte, p *parser.Parser, args []interface{}) []byte {
	var model ArgAppender
	if len(args) > 0 {
		model, _ = args[0].(ArgAppender)
	}

	var argsIndex int
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

			if f.namedArgs != nil {
				if value, ok := f.namedArgs.Get(name); ok {
					dst = f.appendArg(dst, value)
					continue
				}
			}

			if model != nil {
				var ok bool
				dst, ok = model.AppendArg(f, dst, name)
				if ok {
					continue
				}
			}

			if f.model != nil {
				var ok bool
				dst, ok = f.model.AppendArg(f, dst, name)
				if ok {
					continue
				}
			}

		restore_arg:
			dst = append(dst, '?')
			dst = append(dst, name...)
			continue
		}

		if argsIndex >= len(args) {
			dst = append(dst, '?')
			continue
		}

		arg := args[argsIndex]
		argsIndex++

		dst = f.appendArg(dst, arg)
	}

	return dst
}

func (f Formatter) appendArg(b []byte, arg interface{}) []byte {
	switch arg := arg.(type) {
	case QueryAppender:
		bb, err := arg.AppendQuery(f, b)
		if err != nil {
			return dialect.AppendError(b, err)
		}
		return bb
	default:
		return f.dialect.Append(f, b, arg)
	}
}
