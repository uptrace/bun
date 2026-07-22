package pgdriver

import "testing"

func TestWithInsecure_TLSVerification(t *testing.T) {
	// WithInsecure(false): TLS enabled AND certificate verification on.
	conf := &Config{}
	WithInsecure(false)(conf)
	if conf.TLSConfig == nil {
		t.Fatal("WithInsecure(false): TLSConfig is nil, want a verifying config")
	}
	if conf.TLSConfig.InsecureSkipVerify {
		t.Fatal("WithInsecure(false): InsecureSkipVerify is true, want false (verified)")
	}

	// WithInsecure(true): plaintext, no TLS.
	conf = &Config{}
	WithInsecure(true)(conf)
	if conf.TLSConfig != nil {
		t.Fatalf("WithInsecure(true): TLSConfig = %v, want nil (plaintext)", conf.TLSConfig)
	}
}

func TestHostWithoutPort(t *testing.T) {
	tests := map[string]string{
		"localhost:5432":      "localhost",
		"db.example.com:5432": "db.example.com",
		"localhost":           "localhost",
		"[::1]:5432":          "::1",
		"192.168.1.10:5432":   "192.168.1.10",
	}
	for in, want := range tests {
		if got := hostWithoutPort(in); got != want {
			t.Errorf("hostWithoutPort(%q) = %q, want %q", in, got, want)
		}
	}
}
