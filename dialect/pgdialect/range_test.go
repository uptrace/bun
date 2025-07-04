package pgdialect_test

import (
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestRange(t *testing.T) {
	t.Run("scan", func(t *testing.T) {
		for _, tt := range []struct {
			Name     string
			Value    string
			Expected sql.Scanner
		}{
			{
				Name:  "int8range",
				Value: "[1,5)",
				Expected: &pgdialect.Range[int64]{
					Lower:      1,
					LowerBound: pgdialect.RangeBoundInclusiveLeft,
					Upper:      5,
					UpperBound: pgdialect.RangeBoundExclusiveRight,
				},
			},
			{
				Name:  "daterange",
				Value: "[1995-11-01,1995-12-01)",
				Expected: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 0, 0, 0, 0, time.UTC),
					LowerBound: pgdialect.RangeBoundInclusiveLeft,
					Upper:      time.Date(1995, time.December, 1, 0, 0, 0, 0, time.UTC),
					UpperBound: pgdialect.RangeBoundExclusiveRight,
				},
			},
			{
				Name:  "tstzrange",
				Value: `["1995-11-01 10:00:00+00","1995-12-01 10:00:00+00")`,
				Expected: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 10, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeBoundInclusiveLeft,
					Upper:      time.Date(1995, time.December, 1, 10, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeBoundExclusiveRight,
				},
			},
			{
				Name:     "empty",
				Value:    "empty",
				Expected: &pgdialect.Range[time.Time]{},
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				r := tt.Expected
				assert.NoError(t, r.Scan([]byte(tt.Value)))
				assert.Equal(t, tt.Expected, r)
			})
		}
	})

	t.Run("value", func(t *testing.T) {
		for _, tt := range []struct {
			Name     string
			Value    driver.Valuer
			Expected string
		}{
			{
				Name: "int8range",
				Value: &pgdialect.Range[int64]{
					Lower:      1,
					LowerBound: pgdialect.RangeBoundInclusiveLeft,
					Upper:      5,
					UpperBound: pgdialect.RangeBoundExclusiveRight,
				},
				Expected: "[1,5)",
			},
			{
				Name: "daterange",
				Value: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 0, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeBoundInclusiveLeft,
					Upper:      time.Date(1995, time.December, 1, 0, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeBoundExclusiveRight,
				},
				Expected: `["1995-11-01 00:00:00+00:00","1995-12-01 00:00:00+00:00")`,
			},
			{
				Name: "tstzrange",
				Value: &pgdialect.Range[time.Time]{
					Lower:      time.Date(1995, time.November, 1, 10, 0, 0, 0, time.Local),
					LowerBound: pgdialect.RangeBoundInclusiveLeft,
					Upper:      time.Date(1995, time.December, 1, 10, 0, 0, 0, time.Local),
					UpperBound: pgdialect.RangeBoundExclusiveRight,
				},
				Expected: `["1995-11-01 10:00:00+00:00","1995-12-01 10:00:00+00:00")`,
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				out, err := tt.Value.Value()
				assert.NoError(t, err)
				assert.Equal(t, tt.Expected, out)
			})
		}
	})
}
