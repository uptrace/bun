package sqliteshim_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestShim(t *testing.T) {
	sqldb, err := sqlOpen(t, sqliteshim.ShimName)
	require.NoError(t, err)
	_, err = sqldb.Exec("SELECT 1")
	if !sqliteshim.HasDriver() {
		assert.ErrorAs(t, err, new(*sqliteshim.UnsupportedError))
		return
	}
	assert.NoError(t, err)
}

func TestDriver(t *testing.T) {
	if !sqliteshim.HasDriver() {
		t.SkipNow()
	}
	sqldb, err := sqlOpen(t, sqliteshim.DriverName())
	require.NoError(t, err)
	_, err = sqldb.Exec("SELECT 1")
	require.NoError(t, err)
}

func TestNoImports(t *testing.T) {
	if sqliteshim.HasDriver() {
		t.SkipNow()
	}
	drivers := []string{
		"sqlite",  // modernc
		"sqlite3", // mattn
	}
	for _, driverName := range drivers {
		t.Run(driverName, func(t *testing.T) {
			_, err := sqlOpen(t, driverName)
			require.Error(t, err)
		})
	}
}

func sqlOpen(t *testing.T, driverName string) (*sql.DB, error) {
	sqldb, err := sql.Open(driverName, ":memory:")
	if err != nil {
		return sqldb, err
	}
	t.Cleanup(func() {
		assert.NoError(t, sqldb.Close())
	})
	return sqldb, nil
}
