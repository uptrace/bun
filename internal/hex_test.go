package internal

import "testing"

func TestHexEncoder_Write(t *testing.T) {
	enc := NewHexEncoder(nil)
	n, err := enc.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if n != 4 {
		t.Errorf("Write() returned n = %d, want 4", n)
	}
	if err != nil {
		t.Errorf("Write() returned err = %v, want nil", err)
	}
	enc.Close()

	want := `'\xdeadbeef'`
	if got := string(enc.Bytes()); want != got {
		t.Errorf("Bytes() = %q, want %q", got, want)
	}
}

func TestHexEncoder_Empty(t *testing.T) {
	enc := NewHexEncoder(nil)
	enc.Close()

	if got := string(enc.Bytes()); got != "NULL" {
		t.Errorf("Bytes() = %q, want %q", got, "NULL")
	}
}
