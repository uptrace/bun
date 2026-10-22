package schema

import "testing"

// FormatQuery runs for every statement built from a template with arguments;
// strings take the dialect's AppendString path.
func BenchmarkFormatQuery(b *testing.B) {
	gen := NewQueryGen(lcTestDialect{newNopDialect()})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.FormatQuery("SELECT * FROM t WHERE id = ? AND name = ? AND note = ?", 42, "alice", "it's a note")
	}
}
