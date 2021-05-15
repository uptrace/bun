package sqlfmt

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal/parser"
)

var defaultFmter = NewFormatter(feature.DefaultFeatures)

type QueryFormatter interface {
	HasFeature(v feature.Feature) bool
	FormatQuery(b []byte, query string, args ...interface{}) []byte
}

//------------------------------------------------------------------------------

type NopFormatter struct{}

func NewNopFormatter() NopFormatter {
	return NopFormatter{}
}

func (f NopFormatter) HasFeature(v feature.Feature) bool {
	return feature.DefaultFeatures.Has(v)
}

func (f NopFormatter) FormatQuery(b []byte, query string, args ...interface{}) []byte {
	return append(b, query...)
}

func IsNopFormatter(fmter QueryFormatter) bool {
	_, ok := fmter.(NopFormatter)
	return ok
}

//------------------------------------------------------------------------------

type ArgAppender interface {
	AppendArg(fmter QueryFormatter, b []byte, name string) ([]byte, bool)
}

type Formatter struct {
	features  feature.Feature
	model     ArgAppender
	namedArgs map[string]interface{}
}

var _ QueryFormatter = (*Formatter)(nil)

func NewFormatter(features feature.Feature) Formatter {
	return Formatter{
		features: features,
	}
}

func (f Formatter) String() string {
	if len(f.namedArgs) == 0 {
		return ""
	}

	keys := make([]string, 0, len(f.namedArgs))
	for key := range f.namedArgs {
		keys = append(keys, key)
	}

	ss := make([]string, len(keys))

	sort.Strings(keys)
	for i, k := range keys {
		ss[i] = fmt.Sprintf("%s=%v", k, f.namedArgs[k])
	}

	return " " + strings.Join(ss, " ")
}

func (f Formatter) clone() Formatter {
	clone := f

	if len(f.namedArgs) > 0 {
		clone.namedArgs = make(map[string]interface{}, len(f.namedArgs))
	}
	for name, value := range f.namedArgs {
		clone.namedArgs[name] = value
	}

	return clone
}

func (f *Formatter) setArg(arg string, value interface{}) {
	if f.namedArgs == nil {
		f.namedArgs = make(map[string]interface{})
	}
	f.namedArgs[arg] = value
}

func (f Formatter) WithModel(model ArgAppender) Formatter {
	clone := f.clone()
	clone.model = model
	return clone
}

func (f Formatter) WithArg(arg string, value interface{}) Formatter {
	clone := f.clone()
	clone.setArg(arg, value)
	return clone
}

func (f Formatter) Arg(arg string) interface{} {
	return f.namedArgs[arg]
}

func (f Formatter) FormatQueryBytes(dst, query []byte, args ...interface{}) []byte {
	if (args == nil && f.namedArgs == nil) || bytes.IndexByte(query, '?') == -1 {
		return append(dst, query...)
	}
	return f.append(dst, parser.New(query), args)
}

func (f Formatter) FormatQuery(dst []byte, query string, args ...interface{}) []byte {
	if (args == nil && f.namedArgs == nil) || strings.IndexByte(query, '?') == -1 {
		return append(dst, query...)
	}
	return f.append(dst, parser.NewString(query), args)
}

func (f Formatter) append(dst []byte, p *parser.Parser, args []interface{}) []byte {
	var argsIndex int

	var model ArgAppender
	if len(args) > 0 {
		model, _ = args[0].(ArgAppender)
	}

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
				if arg, ok := f.namedArgs[name]; ok {
					dst = f.appendArg(dst, arg)
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
			return AppendError(b, err)
		}
		return bb
	default:
		return Append(f, b, arg)
	}
}

func (f Formatter) HasFeature(v feature.Feature) bool {
	return f.features.Has(v)
}
