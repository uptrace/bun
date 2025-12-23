package parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_Valid(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true",
			fields: fields{
				b: []byte("users AS u"),
				i: 0,
			},
			want: true,
		},
		{
			name: "return false",
			fields: fields{
				b: []byte("users AS u"),
				i: 10,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			require.Equal(t, tt.want, p.Valid())
		})
	}
}

func TestParser_Read(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	tests := []struct {
		name        string
		fields      fields
		want        byte
		idAfterRead int
	}{
		{
			name: "success to read first byte",
			fields: fields{
				b: []byte("users AS u"),
				i: 0,
			},
			want:        'u',
			idAfterRead: 1,
		},
		{
			name: "fail to read when parser is invalid",
			fields: fields{
				b: []byte("users AS u"),
				i: 10,
			},
			want:        0,
			idAfterRead: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			require.Equal(t, tt.want, p.Read())
			require.Equal(t, tt.idAfterRead, p.i)
		})
	}
}

func TestParser_Peek(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	tests := []struct {
		name        string
		fields      fields
		want        byte
		idAfterPeek int
	}{
		{
			name: "success to peek first byte",
			fields: fields{
				b: []byte("users AS u"),
				i: 0,
			},
			want:        'u',
			idAfterPeek: 0,
		},
		{
			name: "fail to peek when parser is invalid",
			fields: fields{
				b: []byte("users AS u"),
				i: 10,
			},
			want:        0,
			idAfterPeek: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			require.Equal(t, tt.want, p.Peek())
			require.Equal(t, tt.idAfterPeek, p.i)
		})
	}
}

func TestParser_Skip(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	type args struct {
		skip byte
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		want        error
		idAfterSkip int
	}{
		{
			name: "when to skip",
			fields: fields{
				b: []byte("? = ?"),
				i: 0,
			},
			args: args{
				skip: '?',
			},
			want:        nil,
			idAfterSkip: 1,
		},
		{
			name: "when not to skip",
			fields: fields{
				b: []byte("? = ?"),
				i: 0,
			},
			args: args{
				skip: '!',
			},
			want:        errors.New("got '?', wanted '!'"),
			idAfterSkip: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			require.Equal(t, tt.want, p.Skip(tt.args.skip))
		})
	}
}

func TestParser_SkipPrefix(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	type args struct {
		skip []byte
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		want        error
		idAfterSkip int
	}{
		{
			name: "when to skip",
			fields: fields{
				b: []byte("? = ?"),
				i: 0,
			},
			args: args{
				skip: []byte("? = "),
			},
			want:        nil,
			idAfterSkip: 4,
		},
		{
			name: "when not to skip",
			fields: fields{
				b: []byte("? = ?"),
				i: 0,
			},
			args: args{
				skip: []byte("hoge"),
			},
			want:        errors.New(`got "? = ?", wanted prefix "hoge"`),
			idAfterSkip: 0,
		},
		{
			name: "return error when argument is longer than the remaining bytes",
			fields: fields{
				b: []byte("? = ?"),
				i: 0,
			},
			args: args{
				skip: []byte("? = ? hoge"),
			},
			want:        errors.New(`got "? = ?", wanted prefix "? = ? hoge"`),
			idAfterSkip: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			require.Equal(t, tt.want, p.SkipPrefix(tt.args.skip))
			require.Equal(t, tt.idAfterSkip, p.i)
		})
	}
}

func TestParser_ReadSep(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	type args struct {
		sep byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
		wantOk bool
	}{
		{
			name: "when there are no separators",
			fields: fields{
				b: []byte("foo"),
				i: 0,
			},
			args: args{
				sep: '?',
			},
			want:   []byte("foo"),
			wantOk: false,
		},
		{
			name: "single question mark",
			fields: fields{
				b: []byte("(?) AS foo"),
				i: 0,
			},
			args: args{
				sep: '?',
			},
			want:   []byte("("),
			wantOk: true,
		},
		{
			name: "look at first question mark when there are two",
			fields: fields{
				b: []byte("? = ?"),
				i: 0,
			},
			args: args{
				sep: '?',
			},
			want:   []byte(""),
			wantOk: true,
		},
		{
			name: "look at second question mark when there are two",
			fields: fields{
				b: []byte("? = ?"),
				i: 1,
			},
			args: args{
				sep: '?',
			},
			want:   []byte(" = "),
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			got, gotOk := p.ReadSep(tt.args.sep)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantOk, gotOk)
		})
	}
}

func TestParser_ReadIdentifier(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		numeric bool
	}{
		{
			name: "read identifier closed by parenthesis",
			fields: fields{
				b: []byte("(?)"),
				i: 0,
			},
			want:    "?",
			numeric: false,
		},
		{
			name: "read space after question mark",
			fields: fields{
				b: []byte("? = ?"),
				i: 1,
			},
			want:    "",
			numeric: false,
		},
		{
			name: "read number after question mark",
			fields: fields{
				b: []byte("?0, ?1"),
				i: 1,
			},
			want:    "0",
			numeric: true,
		},
		{
			name: "read supported identifier `TableName`",
			fields: fields{
				b: []byte("?TableName"),
				i: 1,
			},
			want:    "TableName",
			numeric: false,
		},
		{
			name: "read supported identifier `TableAlias`",
			fields: fields{
				b: []byte("?TableAlias"),
				i: 1,
			},
			want:    "TableAlias",
			numeric: false,
		},
		{
			name: "read supported identifier `PKs`",
			fields: fields{
				b: []byte("?PKs"),
				i: 1,
			},
			want:    "PKs",
			numeric: false,
		},
		{
			name: "read supported identifier `TablePKs`",
			fields: fields{
				b: []byte("?TablePKs"),
				i: 1,
			},
			want:    "TablePKs",
			numeric: false,
		},
		{
			name: "read supported identifier `Columns`",
			fields: fields{
				b: []byte("?Columns"),
				i: 1,
			},
			want:    "Columns",
			numeric: false,
		},
		{
			name: "read supported identifier `TableColumns`",
			fields: fields{
				b: []byte("?TableColumns"),
				i: 1,
			},
			want:    "TableColumns",
			numeric: false,
		},
		{
			name: "read first identifier",
			fields: fields{
				b: []byte("?TableName AS ?TableAlias"),
				i: 1,
			},
			want:    "TableName",
			numeric: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			got, gotNumeric := p.ReadIdentifier()
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.numeric, gotNumeric)
		})
	}
}

func TestParser_ReadNumber(t *testing.T) {
	type fields struct {
		b []byte
		i int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "read single digit number",
			fields: fields{
				b: []byte("?0"),
				i: 1,
			},
			want: 0,
		},
		{
			name: "read double digit number",
			fields: fields{
				b: []byte("?10"),
				i: 1,
			},
			want: 10,
		},
		{
			name: "return 0 when there is no number",
			fields: fields{
				b: []byte("?TableName"),
				i: 1,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				b: tt.fields.b,
				i: tt.fields.i,
			}
			require.Equal(t, tt.want, p.ReadNumber())
		})
	}

}

func Test_isNum(t *testing.T) {
	numbers := "0123456789"
	for i := 0; i < len(numbers); i++ {
		require.True(t, isNum(numbers[i]))
	}
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < len(alphabet); i++ {
		require.False(t, isNum(alphabet[i]))
	}
	symbols := "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	for i := 0; i < len(symbols); i++ {
		require.False(t, isNum(symbols[i]))
	}
}

func Test_isAlpha(t *testing.T) {
	numbers := "0123456789"
	for i := 0; i < len(numbers); i++ {
		require.False(t, isAlpha(numbers[i]))
	}
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < len(alphabet); i++ {
		require.True(t, isAlpha(alphabet[i]))
	}
	symbols := "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	for i := 0; i < len(symbols); i++ {
		require.False(t, isNum(symbols[i]))
	}
}
