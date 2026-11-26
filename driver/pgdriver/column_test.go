package pgdriver

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimeUTC(t *testing.T) {
	inputs := []string{
		"2026-09-30 12:34:56",
		"2026-09-30 12:34:56.1",
		"2026-09-30 12:34:56.123456",
		"2026-09-30 12:34:56.123456789",
		"2026-09-30 12:34:56.1234567890",
		"2026-09-30 12:34:56.",
		"2026-09-30 12:34:56.12a",
		"0000-01-01 00:00:00",
		"9999-12-31 23:59:59.999999",
		"2024-02-29 00:00:00",
		"2026-02-29 00:00:00",
		"2000-02-29 00:00:00",
		"2100-02-29 00:00:00",
		"2026-04-31 00:00:00",
		"2026-00-10 00:00:00",
		"2026-13-10 00:00:00",
		"2026-09-00 00:00:00",
		"2026-09-30 24:00:00",
		"2026-09-30 23:60:00",
		"2026-09-30 23:59:60",
		"2026-09-30 1x:34:56",
	}
	for _, s := range inputs {
		for _, in := range []string{s, s + "+00"} {
			want, wantErr := time.ParseInLocation(timestampFormat, strings.TrimSuffix(in, "+00"), time.UTC)
			got, err := ParseTime(in)
			if wantErr != nil {
				require.EqualError(t, err, wantErr.Error(), in)
				continue
			}
			require.NoError(t, err, in)
			require.Equal(t, want, got, in)
			require.Equal(t, time.UTC, got.Location(), in)
		}
	}
}

func BenchmarkParseTime(b *testing.B) {
	for _, s := range []string{
		"2026-09-30 12:34:56.123456+00",
		"2026-09-30 12:34:56",
		"2026-09-30 12:34:56.123456-07",
		"2026-09-30 12:34:56.123456+05:30",
	} {
		b.Run(s, func(b *testing.B) {
			for b.Loop() {
				if _, err := ParseTime(s); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
