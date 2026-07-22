package parser

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/uptrace/bun/internal"
)

type Parser struct {
	b []byte
	i int
}

func New(b []byte) *Parser {
	return &Parser{
		b: b,
	}
}

func NewString(s string) *Parser {
	return New(internal.Bytes(s))
}

func (p *Parser) Reset(b []byte) {
	p.b = b
	p.i = 0
}

func (p *Parser) Valid() bool {
	return p.i < len(p.b)
}

func (p *Parser) Remaining() []byte {
	return p.b[p.i:]
}

func (p *Parser) ReadByte() (byte, error) {
	if p.Valid() {
		ch := p.b[p.i]
		p.Advance()
		return ch, nil
	}
	return 0, io.ErrUnexpectedEOF
}

func (p *Parser) Read() byte {
	if p.Valid() {
		ch := p.b[p.i]
		p.Advance()
		return ch
	}
	return 0
}

func (p *Parser) Unread() {
	if p.i > 0 {
		p.i--
	}
}

func (p *Parser) Peek() byte {
	if p.Valid() {
		return p.b[p.i]
	}
	return 0
}

func (p *Parser) Advance() {
	p.i++
}

func (p *Parser) Skip(skip byte) error {
	ch := p.Peek()
	if ch == skip {
		p.Advance()
		return nil
	}
	return fmt.Errorf("got %q, wanted %q", ch, skip)
}

func (p *Parser) SkipPrefix(skip []byte) error {
	if !bytes.HasPrefix(p.b[p.i:], skip) {
		return fmt.Errorf("got %q, wanted prefix %q", p.b, skip)
	}
	p.i += len(skip)
	return nil
}

func (p *Parser) CutPrefix(skip []byte) bool {
	if !bytes.HasPrefix(p.b[p.i:], skip) {
		return false
	}
	p.i += len(skip)
	return true
}

func (p *Parser) ReadSep(sep byte) ([]byte, bool) {
	ind := bytes.IndexByte(p.b[p.i:], sep)
	if ind == -1 {
		b := p.b[p.i:]
		p.i = len(p.b)
		return b, false
	}

	b := p.b[p.i : p.i+ind]
	p.i += ind + 1
	return b, true
}

// ReadUntilPlaceholder reads up to the next '?' placeholder that is NOT inside a
// single-quoted string literal or a "--" / "/* */" comment, returning the text
// before it. The bool reports whether a placeholder was found (and, if so,
// consumed). It is the literal/comment-aware counterpart of ReadSep('?').
//
// This prevents a '?' that appears inside a string literal or comment in the
// query template from being mistaken for a bind placeholder, which would shift
// every subsequent positional argument.
//
// Double-quoted identifiers are intentionally NOT skipped: bun uses named
// placeholders inside quoted identifiers (e.g. multi-tenant "?tenant".table via
// WithNamedArg), so a '?' there must still be treated as a placeholder.
//
// Scope/limitations (intentional, to avoid ever skipping a real placeholder):
//   - Backslash escaping inside string literals is NOT interpreted. This matches
//     PostgreSQL with standard_conforming_strings=on (the default since 9.1),
//     where '\' is an ordinary character.
//   - Dollar-quoted strings ($tag$...$tag$) and escape strings (E'...') are not
//     specially tracked.
//
// In those unhandled cases the behaviour degrades to the previous (ReadSep)
// behaviour rather than skipping a genuine placeholder.
func (p *Parser) ReadUntilPlaceholder() ([]byte, bool) {
	b := p.b
	start := p.i
	i := p.i
	for i < len(b) {
		switch b[i] {
		case '?':
			p.i = i + 1
			return b[start:i], true
		case '\'':
			i = skipQuoted(b, i, '\'')
		case '-':
			if i+1 < len(b) && b[i+1] == '-' {
				i += 2
				for i < len(b) && b[i] != '\n' {
					i++
				}
			} else {
				i++
			}
		case '/':
			if i+1 < len(b) && b[i+1] == '*' {
				i = skipBlockComment(b, i)
			} else {
				i++
			}
		default:
			i++
		}
	}
	p.i = len(b)
	return b[start:], false
}

// skipQuoted returns the index just past a string literal that starts at
// b[i]==quote. A doubled quote (”) is treated as an escaped quote and does not
// terminate the literal. If the literal is unterminated, the end of input is
// returned.
func skipQuoted(b []byte, i int, quote byte) int {
	i++ // opening quote
	for i < len(b) {
		if b[i] == quote {
			if i+1 < len(b) && b[i+1] == quote {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return i
}

// skipBlockComment returns the index just past a /* */ comment that starts at
// b[i:]=="/*". Block comments nest, matching PostgreSQL.
func skipBlockComment(b []byte, i int) int {
	i += 2
	depth := 1
	for i < len(b) && depth > 0 {
		if b[i] == '/' && i+1 < len(b) && b[i+1] == '*' {
			depth++
			i += 2
		} else if b[i] == '*' && i+1 < len(b) && b[i+1] == '/' {
			depth--
			i += 2
		} else {
			i++
		}
	}
	return i
}

func (p *Parser) ReadIdentifier() (string, bool) {
	if p.i < len(p.b) && p.b[p.i] == '(' {
		s := p.i + 1
		if ind := bytes.IndexByte(p.b[s:], ')'); ind != -1 {
			b := p.b[s : s+ind]
			if isIdent(b) {
				p.i = s + ind + 1
				return internal.String(b), false
			}
		}
	}

	ind := len(p.b) - p.i
	var alpha bool
	for i, c := range p.b[p.i:] {
		if isNum(c) {
			continue
		}
		if isAlpha(c) || (i > 0 && alpha && c == '_') {
			alpha = true
			continue
		}
		ind = i
		break
	}
	if ind == 0 {
		return "", false
	}
	b := p.b[p.i : p.i+ind]
	p.i += ind
	return internal.String(b), !alpha
}

func (p *Parser) ReadNumber() int {
	ind := len(p.b) - p.i
	for i, c := range p.b[p.i:] {
		if !isNum(c) {
			ind = i
			break
		}
	}
	if ind == 0 {
		return 0
	}
	n, err := strconv.Atoi(string(p.b[p.i : p.i+ind]))
	if err != nil {
		panic(err)
	}
	p.i += ind
	return n
}

func isNum(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isIdent reports whether b is a valid identifier consisting of
// letters, digits, and underscores, with at least one letter.
func isIdent(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	hasAlpha := false
	for i, c := range b {
		if isAlpha(c) {
			hasAlpha = true
			continue
		}
		if isNum(c) {
			continue
		}
		if c == '_' && i > 0 {
			continue
		}
		return false
	}
	return hasAlpha
}
