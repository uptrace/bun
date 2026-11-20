package tagparser

import (
	"fmt"
	"strings"
)

type Tag struct {
	Name    string
	Options map[string][]string

	// Overrides contains semicolon-separated key=value overrides for each option.
	// Like Option, it describes the last occurrence of a repeated option.
	Overrides map[string]map[string]string
}

func (t Tag) IsZero() bool {
	return t.Name == "" && t.Options == nil && t.Overrides == nil
}

func (t Tag) HasOption(name string) bool {
	_, ok := t.Options[name]
	return ok
}

func (t Tag) Option(name string) (string, bool) {
	if vs, ok := t.Options[name]; ok {
		return vs[len(vs)-1], true
	}
	return "", false
}

func Parse(s string) Tag {
	if s == "" {
		return Tag{}
	}
	p := parser{
		s: s,
	}
	p.parse()
	return p.tag
}

type parser struct {
	s string
	i int

	tag      Tag
	seenName bool // for empty names
}

func (p *parser) setName(name string) {
	if p.seenName {
		p.addOption(name, "")
	} else {
		p.seenName = true
		p.tag.Name = name
	}
}

func (p *parser) addOption(key, value string) {
	p.seenName = true
	if key == "" {
		return
	}
	value, overrides := parseOverrides(value)
	if overrides != nil {
		if p.tag.Overrides == nil {
			p.tag.Overrides = make(map[string]map[string]string)
		}
		p.tag.Overrides[key] = overrides
	} else {
		delete(p.tag.Overrides, key)
	}
	if strings.HasPrefix(value, `"`) {
		var quoted = parser{s: value, i: 1}
		unquoted := quoted.parseQuotedValue()
		// Decode a quoted tag value, but preserve quotes that are part of a
		// larger expression, such as "schema"."type" or "type"[].
		if quoted.i == len(value) {
			value = unquoted
		}
	}
	if p.tag.Options == nil {
		p.tag.Options = make(map[string][]string)
	}
	if vs, ok := p.tag.Options[key]; ok {
		p.tag.Options[key] = append(vs, value)
	} else {
		p.tag.Options[key] = []string{value}
	}
}

func (p *parser) parse() {
	for p.valid() {
		p.parseKeyValue()
		if p.peek() == ',' {
			p.i++
		}
	}
}

func (p *parser) parseKeyValue() {
	start := p.i

	for p.valid() {
		switch c := p.read(); c {
		case ',':
			key := p.s[start : p.i-1]
			p.setName(key)
			return
		case ':':
			key := p.s[start : p.i-1]
			value := p.parseValue()
			p.addOption(key, value)
			return
		case '"':
			key := p.parseQuotedValue()
			p.setName(key)
			return
		}
	}

	key := p.s[start:p.i]
	p.setName(key)
}

func (p *parser) parseValue() string {
	start := p.i
	p.i += valueSeparator(p.s[p.i:], ',')
	value := p.s[start:p.i]
	if p.peek() == ',' {
		p.i++
	}
	return value
}

// parseOverrides separates an option's generic value from key=value overrides.
// Values and keys are interpreted by the caller, not by the tag parser.
func parseOverrides(s string) (string, map[string]string) {
	i := valueSeparator(s, ';')
	value := s[:i]
	var overrides map[string]string
	for i < len(s) {
		s = s[i+1:]
		i = valueSeparator(s, ';')
		part := strings.TrimSpace(s[:i])
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			panic(fmt.Errorf("bun: can't parse option override: %q", part))
		}
		if overrides == nil {
			overrides = make(map[string]string)
		}
		overrides[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return value, overrides
}

// Separators inside parameters or quoted values are not delimiters.
func valueSeparator(s string, separator byte) int {
	var quote byte
	var depth int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if c == separator && depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

func (p *parser) parseQuotedValue() string {
	if i := strings.IndexByte(p.s[p.i:], '"'); i >= 0 && p.s[p.i+i-1] != '\\' {
		s := p.s[p.i : p.i+i]
		p.i += i + 1
		return s
	}

	b := make([]byte, 0, 16)

	for p.valid() {
		switch c := p.read(); c {
		case '\\':
			b = append(b, p.read())
		case '"':
			return string(b)
		default:
			b = append(b, c)
		}
	}

	return ""
}

func (p *parser) valid() bool {
	return p.i < len(p.s)
}

func (p *parser) read() byte {
	if !p.valid() {
		return 0
	}
	c := p.s[p.i]
	p.i++
	return c
}

func (p *parser) peek() byte {
	if !p.valid() {
		return 0
	}
	c := p.s[p.i]
	return c
}
