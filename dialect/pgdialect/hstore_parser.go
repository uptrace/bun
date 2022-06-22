package pgdialect

import (
	"bytes"
	"fmt"
)

type hstoreParser struct {
	*streamParser
	err error
}

func newHStoreParser(b []byte) *hstoreParser {
	p := &hstoreParser{
		streamParser: newStreamParser(b, 0),
	}
	if len(b) < 6 || b[0] != '"' {
		p.err = fmt.Errorf("bun: can't parse hstore: %q", b)
	}
	return p
}

func (p *hstoreParser) NextKey() (string, error) {
	if p.err != nil {
		return "", p.err
	}

	err := p.skipByte('"')
	if err != nil {
		return "", err
	}

	key, err := p.readSubstring()
	if err != nil {
		return "", err
	}

	const separator = "=>"

	for i := range separator {
		err = p.skipByte(separator[i])
		if err != nil {
			return "", err
		}
	}

	return string(key), nil
}

func (p *hstoreParser) NextValue() (string, error) {
	if p.err != nil {
		return "", p.err
	}

	c, err := p.readByte()
	if err != nil {
		return "", err
	}

	switch c {
	case '"':
		value, err := p.readSubstring()
		if err != nil {
			return "", err
		}

		if p.peek() == ',' {
			p.skipNext()
		}

		if p.peek() == ' ' {
			p.skipNext()
		}

		return string(value), nil
	default:
		value := p.readSimple()
		if bytes.Equal(value, []byte("NULL")) {
			value = nil
		}

		if p.peek() == ',' {
			p.skipNext()
		}

		return string(value), nil
	}
}

func (p *hstoreParser) readSimple() []byte {
	p.unreadByte()

	if i := bytes.IndexByte(p.b[p.i:], ','); i >= 0 {
		b := p.b[p.i : p.i+i]
		p.i += i
		return b
	}

	b := p.b[p.i:len(p.b)]
	p.i = len(p.b)
	return b
}

func (p *hstoreParser) readSubstring() ([]byte, error) {
	c, err := p.readByte()
	if err != nil {
		return nil, err
	}

	p.buf = p.buf[:0]
	for {
		if c == '"' {
			break
		}

		next, err := p.readByte()
		if err != nil {
			return nil, err
		}

		if c == '\\' {
			switch next {
			case '\\', '"':
				p.buf = append(p.buf, next)

				c, err = p.readByte()
				if err != nil {
					return nil, err
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

	return p.buf, nil
}
