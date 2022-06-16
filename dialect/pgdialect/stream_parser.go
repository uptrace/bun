package pgdialect

import (
	"fmt"
	"io"
)

type streamParser struct {
	b []byte
	i int

	buf []byte
}

func newStreamParser(b []byte, start int) *streamParser {
	return &streamParser{
		b: b,
		i: start,
	}
}

func (p *streamParser) valid() bool {
	return p.i < len(p.b)
}

func (p *streamParser) skipByte(skip byte) error {
	c, err := p.readByte()
	if err != nil {
		return err
	}
	if c == skip {
		return nil
	}
	p.unreadByte()
	return fmt.Errorf("got %q, wanted %q", c, skip)
}

func (p *streamParser) readByte() (byte, error) {
	if p.valid() {
		c := p.b[p.i]
		p.i++
		return c, nil
	}
	return 0, io.EOF
}

func (p *streamParser) unreadByte() {
	p.i--
}

func (p *streamParser) peek() byte {
	if p.valid() {
		return p.b[p.i]
	}
	return 0
}

func (p *streamParser) skipNext() {
	p.i++
}
