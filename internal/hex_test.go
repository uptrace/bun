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
			writes: [][]byte{{0x00, 0xab, 0xff}},
			want:   "'\\x00abff'",
		},
		{
			name:   "multiple writes",
			writes: [][]byte{{0x01}, {0x23, 0x45}},
			want:   "'\\x012345'",
		},
		{
			name:   "empty write",
			writes: [][]byte{{}},
			want:   "'\\x'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewHexEncoder(nil)

			for _, part := range tt.writes {
				n, err := enc.Write(part)
				if err != nil {
					t.Fatalf("Write returned an error: %v", err)
				}
				if n != len(part) {
					t.Fatalf("Write returned %d, want %d", n, len(part))
				}
			}

			if err := enc.Close(); err != nil {
				t.Fatalf("Close returned an error: %v", err)
			}

			if got := string(enc.Bytes()); got != tt.want {
				t.Fatalf("encoded value = %q, want %q", got, tt.want)
			}
		})
	}
}
