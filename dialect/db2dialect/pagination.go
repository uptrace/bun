package db2dialect

import (
	"fmt"
)

// AppendOffsetLimit generates DB2 OFFSET...FETCH NEXT SQL syntax
// Used for pagination: OFFSET n ROWS FETCH NEXT m ROWS ONLY
// Panics if offset or limit is negative, since DB2 has no valid SQL representation for it.
func AppendOffsetLimit(b []byte, offset, limit int64) []byte {
	if offset < 0 || limit < 0 {
		panic(fmt.Sprintf("db2dialect: offset and limit must be non-negative (offset=%d, limit=%d)", offset, limit))
	}

	if offset > 0 {
		b = append(b, " OFFSET "...)
		b = appendInt(b, offset)
		b = append(b, " ROWS"...)
	}

	if limit > 0 {
		b = append(b, " FETCH NEXT "...)
		b = appendInt(b, limit)
		b = append(b, " ROWS ONLY"...)
	}

	return b
}

// appendInt is a helper to append an int64 as ASCII digits
// Uses uint64 arithmetic so math.MinInt64 does not overflow when negated.
func appendInt(b []byte, num int64) []byte {
	if num == 0 {
		return append(b, '0')
	}

	var u uint64
	if num < 0 {
		b = append(b, '-')
		u = uint64(-(num + 1)) + 1
	} else {
		u = uint64(num)
	}

	// Convert number to string by repeatedly dividing by 10
	var digits [20]byte
	i := len(digits)
	for u > 0 {
		i--
		digits[i] = byte(u%10) + '0'
		u /= 10
	}

	return append(b, digits[i:]...)
}
