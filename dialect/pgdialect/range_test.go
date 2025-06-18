package pgdialect_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/schema"
)

func TestRange(t *testing.T) {
	t.Run("scan", func(t *testing.T) {
		for _, tt := range []struct {
			Name     string
			Value    string
			Expected any
		}{
			{
				Name:  "daterange",
				Value: " [1995-11-01,1995-12-01) ",
				Expected: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 0, 0, 0, 0, time.UTC),
					LowerBound: pgdialect.RangeLowerBoundInclusive,
					Upper:      time.Date(1995, time.December, 1, 0, 0, 0, 0, time.UTC),
					UpperBound: pgdialect.RangeUpperBoundExclusive,
				},
			},
			{
				Name:  "tstzrange",
				Value: `["1995-11-01 10:00:00+00","1995-12-01 10:00:00+00")`,
				Expected: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 10, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeLowerBoundInclusive,
					Upper:      time.Date(1995, time.December, 1, 10, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeUpperBoundExclusive,
				},
			},
			{
				Name:     "empty",
				Value:    "empty",
				Expected: &pgdialect.Range[time.Time]{},
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				r := &pgdialect.Range[time.Time]{}
				assert.NoError(t, r.Scan(tt.Value))
				assert.Equal(t, tt.Expected, r)
			})
		}
	})

	t.Run("append_query", func(t *testing.T) {
		for _, tt := range []struct {
			Name     string
			Value    schema.QueryAppender
			Expected string
		}{
			{
				Name: "daterange",
				Value: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 0, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeLowerBoundInclusive,
					Upper:      time.Date(1995, time.December, 1, 0, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeUpperBoundExclusive,
				},
				Expected: `'["1995-11-01 00:00:00+00:00","1995-12-01 00:00:00+00:00")'`,
			},
			{
				Name: "tstzrange",
				Value: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 10, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeLowerBoundInclusive,
					Upper:      time.Date(1995, time.December, 1, 10, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeUpperBoundExclusive,
				},
				Expected: `'["1995-11-01 10:00:00+00:00","1995-12-01 10:00:00+00:00")'`,
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				out, err := tt.Value.AppendQuery(schema.NewFormatter(pgdialect.New()), []byte{})
				assert.NoError(t, err)
				assert.Equal(t, tt.Expected, string(out))
			})
		}
	})
}

func TestNullRange(t *testing.T) {
	t.Run("scan", func(t *testing.T) {
		for _, tt := range []struct {
			Name     string
			Value    any
			Expected sql.Scanner
		}{
			{
				Name:  "not_null",
				Value: " [1995-11-01,1995-12-01) ",
				Expected: &pgdialect.NullRange[time.Time]{
					Range: pgdialect.Range[time.Time]{
						Lower:      time.Date(1995, time.November, 1, 0, 0, 0, 0, time.UTC),
						LowerBound: pgdialect.RangeLowerBoundInclusive,
						Upper:      time.Date(1995, time.December, 1, 0, 0, 0, 0, time.UTC),
						UpperBound: pgdialect.RangeUpperBoundExclusive,
					},
					Valid: true,
				},
			},
			{
				Name:  "null",
				Value: nil,
				Expected: &pgdialect.NullRange[time.Time]{
					Valid: false,
				},
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				r := tt.Expected
				assert.NoError(t, r.Scan(tt.Value))
				assert.Equal(t, tt.Expected, r)
			})
		}
	})

	t.Run("append_query", func(t *testing.T) {
		for _, tt := range []struct {
			Name     string
			Value    schema.QueryAppender
			Expected string
		}{
			{
				Name: "daterange",
				Value: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 0, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeLowerBoundInclusive,
					Upper:      time.Date(1995, time.December, 1, 0, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeUpperBoundExclusive,
				},
				Expected: `'["1995-11-01 00:00:00+00:00","1995-12-01 00:00:00+00:00")'`,
			},
			{
				Name: "tstzrange",
				Value: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 10, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeLowerBoundInclusive,
					Upper:      time.Date(1995, time.December, 1, 10, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeUpperBoundExclusive,
				},
				Expected: `'["1995-11-01 10:00:00+00:00","1995-12-01 10:00:00+00:00")'`,
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				out, err := tt.Value.AppendQuery(schema.NewFormatter(pgdialect.New()), []byte{})
				assert.NoError(t, err)
				assert.Equal(t, tt.Expected, string(out))
			})
		}
	})
}
