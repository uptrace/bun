package bun

import (
	"context"
	"database/sql"
)

type scanModel struct {
	hookStubs

	dest []interface{}
}

var _ model = scanModel{}

func newScanModel(dest []interface{}) scanModel {
	return scanModel{
		dest: dest,
	}
}

func (m scanModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	if !rows.Next() {
		return 0, errNoRows(rows)
	}

	if err := rows.Scan(m.dest...); err != nil {
		return 0, err
	}

	return 1, nil
}

func (m scanModel) ScanRow(ctx context.Context, rows *sql.Rows) error {
	return rows.Scan(m.dest...)
}
