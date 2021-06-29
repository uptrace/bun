package dialect_test

import (
	"testing"

	"github.com/uptrace/bun/dialect"
)

func BenchmarkAppendString(b *testing.B) {
	var dest []byte
	for i := 0; i < b.N; i++ {
		dest = dialect.AppendString(dest[:], "Hello, 世界")
	}
}
