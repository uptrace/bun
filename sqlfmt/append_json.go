package sqlfmt

import "github.com/uptrace/bun/internal/parser"

func AppendJSON(b, jsonb []byte) []byte {
	b = append(b, '\'')

	p := parser.New(jsonb)
	for p.Valid() {
		c := p.Read()
		switch c {
		case '"':
			b = append(b, '"')
		case '\'':
			b = append(b, "''"...)
		case '\000':
			continue
		case '\\':
			if p.SkipBytes([]byte("u0000")) {
				b = append(b, "\\\\u0000"...)
			} else {
				b = append(b, '\\')
				if p.Valid() {
					b = append(b, p.Read())
				}
			}
		default:
			b = append(b, c)
		}
	}

	b = append(b, '\'')

	return b
}
