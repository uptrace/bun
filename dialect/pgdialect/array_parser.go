package pgdialect

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
)

type arrayParser struct {
	*streamParser
	err error
}

func newArrayParser(b []byte) *arrayParser {
	p := &arrayParser{
		streamParser: newStreamParser(b, 1),
	}
	if len(b) < 2 || b[0] != '{' || b[len(b)-1] != '}' {
		p.err = fmt.Errorf("bun: can't parse array: %q", b)
	}
	return p
}

func (p *arrayParser) NextElem() ([]byte, error) {
	if p.err != nil {
		return nil, p.err
	}

	c, err := p.readByte()
	if err != nil {
		return nil, err
	}

	switch c {
	case '}':
		return nil, io.EOF
	case '"':
		b, err := p.readSubstring()
		if err != nil {
			return nil, err
		}

		if p.peek() == ',' {
			p.skipNext()
		}

		return b, nil
	default:
		b := p.readSimple()
		if bytes.Equal(b, []byte("NULL")) {
			b = nil
		}

		if p.peek() == ',' {
			p.skipNext()
		}

		return b, nil
	}
}

func (p *arrayParser) readSimple() []byte {
	p.unreadByte()

	if i := bytes.IndexByte(p.b[p.i:], ','); i >= 0 {
		b := p.b[p.i : p.i+i]
		p.i += i
		return b
	}

	b := p.b[p.i : len(p.b)-1]
	p.i = len(p.b) - 1
	return b
}

func (p *arrayParser) readSubstring() ([]byte, error) {
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
		if c == '\'' && next == '\'' {
			p.buf = append(p.buf, next)
			c, err = p.readByte()
			if err != nil {
				return nil, err
			}
			continue
		}

		p.buf = append(p.buf, c)
		c = next
	}

	if bytes.HasPrefix(p.buf, []byte("\\x")) && len(p.buf)%2 == 0 {
		data := p.buf[2:]
		buf := make([]byte, hex.DecodedLen(len(data)))
		n, err := hex.Decode(buf, data)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}

	return p.buf, nil
}
