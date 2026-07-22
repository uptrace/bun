package internal

import "testing"

func TestHexEncoder(t *testing.T) {
	tests := []struct {
		name   string
		writes [][]byte
		want   string
	}{
		{
			name: "no writes",
			want: "NULL",
		},
		{
			name:   "single write",
			writes: [][]byte{{0x00, 0x01, 0xfe, 0xff}},
			want:   `'\x0001feff'`,
		},
		{
			name:   "multiple writes",
			writes: [][]byte{{0xab}, {0xcd}},
			want:   `'\xabcd'`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enc := NewHexEncoder(nil)
			for _, write := range test.writes {
				if _, err := enc.Write(write); err != nil {
					t.Fatalf("Write() error = %v", err)
				}
			}
			if err := enc.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if got := string(enc.Bytes()); got != test.want {
				t.Fatalf("Bytes() = %q, want %q", got, test.want)
			}
		})
	}
}
