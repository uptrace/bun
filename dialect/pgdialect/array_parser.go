package pgdialect

import (
	"fmt"
	"io"
	"strings"
)

type arrayParser struct {
	s string
	i int

	buf []byte
	err error
}

func newArrayParser(s string) *arrayParser {
	p := &arrayParser{
		s: s,
		i: 1,
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		p.err = arrayParserError(s)
	}
	return p
}

func (p *arrayParser) NextElem() (string, error) {
	if p.err != nil {
		return "", p.err
	}

	c, err := p.readByte()
	if err != nil {
		return "", err
	}

	switch c {
	case '}':
		return "", io.EOF
	case '"':
		s, err := p.readSubstring()
		if err != nil {
			return "", err
		}

		if p.peek() == ',' {
			p.skipNext()
		}

		return s, nil
	default:
		s := p.readSimple()
		if s == "NULL" {
			s = ""
		}

		if p.peek() == ',' {
			p.skipNext()
		}

		return s, nil
	}
}

func (p *arrayParser) readSimple() string {
	p.unreadByte()

	if i := strings.IndexByte(p.s[p.i:], ','); i >= 0 {
		s := p.s[p.i : p.i+i]
		p.i += i
		return s
	}

	s := p.s[p.i : len(p.s)-1]
	p.i = len(p.s) - 1
	return s
}

func (p *arrayParser) readSubstring() (string, error) {
	c, err := p.readByte()
	if err != nil {
		return "", err
	}

	p.buf = p.buf[:0]
	for {
		if c == '"' {
			break
		}

		next, err := p.readByte()
		if err != nil {
			return "", err
		}

		if c == '\\' {
			switch next {
			case '\\', '"':
				p.buf = append(p.buf, next)

				c, err = p.readByte()
				if err != nil {
					return "", err
				}
			default:
				p.buf = append(p.buf, '\\')
				c = next
			}
			continue
		}

		p.buf = append(p.buf, c)
		c = next
	}

	return string(p.buf), nil
}

func (p *arrayParser) valid() bool {
	return p.i < len(p.s)
}

func (p *arrayParser) readByte() (byte, error) {
	if p.valid() {
		c := p.s[p.i]
		p.i++
		return c, nil
	}
	return 0, io.EOF
}

func (p *arrayParser) unreadByte() {
	p.i--
}

func (p *arrayParser) peek() byte {
	if p.valid() {
		return p.s[p.i]
	}
	return 0
}

func (p *arrayParser) skipNext() {
	p.i++
}

func arrayParserError(s string) error {
	return fmt.Errorf("bun: can't parse array: %q", s)
}
